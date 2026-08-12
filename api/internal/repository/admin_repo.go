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

func NewPostgresAdminRepo(pool *pgxpool.Pool) AdminRepository {
	return &postgresAdminRepo{pool: pool}
}

func (r *postgresAdminRepo) CountAdmins(ctx context.Context) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM admin_users").Scan(&count)
	if err != nil {
		slog.Error("Failed to count admin users in PostgreSQL", "error", err)
		return 0, fmt.Errorf("failed to count admin users: %w", err)
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
		slog.Error("Failed to insert admin user into PostgreSQL", "username", username, "error", err)
		return nil, fmt.Errorf("failed to insert admin user: %w", err)
	}
	slog.Info("Successfully created new admin user in PostgreSQL", "username", username, "user_id", user.ID)
	return &user, nil
}

func (r *postgresAdminRepo) GetAdminByUsername(ctx context.Context, username string) (*domain.AdminUser, error) {
	var user domain.AdminUser
	var lastLogin *time.Time

	query := `SELECT id, username, password_hash, created_at, last_login_at FROM admin_users WHERE username = $1`
	err := r.pool.QueryRow(ctx, query, username).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt, &lastLogin)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		slog.Error("Failed to query admin user by username", "username", username, "error", err)
		return nil, err
	}

	user.LastLoginAt = lastLogin
	return &user, nil
}

func (r *postgresAdminRepo) GetAdminByID(ctx context.Context, id int) (*domain.AdminUser, error) {
	var user domain.AdminUser
	var lastLogin *time.Time

	query := `SELECT id, username, password_hash, created_at, last_login_at FROM admin_users WHERE id = $1`
	err := r.pool.QueryRow(ctx, query, id).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt, &lastLogin)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		slog.Error("Failed to query admin user by ID", "user_id", id, "error", err)
		return nil, err
	}

	user.LastLoginAt = lastLogin
	return &user, nil
}

func (r *postgresAdminRepo) UpdateLastLogin(ctx context.Context, id int) error {
	now := time.Now()
	_, err := r.pool.Exec(ctx, "UPDATE admin_users SET last_login_at = $1 WHERE id = $2", now, id)
	if err != nil {
		slog.Error("Failed to update admin last login timestamp", "user_id", id, "error", err)
		return err
	}
	return nil
}

func (r *postgresAdminRepo) UpdatePasswordHash(ctx context.Context, id int, passwordHash string) error {
	_, err := r.pool.Exec(ctx, "UPDATE admin_users SET password_hash = $1 WHERE id = $2", passwordHash, id)
	if err != nil {
		slog.Error("Failed to update admin password hash in PostgreSQL", "user_id", id, "error", err)
		return fmt.Errorf("failed to update password: %w", err)
	}
	slog.Info("Successfully updated admin password in PostgreSQL", "user_id", id)
	return nil
}
