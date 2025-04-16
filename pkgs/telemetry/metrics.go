// Package telemetry contains the logging and metrics
package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// SubscriptionCount counts new subscriptions
	SubscriptionCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "subscription_count",
		Help: "Number of subscriptions active",
	})
	// RewardCount counts redeemed rewards
	RewardCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "reward_count",
		Help: "Number of rewards redeemed",
	})
	// FollowCount counts new followers
	FollowCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "follow_count",
		Help: "Number of followers",
	})
	// CheerCount counts new followers
	CheerCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cheer_count",
		Help: "Number of cheer events",
	})
	// APICallCount counts API calls
	APICallCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "api_count",
		Help: "Number of API calls",
	})
)
