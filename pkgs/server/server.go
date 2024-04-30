// Package server Generates the server and handlers to respond to requests
package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/mvaldes14/twitch-bot/pkgs/commands"
	"github.com/mvaldes14/twitch-bot/pkgs/logs"
	"github.com/mvaldes14/twitch-bot/pkgs/subscriptions"
	"github.com/mvaldes14/twitch-bot/pkgs/types"
	"github.com/mvaldes14/twitch-bot/pkgs/utils"
)

var es elasticsearch.Client = *logs.NewClient()

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

func chatHandler(w http.ResponseWriter, r *http.Request) {
	eventHeaderType := r.Header.Get("Twitch-Eventsub-Message-Type")
	if eventHeaderType == "webhook_callback_verification" {
		log.Println("Responding to challenge")
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
		// Send to elastic
		logs.IndexEvent(es, chatEvent.Event.ChatterUserName, chatEvent.Event.Message.Text, "chat")
		// Send to parser to respond
		commands.ParseMessage(chatEvent)
	} else {
		fmt.Print("Unsupported request type")
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
		log.Println("Responding to challenge")
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
		// send to elastic
		msg := fmt.Sprintf("User: %v, followed on: %v", followEventResponse.Event.UserName, followEventResponse.Event.FollowedAt)
		logs.IndexEvent(es, followEventResponse.Event.UserName, msg, "follow")
		// Send to chat
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
		log.Println("Responding to challenge")
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(subscriptionResponse.Challenge))
	} else if headerType == "notification" {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return
		}
		defer r.Body.Close()
		var subEventResponse types.SubscriptionEvent
		json.Unmarshal(body, &subEventResponse)
		// send to elastic
		msg := fmt.Sprintf("User: %v, Tier: %v", subEventResponse.Event.UserName, subEventResponse.Event.Tier)
		logs.IndexEvent(es, subEventResponse.Event.UserName, msg, "sub")
		// send to chat
		commands.SendMessage(fmt.Sprintf("Gracias por el sub: %v", subEventResponse.Event.UserName))
	}
}

func cheerHandler(w http.ResponseWriter, r *http.Request) {
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
		var cheerEventResponse types.CheerEvent
		json.Unmarshal(body, &cheerEventResponse)
		// send to elastic
		msg := fmt.Sprintf("User: %v, Bits: %v", cheerEventResponse.Event.UserName, cheerEventResponse.Event.Bits)
		logs.IndexEvent(es, cheerEventResponse.Event.UserName, msg, "sub")
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
	http.HandleFunc("/cheer", cheerHandler)
	log.Println("Running and listening")
	http.ListenAndServe(":3000", nil)
}
