package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/mvaldes14/twitch-bot/pkgs/subscriptions"
	"github.com/mvaldes14/twitch-bot/pkgs/types"
)

const url = "https://api.twitch.tv/helix/eventsub/subscriptions"

func buildHeaders() types.RequestHeader {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	}
	token := os.Getenv("TWITCH_TOKEN")
	clientID := os.Getenv("TWITCH_CLIENT_ID")
	return types.RequestHeader{
		Token:    token,
		ClientID: clientID,
	}
}

func validateSubscription(subType string) bool {
	req, err := http.NewRequest("GET", url, nil)
	var validateResponse types.ValidateSubscription
	defer req.Body.Close()
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return false
	}
	json.Unmarshal(body, &validateResponse)

	fmt.Println(subType)
	for k, v := range validateResponse.Data {
		fmt.Println(k, v)
	}

	return false

}

func chatHandler(w http.ResponseWriter, r *http.Request) {
	// Generate subscription type for chats
	chatSubType := types.SubscriptionType{
		Name:    "chat",
		Version: "1",
		Type:    "channel.chat.message",
	}
	payload := subscriptions.GeneratePayload(chatSubType)

	// subscribe to eventsub
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(payload)))

	if err != nil {
		return
	}

	// Add key headers to request
	headers := buildHeaders()
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+headers.Token)
	req.Header.Set("Client-Id", headers.ClientID)
	// Create an HTTP client
	client := &http.Client{}

	// Send the request and get the response
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return
	}
	defer resp.Body.Close()

	// Print the response status code
	fmt.Println("Response Header:", r.Header)
	// Response Header: map[Twitch-Eventsub-Message-Type:[webhook_callback_verification]  Twitch-Eventsub-Subscription-Type:[channel.chat.message] ]
	eventHeaderType := r.Header.Get("Twitch-Eventsub-Subscription-Type")
	if eventHeaderType == "webbook_call_verification" {
		var challengeResponse types.SubscribeEvent
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return
		}
		json.Unmarshal(body, &challengeResponse)
		w.Header().Add("Content-Type", "plain/text")
		w.Write([]byte(challengeResponse.Challenge))
		// Validate sub is enabled
		validateSubscription(chatSubType.Type)

	}

	var chatEvent types.ChatMessageEvent

	// Grab the response and look for field
	fmt.Println("Chat: Request received")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return
	}
	r.Body.Close()
	json.Unmarshal(body, &chatEvent)
	var response types.ChatMessageEvent
	fmt.Println(response.Event.ChatterUserLogin, response.Event.Message)
}

func NewServer() {
	http.HandleFunc("/chat", chatHandler)
	// http.HandleFunc("/follow", chatHandler)
	// http.HandleFunc("/subs", chatHandler)
	fmt.Println("Running and listening")
	http.ListenAndServe(":3000", nil)
}
