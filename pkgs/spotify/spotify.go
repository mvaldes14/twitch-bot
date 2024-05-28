// Package spotify interacts with spotify
package spotify

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/mvaldes14/twitch-bot/pkgs/types"
)

const (
	tokenURL   = "https://accounts.spotify.com/api/token"
	nextURL    = "https://api.spotify.com/v1/me/player/next"              // POST
	currentURL = "https://api.spotify.com/v1/me/player/currently-playing" // GET
)

var (
	currentToken string
)

// RefreshToken generates a new token for the spotify api
func RefreshToken() string {
	log.Println("Refreshing token")
	refreshToken := os.Getenv("SPOTIFY_REFRESH_TOKEN")
	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")
	token := base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientSecret))
	params := url.Values{}
	params.Set("grant_type", "refresh_token")
	params.Set("refresh_token", refreshToken)

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(params.Encode()))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+token)
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}
	var t types.SpotifyTokenResponse
	json.Unmarshal(resBody, &t)
	currentToken = t.AccessToken
	return t.AccessToken
}

// NextSong Changes the currently playing song
func NextSong(token string) {
	req, err := http.NewRequest("POST", nextURL, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	// Token is valid
	if res.StatusCode == http.StatusNoContent {
		log.Println("Song changed")
	}
	// Token is invalid
	if res.StatusCode == http.StatusUnauthorized {
		log.Println("Unauthorized")
		token := RefreshToken()
		NextSong(token)
	}
}

// GetSong returns the current song playing via chat
func GetSong(token string) types.SpotifyCurrentlyPlaying {
	req, err := http.NewRequest("GET", currentURL, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	// Token is valid
	if res.StatusCode == http.StatusOK {
		log.Println("Song found")
	}
	// Token is invalid
	if res.StatusCode == http.StatusUnauthorized {
		log.Println("Unauthorized")
		token := RefreshToken()
		GetSong(token)
	}
	body, err := io.ReadAll(res.Body)
	var currentlyPlaying types.SpotifyCurrentlyPlaying
	json.Unmarshal(body, &currentlyPlaying)
	return currentlyPlaying
}
