// Package Service is the base layout for all other services and their common functionality
package service

import (
	"net/http"
	"time"

	"github.com/mvaldes14/twitch-bot/pkgs/telemetry"
)

// Service defines the common attributes for all other services
type Service struct {
	Logger  *telemetry.BotLogger
	Metrics *telemetry.BotMetrics
	Client  *http.Client
}

// NewService starts and returns the common things for any services
func NewService(module string) *Service {
	return &Service{
		Logger:  telemetry.NewLogger(module),
		Metrics: telemetry.NewMetrics(),
		Client:  &http.Client{Timeout: 60 * time.Second},
	}
}
