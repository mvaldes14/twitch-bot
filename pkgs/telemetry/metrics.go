// Package telemetry contains the logging and metrics
package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics contains all the metrics for the bot
type Metrics interface {
	IncrementCount(name, description string)
}

// BotMetrics implements Metrics interface
type BotMetrics struct {
	Name        string
	Description string
}

// NewMetrics returns a new Metric Service
func NewMetrics() *BotMetrics {
	return &BotMetrics{}
}

// IncrementCount defines the default way to increment a counter in a prometheus metric
func (b BotMetrics) IncrementCount(name, description string) {
	incrementCounter := promauto.NewCounter(prometheus.CounterOpts{
		Name: name,
		Help: description,
	})
	incrementCounter.Inc()
}

// Existing metric names to migrate
// Name: "subscription_count",
// Name: "reward_count",
// Name: "follow_count",
// Name: "cheer_count",
// Name: "api_count",
// Name: "stream_duration_seconds",
// Name: "spotify_song_changed_count",
// Name: "chat_message_count",
