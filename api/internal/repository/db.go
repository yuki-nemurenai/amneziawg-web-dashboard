package repository

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

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
		return nil, fmt.Errorf("failed to parse postgres conn string: %w", err)
	}

	config.MaxConns = 20
	config.MinConns = 2
	config.MaxConnLifetime = 30 * time.Minute

	var pool *pgxpool.Pool
	var pingErr error

	for i := 1; i <= 10; i++ {
		pool, pingErr = pgxpool.NewWithConfig(ctx, config)
		if pingErr == nil {
			pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			pingErr = pool.Ping(pingCtx)
			cancel()
			if pingErr == nil {
				break
			}
		}
		slog.Info("Waiting for PostgreSQL connection (pgx)...", "attempt", i, "error", pingErr)
		time.Sleep(2 * time.Second)
	}

	if pingErr != nil {
		return nil, fmt.Errorf("could not connect to postgres via pgx after retries: %w", pingErr)
	}

	slog.Info("Successfully connected to PostgreSQL database via pgx/v5 pool")

	// Run auto migrations
	if err := migrateSchema(ctx, pool); err != nil {
		return nil, fmt.Errorf("database migration failed: %w", err)
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

	for _, query := range queries {
		if _, err := pool.Exec(ctx, query); err != nil {
			slog.Error("Database migration failed", "error", err, "query", query)
			return err
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
