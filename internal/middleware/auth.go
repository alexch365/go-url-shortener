package middleware

import (
	"context"
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

type userCtxKeyType string

const (
	jwtSecret                 = "55c21cba3f534ae292ab2cc6921e6bc7"
	cookieName                = "shortener_token"
	tokenExp                  = 3 * time.Hour
	userCtxKey userCtxKeyType = "user_id"
)

func NewClaims() *Claims {
	return &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenExp)),
		},
		UserID: uuid.New().String(),
	}
}

func (claims *Claims) writeToken() (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret))
}

func (claims *Claims) parseToken(tokenString string) (*jwt.Token, error) {
	return jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(jwtSecret), nil
	})
}

func setCookie(w http.ResponseWriter, claims *Claims) {
	token, err := claims.writeToken()
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

func withUserID(r *http.Request, userID string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), userCtxKey, userID))
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := NewClaims()
		cookie, err := r.Cookie(cookieName)
		if err != nil {
			setCookie(w, claims)
		} else {
			token, err := claims.parseToken(cookie.Value)
			if err != nil || !token.Valid {
				setCookie(w, claims)
			} else if claims.UserID == "" {
				http.Error(w, "No UserID in token", http.StatusUnauthorized)
				return
			}
		}

		next.ServeHTTP(w, withUserID(r, claims.UserID))
	})
}

func GetUserID(ctx context.Context) string {
	userID, ok := ctx.Value(userCtxKey).(string)
	if !ok {
		return ""
	}
	return userID
}
