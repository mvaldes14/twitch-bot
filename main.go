// package main starts the server
package main

import (
	"github.com/mvaldes14/twitch-bot/pkgs/secrets"

	"github.com/mvaldes14/twitch-bot/pkgs/server"
	"github.com/mvaldes14/twitch-bot/pkgs/telemetry"
)

const port = ":3000"

func main() {
	logger := telemetry.NewLogger("main")
	s := secrets.NewSecretService()
	s.InitSecrets()

	logger.Info("Starting server on port" + port)
	srv := server.NewServer(port)
	if err := srv.ListenAndServe(); err != nil {
		logger.Error("Could not start server", err)
	}

}
