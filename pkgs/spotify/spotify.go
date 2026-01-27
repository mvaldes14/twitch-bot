// Package spotify interacts with spotify
package spotify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/mvaldes14/twitch-bot/pkgs/cache"
	"github.com/mvaldes14/twitch-bot/pkgs/telemetry"
)

const (
	tokenURL          = "https://accounts.spotify.com/api/token"
	nextURL           = "https://api.spotify.com/v1/me/player/next"              // POST
	currentURL        = "https://api.spotify.com/v1/me/player/currently-playing" // GET
	playlistURL       = "https://api.spotify.com/v1/playlists/"                  // +id GET
	getPlaylistURL    = "https://api.spotify.com/v1/playlists/"                  // +id/tracks GET
	deletePlaylistURL = "https://api.spotify.com/v1/playlists/"                  // +id/tracks DELETE
	defaultPlaylistID = "72Cwey4JPR3DV3cdUS72xG"
	requestTimeout    = 30 * time.Second
)

var (
	errSpotifyNoToken  = errors.New("Failed to produce a new token")
	errInvalidRequest  = errors.New("Failed to create HTTP request")
	errHTTPRequest     = errors.New("HTTP request failed")
	errResponseParsing = errors.New("Failed to parse response")
	errInvalidURL      = errors.New("Invalid URL to add to Spotify Playlist")
)

// Spotify struct for spotify
type Spotify struct {
	Log        *telemetry.CustomLogger
	Cache      *cache.Service
	PlaylistID string
	httpClient *http.Client
}

// NewSpotify creates a new spotify instance
func NewSpotify() *Spotify {
	logger := telemetry.NewLogger("spotify")
	cache := cache.NewCacheService()
	playlistID := os.Getenv("SPOTIFY_PLAYLIST_ID")
	if playlistID == "" {
		playlistID = defaultPlaylistID
	}
	return &Spotify{
		Log:        logger,
		Cache:      cache,
		PlaylistID: playlistID,
		httpClient: &http.Client{Timeout: requestTimeout},
	}
}

// getValidToken returns a valid token, refreshing if necessary
func (s *Spotify) getValidToken() (string, error) {
	if cachedToken, err := s.Cache.GetToken("SPOTIFY_TOKEN"); err == nil && cachedToken != "" {
		s.Log.Info("Using cached token")
		return cachedToken, nil
	}
	return "", errSpotifyNoToken

}

// NextSong Changes the currently playing song
func (s *Spotify) NextSong() error {
	ctx := context.Background()
	ctx, span := telemetry.StartExternalSpan(ctx, "spotify.next_song", "spotify", "next_song")
	defer span.End()

	token, err := s.getValidToken()
	if err != nil {
		telemetry.RecordError(span, err)
		return fmt.Errorf("failed to get valid token: %w", err)
	}

	telemetry.IncrementSpotifySongChanged(ctx)
	s.Log.Info("Changing song")

	req, err := http.NewRequestWithContext(ctx, "POST", nextURL, nil)
	if err != nil {
		s.Log.Error("Error Generating Request for next song", err)
		telemetry.RecordError(span, err)
		return errInvalidRequest
	}

	req.Header.Set("Authorization", "Bearer "+token)
	res, err := s.httpClient.Do(req)
	if err != nil {
		s.Log.Error("Error Sending Request for next song", err)
		telemetry.RecordError(span, err)
		return errHTTPRequest
	}
	defer res.Body.Close()

	telemetry.SetSpanStatus(span, res.StatusCode)

	switch res.StatusCode {
	case http.StatusNoContent:
		s.Log.Info("Song changed")
		return nil
	case http.StatusUnauthorized:
		s.Log.Info("Token unauthorized, clearing cache")
		_ = s.Cache.DeleteToken("SPOTIFY_TOKEN")
		return fmt.Errorf("unauthorized: token may be expired")
	default:
		s.Log.Error("Unexpected response status", fmt.Errorf("status: %d", res.StatusCode))
		return fmt.Errorf("unexpected status: %d", res.StatusCode)
	}
}

