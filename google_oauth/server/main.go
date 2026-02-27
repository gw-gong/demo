package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"crypto/rand"
	"encoding/gob"
	"encoding/hex"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	clientID     = "352562790655-4tecvuqivispfalnv99ksqqst3nbjnna.apps.googleusercontent.com"
	redirectURL  = "http://127.0.0.1:8081/auth/callback"
	userinfoURL  = "https://www.googleapis.com/oauth2/v2/userinfo"
)

var (
	oauth2Scopes = []string{
		"openid",
		"https://www.googleapis.com/auth/userinfo.email",
		"https://www.googleapis.com/auth/userinfo.profile",
	}
)

func main() {
	_ = godotenv.Load()
	gob.Register((*UserInfo)(nil))

	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	if clientSecret == "" {
		panic("GOOGLE_CLIENT_SECRET is required in .env")
	}
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://127.0.0.1:5173"
	}
	sessionSecret := os.Getenv("SESSION_SECRET")
	if sessionSecret == "" {
		sessionSecret = "google-oauth-demo-secret-change-in-production"
	}

	oauth2Config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       oauth2Scopes,
		Endpoint:     google.Endpoint,
	}

	r := gin.Default()
	store := cookie.NewStore([]byte(sessionSecret))
	r.Use(sessions.Sessions("session", store))
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
		session := sessions.Default(c)
		session.Set("oauth_state", state)
		_ = session.Save()
		url := oauth2Config.AuthCodeURL(state)
		c.Redirect(http.StatusFound, url)
	})

	r.GET("/auth/callback", func(c *gin.Context) {
		session := sessions.Default(c)
		storedState, ok := session.Get("oauth_state").(string)
		if !ok {
			c.Redirect(http.StatusFound, frontendURL+"?error=missing_state")
			return
		}
		state := c.Query("state")
		if state == "" || state != storedState {
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
		session.Set("user", user)
		session.Delete("oauth_state")
		_ = session.Save()
		c.Redirect(http.StatusFound, frontendURL)
	})

	r.GET("/api/me", func(c *gin.Context) {
		session := sessions.Default(c)
		user := session.Get("user")
		if user == nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.JSON(http.StatusOK, user)
	})

	_ = r.Run(":8081")
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
