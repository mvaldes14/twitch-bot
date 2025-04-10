// Package telemetry contains the logging and metrics
package telemetry

import (
	"fmt"
	"log/slog"
	"os"
)

// TODO: Redo this with our custom logging implementation for INFO,ERROR in JSON
type Log struct{}

// NewLogger Returns a logger in json for the bot
func NewLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, nil))
}

func (l Log) Info(msg string) {
	fmt.Sprintf("INFO: %v", msg)
}
