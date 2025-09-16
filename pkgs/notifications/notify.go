// Package notifications interacts with discord/gotify api to send messages to a channel
package notifications

import (
	"bytes"
	"errors"
	"fmt"
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
	errMissingToken   = errors.New("Missing gotify application token")
)

// Discord struct to hold the logger
type Discord struct {
	Log telemetry.CustomLogger
}

// Gotify struct to hold the logger
type Gotify struct {
	Log telemetry.CustomLogger
}

// NewDiscord create a new discord instance
func NewDiscord() *Discord {
	logger := *telemetry.NewLogger("discord")
	return &Discord{
		Log: logger,
	}
}

// NewGotify create a new gotify instance
func NewGotify() *Gotify {
	logger := *telemetry.NewLogger("gotify")
	return &Gotify{
		Log: logger,
	}
}

// NotifyChannel sends a message to a discord channel
func (d *Discord) NotifyChannel(msg string) error {
	d.Log.Info("Sending message to discord")
	url := os.Getenv(discordWebhookURL)
	payload := fmt.Sprintf(`{"content": "%s"}`, msg)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(payload)))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		d.Log.Error("Failed to generate payload for discord", err)
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		d.Log.Error("Error sending message to discord", errMessageDiscord)
		return errMessageDiscord
	}
	return nil
}

// NotifyGotify sends a message to gotify
func (g *Gotify) NotifyGotify(msg string) error {
	token := os.Getenv(gotifyAppToken)
	req, err := http.NewRequest("POST", fmt.Sprintf("%s?token=%s", gotifyURL, token), bytes.NewBuffer([]byte(msg)))
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		g.Log.Error("Error sending message to gotify", errMessageGotify)
		return errMessageGotify
	}
	return nil

}
