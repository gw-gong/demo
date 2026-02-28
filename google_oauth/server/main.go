package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	redirectURL  = "http://127.0.0.1:8080/auth/callback"
	userinfoURL  = "https://www.googleapis.com/oauth2/v2/userinfo"
	cookieName   = "token"
	jwtExpiry    = 24 * time.Hour
)

var (
	oauth2Scopes = []string{
		"openid",
		"https://www.googleapis.com/auth/userinfo.email",
		"https://www.googleapis.com/auth/userinfo.profile",
	}
	stateStore sync.Map // key: state string, value: struct{}
	userStore  sync.Map // key: user.ID, value: *UserInfo
)

type jwtClaims struct {
	jwt.RegisteredClaims
	Sub string `json:"sub"`
}

func main() {
	_ = godotenv.Load()

	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	if clientID == "" {
		panic("GOOGLE_CLIENT_ID is required in .env")
	}
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	if clientSecret == "" {
		panic("GOOGLE_CLIENT_SECRET is required in .env")
	}
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://127.0.0.1:5173"
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = os.Getenv("SESSION_SECRET")
	}
	if jwtSecret == "" {
		jwtSecret = "google-oauth-demo-secret-change-in-production"
	}
	secretBytes := []byte(jwtSecret)

	oauth2Config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       oauth2Scopes,
		Endpoint:     google.Endpoint,
	}

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{frontendURL},
		AllowCredentials: true,
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
	}))

	r.GET("/auth/login", func(c *gin.Context) {
		state, err := randomState()
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		stateStore.Store(state, struct{}{})
		url := oauth2Config.AuthCodeURL(state)
		c.Redirect(http.StatusFound, url)
	})

	r.GET("/auth/callback", func(c *gin.Context) {
		state := c.Query("state")
		if state == "" {
			c.Redirect(http.StatusFound, frontendURL+"?error=missing_state")
			return
		}
		_, loaded := stateStore.LoadAndDelete(state)
		if !loaded {
			c.Redirect(http.StatusFound, frontendURL+"?error=invalid_state")
			return
		}
		code := c.Query("code")
		if code == "" {
			c.Redirect(http.StatusFound, frontendURL+"?error=missing_code")
			return
		}
		token, err := oauth2Config.Exchange(c.Request.Context(), code)
		if err != nil {
			c.Redirect(http.StatusFound, frontendURL+"?error=exchange_failed")
			return
		}
		user, err := fetchUserInfo(token.AccessToken)
		if err != nil {
			c.Redirect(http.StatusFound, frontendURL+"?error=userinfo_failed")
			return
		}
		userStore.Store(user.ID, user)
		tokenString, err := signJWT(user.ID, secretBytes, jwtExpiry)
		if err != nil {
			c.Redirect(http.StatusFound, frontendURL+"?error=jwt_failed")
			return
		}
		http.SetCookie(c.Writer, &http.Cookie{
			Name:     cookieName,
			Value:    tokenString,
			Path:     "/",
			MaxAge:   int(jwtExpiry.Seconds()),
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   false,
		})
		c.Redirect(http.StatusFound, frontendURL)
	})

	r.GET("/api/me", func(c *gin.Context) {
		tokenString, err := c.Cookie(cookieName)
		if err != nil || tokenString == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		sub, err := parseJWT(tokenString, secretBytes)
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		val, ok := userStore.Load(sub)
		if !ok {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		user := val.(*UserInfo)
		c.JSON(http.StatusOK, user)
	})

	r.GET("/auth/logout", func(c *gin.Context) {
		tokenString, _ := c.Cookie(cookieName)
		if tokenString != "" {
			if sub, err := parseJWT(tokenString, secretBytes); err == nil {
				userStore.Delete(sub)
			}
		}
		http.SetCookie(c.Writer, &http.Cookie{
			Name:     cookieName,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
		c.Redirect(http.StatusFound, frontendURL)
	})

	_ = r.Run(":8080")
}

func signJWT(sub string, secret []byte, exp time.Duration) (string, error) {
	now := time.Now()
	claims := jwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(exp)),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   sub,
		},
		Sub: sub,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func parseJWT(tokenString string, secret []byte) (sub string, err error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if err != nil || !token.Valid {
		return "", err
	}
	claims, ok := token.Claims.(*jwtClaims)
	if !ok {
		return "", jwt.ErrTokenInvalidClaims
	}
	return claims.Sub, nil
}

func randomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func fetchUserInfo(accessToken string) (*UserInfo, error) {
	req, err := http.NewRequest(http.MethodGet, userinfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var user UserInfo
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

type UserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}
