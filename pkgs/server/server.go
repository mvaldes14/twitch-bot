// Package server Generates the server and handlers to respond to requests
package server

import (
	"net/http"

	"github.com/mvaldes14/twitch-bot/pkgs/routes"
	"github.com/mvaldes14/twitch-bot/pkgs/secrets"
	"github.com/mvaldes14/twitch-bot/pkgs/subscriptions"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewServer creates the http server
func NewServer(port string) *http.Server {
	secretService := secrets.NewSecretService()
	subs := subscriptions.NewSubscription(secretService)
	rs := routes.NewRouter(subs, secretService)
	api := http.NewServeMux()
	api.HandleFunc("POST /create", rs.CreateHandler)
	api.HandleFunc("POST /delete", rs.DeleteHandler)
	api.HandleFunc("GET /list", rs.ListHandler)
	api.HandleFunc("GET /test", rs.TestHandler)

	router := http.NewServeMux()
	router.HandleFunc("GET /follow", rs.FollowHandler)
	router.HandleFunc("GET /chat", rs.ChatHandler)
	router.HandleFunc("GET /sub", rs.SubHandler)
	router.HandleFunc("GET /cheer", rs.CheerHandler)
	router.HandleFunc("GET /reward", rs.RewardHandler)
	router.HandleFunc("GET /stream-online", rs.StreamOnlineHandler)
	router.HandleFunc("GET /health", rs.HealthHandler)
	router.HandleFunc("GET /playing", rs.PlayingHandler)
	router.HandleFunc("GET /test", rs.TestHandler)
	router.Handle("GET /metrics", promhttp.Handler())

	router.Handle("GET /api/", http.StripPrefix("/api", rs.CheckAuthAdmin(api)))

	srv := &http.Server{
		Addr:    port,
		Handler: rs.MiddleWareRoute(router),
	}
	return srv
}
