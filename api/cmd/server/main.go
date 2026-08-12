package main

import (
	"context"
	"embed"
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
	// Structured logger setup using log/slog
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	slog.Info("Starting AmneziaWG Web Backend Service",
		"port", port,
		"config_path", configPath,
		"clients_dir", clientsDir,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initialize PostgreSQL Connection Pool via jackc/pgx/v5
	pool, err := repository.InitDB(ctx)
	if err != nil {
		slog.Error("Failed to initialize PostgreSQL database pool (pgx/v5)", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		localPath := "awg0.conf"
		if _, localErr := os.Stat(localPath); localErr == nil {
			slog.Info("Using local awg0.conf", "path", localPath)
			configPath = localPath
		} else {
			slog.Warn("Config file does not exist, creating placeholder", "path", configPath)
			_ = os.MkdirAll(filepath.Dir(configPath), 0755)
		}
	}

	// Layered Architecture Initialization
	adminRepo := repository.NewPostgresAdminRepo(pool)
	configRepo := repository.NewPostgresConfigRepo(pool, configPath)

	authService := service.NewAuthService(adminRepo)
	ipService := service.NewIPService()
	awgService := service.NewAWGService(configRepo, ipService, clientsDir, interfaceName)
	awgService.StartMetricsCollector()

	var webFS fs.FS
	distSub, err := fs.Sub(embeddedFrontend, "web/dist")
	if err == nil {
		if _, statErr := distSub.Open("index.html"); statErr == nil {
			webFS = distSub
			slog.Info("Embedded React Assets loaded")
		}
	}

	appRouter := router.NewRouter(awgService, authService, webFS)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      appRouter,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info(fmt.Sprintf("Server listening on http://0.0.0.0:%d", port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP Server error", "error", err)
			os.Exit(1)
		}
	}()

	<-shutdownChan
	slog.Info("Shutting down server gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Forced shutdown error", "error", err)
	} else {
		slog.Info("Server stopped cleanly")
	}
}
