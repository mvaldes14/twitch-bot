// package routes handles all routes
package routes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/mvaldes14/twitch-bot/pkgs/subscriptions"
)

const (
	userID      = "1792311"
	callbackURL = "https://bots.mvaldes.dev"
	secret      = "superSecret123"
	url         = "https://api.twitch.tv/helix/eventsub/subscriptions"
)

// MakeRequestMarshallJSON receives a request and marshals the response into a struct
func (rt *Router) MakeRequestMarshallJSON(r *RequestJSON, jsonType any) error {
	req, err := http.NewRequest(r.Method, r.URL, bytes.NewBuffer([]byte(r.Payload)))
	if err != nil {
		return nil
	}
	for k, v := range r.Headers {
		req.Header.Set(k, v)
	}
	// Create an HTTP client
	client := &http.Client{}
	// Send the request and get the response
	rt.Log.Info("Sending request to Twitch API")
	resp, err := client.Do(req)
	if err != nil {
		rt.Log.Error("Error sending request to Twitch:", err)
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return json.Unmarshal(body, jsonType)
}

// GeneratePayload Builds the payload for each subscription type
func (rt *Router) GeneratePayload(subType subscriptions.SubscriptionType) string {
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
	case "subscribe", "cheer", "reward", "stream":
	}

	// Map subscription names to their endpoint paths
	endpointPath := map[string]string{
		"subscribe": "sub",
		"chat":      "chat",
		"follow":    "follow",
		"cheer":     "cheer",
		"reward":    "reward",
		"stream":    "stream",
	}[subType.Name]

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
	payloadJSON, _ := json.Marshal(payloadStruct)
	var payload string
	payload = string(payloadJSON)
	return payload
}
