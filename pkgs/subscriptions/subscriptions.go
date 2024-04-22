package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

const URL string = "https://api.twitch.tv/helix/eventsub/subscriptions"

// Create a subscription
func subscribeEvent() {
	payload := []byte(`{"type":"channel.chat.message","version":"1","condition":{"broadcaster_user_id":"1792311", "user_id": "1792311"},"transport":{"method":"webhook","callback":"https://dirty-garlics-pull.loca.lt/","secret":"s3cre77890ab"}}`)
	req, err := http.NewRequest("POST", URL, bytes.NewBuffer(payload))

	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer redacted")
	req.Header.Set("Client-Id", "redacted")
	// Create an HTTP client
	client := &http.Client{}

	// Send the request and get the response
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return
	}
	defer resp.Body.Close()

	// Print the response status code
	fmt.Println("Response Status:", resp.Status)
	fmt.Println("Response Header:", resp.Header)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	fmt.Println("Response Body:", string(body))
}

func main() {

	subscribeEvent()
}
