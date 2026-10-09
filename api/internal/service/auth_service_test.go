package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/domain"
)

var testNow = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

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
		CreatedAt:    testNow,
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

// fakeClock is a time source that tests move by hand.
type fakeClock struct{ now time.Time }

func (c *fakeClock) Now() time.Time { return c.now }

// newTestAuthService returns an AuthService with no administrators and a clock
// set to testNow.
func newTestAuthService() (*authService, *fakeClock) {
	clock := &fakeClock{now: testNow}
	return &authService{
		adminRepo: &mockAdminRepo{admins: make(map[string]*domain.AdminUser)},
		jwtSecret: []byte("test-secret"),
		now:       clock.Now,
	}, clock
}

// setupAdmin creates the admin user with password password123.
func setupAdmin(t *testing.T, svc *authService) {
	t.Helper()
	if _, err := svc.SetupAdmin(t.Context(), domain.SetupRequest{Username: "admin", Password: "password123"}); err != nil {
		t.Fatalf("SetupAdmin() error = %v", err)
	}
}

func TestGetAuthStatusNeedsSetupUntilAdminExists(t *testing.T) {
	svc, _ := newTestAuthService()

	status, err := svc.GetAuthStatus(t.Context())
	if err != nil {
		t.Fatalf("GetAuthStatus() error = %v", err)
	}
	if !status.NeedsSetup {
		t.Errorf("NeedsSetup = false before setup, want true")
	}

	setupAdmin(t, svc)

	status, err = svc.GetAuthStatus(t.Context())
	if err != nil {
		t.Fatalf("GetAuthStatus() error = %v", err)
	}
	if status.NeedsSetup {
		t.Errorf("NeedsSetup = true after setup, want false")
	}
}

func TestSetupAdminRejectsInvalidRequest(t *testing.T) {
	tests := []struct {
		name string
		req  domain.SetupRequest
	}{
		{name: "short username", req: domain.SetupRequest{Username: "ab", Password: "password123"}},
		{name: "short password", req: domain.SetupRequest{Username: "admin", Password: "12345"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := newTestAuthService()
			if _, err := svc.SetupAdmin(t.Context(), tt.req); err == nil {
				t.Errorf("SetupAdmin(%+v) error = nil, want error", tt.req)
			}
		})
	}
}

func TestSetupAdminRejectsSecondAdmin(t *testing.T) {
	svc, _ := newTestAuthService()
	setupAdmin(t, svc)

	if _, err := svc.SetupAdmin(t.Context(), domain.SetupRequest{Username: "admin2", Password: "password123"}); err == nil {
		t.Errorf("second SetupAdmin() error = nil, want error")
	}
}

func TestLoginIssuesTokenForAdmin(t *testing.T) {
	svc, _ := newTestAuthService()
	setupAdmin(t, svc)

	resp, err := svc.Login(t.Context(), domain.LoginRequest{Username: "admin", Password: "password123"})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	user, err := svc.ValidateToken(resp.Token)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}
	if user.Username != "admin" {
		t.Errorf("ValidateToken().Username = %q, want %q", user.Username, "admin")
	}
}

func TestLoginRejectsInvalidCredentials(t *testing.T) {
	tests := []struct {
		name string
		req  domain.LoginRequest
	}{
		{name: "wrong password", req: domain.LoginRequest{Username: "admin", Password: "wrongpassword"}},
		{name: "unknown user", req: domain.LoginRequest{Username: "root", Password: "password123"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := newTestAuthService()
			setupAdmin(t, svc)

			if _, err := svc.Login(t.Context(), tt.req); !errors.Is(err, errInvalidCredentials) {
				t.Errorf("Login(%+v) error = %v, want %v", tt.req, err, errInvalidCredentials)
			}
		})
	}
}

func TestValidateTokenRejectsExpiredToken(t *testing.T) {
	svc, clock := newTestAuthService()
	setupAdmin(t, svc)
	resp, err := svc.Login(t.Context(), domain.LoginRequest{Username: "admin", Password: "password123"})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	clock.now = testNow.Add(tokenLifetime + time.Second)

	if _, err := svc.ValidateToken(resp.Token); err == nil {
		t.Errorf("ValidateToken() of an expired token error = nil, want error")
	}
}
