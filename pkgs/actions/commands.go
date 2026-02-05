// Package actions handles Twitch chat commands and actions
package actions

import (
	"bytes"
	"context"
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
	"go.opentelemetry.io/otel/attribute"
)

const (
	messageEndpoint  = "https://api.twitch.tv/helix/chat/messages"
	channelsEndpoint = "https://api.twitch.tv/helix/channels"
	userID           = "1792311"
	softwareID       = 1469308723
)

var (
	errUpdateChannel = errors.New("updating channel info")
	errUnauthorized  = errors.New("401 unauthorized: token expired")
)

// Actions handles all Twitch chat actions and commands
type Actions struct {
	Log     *telemetry.CustomLogger
	Secrets *secrets.SecretService
	Spotify *spotify.Spotify
}

// NewActions creates a new Actions instance
func NewActions(secretService *secrets.SecretService) *Actions {
	logger := telemetry.NewLogger("actions")
	spotifyClient := spotify.NewSpotify()
	return &Actions{
		Log:     logger,
		Secrets: secretService,
		Spotify: spotifyClient,
	}
}

// ParseMessage Parses the incoming messages from stream
func (a *Actions) ParseMessage(msg subscriptions.ChatMessageEvent) {
	ctx := context.Background()
	payload := fmt.Sprintf("%s: %s", msg.Event.ChatterUserName, msg.Event.Message.Text)
	a.Log.Chat(payload)
	// Simple commands
	switch msg.Event.Message.Text {
	case "!commands":
		telemetry.IncrementCommandExecuted(ctx, "commands")
		_ = a.SendMessage("!github, !dotfiles, !song, !social, !blog, !youtube ")
	case "!github":
		telemetry.IncrementCommandExecuted(ctx, "github")
		_ = a.SendMessage("https://links.mvaldes.dev/gh")
	case "!dotfiles":
		telemetry.IncrementCommandExecuted(ctx, "dotfiles")
		_ = a.SendMessage("https://links.mvaldes.dev/dotfiles")
	case "!test":
		telemetry.IncrementCommandExecuted(ctx, "test")
		_ = a.SendMessage("Test Me")
	case "!social":
		telemetry.IncrementCommandExecuted(ctx, "social")
		_ = a.SendMessage("https://links.mvaldes.dev/twitter")
	case "!blog":
		telemetry.IncrementCommandExecuted(ctx, "blog")
		_ = a.SendMessage("https://mvaldes.dev")
	case "!discord":
		telemetry.IncrementCommandExecuted(ctx, "discord")
		_ = a.SendMessage("https://links.mvaldes.dev/discord")
	case "!youtube":
		telemetry.IncrementCommandExecuted(ctx, "youtube")
		_ = a.SendMessage("https://links.mvaldes.dev/youtube")
	case "!song":
		telemetry.IncrementCommandExecuted(ctx, "song")
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
		telemetry.IncrementCommandExecuted(ctx, "today")
		a.Log.Info("Today command running")
		a.updateChannel(msg)
	}
}

// SendMessage sends a message to the Twitch chat room.
// On 401 Unauthorized, it triggers a token refresh and retries once.
func (a *Actions) SendMessage(text string) error {
	ctx := context.Background()
	_, span := telemetry.StartExternalSpan(ctx, "twitch.send_message", "twitch", "send_message")
	defer span.End()

	err := a.sendMessageInternal(ctx, text)
	if err == nil {
		telemetry.IncrementMessageSent(ctx, "success")
		return nil
	}

	// If we got a 401, refresh the token and retry once
	if errors.Is(err, errUnauthorized) {
		a.Log.Info("Got 401 sending message, refreshing app token and retrying")
		telemetry.AddSpanAttributes(span, attribute.Bool("token.refreshed_on_401", true))
		telemetry.IncrementTokenRefreshOn401(ctx, "send_message")
		if refreshErr := a.Secrets.RefreshAppTokenAndStore(); refreshErr != nil {
			a.Log.Error("Failed to refresh app token after 401", refreshErr)
			telemetry.RecordError(span, refreshErr)
			telemetry.IncrementMessageSent(ctx, "error")
			return err
		}
		retryErr := a.sendMessageInternal(ctx, text)
		if retryErr != nil {
			telemetry.RecordError(span, retryErr)
			telemetry.IncrementMessageSent(ctx, "error")
		} else {
			telemetry.IncrementMessageSent(ctx, "success")
		}
		return retryErr
	}

	telemetry.RecordError(span, err)
	telemetry.IncrementMessageSent(ctx, "error")
	return err
}

