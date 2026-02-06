// Package telemetry contains the logging and metrics
package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	meter = otel.Meter("twitch.bot")

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

	// Token lifecycle metrics
	TokenRefreshTotal    metric.Int64Counter
	TokenRefreshOn401    metric.Int64Counter
	TokenValidationTotal metric.Int64Counter
	TokenTTLSeconds      metric.Float64Gauge

	// Cache metrics
	CacheOperationTotal metric.Int64Counter

	// Command and message metrics
	CommandExecutedTotal metric.Int64Counter
	MessageSentTotal     metric.Int64Counter

	// Notification metrics
	NotificationSentTotal metric.Int64Counter

	// Spotify operation metrics
	SpotifyOperationTotal metric.Int64Counter
)

// InitMetrics initializes all OTEL metrics
func InitMetrics() error {
	var err error

	SubscriptionCount, err = meter.Int64Counter(
		"twitch.subscription_count",
		metric.WithDescription("Number of subscriptions active"),
	)
	if err != nil {
		return err
	}

	RewardCount, err = meter.Int64Counter(
		"twitch.reward_count",
		metric.WithDescription("Number of rewards redeemed"),
	)
	if err != nil {
		return err
	}

	FollowCount, err = meter.Int64Counter(
		"twitch.follow_count",
		metric.WithDescription("Number of followers"),
	)
	if err != nil {
		return err
	}

	CheerCount, err = meter.Int64Counter(
		"twitch.cheer_count",
		metric.WithDescription("Number of cheer events"),
	)
	if err != nil {
		return err
	}

	APICallCount, err = meter.Int64Counter(
		"twitch.api_count",
		metric.WithDescription("Number of API calls"),
	)
	if err != nil {
		return err
	}

	StreamDuration, err = meter.Float64Gauge(
		"twitch.stream_duration_seconds",
		metric.WithDescription("Duration of streams in seconds"),
	)
	if err != nil {
		return err
	}

	SpotifySongChanged, err = meter.Int64Counter(
		"twitch.spotify_song_changed_count",
		metric.WithDescription("Number of times the Spotify song changed"),
	)
	if err != nil {
		return err
	}

	ChatMessageCount, err = meter.Int64Counter(
		"twitch.chat_message_count",
		metric.WithDescription("Number of chat messages per stream"),
	)
	if err != nil {
		return err
	}

	// Token lifecycle metrics
	TokenRefreshTotal, err = meter.Int64Counter(
		"twitch.token_refresh_total",
		metric.WithDescription("Total token refresh attempts by type and result"),
	)
	if err != nil {
		return err
	}

	TokenRefreshOn401, err = meter.Int64Counter(
		"twitch.token_refresh_on_401_total",
		metric.WithDescription("Token refreshes triggered by 401 responses"),
	)
	if err != nil {
		return err
	}

	TokenValidationTotal, err = meter.Int64Counter(
		"twitch.token_validation_total",
		metric.WithDescription("Token validation checks by type and validity"),
	)
	if err != nil {
		return err
	}

	TokenTTLSeconds, err = meter.Float64Gauge(
		"twitch.token_ttl_seconds",
		metric.WithDescription("Remaining TTL of tokens in seconds"),
	)
	if err != nil {
		return err
	}

	// Cache metrics
	CacheOperationTotal, err = meter.Int64Counter(
		"twitch.cache_operation_total",
		metric.WithDescription("Cache operations by type and result"),
	)
	if err != nil {
		return err
	}

	// Command and message metrics
	CommandExecutedTotal, err = meter.Int64Counter(
		"twitch.command_executed_total",
		metric.WithDescription("Chat commands executed by command name"),
	)
	if err != nil {
		return err
	}

	MessageSentTotal, err = meter.Int64Counter(
		"twitch.message_sent_total",
		metric.WithDescription("Messages sent to Twitch chat by result"),
	)
	if err != nil {
		return err
	}

	// Notification metrics
	NotificationSentTotal, err = meter.Int64Counter(
		"twitch.notification_sent_total",
		metric.WithDescription("Notifications sent by service and result"),
	)
	if err != nil {
		return err
	}

	// Spotify operation metrics
	SpotifyOperationTotal, err = meter.Int64Counter(
		"twitch.spotify_operation_total",
		metric.WithDescription("Spotify API operations by type and result"),
	)
	if err != nil {
		return err
	}

	return nil
}

