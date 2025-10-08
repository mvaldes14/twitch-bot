// Package notifications interacts with discord/gotify api to send messages to a channel
package notifications

import (
	"bytes"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"

	"github.com/mvaldes14/twitch-bot/pkgs/service"
)

const (
	discordWebhookURL = "DISCORD_WEBHOOK"
	gotifyURL         = "https://gotify.mvaldes.dev/message"
	gotifyAppToken    = "GOTIFY_APPLICATION_TOKEN"
)

// TODO: Think of all the possible errors we can throw based on the service
var (
	errMessageDiscord = errors.New("Error sending message to discord")
	errMessageGotify  = errors.New("Error sending message to gotify")
	errMissingDiscord = errors.New("Missing discord webhook URL in environment")
	errMissingGotify  = errors.New("Missing gotify application token in environment")
)

// NotificationService struct to hold the properties
type NotificationService struct {
	Service *service.Service
}

// NewNotificationService returns a new instance of NotificationService
func NewNotificationService() *NotificationService {
	service := service.NewService("notifications")
	return &NotificationService{service}
}

// SendNotification sends a message to a discord channel
func (n *NotificationService) SendNotification(msg string) {
	n.Service.Logger.Info("Sending message to discord")
	url := os.Getenv(discordWebhookURL)
	if url == "" {
		n.Service.Logger.Error(errMissingDiscord)
	}
	payload := fmt.Sprintf(`{"content": "%s"}`, msg)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(payload)))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		n.Service.Logger.Error(err)
	}

	resp, err := n.Service.Client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		n.Service.Logger.Error(errMessageDiscord)
	}

	n.Service.Logger.Info("Sending message to gotify")
	token := os.Getenv(gotifyAppToken)
	if token == "" {
		n.Service.Logger.Error(errMissingGotify)
	}
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	w.WriteField("title", "Twitch Bot Notification")
	w.WriteField("message", msg)
	w.Close()

	req, err = http.NewRequest("POST", fmt.Sprintf("%s?token=%s", gotifyURL, token), &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	if err != nil {
		n.Service.Logger.Error(err)
	}
	resp, err = n.Service.Client.Do(req)
	if err != nil {
		n.Service.Logger.Error(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		n.Service.Logger.Error(errMessageGotify)
	}
	n.Service.Logger.Info("Sent message to gotify with status code: " + string(resp.StatusCode))
}
