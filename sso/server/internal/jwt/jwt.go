package jwt

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	jwt.RegisteredClaims
	SessionID string `json:"sid"`
	Sub       string `json:"sub"` // userID
}

func Sign(sessionID, userID string, secret []byte, exp time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(exp)),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   userID,
		},
		SessionID: sessionID,
		Sub:       userID,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func Parse(tokenString string, secret []byte) (sessionID, userID string, err error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if err != nil || !token.Valid {
		return "", "", err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return "", "", jwt.ErrTokenInvalidClaims
	}
	return claims.SessionID, claims.Sub, nil
}
