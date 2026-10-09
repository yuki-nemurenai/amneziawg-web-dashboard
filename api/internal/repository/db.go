// Package repository stores the dashboard state in PostgreSQL and mirrors the
// server configuration to the AmneziaWG configuration file.
package repository

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	connectAttempts   = 10
	connectRetryDelay = 2 * time.Second
	pingTimeout       = 3 * time.Second
)

// InitDB connects to PostgreSQL and creates the schema. It retries the
// connection because the database container may still be starting when the API
// starts. The connection comes from DATABASE_URL or, if it is empty, from the
// DB_* variables.
func InitDB(ctx context.Context) (*pgxpool.Pool, error) {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		host := getEnvOrDefault("DB_HOST", "localhost")
		port := getEnvOrDefault("DB_PORT", "5432")
		user := getEnvOrDefault("DB_USER", "awg")
		pass := getEnvOrDefault("DB_PASSWORD", "awgsecretpassword")
		name := getEnvOrDefault("DB_NAME", "awg_db")
		sslmode := getEnvOrDefault("DB_SSLMODE", "disable")

		connStr = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
			user, pass, host, port, name, sslmode)
	}

	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("parse database URL: %w", err)
	}

	config.MaxConns = 20
	config.MinConns = 2
	config.MaxConnLifetime = 30 * time.Minute

	var pool *pgxpool.Pool
	for attempt := 1; ; attempt++ {
		pool, err = connect(ctx, config)
		if err == nil {
			break
		}
		if attempt == connectAttempts {
			return nil, fmt.Errorf("connect to PostgreSQL after %d attempts: %w", attempt, err)
		}
		slog.Info("Waiting for PostgreSQL connection (pgx)...", "attempt", attempt, "error", err)
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("connect to PostgreSQL: %w", context.Cause(ctx))
		case <-time.After(connectRetryDelay):
		}
	}

	slog.Info("Successfully connected to PostgreSQL database via pgx/v5 pool")

	if err := migrateSchema(ctx, pool); err != nil {
		pool.Close()
		return nil, fmt.Errorf("migrate schema: %w", err)
	}

	return pool, nil
}

// connect opens a pool and checks that the database answers, because
// pgxpool.NewWithConfig does not connect until the first query.
func connect(ctx context.Context, config *pgxpool.Config) (*pgxpool.Pool, error) {
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

func migrateSchema(ctx context.Context, pool *pgxpool.Pool) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS admin_users (
			id SERIAL PRIMARY KEY,
			username VARCHAR(100) UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			last_login_at TIMESTAMP WITH TIME ZONE
		);`,
		`CREATE TABLE IF NOT EXISTS server_config (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			private_key TEXT NOT NULL,
			public_key TEXT NOT NULL,
			address TEXT NOT NULL,
			listen_port TEXT NOT NULL,
			endpoint TEXT,
			dns TEXT,
			lan_allowed TEXT,
			persistent_keepalive TEXT,
			post_up TEXT,
			post_down TEXT,
			jc TEXT, jmin TEXT, jmax TEXT,
			s1 TEXT, s2 TEXT, s3 TEXT, s4 TEXT,
			h1 TEXT, h2 TEXT, h3 TEXT, h4 TEXT,
			i1 TEXT, i2 TEXT, i3 TEXT, i4 TEXT, i5 TEXT,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS peers (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) UNIQUE NOT NULL,
			public_key TEXT UNIQUE NOT NULL,
			preshared_key TEXT,
			ip VARCHAR(50) UNIQUE NOT NULL,
			allowed_ips TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS traffic_history (
			id SERIAL PRIMARY KEY,
			timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			total_rx_bytes BIGINT NOT NULL,
			total_tx_bytes BIGINT NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_traffic_history_timestamp ON traffic_history(timestamp);`,
		`CREATE TABLE IF NOT EXISTS peer_traffic_history (
			id SERIAL PRIMARY KEY,
			timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			public_key TEXT NOT NULL,
			rx_bytes BIGINT NOT NULL,
			tx_bytes BIGINT NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_peer_traffic_history_timestamp ON peer_traffic_history(timestamp);`,
		`CREATE INDEX IF NOT EXISTS idx_peer_traffic_history_pubkey ON peer_traffic_history(public_key);`,
	}

	for i, query := range queries {
		if _, err := pool.Exec(ctx, query); err != nil {
			return fmt.Errorf("apply migration %d: %w", i+1, err)
		}
	}

	slog.Info("PostgreSQL database migrations applied successfully (pgx)")
	return nil
}

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
