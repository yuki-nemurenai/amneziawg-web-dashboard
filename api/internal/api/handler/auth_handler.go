package handler

import (
	"net/http"

	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/api/middleware"
	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/domain"
	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/service"
)

// AuthHandler serves /api/auth: first-time setup, login and the current
// administrator.
type AuthHandler struct {
	authService service.AuthService
}

// NewAuthHandler returns an AuthHandler backed by authService.
func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// GetStatus serves GET /api/auth/status.
func (h *AuthHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	status, err := h.authService.GetAuthStatus(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

// Setup serves POST /api/auth/setup.
func (h *AuthHandler) Setup(w http.ResponseWriter, r *http.Request) {
	var req domain.SetupRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	res, err := h.authService.SetupAdmin(r.Context(), req)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, res)
}

// Login serves POST /api/auth/login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req domain.LoginRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	res, err := h.authService.Login(r.Context(), req)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// Me serves GET /api/auth/me with the administrator from the token.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, user)
}

// ChangePassword serves POST /api/auth/change-password for the administrator
// from the token.
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(w, r)
	if !ok {
		return
	}

	var req domain.ChangePasswordRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	if err := h.authService.ChangePassword(r.Context(), user.ID, req); err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, messageResponse{Message: "Password updated successfully"})
}

// currentUser returns the administrator that AuthMiddleware put into the
// request context, or answers 401 if there is none.
func currentUser(w http.ResponseWriter, r *http.Request) (*domain.AdminUser, bool) {
	user, ok := r.Context().Value(middleware.UserContextKey).(*domain.AdminUser)
	if !ok || user == nil {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "Unauthorized"})
		return nil, false
	}
	return user, true
}
