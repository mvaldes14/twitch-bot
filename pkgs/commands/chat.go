// Package commands responds to chat events
package commands

import (
	"bytes"
	"fmt"
	"log/slog"
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
		logger.Info("Today command")
		updateChannel(msg)
	}
}

func updateChannel(action types.ChatMessageEvent) {
	logger.Info("Changing the channel information")
	// Check if user is me so I can update the channel
	if action.Event.BroadcasterUserID == userID {
		// Build the new payload ,
		splitMsg := strings.Split(action.Event.Message.Text, " ")
		msg := strings.Join(splitMsg[1:], " ")
		payload := fmt.Sprintf(`{
      "game_id":"%v",
      "title":"🚨[Devops]🚨- %v",
      "tags":["devops","Español","SpanishAndEnglish","coding","neovim","k8s","terraform","go","homelab", "nix", "gaming"],
      "broadcaster_language":"es"}`,
			softwareID, msg)
		logger.Info("Today Command Payload", slog.String("Payload", payload))
		// Send request to update channel information
		req, err := http.NewRequest("PATCH", "https://api.twitch.tv/helix/channels?broadcaster_id="+userID, bytes.NewBuffer([]byte(payload)))
		if err != nil {
			logger.Error("Could not form request to update channel info")
		}
		headers := utils.BuildSecretHeaders()
		userToken := utils.GetUserToken()
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+userToken)
		req.Header.Set("Client-Id", headers.ClientID)

		for {
			client := &http.Client{}
			res, err := client.Do(req)
			if err != nil {
				logger.Error("Request could not be sent to update channel")
			}
			if res.StatusCode != http.StatusNoContent {
				logger.Error("Could not update channel", slog.Int("error", res.StatusCode))
				// Attempt to refresh the token
				token := utils.GenerateNewToken()
				utils.StoreNewTokens(token)
			}
			if res.StatusCode == http.StatusOK {
				continue
			}
		}
	}
}
