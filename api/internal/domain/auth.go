package domain

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("record not found")

// AdminUser represents an administrator in PostgreSQL
type AdminUser struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
}

// LoginRequest payload for logging in
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// SetupRequest payload for initial admin registration
type SetupRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// AuthResponse returns JWT token and User info
type AuthResponse struct {
	Token string     `json:"token"`
	User  *AdminUser `json:"user"`
}

// AuthStatusResponse indicates whether first-time setup is required
type AuthStatusResponse struct {
	NeedsSetup bool       `json:"needs_setup"`
	User       *AdminUser `json:"user,omitempty"`
}

// ChangePasswordRequest payload for updating user password
type ChangePasswordRequest struct {
	NewPassword string `json:"new_password"`
}
