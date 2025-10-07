// Package telemetry contains the logging and metrics
package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics contains all the metrics for the bot
type Metrics interface {
	IncrementSubscriptionCount()
	IncrementRewardCount()
	IncrementFollowCount()
	IncrementCheerCount()
	IncrementAPICallCount()
	IncrementSpotifySongChanged()
	IncrementChatMessageCount()
	SetStreamDuration(seconds float64)
}

// BotMetrics implements Metrics interface
type BotMetrics struct{}

func NewMetrics() *BotMetrics {
	return &BotMetrics{}
}

// IncrementSubscriptionCount counts new subscriptions
func (b BotMetrics) IncrementSubscriptionCount() {
	SubscriptionCount := promauto.NewCounter(prometheus.CounterOpts{
		Name: "subscription_count",
		Help: "Number of subscriptions active",
	})
	SubscriptionCount.Inc()
}

// IncrementRewardCount counts redeemed rewards
func (b BotMetrics) IncrementRewardCount() {
	RewardCount := promauto.NewCounter(prometheus.CounterOpts{
		Name: "reward_count",
		Help: "Number of rewards redeemed",
	})
	RewardCount.Inc()
}

// IncrementFollowCount counts new followers
func (b BotMetrics) IncrementFollowCount() {
	FollowCount := promauto.NewCounter(prometheus.CounterOpts{
		Name: "follow_count",
		Help: "Number of followers",
	})
	FollowCount.Inc()
}

// IncrementCheerCount counts new followers
func (b BotMetrics) IncrementCheerCount() {
	CheerCount := promauto.NewCounter(prometheus.CounterOpts{
		Name: "cheer_count",
		Help: "Number of cheer events",
	})
	CheerCount.Inc()
}

// IncrementAPICallCount counts API calls
func (b BotMetrics) IncrementAPICallCount() {
	APICallCount := promauto.NewCounter(prometheus.CounterOpts{
		Name: "api_count",
		Help: "Number of API calls",
	})
	APICallCount.Inc()
}

// SetStreamDuration tracks how long streams last
func (b BotMetrics) SetStreamDuration(seconds float64) {
	StreamDuration := promauto.NewGauge(prometheus.GaugeOpts{
		Name: "stream_duration_seconds",
		Help: "Duration of streams in seconds",
	})
	StreamDuration.Set(seconds)
}

// IncrementSpotifySongChanged counts the number of times the Spotify song changes
func (b BotMetrics) IncrementSpotifySongChanged() {
	SpotifySongChanged := promauto.NewCounter(prometheus.CounterOpts{
		Name: "spotify_song_changed_count",
		Help: "Number of times the Spotify song changed",
	})
	SpotifySongChanged.Inc()
}

// IncrementChatMessageCount counts the number of chat messages per stream
func (b BotMetrics) IncrementChatMessageCount() {
	ChatMessageCount := promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_message_count",
		Help: "Number of chat messages per stream",
	})
	ChatMessageCount.Inc()
}
