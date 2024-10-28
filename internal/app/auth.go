package app

import (
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"net/http"
	"time"
)

type Claims struct {
	jwt.RegisteredClaims
	UserID string
}

var currentClaims Claims

const (
	jwtSecret  = "55c21cba3f534ae292ab2cc6921e6bc7"
	cookieName = "shortener_token"
	tokenExp   = 3 * time.Hour
)

func createToken(userID string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenExp)),
		},
		UserID: userID,
	})
	return token.SignedString([]byte(jwtSecret))
}

func parseToken(tokenString string) (*jwt.Token, error) {
	return jwt.ParseWithClaims(tokenString, &currentClaims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(jwtSecret), nil
	})
}

func readCookie(cookie *http.Cookie) bool {
	if cookie == nil {
		return false
	}
	token, err := parseToken(cookie.Value)
	if err != nil || !token.Valid {
		return false
	}
	return true
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(cookieName)
		readSuccess := readCookie(cookie)

		if err == nil && readSuccess && currentClaims.UserID == "" {
			http.Error(w, "No UserID in token", http.StatusUnauthorized)
			return
		}

		if err != nil || !readSuccess {
			userID := generateUserID()
			token, err := createToken(userID)
			if err != nil {
				http.Error(w, "Could not create token", http.StatusInternalServerError)
				return
			}

			http.SetCookie(w, &http.Cookie{
				Name:     cookieName,
				Value:    token,
				Path:     "/",
				HttpOnly: true,
				Expires:  time.Now().Add(tokenExp),
			})
		}

		next.ServeHTTP(w, r)
	})
}

func generateUserID() string {
	return uuid.New().String()
}
