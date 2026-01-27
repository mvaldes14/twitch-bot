// Package actions handles Twitch chat commands and actions
package actions

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
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
	spotifyClient := spotify.NewSpotify()
	return &Actions{
		Log:     logger,
		Secrets: secrets,
		Spotify: spotifyClient,
	}
}

// ParseMessage Parses the incoming messages from stream
func (a *Actions) ParseMessage(msg subscriptions.ChatMessageEvent) {
	payload := fmt.Sprintf("%s: %s", msg.Event.ChatterUserName, msg.Event.Message.Text)
	a.Log.Chat(payload)
	// Simple commands
	switch msg.Event.Message.Text {
	case "!commands":
		_ = a.SendMessage("!github, !dotfiles, !song, !social, !blog, !youtube ")
	case "!github":
		_ = a.SendMessage("https://links.mvaldes.dev/gh")
	case "!dotfiles":
		_ = a.SendMessage("https://links.mvaldes.dev/dotfiles")
	case "!test":
		_ = a.SendMessage("Test Me")
	case "!social":
		_ = a.SendMessage("https://links.mvaldes.dev/twitter")
	case "!blog":
		_ = a.SendMessage("https://mvaldes.dev")
	case "!discord":
		_ = a.SendMessage("https://links.mvaldes.dev/discord")
	case "!youtube":
		_ = a.SendMessage("https://links.mvaldes.dev/youtube")
	case "!song":
		song, err := a.Spotify.GetSong()
		if err != nil {
			a.Log.Error("Failed to get current song", err)
			_ = a.SendMessage("Sorry, couldn't get the current song")
			return
		}
		if song.Item.Name == "" || len(song.Item.Artists) == 0 {
			_ = a.SendMessage("No song currently playing")
			return
		}
		songMsg := fmt.Sprintf("Now playing: %v - %v", song.Item.Artists[0].Name, song.Item.Name)
		a.Log.Info(songMsg)
		_ = a.SendMessage(songMsg)
	}
	// Complex commands
	if strings.HasPrefix(msg.Event.Message.Text, "!today") {
		a.Log.Info("Today command running")
		a.updateChannel(msg)
	}
}

// SendMessage sends a message to the Twitch chat room
func (a *Actions) SendMessage(text string) error {
	ctx := context.Background()
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

	req, err := http.NewRequestWithContext(ctx, "POST", messageEndpoint, bytes.NewBuffer(payload))
	if err != nil {
		a.Log.Error("Failed to create request", err)
		return err
	}

	headers, err := a.Secrets.BuildSecretHeaders()
	if err != nil {
		a.Log.Error("Failed to build headers to send message", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+headers.Token)
	req.Header.Set("Client-Id", headers.ClientID)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		a.Log.Error("failed to send message", err)
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		a.Log.Info("Unexpected status code while sending message, response: " + strconv.Itoa(res.StatusCode))
		return fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	return nil
}

func (a *Actions) updateChannel(action subscriptions.ChatMessageEvent) {
	ctx := context.Background()
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
		req, err := http.NewRequestWithContext(ctx, "PATCH", "https://api.twitch.tv/helix/channels?broadcaster_id="+userID, bytes.NewBuffer([]byte(payload)))
		if err != nil {
			a.Log.Error("Could not form request to update channel info", err)
			return
		}

		headers, err := a.Secrets.BuildSecretHeaders()
		if err != nil {
			a.Log.Error("Failed to build headers to update channel", err)
			return
		}
		userToken := os.Getenv("TWITCH_USER_TOKEN")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+userToken)
		req.Header.Set("Client-Id", headers.ClientID)

		const maxRetries = 3
		client := &http.Client{}
		for i := 0; i < maxRetries; i++ {
			res, err := client.Do(req)
			if err != nil {
				a.Log.Error("Request could not be sent to update channel", err)
				return
			}
			defer res.Body.Close()
			if res.StatusCode == http.StatusOK {
				a.Log.Info("Channel updated successfully")
				return
			}
			if res.StatusCode == http.StatusBadRequest {
				a.Log.Error("Received a bad request", errUpdateChannel)
			}
		}
		a.Log.Error("Failed to update channel after retries", errUpdateChannel)
	}
}
