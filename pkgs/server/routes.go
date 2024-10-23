package server

import (
	"encoding/json"
	"fmt"
	"github.com/mvaldes14/twitch-bot/pkgs/commands"
	"github.com/mvaldes14/twitch-bot/pkgs/obs"
	"github.com/mvaldes14/twitch-bot/pkgs/spotify"
	"github.com/mvaldes14/twitch-bot/pkgs/subscriptions"
	"github.com/mvaldes14/twitch-bot/pkgs/types"
	"github.com/mvaldes14/twitch-bot/pkgs/utils"
	"io"
	"net/http"
)

func deleteHandler(_ http.ResponseWriter, _ *http.Request) {
	subsList := subscriptions.GetSubscriptions()
	subscriptions.CleanSubscriptions(subsList)
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Write([]byte("msg: OK\n"))
}

func listHandler(_ http.ResponseWriter, _ *http.Request) {
	subsList := subscriptions.GetSubscriptions()
	logger.Info("Current Subscription List")
	for _, sub := range subsList.Data {
		logger.Info("Status:" + sub.Status + " ,Type:" + sub.Type)
	}
}

func createHandler(_ http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	subType := query.Get("type")
	switch subType {
	case "chat":
		// Generate subscription type for chats
		subType := types.SubscriptionType{
			Name:    "chat",
			Version: "1",
			Type:    "channel.chat.message",
		}
		payload := utils.GeneratePayload(subType)
		subscriptions.CreateSubscription(payload)
	case "follow":
		// Generate subscription type for follow
		chatSubType := types.SubscriptionType{
			Name:    "follow",
			Version: "2",
			Type:    "channel.follow",
		}
		payload := utils.GeneratePayload(chatSubType)
		subscriptions.CreateSubscription(payload)
	case "subscription":
		// Generate subscription type for subscriptionsser
		chatSubType := types.SubscriptionType{
			Name:    "subscribe",
			Version: "1",
			Type:    "channel.subscribe",
		}
		payload := utils.GeneratePayload(chatSubType)
		subscriptions.CreateSubscription(payload)
	case "cheer":
		// Generate subscription type for subscriptions
		chatSubType := types.SubscriptionType{
			Name:    "cheer",
			Version: "1",
			Type:    "channel.cheer",
		}
		payload := utils.GeneratePayload(chatSubType)
		subscriptions.CreateSubscription(payload)
	case "reward":
		// Generate subscription type for subscriptions
		chatSubType := types.SubscriptionType{
			Name:    "reward",
			Version: "1",
			Type:    "channel.channel_points_custom_reward_redemption.add",
		}
		payload := utils.GeneratePayload(chatSubType)
		subscriptions.CreateSubscription(payload)
	}
}

func handleHeaders(w http.ResponseWriter, r *http.Request) {
	eventHeaderType := r.Header.Get("Twitch-Eventsub-Message-Type")
	if eventHeaderType == "webhook_callback_verification" {
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
}

func chatHandler(w http.ResponseWriter, r *http.Request) {
	handleHeaders(w, r)
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

func followHandler(w http.ResponseWriter, r *http.Request) {
	handleHeaders(w, r)
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

func subHandler(w http.ResponseWriter, r *http.Request) {
	handleHeaders(w, r)
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

func cheerHandler(w http.ResponseWriter, r *http.Request) {
	handleHeaders(w, r)
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

func rewardHandler(w http.ResponseWriter, r *http.Request) {
	handleHeaders(w, r)
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

func testHandler(w http.ResponseWriter, r *http.Request) {
	logger.Info("Test")
	test := utils.GenerateNewToken()
	utils.StoreNewTokens(test)
}
