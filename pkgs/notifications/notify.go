// Package notifications interacts with discord/gotify api to send messages to a channel
package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"

	"github.com/mvaldes14/twitch-bot/pkgs/httpclient"
	"github.com/mvaldes14/twitch-bot/pkgs/telemetry"
)

const (
	discordWebhookURL = "DISCORD_WEBHOOK"
	gotifyURL         = "https://gotify.mvaldes.dev/message"
	gotifyAppToken    = "GOTIFY_APPLICATION_TOKEN"
)

var (
	errMessageDiscord = errors.New("error sending message to discord")
	errMessageGotify  = errors.New("error sending message to gotify")
)

// NotificationService struct to hold the properties
type NotificationService struct {
	Log    *telemetry.CustomLogger
	Client *http.Client
}

// NewNotificationService returns a new instance of NotificationService
func NewNotificationService() *NotificationService {
	return &NotificationService{
		Log:    telemetry.NewLogger("discord"),
		Client: httpclient.Shared,
	}
}

// SendNotification sends a message to discord and gotify.
// Both are attempted on a best-effort basis; the returned error aggregates
// every delivery that failed so a discord-only failure is not swallowed.
func (n *NotificationService) SendNotification(ctx context.Context, msg string) error {
	return errors.Join(
		n.sendDiscord(ctx, msg),
		n.sendGotify(ctx, msg),
	)
}

// sendDiscord posts the message to the configured discord webhook.
func (n *NotificationService) sendDiscord(ctx context.Context, msg string) error {
	n.Log.Info("Sending message to discord")
	url := os.Getenv(discordWebhookURL)

	// Marshal rather than format so quotes and control characters in msg
	// cannot break the payload.
	payload, err := json.Marshal(struct {
		Content string `json:"content"`
	}{Content: msg})
	if err != nil {
		n.Log.Error("Failed to marshal payload for discord", err)
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payload))
	if err != nil {
		n.Log.Error("Failed to generate payload for discord", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.Client.Do(req)
	if err != nil {
		n.Log.Error("Failed to send discord request", err)
		telemetry.IncrementNotificationSent(ctx, "discord", "error")
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		n.Log.Error("ERROR", errMessageDiscord)
		telemetry.IncrementNotificationSent(ctx, "discord", "error")
		return fmt.Errorf("%w: status %d", errMessageDiscord, resp.StatusCode)
	}

	telemetry.IncrementNotificationSent(ctx, "discord", "success")
	return nil
}

// sendGotify posts the message to gotify as a multipart form.
func (n *NotificationService) sendGotify(ctx context.Context, msg string) error {
	n.Log.Info("Sending message to gotify")
	token := os.Getenv(gotifyAppToken)
	if token == "" {
		n.Log.Error("Gotify token not set", errMessageGotify)
		telemetry.IncrementNotificationSent(ctx, "gotify", "error")
		return errMessageGotify
	}

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	if err := w.WriteField("title", "Twitch Bot Notification"); err != nil {
		return err
	}
	if err := w.WriteField("message", msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s?token=%s", gotifyURL, token), &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := n.Client.Do(req)
	if err != nil {
		telemetry.IncrementNotificationSent(ctx, "gotify", "error")
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		n.Log.Error("Error sending message to gotify", errMessageGotify)
		telemetry.IncrementNotificationSent(ctx, "gotify", "error")
		return fmt.Errorf("%w: status %d", errMessageGotify, resp.StatusCode)
	}

	telemetry.IncrementNotificationSent(ctx, "gotify", "success")
	n.Log.Info("Sent message to gotify with status code", resp.StatusCode)
	return nil
}
