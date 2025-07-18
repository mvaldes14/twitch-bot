// Package routes defines all routes for handlers and functionality
package routes

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"text/template"
	"time"

	"github.com/mvaldes14/twitch-bot/pkgs/actions"
	"github.com/mvaldes14/twitch-bot/pkgs/cache"
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
	errorNoMusicPlaying      = errors.New("Nothing is playing on spotify")
)

// RequestJSON represents a JSON HTTP request
type RequestJSON struct {
	Method  string
	URL     string
	Payload string
	Headers map[string]string
}

// SongData represents the data for the song
type SongData struct {
	Title    string
	Artist   string
	AlbumArt string
}

// Router is the struct that handles all routes
type Router struct {
	Subs            *subscriptions.Subscription
	Secrets         *secrets.SecretService
	Actions         *actions.Actions
	Spotify         *spotify.Spotify
	Log             *telemetry.CustomLogger
	Discord         *discord.Discord
	streamStartTime time.Time
	Cache           *cache.Service
}

// SubscriptionTypeRequest is the struct for generating new subscriptions
type SubscriptionTypeRequest struct {
	Type      string `json:"type"`
	createdAt time.Time
}

// NewRouter creates a new router
func NewRouter(subs *subscriptions.Subscription, secretService *secrets.SecretService) *Router {
	actions := actions.NewActions(secretService)
	spotify := spotify.NewSpotify()
	discord := discord.NewDiscord()
	logger := telemetry.NewLogger("router")
	cache := cache.NewCacheService()
	return &Router{
		Log:     logger,
		Subs:    subs,
		Secrets: secretService,
		Actions: actions,
		Spotify: spotify,
		Discord: discord,
		Cache:   cache,
	}
}

// CheckAuthAdmin validates for headers for admin routes
func (rt *Router) CheckAuthAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		telemetry.APICallCount.Inc()
		token := os.Getenv(adminToken)
		if token == "" {
			rt.Log.Error("Admin Token missing", errorTokenNotFound)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.Header.Get("Authorization") == token {
			next.ServeHTTP(w, r)
		} else {
			rt.Log.Error("Admin token is invalid", errorTokenNotValid)
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
		w.WriteHeader(http.StatusInternalServerError)
	}
	if subsList.Total == 0 {
		rt.Log.Info("No subscriptions found")
		return
	}
	rt.Log.Info("Current Subscription List: ", subsList.Total)
	for _, sub := range subsList.Data {
		rt.Log.Info("Status:" + sub.Status + " ,Type:" + sub.Type)
		subItem := fmt.Sprintf("ID:%s, Status: %s, Type: %s\n", sub.ID, sub.Status, sub.Type)
		w.Write([]byte(subItem))
	}
}

// CreateHandler creates a subscription based on the parameter
func (rt *Router) CreateHandler(w http.ResponseWriter, r *http.Request) {
	requestType, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Could not parse payload", http.StatusInternalServerError)
	}
	defer r.Body.Close()
	var requestTypeString SubscriptionTypeRequest
	err = json.Unmarshal(requestType, &requestTypeString)

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
		"streamon": {
			Name:    "stream",
			Version: "1",
			Type:    "stream.online",
		},
		"streamoff": {
			Name:    "stream",
			Version: "1",
			Type:    "stream.offline",
		},
	}
	if subTypeConfig, ok := subscriptionTypes[string(requestTypeString.Type)]; ok {
		payload := rt.GeneratePayload(subTypeConfig)
		rt.Subs.CreateSubscription(payload)
		rt.Log.Info("Subscription created: " + requestTypeString.Type)
	} else {
		rt.Log.Error("Invalid subscription", errorInvalidSbuscription)
	}
}

// ChatHandler responds to chat messages
func (rt *Router) ChatHandler(_ http.ResponseWriter, r *http.Request) {
	telemetry.ChatMessageCount.Inc()
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
	rt.Actions.SendMessage(fmt.Sprintf("Gracias por el sub: %v", subEventResponse.Event.UserName))
}

