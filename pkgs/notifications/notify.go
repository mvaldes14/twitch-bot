// Package notifications interacts with discord/gotify api to send messages to a channel
package notifications

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"

	"github.com/mvaldes14/twitch-bot/pkgs/telemetry"
)

const (
	discordWebhookURL = "DISCORD_WEBHOOK"
	gotifyURL         = "https://gotify.mvaldes.dev/message"
	gotifyAppToken    = "GOTIFY_APPLICATION_TOKEN"
)

var (
	errMessageDiscord = errors.New("Error sending message to discord")
	errMessageGotify  = errors.New("Error sending message to gotify")
)

// NotificationService struct to hold the properties
type NotificationService struct {
	Log    telemetry.CustomLogger
	Client http.Client
}

// NewNotificationService returns a new instance of NotificationService
func NewNotificationService() *NotificationService {
	logger := *telemetry.NewLogger("discord")
	client := &http.Client{}
	return &NotificationService{
		Log:    logger,
		Client: *client,
	}
}

// SendNotification sends a message to a discord channel
func (n *NotificationService) SendNotification(msg string) error {
	ctx := context.Background()
	n.Log.Info("Sending message to discord")
	url := os.Getenv(discordWebhookURL)
	payload := fmt.Sprintf(`{"content": "%s"}`, msg)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer([]byte(payload)))
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
	} else {
		telemetry.IncrementNotificationSent(ctx, "discord", "success")
	}

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

	req, err = http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s?token=%s", gotifyURL, token), &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err = n.Client.Do(req)
	if err != nil {
		telemetry.IncrementNotificationSent(ctx, "gotify", "error")
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		n.Log.Error("Error sending message to gotify", errMessageGotify)
		telemetry.IncrementNotificationSent(ctx, "gotify", "error")
		return errMessageGotify
	}
	telemetry.IncrementNotificationSent(ctx, "gotify", "success")
	n.Log.Info("Sent message to gotify with status code", resp.StatusCode)
	return nil
}
