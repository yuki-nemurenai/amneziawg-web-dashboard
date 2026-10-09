package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/domain"
	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/repository"
)

const (
	tokenLifetime     = 24 * time.Hour
	minUsernameLength = 3
	minPasswordLength = 6
)

// errInvalidCredentials does not tell which of the two was wrong, so the
// login form cannot be used to find existing usernames.
var errInvalidCredentials = fmt.Errorf("%w: invalid username or password", domain.ErrUnauthorized)

// AuthService registers the administrator and issues the JWTs that protect
// the API.
type AuthService interface {
	GetAuthStatus(ctx context.Context) (*domain.AuthStatusResponse, error)
	// SetupAdmin creates the first administrator. It fails once one exists,
	// so the open setup endpoint cannot add more.
	SetupAdmin(ctx context.Context, req domain.SetupRequest) (*domain.AuthResponse, error)
	Login(ctx context.Context, req domain.LoginRequest) (*domain.AuthResponse, error)
	// ValidateToken returns the administrator a valid, unexpired token was
	// issued to. The user has only the ID and username from the token.
	ValidateToken(tokenString string) (*domain.AdminUser, error)
	ChangePassword(ctx context.Context, userID int, req domain.ChangePasswordRequest) error
}

type authService struct {
	adminRepo repository.AdminRepository
	jwtSecret []byte
	now       func() time.Time
}

type tokenClaims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// NewAuthService returns an AuthService that signs tokens with jwtSecret.
func NewAuthService(adminRepo repository.AdminRepository, jwtSecret []byte) AuthService {
	return &authService{
		adminRepo: adminRepo,
		jwtSecret: jwtSecret,
		now:       time.Now,
	}
}

func (s *authService) GetAuthStatus(ctx context.Context) (*domain.AuthStatusResponse, error) {
	count, err := s.adminRepo.CountAdmins(ctx)
	if err != nil {
		return nil, err
	}

	return &domain.AuthStatusResponse{
		NeedsSetup: count == 0,
	}, nil
}

func (s *authService) SetupAdmin(ctx context.Context, req domain.SetupRequest) (*domain.AuthResponse, error) {
	count, err := s.adminRepo.CountAdmins(ctx)
	if err != nil {
		return nil, err
	}
	if count > 0 {
		slog.Warn("AuthService: initial setup attempted when admins already exist")
		return nil, fmt.Errorf("%w: initial setup has already been completed", domain.ErrConflict)
	}

	if len(req.Username) < minUsernameLength {
		return nil, fmt.Errorf("%w: username must be at least %d characters", domain.ErrInvalidInput, minUsernameLength)
	}
	if len(req.Password) < minPasswordLength {
		return nil, fmt.Errorf("%w: password must be at least %d characters", domain.ErrInvalidInput, minPasswordLength)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.adminRepo.CreateAdmin(ctx, req.Username, string(hash))
	if err != nil {
		return nil, err
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	slog.Info("AuthService: initial administrator registered successfully", "username", user.Username)
	return &domain.AuthResponse{
		Token: token,
		User:  user,
	}, nil
}

func (s *authService) Login(ctx context.Context, req domain.LoginRequest) (*domain.AuthResponse, error) {
	user, err := s.adminRepo.GetAdminByUsername(ctx, req.Username)
	if errors.Is(err, domain.ErrNotFound) {
		slog.Warn("AuthService: login failed — user not found", "username", req.Username)
		return nil, errInvalidCredentials
	}
	if err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		slog.Warn("AuthService: login failed — invalid password", "username", req.Username)
		return nil, errInvalidCredentials
	}

	// The login succeeds even if the timestamp is not saved: it is
	// informational only.
	if err := s.adminRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		slog.Warn("AuthService: failed to save last login time", "user_id", user.ID, "error", err)
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	slog.Info("AuthService: admin logged in successfully", "username", user.Username, "user_id", user.ID)
	return &domain.AuthResponse{
		Token: token,
		User:  user,
	}, nil
}

func (s *authService) ValidateToken(tokenString string) (*domain.AdminUser, error) {
	claims := &tokenClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	}, jwt.WithTimeFunc(s.now))

	if err != nil || !token.Valid {
		slog.Warn("AuthService: JWT token validation failed", "error", err)
		return nil, fmt.Errorf("%w: invalid or expired authentication token", domain.ErrUnauthorized)
	}

	return &domain.AdminUser{
		ID:       claims.UserID,
		Username: claims.Username,
	}, nil
}

func (s *authService) ChangePassword(ctx context.Context, userID int, req domain.ChangePasswordRequest) error {
	if len(req.NewPassword) < minPasswordLength {
		return fmt.Errorf("%w: new password must be at least %d characters", domain.ErrInvalidInput, minPasswordLength)
	}

	user, err := s.adminRepo.GetAdminByID(ctx, userID)
	if err != nil {
		return err
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	if err := s.adminRepo.UpdatePasswordHash(ctx, userID, string(newHash)); err != nil {
		return err
	}

	slog.Info("AuthService: user password updated successfully", "user_id", userID, "username", user.Username)
	return nil
}

func (s *authService) generateToken(user *domain.AdminUser) (string, error) {
	now := s.now()
	claims := &tokenClaims{
		UserID:    user.ID,
		Username:  user.Username,
		ExpiresAt: jwt.NewNumericDate(now.Add(tokenLifetime)),
		IssuedAt:  jwt.NewNumericDate(now),
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return token, nil
}
