// Package subscriptions Handles all subscriptions actions in Twitch
package subscriptions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/mvaldes14/twitch-bot/pkgs/types"
	"github.com/mvaldes14/twitch-bot/pkgs/utils"
)

const url = "https://api.twitch.tv/helix/eventsub/subscriptions"

var logger = utils.Logger()

// CreateSubscription Generates  a new subscription on an event type
func CreateSubscription(payload string) *http.Response {
	// subscribe to eventsub
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(payload)))
	if err != nil {
		return nil
	}
	// Add key headers to request
	headers := utils.BuildSecretHeaders()
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+headers.Token)
	req.Header.Set("Client-Id", headers.ClientID)
	// Create an HTTP client
	client := &http.Client{}
	// Send the request and get the response
	logger.Info("Sending request")
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("Error sending request:", "caused by", err)
		return nil
	}
	defer resp.Body.Close()
	logger.Info("Subscription response:", "info", resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	logger.Info("Subscription", "message", string(body))
	return resp
}

// GetSubscriptions Retrieves all subscriptions for the application
func GetSubscriptions() types.ValidateSubscription {
	req, err := http.NewRequest("GET", url, nil)
	headers := utils.BuildSecretHeaders()
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+headers.Token)
	req.Header.Set("Client-Id", headers.ClientID)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("Error sending request:", "caused by", err)
	}
	body, err := io.ReadAll(resp.Body)
	var subscriptionList types.ValidateSubscription
	json.Unmarshal(body, &subscriptionList)
	return subscriptionList
}

// CleanSubscriptions Removes all existing subscriptions
func CleanSubscriptions(subs types.ValidateSubscription) {
	if subs.Total > 0 {
		for _, sub := range subs.Data {
			deleteURL := fmt.Sprintf("%v?id=%v", url, sub.ID)
			req, err := http.NewRequest("DELETE", deleteURL, nil)
			if err != nil {
				return
			}
			headers := utils.BuildSecretHeaders()
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+headers.Token)
			req.Header.Set("Client-Id", headers.ClientID)
			client := &http.Client{}
			resp, err := client.Do(req)
			if resp.StatusCode == http.StatusNoContent {
				logger.Info("Subscription deleted:" + sub.ID)
			}
		}
	} else {
		logger.Info("No subscriptions to delete")
	}
}
