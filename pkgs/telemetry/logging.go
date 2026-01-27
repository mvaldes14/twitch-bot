// Package telemetry contains the logging and metrics
package telemetry

import (
	"encoding/json"
	"io"
	"os"
	"time"
)

// CustomLogger is a custom logger that gets a prefix from the package it was called from
type CustomLogger struct {
	module string
	output io.Writer
}

type logInfoMessage struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   any    `json:"message"`
	Module    string `json:"module"`
}

type logErrorMessage struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   any    `json:"message"`
	Module    string `json:"module"`
	Error     string `json:"error"`
}

// NewLogger Returns a logger in json for the bot
func NewLogger(module string) *CustomLogger {
	output := io.Writer(os.Stdout)
	return &CustomLogger{module, output}
}

// Info logs an info message
func (l CustomLogger) Info(msg ...any) {
	timestamp := time.Now().Format(time.RFC3339)
	event := logInfoMessage{
		Timestamp: timestamp,
		Level:     "info",
		Message:   msg,
		Module:    l.module,
	}
	_ = json.NewEncoder(l.output).Encode(event)
}

// Error logs an error message
func (l CustomLogger) Error(msg string, e error) {
	timestamp := time.Now().Format(time.RFC3339)
	event := logErrorMessage{
		Timestamp: timestamp,
		Level:     "error",
		Message:   msg,
		Module:    l.module,
		Error:     e.Error(),
	}
	_ = json.NewEncoder(l.output).Encode(event)
}

// Chat logs a chat message
func (l CustomLogger) Chat(msg string) {
	timestamp := time.Now().Format(time.RFC3339)
	event := logErrorMessage{
		Timestamp: timestamp,
		Level:     "chat",
		Message:   msg,
		Module:    l.module,
	}
	_ = json.NewEncoder(l.output).Encode(event)
}
