// Package telemetry contains the logging and metrics
package telemetry

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

// Logger is a custom logger that gets a prefix from the package it was called from
type CustomLogger struct {
	module string
	logger *log.Logger
}

type logMessage struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	Module    string `json:"module"`
	Error     error  `json:"error,omitempty"`
}

// example log format
//{time:unixRFC, loglevel:info/error, msg:any }

// NewLogger Returns a logger in json for the bot
func NewLogger(module string) *CustomLogger {
	logger := log.New(os.Stdout, module, 0)
	return &CustomLogger{module, logger}
}

// Info logs an info message
func (l CustomLogger) Info(msg string, fields ...any) {
	timestamp := time.Now().Format(time.RFC3339)
	event := logMessage{
		Timestamp: timestamp,
		Level:     "info",
		Message:   msg,
		Module:    l.module,
	}
	json.NewEncoder(l.logger.Writer()).Encode(event)
}

// Info logs an info message
func (l CustomLogger) Error(msg string, err error) {
	timestamp := time.Now().Format(time.RFC3339)
	event := logMessage{
		Timestamp: timestamp,
		Level:     "error",
		Message:   msg,
		Module:    l.module,
		Error:     err,
	}
	json.NewEncoder(l.logger.Writer()).Encode(event)
}
