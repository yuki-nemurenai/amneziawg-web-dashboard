// Package domain holds the models and errors of the dashboard. It depends on
// nothing else in the project, so every layer can share it.
package domain

import (
	"errors"
	"time"
)

// ErrNotFound reports that a requested admin, peer or server configuration
// does not exist. Repositories wrap it so callers can check it with errors.Is.
var ErrNotFound = errors.New("record not found")

// AdminUser is an administrator of the dashboard. PasswordHash is never
// serialized, so the struct can be returned by the API as is.
type AdminUser struct {
	ID           int        `json:"id"`
	Username     string     `json:"username"`
	PasswordHash string     `json:"-"`
	CreatedAt    time.Time  `json:"created_at"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
}

// LoginRequest is the body of POST /api/auth/login.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// SetupRequest is the body of POST /api/auth/setup, which creates the first
// administrator.
type SetupRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// AuthResponse carries the JWT issued after setup or login.
type AuthResponse struct {
	Token string     `json:"token"`
	User  *AdminUser `json:"user"`
}

// AuthStatusResponse tells the UI whether to show the first-time setup form.
type AuthStatusResponse struct {
	NeedsSetup bool       `json:"needs_setup"`
	User       *AdminUser `json:"user,omitempty"`
}

// ChangePasswordRequest is the body of POST /api/auth/change-password.
type ChangePasswordRequest struct {
	NewPassword string `json:"new_password"`
}
