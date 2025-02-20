// Package discord interacts with discord api to send messages to a channel
package discord

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"os"
)

// NotifyChannel sends a message to a discord channel
func NotifyChannel(msg string) error {
	fmt.Println("Sending message to discord")
	url := os.Getenv("DISCORD_WEBHOOK")
	payload := fmt.Sprintf(`{"content": "%s"}`, msg)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(payload)))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		fmt.Println(err)
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
		return errors.New("Error sending message to discord")
	}
	return nil

}
