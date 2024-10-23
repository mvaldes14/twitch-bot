// Package server Generates the server and handlers to respond to requests
package server

import (
	"net/http"

	"github.com/mvaldes14/twitch-bot/pkgs/utils"
)

var logger = utils.Logger()

// NewServer creates the http server
func NewServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/create", createHandler)
	mux.HandleFunc("/delete", deleteHandler)
	mux.HandleFunc("/list", listHandler)
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/chat", chatHandler)
	mux.HandleFunc("/follow", followHandler)
	mux.HandleFunc("/sub", subHandler)
	mux.HandleFunc("/cheer", cheerHandler)
	mux.HandleFunc("/reward", rewardHandler)
	mux.HandleFunc("/test", testHandler)
	logger.Info("Running and listening")

	srv := &http.Server{
		Addr:    ":3000",
		Handler: mux,
	}
	err := srv.ListenAndServe()
	logger.Error("FATAL", err)
}
