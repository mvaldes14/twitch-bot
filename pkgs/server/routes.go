package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/mvaldes14/twitch-bot/pkgs/commands"
	"github.com/mvaldes14/twitch-bot/pkgs/obs"
	"github.com/mvaldes14/twitch-bot/pkgs/spotify"
	"github.com/mvaldes14/twitch-bot/pkgs/subscriptions"
	"github.com/mvaldes14/twitch-bot/pkgs/types"
	"github.com/mvaldes14/twitch-bot/pkgs/utils"
)

// respondToChallenge responds to challenge for a subscription on twitch eventsub
func respondToChallenge(w http.ResponseWriter, r *http.Request) {
	logger.Info("Responding to challenge")
	var challengeResponse types.SubscribeEvent
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return
	}
	defer r.Body.Close()
	json.Unmarshal(body, &challengeResponse)
	w.Header().Add("Content-Type", "plain/text")
	w.Write([]byte(challengeResponse.Challenge))
	logger.Info("Response sent")
}

// middleWareRoute checks for headers in all requests
func middleWareRoute(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Twitch-Eventsub-Message-Type ") == "webhook_callback_verification" {
			respondToChallenge(w, r)
		} else {
			next.ServeHTTP(w, r)
		}
	})
}

// deleteHandler deletes all subscriptions
func deleteHandler(_ http.ResponseWriter, _ *http.Request) {
	subsList := subscriptions.GetSubscriptions()
	subscriptions.CleanSubscriptions(subsList)
}

// healthHandler returns a healthy message
func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Write([]byte("msg: OK\n"))
}

// listHandler returns the current subscription list
func listHandler(_ http.ResponseWriter, _ *http.Request) {
	subsList := subscriptions.GetSubscriptions()
	logger.Info("Current Subscription List")
	for _, sub := range subsList.Data {
		logger.Info("Status:" + sub.Status + " ,Type:" + sub.Type)
	}
}

// createHandler creates a subscription based on the parameter
func createHandler(_ http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	subType := query.Get("type")
	switch subType {
	case "chat":
		subTypeForm := types.SubscriptionType{
			Name:    "chat",
			Version: "1",
			Type:    "channel.chat.message",
		}
		payload := utils.GeneratePayload(subTypeForm)
		subscriptions.CreateSubscription(payload)
	case "follow":
		subTypeForm := types.SubscriptionType{
			Name:    "follow",
			Version: "2",
			Type:    "channel.follow",
		}
		payload := utils.GeneratePayload(subTypeForm)
		subscriptions.CreateSubscription(payload)
	case "subscription":
		subTypeForm := types.SubscriptionType{
			Name:    "subscribe",
			Version: "1",
			Type:    "channel.subscribe",
		}
		payload := utils.GeneratePayload(subTypeForm)
		subscriptions.CreateSubscription(payload)
	case "cheer":
		subTypeForm := types.SubscriptionType{
			Name:    "cheer",
			Version: "1",
			Type:    "channel.cheer",
		}
		payload := utils.GeneratePayload(subTypeForm)
		subscriptions.CreateSubscription(payload)
	case "reward":
		subTypeForm := types.SubscriptionType{
			Name:    "reward",
			Version: "1",
			Type:    "channel.channel_points_custom_reward_redemption.add",
		}
		payload := utils.GeneratePayload(subTypeForm)
		subscriptions.CreateSubscription(payload)
	}
}

// chatHandler responds to chat messages
func chatHandler(w http.ResponseWriter, r *http.Request) {
	var chatEvent types.ChatMessageEvent
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return
	}
	defer r.Body.Close()
	json.Unmarshal(body, &chatEvent)
	// Send to parser to respond
	commands.ParseMessage(chatEvent)
}

// followHandler responds to follow events
func followHandler(w http.ResponseWriter, r *http.Request) {
	var followEventResponse types.FollowEvent
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return
	}
	defer r.Body.Close()
	json.Unmarshal(body, &followEventResponse)
	// Send to chat
	commands.SendMessage(fmt.Sprintf("Gracias por el follow: %v", followEventResponse.Event.UserName))
}

// subHandler responds to subscription events
func subHandler(w http.ResponseWriter, r *http.Request) {
	var subEventResponse types.SubscriptionEvent
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return
	}
	defer r.Body.Close()
	json.Unmarshal(body, &subEventResponse)
	// send to chat
	commands.SendMessage(fmt.Sprintf("Gracias por el sub: %v", subEventResponse.Event.UserName))
}

// cheerHandler responds to cheer events
func cheerHandler(w http.ResponseWriter, r *http.Request) {
	var cheerEventResponse types.CheerEvent
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return
	}
	defer r.Body.Close()
	json.Unmarshal(body, &cheerEventResponse)
	// send to chat
	commands.SendMessage(fmt.Sprintf("Gracias por los bits: %v", cheerEventResponse.Event.UserName))
}

// rewardHandler responds to reward events
func rewardHandler(w http.ResponseWriter, r *http.Request) {
	var rewardEventResponse types.RewardEvent
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return
	}
	defer r.Body.Close()
	json.Unmarshal(body, &rewardEventResponse)
	if rewardEventResponse.Event.Reward.Title == "Random Sound" {
		obs.Generate("sound")
	}
	if rewardEventResponse.Event.Reward.Title == "Next Song" {
		token := spotify.RefreshToken()
		spotify.NextSong(token)
	}
	if rewardEventResponse.Event.Reward.Title == "Add Song" {
		spotifyURL := rewardEventResponse.Event.UserInput
		token := spotify.RefreshToken()
		spotify.AddToPlaylist(token, spotifyURL)
	}
	if rewardEventResponse.Event.Reward.Title == "Reset Playlist" {
		token := spotify.RefreshToken()
		spotify.DeleteSongPlaylist(token)
	}
}

// testHandler is used to test if the bot is responding to messages
func testHandler(w http.ResponseWriter, r *http.Request) {
	logger.Info("Test")
	test := utils.GenerateNewToken()
	utils.StoreNewTokens(test)
}
