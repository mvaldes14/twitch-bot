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

	"github.com/mvaldes14/twitch-bot/pkgs/httpclient"
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

// channelTags are the tags applied to the channel by the !today command.
var channelTags = []string{
	"devops", "Español", "SpanishAndEnglish", "coding", "neovim",
	"k8s", "terraform", "go", "homelab", "nix", "gaming",
}

// Actions handles all Twitch chat actions and commands
type Actions struct {
	Log     *telemetry.CustomLogger
	Secrets *secrets.SecretService
	Spotify *spotify.Spotify
}

// NewActions creates a new Actions instance.
// It fails if the Spotify client cannot be constructed.
func NewActions(secretService *secrets.SecretService) (*Actions, error) {
	logger := telemetry.NewLogger("actions")
	spotifyClient, err := spotify.NewSpotify()
	if err != nil {
		return nil, fmt.Errorf("actions require a spotify client: %w", err)
	}
	return &Actions{
		Log:     logger,
		Secrets: secretService,
		Spotify: spotifyClient,
	}, nil
}

// say sends a chat message and logs any failure. Chat replies are best effort:
// a failed reply is reported but must not abort processing of the message.
func (a *Actions) say(ctx context.Context, text string) {
	if err := a.SendMessage(ctx, text); err != nil {
		a.Log.Error("Failed to send chat message", err)
	}
}

// ParseMessage Parses the incoming messages from stream
func (a *Actions) ParseMessage(ctx context.Context, msg subscriptions.ChatMessageEvent) {
	payload := fmt.Sprintf("%s: %s", msg.Event.ChatterUserName, msg.Event.Message.Text)
	a.Log.Chat(payload)
	// Simple commands
	switch msg.Event.Message.Text {
	case "!commands":
		telemetry.IncrementCommandExecuted(ctx, "commands")
		a.say(ctx, "!github, !dotfiles, !song, !social, !blog, !youtube ")
	case "!github":
		telemetry.IncrementCommandExecuted(ctx, "github")
		a.say(ctx, "https://links.mvaldes.dev/gh")
	case "!dotfiles":
		telemetry.IncrementCommandExecuted(ctx, "dotfiles")
		a.say(ctx, "https://links.mvaldes.dev/dotfiles")
	case "!test":
		telemetry.IncrementCommandExecuted(ctx, "test")
		a.say(ctx, "Test Me")
	case "!social":
		telemetry.IncrementCommandExecuted(ctx, "social")
		a.say(ctx, "https://links.mvaldes.dev/twitter")
	case "!blog":
		telemetry.IncrementCommandExecuted(ctx, "blog")
		a.say(ctx, "https://mvaldes.dev")
	case "!discord":
		telemetry.IncrementCommandExecuted(ctx, "discord")
		a.say(ctx, "https://links.mvaldes.dev/discord")
	case "!youtube":
		telemetry.IncrementCommandExecuted(ctx, "youtube")
		a.say(ctx, "https://links.mvaldes.dev/youtube")
	case "!song":
		telemetry.IncrementCommandExecuted(ctx, "song")
		song, err := a.Spotify.GetSong(ctx)
		if err != nil {
			a.Log.Error("Failed to get current song", err)
			a.say(ctx, "Sorry, couldn't get the current song")
			return
		}
		if song.Item.Name == "" || len(song.Item.Artists) == 0 {
			a.say(ctx, "No song currently playing")
			return
		}
		songMsg := fmt.Sprintf("Now playing: %v - %v", song.Item.Artists[0].Name, song.Item.Name)
		a.Log.Info(songMsg)
		a.say(ctx, songMsg)
	}
	// Complex commands
	if strings.HasPrefix(msg.Event.Message.Text, "!today") {
		telemetry.IncrementCommandExecuted(ctx, "today")
		a.Log.Info("Today command running")
		a.updateChannel(ctx, msg)
	}
}

// SendMessage sends a message to the Twitch chat room.
// On 401 Unauthorized, it triggers a token refresh and retries once.
func (a *Actions) SendMessage(ctx context.Context, text string) error {
	ctx, span := telemetry.StartExternalSpan(ctx, "twitch.send_message", "twitch", "send_message")
	defer span.End()

	// sendMessageInternal builds and validates the credentials for each
	// attempt, so the retry below picks up a freshly refreshed token.
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

	// Validate headers exist before making API call
	headers, err := a.Secrets.BuildSecretHeaders()
	if err != nil {
		headerErr := fmt.Errorf("failed to build required headers for sending message: %w", err)
		a.Log.Error("Cannot send message - missing required credentials", headerErr)
		return headerErr
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+headers.Token)
	req.Header.Set("Client-Id", headers.ClientID)

	res, err := httpclient.Shared.Do(req)
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

// channelUpdate is the payload sent to the Twitch channels endpoint.
type channelUpdate struct {
	GameID              string   `json:"game_id"`
	Title               string   `json:"title"`
	Tags                []string `json:"tags"`
	BroadcasterLanguage string   `json:"broadcaster_language"`
}

func (a *Actions) updateChannel(ctx context.Context, action subscriptions.ChatMessageEvent) {
	ctx, span := telemetry.StartExternalSpan(ctx, "twitch.update_channel", "twitch", "update_channel")
	defer span.End()

	a.Log.Info("Changing the channel information")
	// Check if user is me so I can update the channel
	if action.Event.BroadcasterUserID != userID {
		return
	}

	// Build the new payload. The title comes from chat, so it is marshalled
	// rather than formatted into a JSON literal.
	splitMsg := strings.Split(action.Event.Message.Text, " ")
	msg := strings.Join(splitMsg[1:], " ")
	payloadBody, err := json.Marshal(channelUpdate{
		GameID:              strconv.Itoa(softwareID),
		Title:               fmt.Sprintf("🚨[Devops]🚨- %s", msg),
		Tags:                channelTags,
		BroadcasterLanguage: "es",
	})
	if err != nil {
		a.Log.Error("Could not marshal channel update payload", err)
		telemetry.RecordError(span, err)
		return
	}
	a.Log.Info("Today Command Ran")

	const maxAttempts = 2
	for attempt := 0; attempt < maxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, "PATCH", channelsEndpoint+"?broadcaster_id="+userID, bytes.NewBuffer(payloadBody))
		if err != nil {
			a.Log.Error("Could not form request to update channel info", err)
			telemetry.RecordError(span, err)
			return
		}

		// Validate headers exist before making API call
		headers, err := a.Secrets.BuildSecretHeaders()
		if err != nil {
			headerErr := fmt.Errorf("failed to build required headers for updating channel: %w", err)
			a.Log.Error("Cannot update channel - missing required credentials", headerErr)
			telemetry.RecordError(span, headerErr)
			return
		}

		userToken, err := a.Secrets.GetUserToken()
		if err != nil {
			headerErr := fmt.Errorf("failed to get user token from cache: %w", err)
			a.Log.Error("Cannot update channel - user token missing from cache", headerErr)
			telemetry.RecordError(span, headerErr)
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+userToken)
		req.Header.Set("Client-Id", headers.ClientID)

		res, err := httpclient.Shared.Do(req)
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
