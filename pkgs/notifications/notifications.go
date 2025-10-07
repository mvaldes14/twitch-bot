// Package notifications interacts with discord/gotify api to send messages to a channel
package notifications

import (
	"bytes"
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
	Logger telemetry.BotLogger
	Client http.Client
}

// NewNotificationService returns a new instance of NotificationService
func NewNotificationService() *NotificationService {
	logger := *telemetry.NewLogger("discord")
	client := &http.Client{}
	return &NotificationService{
		Logger: logger,
		Client: *client,
	}
}

// SendNotification sends a message to a discord channel
func (n *NotificationService) SendNotification(msg string) error {
	n.Logger.Info("Sending message to discord")
	url := os.Getenv(discordWebhookURL)
	payload := fmt.Sprintf(`{"content": "%s"}`, msg)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(payload)))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		n.Logger.Error(err)
		return err
	}

	resp, err := n.Client.Do(req)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		n.Logger.Error(errMessageDiscord)
		// return errMessageDiscord
	}

	n.Logger.Info("Sending message to gotify")
	token := os.Getenv(gotifyAppToken)
	if token == "" {
		n.Logger.Error(errMessageGotify)
	}
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	w.WriteField("title", "Twitch Bot Notification")
	w.WriteField("message", msg)
	w.Close()

	req, err = http.NewRequest("POST", fmt.Sprintf("%s?token=%s", gotifyURL, token), &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	if err != nil {
		return err
	}
	resp, err = n.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		n.Logger.Error(errMessageGotify)
		return errMessageGotify
	}
	n.Logger.Info("Sent message to gotify with status code: " + string(resp.StatusCode))
	return nil
}
