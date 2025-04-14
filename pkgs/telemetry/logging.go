// Package telemetry contains the logging and metrics
package telemetry

import (
	"encoding/json"
	"io"
	"os"
	"time"
)

// Logger is a custom logger that gets a prefix from the package it was called from
type CustomLogger struct {
	module string
	output io.Writer
}

type logMessage struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   any    `json:"message"`
	Module    string `json:"module"`
	Error     error  `json:"error,omitempty"`
}

// NewLogger Returns a logger in json for the bot
func NewLogger(module string) *CustomLogger {
	output := io.Writer(os.Stdout)
	return &CustomLogger{module, output}
}

// Info logs an info message
func (l CustomLogger) Info(msg any) {
	timestamp := time.Now().Format(time.RFC3339)
	event := logMessage{
		Timestamp: timestamp,
		Level:     "info",
		Message:   msg,
		Module:    l.module,
	}
	json.NewEncoder(l.output).Encode(event)
}

// Info logs an info message
func (l CustomLogger) Error(msg any, err error) {
	timestamp := time.Now().Format(time.RFC3339)
	event := logMessage{
		Timestamp: timestamp,
		Level:     "error",
		Message:   msg,
		Module:    l.module,
		Error:     err,
	}
	json.NewEncoder(l.output).Encode(event)
}
