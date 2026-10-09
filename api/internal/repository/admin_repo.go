package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/domain"
)

// AdminRepository stores the dashboard administrators. Lookups of a missing
// administrator return an error wrapping domain.ErrNotFound.
type AdminRepository interface {
	CountAdmins(ctx context.Context) (int, error)
	CreateAdmin(ctx context.Context, username, passwordHash string) (*domain.AdminUser, error)
	GetAdminByUsername(ctx context.Context, username string) (*domain.AdminUser, error)
	GetAdminByID(ctx context.Context, id int) (*domain.AdminUser, error)
	UpdateLastLogin(ctx context.Context, id int) error
	UpdatePasswordHash(ctx context.Context, id int, passwordHash string) error
}

type postgresAdminRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresAdminRepo returns an AdminRepository backed by the admin_users
// table.
func NewPostgresAdminRepo(pool *pgxpool.Pool) AdminRepository {
	return &postgresAdminRepo{pool: pool}
}

func (r *postgresAdminRepo) CountAdmins(ctx context.Context) (int, error) {
	var count int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM admin_users").Scan(&count); err != nil {
		return 0, fmt.Errorf("count admins: %w", err)
	}
	return count, nil
}

func (r *postgresAdminRepo) CreateAdmin(ctx context.Context, username, passwordHash string) (*domain.AdminUser, error) {
	var user domain.AdminUser
	user.Username = username
	user.PasswordHash = passwordHash
	user.CreatedAt = time.Now()

	query := `
		INSERT INTO admin_users (username, password_hash, created_at)
		VALUES ($1, $2, $3)
		RETURNING id, username, created_at
	`
	err := r.pool.QueryRow(ctx, query, username, passwordHash, user.CreatedAt).Scan(&user.ID, &user.Username, &user.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert admin %q: %w", username, err)
	}
	slog.Info("Successfully created new admin user in PostgreSQL", "username", username, "user_id", user.ID)
	return &user, nil
}

func (r *postgresAdminRepo) GetAdminByUsername(ctx context.Context, username string) (*domain.AdminUser, error) {
	var user domain.AdminUser
	var lastLogin *time.Time

	query := `SELECT id, username, password_hash, created_at, last_login_at FROM admin_users WHERE username = $1`
	err := r.pool.QueryRow(ctx, query, username).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt, &lastLogin)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("admin %q: %w", username, domain.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("get admin %q: %w", username, err)
	}

	user.LastLoginAt = lastLogin
	return &user, nil
}

func (r *postgresAdminRepo) GetAdminByID(ctx context.Context, id int) (*domain.AdminUser, error) {
	var user domain.AdminUser
	var lastLogin *time.Time

	query := `SELECT id, username, password_hash, created_at, last_login_at FROM admin_users WHERE id = $1`
	err := r.pool.QueryRow(ctx, query, id).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt, &lastLogin)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("admin %d: %w", id, domain.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("get admin %d: %w", id, err)
	}

	user.LastLoginAt = lastLogin
	return &user, nil
}

func (r *postgresAdminRepo) UpdateLastLogin(ctx context.Context, id int) error {
	if _, err := r.pool.Exec(ctx, "UPDATE admin_users SET last_login_at = $1 WHERE id = $2", time.Now(), id); err != nil {
		return fmt.Errorf("update last login of admin %d: %w", id, err)
	}
	return nil
}

func (r *postgresAdminRepo) UpdatePasswordHash(ctx context.Context, id int, passwordHash string) error {
	if _, err := r.pool.Exec(ctx, "UPDATE admin_users SET password_hash = $1 WHERE id = $2", passwordHash, id); err != nil {
		return fmt.Errorf("update password of admin %d: %w", id, err)
	}
	slog.Info("Successfully updated admin password in PostgreSQL", "user_id", id)
	return nil
}
