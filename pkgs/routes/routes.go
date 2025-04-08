// Package routes defines all routes for handlers and functionality
package routes

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/mvaldes14/twitch-bot/pkgs/discord"
	"github.com/mvaldes14/twitch-bot/pkgs/secrets"
	"github.com/mvaldes14/twitch-bot/pkgs/spotify"
	"github.com/mvaldes14/twitch-bot/pkgs/subscriptions"
	"github.com/mvaldes14/twitch-bot/pkgs/types"
)

// Router is the struct that handles all routes
type Router struct {
	Log     *slog.Logger
	Subs    subscriptions.SubscriptionsMethods
	Secrets secrets.SecretManager
}

// NewRouter creates a new router
func NewRouter(logger *slog.Logger, subs subscriptions.SubscriptionsMethods, secretService secrets.SecretManager) *Router {
	return &Router{
		Log:     logger,
		Subs:    subs,
		Secrets: secretService,
	}
}

// CheckAuthAdmin validates for headers for admin routes
func (rt *Router) CheckAuthAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := os.Getenv("ADMIN_TOKEN")
		if token == "" {
			w.WriteHeader(http.StatusUnauthorized)
		}
		if r.Header.Get("Token") == token {
			next.ServeHTTP(w, r)
		} else {
			w.WriteHeader(http.StatusUnauthorized)
		}
	})
}

// MiddleWareRoute checks for headers in all requests
func (rt *Router) MiddleWareRoute(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Twitch-Eventsub-Message-Type") == "webhook_callback_verification" {
			rt.respondToChallenge(w, r)
		} else {
			next.ServeHTTP(w, r)
		}
	})
}

// HANDLERS
// respondToChallenge responds to challenge for a subscription on twitch eventsub
func (rt *Router) respondToChallenge(w http.ResponseWriter, r *http.Request) {
	rt.Log.Info("Responding to challenge")
	var challengeResponse types.SubscribeEvent
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return
	}
	defer r.Body.Close()
	json.Unmarshal(body, &challengeResponse)
	w.Header().Add("Content-Type", "plain/text")
	w.Write([]byte(challengeResponse.Challenge))
	rt.Log.Info("Response sent")
}

// DeleteHandler deletes all subscriptions
func (rt *Router) DeleteHandler(_ http.ResponseWriter, _ *http.Request) {
	subsList := rt.Subs.GetSubscriptions()
	rt.Subs.CleanSubscriptions(subsList)
}

// HealthHandler returns a healthy message
func (rt *Router) HealthHandler(w http.ResponseWriter, _ *http.Request) {
	json.NewEncoder(w).Encode("Healthy")
}

// ListHandler returns the current subscription list
func (rt *Router) ListHandler(w http.ResponseWriter, _ *http.Request) {
	subsList := rt.Subs.GetSubscriptions()
	rt.Log.Info("Current Subscription List")
	for _, sub := range subsList.Data {
		rt.Log.Info("Status:" + sub.Status + " ,Type:" + sub.Type)
		subItem := fmt.Sprintf("ID:%s, Status: %s, Type: %s\n", sub.ID, sub.Status, sub.Type)
		w.Write([]byte(subItem))
	}
}

// TODO: Figure a way to respond to 401s which means the token is expired

