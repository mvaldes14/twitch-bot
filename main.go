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

	logger.Info("Starting server on port" + port)
	if err := srv.ListenAndServe(); err != nil {
		logger.Error("Could not start server", err)
	}
}
