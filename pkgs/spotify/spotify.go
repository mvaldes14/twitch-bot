// Package spotify interacts with spotify
package spotify

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/mvaldes14/twitch-bot/pkgs/cache"
	"github.com/mvaldes14/twitch-bot/pkgs/service"
)

const (
	nextURL    = "https://api.spotify.com/v1/me/player/next"              // POST
	currentURL = "https://api.spotify.com/v1/me/player/currently-playing" // GET
)

// TODO: Think of all the possible errors we can throw based on the service
var (
	errSpotifyNoToken  = errors.New("Failed to produce a new token")
	errInvalidRequest  = errors.New("Failed to create HTTP request")
	errHTTPRequest     = errors.New("HTTP request failed")
	errResponseParsing = errors.New("Failed to parse response")
)

// Spotify struct for spotify
type Spotify struct {
	Service *service.Service
	Cache   *cache.CacheService
}

// NewSpotify creates a new spotify instance
func NewSpotify() *Spotify {
	cache := cache.NewCacheService()
	service := service.NewService("spotify")
	return &Spotify{
		Cache:   cache,
		Service: service,
	}
}

// getValidToken returns a valid token, refreshing if necessary
func (s *Spotify) getValidToken() (string, error) {
	if cachedToken, err := s.Cache.GetToken("SPOTIFY_TOKEN"); err == nil && cachedToken.Value != "" {
		s.Service.Logger.Info("Using cached token")
		return cachedToken.Value, nil
	}
	return "", errSpotifyNoToken

}

// NextSong Changes the currently playing song
func (s *Spotify) NextSong() error {
	token, err := s.getValidToken()
	if err != nil {
		return fmt.Errorf("failed to get valid token: %w", err)
	}

	s.Service.Metrics.IncrementCount("spotify_song_changed_count", "Number of times the Spotify song changed")
	s.Service.Logger.Info("Changing song")

	req, err := http.NewRequest("POST", nextURL, nil)
	if err != nil {
		s.Service.Logger.Error(err)
		return errInvalidRequest
	}

	req.Header.Set("Authorization", "Bearer "+token)
	res, err := s.Service.Client.Do(req)
	if err != nil {
		s.Service.Logger.Error(err)
		return errHTTPRequest
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusNoContent:
		s.Service.Logger.Info("Song changed")
		return nil
	case http.StatusUnauthorized:
		s.Cache.DeleteToken("SPOTIFY_TOKEN")
		return fmt.Errorf("Unauthorized: token may be expired")
	default:
		return fmt.Errorf("unexpected status: %d", res.StatusCode)
	}
}

// GetCurrentSong returns the current song playing via chat
func (s *Spotify) GetCurrentSong() (SpotifyCurrentlyPlaying, error) {
	var currentlyPlaying SpotifyCurrentlyPlaying

	token, err := s.getValidToken()
	if err != nil {
		return currentlyPlaying, fmt.Errorf("failed to get valid token: %w", err)
	}

	req, err := http.NewRequest("GET", currentURL, nil)
	if err != nil {
		s.Service.Logger.Error(err)
		return currentlyPlaying, errInvalidRequest
	}

	req.Header.Set("Authorization", "Bearer "+token)
	res, err := s.Service.Client.Do(req)
	if err != nil {
		s.Service.Logger.Error(err)
		return currentlyPlaying, errHTTPRequest
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusUnauthorized {
		s.Cache.DeleteToken("SPOTIFY_TOKEN")
		return currentlyPlaying, fmt.Errorf("unauthorized: token may be expired")
	}

	if res.StatusCode != http.StatusOK {
		return currentlyPlaying, fmt.Errorf("unexpected status: %d", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		s.Service.Logger.Error(err)
		return currentlyPlaying, errResponseParsing
	}

	if err = json.Unmarshal(body, &currentlyPlaying); err != nil {
		s.Service.Logger.Error(err)
		return currentlyPlaying, errResponseParsing
	}

	return currentlyPlaying, nil
}
