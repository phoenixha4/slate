// Command server is the entry point for the Go todo application.
// It loads configuration, opens a database connection, runs schema migrations,
// and starts an HTTP server that serves the REST API and embedded frontend.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/phoenixha4/slate/internal/config"
	"github.com/phoenixha4/slate/internal/db"
	"github.com/phoenixha4/slate/internal/server"
	"github.com/phoenixha4/slate/internal/store"
	"github.com/phoenixha4/slate/internal/telemetry"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		if err := runHealthcheck(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	// Load and validate configuration from environment variables.
	cfg, err := config.Load()
	if err != nil {
		slog.New(slog.NewJSONHandler(os.Stdout, nil)).Error("configuration error", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// Initialise OpenTelemetry. This sets the global slog default to an
	// OTEL-backed handler, so all slog calls below flow through the pipeline.
	shutdownTelemetry, err := telemetry.Setup(ctx, telemetry.Config{
		ServiceName:    cfg.ServiceName,
		ServiceVersion: cfg.ServiceVersion,
		OTLPEndpoint:   cfg.OTLPEndpoint,
		LogLevel:       cfg.LogLevel,
		LogFormat:      cfg.LogFormat,
	})
	if err != nil {
		slog.New(slog.NewJSONHandler(os.Stdout, nil)).Error("telemetry setup failed", "error", err)
		os.Exit(1)
	}
	defer shutdownTelemetry()

	// From here on, slog.Default() is OTEL-backed.
	log := slog.Default()
	log.Info("configuration loaded", "app_env", cfg.AppEnv, "log_level", cfg.LogLevel.String(), "log_format", cfg.LogFormat)

	// Connect to PostgreSQL and run forward-only schema migrations.
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	log.Info("database connected")

	// Wire together store → handlers → server.
	st := store.NewPostgresStore(pool)
	handler := server.New(st, log, server.Options{
		CORSAllowedOrigins: cfg.CORSAllowedOrigins,
		ReadinessTimeout:   cfg.ReadinessTimeout,
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	// Start the server in a goroutine so we can listen for shutdown signals.
	go func() {
		log.Info("server starting", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Block until SIGINT or SIGTERM is received.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Graceful shutdown: let in-flight requests finish (up to ShutdownTimeout).
	log.Info("shutting down gracefully...")
	shutdownCtx, cancel := context.WithTimeout(ctx, cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("shutdown error", "error", err)
	}
	log.Info("server stopped")
}

func runHealthcheck() error {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	url := os.Getenv("HEALTHCHECK_URL")
	if url == "" {
		url = "http://" + net.JoinHostPort("127.0.0.1", port) + "/healthz"
	}

	client := &http.Client{Timeout: 3 * time.Second}
	res, err := client.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("healthcheck failed: %s", res.Status)
	}
	return nil
}
