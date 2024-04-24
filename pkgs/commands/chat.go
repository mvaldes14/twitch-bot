// Package commands responds to chat events
package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mvaldes14/twitch-bot/pkgs/types"
	"github.com/mvaldes14/twitch-bot/pkgs/utils"
)

const (
	url    = "https://api.twitch.tv/helix/chat/messages"
	userID = "1792311"
)

// ParseMessage Parses the incoming messages from stream
func ParseMessage(msg types.ChatMessageEvent) {
	switch msg.Event.Message.Text {
	case "!commands":
		sendMessage("!github, !dotfiles")
	case "!github":
		sendMessage("https://links.mvaldes.dev/gh")
	case "!dotfiles":
		sendMessage("https://links.mvaldes.dev/dotfiles")
	case "!test":
		sendMessage("Test Me")
	}
}

func sendMessage(text string) {
	message := types.ChatMessage{
		BroadcasterID: userID,
		SenderID:      userID,
		Message:       text,
	}
	payload, err := json.Marshal(message)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	headers := utils.BuildHeaders()
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+headers.Token)
	req.Header.Set("Client-Id", headers.ClientID)
	if err != nil {
		return
	}
	client := &http.Client{}

	res, err := client.Do(req)
	if err != nil {
		return
	}
	fmt.Println(res.StatusCode)
}
