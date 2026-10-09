// Command server runs the AmneziaWG dashboard API. It manages the AmneziaWG
// interface of the host it runs on and stores its state in PostgreSQL.
package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/api/router"
	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/repository"
	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/service"
	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/system"
)

//go:embed web/dist/*
var embeddedFrontend embed.FS

func main() {
	var (
		port          int
		configPath    string
		clientsDir    string
		interfaceName string
	)

	rootCmd := &cobra.Command{
		Use:   "github.com/yuki-nemurenai/amneziawg-web-dashboard/api",
		Short: "AmneziaWG Web Backend",
		Run: func(cmd *cobra.Command, args []string) {
			runServer(port, configPath, clientsDir, interfaceName)
		},
	}

	rootCmd.Flags().IntVar(&port, "port", 8080, "HTTP server port")
	rootCmd.Flags().StringVar(&configPath, "config", "/etc/amnezia/amneziawg/awg0.conf", "Path to AmneziaWG server config")
	rootCmd.Flags().StringVar(&clientsDir, "clients-dir", "/etc/amnezia/amneziawg/clients", "Directory to store generated client configs")
	rootCmd.Flags().StringVar(&interfaceName, "interface", "awg0", "AmneziaWG interface name")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func runServer(port int, configPath string, clientsDir string, interfaceName string) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	slog.Info("Starting AmneziaWG Web Backend Service",
		"port", port,
		"config_path", configPath,
		"clients_dir", clientsDir,
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	initCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	pool, err := repository.InitDB(initCtx)
	if err != nil {
		slog.Error("Failed to initialize PostgreSQL database pool (pgx/v5)", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if _, err := os.Stat(configPath); errors.Is(err, fs.ErrNotExist) {
		localPath := "awg0.conf"
		if _, localErr := os.Stat(localPath); localErr == nil {
			slog.Info("Using local awg0.conf", "path", localPath)
			configPath = localPath
		} else {
			slog.Warn("Config file does not exist, creating placeholder", "path", configPath)
			if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
				slog.Error("Failed to create config directory", "path", filepath.Dir(configPath), "error", err)
			}
		}
	}

	adminRepo := repository.NewPostgresAdminRepo(pool)
	configRepo := repository.NewPostgresConfigRepo(initCtx, pool, configPath)

	authService := service.NewAuthService(adminRepo)
	awgService := service.NewAWGService(initCtx, service.AWGConfig{
		Repo:          configRepo,
		IPService:     service.NewIPService(),
		Commander:     system.Exec{},
		Locator:       system.NewLocator(os.Getenv("PUBLIC_IP")),
		ClientsDir:    clientsDir,
		InterfaceName: interfaceName,
	})
	awgService.StartMetricsCollector(ctx)

	var webFS fs.FS
	distSub, err := fs.Sub(embeddedFrontend, "web/dist")
	if err == nil {
		if _, statErr := distSub.Open("index.html"); statErr == nil {
			webFS = distSub
			slog.Info("Embedded React Assets loaded")
		}
	}

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      router.NewRouter(awgService, authService, webFS),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("Server listening", "addr", fmt.Sprintf("http://0.0.0.0:%d", port))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP Server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("Shutting down server gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Forced shutdown error", "error", err)
	} else {
		slog.Info("Server stopped cleanly")
	}
}
