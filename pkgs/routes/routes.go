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
	"github.com/mvaldes14/twitch-bot/pkgs/notifications"
	"github.com/mvaldes14/twitch-bot/pkgs/secrets"
	"github.com/mvaldes14/twitch-bot/pkgs/spotify"
	"github.com/mvaldes14/twitch-bot/pkgs/subscriptions"
	"github.com/mvaldes14/twitch-bot/pkgs/telemetry"
	"go.opentelemetry.io/otel/attribute"
)

const (
	adminToken = "ADMIN_TOKEN"
)

var (
	errorInvalidSbuscription = errors.New("could not generate a valid subscription")
	errorNoMusicPlaying      = errors.New("nothing is playing on spotify")
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
	Notification    *notifications.NotificationService
	streamStartTime time.Time
	Cache           *cache.Service
}

// SubscriptionTypeRequest is the struct for generating new subscriptions
type SubscriptionTypeRequest struct {
	Type string `json:"type"`
}

// NewRouter creates a new router
func NewRouter(subs *subscriptions.Subscription, secretService *secrets.SecretService) *Router {
	actionsService := actions.NewActions(secretService)
	spotifyClient := spotify.NewSpotify()
	notify := notifications.NewNotificationService()
	logger := telemetry.NewLogger("router")
	cacheService := cache.NewCacheService()
	return &Router{
		Log:          logger,
		Subs:         subs,
		Secrets:      secretService,
		Actions:      actionsService,
		Spotify:      spotifyClient,
		Notification: notify,
		Cache:        cacheService,
	}
}

