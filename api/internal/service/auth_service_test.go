package service

import (
	"context"
	"testing"
	"time"

	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/domain"
)

type mockAdminRepo struct {
	admins map[string]*domain.AdminUser
}

func (m *mockAdminRepo) CountAdmins(ctx context.Context) (int, error) {
	return len(m.admins), nil
}

func (m *mockAdminRepo) CreateAdmin(ctx context.Context, username, passwordHash string) (*domain.AdminUser, error) {
	u := &domain.AdminUser{
		ID:           len(m.admins) + 1,
		Username:     username,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now(),
	}
	m.admins[username] = u
	return u, nil
}

func (m *mockAdminRepo) GetAdminByUsername(ctx context.Context, username string) (*domain.AdminUser, error) {
	if u, ok := m.admins[username]; ok {
		return u, nil
	}
	return nil, domain.ErrNotFound
}

func (m *mockAdminRepo) UpdateLastLogin(ctx context.Context, id int) error {
	return nil
}

func (m *mockAdminRepo) GetAdminByID(ctx context.Context, id int) (*domain.AdminUser, error) {
	for _, u := range m.admins {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *mockAdminRepo) UpdatePasswordHash(ctx context.Context, id int, passwordHash string) error {
	for _, u := range m.admins {
		if u.ID == id {
			u.PasswordHash = passwordHash
			return nil
		}
	}
	return domain.ErrNotFound
}

func TestAuthService(t *testing.T) {
	ctx := context.Background()
	repo := &mockAdminRepo{admins: make(map[string]*domain.AdminUser)}
	svc := NewAuthService(repo)

	// 1. Initial Status check -> needs setup
	status, err := svc.GetAuthStatus(ctx)
	if err != nil {
		t.Fatalf("GetAuthStatus failed: %v", err)
	}
	if !status.NeedsSetup {
		t.Errorf("Expected NeedsSetup=true for empty database")
	}

	// 2. Setup Initial Admin
	resp, err := svc.SetupAdmin(ctx, domain.SetupRequest{
		Username: "admin",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("SetupAdmin failed: %v", err)
	}
	if resp.Token == "" || resp.User.Username != "admin" {
		t.Errorf("Invalid setup response: %+v", resp)
	}

	// 3. Status check after setup -> needs setup false
	status2, _ := svc.GetAuthStatus(ctx)
	if status2.NeedsSetup {
		t.Errorf("Expected NeedsSetup=false after initial admin creation")
	}

	// 4. Duplicate setup should fail
	_, err = svc.SetupAdmin(ctx, domain.SetupRequest{
		Username: "admin2",
		Password: "password123",
	})
	if err == nil {
		t.Errorf("Expected duplicate setup to fail")
	}

	// 5. Valid Login
	loginResp, err := svc.Login(ctx, domain.LoginRequest{
		Username: "admin",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	// 6. Validate Token
	userClaims, err := svc.ValidateToken(loginResp.Token)
	if err != nil {
		t.Fatalf("Token validation failed: %v", err)
	}
	if userClaims.Username != "admin" {
		t.Errorf("Expected username 'admin', got '%s'", userClaims.Username)
	}

	// 7. Invalid Login Password
	_, err = svc.Login(ctx, domain.LoginRequest{
		Username: "admin",
		Password: "wrongpassword",
	})
	if err == nil {
		t.Errorf("Expected wrong password login to fail")
	}
}