// CreateHandler creates a subscription based on the parameter
func (rt *Router) CreateHandler(_ http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	subType := query.Get("type")
	switch subType {
	case "chat":
		subTypeForm := types.SubscriptionType{
			Name:    "chat",
			Version: "1",
			Type:    "channel.chat.message",
		}
		payload := GeneratePayload(subTypeForm)
		rt.Subs.CreateSubscription(payload)
	case "follow":
		subTypeForm := types.SubscriptionType{
			Name:    "follow",
			Version: "2",
			Type:    "channel.follow",
		}
		payload := GeneratePayload(subTypeForm)
		rt.Subs.CreateSubscription(payload)
	case "subscription":
		subTypeForm := types.SubscriptionType{
			Name:    "subscribe",
			Version: "1",
			Type:    "channel.subscribe",
		}
		payload := GeneratePayload(subTypeForm)
		rt.Subs.CreateSubscription(payload)
	case "cheer":
		subTypeForm := types.SubscriptionType{
			Name:    "cheer",
			Version: "1",
			Type:    "channel.cheer",
		}
		payload := GeneratePayload(subTypeForm)
		rt.Subs.CreateSubscription(payload)
	case "reward":
		subTypeForm := types.SubscriptionType{
			Name:    "reward",
			Version: "1",
			Type:    "channel.channel_points_custom_reward_redemption.add",
		}
		payload := GeneratePayload(subTypeForm)
		rt.Subs.CreateSubscription(payload)
	case "stream":
		subTypeForm := types.SubscriptionType{
			Name:    "stream",
			Version: "1",
			Type:    "stream.online",
		}
		payload := GeneratePayload(subTypeForm)
		rt.Subs.CreateSubscription(payload)
	}

}

// ChatHandler responds to chat messages
func (rt *Router) ChatHandler(_ http.ResponseWriter, r *http.Request) {
	var chatEvent types.ChatMessageEvent
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return
	}
	defer r.Body.Close()
	json.Unmarshal(body, &chatEvent)
	// Send to parser to respond
	// commands.ParseMessage(chatEvent)
}

// FollowHandler responds to follow events
func (rt *Router) FollowHandler(_ http.ResponseWriter, r *http.Request) {
	var followEventResponse types.FollowEvent
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return
	}
	defer r.Body.Close()
	json.Unmarshal(body, &followEventResponse)
	// Send to chat
	// commands.SendMessage(fmt.Sprintf("Gracias por el follow: %v", followEventResponse.Event.UserName))
}

// SubHandler responds to subscription events
func (rt *Router) SubHandler(_ http.ResponseWriter, r *http.Request) {
	var subEventResponse types.SubscriptionEvent
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return
	}
	defer r.Body.Close()
	json.Unmarshal(body, &subEventResponse)
	// send to chat
	// commands.SendMessage(fmt.Sprintf("Gracias por el sub: %v", subEventResponse.Event.UserName))
}

// CheerHandler responds to cheer events
func (rt *Router) CheerHandler(_ http.ResponseWriter, r *http.Request) {
	var cheerEventResponse types.CheerEvent
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return
	}
	defer r.Body.Close()
	json.Unmarshal(body, &cheerEventResponse)
	// send to chat
	// commands.SendMessage(fmt.Sprintf("Gracias por los bits: %v", cheerEventResponse.Event.UserName))
}

// RewardHandler responds to reward events
func (rt *Router) RewardHandler(_ http.ResponseWriter, r *http.Request) {
	var rewardEventResponse types.RewardEvent
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return
	}
	defer r.Body.Close()
	json.Unmarshal(body, &rewardEventResponse)
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

// TestHandler is used to test if the bot is responding to messages
func (rt *Router) TestHandler(_ http.ResponseWriter, _ *http.Request) {
	test := rt.Secrets.GenerateNewToken()
	rt.Secrets.StoreNewTokens(test)
}

// StreamHandler sends a message to discord
func (rt *Router) StreamHandler(_ http.ResponseWriter, _ *http.Request) {
	err := discord.NotifyChannel("En vivo y en directo @everyone - https://links.mvaldes.dev/stream")
	if err != nil {
		rt.Log.Error("Error", "sending message to discord", err)
	}
	req, err := http.NewRequest("POST", "https://automate.mvaldes.dev/webhook/stream-live", nil)
	if err != nil {
		rt.Log.Error("Error", "could not generate request for x post", err)
	}
	req.Header.Add("Token", os.Getenv("ADMIN_TOKEN"))
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		rt.Log.Error("Error", "Could not send request to webhook for X post", err)
	}
	if resp.StatusCode == 200 {
		rt.Log.Info("Info", "posting tweet to X", " ")
	}
}
