// Package server Generates the server and handlers to respond to requests
package server

import (
	"log/slog"
	"net/http"

	"github.com/mvaldes14/twitch-bot/pkgs/routes"
	"github.com/mvaldes14/twitch-bot/pkgs/secrets"
	"github.com/mvaldes14/twitch-bot/pkgs/subscriptions"
	"github.com/mvaldes14/twitch-bot/pkgs/telemetry"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewServer creates the http server
func NewServer(logger *slog.Logger, port string) *http.Server {
	secretService := secrets.NewSecretService(logger)
	subs := subscriptions.NewSubscription(logger, secretService)
	rs := routes.NewRouter(logger, subs, secretService)
	api := http.NewServeMux()
	api.HandleFunc("/create", rs.CreateHandler)
	api.HandleFunc("/delete", rs.DeleteHandler)
	api.HandleFunc("/list", rs.ListHandler)
	api.HandleFunc("/test", rs.TestHandler)

	router := http.NewServeMux()
	router.HandleFunc("/follow", rs.FollowHandler)
	router.HandleFunc("/chat", rs.ChatHandler)
	router.HandleFunc("/sub", rs.SubHandler)
	router.HandleFunc("/cheer", rs.CheerHandler)
	router.HandleFunc("/reward", rs.RewardHandler)
	router.HandleFunc("/stream", rs.StreamHandler)
	router.HandleFunc("/health", rs.HealthHandler)
	router.Handle("/metrics", promhttp.Handler())
	router.HandleFunc("/testmetrics", func(w http.ResponseWriter, r *http.Request) {
		telemetry.TestMetric.Inc()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{\"status\": \"ok\"}"))
	})
	logger.Info("Running and listening")

	router.Handle("/api/", http.StripPrefix("/api", rs.CheckAuthAdmin(api)))

	srv := &http.Server{
		Addr:    port,
		Handler: rs.MiddleWareRoute(router),
	}
	return srv
}
