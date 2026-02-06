// Package telemetry contains the logging and metrics
package telemetry

import (
	"encoding/json"
	"fmt"
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
	Timestamp string         `json:"timestamp"`
	Level     string         `json:"level"`
	Message   any            `json:"message"`
	Module    string         `json:"module"`
	Body      string         `json:"body,omitempty"`
	Operation string         `json:"operation,omitempty"`
	Status    string         `json:"status,omitempty"`
	Details   map[string]any `json:"details,omitempty"`
}

type logErrorMessage struct {
	Timestamp string         `json:"timestamp"`
	Level     string         `json:"level"`
	Message   any            `json:"message"`
	Module    string         `json:"module"`
	Error     string         `json:"error"`
	Body      string         `json:"body,omitempty"`
	Operation string         `json:"operation,omitempty"`
	Details   map[string]any `json:"details,omitempty"`
}

// NewLogger Returns a logger in json for the bot
func NewLogger(module string) *CustomLogger {
	output := io.Writer(os.Stdout)
	return &CustomLogger{module, output}
}

// parseStructuredLog extracts structured data from formatted log messages
// Supports formats like: "[TAG] message" or "[TAG: value] message"
// Returns the parsed body/operation/status for dashboard consumption
func parseStructuredLog(msg string) (body string, operation string, status string, details map[string]any) {
	details = make(map[string]any)
	body = msg

	// Extract tags like [SOURCE: ENV VAR], [CACHE HIT], etc.
	if len(msg) > 0 && msg[0] == '[' {
		if closeIdx := findClosingBracket(msg); closeIdx > 0 {
			tag := msg[1:closeIdx]
			body = msg[closeIdx+2:] // Skip "] "

			// Parse tag format: "TAG" or "TAG: VALUE"
			if colonIdx := findColon(tag); colonIdx > 0 {
				operation = tag[:colonIdx]
				status = tag[colonIdx+2:] // Skip ": "
				details["tag"] = tag
				details["operation"] = operation
				details["status"] = status
			} else {
				operation = tag
				details["tag"] = tag
				details["operation"] = operation
			}
		}
	}

	return
}

func findClosingBracket(s string) int {
	for i := 1; i < len(s); i++ {
		if s[i] == ']' {
			return i
		}
	}
	return -1
}

func findColon(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == ':' {
			return i
		}
	}
	return -1
}

// Info logs an info message with structured JSON output
func (l CustomLogger) Info(msg ...any) {
	timestamp := time.Now().Format(time.RFC3339)

	// Convert message to string for parsing
	var msgStr string
	if len(msg) > 0 {
		msgStr = fmt.Sprint(msg[0])
	}

	// Parse structured content
	body, operation, status, details := parseStructuredLog(msgStr)

	event := logInfoMessage{
		Timestamp: timestamp,
		Level:     "info",
		Message:   msg,
		Module:    l.module,
		Body:      body,
		Operation: operation,
		Status:    status,
		Details:   details,
	}
	_ = json.NewEncoder(l.output).Encode(event)
}

// Error logs an error message with structured JSON output
func (l CustomLogger) Error(msg string, e error) {
	timestamp := time.Now().Format(time.RFC3339)

	// Parse structured content from error message
	body, operation, _, details := parseStructuredLog(msg)

	event := logErrorMessage{
		Timestamp: timestamp,
		Level:     "error",
		Message:   msg,
		Module:    l.module,
		Error:     e.Error(),
		Body:      body,
		Operation: operation,
		Details:   details,
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
