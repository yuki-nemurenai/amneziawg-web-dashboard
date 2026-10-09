package main

import (
	"errors"
	"net"
	"net/url"
)

// envConfig is the configuration that comes from environment variables. The
// secrets have no defaults, so a forgotten variable stops the server instead
// of running it with a key known from the source code.
type envConfig struct {
	databaseURL string
	jwtSecret   []byte
	// listenPort is AWG_PORT, the UDP port Docker publishes; empty keeps the
	// port stored in the database.
	listenPort string
	publicIP   string
}

// loadEnvConfig reads the configuration from the environment through getenv.
// The database is DATABASE_URL or, if it is empty, the DB_* variables.
func loadEnvConfig(getenv func(string) string) (envConfig, error) {
	cfg := envConfig{
		databaseURL: getenv("DATABASE_URL"),
		jwtSecret:   []byte(getenv("JWT_SECRET")),
		listenPort:  getenv("AWG_PORT"),
		publicIP:    getenv("PUBLIC_IP"),
	}

	var errs []error
	if len(cfg.jwtSecret) == 0 {
		errs = append(errs, errors.New("JWT_SECRET is not set"))
	}
	if cfg.databaseURL == "" {
		password := getenv("DB_PASSWORD")
		if password == "" {
			errs = append(errs, errors.New("neither DATABASE_URL nor DB_PASSWORD is set"))
		}
		u := url.URL{
			Scheme:   "postgres",
			User:     url.UserPassword(envOr(getenv, "DB_USER", "awg"), password),
			Host:     net.JoinHostPort(envOr(getenv, "DB_HOST", "localhost"), envOr(getenv, "DB_PORT", "5432")),
			Path:     envOr(getenv, "DB_NAME", "awg_db"),
			RawQuery: url.Values{"sslmode": {envOr(getenv, "DB_SSLMODE", "disable")}}.Encode(),
		}
		cfg.databaseURL = u.String()
	}

	return cfg, errors.Join(errs...)
}

func envOr(getenv func(string) string, key, fallback string) string {
	if v := getenv(key); v != "" {
		return v
	}
	return fallback
}
