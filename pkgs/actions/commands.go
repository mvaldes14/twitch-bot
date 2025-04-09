// Package actions handles Twitch chat commands and actions
package actions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/mvaldes14/twitch-bot/pkgs/secrets"
	"github.com/mvaldes14/twitch-bot/pkgs/spotify"
	"github.com/mvaldes14/twitch-bot/pkgs/subscriptions"
)

const (
	messageEndpoint  = "https://api.twitch.tv/helix/chat/messages"
	channelsEndpoint = "https://api.twitch.tv/helix/channels"
	userID           = "1792311"
	softwareID       = 1469308723
)

// Actions handles all Twitch chat actions and commands
type Actions struct {
	Logger  *slog.Logger
	Secrets secrets.SecretManager
	Spotify *spotify.Spotify
}

// NewActions creates a new Actions instance
func NewActions(logger *slog.Logger, secrets secrets.SecretManager) *Actions {
	return &Actions{
		Logger:  logger,
		Secrets: secrets,
	}
}

// ParseMessage Parses the incoming messages from stream
func (a *Actions) ParseMessage(msg subscriptions.ChatMessageEvent) {
	// Simple commands
	switch msg.Event.Message.Text {
	case "!commands":
		a.SendMessage("!github, !dotfiles, !song, !social, !blog, !youtube ")
	case "!github":
		a.SendMessage("https://links.mvaldes.dev/gh")
	case "!dotfiles":
		a.SendMessage("https://links.mvaldes.dev/dotfiles")
	case "!test":
		a.SendMessage("Test Me")
	case "!social":
		a.SendMessage("https://links.mvaldes.dev/twitter")
	case "!blog":
		a.SendMessage("https://mvaldes.dev")
	case "!discord":
		a.SendMessage("https://links.mvaldes.dev/discord")
	case "!youtube":
		a.SendMessage("https://links.mvaldes.dev/youtube")
	case "!song":
		song := a.Spotify.GetSong(a.Spotify.RefreshToken())
		msg := fmt.Sprintf("Now playing: %v - %v", song.Item.Artists[0].Name, song.Item.Name)
		a.SendMessage(msg)
	}
	// Complex commands
	if strings.HasPrefix(msg.Event.Message.Text, "!today") {
		a.Logger.Info("Today command")
		a.updateChannel(msg)
	}
}

// SendMessage sends a message to the Twitch chat room
func (a *Actions) SendMessage(text string) error {
	message := subscriptions.ChatMessage{
		BroadcasterID: userID,
		SenderID:      userID,
		Message:       text,
	}

	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequest("POST", messageEndpoint, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	headers := a.Secrets.BuildSecretHeaders()
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+headers.Token)
	req.Header.Set("Client-Id", headers.ClientID)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	return nil
}

func (a *Actions) updateChannel(action subscriptions.ChatMessageEvent) {
	a.Logger.Info("Changing the channel information")
	// Check if user is me so I can update the channel
	if action.Event.BroadcasterUserID == userID {
		// Build the new payload
		splitMsg := strings.Split(action.Event.Message.Text, " ")
		msg := strings.Join(splitMsg[1:], " ")
		payload := fmt.Sprintf(`{
      "game_id":"%v",
      "title":"🚨[Devops]🚨- %v",
      "tags":["devops","Español","SpanishAndEnglish","coding","neovim","k8s","terraform","go","homelab", "nix", "gaming"],
      "broadcaster_language":"es"}`,
			softwareID, msg)
		a.Logger.Info("Today Command Payload", slog.String("Payload", payload))

		// Send request to update channel information
		req, err := http.NewRequest("PATCH", "https://api.twitch.tv/helix/channels?broadcaster_id="+userID, bytes.NewBuffer([]byte(payload)))
		if err != nil {
			a.Logger.Error("Could not form request to update channel info")
			return
		}

		headers := a.Secrets.BuildSecretHeaders()
		userToken := a.Secrets.GetUserToken()
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+userToken)
		req.Header.Set("Client-Id", headers.ClientID)

		for {
			client := &http.Client{}
			res, err := client.Do(req)
			if err != nil {
				a.Logger.Error("Request could not be sent to update channel")
				return
			}
			if res.StatusCode != http.StatusBadRequest {
				a.Logger.Error("Could not update channel", slog.Int("error", res.StatusCode))
				// Attempt to refresh the token
				token := a.Secrets.GenerateNewToken()
				a.Secrets.StoreNewTokens(token)
			}
			if res.StatusCode == http.StatusOK {
				break
			}
		}
	}
}
