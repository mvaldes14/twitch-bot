package routes

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mvaldes14/twitch-bot/pkgs/subscriptions"
	"github.com/mvaldes14/twitch-bot/pkgs/telemetry"
	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

type fakeActions struct{}

func (fakeActions) ParseMessage(context.Context, subscriptions.ChatMessageEvent) {}
func (fakeActions) SendMessage(context.Context, string) error                    { return nil }

func setupMetricTest(t *testing.T) *sdkmetric.ManualReader {
	t.Helper()

	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	otel.SetMeterProvider(provider)
	t.Cleanup(func() {
		_ = provider.Shutdown(context.Background())
	})

	if err := telemetry.InitMetrics(); err != nil {
		t.Fatalf("InitMetrics() returned error: %v", err)
	}

	return reader
}

func collectInt64Metric(t *testing.T, reader *sdkmetric.ManualReader, name string) int64 {
	t.Helper()

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("Collect() returned error: %v", err)
	}

	for _, scope := range rm.ScopeMetrics {
		for _, metric := range scope.Metrics {
			if metric.Name != name {
				continue
			}
			sum, ok := metric.Data.(metricdata.Sum[int64])
			if !ok {
				t.Fatalf("metric %s has type %T, want metricdata.Sum[int64]", name, metric.Data)
			}
			var total int64
			for _, point := range sum.DataPoints {
				total += point.Value
			}
			return total
		}
	}
	return 0
}

func TestFollowHandlerIncrementsFollowMetricOnValidEvent(t *testing.T) {
	reader := setupMetricTest(t)
	rt := &Router{Log: telemetry.NewLogger("routes_test"), Actions: fakeActions{}}

	body := `{"event":{"user_name":"new_follower"}}`
	req := httptest.NewRequest("POST", "/follow", strings.NewReader(body))

	rt.FollowHandler(httptest.NewRecorder(), req)

	if got := collectInt64Metric(t, reader, "twitch.follow_count"); got != 1 {
		t.Fatalf("twitch.follow_count = %d, want 1", got)
	}
}

func TestSubHandlerIncrementsSubscriptionMetricOnValidEvent(t *testing.T) {
	reader := setupMetricTest(t)
	rt := &Router{Log: telemetry.NewLogger("routes_test"), Actions: fakeActions{}}

	body := `{"event":{"user_name":"new_subscriber"}}`
	req := httptest.NewRequest("POST", "/sub", strings.NewReader(body))

	rt.SubHandler(httptest.NewRecorder(), req)

	if got := collectInt64Metric(t, reader, "twitch.subscription_count"); got != 1 {
		t.Fatalf("twitch.subscription_count = %d, want 1", got)
	}
}

func TestFollowAndSubHandlersDoNotIncrementMetricForInvalidPayload(t *testing.T) {
	reader := setupMetricTest(t)
	rt := &Router{Log: telemetry.NewLogger("routes_test"), Actions: fakeActions{}}

	rt.FollowHandler(httptest.NewRecorder(), httptest.NewRequest("POST", "/follow", strings.NewReader(`{`)))
	rt.SubHandler(httptest.NewRecorder(), httptest.NewRequest("POST", "/sub", strings.NewReader(`{`)))

	if got := collectInt64Metric(t, reader, "twitch.follow_count"); got != 0 {
		t.Fatalf("twitch.follow_count = %d, want 0", got)
	}
	if got := collectInt64Metric(t, reader, "twitch.subscription_count"); got != 0 {
		t.Fatalf("twitch.subscription_count = %d, want 0", got)
	}
}
