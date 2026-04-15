package api

import (
	"context"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const userContextKey contextKey = "user"

type authClaims struct {
	UserID     string `json:"id"`
	UserName   string `json:"username"`
	UserAvatar string `json:"avatar"`
	jwt.RegisteredClaims
}

type userContext struct {
	ID     string
	Name   string
	Avatar string
}

func getUser(r *http.Request) (userContext, bool) {
	user, ok := r.Context().Value(userContextKey).(userContext)
	return user, ok
}

// extendCookie generates a new token and sets the Set-Cookie header
func (s *server) extendCookie(w http.ResponseWriter, claims *authClaims, secret string, duration time.Duration) {
	now := time.Now()
	claims.RegisteredClaims.ExpiresAt = jwt.NewNumericDate(now.Add(duration))
	claims.RegisteredClaims.IssuedAt = jwt.NewNumericDate(now)

	newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := newToken.SignedString([]byte(secret))
	if err != nil {
		return
	}

	s.setAuthTokenCookie(w, tokenString)
}
// auth middleware injects userContext and extends auth token in cookies
func (s *server) authMiddleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(authCookieName)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			claims := &authClaims{}
			token, err := jwt.ParseWithClaims(cookie.Value, claims, func(t *jwt.Token) (any, error) {
				return []byte(secret), nil
			})

			if err != nil || !token.Valid {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Check if we should extend: "If more than 1/2 of the lifetime has passed"
			exp, _ := claims.GetExpirationTime()
			if exp != nil && time.Until(exp.Time) < (tokenTTL/2) {
				s.extendCookie(w, claims, secret, tokenTTL)
			}

			// Inject into context...
			ctx := context.WithValue(r.Context(), userContextKey, userContext{
				ID:     claims.UserID,
				Name:   claims.UserName,
				Avatar: claims.UserAvatar,
			})

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
