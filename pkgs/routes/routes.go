// Package routes defines all routes for handlers and functionality
package routes

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/mvaldes14/twitch-bot/pkgs/actions"
	"github.com/mvaldes14/twitch-bot/pkgs/discord"
	"github.com/mvaldes14/twitch-bot/pkgs/secrets"
	"github.com/mvaldes14/twitch-bot/pkgs/spotify"
	"github.com/mvaldes14/twitch-bot/pkgs/subscriptions"
	"github.com/mvaldes14/twitch-bot/pkgs/telemetry"
)

const (
	adminToken = "ADMIN_TOKEN"
)

var (
	errorTokenNotFound       = errors.New("Token not found for API protected routes")
	errorTokenNotValid       = errors.New("Token not valid for API protected routes")
	errorInvalidSbuscription = errors.New("Could not generate a valid subscription")
)

// RequestJSON represents a JSON HTTP request
type RequestJSON struct {
	Method  string
	URL     string
	Payload string
	Headers map[string]string
}

// Router is the struct that handles all routes
type Router struct {
	Subs    *subscriptions.Subscription
	Secrets *secrets.SecretService
	Actions *actions.Actions
	Spotify *spotify.Spotify
	Log     *telemetry.CustomLogger
	Discord *discord.Discord
}

// NewRouter creates a new router
func NewRouter(subs *subscriptions.Subscription, secretService *secrets.SecretService) *Router {
	actions := actions.NewActions(secretService)
	spotify := spotify.NewSpotify()
	discord := discord.NewDiscord()
	logger := telemetry.NewLogger("router")
	return &Router{
		Log:     logger,
		Subs:    subs,
		Secrets: secretService,
		Actions: actions,
		Spotify: spotify,
		Discord: discord,
	}
}

// CheckAuthAdmin validates for headers for admin routes
func (rt *Router) CheckAuthAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		telemetry.APICallCount.Inc()
		rt.Log.Info("Checking for admin token")
		token := os.Getenv(adminToken)
		if token == "" {
			rt.Log.Error("Admin Token missing", errorTokenNotFound)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.Header.Get("Token") == token {
			rt.Log.Info("Token is valid")
			next.ServeHTTP(w, r)
		} else {
			rt.Log.Error("Admin Token is invalid", errorTokenNotValid)
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
	var challengeResponse subscriptions.SubscribeEvent
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return
	}
	defer r.Body.Close()
	json.Unmarshal(body, &challengeResponse)
	w.Header().Add("Content-Type", "plain/text")
	w.Write([]byte(challengeResponse.Challenge))
	rt.Log.Info("Response sent to challenge")
}

// DeleteHandler deletes all subscriptions
func (rt *Router) DeleteHandler(w http.ResponseWriter, _ *http.Request) {
	subsList, err := rt.Subs.GetSubscriptions()
	if err != nil {
		rt.Log.Error("Could not get subscriptions", err)
	}
	err = rt.Subs.DeleteSubscriptions(subsList)
	if err != nil {
		rt.Log.Error("Could not delete subscriptions", err)
	}
	rt.Log.Info("Deleted all subscriptions")
	w.WriteHeader(http.StatusOK)
}

// HealthHandler returns a healthy message
func (rt *Router) HealthHandler(w http.ResponseWriter, _ *http.Request) {
	json.NewEncoder(w).Encode("Healthy")
}

// ListHandler returns the current subscription list
func (rt *Router) ListHandler(w http.ResponseWriter, _ *http.Request) {
	subsList, err := rt.Subs.GetSubscriptions()
	if err != nil {
		rt.Log.Error("Could not get subscriptions", err)
	}
	rt.Log.Info("Current Subscription List")
	for _, sub := range subsList.Data {
		rt.Log.Info("Status:" + sub.Status + " ,Type:" + sub.Type)
		subItem := fmt.Sprintf("ID:%s, Status: %s, Type: %s\n", sub.ID, sub.Status, sub.Type)
		w.Write([]byte(subItem))
	}
}

// CreateHandler creates a subscription based on the parameter
func (rt *Router) CreateHandler(_ http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	subType := query.Get("type")
	subscriptionTypes := map[string]subscriptions.SubscriptionType{
		"chat": {
			Name:    "chat",
			Version: "1",
			Type:    "channel.chat.message",
		},
		"follow": {
			Name:    "follow",
			Version: "2",
			Type:    "channel.follow",
		},
		"subscription": {
			Name:    "subscribe",
			Version: "1",
			Type:    "channel.subscribe",
		},
		"cheer": {
			Name:    "cheer",
			Version: "1",
			Type:    "channel.cheer",
		},
		"reward": {
			Name:    "reward",
			Version: "1",
			Type:    "channel.channel_points_custom_reward_redemption.add",
		},
		"stream": {
			Name:    "stream",
			Version: "1",
			Type:    "stream.online",
		},
	}
	if subTypeConfig, ok := subscriptionTypes[subType]; ok {
		payload := rt.GeneratePayload(subTypeConfig)
		rt.Subs.CreateSubscription(payload)
		rt.Log.Info("Subscription created: " + subType)
	} else {
		rt.Log.Error("Invalid subscription", errorInvalidSbuscription)
	}
}