// IncrementSubscriptionCount records a new subscription event.
func IncrementSubscriptionCount(ctx context.Context) {
	if SubscriptionCount != nil {
		SubscriptionCount.Add(ctx, 1)
	}
}

// IncrementRewardCount records a channel point reward redemption.
func IncrementRewardCount(ctx context.Context) {
	if RewardCount != nil {
		RewardCount.Add(ctx, 1)
	}
}

// IncrementFollowCount records a new follower event.
func IncrementFollowCount(ctx context.Context) {
	if FollowCount != nil {
		FollowCount.Add(ctx, 1)
	}
}

// IncrementCheerCount records a cheer/bits event.
func IncrementCheerCount(ctx context.Context) {
	if CheerCount != nil {
		CheerCount.Add(ctx, 1)
	}
}

// IncrementAPICallCount records an API call.
func IncrementAPICallCount(ctx context.Context) {
	if APICallCount != nil {
		APICallCount.Add(ctx, 1)
	}
}

// RecordStreamDuration records the duration of a stream in seconds.
func RecordStreamDuration(ctx context.Context, duration float64) {
	if StreamDuration != nil {
		StreamDuration.Record(ctx, duration)
	}
}

// IncrementSpotifySongChanged records a Spotify song change event.
func IncrementSpotifySongChanged(ctx context.Context) {
	if SpotifySongChanged != nil {
		SpotifySongChanged.Add(ctx, 1)
	}
}

// IncrementChatMessageCount records a chat message.
func IncrementChatMessageCount(ctx context.Context) {
	if ChatMessageCount != nil {
		ChatMessageCount.Add(ctx, 1)
	}
}

// IncrementTokenRefreshTotal records a token refresh attempt with type and result labels.
func IncrementTokenRefreshTotal(ctx context.Context, tokenType, result string) {
	if TokenRefreshTotal != nil {
		TokenRefreshTotal.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("token_type", tokenType),
				attribute.String("result", result),
			),
		)
	}
}

// IncrementTokenRefreshOn401 records a token refresh triggered by a 401 response.
func IncrementTokenRefreshOn401(ctx context.Context, operation string) {
	if TokenRefreshOn401 != nil {
		TokenRefreshOn401.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("operation", operation),
			),
		)
	}
}

// IncrementTokenValidationTotal records a token validation check with type and validity labels.
func IncrementTokenValidationTotal(ctx context.Context, tokenType string, valid bool) {
	if TokenValidationTotal != nil {
		TokenValidationTotal.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("token_type", tokenType),
				attribute.Bool("valid", valid),
			),
		)
	}
}

// RecordTokenTTL records the remaining TTL of a token in seconds.
func RecordTokenTTL(ctx context.Context, tokenType string, ttlSeconds float64) {
	if TokenTTLSeconds != nil {
		TokenTTLSeconds.Record(ctx, ttlSeconds,
			metric.WithAttributes(
				attribute.String("token_type", tokenType),
			),
		)
	}
}

// IncrementCacheOperation records a Redis cache operation with operation type and result labels.
func IncrementCacheOperation(ctx context.Context, operation, result string) {
	if CacheOperationTotal != nil {
		CacheOperationTotal.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("operation", operation),
				attribute.String("result", result),
			),
		)
	}
}

// IncrementCommandExecuted records a chat command execution with the command name label.
func IncrementCommandExecuted(ctx context.Context, command string) {
	if CommandExecutedTotal != nil {
		CommandExecutedTotal.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("command", command),
			),
		)
	}
}

// IncrementMessageSent records a Twitch chat message send attempt with result label.
func IncrementMessageSent(ctx context.Context, result string) {
	if MessageSentTotal != nil {
		MessageSentTotal.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("result", result),
			),
		)
	}
}

// IncrementNotificationSent records a notification delivery attempt with service and result labels.
func IncrementNotificationSent(ctx context.Context, service, result string) {
	if NotificationSentTotal != nil {
		NotificationSentTotal.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("service", service),
				attribute.String("result", result),
			),
		)
	}
}

// IncrementSpotifyOperation records a Spotify API operation with operation type and result labels.
func IncrementSpotifyOperation(ctx context.Context, operation, result string) {
	if SpotifyOperationTotal != nil {
		SpotifyOperationTotal.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("operation", operation),
				attribute.String("result", result),
			),
		)
	}
}
