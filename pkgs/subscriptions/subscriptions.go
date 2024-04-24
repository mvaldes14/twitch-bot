package subscriptions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/mvaldes14/twitch-bot/pkgs/server"
	"github.com/mvaldes14/twitch-bot/pkgs/types"
)

const (
	userID      = "1792311"
	callbackURL = "https://every-hoops-act.loca.lt/"
	secret      = "superSecret123"
	url         = "https://api.twitch.tv/helix/eventsub/subscriptions"
)

// Create a subscription
// TODO: Respond to challenge
func generatePayload(subType types.SubscriptionType) string {
	var payload string

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
        "callback":"%v","secret":"%v"
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
        "callback": "%v","secret": "%v"
      }
    }`, subType.Type, subType.Version, userID, userID, callbackURL, secret)

	case "sub":
		payload = fmt.Sprintf(`{
      "type": "%v",
      "version": "%v",
      "condition": {
          "broadcaster_user_id": "%v"
      },
      "transport": {
          "method": "webhook",
          "callback": "%v",
          "secret": "%v"
      }
    }`, subType.Type, subType.Version, userID, callbackURL, secret)

	}
	return payload
}

func createChatSubscription() {
	// Generate subscription type for chats
	chatSubType := types.SubscriptionType{
		Name:    "chat",
		Version: "1",
		Type:    "channel.chat.message",
	}
	payload := generatePayload(chatSubType)
	// subscribe to eventsub
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(payload)))
	if err != nil {
		return
	}
	// Add key headers to request
	headers := server.BuildHeaders()
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+headers.Token)
	req.Header.Set("Client-Id", headers.ClientID)

	// Create an HTTP client
	client := &http.Client{}

	// Send the request and get the response
	fmt.Println("Sending request")
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return
	}
	defer resp.Body.Close()
}

func validateSubscription(subType string) bool {
	req, err := http.NewRequest("GET", url, nil)
	headers := server.BuildHeaders()
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+headers.Token)
	req.Header.Set("Client-Id", headers.ClientID)
	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return false
	}
	body, err := io.ReadAll(resp.Body)
	var validateResponse types.ValidateSubscription
	json.Unmarshal(body, &validateResponse)
	for _, v := range validateResponse.Data {
		if v.Status == "webhook_callback_verification_failed" {
			cleanInvalidSubscriptions(v.ID)
		}
		if v.Status == "active" && v.Type == subType {
			fmt.Printf("Active sub found for: %v", subType)
			return true
		}
	}
	return false
}

func cleanInvalidSubscriptions(id string) {
	deleteURL := fmt.Sprintf("%v?=%v", url, id)
	req, err := http.NewRequest("DELETE", deleteURL, nil)
	if err != nil {
		return
	}
	headers := server.BuildHeaders()
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+headers.Token)
	req.Header.Set("Client-Id", headers.ClientID)
	client := &http.Client{}

	resp, err := client.Do(req)
	if resp.StatusCode == http.StatusNoContent {
		fmt.Println("subscription deleted:", id)
	}
}
