// Package subscriptions Handles all subscriptions actions in Twitch
package subscriptions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/mvaldes14/twitch-bot/pkgs/secrets"
	"github.com/mvaldes14/twitch-bot/pkgs/telemetry"
)

// URl endpoint for all twitch subscriptions
const url = "https://api.twitch.tv/helix/eventsub/subscriptions"

// type Subscription is the struct that handles all subscriptions
type Subscription struct {
	Secrets *secrets.SecretService
	Log     *telemetry.CustomLogger
}

// NewSubscription creates a new subscription
func NewSubscription(secretService *secrets.SecretService) *Subscription {
	log := telemetry.NewLogger("subscriptions")
	return &Subscription{
		Secrets: secretService,
		Log:     log,
	}
}

// CreateSubscription Generates  a new subscription on an event type
func (s *Subscription) CreateSubscription(payload string) *http.Response {
	// subscribe to eventsub
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(payload)))
	if err != nil {
		return nil
	}
	// Add key headers to request
	headers := s.Secrets.BuildSecretHeaders()
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+headers.Token)
	req.Header.Set("Client-Id", headers.ClientID)
	// Create an HTTP client
	client := &http.Client{}
	// Send the request and get the response
	s.Log.Info("Sending request")
	resp, err := client.Do(req)
	if err != nil {
		s.Log.Error("Error sending request:", err)
		return nil
	}
	defer resp.Body.Close()
	s.Log.Info(string(resp.StatusCode))
	body, _ := io.ReadAll(resp.Body)
	s.Log.Info(string(body))
	return resp
}

// GetSubscriptions Retrieves all subscriptions for the application
func (s *Subscription) GetSubscriptions() ValidateSubscription {
	req, _ := http.NewRequest("GET", url, nil)
	headers := s.Secrets.BuildSecretHeaders()
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+headers.Token)
	req.Header.Set("Client-Id", headers.ClientID)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		s.Log.Error("Error sending request:", err)
	}
	body, _ := io.ReadAll(resp.Body)
	var subscriptionList ValidateSubscription
	json.Unmarshal(body, &subscriptionList)
	return subscriptionList
}

// CleanSubscriptions Removes all existing subscriptions
func (s *Subscription) CleanSubscriptions(subs ValidateSubscription) {
	if subs.Total > 0 {
		for _, sub := range subs.Data {
			deleteURL := fmt.Sprintf("%v?id=%v", url, sub.ID)
			req, err := http.NewRequest("DELETE", deleteURL, nil)
			if err != nil {
				return
			}
			headers := s.Secrets.BuildSecretHeaders()
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+headers.Token)
			req.Header.Set("Client-Id", headers.ClientID)
			client := &http.Client{}
			resp, _ := client.Do(req)
			if resp.StatusCode == http.StatusNoContent {
				s.Log.Info("Subscription deleted:" + sub.ID)
			}
		}
	} else {
		s.Log.Info("No subscriptions to delete")
	}
}

// func DeleteSubscription deletes a subscription with an ID
func (s *Subscription) DeleteSubscription(id int) {
	deleteURL := fmt.Sprintf("%v?id=%v", url, id)
	req, err := http.NewRequest("DELETE", deleteURL, nil)
	if err != nil {
		return
	}
	headers := s.Secrets.BuildSecretHeaders()
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+headers.Token)
	req.Header.Set("Client-Id", headers.ClientID)
	client := &http.Client{}
	resp, _ := client.Do(req)
	if resp.StatusCode == http.StatusNoContent {
		s.Log.Info(string(id))
	}
}
