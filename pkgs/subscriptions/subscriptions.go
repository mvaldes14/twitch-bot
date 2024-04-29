// Package subscriptions Handles all subscriptions actions in Twitch
package subscriptions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/mvaldes14/twitch-bot/pkgs/types"
	"github.com/mvaldes14/twitch-bot/pkgs/utils"
)

const url = "https://api.twitch.tv/helix/eventsub/subscriptions"

// CreateSubscription Generates  a new subscription on an event type
func CreateSubscription(payload string) *http.Response {
	// subscribe to eventsub
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(payload)))
	if err != nil {
		return nil
	}
	// Add key headers to request
	headers := utils.BuildHeaders()
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+headers.Token)
	req.Header.Set("Client-Id", headers.ClientID)

	// Create an HTTP client
	client := &http.Client{}

	// Send the request and get the response
	log.Println("Sending request")
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error sending request:", err)
		return nil
	}
	log.Println("Subscription response:", resp.StatusCode)
	return resp
}

// GetSubscriptions Retrieves all subscriptions for the application
func GetSubscriptions() types.ValidateSubscription {
	req, err := http.NewRequest("GET", url, nil)
	headers := utils.BuildHeaders()
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+headers.Token)
	req.Header.Set("Client-Id", headers.ClientID)
	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error sending request:", err)
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
			headers := utils.BuildHeaders()
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+headers.Token)
			req.Header.Set("Client-Id", headers.ClientID)
			client := &http.Client{}

			resp, err := client.Do(req)
			if resp.StatusCode == http.StatusNoContent {
				log.Println("Subscription deleted:", sub.ID)
			}
		}
	} else {
		log.Println("No subscriptions to delete")
	}
}