// CheckAuthAdmin validates authorization headers for admin routes
// Validates ADMIN_TOKEN environment variable exists before attempting auth
func (rt *Router) CheckAuthAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := telemetry.StartSpan(r.Context(), "auth_admin_check")
		defer span.End()

		telemetry.IncrementAPICallCount(ctx)

		// Validate ADMIN_TOKEN environment variable exists before attempting auth
		token := os.Getenv(adminToken)
		if token == "" {
			errMsg := fmt.Errorf("ADMIN_TOKEN not found in environment - required for admin endpoints. Pass ADMIN_TOKEN as an environment variable at startup")
			rt.Log.Error("Cannot authenticate request - ADMIN_TOKEN missing from environment configuration", errMsg)
			telemetry.RecordError(span, errMsg)
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "admin authentication not configured",
			})
			return
		}

		// Check if provided token matches environment token
		if r.Header.Get("Authorization") == token {
			next.ServeHTTP(w, r.WithContext(ctx))
		} else {
			errMsg := fmt.Errorf("invalid ADMIN_TOKEN provided in Authorization header")
			rt.Log.Error("Admin authentication failed - provided token does not match", errMsg)
			telemetry.RecordError(span, errMsg)
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

// TracingMiddleware wraps handlers with OpenTelemetry tracing
func (rt *Router) TracingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := telemetry.StartHTTPSpan(r.Context(), r.URL.Path, r)
		defer span.End()

		// Create a response writer wrapper to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Serve the request with the traced context
		next.ServeHTTP(rw, r.WithContext(ctx))

		// Set span status based on HTTP status code
		telemetry.SetSpanStatus(span, rw.statusCode)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// HANDLERS
// respondToChallenge responds to challenge for a subscription on twitch eventsub
func (rt *Router) respondToChallenge(w http.ResponseWriter, r *http.Request) {
	_, span := telemetry.StartSpan(r.Context(), "eventsub_challenge_verification")
	defer span.End()

	rt.Log.Info("Responding to challenge")
	var challengeResponse subscriptions.SubscribeEvent
	body, err := io.ReadAll(r.Body)
	if err != nil {
		rt.Log.Error("Failed to read challenge request body", err)
		telemetry.RecordError(span, err)
		return
	}
	defer r.Body.Close()

	if err := json.Unmarshal(body, &challengeResponse); err != nil {
		rt.Log.Error("Failed to unmarshal challenge response", err)
		telemetry.RecordError(span, err)
		telemetry.AddSpanAttributes(span,
			attribute.String("challenge.status", "unmarshal_failed"),
		)
		return
	}

	telemetry.AddSpanAttributes(span,
		attribute.String("challenge.subscription_id", challengeResponse.Subscription.ID),
		attribute.String("challenge.subscription_type", challengeResponse.Subscription.Type),
		attribute.String("challenge.subscription_version", challengeResponse.Subscription.Version),
		attribute.String("challenge.subscription_status", challengeResponse.Subscription.Status),
	)

	w.Header().Add("Content-Type", "plain/text")
	_, _ = w.Write([]byte(challengeResponse.Challenge))

	rt.Log.Info(fmt.Sprintf("Challenge response sent - Subscription ID: %s, Type: %s", challengeResponse.Subscription.ID, challengeResponse.Subscription.Type))
	telemetry.AddSpanAttributes(span,
		attribute.String("challenge.status", "success"),
	)
}

// DeleteHandler deletes all subscriptions
func (rt *Router) DeleteHandler(w http.ResponseWriter, _ *http.Request) {
	subsList, err := rt.Subs.GetSubscriptions()
	if err != nil {
		rt.Log.Error("Could not get subscriptions", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = rt.Subs.DeleteSubscriptions(subsList)
	if err != nil {
		rt.Log.Error("Could not delete subscriptions", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// HealthHandler returns a healthy message
func (rt *Router) HealthHandler(w http.ResponseWriter, _ *http.Request) {
	_ = json.NewEncoder(w).Encode("Healthy")
}

// ListHandler returns the current subscription list
func (rt *Router) ListHandler(w http.ResponseWriter, _ *http.Request) {
	subsList, err := rt.Subs.GetSubscriptions()
	if err != nil {
		rt.Log.Error("Could not get subscriptions", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if subsList.Total == 0 {
		rt.Log.Info("No subscriptions found")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"total": 0,
			"data":  []interface{}{},
		})
		return
	}

	rt.Log.Info(fmt.Sprintf("Current Subscription List: %d", subsList.Total))
	for _, sub := range subsList.Data {
		rt.Log.Info(fmt.Sprintf("Status: %s, Type: %s, ID: %s", sub.Status, sub.Type, sub.ID))
	}

	_ = json.NewEncoder(w).Encode(subsList)
}

// CreateHandler creates a subscription based on the parameter
func (rt *Router) CreateHandler(w http.ResponseWriter, r *http.Request) {
	_, span := telemetry.StartSpan(r.Context(), "create_subscription")
	defer span.End()

	requestType, err := io.ReadAll(r.Body)
	if err != nil {
		rt.Log.Error("Could not parse payload", err)
		telemetry.RecordError(span, err)
		http.Error(w, "Could not parse payload", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	var requestTypeString SubscriptionTypeRequest
	if err = json.Unmarshal(requestType, &requestTypeString); err != nil {
		rt.Log.Error("Could not unmarshal payload", err)
		telemetry.RecordError(span, err)
		http.Error(w, "Could not unmarshal payload", http.StatusBadRequest)
		return
	}

	telemetry.AddSpanAttributes(span,
		attribute.String("subscription.type_requested", requestTypeString.Type),
	)

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

	if subTypeConfig, ok := subscriptionTypes[requestTypeString.Type]; ok {
		telemetry.AddSpanAttributes(span,
			attribute.String("subscription.config_name", subTypeConfig.Name),
			attribute.String("subscription.config_version", subTypeConfig.Version),
			attribute.String("subscription.config_type", subTypeConfig.Type),
		)

		payload := rt.GeneratePayload(subTypeConfig)
		rt.Log.Info("Creating subscription for type: " + requestTypeString.Type)

		if err := rt.Subs.CreateSubscription(payload); err != nil {
			rt.Log.Error("Failed to create subscription", err)
			telemetry.RecordError(span, err)
			telemetry.AddSpanAttributes(span,
				attribute.String("subscription.status", "failed"),
			)
			http.Error(w, "Failed to create subscription", http.StatusInternalServerError)
			return
		}

		rt.Log.Info("Subscription created successfully: " + requestTypeString.Type)
		telemetry.AddSpanAttributes(span,
			attribute.String("subscription.status", "success"),
		)
		w.WriteHeader(http.StatusOK)
	} else {
		rt.Log.Error("Invalid subscription type requested: "+requestTypeString.Type, errorInvalidSbuscription)
		telemetry.RecordError(span, errorInvalidSbuscription)
		telemetry.AddSpanAttributes(span,
			attribute.String("subscription.status", "invalid_type"),
		)
		http.Error(w, "Invalid subscription type", http.StatusBadRequest)
	}
}

// ChatHandler responds to chat messages
func (rt *Router) ChatHandler(_ http.ResponseWriter, r *http.Request) {
	ctx, span := telemetry.StartSpan(r.Context(), "handle_chat_message")
	defer span.End()

	rt.Log.Info("Received chat message event")

	telemetry.IncrementChatMessageCount(ctx)
	var chatEvent subscriptions.ChatMessageEvent
	body, err := io.ReadAll(r.Body)
	if err != nil {
		rt.Log.Error("Failed to read chat event request body", err)
		telemetry.RecordError(span, err)
		return
	}
	defer r.Body.Close()

	if err := json.Unmarshal(body, &chatEvent); err != nil {
		rt.Log.Error("Failed to unmarshal chat event payload", err)
		telemetry.RecordError(span, err)
		return
	}

	rt.Log.Info(fmt.Sprintf("Processing chat message from user: %s, message: %s", chatEvent.Event.ChatterUserName, chatEvent.Event.Message.Text))

	// Add chat-specific attributes
	telemetry.AddSpanAttributes(span,
		attribute.String("chat.user", chatEvent.Event.ChatterUserName),
		attribute.String("chat.message", chatEvent.Event.Message.Text),
	)

	//	Send to parser to respond
	rt.Actions.ParseMessage(chatEvent)
	rt.Log.Info(fmt.Sprintf("Successfully processed chat message from: %s", chatEvent.Event.ChatterUserName))
}

// FollowHandler responds to follow events
func (rt *Router) FollowHandler(_ http.ResponseWriter, r *http.Request) {
	ctx, span := telemetry.StartSpan(r.Context(), "handle_follow")
	defer span.End()

	rt.Log.Info("Received follow event")

	telemetry.IncrementFollowCount(ctx)
	var followEventResponse subscriptions.FollowEvent
	body, err := io.ReadAll(r.Body)
	if err != nil {
		rt.Log.Error("Failed to read follow event request body", err)
		telemetry.RecordError(span, err)
		return
	}
	defer r.Body.Close()

	if err := json.Unmarshal(body, &followEventResponse); err != nil {
		rt.Log.Error("Failed to unmarshal follow event payload", err)
		telemetry.RecordError(span, err)
		return
	}

	rt.Log.Info(fmt.Sprintf("New follower: %s", followEventResponse.Event.UserName))

	telemetry.AddSpanAttributes(span,
		attribute.String("follow.user", followEventResponse.Event.UserName),
	)

	// Send to chat
	if err := rt.Actions.SendMessage(fmt.Sprintf("Gracias por el follow: %v", followEventResponse.Event.UserName)); err != nil {
		rt.Log.Error("Failed to send follow thank you message to chat", err)
		telemetry.RecordError(span, err)
		return
	}
	rt.Log.Info(fmt.Sprintf("Successfully processed follow from: %s", followEventResponse.Event.UserName))
}

// SubHandler responds to subscription events
func (rt *Router) SubHandler(_ http.ResponseWriter, r *http.Request) {
	ctx, span := telemetry.StartSpan(r.Context(), "handle_subscription")
	defer span.End()

	rt.Log.Info("Received subscription event")

	telemetry.IncrementSubscriptionCount(ctx)
	var subEventResponse subscriptions.SubscriptionEvent
	body, err := io.ReadAll(r.Body)
	if err != nil {
		rt.Log.Error("Failed to read subscription event request body", err)
		telemetry.RecordError(span, err)
		return
	}
	defer r.Body.Close()

	if err := json.Unmarshal(body, &subEventResponse); err != nil {
		rt.Log.Error("Failed to unmarshal subscription event payload", err)
		telemetry.RecordError(span, err)
		return
	}

	rt.Log.Info(fmt.Sprintf("New subscriber: %s", subEventResponse.Event.UserName))

	telemetry.AddSpanAttributes(span,
		attribute.String("subscription.user", subEventResponse.Event.UserName),
	)

	// send to chat
	if err := rt.Actions.SendMessage(fmt.Sprintf("Gracias por el sub: %v", subEventResponse.Event.UserName)); err != nil {
		rt.Log.Error("Failed to send subscription thank you message to chat", err)
		telemetry.RecordError(span, err)
		return
	}
	rt.Log.Info(fmt.Sprintf("Successfully processed subscription from: %s", subEventResponse.Event.UserName))
}

// CheerHandler responds to cheer events
func (rt *Router) CheerHandler(_ http.ResponseWriter, r *http.Request) {
	ctx, span := telemetry.StartSpan(r.Context(), "handle_cheer")
	defer span.End()

	rt.Log.Info("Received cheer event")

	telemetry.IncrementCheerCount(ctx)
	var cheerEventResponse subscriptions.CheerEvent
	body, err := io.ReadAll(r.Body)
	if err != nil {
		rt.Log.Error("Failed to read cheer event request body", err)
		telemetry.RecordError(span, err)
		return
	}
	defer r.Body.Close()

	if err := json.Unmarshal(body, &cheerEventResponse); err != nil {
		rt.Log.Error("Failed to unmarshal cheer event payload", err)
		telemetry.RecordError(span, err)
		return
	}

	rt.Log.Info(fmt.Sprintf("Cheer received from: %s, bits: %d", cheerEventResponse.Event.UserName, cheerEventResponse.Event.Bits))

	telemetry.AddSpanAttributes(span,
		attribute.String("cheer.user", cheerEventResponse.Event.UserName),
		attribute.Int("cheer.bits", cheerEventResponse.Event.Bits),
	)

	// send to chat
	if err := rt.Actions.SendMessage(fmt.Sprintf("Gracias por los bits: %v", cheerEventResponse.Event.UserName)); err != nil {
		rt.Log.Error("Failed to send cheer thank you message to chat", err)
		telemetry.RecordError(span, err)
		return
	}
	rt.Log.Info(fmt.Sprintf("Successfully processed cheer from: %s", cheerEventResponse.Event.UserName))
}

// RewardHandler responds to reward events
func (rt *Router) RewardHandler(_ http.ResponseWriter, r *http.Request) {
	ctx, span := telemetry.StartSpan(r.Context(), "handle_reward")
	defer span.End()

	rt.Log.Info("Received reward redemption event")

	telemetry.IncrementRewardCount(ctx)
	var rewardEventResponse subscriptions.RewardEvent
	body, err := io.ReadAll(r.Body)
	if err != nil {
		rt.Log.Error("Failed to read reward event request body", err)
		telemetry.RecordError(span, err)
		return
	}
	defer r.Body.Close()

	if err := json.Unmarshal(body, &rewardEventResponse); err != nil {
		rt.Log.Error("Failed to unmarshal reward event payload", err)
		telemetry.RecordError(span, err)
		return
	}

	rt.Log.Info(fmt.Sprintf("Reward redeemed by: %s, reward: %s", rewardEventResponse.Event.UserName, rewardEventResponse.Event.Reward.Title))

	telemetry.AddSpanAttributes(span,
		attribute.String("reward.title", rewardEventResponse.Event.Reward.Title),
		attribute.String("reward.user", rewardEventResponse.Event.UserName),
	)

	if rewardEventResponse.Event.Reward.Title == "Next Song" {
		rt.Log.Info("Processing Next Song reward")
		if err := rt.Spotify.NextSong(); err != nil {
			rt.Log.Error("Failed to skip to next song", err)
			telemetry.RecordError(span, err)
			return
		}
		rt.Log.Info("Successfully skipped to next song")
	}
	if rewardEventResponse.Event.Reward.Title == "Add Song" {
		rt.Log.Info("Processing Add Song reward")
		spotifyURL := rewardEventResponse.Event.UserInput
		telemetry.AddSpanAttributes(span, attribute.String("spotify.url", spotifyURL))
		if err := rt.Spotify.AddToPlaylist(spotifyURL); err != nil {
			rt.Log.Error("Failed to add song to playlist", err)
			telemetry.RecordError(span, err)
			return
		}
		rt.Log.Info(fmt.Sprintf("Successfully added song to playlist: %s", spotifyURL))
	}
	if rewardEventResponse.Event.Reward.Title == "Reset Playlist" {
		rt.Log.Info("Processing Reset Playlist reward")
		if err := rt.Spotify.DeleteSongPlaylist(); err != nil {
			rt.Log.Error("Failed to reset playlist", err)
			telemetry.RecordError(span, err)
			return
		}
		rt.Log.Info("Successfully reset playlist")
	}

	rt.Log.Info(fmt.Sprintf("Successfully processed reward from: %s", rewardEventResponse.Event.UserName))
}

// TestHandler is used to test if the bot is responding to messages
func (rt *Router) TestHandler(_ http.ResponseWriter, _ *http.Request) {
	rt.Log.Info("Testing")
	// rt.Actions.SendMessage("Test")
	_ = rt.Notification.SendNotification("Test Message from Twitch Bot")
	// rt.Spotify.NextSong()
}

// StreamOnlineHandler sends a message to discord
// Validates ADMIN_TOKEN environment variable exists before making requests
func (rt *Router) StreamOnlineHandler(_ http.ResponseWriter, r *http.Request) {
	ctx, span := telemetry.StartSpan(r.Context(), "handle_stream_online")
	defer span.End()

	rt.Log.Info("Received stream online event")

	rt.streamStartTime = time.Now()
	telemetry.AddSpanAttributes(span,
		attribute.String("stream.event", "online"),
		attribute.String("stream.start_time", rt.streamStartTime.Format(time.RFC3339)),
	)

	rt.Log.Info(fmt.Sprintf("Stream started at: %s", rt.streamStartTime.Format(time.RFC3339)))

	err := rt.Notification.SendNotification("En vivo y en directo @everyone - https://links.mvaldes.dev/stream")
	if err != nil {
		rt.Log.Error("Failed to send stream online notification to discord", err)
		telemetry.RecordError(span, err)
	} else {
		rt.Log.Info("Successfully sent stream online notification to discord")
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://automate.mvaldes.dev/webhook/stream-live", http.NoBody)
	if err != nil {
		rt.Log.Error("Could not generate request for X post", err)
		telemetry.RecordError(span, err)
		return
	}

	// Validate ADMIN_TOKEN exists before attempting to use it
	adminTokenValue := os.Getenv(adminToken)
	if adminTokenValue == "" {
		errMsg := fmt.Errorf("ADMIN_TOKEN not found in environment - required for webhook notifications. Pass ADMIN_TOKEN as an environment variable at startup")
		rt.Log.Error("Cannot send stream notification - ADMIN_TOKEN missing from environment", errMsg)
		telemetry.RecordError(span, errMsg)
		return
	}

	req.Header.Add("Token", adminTokenValue)
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		rt.Log.Error("Could not send request to webhook for X post", err)
		telemetry.RecordError(span, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		rt.Log.Info("Successfully executed notification workflows")
	} else {
		rt.Log.Info(fmt.Sprintf("Webhook returned non-OK status: %d", resp.StatusCode))
	}

	rt.Log.Info("Successfully processed stream online event")
}

// StreamOfflineHandler tracks when streams end
func (rt *Router) StreamOfflineHandler(_ http.ResponseWriter, r *http.Request) {
	ctx, span := telemetry.StartSpan(r.Context(), "handle_stream_offline")
	defer span.End()

	rt.Log.Info("Received stream offline event")

	if !rt.streamStartTime.IsZero() {
		duration := time.Since(rt.streamStartTime).Seconds()
		telemetry.RecordStreamDuration(ctx, duration)
		telemetry.AddSpanAttributes(span,
			attribute.String("stream.event", "offline"),
			attribute.Float64("stream.duration_seconds", duration),
		)
		rt.Log.Info(fmt.Sprintf("Stream ended, duration: %.2f seconds", duration))
		rt.streamStartTime = time.Time{} // Reset
	} else {
		rt.Log.Info("Stream offline event received but no start time was recorded")
	}

	rt.Log.Info("Successfully processed stream offline event")
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
