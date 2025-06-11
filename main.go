// package main starts the server
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/mvaldes14/twitch-bot/pkgs/cache"
	"github.com/mvaldes14/twitch-bot/pkgs/secrets"
	"github.com/mvaldes14/twitch-bot/pkgs/spotify"

	"github.com/mvaldes14/twitch-bot/pkgs/server"
	"github.com/mvaldes14/twitch-bot/pkgs/telemetry"
)

const (
	twitchUserExpiration = 5259487
	twitchAppExpiration  = 14400
	SpotifyExpiration    = 3600
)

var (
	s  = secrets.NewSecretService()
	c  = cache.NewCacheService()
	sp = spotify.NewSpotify()
)

func validateDopplerToken() {
	doplerToken := os.Getenv("DOPPLER_TOKEN")
	if doplerToken == "" {
		panic("Doppler token is not set in the environment")
	}
}

func refreshTokens() {
	twitchUToken, err := c.GetToken("TWITCH_USER_TOKEN")
	if err == nil {
		os.Setenv("TWITCH_USER_TOKEN", twitchUToken)
	}

	twitchAToken, err := c.GetToken("TWITCH_APP_TOKEN")
	if err == nil {
		os.Setenv("TWITCH_APP_TOKEN", twitchAToken)
	}

	spotifyToken, err := c.GetToken("SPOTIFY_TOKEN")
	if err == nil {
		os.Setenv("SPOTIFY_TOKEN", spotifyToken)
	}
	switch {
	case twitchUToken == "":
		twitchUserToken, err := s.GenerateUserToken()
		if err != nil {
			fmt.Println(err)
		}
		c.StoreToken(cache.Token{
			Key:        "TWITCH_USER_TOKEN",
			Value:      twitchUserToken,
			Expiration: time.Duration(twitchUserExpiration) * time.Second,
		})
	case twitchAToken == "":
		twitchAppToken, err := s.RefreshAppToken()
		if err != nil {
			fmt.Println(err)
		}
		c.StoreToken(cache.Token{
			Key:        "TWITCH_APP_TOKEN",
			Value:      twitchAppToken,
			Expiration: time.Duration(twitchAppExpiration) * time.Second,
		})
	case spotifyToken == "":
		spotifyToken, err := sp.GetSpotifyToken()
		if err != nil {
			fmt.Println(err)
		}
		c.StoreToken(cache.Token{
			Key:        "SPOTIFY_TOKEN",
			Value:      spotifyToken,
			Expiration: time.Duration(SpotifyExpiration) * time.Second,
		})
	}
}

func main() {
	const port = ":3000"
	logger := telemetry.NewLogger("main")

	// Validate if doppler token is available otherwise stop the server
	validateDopplerToken()

	// Store tokens that expire in cache
	refreshTokens()

	logger.Info("Starting server on port" + port)
	srv := server.NewServer(port)
	if err := srv.ListenAndServe(); err != nil {
		logger.Error("Could not start server", err)
	}

}
