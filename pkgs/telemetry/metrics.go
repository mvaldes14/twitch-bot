// Package telemetry contains the logging and metrics
package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// TestMetric is a test metric to check instrumentation
	TestMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "test_metric",
		Help: "A test metric",
	})
	SubscriptionCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "subscription_count",
		Help: "Number of subscriptions active",
	})
)
