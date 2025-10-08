// Package subscriptions Handles all subscriptions actions in Twitch
package subscriptions

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/mvaldes14/twitch-bot/pkgs/cache"
	"github.com/mvaldes14/twitch-bot/pkgs/secrets"
	"github.com/mvaldes14/twitch-bot/pkgs/service"
)

const (
	// URL endpoint for all twitch subscriptions
	URL = "https://api.twitch.tv/helix/eventsub/subscriptions"
)

// TODO: Think of all the possible errors we can throw based on the service
var (
	errFailedSubscriptionCreation = errors.New("Failed to create new subscription")
	errFailedSubscriptionDeletion = errors.New("Failed to delete subscription")
	errFailedToFormRequest        = errors.New("Failed to form request")
)

// Subscription is the struct that handles all subscriptions
type Subscription struct {
	Secrets *secrets.SecretService
	Service *service.Service
	Cache   *cache.CacheService
}

// NewSubscription creates a new subscription
func NewSubscription(secretService *secrets.SecretService) *Subscription {
	service := service.NewService("subscriptions")
	cache := cache.NewCacheService()
	return &Subscription{
		Secrets: secretService,
		Cache:   cache,
		Service: service,
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
		s.Service.Logger.Error(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+headers.Token)
	req.Header.Set("Client-Id", headers.ClientID)
	// Create an HTTP client
	// Send the request and get the response
	s.Service.Logger.Info("Sending request for subscription for:" + payload)
	resp, err := s.Service.Client.Do(req)
	if err != nil {
		s.Service.Logger.Error(err)
		return nil
	}
	defer resp.Body.Close()
	s.Service.Logger.Info("Subscription created for: " + payload)
	return errFailedSubscriptionCreation
}

// GetSubscriptions Retrieves all subscriptions for the application
func (s *Subscription) GetSubscriptions() (ValidateSubscription, error) {
	req, _ := http.NewRequest("GET", URL, nil)
	token, err := s.Cache.GetToken("TWITCH_USER_TOKEN")
	if err != nil {
		s.Service.Logger.Error(err)
	}
	clientID, err := s.Cache.GetToken("TWITCH_CLIENT_ID")
	if err != nil {
		s.Service.Logger.Error(err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token.Value)
	req.Header.Set("Client-Id", clientID.Value)
	resp, err := s.Service.Client.Do(req)
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
	s.Service.Logger.Info("Response from Twitch: " + string(body))
	var subscriptionList ValidateSubscription
	err = json.Unmarshal(body, &subscriptionList)
	if err != nil {
		s.Service.Logger.Error(err)
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
				s.Service.Logger.Error(err)
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+headers.Token)
			req.Header.Set("Client-Id", headers.ClientID)
			s.Service.Logger.Info("Deleting subscription:" + sub.ID)
			resp, _ := s.Service.Client.Do(req)
			if resp.StatusCode == http.StatusNoContent {
				s.Service.Logger.Info("Subscription deleted:" + sub.ID)
			} else {
				return errFailedSubscriptionDeletion
			}
		}
	}
	s.Service.Logger.Info("No subscriptions to delete")
	return nil
}
