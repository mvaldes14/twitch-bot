// Package telemetry contains the logging and metrics
package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var (
	meter = otel.Meter("twitch-bot")

	// SubscriptionCount counts new subscriptions
	SubscriptionCount metric.Int64Counter
	// RewardCount counts redeemed rewards
	RewardCount metric.Int64Counter
	// FollowCount counts new followers
	FollowCount metric.Int64Counter
	// CheerCount counts new followers
	CheerCount metric.Int64Counter
	// APICallCount counts API calls
	APICallCount metric.Int64Counter
	// StreamDuration tracks how long streams last
	StreamDuration metric.Float64Gauge
	// SpotifySongChanged counts the number of times the Spotify song changes
	SpotifySongChanged metric.Int64Counter
	// ChatMessageCount counts the number of chat messages per stream
	ChatMessageCount metric.Int64Counter
)

// InitMetrics initializes all OTEL metrics
func InitMetrics() error {
	var err error

	SubscriptionCount, err = meter.Int64Counter(
		"subscription_count",
		metric.WithDescription("Number of subscriptions active"),
	)
	if err != nil {
		return err
	}

	RewardCount, err = meter.Int64Counter(
		"reward_count",
		metric.WithDescription("Number of rewards redeemed"),
	)
	if err != nil {
		return err
	}

	FollowCount, err = meter.Int64Counter(
		"follow_count",
		metric.WithDescription("Number of followers"),
	)
	if err != nil {
		return err
	}

	CheerCount, err = meter.Int64Counter(
		"cheer_count",
		metric.WithDescription("Number of cheer events"),
	)
	if err != nil {
		return err
	}

	APICallCount, err = meter.Int64Counter(
		"api_count",
		metric.WithDescription("Number of API calls"),
	)
	if err != nil {
		return err
	}

	StreamDuration, err = meter.Float64Gauge(
		"stream_duration_seconds",
		metric.WithDescription("Duration of streams in seconds"),
	)
	if err != nil {
		return err
	}

	SpotifySongChanged, err = meter.Int64Counter(
		"spotify_song_changed_count",
		metric.WithDescription("Number of times the Spotify song changed"),
	)
	if err != nil {
		return err
	}

	ChatMessageCount, err = meter.Int64Counter(
		"chat_message_count",
		metric.WithDescription("Number of chat messages per stream"),
	)
	if err != nil {
		return err
	}

	return nil
}

// Helper functions to simplify metric recording
func IncrementSubscriptionCount(ctx context.Context) {
	if SubscriptionCount != nil {
		SubscriptionCount.Add(ctx, 1)
	}
}

func IncrementRewardCount(ctx context.Context) {
	if RewardCount != nil {
		RewardCount.Add(ctx, 1)
	}
}

func IncrementFollowCount(ctx context.Context) {
	if FollowCount != nil {
		FollowCount.Add(ctx, 1)
	}
}

func IncrementCheerCount(ctx context.Context) {
	if CheerCount != nil {
		CheerCount.Add(ctx, 1)
	}
}

func IncrementAPICallCount(ctx context.Context) {
	if APICallCount != nil {
		APICallCount.Add(ctx, 1)
	}
}

func RecordStreamDuration(ctx context.Context, duration float64) {
	if StreamDuration != nil {
		StreamDuration.Record(ctx, duration)
	}
}

func IncrementSpotifySongChanged(ctx context.Context) {
	if SpotifySongChanged != nil {
		SpotifySongChanged.Add(ctx, 1)
	}
}

func IncrementChatMessageCount(ctx context.Context) {
	if ChatMessageCount != nil {
		ChatMessageCount.Add(ctx, 1)
	}
}
