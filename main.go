// package main starts the server
package main

import (
	"github.com/mvaldes14/twitch-bot/pkgs/server"
	"github.com/mvaldes14/twitch-bot/pkgs/telemetry"
)

func main() {
	const port = ":3000"
	logger := telemetry.NewLogger()

	srv := server.NewServer(logger, port)

	logger.Info("INFO", "Starting server on port", port)
	if err := srv.ListenAndServe(); err != nil {
		logger.Error("Could not start server", "error", err)
	}
}
