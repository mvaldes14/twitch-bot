// package discord interacts with discord api to send messages to a channel
package discord

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
)

// NotifyChannel sends a message to a discord channel
func NotifyChannel(msg string) (string, error) {
	fmt.Println("Sending message to discord")
	url := os.Getenv("DISCORD_WEBHOOK")
	payload := fmt.Sprintf(`{"content": "%s"}`, msg)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(payload)))
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
