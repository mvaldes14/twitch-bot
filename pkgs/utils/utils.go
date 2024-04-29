// Package utils Holds all of the utilities used by the bot
package utils

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/mvaldes14/twitch-bot/pkgs/types"
)

const (
	userID      = "1792311"
	callbackURL = "https://bot.mvaldes.dev"
	secret      = "superSecret123"
	url         = "https://api.twitch.tv/helix/eventsub/subscriptions"
)

// BuildHeaders Returns the secrets from env variables to build headers for requests
func BuildHeaders() types.RequestHeader {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}
	token := os.Getenv("TWITCH_TOKEN")
	clientID := os.Getenv("TWITCH_CLIENT_ID")
	return types.RequestHeader{
		Token:    token,
		ClientID: clientID,
	}
}

// GeneratePayload Builds the payload for each subscription type
func GeneratePayload(subType types.SubscriptionType) string {
	var payload string

	switch subType.Name {
	case "chat":
		payload = fmt.Sprintf(`{
      "type":"%v",
      "version":"%v",
      "condition":{
        "broadcaster_user_id":"%v",
        "user_id": "%v"
      },
      "transport": {
        "method":"webhook",
        "callback":"%v/chat","secret":"%v"
      }
    }`, subType.Type, subType.Version, userID, userID, callbackURL, secret)

	case "follow":
		payload = fmt.Sprintf(`{
      "type": "%v",
      "version": "%v",
      "condition": {
        "broadcaster_user_id": "%v",
        "moderator_user_id": "%v"
      },
      "transport": {
        "method": "webhook",
        "callback": "%v/follow",
        "secret": "%v"
      }
    }`, subType.Type, subType.Version, userID, userID, callbackURL, secret)

	case "subscribe":
		payload = fmt.Sprintf(`{
      "type": "%v",
      "version": "%v",
      "condition": {
          "broadcaster_user_id": "%v"
      },
      "transport": {
          "method": "webhook",
          "callback": "%v/sub",
          "secret": "%v"
      }
    }`, subType.Type, subType.Version, userID, callbackURL, secret)

	}
	return payload
}
