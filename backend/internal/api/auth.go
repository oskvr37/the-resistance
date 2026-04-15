package api

import (
	"encoding/json"
	"net/http"
	"regexp"
	"resistance/internal/response"
	"resistance/internal/store"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	tokenTTL          = 7 * 24 * time.Hour
	authCookieName    = "token"
	validRequestInput = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
)

// todo add validator

type loginRequest struct {
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
}

func generateUserToken(username, avatar, secret string) (string, string, error) {
	userID := uuid.New().String()

	claims := authClaims{
		UserID:     userID,
		UserName:   username,
		UserAvatar: avatar,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))

	return tokenString, userID, err
}

func (s *server) postAuthHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 4096)

	var req loginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if len(req.Username) < 3 || len(req.Username) > 32 {
		http.Error(w, "Username must be between 3 and 32 chars", http.StatusBadRequest)
		return
	}

	if !validRequestInput.MatchString(req.Username) {
		http.Error(w, "Username contains forbidden characters", http.StatusBadRequest)
		return
	}

	if len(req.Avatar) != 16 {
		http.Error(w, "Avatar must be 16 chars", http.StatusBadRequest)
		return
	}

	if !validRequestInput.MatchString(req.Avatar) {
		http.Error(w, "Avatar contains forbidden characters", http.StatusBadRequest)
		return
	}

	token, userID, err := generateUserToken(req.Username, req.Avatar, s.Config.JWTSecret)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	s.setAuthTokenCookie(w, token)

	type authResponse struct {
		ID string `json:"id"`
	}

	writeResponse(w, &store.RepositoryResponse[*authResponse]{
		Code: response.OK,
		Data: &authResponse{
			ID: userID,
		},
	})
}

func (s *server) setAuthTokenCookie(w http.ResponseWriter, token string) {
	isProd := s.Config.Environment != "development"

	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   isProd,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(tokenTTL / time.Second),
	})
}
