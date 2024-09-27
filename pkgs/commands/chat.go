// Package commands responds to chat events
package commands

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"

	"github.com/mvaldes14/twitch-bot/pkgs/spotify"
	"github.com/mvaldes14/twitch-bot/pkgs/types"
	"github.com/mvaldes14/twitch-bot/pkgs/utils"
)

const (
	messageEndpoint  = "https://api.twitch.tv/helix/chat/messages"
	channelsEndpoint = "https://api.twitch.tv/helix/channels"
	userID           = "1792311"
	admin            = "mr_mvaldes"
	softwareID       = 1469308723
)

var logger = utils.Logger()

// ParseMessage Parses the incoming messages from stream
func ParseMessage(msg types.ChatMessageEvent) {
	// Simple commands
	switch msg.Event.Message.Text {
	case "!commands":
		SendMessage("!github, !dotfiles, !song, !social, !blog ")
	case "!github":
		SendMessage("https://links.mvaldes.dev/gh")
	case "!dotfiles":
		SendMessage("https://links.mvaldes.dev/dotfiles")
	case "!test":
		SendMessage("Test Me")
	case "!social":
		SendMessage("https://links.mvaldes.dev/twitter")
	case "!blog":
		SendMessage("https://mvaldes.dev")
	case "!song":
		token := spotify.RefreshToken()
		song := spotify.GetSong(token)
		msg := fmt.Sprintf("Now playing: %v - %v", song.Item.Artists[0].Name, song.Item.Name)
		SendMessage(msg)
	}
	// Complex commands
	if strings.HasPrefix(msg.Event.Message.Text, "!today") {
		updateChannel(msg)
	}
}

func updateChannel(action types.ChatMessageEvent) {
	// Check if user is me so I can update the channel
	if action.Event.BroadcasterUserName == admin {
		// Build the new payload,
		splitMsg := strings.Split(action.Event.Message.Text, " ")
		msg := strings.Join(splitMsg[1:], " ")
		payload := fmt.Sprintf(`{
      "game_id":"%v",
      "title":"🚨[Devops]🚨- %v",
      "tags":["devops","Español","SpanishAndEnglish","coding","neovim","k8s","terraform","go","homelab", "nix"],
      "broadcaster_language":"en"}`,
			softwareID, msg)
		// Send request to update channel information
		req, err := http.NewRequest("PATCH", "https://api.twitch.tv/helix/channels?broadcaster_id=1792311", bytes.NewBuffer([]byte(payload)))
		if err != nil {
			logger.Error("Could not form request to update channel info")
		}
		headers := utils.BuildHeaders()
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+headers.Token)
		req.Header.Set("Client-Id", headers.ClientID)

		client := &http.Client{}
		res, err := client.Do(req)
		if err != nil {
			logger.Error("Request could not be sent to update channel")
		}
		if res.StatusCode != http.StatusNoContent {
			logger.Error("Could not update channel", res)
		}
	}
}
