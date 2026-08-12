package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/service"
)

type contextKey string

const UserContextKey contextKey = "user"

func AuthMiddleware(authService service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header (Bearer <token>)
			authHeader := r.Header.Get("Authorization")
			tokenStr := ""

			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
			} else if queryToken := r.URL.Query().Get("token"); queryToken != "" {
				tokenStr = queryToken
			} else if cookie, err := r.Cookie("awg_token"); err == nil {
				tokenStr = cookie.Value
			}

			if tokenStr == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "Authentication token required"})
				return
			}

			user, err := authService.ValidateToken(tokenStr)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}

			// Attach user to context
			ctx := context.WithValue(r.Context(), UserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
