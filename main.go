// package main starts the server
package main

import (
	"log/slog"
	"os"

	"github.com/mvaldes14/twitch-bot/pkgs/server"
)

// NewLogger Returns a logger in json for the bot
func NewLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, nil))
}

func main() {
	logger := NewLogger()
	const port = ":3000"

	srv := server.NewServer(logger, port)

	logger.Info("Starting server on %d", port)
	if err := srv.ListenAndServe(); err != nil {
		logger.Error("Could not start server", "error", err)
	}

}
