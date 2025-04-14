// Package discord interacts with discord api to send messages to a channel
package discord

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/mvaldes14/twitch-bot/pkgs/telemetry"
)

// Discord struct to hold the logger
type Discord struct {
	Log *telemetry.CustomLogger
}

// NewDiscord create a new discord instance
func NewDiscord() *Discord {
	logger := *telemetry.NewLogger("discord")
	return &Discord{
		Log: &logger,
	}
}

// NotifyChannel sends a message to a discord channel
func (d *Discord) NotifyChannel(msg string) error {
	d.Log.Info("Sending message to discord")
	url := os.Getenv("DISCORD_WEBHOOK")
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
		err := errors.New("Error sending message to discord")
		d.Log.Error("Error sending message to discord", err)
		return err
	}
	return nil
}