// GetSong returns the current song playing via chat
func (s *Spotify) GetSong() (SpotifyCurrentlyPlaying, error) {
	var currentlyPlaying SpotifyCurrentlyPlaying
	ctx := context.Background()

	token, err := s.getValidToken()
	if err != nil {
		return currentlyPlaying, fmt.Errorf("failed to get valid token: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", currentURL, nil)
	if err != nil {
		s.Log.Error("Error Generating Request for get song", err)
		return currentlyPlaying, errInvalidRequest
	}

	req.Header.Set("Authorization", "Bearer "+token)
	res, err := s.httpClient.Do(req)
	if err != nil {
		s.Log.Error("Error Sending Request for get song", err)
		return currentlyPlaying, errHTTPRequest
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusUnauthorized {
		_ = s.Cache.DeleteToken("SPOTIFY_TOKEN")
		return currentlyPlaying, fmt.Errorf("unauthorized: token may be expired")
	}

	if res.StatusCode != http.StatusOK {
		return currentlyPlaying, fmt.Errorf("unexpected status: %d", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		s.Log.Error("Error reading response body", err)
		return currentlyPlaying, errResponseParsing
	}

	if err = json.Unmarshal(body, &currentlyPlaying); err != nil {
		s.Log.Error("Error unmarshalling song response", err)
		return currentlyPlaying, errResponseParsing
	}

	return currentlyPlaying, nil
}

func (s *Spotify) parseSong(url string) (string, error) {
	if url == "" {
		return "", fmt.Errorf("empty URL provided")
	}

	splitURL := strings.Split(url, "/")
	if len(splitURL) < 2 {
		return "", fmt.Errorf("invalid URL format: %s", url)
	}

	trackID := splitURL[len(splitURL)-1]
	if trackID == "" {
		return "", fmt.Errorf("no track ID found in URL: %s", url)
	}

	// Remove query parameters if present
	if idx := strings.Index(trackID, "?"); idx != -1 {
		trackID = trackID[:idx]
	}

	if trackID == "" {
		return "", fmt.Errorf("empty track ID after parsing: %s", url)
	}

	return trackID, nil
}

// AddToPlaylist includes a song to the playlist
func (s *Spotify) AddToPlaylist(song string) error {
	if song == "" {
		return fmt.Errorf("empty song URL provided")
	}

	if !s.validateURL(song) {
		s.Log.Error("Invalid URL", errInvalidURL)
		return errInvalidURL
	}

	token, err := s.getValidToken()
	if err != nil {
		return fmt.Errorf("failed to get valid token: %w", err)
	}

	s.Log.Info("Valid URL", song)
	addPlaylistURL := fmt.Sprintf("https://api.spotify.com/v1/playlists/%v/tracks", s.PlaylistID)

	songID, err := s.parseSong(song)
	if err != nil {
		s.Log.Error("Error parsing song URL", err)
		return fmt.Errorf("failed to parse song URL: %w", err)
	}

	ctx := context.Background()
	reqBody := fmt.Sprintf("{\"uris\":[\"spotify:track:%v\"]}", songID)
	req, err := http.NewRequestWithContext(ctx, "POST", addPlaylistURL, bytes.NewBuffer([]byte(reqBody)))
	if err != nil {
		s.Log.Error("Cannot construct request with parameters given", err)
		return errInvalidRequest
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	res, err := s.httpClient.Do(req)
	if err != nil {
		s.Log.Error("Error sending request to add to playlist", err)
		return errHTTPRequest
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusCreated, http.StatusOK:
		s.Log.Info("Successfully added song to playlist")
		return nil
	case http.StatusUnauthorized:
		_ = s.Cache.DeleteToken("SPOTIFY_TOKEN")
		return fmt.Errorf("unauthorized: token may be expired")
	default:
		respBody, _ := io.ReadAll(res.Body)
		s.Log.Error("Unexpected response status", fmt.Errorf("status: %d, body: %s", res.StatusCode, string(respBody)))
		return fmt.Errorf("unexpected status: %d", res.StatusCode)
	}
}

func (s *Spotify) validateURL(url string) bool {
	return strings.Contains(url, "https://open.spotify.com/track/")
}

// GetSongsPlaylistIDs returns a list of track IDs from the playlist
func (s *Spotify) GetSongsPlaylistIDs() ([]string, error) {
	ctx := context.Background()
	token, err := s.getValidToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get valid token: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", getPlaylistURL+s.PlaylistID+"/tracks", nil)
	if err != nil {
		s.Log.Error("Error Generating Request for get song playlist", err)
		return nil, errInvalidRequest
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	res, err := s.httpClient.Do(req)
	if err != nil {
		s.Log.Error("Error Sending Request for get song playlist", err)
		return nil, errHTTPRequest
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusUnauthorized {
		_ = s.Cache.DeleteToken("SPOTIFY_TOKEN")
		return nil, fmt.Errorf("unauthorized: token may be expired")
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		s.Log.Error("Error parsing body from get song playlist", err)
		return nil, errResponseParsing
	}

	var playlistResponse SpotifyPlaylistItemList
	if err = json.Unmarshal(body, &playlistResponse); err != nil {
		s.Log.Error("Error unmarshalling playlist response", err)
		return nil, errResponseParsing
	}

	var songIDs []string
	for _, item := range playlistResponse.Items {
		if item.Track.ID != "" {
			songIDs = append(songIDs, item.Track.ID)
		}
	}
	return songIDs, nil
}

// GetSongsPlaylist returns a list of formatted song names from the playlist
func (s *Spotify) GetSongsPlaylist() ([]string, error) {
	ctx := context.Background()
	token, err := s.getValidToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get valid token: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", getPlaylistURL+s.PlaylistID+"/tracks", nil)
	if err != nil {
		s.Log.Error("Error Generating Request for get song playlist", err)
		return nil, errInvalidRequest
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	res, err := s.httpClient.Do(req)
	if err != nil {
		s.Log.Error("Error Sending Request for get song playlist", err)
		return nil, errHTTPRequest
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusUnauthorized {
		_ = s.Cache.DeleteToken("SPOTIFY_TOKEN")
		return nil, fmt.Errorf("unauthorized: token may be expired")
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		s.Log.Error("Error parsing body from get song playlist", err)
		return nil, errResponseParsing
	}

	var playlistResponse SpotifyPlaylistItemList
	if err = json.Unmarshal(body, &playlistResponse); err != nil {
		s.Log.Error("Error unmarshalling playlist response", err)
		return nil, errResponseParsing
	}

	var songList []string
	for _, item := range playlistResponse.Items {
		if item.Track.Name != "" && len(item.Track.Artists) > 0 {
			songList = append(songList, fmt.Sprintf("%v - %v", item.Track.Name, item.Track.Artists[0].Name))
		}
	}
	return songList, nil
}

// DeleteSongPlaylist wipes the playlist to start fresh
func (s *Spotify) DeleteSongPlaylist() error {
	token, err := s.getValidToken()
	if err != nil {
		return fmt.Errorf("failed to get valid token: %w", err)
	}

	songs, err := s.GetSongsPlaylistIDs()
	if err != nil {
		return fmt.Errorf("failed to get playlist songs: %w", err)
	}

	if len(songs) == 0 {
		s.Log.Info("Playlist is already empty")
		return nil
	}

	ctx := context.Background()
	formatSongs := s.generateURISongs(songs)
	reqBody := fmt.Sprintf("{\"tracks\":[%v]}", strings.Join(formatSongs, ","))

	req, err := http.NewRequestWithContext(ctx, "DELETE", deletePlaylistURL+s.PlaylistID+"/tracks", bytes.NewBuffer([]byte(reqBody)))
	if err != nil {
		s.Log.Error("Error Generating Request for delete playlist", err)
		return errInvalidRequest
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	res, err := s.httpClient.Do(req)
	if err != nil {
		s.Log.Error("Error Sending Request for delete playlist", err)
		return errHTTPRequest
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusOK:
		s.Log.Info("Successfully cleared playlist")
		return nil
	case http.StatusUnauthorized:
		_ = s.Cache.DeleteToken("SPOTIFY_TOKEN")
		return fmt.Errorf("unauthorized: token may be expired")
	default:
		respBody, _ := io.ReadAll(res.Body)
		s.Log.Error("Unexpected response status", fmt.Errorf("status: %d, body: %s", res.StatusCode, string(respBody)))
		return fmt.Errorf("unexpected status: %d", res.StatusCode)
	}
}

func (s *Spotify) generateURISongs(songs []string) []string {
	var songStructs []string
	for _, song := range songs {
		songStructs = append(songStructs, fmt.Sprintf("{\"uri\":\"spotify:track:%v\"}", song))
	}
	return songStructs
}
