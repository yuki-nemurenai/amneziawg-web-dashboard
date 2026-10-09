package main

import (
	"testing"
)

// mapEnv returns a getenv over vars.
func mapEnv(vars map[string]string) func(string) string {
	return func(key string) string { return vars[key] }
}

func TestLoadEnvConfigBuildsDatabaseURL(t *testing.T) {
	tests := []struct {
		name string
		vars map[string]string
		want string
	}{
		{
			name: "defaults",
			vars: map[string]string{"JWT_SECRET": "s", "DB_PASSWORD": "pw"},
			want: "postgres://awg:pw@localhost:5432/awg_db?sslmode=disable",
		},
		{
			name: "password with URL characters",
			vars: map[string]string{"JWT_SECRET": "s", "DB_PASSWORD": "p@ss/w:rd#1", "DB_HOST": "postgres"},
			want: "postgres://awg:p%40ss%2Fw%3Ard%231@postgres:5432/awg_db?sslmode=disable",
		},
		{
			name: "DATABASE_URL wins",
			vars: map[string]string{"JWT_SECRET": "s", "DATABASE_URL": "postgres://u:p@db/x"},
			want: "postgres://u:p@db/x",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := loadEnvConfig(mapEnv(tt.vars))
			if err != nil {
				t.Fatalf("loadEnvConfig() error = %v", err)
			}
			if cfg.databaseURL != tt.want {
				t.Errorf("databaseURL = %q, want %q", cfg.databaseURL, tt.want)
			}
		})
	}
}

func TestLoadEnvConfigRequiresSecrets(t *testing.T) {
	tests := []struct {
		name string
		vars map[string]string
	}{
		{name: "no JWT secret", vars: map[string]string{"DB_PASSWORD": "pw"}},
		{name: "no database password", vars: map[string]string{"JWT_SECRET": "s"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := loadEnvConfig(mapEnv(tt.vars)); err == nil {
				t.Errorf("loadEnvConfig(%v) error = nil, want error", tt.vars)
			}
		})
	}
}