// ChatHandler responds to chat messages
func (rt *Router) ChatHandler(_ http.ResponseWriter, r *http.Request) {
	var chatEvent subscriptions.ChatMessageEvent
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return
	}
	defer r.Body.Close()
	json.Unmarshal(body, &chatEvent)
	//	Send to parser to respond
	rt.Actions.ParseMessage(chatEvent)
}

// FollowHandler responds to follow events
func (rt *Router) FollowHandler(_ http.ResponseWriter, r *http.Request) {
	telemetry.FollowCount.Inc()
	var followEventResponse subscriptions.FollowEvent
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return
	}
	defer r.Body.Close()
	json.Unmarshal(body, &followEventResponse)
	// Send to chat
	rt.Actions.SendMessage(fmt.Sprintf("Gracias por el follow: %v", followEventResponse.Event.UserName))
}

// SubHandler responds to subscription events
func (rt *Router) SubHandler(_ http.ResponseWriter, r *http.Request) {
	var subEventResponse subscriptions.SubscriptionEvent
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return
	}
	defer r.Body.Close()
	json.Unmarshal(body, &subEventResponse)
	// send to chat
	telemetry.SubscriptionCount.Inc()
	rt.Actions.SendMessage(fmt.Sprintf("Gracias por el sub: %v", subEventResponse.Event.UserName))
}

// CheerHandler responds to cheer events
func (rt *Router) CheerHandler(_ http.ResponseWriter, r *http.Request) {
	var cheerEventResponse subscriptions.CheerEvent
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return
	}
	defer r.Body.Close()
	json.Unmarshal(body, &cheerEventResponse)
	// send to chat
	telemetry.CheerCount.Inc()
	rt.Actions.SendMessage(fmt.Sprintf("Gracias por los bits: %v", cheerEventResponse.Event.UserName))
}

// RewardHandler responds to reward events
func (rt *Router) RewardHandler(_ http.ResponseWriter, r *http.Request) {
	var rewardEventResponse subscriptions.RewardEvent
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return
	}
	defer r.Body.Close()
	json.Unmarshal(body, &rewardEventResponse)
	telemetry.RewardCount.Inc()
	// if rewardEventResponse.Event.Reward.Title == "Next Song" {
	// 	// token := rt.Spotify.GetSpotifyToken()
	// 	// rt.Spotify.NextSong(token)
	// }
	// if rewardEventResponse.Event.Reward.Title == "Add Song" {
	// 	spotifyURL := rewardEventResponse.Event.UserInput
	// 	// token := rt.Spotify.GetSpotifyToken()
	// 	// rt.Spotify.AddToPlaylist(token, spotifyURL)
	// }
	// if rewardEventResponse.Event.Reward.Title == "Reset Playlist" {
	// 	// token := rt.Spotify.GetSpotifyToken()
	// 	rt.Spotify.DeleteSongPlaylist(token)
	// }
}

// TestHandler is used to test if the bot is responding to messages
func (rt *Router) TestHandler(_ http.ResponseWriter, _ *http.Request) {
	rt.Log.Info("Testing")
	rt.Actions.SendMessage("Test")
}

// StreamHandler sends a message to discord
func (rt *Router) StreamHandler(_ http.ResponseWriter, _ *http.Request) {
	err := rt.Discord.NotifyChannel("En vivo y en directo @everyone - https://links.mvaldes.dev/stream")
	if err != nil {
		rt.Log.Error("Sending message to discord", err)
	}
	req, err := http.NewRequest("POST", "https://automate.mvaldes.dev/webhook/stream-live", nil)
	if err != nil {
		rt.Log.Error("Could not generate request for X post", err)
	}
	req.Header.Add("Token", os.Getenv(adminToken))
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		rt.Log.Error("Could not send request to webhook for X post", err)
	}
	if resp.StatusCode == 200 {
		rt.Log.Info("Posting message to X")
	}
}

// PlayingHandler displays music playing in spotify
func (rt *Router) PlayingHandler(_ http.ResponseWriter, _ *http.Request) {
	rt.Log.Info("Serving song")
	token, _ := rt.Spotify.GetSpotifyToken()
	if token != "" {
		rt.Log.Info("New token is" + token)
	}

}
