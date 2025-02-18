// Package utils Holds all of the utilities used by the bot
package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/mvaldes14/twitch-bot/pkgs/types"
)

const (
	userID      = "1792311"
	callbackURL = "https://bots.mvaldes.dev"
	secret      = "superSecret123"
	url         = "https://api.twitch.tv/helix/eventsub/subscriptions"
)

// Logger Returns a logger in json for the bot
func Logger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, nil))
}

// MakeRequestMarshallJson receives a request and marshals the response into a struct
func MakeRequestMarshallJson(r *types.RequestJson, jsonType interface{}) error {
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
	logger.Info("Sending request")
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("Error", "Sending request:", err)
		return nil
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return json.Unmarshal(body, jsonType)
}

// BuildSecretHeaders Returns the secrets from env variables to build headers for requests
func BuildSecretHeaders() types.RequestHeader {
	token := os.Getenv("TWITCH_TOKEN")
	clientID := os.Getenv("TWITCH_CLIENT_ID")
	return types.RequestHeader{
		Token:    token,
		ClientID: clientID,
	}
}

// GeneratePayload Builds the payload for each subscription type
func GeneratePayload(subType types.SubscriptionType) string {
	var payload string

	// TODO: Redo this with a generic payload, its repetitive
	// Also include the callback endpoint to form the webhook

	switch subType.Name {
	case "chat":
		payload = fmt.Sprintf(`{
      "type":"%v",
      "version":"%v",
      "condition":{
        "broadcaster_user_id":"%v",
        "user_id": "%v"
      },
      "transport": {
        "method":"webhook",
        "callback":"%v/chat","secret":"%v"
      }
    }`, subType.Type, subType.Version, userID, userID, callbackURL, secret)

	case "follow":
		payload = fmt.Sprintf(`{
      "type": "%v",
      "version": "%v",
      "condition": {
        "broadcaster_user_id": "%v",
        "moderator_user_id": "%v"
      },
      "transport": {
        "method": "webhook",
        "callback": "%v/follow",
        "secret": "%v"
      }
    }`, subType.Type, subType.Version, userID, userID, callbackURL, secret)

	case "subscribe":
		payload = fmt.Sprintf(`{
      "type": "%v",
      "version": "%v",
      "condition": {
          "broadcaster_user_id": "%v"
      },
      "transport": {
          "method": "webhook",
          "callback": "%v/sub",
          "secret": "%v"
      }
    }`, subType.Type, subType.Version, userID, callbackURL, secret)

	case "cheer":
		payload = fmt.Sprintf(`{
      "type": "%v",
      "version": "%v",
      "condition": {
          "broadcaster_user_id": "%v"
      },
      "transport": {
          "method": "webhook",
          "callback": "%v/cheer",
          "secret": "%v"
      }
    }`, subType.Type, subType.Version, userID, callbackURL, secret)

	case "reward":
		payload = fmt.Sprintf(`{
      "type": "%v",
      "version": "%v",
      "condition": {
          "broadcaster_user_id": "%v"
      },
      "transport": {
          "method": "webhook",
          "callback": "%v/reward",
          "secret": "%v"
      }
    }`, subType.Type, subType.Version, userID, callbackURL, secret)

	case "stream":
		payload = fmt.Sprintf(`{
      "type": "%v",
      "version": "%v",
      "condition": {
          "broadcaster_user_id": "%v"
      },
      "transport": {
          "method": "webhook",
          "callback": "%v/stream",
          "secret": "%v"
      }
    }`, subType.Type, subType.Version, userID, callbackURL, secret)
	}
	return payload
}
