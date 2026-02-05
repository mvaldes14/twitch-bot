// Package subscriptions Handles all subscriptions actions in Twitch
package subscriptions

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/mvaldes14/twitch-bot/pkgs/cache"
	"github.com/mvaldes14/twitch-bot/pkgs/secrets"
	"github.com/mvaldes14/twitch-bot/pkgs/telemetry"
)

const (
	// URL endpoint for all twitch subscriptions
	URL = "https://api.twitch.tv/helix/eventsub/subscriptions"
)

var (
	errFailedSubscriptionDeletion = errors.New("failed to delete subscription")
	errFailedToFormRequest        = errors.New("failed to form request")
)

// Subscription is the struct that handles all subscriptions
type Subscription struct {
	Secrets *secrets.SecretService
	Log     *telemetry.CustomLogger
	Cache   *cache.Service
}

// NewSubscription creates a new subscription
func NewSubscription(secretService *secrets.SecretService) *Subscription {
	log := telemetry.NewLogger("subscriptions")
	cacheService := cache.NewCacheService()
	return &Subscription{
		Secrets: secretService,
		Log:     log,
		Cache:   cacheService,
	}
}

// CreateSubscription Generates  a new subscription on an event type
func (s *Subscription) CreateSubscription(payload string) error {
	ctx := context.Background()
	// subscribe to eventsub
	req, err := http.NewRequestWithContext(ctx, "POST", URL, bytes.NewBuffer([]byte(payload)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	// Add key headers to request
	headers, err := s.Secrets.BuildSecretHeaders()
	if err != nil {
		s.Log.Error("Error getting headers for CreateSubscription", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+headers.Token)
	req.Header.Set("Client-Id", headers.ClientID)
	// Create an HTTP client
	client := &http.Client{}
	// Send the request and get the response
	s.Log.Info("Sending request for subscription for:" + payload)
	resp, err := client.Do(req)
	if err != nil {
		s.Log.Error("Error sending request for new subscription", err)
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body for error details
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.Log.Error("Error reading response body for subscription creation", err)
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Check if the subscription was created successfully
	if resp.StatusCode != http.StatusCreated {
		s.Log.Error(fmt.Sprintf("Failed to create subscription - Status: %d, Response: %s", resp.StatusCode, string(body)), nil)
		return fmt.Errorf("failed to create subscription: status code %d", resp.StatusCode)
	}

	// Unmarshal response to get subscription details
	var subscriptionResponse ValidateSubscription
	if err := json.Unmarshal(body, &subscriptionResponse); err != nil {
		s.Log.Error("Error unmarshalling subscription response", err)
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Verify subscription was actually created
	if subscriptionResponse.Total > 0 && len(subscriptionResponse.Data) > 0 {
		createdSub := subscriptionResponse.Data[0]
		s.Log.Info(fmt.Sprintf("Subscription created successfully - ID: %s, Type: %s, Status: %s", createdSub.ID, createdSub.Type, createdSub.Status))
		return nil
	}

	s.Log.Error("Subscription response received but no subscription data found", errors.New("empty subscription data"))
	return errors.New("no subscription data in response")
}

// GetSubscriptions Retrieves all subscriptions for the application
func (s *Subscription) GetSubscriptions() (ValidateSubscription, error) {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, "GET", URL, nil)
	if err != nil {
		return ValidateSubscription{}, fmt.Errorf("failed to create request: %w", err)
	}
	headers, err := s.Secrets.BuildSecretHeaders()
	if err != nil {
		s.Log.Error("Error getting headers for GetSubscriptions", err)
		return ValidateSubscription{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+headers.Token)
	req.Header.Set("Client-Id", headers.ClientID)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		s.Log.Error("Error sending request:", err)
		return ValidateSubscription{}, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		s.Log.Error(fmt.Sprintf("Failed to get subscriptions - Status: %d, Response: %s", resp.StatusCode, string(body)), nil)
		return ValidateSubscription{}, fmt.Errorf("failed to get subscriptions: status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.Log.Error("Error reading response body:", err)
		return ValidateSubscription{}, fmt.Errorf("failed to read response: %w", err)
	}
	var subscriptionList ValidateSubscription
	err = json.Unmarshal(body, &subscriptionList)
	if err != nil {
		s.Log.Error("Error unmarshalling response:", err)
		return ValidateSubscription{}, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Log subscription details
	s.Log.Info(fmt.Sprintf("Retrieved %d subscriptions (Total Cost: %d/%d)", subscriptionList.Total, subscriptionList.TotalCost, subscriptionList.MaxTotalCost))
	for _, sub := range subscriptionList.Data {
		s.Log.Info(fmt.Sprintf("  - ID: %s, Type: %s, Status: %s, Version: %s, Cost: %d", sub.ID, sub.Type, sub.Status, sub.Version, sub.Cost))
	}

	return subscriptionList, nil
}

// DeleteSubscriptions Removes all existing subscriptions
func (s *Subscription) DeleteSubscriptions(subs ValidateSubscription) error {
	ctx := context.Background()
	if subs.Total > 0 {
		for _, sub := range subs.Data {
			deleteURL := fmt.Sprintf("%v?id=%v", URL, sub.ID)
			req, err := http.NewRequestWithContext(ctx, "DELETE", deleteURL, nil)
			if err != nil {
				return errFailedToFormRequest
			}
			headers, err := s.Secrets.BuildSecretHeaders()
			if err != nil {
				s.Log.Error("Error getting headers for DeleteSubscriptions", err)
				continue
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+headers.Token)
			req.Header.Set("Client-Id", headers.ClientID)
			s.Log.Info("Deleting subscription:" + sub.ID)
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				s.Log.Error("Error deleting subscription:", err)
				continue
			}
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusNoContent {
				s.Log.Info("Subscription deleted:" + sub.ID)
			} else {
				s.Log.Error("Failed to delete subscription", errFailedSubscriptionDeletion)
			}
		}
	} else {
		s.Log.Info("No subscriptions to delete")
	}
	return nil
}
