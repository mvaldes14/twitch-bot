// package main starts the server
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mvaldes14/twitch-bot/pkgs/cache"
	"github.com/mvaldes14/twitch-bot/pkgs/secrets"
	"github.com/mvaldes14/twitch-bot/pkgs/server"
	"github.com/mvaldes14/twitch-bot/pkgs/telemetry"
)

const port = ":3000"

// Required environment variables for the application
var requiredEnvVars = []string{
	"TWITCH_CLIENT_ID",
	"TWITCH_CLIENT_SECRET",
	"TWITCH_REFRESH_TOKEN",
	"TWITCH_USER_TOKEN",
	"ADMIN_TOKEN",
	"SPOTIFY_CLIENT_ID",
	"SPOTIFY_CLIENT_SECRET",
	"SPOTIFY_REFRESH_TOKEN",
	"REDIS_URL",
	"OTEL_EXPORTER_OTLP_ENDPOINT",
}

// validateRequiredEnvVars checks that all required environment variables are present
// and returns a detailed error message for each missing variable
func validateRequiredEnvVars() error {
	var missingVars []string
	for _, envVar := range requiredEnvVars {
		if os.Getenv(envVar) == "" {
			missingVars = append(missingVars, envVar)
		}
	}
	if len(missingVars) > 0 {
		return fmt.Errorf("missing required environment variables: %v", missingVars)
	}
	return nil
}

func main() {
	logger := telemetry.NewLogger("main")
	if err := run(logger); err != nil {
		logger.Error("Startup failed", err)
		os.Exit(1)
	}
}

// run holds the whole lifecycle so every deferred cleanup still executes on the
// error paths. main is the only place allowed to call os.Exit.
func run(logger *telemetry.CustomLogger) error {
	ctx := context.Background()

	// Validate required environment variables before initialization
	if err := validateRequiredEnvVars(); err != nil {
		return fmt.Errorf("environment validation failed: %w", err)
	}
	logger.Info("Environment variables validated successfully")

	// Initialize OpenTelemetry
	otelConfig := telemetry.GetConfigFromEnv()
	if err := telemetry.InitOTEL(ctx, otelConfig); err != nil {
		return fmt.Errorf("failed to initialize OpenTelemetry: %w", err)
	}
	logger.Info("OpenTelemetry initialized successfully")

	// Initialize OTEL metrics
	if err := telemetry.InitMetrics(); err != nil {
		return fmt.Errorf("failed to initialize metrics: %w", err)
	}
	logger.Info("Metrics initialized successfully")

	// Ensure OTEL providers are shut down on exit
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := telemetry.Shutdown(shutdownCtx); err != nil {
			logger.Error("Failed to shutdown OpenTelemetry", err)
		}
	}()

	s, err := secrets.NewSecretService()
	if err != nil {
		return fmt.Errorf("failed to initialize secret service: %w", err)
	}
	s.InitSecrets(ctx)

	// Start background token renewal (cancelled on shutdown)
	renewCtx, renewCancel := context.WithCancel(ctx)
	defer renewCancel()
	s.StartTokenRenewal(renewCtx)

	logger.Info("Starting server on port" + port)
	srv, err := server.NewServer(port, s)
	if err != nil {
		return fmt.Errorf("failed to build server: %w", err)
	}

	// Channel to listen for interrupt signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Run server in a goroutine. A bind failure must reach main rather than
	// leaving it blocked on a signal that will never arrive.
	serverErr := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
			return
		}
		serverErr <- nil
	}()

	// Wait for an interrupt signal or a server failure
	select {
	case <-stop:
		logger.Info("Shutting down server gracefully...")
	case err := <-serverErr:
		if err != nil {
			return fmt.Errorf("server error: %w", err)
		}
		logger.Info("Server stopped serving, shutting down...")
	}

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server shutdown failed", err)
	}

	if err := cache.Close(); err != nil {
		logger.Error("Failed to close Redis connection", err)
	}

	logger.Info("Server stopped")
	return nil
}
