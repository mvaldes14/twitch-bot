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
		defer r.Body.Close()
		json.Unmarshal(body, &challengeResponse)
		w.Header().Add("Content-Type", "plain/text")
		w.Write([]byte(challengeResponse.Challenge))
	} else if eventHeaderType == "notification" {
		var chatEvent types.ChatMessageEvent
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return
		}
		defer r.Body.Close()
		json.Unmarshal(body, &chatEvent)
		fmt.Printf("User: %v msg: %v\n", chatEvent.Event.ChatterUserName, chatEvent.Event.Message.Text)
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
			Version: "1",
			Type:    "channel.subscribe",
		}
		payload := utils.GeneratePayload(chatSubType)
		subscriptions.CreateSubscription(payload)
	}
}

func followHandler(w http.ResponseWriter, r *http.Request) {
	headerType := r.Header.Get("Twitch-Eventsub-Message-Type")
	if headerType == "webhook_callback_verification" {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return
		}
		defer r.Body.Close()
		var subscriptionResponse types.SubscribeEvent
		json.Unmarshal(body, &subscriptionResponse)
		fmt.Println("Responding to challenge")
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(subscriptionResponse.Challenge))
	} else if headerType == "notification" {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return
		}
		defer r.Body.Close()
		var followEventResponse types.FollowEvent
		json.Unmarshal(body, &followEventResponse)
		commands.SendMessage(fmt.Sprintf("Gracias por el follow: %v", followEventResponse.Event.UserName))
	}
}

func subHandler(w http.ResponseWriter, r *http.Request) {
	headerType := r.Header.Get("Twitch-Eventsub-Message-Type")
	if headerType == "webhook_callback_verification" {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return
		}
		defer r.Body.Close()
		var subscriptionResponse types.SubscribeEvent
		json.Unmarshal(body, &subscriptionResponse)
		fmt.Println("Responding to challenge")
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(subscriptionResponse.Challenge))
	} else if headerType == "notification" {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return
		}
		defer r.Body.Close()
		var followEventResponse types.SubscriptionEvent
		json.Unmarshal(body, &followEventResponse)
		commands.SendMessage(fmt.Sprintf("Gracias por el sub: %v", followEventResponse.Event.UserName))
	}
}

// NewServer creates the http server
func NewServer() {
	http.HandleFunc("/create", createHandler)
	http.HandleFunc("/delete", deleteHandler)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/chat", chatHandler)
	http.HandleFunc("/follow", followHandler)
	http.HandleFunc("/sub", subHandler)
	fmt.Println("Running and listening")
	http.ListenAndServe(":3000", nil)
}
