package discord

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
)

// func NotifyChannel sends a message to a discord channel
func NotifyChannel(msg string) (string, error) {
	url := os.Getenv("DISCORD_WEBHOOK")
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(msg)))
	if err != nil {
		return "", err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Println("Error sending message to discord")
	}
	return "Ok", nil

}
