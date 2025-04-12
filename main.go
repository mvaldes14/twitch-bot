// package main starts the server
package main

import (
	"github.com/mvaldes14/twitch-bot/pkgs/server"
	"github.com/mvaldes14/twitch-bot/pkgs/telemetry"
)

func main() {
	const port = ":3000"
	logger := telemetry.NewLogger("main")

	srv := server.NewServer(port)

	// logger.Info("INFO", "Starting server on port", port)
	logger.Info("Starting server")
	if err := srv.ListenAndServe(); err != nil {
		// logger.Error("Could not start server", "error", err)
		logger.Info("Error Starting server")

	}
}
