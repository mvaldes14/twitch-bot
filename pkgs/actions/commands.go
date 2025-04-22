// Package actions handles Twitch chat commands and actions
package actions

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/mvaldes14/twitch-bot/pkgs/secrets"
	"github.com/mvaldes14/twitch-bot/pkgs/spotify"
	"github.com/mvaldes14/twitch-bot/pkgs/subscriptions"
	"github.com/mvaldes14/twitch-bot/pkgs/telemetry"
)

const (
	messageEndpoint  = "https://api.twitch.tv/helix/chat/messages"
	channelsEndpoint = "https://api.twitch.tv/helix/channels"
	userID           = "1792311"
	softwareID       = 1469308723
)

var (
	errUpdateChannel = errors.New("updating channel info")
)

// Actions handles all Twitch chat actions and commands
type Actions struct {
	Log     *telemetry.CustomLogger
	Secrets *secrets.SecretService
	Spotify *spotify.Spotify
}

// NewActions creates a new Actions instance
func NewActions(secrets *secrets.SecretService) *Actions {
	logger := telemetry.NewLogger("actions")
	return &Actions{
		Log:     logger,
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
		song := a.Spotify.GetSong()
		msg := fmt.Sprintf("Now playing: %v - %v", song.Item.Artists[0].Name, song.Item.Name)
		a.Log.Info(msg)
		a.SendMessage(msg)
	}
	// Complex commands
	if strings.HasPrefix(msg.Event.Message.Text, "!today") {
		a.Log.Info("Today command running")
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
		a.Log.Error("Failed to marshal message:", err)
		return err
	}

	req, err := http.NewRequest("POST", messageEndpoint, bytes.NewBuffer(payload))
	if err != nil {
		a.Log.Error("Failed to create request", err)
		return err
	}

	headers, err := a.Secrets.BuildSecretHeaders()
	if err != nil {
		a.Log.Error("Failed to build headers to send message", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+headers.Token)
	req.Header.Set("Client-Id", headers.ClientID)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		a.Log.Error("failed to send message: %w", err)
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		a.Log.Info("Unexpected status code while sending message, response: " + strconv.Itoa(res.StatusCode))
		return err
	}

	return nil
}

func (a *Actions) updateChannel(action subscriptions.ChatMessageEvent) {
	a.Log.Info("Changing the channel information")
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
		a.Log.Info("Today Command Ran")

		// Send request to update channel information
		req, err := http.NewRequest("PATCH", "https://api.twitch.tv/helix/channels?broadcaster_id="+userID, bytes.NewBuffer([]byte(payload)))
		if err != nil {
			a.Log.Error("Could not form request to update channel info", err)
			return
		}

		headers, err := a.Secrets.BuildSecretHeaders()
		if err != nil {
			a.Log.Error("Failed to build headers to update channel", err)
		}
		userToken := a.Secrets.GetUserToken()
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+userToken)
		req.Header.Set("Client-Id", headers.ClientID)

		for {
			client := &http.Client{}
			res, err := client.Do(req)
			if err != nil {
				a.Log.Error("Request could not be sent to update channel", err)
				return
			}
			if res.StatusCode != http.StatusBadRequest {
				a.Log.Error("Received a bad messag ", errUpdateChannel)
			}
			if res.StatusCode == http.StatusOK {
				break
			}
		}
	}
}
