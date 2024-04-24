package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/mvaldes14/twitch-bot/pkgs/types"
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
	}
	if eventHeaderType == "notification" {
		var chatEvent types.ChatMessageEvent
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return
		}
		r.Body.Close()
		json.Unmarshal(body, &chatEvent)
		fmt.Printf("User: %v msg: %v", chatEvent.Event.BroadcasterUserLogin, chatEvent.Event.Message.Text)
	}
}

func NewServer() {
	http.HandleFunc("/chat", chatHandler)
	// http.HandleFunc("/follow", chatHandler)
	// http.HandleFunc("/subs", chatHandler)
	fmt.Println("Running and listening")
	http.ListenAndServe(":3000", nil)
}
