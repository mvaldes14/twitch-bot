package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const URL string = "https://api.twitch.tv/helix/eventsub/subscriptions"

type subscriptionResponse struct {
	Challenge    string `json:"challenge"`
	Subscription struct {
		ID        string `json:"id"`
		Status    string `json:"status"`
		Type      string `json:"type"`
		Version   string `json:"version"`
		Condition struct {
			BroadcasterUserID string `json:"broadcaster_user_id"`
			UserID            string `json:"user_id"`
		} `json:"condition"`
		Transport struct {
			Method    string `json:"method"`
			SessionID string `json:"session_id"`
		} `json:"transport"`
		CreatedAt time.Time `json:"created_at"`
		Cost      int       `json:"cost"`
	} `json:"subscription"`
	Event struct {
		BroadcasterUserID    string `json:"broadcaster_user_id"`
		BroadcasterUserLogin string `json:"broadcaster_user_login"`
		BroadcasterUserName  string `json:"broadcaster_user_name"`
		ChatterUserID        string `json:"chatter_user_id"`
		ChatterUserLogin     string `json:"chatter_user_login"`
		ChatterUserName      string `json:"chatter_user_name"`
		MessageID            string `json:"message_id"`
		Message              struct {
			Text      string `json:"text"`
			Fragments []struct {
				Type      string      `json:"type"`
				Text      string      `json:"text"`
				Cheermote interface{} `json:"cheermote"`
				Emote     interface{} `json:"emote"`
				Mention   interface{} `json:"mention"`
			} `json:"fragments"`
		} `json:"message"`
		Color  string `json:"color"`
		Badges []struct {
			SetID string `json:"set_id"`
			ID    string `json:"id"`
			Info  string `json:"info"`
		} `json:"badges"`
		MessageType                 string      `json:"message_type"`
		Cheer                       interface{} `json:"cheer"`
		Reply                       interface{} `json:"reply"`
		ChannelPointsCustomRewardID interface{} `json:"channel_points_custom_reward_id"`
	} `json:"event"`
}

func mainHandler(w http.ResponseWriter, r *http.Request) {
	// Grab the response and look for field
	fmt.Println("Request received")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return
	}
	r.Body.Close()
	var response subscriptionResponse
	error := json.Unmarshal(body, &response)
	if error != nil {
		return
	}
	if response.Challenge != "" {
		fmt.Println("Responding with challenge")
		w.Write([]byte(response.Challenge))
	}

	fmt.Println(response.Event.ChatterUserName, response.Event.Message.Text)
}

func main() {
	http.HandleFunc("/", mainHandler)
	fmt.Println("Running and listening")
	http.ListenAndServe(":3000", nil)
}
