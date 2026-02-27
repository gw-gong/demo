package main

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"sso/server/internal/jwt"
	"sso/server/internal/session"
	"sso/server/internal/users"
)

const (
	sessionCookieName = "sso_session"
	stPrefix          = "ST-"
	jwtExpiry         = 24 * time.Hour
	stExpiry          = 5 * time.Minute
)

type stRecord struct {
	Service   string
	UserID    string
	SessionID string
	Expire    time.Time
}

var (
	allowedOrigins []string
	stStore        = struct {
		sync.Mutex
		m map[string]*stRecord
	}{m: make(map[string]*stRecord)}
)

func main() {
	_ = godotenv.Load()

	mockDB := os.Getenv("MOCK_DB")
	if mockDB == "" {
		mockDB = "./mock-db"
	}
	mockDB, _ = filepath.Abs(mockDB)
	sessionPath := os.Getenv("SESSION_FILE_PATH")
	if sessionPath == "" {
		sessionPath = filepath.Join(mockDB, "sessions.txt")
	}
	usersPath := os.Getenv("USERS_FILE")
	if usersPath == "" {
		usersPath = filepath.Join(mockDB, "users.txt")
	}
	ssoOrigin := os.Getenv("SSO_ORIGIN")
	if ssoOrigin == "" {
		ssoOrigin = "http://127.0.0.1:8080"
	}
	ssoFrontend := os.Getenv("SSO_FRONTEND_URL")
	if ssoFrontend == "" {
		ssoFrontend = "http://127.0.0.1:3000"
	}
	allowedStr := os.Getenv("ALLOWED_SERVICE_BASES")
	if allowedStr != "" {
		allowedOrigins = strings.Split(allowedStr, ",")
		for i := range allowedOrigins {
			allowedOrigins[i] = strings.TrimSpace(allowedOrigins[i])
		}
	} else {
		allowedOrigins = []string{"http://127.0.0.1:3001", "http://127.0.0.1:3002", "http://127.0.0.1:8081/callback", "http://127.0.0.1:8082/callback"}
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "sso-demo-secret-change-in-production"
	}
	secretBytes := []byte(jwtSecret)

	userList, err := users.Load(usersPath)
	if err != nil {
		panic("load users: " + err.Error())
	}
	if len(userList) < 2 {
		panic("users.txt should contain at least two users")
	}

	sessions := session.NewStore(sessionPath)

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     append([]string{ssoFrontend}, allowedOrigins...),
		AllowCredentials: true,
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
	}))

	r.GET("/login", func(c *gin.Context) {
		service := c.Query("service")
		if service == "" || !allowService(service) {
			c.String(http.StatusBadRequest, "missing or invalid service")
			return
		}
		sessionID, _ := c.Cookie(sessionCookieName)
		if sessionID != "" {
			userID, ok := sessions.Find(sessionID)
			if ok {
				ticket, err := createST(service, userID, sessionID)
				if err != nil {
					c.AbortWithStatus(http.StatusInternalServerError)
					return
				}
				c.Redirect(http.StatusFound, service+"?ticket="+ticket)
				return
			}
		}
		c.Redirect(http.StatusFound, ssoFrontend+"/?service="+service)
	})

	r.POST("/login", func(c *gin.Context) {
		service := c.PostForm("service")
		if service == "" || !allowService(service) {
			c.String(http.StatusBadRequest, "missing or invalid service")
			return
		}
		username := c.PostForm("username")
		password := c.PostForm("password")
		user := users.Validate(userList, username, password)
		if user == nil {
			c.Redirect(http.StatusFound, ssoFrontend+"/?service="+service+"&error=invalid_credentials")
			return
		}
		sessionID, err := randomID()
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		expireAt := time.Now().Add(jwtExpiry)
		if err := sessions.Create(sessionID, user.ID, expireAt); err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		c.SetCookie(sessionCookieName, sessionID, int(jwtExpiry.Seconds()), "/", "", false, true)
		ticket, err := createST(service, user.ID, sessionID)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		c.Redirect(http.StatusFound, service+"?ticket="+ticket)
	})

	r.GET("/service_validate", func(c *gin.Context) {
		service := c.Query("service")
		ticket := c.Query("ticket")
		if service == "" || ticket == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing service or ticket"})
			return
		}
		sessionID, userID, ok := consumeST(ticket, service)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired ticket"})
			return
		}
		user := users.FindByID(userList, userID)
		if user == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "user not found"})
			return
		}
		token, err := jwt.Sign(sessionID, user.ID, secretBytes, jwtExpiry)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "jwt failed"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"user": user, "token": token})
	})

	r.GET("/api/me", func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if strings.HasPrefix(tokenString, "Bearer ") {
			tokenString = strings.TrimPrefix(tokenString, "Bearer ")
		}
		if tokenString == "" {
			tokenString, _ = c.Cookie("token")
		}
		if tokenString == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		sessionID, userID, err := jwt.Parse(tokenString, secretBytes)
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		_, ok := sessions.Find(sessionID)
		if !ok {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		user := users.FindByID(userList, userID)
		if user == nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.JSON(http.StatusOK, gin.H{"user": user})
	})

	r.GET("/api/silent-ticket", func(c *gin.Context) {
		service := c.Query("service")
		if service == "" || !allowService(service) {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		sessionID, _ := c.Cookie(sessionCookieName)
		if sessionID == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		userID, ok := sessions.Find(sessionID)
		if !ok {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		ticket, err := createST(service, userID, sessionID)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ticket": ticket})
	})

	r.GET("/api/silent-info", func(c *gin.Context) {
		sessionID, _ := c.Cookie(sessionCookieName)
		if sessionID == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		userID, ok := sessions.Find(sessionID)
		if !ok {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		user := users.FindByID(userList, userID)
		if user == nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		token, err := jwt.Sign(sessionID, user.ID, secretBytes, jwtExpiry)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		c.JSON(http.StatusOK, gin.H{"user": user, "token": token})
	})

	r.GET("/silent-check", func(c *gin.Context) {
		origin := c.Query("origin")
		serviceParam := c.Query("service")
		if origin == "" {
			origin = "*"
		}
		serviceJS := "''"
		if serviceParam != "" {
			serviceJS = quoteJS(serviceParam)
		}
		html := `<!DOCTYPE html><html><head><meta charset="utf-8"></head><body>
<script>
(function() {
  var origin = ` + quoteJS(origin) + `;
  var service = ` + serviceJS + `;
  if (!service) return;
  fetch("/api/silent-ticket?service=" + encodeURIComponent(service), { credentials: "include" })
    .then(function(r) {
      if (!r.ok) return r.json().then(function() { throw new Error(); });
      return r.json();
    })
    .then(function(data) {
      if (data && data.ticket) {
        try { window.parent.postMessage({ type: "SSO_TICKET", ticket: data.ticket }, origin); } catch (e) {}
      }
    })
    .catch(function() {});
})();
</script></body></html>`
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
	})

	_ = r.Run(":8080")
}

func allowService(service string) bool {
	for _, o := range allowedOrigins {
		if strings.HasPrefix(service, o) || service == o {
			return true
		}
	}
	return false
}

func randomID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func createST(service, userID, sessionID string) (string, error) {
	id, err := randomID()
	if err != nil {
		return "", err
	}
	ticket := stPrefix + id
	stStore.Lock()
	stStore.m[ticket] = &stRecord{Service: service, UserID: userID, SessionID: sessionID, Expire: time.Now().Add(stExpiry)}
	stStore.Unlock()
	return ticket, nil
}

func consumeST(ticket, service string) (sessionID, userID string, ok bool) {
	if !strings.HasPrefix(ticket, stPrefix) {
		return "", "", false
	}
	stStore.Lock()
	rec := stStore.m[ticket]
	delete(stStore.m, ticket)
	stStore.Unlock()
	if rec == nil || rec.Expire.Before(time.Now()) || rec.Service != service {
		return "", "", false
	}
	return rec.SessionID, rec.UserID, true
}

func quoteJS(s string) string {
	return "\"" + strings.ReplaceAll(strings.ReplaceAll(s, "\\", "\\\\"), "\"", "\\\"") + "\""
}
