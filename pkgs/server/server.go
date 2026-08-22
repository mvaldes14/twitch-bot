// Package server Generates the server and handlers to respond to requests
package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/mvaldes14/twitch-bot/pkgs/routes"
	"github.com/mvaldes14/twitch-bot/pkgs/secrets"
	"github.com/mvaldes14/twitch-bot/pkgs/subscriptions"
)

// NewServer creates the http server.
// The caller owns the SecretService so the handlers share the same instance
// whose tokens the background renewal goroutine refreshes.
func NewServer(port string, secretService *secrets.SecretService) (*http.Server, error) {
	subs, err := subscriptions.NewSubscription(secretService)
	if err != nil {
		return nil, fmt.Errorf("failed to build subscription service: %w", err)
	}
	rs, err := routes.NewRouter(subs, secretService)
	if err != nil {
		return nil, fmt.Errorf("failed to build router: %w", err)
	}
	api := http.NewServeMux()
	api.HandleFunc("POST /create", rs.CreateHandler)
	api.HandleFunc("POST /delete", rs.DeleteHandler)
	api.HandleFunc("GET /list", rs.ListHandler)
	api.HandleFunc("GET /test", rs.TestHandler)

	router := http.NewServeMux()
	router.HandleFunc("/follow", rs.FollowHandler)
	router.HandleFunc("/chat", rs.ChatHandler)
	router.HandleFunc("/sub", rs.SubHandler)
	router.HandleFunc("/cheer", rs.CheerHandler)
	router.HandleFunc("/reward", rs.RewardHandler)
	router.HandleFunc("/stream-online", rs.StreamOnlineHandler)
	router.HandleFunc("/stream-offline", rs.StreamOfflineHandler)
	router.HandleFunc("/health", rs.HealthHandler)
	router.HandleFunc("/playing", rs.PlayingHandler)
	router.HandleFunc("/playlist", rs.PlaylistHandler)
	router.HandleFunc("/test", rs.TestHandler)

	router.Handle("/api/", http.StripPrefix("/api", rs.CheckAuthAdmin(api)))

	srv := &http.Server{
		Addr:              port,
		Handler:           rs.TracingMiddleware(rs.MiddleWareRoute(router)),
		ReadHeaderTimeout: 10 * time.Second,
	}
	return srv, nil
}
