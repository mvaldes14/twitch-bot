// Package commands contains the utilities for the bot
package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mvaldes14/twitch-bot/pkgs/types"
	"github.com/mvaldes14/twitch-bot/pkgs/utils"
)

// SendMessage Allows you to send a message to the chat room
func SendMessage(text string) {
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
