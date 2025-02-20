// Package server Generates the server and handlers to respond to requests
package server

import (
	"net/http"

	"github.com/mvaldes14/twitch-bot/pkgs/utils"
)

var logger = utils.Logger()

// NewServer creates the http server
func NewServer() {
	api := http.NewServeMux()
	api.HandleFunc("/create", createHandler)
	api.HandleFunc("/delete", deleteHandler)
	api.HandleFunc("/list", listHandler)
	api.HandleFunc("/test", testHandler)

	router := http.NewServeMux()
	router.HandleFunc("/follow", followHandler)
	router.HandleFunc("/chat", chatHandler)
	router.HandleFunc("/sub", subHandler)
	router.HandleFunc("/cheer", cheerHandler)
	router.HandleFunc("/reward", rewardHandler)
	router.HandleFunc("/stream", streamHandler)
	router.HandleFunc("/health", healthHandler)
	logger.Info("Running and listening")

	router.Handle("/api/", http.StripPrefix("/api", checkAuthAdmin(api)))

	srv := &http.Server{
		Addr:    ":3000",
		Handler: middleWareRoute(router),
	}
	err := srv.ListenAndServe()
	logger.Error("FATAL", "Error starting the server", err)
}