// CheerHandler responds to cheer events
func (rt *Router) CheerHandler(_ http.ResponseWriter, r *http.Request) {
	telemetry.CheerCount.Inc()
	var cheerEventResponse subscriptions.CheerEvent
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return
	}
	defer r.Body.Close()
	json.Unmarshal(body, &cheerEventResponse)
	// send to chat
	rt.Actions.SendMessage(fmt.Sprintf("Gracias por los bits: %v", cheerEventResponse.Event.UserName))
}

// RewardHandler responds to reward events
func (rt *Router) RewardHandler(_ http.ResponseWriter, r *http.Request) {
	telemetry.RewardCount.Inc()
	var rewardEventResponse subscriptions.RewardEvent
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return
	}
	defer r.Body.Close()
	json.Unmarshal(body, &rewardEventResponse)
	if rewardEventResponse.Event.Reward.Title == "Next Song" {
		if err := rt.Spotify.NextSong(); err != nil {
			rt.Log.Error("Failed to skip to next song", err)
		}
	}
	if rewardEventResponse.Event.Reward.Title == "Add Song" {
		rt.Log.Info("Adding song to playlist")
		spotifyURL := rewardEventResponse.Event.UserInput
		if err := rt.Spotify.AddToPlaylist(spotifyURL); err != nil {
			rt.Log.Error("Failed to add song to playlist", err)
		}
	}
	if rewardEventResponse.Event.Reward.Title == "Reset Playlist" {
		if err := rt.Spotify.DeleteSongPlaylist(); err != nil {
			rt.Log.Error("Failed to reset playlist", err)
		}
	}
}

// TestHandler is used to test if the bot is responding to messages
func (rt *Router) TestHandler(_ http.ResponseWriter, _ *http.Request) {
	rt.Log.Info("Testing")
	rt.Actions.SendMessage("Test")
	rt.Spotify.NextSong()
}

// StreamOnlineHandler sends a message to discord
func (rt *Router) StreamOnlineHandler(_ http.ResponseWriter, _ *http.Request) {
	rt.streamStartTime = time.Now()
	telemetry.StreamDuration.Observe(0)
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

// StreamOfflineHandler tracks when streams end
func (rt *Router) StreamOfflineHandler(_ http.ResponseWriter, _ *http.Request) {
	if !rt.streamStartTime.IsZero() {
		duration := time.Since(rt.streamStartTime).Seconds()
		telemetry.StreamDuration.Observe(duration)
		rt.Log.Info("Stream ended", "duration", duration)
		rt.streamStartTime = time.Time{} // Reset
	}
}

// PlayingHandler displays music playing in spotify
func (rt *Router) PlayingHandler(w http.ResponseWriter, _ *http.Request) {
	song, err := rt.Spotify.GetSong()
	if err != nil {
		rt.Log.Error("Failed to get current song", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !song.IsPlaying {
		rt.Log.Error("No Music", errorNoMusicPlaying)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if song.Item.Name == "" || len(song.Item.Artists) == 0 || len(song.Item.Album.Images) == 0 {
		rt.Log.Error("Incomplete song data", errorNoMusicPlaying)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	data := SongData{
		Title:    song.Item.Name,
		Artist:   song.Item.Artists[0].Name,
		AlbumArt: song.Item.Album.Images[0].URL,
	}
	tmpl, err := template.ParseFiles("./templates/index.html")
	if err != nil {
		rt.Log.Error("Error parsing template", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Error executing template", http.StatusInternalServerError)
		return
	}
}

// PlaylistHandler displays the playlist
func (rt *Router) PlaylistHandler(w http.ResponseWriter, _ *http.Request) {
	songs, err := rt.Spotify.GetSongsPlaylist()
	if err != nil {
		rt.Log.Error("Failed to get playlist songs", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	rt.Log.Info(songs)
}
