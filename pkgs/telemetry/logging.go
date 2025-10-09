// Package telemetry contains the logging and metrics
package telemetry

import (
	"encoding/json"
	"io"
	"os"
	"time"
)

// Logger interface for logging
type Logger interface {
	Log(module string, msg string) io.Writer
	Error(module string, err error) io.Writer
	Chat(module string, msg string) io.Writer
}

// BotLogger implements Logger interface
type BotLogger struct {
	Timestamp string    `json:"timestamp"`
	Level     string    `json:"level"`
	Module    string    `json:"module"`
	Message   any       `json:"message"`
	Output    io.Writer `json:"-"`
}

// NewLogger Returns a logger in json for the bot
func NewLogger(module string) *BotLogger {
	output := io.Writer(os.Stdout)
	return &BotLogger{Module: module, Output: output}
}

// Info logs an info message
func (b BotLogger) Info(msg string) {
	timestamp := time.Now().Format(time.RFC3339)
	event := BotLogger{
		Timestamp: timestamp,
		Level:     "info",
		Message:   msg,
		Module:    b.Module,
	}
	json.NewEncoder(b.Output).Encode(event)
}

// Info logs an error message
func (b BotLogger) Error(msg error) {
	timestamp := time.Now().Format(time.RFC3339)
	event := BotLogger{
		Timestamp: timestamp,
		Level:     "error",
		Message:   msg.Error(),
		Module:    b.Module,
	}
	json.NewEncoder(b.Output).Encode(event)
}

// Chat logs what chat says
func (b BotLogger) Chat(msg string) {
	timestamp := time.Now().Format(time.RFC3339)
	event := BotLogger{
		Timestamp: timestamp,
		Level:     "chat",
		Message:   msg,
		Module:    b.Module,
	}
	json.NewEncoder(b.Output).Encode(event)
}
