package routes

import (
	"encoding/json"
	"testing"

	"github.com/mvaldes14/twitch-bot/pkgs/subscriptions"
	"github.com/mvaldes14/twitch-bot/pkgs/telemetry"
)

func testRouter() *Router {
	return &Router{Log: telemetry.NewLogger("routes_test")}
}

// TestSubscriptionTypesHaveCallbackPaths guards the pairing between the
// subscription configs and the callback paths. A config whose Name has no entry
// in endpointPaths silently produced a callback URL with no path, which made
// Twitch challenge verification fail for that subscription.
func TestSubscriptionTypesHaveCallbackPaths(t *testing.T) {
	for requested, config := range subscriptionTypes {
		if _, ok := endpointPaths[config.Name]; !ok {
			t.Errorf("subscription type %q has Name %q with no entry in endpointPaths", requested, config.Name)
		}
	}
}

func TestRewardSubscriptionIsNotRegistered(t *testing.T) {
	if _, ok := subscriptionTypes["reward"]; ok {
		t.Fatal("reward EventSub registration should not be available because the bot no longer handles reward-based song controls")
	}
	if _, ok := endpointPaths["reward"]; ok {
		t.Fatal("reward callback endpoint should not be registered")
	}
}

func TestGeneratePayloadCallback(t *testing.T) {
	rt := testRouter()

	tests := []struct {
		name         string
		requestType  string
		wantCallback string
	}{
		{"chat", "chat", "https://bots.mvaldes.dev/chat"},
		{"follow", "follow", "https://bots.mvaldes.dev/follow"},
		{"subscription", "subscription", "https://bots.mvaldes.dev/sub"},
		{"cheer", "cheer", "https://bots.mvaldes.dev/cheer"},
		{"stream online", "streamon", "https://bots.mvaldes.dev/stream-online"},
		{"stream offline", "streamoff", "https://bots.mvaldes.dev/stream-offline"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, ok := subscriptionTypes[tt.requestType]
			if !ok {
				t.Fatalf("no subscription config registered for %q", tt.requestType)
			}

			payload, err := rt.GeneratePayload(config)
			if err != nil {
				t.Fatalf("GeneratePayload(%q) returned error: %v", tt.requestType, err)
			}

			var got struct {
				Transport struct {
					Callback string `json:"callback"`
				} `json:"transport"`
			}
			if err := json.Unmarshal([]byte(payload), &got); err != nil {
				t.Fatalf("payload is not valid JSON: %v", err)
			}

			if got.Transport.Callback != tt.wantCallback {
				t.Errorf("callback = %q, want %q", got.Transport.Callback, tt.wantCallback)
			}
		})
	}
}

func TestGeneratePayloadUnknownNameFails(t *testing.T) {
	rt := testRouter()

	_, err := rt.GeneratePayload(subscriptions.SubscriptionType{
		Name:    "nonexistent",
		Version: "1",
		Type:    "some.event",
	})
	if err == nil {
		t.Fatal("GeneratePayload accepted an unknown subscription name; it must fail rather than emit an empty callback path")
	}
}
