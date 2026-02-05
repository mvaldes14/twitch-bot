// package main starts the server
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mvaldes14/twitch-bot/pkgs/secrets"
	"github.com/mvaldes14/twitch-bot/pkgs/server"
	"github.com/mvaldes14/twitch-bot/pkgs/telemetry"
)

const port = ":3000"

func main() {
	ctx := context.Background()
	logger := telemetry.NewLogger("main")

	// Initialize OpenTelemetry
	otelConfig := telemetry.GetConfigFromEnv()
	if err := telemetry.InitOTEL(ctx, otelConfig); err != nil {
		logger.Error("Failed to initialize OpenTelemetry", err)
		os.Exit(1)
	}
	logger.Info("OpenTelemetry initialized successfully")

	// Initialize OTEL metrics
	if err := telemetry.InitMetrics(); err != nil {
		logger.Error("Failed to initialize metrics", err)
		os.Exit(1)
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

	s := secrets.NewSecretService()
	s.InitSecrets()

	// Start background token renewal (cancelled on shutdown)
	renewCtx, renewCancel := context.WithCancel(ctx)
	s.StartTokenRenewal(renewCtx)

	logger.Info("Starting server on port" + port)
	srv := server.NewServer(port)

	// Channel to listen for interrupt signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Run server in a goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			logger.Error("Server error", err)
		}
	}()

	// Wait for interrupt signal
	<-stop
	logger.Info("Shutting down server gracefully...")

	// Stop the token renewal goroutine
	renewCancel()

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server shutdown failed", err)
	}

	logger.Info("Server stopped")
}
