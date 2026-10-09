// Package middleware holds the HTTP middleware of the API: JWT
// authentication and request logging.
package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/service"
)

type contextKey string

// UserContextKey is the request context key of the *domain.AdminUser that
// AuthMiddleware authenticated.
const UserContextKey contextKey = "user"

// AuthMiddleware rejects requests without a valid JWT with 401. The token is
// read from the Authorization header, then from the token query parameter and
// the awg_token cookie, because the UI downloads client configurations with a
// plain link that cannot set headers.
func AuthMiddleware(authService service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header (Bearer <token>)
			authHeader := r.Header.Get("Authorization")
			tokenStr := ""

			if after, ok := strings.CutPrefix(authHeader, "Bearer "); ok {
				tokenStr = after
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
