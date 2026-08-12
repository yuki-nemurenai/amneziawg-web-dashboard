package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/domain"
	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/repository"
)

type AuthService interface {
	GetAuthStatus(ctx context.Context) (*domain.AuthStatusResponse, error)
	SetupAdmin(ctx context.Context, req domain.SetupRequest) (*domain.AuthResponse, error)
	Login(ctx context.Context, req domain.LoginRequest) (*domain.AuthResponse, error)
	ValidateToken(tokenString string) (*domain.AdminUser, error)
	ChangePassword(ctx context.Context, userID int, req domain.ChangePasswordRequest) error
}

type authService struct {
	adminRepo repository.AdminRepository
	jwtSecret []byte
}

type Claims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func NewAuthService(adminRepo repository.AdminRepository) AuthService {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "awg-secret-jwt-key-change-in-production-12345"
	}
	return &authService{
		adminRepo: adminRepo,
		jwtSecret: []byte(secret),
	}
}

func (s *authService) GetAuthStatus(ctx context.Context) (*domain.AuthStatusResponse, error) {
	count, err := s.adminRepo.CountAdmins(ctx)
	if err != nil {
		slog.Error("AuthService: failed to check admin status", "error", err)
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
		return nil, fmt.Errorf("initial setup has already been completed")
	}

	if len(req.Username) < 3 {
		return nil, fmt.Errorf("username must be at least 3 characters")
	}
	if len(req.Password) < 6 {
		return nil, fmt.Errorf("password must be at least 6 characters")
	}

	// Bcrypt Password Hash
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("AuthService: failed to hash password with bcrypt", "error", err)
		return nil, fmt.Errorf("failed to hash password: %w", err)
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
	if err != nil {
		slog.Warn("AuthService: login failed — user not found", "username", req.Username)
		return nil, fmt.Errorf("invalid username or password")
	}

	// Verify Bcrypt Hash
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		slog.Warn("AuthService: login failed — invalid password", "username", req.Username)
		return nil, fmt.Errorf("invalid username or password")
	}

	_ = s.adminRepo.UpdateLastLogin(ctx, user.ID)

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
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		slog.Warn("AuthService: JWT token validation failed", "error", err)
		return nil, fmt.Errorf("invalid or expired authentication token")
	}

	return &domain.AdminUser{
		ID:       claims.UserID,
		Username: claims.Username,
	}, nil
}

func (s *authService) ChangePassword(ctx context.Context, userID int, req domain.ChangePasswordRequest) error {
	if len(req.NewPassword) < 6 {
		return fmt.Errorf("new password must be at least 6 characters")
	}

	user, err := s.adminRepo.GetAdminByID(ctx, userID)
	if err != nil {
		slog.Error("AuthService: ChangePassword failed — user not found", "user_id", userID, "error", err)
		return fmt.Errorf("user not found")
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("AuthService: ChangePassword failed to hash new password", "error", err)
		return fmt.Errorf("failed to process password: %w", err)
	}

	if err := s.adminRepo.UpdatePasswordHash(ctx, userID, string(newHash)); err != nil {
		return err
	}

	slog.Info("AuthService: user password updated successfully", "user_id", userID, "username", user.Username)
	return nil
}

func (s *authService) generateToken(user *domain.AdminUser) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}
