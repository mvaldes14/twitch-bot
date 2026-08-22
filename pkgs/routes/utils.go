// package routes handles all routes
package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/mvaldes14/twitch-bot/pkgs/httpclient"
	"github.com/mvaldes14/twitch-bot/pkgs/subscriptions"
)

const (
	userID      = "1792311"
	callbackURL = "https://bots.mvaldes.dev"
	secret      = "superSecret123"
	url         = "https://api.twitch.tv/helix/eventsub/subscriptions"
)

// endpointPaths maps a subscription name to the callback path Twitch should
// call back on. Keys must match the Name field of every entry in
// subscriptionTypes, and values must match the paths registered in
// server.NewServer, or Twitch will call back to a route that does not exist.
var endpointPaths = map[string]string{
	"subscribe": "sub",
	"chat":      "chat",
	"follow":    "follow",
	"cheer":     "cheer",
	"reward":    "reward",
	"streamon":  "stream-online",
	"streamoff": "stream-offline",
}

// MakeRequestMarshallJSON receives a request and marshals the response into a struct
func (rt *Router) MakeRequestMarshallJSON(ctx context.Context, r *RequestJSON, jsonType any) error {
	req, err := http.NewRequestWithContext(ctx, r.Method, r.URL, bytes.NewBuffer([]byte(r.Payload)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	for k, v := range r.Headers {
		req.Header.Set(k, v)
	}
	// Send the request and get the response
	rt.Log.Info("Sending request to Twitch API")
	resp, err := httpclient.Shared.Do(req)
	if err != nil {
		rt.Log.Error("Error sending request to Twitch:", err)
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	return json.Unmarshal(body, jsonType)
}

// GeneratePayload Builds the payload for each subscription type.
// It fails rather than emitting a callback URL with an empty path.
func (rt *Router) GeneratePayload(subType subscriptions.SubscriptionType) (string, error) {
	rt.Log.Info("Generating payload for subscription type")

	// Define the condition based on subscription type
	condition := map[string]string{
		"broadcaster_user_id": userID,
	}

	switch subType.Name {
	case "chat":
		condition["user_id"] = userID
	case "follow":
		condition["moderator_user_id"] = userID
	case "subscribe", "cheer", "reward", "streamon", "streamoff":
	}

	endpointPath, ok := endpointPaths[subType.Name]
	if !ok {
		rt.Log.Error("No callback path registered for subscription name: "+subType.Name, errorUnknownSubscriptionName)
		return "", errorUnknownSubscriptionName
	}

	// Create a struct for the payload
	payloadStruct := struct {
		Type      string            `json:"type"`
		Version   string            `json:"version"`
		Condition map[string]string `json:"condition"`
		Transport struct {
			Method   string `json:"method"`
			Callback string `json:"callback"`
			Secret   string `json:"secret"`
		} `json:"transport"`
	}{
		Type:      subType.Type,
		Version:   subType.Version,
		Condition: condition,
		Transport: struct {
			Method   string `json:"method"`
			Callback string `json:"callback"`
			Secret   string `json:"secret"`
		}{
			Method:   "webhook",
			Callback: fmt.Sprintf("%s/%s", callbackURL, endpointPath),
			Secret:   secret,
		},
	}

	// Marshal the entire payload
	payloadJSON, err := json.Marshal(payloadStruct)
	if err != nil {
		return "", fmt.Errorf("failed to marshal subscription payload: %w", err)
	}
	return string(payloadJSON), nil
}
