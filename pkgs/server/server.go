// Package server Generates the server and handlers to respond to requests
package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/mvaldes14/twitch-bot/pkgs/commands"
	"github.com/mvaldes14/twitch-bot/pkgs/subscriptions"
	"github.com/mvaldes14/twitch-bot/pkgs/types"
	"github.com/mvaldes14/twitch-bot/pkgs/utils"
)

func chatHandler(w http.ResponseWriter, r *http.Request) {
	eventHeaderType := r.Header.Get("Twitch-Eventsub-Message-Type")
	if eventHeaderType == "webhook_callback_verification" {
		fmt.Println("Responding to challenge")
		var challengeResponse types.SubscribeEvent
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return
		}
		r.Body.Close()
		json.Unmarshal(body, &challengeResponse)
		w.Header().Add("Content-Type", "plain/text")
		w.Write([]byte(challengeResponse.Challenge))
	} else if eventHeaderType == "notification" {
		var chatEvent types.ChatMessageEvent
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return
		}
		r.Body.Close()
		json.Unmarshal(body, &chatEvent)
		fmt.Printf("User: %v msg: %v", chatEvent.Event.BroadcasterUserLogin, chatEvent.Event.Message.Text)
		commands.ParseMessage(chatEvent)
	} else {
		fmt.Print("Unsupported request type")
	}
}

func deleteHandler(_ http.ResponseWriter, _ *http.Request) {
	subsList := subscriptions.GetSubscriptions()
	subscriptions.CleanSubscriptions(subsList)
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Write([]byte("msg: OK\n"))
}

func createHandler(_ http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	subType := query.Get("type")
	switch subType {
	case "chat":
		// Generate subscription type for chats
		chatSubType := types.SubscriptionType{
			Name:    "chat",
			Version: "1",
			Type:    "channel.chat.message",
		}
		payload := utils.GeneratePayload(chatSubType)
		subscriptions.CreateSubscription(payload)
	case "follow":
		// Generate subscription type for follow
		chatSubType := types.SubscriptionType{
			Name:    "follow",
			Version: "2",
			Type:    "channel.follow",
		}
		payload := utils.GeneratePayload(chatSubType)
		subscriptions.CreateSubscription(payload)
	case "subscription":
		// Generate subscription type for subscriptions
		chatSubType := types.SubscriptionType{
			Name:    "subscribe",
			Version: "2",
			Type:    "channel.subscribe",
		}
		payload := utils.GeneratePayload(chatSubType)
		subscriptions.CreateSubscription(payload)
	}
}

// NewServer creates the http server
func NewServer() {
	http.HandleFunc("/create", createHandler)
	http.HandleFunc("/delete", deleteHandler)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/chat", chatHandler)
	// http.HandleFunc("/follow", chatHandler)
	// http.HandleFunc("/subs", chatHandler)
	fmt.Println("Running and listening")
	http.ListenAndServe(":3000", nil)
}
