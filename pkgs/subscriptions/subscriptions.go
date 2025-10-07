// Package subscriptions Handles all subscriptions actions in Twitch
package subscriptions

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/mvaldes14/twitch-bot/pkgs/cache"
	"github.com/mvaldes14/twitch-bot/pkgs/secrets"
	"github.com/mvaldes14/twitch-bot/pkgs/telemetry"
)

const (
	// URL endpoint for all twitch subscriptions
	URL = "https://api.twitch.tv/helix/eventsub/subscriptions"
)

var (
	errFailedSubscriptionCreation = errors.New("Failed to create new subscription")
	errFailedSubscriptionDeletion = errors.New("Failed to delete subscription")
	errFailedToFormRequest        = errors.New("Failed to form request")
)

// Subscription is the struct that handles all subscriptions
type Subscription struct {
	Secrets *secrets.SecretService
	Logger  *telemetry.BotLogger
	Cache   *cache.CacheService
}

// NewSubscription creates a new subscription
func NewSubscription(secretService *secrets.SecretService) *Subscription {
	log := telemetry.NewLogger("subscriptions")
	cache := cache.NewCacheService()
	return &Subscription{
		Secrets: secretService,
		Logger:  log,
		Cache:   cache,
	}
}

// CreateSubscription Generates  a new subscription on an event type
func (s *Subscription) CreateSubscription(payload string) error {
	// subscribe to eventsub
	req, err := http.NewRequest("POST", URL, bytes.NewBuffer([]byte(payload)))
	if err != nil {
		return nil
	}
	// Add key headers to request
	headers, err := s.Secrets.BuildSecretHeaders()
	if err != nil {
		s.Logger.Error(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+headers.Token)
	req.Header.Set("Client-Id", headers.ClientID)
	// Create an HTTP client
	client := &http.Client{}
	// Send the request and get the response
	s.Logger.Info("Sending request for subscription for:" + payload)
	resp, err := client.Do(req)
	if err != nil {
		s.Logger.Error(err)
		return nil
	}
	defer resp.Body.Close()
	s.Logger.Info("Subscription created for: " + payload)
	return errFailedSubscriptionCreation
}

// GetSubscriptions Retrieves all subscriptions for the application
func (s *Subscription) GetSubscriptions() (ValidateSubscription, error) {
	req, _ := http.NewRequest("GET", URL, nil)
	token := os.Getenv("TWITCH_USER_TOKEN")
	clientID := os.Getenv("TWITCH_CLIENT_ID")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Client-Id", clientID)
	client := &http.Client{}
	resp, err := client.Do(req)
	// if err != nil {
	// 	s.Log.Error("Error sending request:", err)
	// }
	// if resp.StatusCode != http.StatusOK {
	// 	s.Log.Error("Error received from Twitch API:", errors.New(resp.Status))
	// 	newToken, err := s.Secrets.GenerateUserToken()
	// 	if newToken.AccessToken == "" || err != nil {
	// 		return ValidateSubscription{}, errors.New("failed to generate new user token")
	// 	}
	// 	err = s.Secrets.StoreNewTokens("TWITCH_USER_TOKEN", newToken.AccessToken)
	// 	if err != nil {
	// 		return ValidateSubscription{}, fmt.Errorf("error received from Twitch API: %s", resp.Status)
	// 	}
	// }
	body, _ := io.ReadAll(resp.Body)
	var subscriptionList ValidateSubscription
	err = json.Unmarshal(body, &subscriptionList)
	if err != nil {
		s.Logger.Error(err)
	}
	return subscriptionList, nil
}

// DeleteSubscriptions Removes all existing subscriptions
func (s *Subscription) DeleteSubscriptions(subs ValidateSubscription) error {
	if subs.Total > 0 {
		for _, sub := range subs.Data {
			deleteURL := fmt.Sprintf("%v?id=%v", URL, sub.ID)
			req, err := http.NewRequest("DELETE", deleteURL, nil)
			if err != nil {
				return errFailedToFormRequest
			}
			headers, err := s.Secrets.BuildSecretHeaders()
			if err != nil {
				s.Logger.Error(err)
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+headers.Token)
			req.Header.Set("Client-Id", headers.ClientID)
			s.Logger.Info("Deleting subscription:" + sub.ID)
			client := &http.Client{}
			resp, _ := client.Do(req)
			if resp.StatusCode == http.StatusNoContent {
				s.Logger.Info("Subscription deleted:" + sub.ID)
			}
			return errFailedSubscriptionDeletion
		}
	} else {
		s.Logger.Info("No subscriptions to delete")
	}
	return nil
}
