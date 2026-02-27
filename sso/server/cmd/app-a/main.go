package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"sso/server/internal/jwt"
	"sso/server/internal/session"
	"sso/server/internal/users"
)

const (
	defaultTokenCookieName = "token_a" // 与 app-b 区分，同 host 下避免互相覆盖
	tokenCookieMaxAge      = 24 * 3600 // 24h
)

func main() {
	mockDB := os.Getenv("MOCK_DB")
	if mockDB == "" {
		mockDB = "./mock-db"
	}
	mockDB, _ = filepath.Abs(mockDB)
	sessionPath := filepath.Join(mockDB, "sessions.txt")
	usersPath := filepath.Join(mockDB, "users.txt")
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "sso-demo-secret-change-in-production"
	}
	secretBytes := []byte(jwtSecret)
	frontendOrigin := os.Getenv("FRONTEND_ORIGIN")
	if frontendOrigin == "" {
		frontendOrigin = "http://127.0.0.1:3001"
	}
	backendOrigin := os.Getenv("BACKEND_ORIGIN")
	if backendOrigin == "" {
		backendOrigin = "http://127.0.0.1:8081"
	}
	callbackURL := strings.TrimSuffix(backendOrigin, "/") + "/callback"
	getTokenURL := strings.TrimSuffix(backendOrigin, "/") + "/get_token"
	ssoOrigin := os.Getenv("SSO_ORIGIN")
	if ssoOrigin == "" {
		ssoOrigin = "http://127.0.0.1:8080"
	}
	ssoOrigin = strings.TrimSuffix(ssoOrigin, "/")
	cookieSecure := os.Getenv("COOKIE_SECURE") == "true"
	cookieSameSiteStr := os.Getenv("COOKIE_SAME_SITE")
	if cookieSameSiteStr == "" {
		cookieSameSiteStr = "Lax"
	}
	tokenCookieName := os.Getenv("COOKIE_NAME")
	if tokenCookieName == "" {
		tokenCookieName = defaultTokenCookieName
	}

	userList, err := users.Load(usersPath)
	if err != nil {
		panic("load users: " + err.Error())
	}
	sessions := session.NewStore(sessionPath)

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{frontendOrigin},
		AllowCredentials: true,
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
	}))

	setTokenCookie := func(c *gin.Context, token string, maxAge int) {
		sameSite := http.SameSiteLaxMode
		if cookieSameSiteStr == "None" {
			sameSite = http.SameSiteNoneMode
		}
		c.SetSameSite(sameSite)
		c.SetCookie(tokenCookieName, token, maxAge, "/", "", cookieSecure, true)
	}

	r.GET("/login", func(c *gin.Context) {
		c.Redirect(http.StatusFound, ssoOrigin+"/login?service="+url.QueryEscape(callbackURL))
	})

	validateTicket := func(ticket, serviceURL string) (user *users.User, token string, err error) {
		reqURL := ssoOrigin + "/service_validate?service=" + url.QueryEscape(serviceURL) + "&ticket=" + url.QueryEscape(ticket)
		resp, err := http.Get(reqURL)
		if err != nil {
			return nil, "", err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, "", nil
		}
		body, _ := io.ReadAll(resp.Body)
		var data struct {
			User  *users.User `json:"user"`
			Token string      `json:"token"`
		}
		if json.Unmarshal(body, &data) != nil {
			return nil, "", nil
		}
		return data.User, data.Token, nil
	}

	r.GET("/callback", func(c *gin.Context) {
		ticket := c.Query("ticket")
		if ticket == "" {
			c.String(http.StatusBadRequest, "missing ticket")
			return
		}
		user, token, err := validateTicket(ticket, callbackURL)
		if err != nil || user == nil || token == "" {
			c.String(http.StatusBadRequest, "invalid or expired ticket")
			return
		}
		setTokenCookie(c, token, tokenCookieMaxAge)
		c.Redirect(http.StatusFound, frontendOrigin+"/")
	})

	r.GET("/get_token", func(c *gin.Context) {
		ticket := c.Query("ticket")
		if ticket == "" {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		user, token, err := validateTicket(ticket, getTokenURL)
		if err != nil || user == nil || token == "" {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		setTokenCookie(c, token, tokenCookieMaxAge)
		c.JSON(http.StatusOK, gin.H{"user": user, "ok": true})
	})

	r.POST("/api/logout", func(c *gin.Context) {
		token, _ := c.Cookie(tokenCookieName)
		if token != "" {
			sessionID, _, _ := jwt.Parse(token, secretBytes)
			if sessionID != "" {
				_ = sessions.Delete(sessionID)
			}
		}
		c.Status(http.StatusOK)
	})

	r.GET("/api/me", func(c *gin.Context) {
		token, _ := c.Cookie(tokenCookieName)
		if token == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		sessionID, userID, err := jwt.Parse(token, secretBytes)
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

	_ = r.Run(":8081")
}