func (a *Actions) sendMessageInternal(ctx context.Context, text string) error {
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

	if res.StatusCode == http.StatusUnauthorized {
		a.Log.Info("Received 401 Unauthorized while sending message")
		return errUnauthorized
	}

	if res.StatusCode != http.StatusOK {
		a.Log.Info("Unexpected status code while sending message, response: " + strconv.Itoa(res.StatusCode))
		return fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	return nil
}

func (a *Actions) updateChannel(action subscriptions.ChatMessageEvent) {
	ctx := context.Background()
	_, span := telemetry.StartExternalSpan(ctx, "twitch.update_channel", "twitch", "update_channel")
	defer span.End()

	a.Log.Info("Changing the channel information")
	// Check if user is me so I can update the channel
	if action.Event.BroadcasterUserID != userID {
		return
	}

	// Build the new payload
	splitMsg := strings.Split(action.Event.Message.Text, " ")
	msg := strings.Join(splitMsg[1:], " ")
	payloadBody := fmt.Sprintf(`{
      "game_id":"%v",
      "title":"🚨[Devops]🚨- %v",
      "tags":["devops","Español","SpanishAndEnglish","coding","neovim","k8s","terraform","go","homelab", "nix", "gaming"],
      "broadcaster_language":"es"}`,
		softwareID, msg)
	a.Log.Info("Today Command Ran")

	const maxAttempts = 2
	for attempt := 0; attempt < maxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, "PATCH", "https://api.twitch.tv/helix/channels?broadcaster_id="+userID, bytes.NewBuffer([]byte(payloadBody)))
		if err != nil {
			a.Log.Error("Could not form request to update channel info", err)
			telemetry.RecordError(span, err)
			return
		}

		headers, err := a.Secrets.BuildSecretHeaders()
		if err != nil {
			a.Log.Error("Failed to build headers to update channel", err)
			telemetry.RecordError(span, err)
			return
		}
		userToken, err := a.Secrets.GetUserToken()
		if err != nil {
			a.Log.Error("Failed to get user token from cache", err)
			telemetry.RecordError(span, err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+userToken)
		req.Header.Set("Client-Id", headers.ClientID)

		client := &http.Client{}
		res, err := client.Do(req)
		if err != nil {
			a.Log.Error("Request could not be sent to update channel", err)
			telemetry.RecordError(span, err)
			return
		}
		_ = res.Body.Close()

		telemetry.SetSpanStatus(span, res.StatusCode)

		if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusNoContent {
			a.Log.Info("Channel updated successfully")
			return
		}

		if res.StatusCode == http.StatusUnauthorized && attempt == 0 {
			a.Log.Info("Got 401 updating channel, refreshing user token and retrying")
			telemetry.AddSpanAttributes(span, attribute.Bool("token.refreshed_on_401", true))
			telemetry.IncrementTokenRefreshOn401(ctx, "update_channel")
			if refreshErr := a.Secrets.RefreshUserTokenAndStore(); refreshErr != nil {
				a.Log.Error("Failed to refresh user token after 401", refreshErr)
				telemetry.RecordError(span, refreshErr)
				return
			}
			continue
		}

		if res.StatusCode == http.StatusBadRequest {
			a.Log.Error("Received a bad request", errUpdateChannel)
			telemetry.RecordError(span, errUpdateChannel)
			return
		}

		unexpectedErr := fmt.Errorf("unexpected status: %d", res.StatusCode)
		a.Log.Error("Unexpected status updating channel", unexpectedErr)
		telemetry.RecordError(span, unexpectedErr)
		return
	}
}
