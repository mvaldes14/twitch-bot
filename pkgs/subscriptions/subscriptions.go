package subscriptions

import (
	"fmt"

	"github.com/mvaldes14/twitch-bot/pkgs/types"
)

const (
	userID      = "1792311"
	callbackURL = "https://evil-buses-argue.loca.lt/"
	secret      = "superSecret123"
)

// Create a subscription
// TODO: Respond to challenge
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
        "callback":"%v","secret":"%v"
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
        "callback": "%v","secret": "%v"
      }
    }`, subType.Type, subType.Version, userID, userID, callbackURL, secret)

	case "sub":
		payload = fmt.Sprintf(`{
      "type": "%v",
      "version": "%v",
      "condition": {
          "broadcaster_user_id": "%v"
      },
      "transport": {
          "method": "webhook",
          "callback": "%v",
          "secret": "%v"
      }
    }`, subType.Type, subType.Version, userID, callbackURL, secret)

	}
	return payload
}
