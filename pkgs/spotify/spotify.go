// Package spotify interacts with spotify
package spotify

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/mvaldes14/twitch-bot/pkgs/secrets"
	"github.com/mvaldes14/twitch-bot/pkgs/telemetry"
)

const (
	tokenURL            = "https://accounts.spotify.com/api/token"
	nextURL             = "https://api.spotify.com/v1/me/player/next"              // POST
	currentURL          = "https://api.spotify.com/v1/me/player/currently-playing" // GET
	playlistID          = "72Cwey4JPR3DV3cdUS72xG"
	playlistURL         = "https://api.spotify.com/v1/playlists/" // +id GET
	getPlaylistURL      = "https://api.spotify.com/v1/playlists/" // +id/tracks GET
	deletePlaylistURL   = "https://api.spotify.com/v1/playlists/" // +id/tracks DELETE
	spotifyRefreshToken = "SPOTIFY_REFRESH_TOKEN"
	spotifyClientID     = "SPOTIFY_CLIENT_ID"
	spotifyClientSecret = "SPOTIFY_CLIENT_SECRET"
)

var (
	token                    Token
	errSpotifyMissingSecrets = errors.New("Missing credentials from environment")
	errSpotifyNoToken        = errors.New("Failed to produce a new token")
)

// Spotify struct for spotify
type Spotify struct {
	Log     *telemetry.CustomLogger
	Secrets *secrets.SecretService
}

// Token struct for a token to use
type Token struct {
	Token     string
	Timestamp time.Time
}

// NewSpotify creates a new spotify instance
func NewSpotify() *Spotify {
	logger := telemetry.NewLogger("spotify")
	secrets := secrets.NewSecretService()
	return &Spotify{
		Log:     logger,
		Secrets: secrets,
	}
}

// checkTokenTimestamp checks if the token is older than 55 minutes
func checkTokenTimestamp(token Token) bool {
	if time.Since(token.Timestamp).Minutes() > 55 {
		return true
	}
	return false
}

// GetSpotifyToken generates a new token for the spotify api
func (s *Spotify) GetSpotifyToken() (Token, error) {
	// check if the token is valid before requesting a new one
	if !checkTokenTimestamp(token) {
		s.Log.Info("Token is still valid, reusing it")
		return token, nil
	}
	refreshToken := os.Getenv(spotifyRefreshToken)
	clientID := os.Getenv(spotifyClientID)
	clientSecret := os.Getenv(spotifyClientSecret)
	if refreshToken == "" || clientID == "" || clientSecret == "" {
		s.Log.Error("Missing Spotify credentials in Doppler", errSpotifyMissingSecrets)
		return token, errSpotifyMissingSecrets
	}
	encodedToken := base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientSecret))
	params := url.Values{}
	params.Set("grant_type", "refresh_token")
	params.Set("refresh_token", refreshToken)

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(params.Encode()))
	if err != nil {
		s.Log.Error("Error forming request for GetSpotifyToken", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+encodedToken)
	client := &http.Client{}
	s.Log.Info("Requesting New Spotify token")
	res, err := client.Do(req)
	if err != nil {
		s.Log.Error("Error sending request to get new token", err)
	}
	body, err := io.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		s.Log.Error("Error getting new token from Spotify API", err)
	}
	var t SpotifyTokenResponse
	err = json.Unmarshal(body, &t)
	if err != nil {
		s.Log.Error("Error unmarshalling token response", err)
		return token, err
	}
	token.Token = t.AccessToken
	token.Timestamp = time.Now()
	s.Log.Info("New token generated", token)
	err = s.Secrets.StoreNewTokens(token.Token)
	if err != nil {
		s.Log.Error("Error storing new token in Doppler", err)
		return token, err
	}
	return token, nil
}

// NextSong Changes the currently playing song
func (s *Spotify) NextSong(token string) {
	req, err := http.NewRequest("POST", nextURL, nil)
	if err != nil {
		s.Log.Error("Error Generating Request for next song", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		s.Log.Error("Error Sending Request for next song", err)
	}
	// Token is valid
	if res.StatusCode == http.StatusNoContent {
		s.Log.Info("Song changed")
	}
	// Token is invalid
	if res.StatusCode == http.StatusUnauthorized {
		s.Log.Info("Unauthorized")
		// token := s.GetSpotifyToken()
		// s.NextSong(token)
	}
}

// GetSong returns the current song playing via chat
func (s *Spotify) GetSong(token string) SpotifyCurrentlyPlaying {
	req, err := http.NewRequest("GET", currentURL, nil)
	if err != nil {
		s.Log.Error("Error Generating Request for get song", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		s.Log.Error("Error Sending Request for get song", err)
	}
	// Token is valid
	if res.StatusCode == http.StatusOK {
		s.Log.Info("Song found")
	}
	// Token is invalid
	if res.StatusCode == http.StatusUnauthorized {
		s.Log.Info("Unauthorized")
		// token := s.GetSpotifyToken()
		// s.GetSong(token)
	}
	body, err := io.ReadAll(res.Body)
	var currentlyPlaying SpotifyCurrentlyPlaying
	json.Unmarshal(body, &currentlyPlaying)
	return currentlyPlaying
}

func (s *Spotify) parseSong(url string) string {
	splitURL := strings.Split(url, "/")
	trackID := splitURL[len(splitURL)-1]
	splitTrackID := strings.Split(trackID, "?")
	trackID = splitTrackID[0]
	return trackID
}

// AddToPlaylist includes a song to the playlist
func (s *Spotify) AddToPlaylist(token string, song string) {
	if s.validateURL(song) {
		addPlaylistURL := fmt.Sprintf("https://api.spotify.com/v1/playlists/%v/tracks", playlistID)
		songID := s.parseSong(song)
		body := fmt.Sprintf("{\"uris\":[\"spotify:track:%v\"]}", songID)
		req, err := http.NewRequest("POST", addPlaylistURL, bytes.NewBuffer([]byte(body)))
		if err != nil {
			s.Log.Error("Cannot construct request with parameters given", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		res, err := client.Do(req)
		if err != nil {
			s.Log.Error("Error sending request to add to playlist", err)
		}
		s.Log.Info("Adding song to playlist" + res.Status)
	}
}

func (s *Spotify) validateURL(url string) bool {
	if strings.Contains(url, "https://open.spotify.com/") {
		return true
	}
	return false
}

func (s *Spotify) getPlaylist(token string) int {
	req, err := http.NewRequest("GET", playlistURL+playlistID, nil)
	if err != nil {
		s.Log.Error("Cannot construct request with parameters given", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		s.Log.Error("Error sending request to get playlist", err)
	}
	body, err := io.ReadAll(res.Body)
	var playlist SpotifyPlaylistResponse
	json.Unmarshal(body, &playlist)
	return playlist.Tracks.Total
}

func (s *Spotify) getSongsPlaylist(playlistID, token string) []string {
	req, err := http.NewRequest("GET", getPlaylistURL+playlistID+"/tracks?", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		s.Log.Error("Error Generating Request for get song playlist", err)
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		s.Log.Error("Error Sending Request for get song playlist", err)
	}
	var playstResponse SpotifyPlaylistItemList
	body, err := io.ReadAll(res.Body)
	if err != nil {
		s.Log.Error("Error parsing body from get song playlist", err)
	}
	json.Unmarshal(body, &playstResponse)

	var songIDs []string
	for _, item := range playstResponse.Items {
		songIDs = append(songIDs, item.Track.ID)
	}
	return songIDs
}

// DeleteSongPlaylist wipes the playlist to start fresh
func (s *Spotify) DeleteSongPlaylist(token string) {
	songs := s.getSongsPlaylist(playlistID, token)
	formatSongs := s.generateURISongs(songs)
	body := fmt.Sprintf("{\"tracks\":[%v]}", strings.Join(formatSongs, ","))
	req, err := http.NewRequest("DELETE", deletePlaylistURL+playlistID+"tracks", bytes.NewBuffer([]byte(body)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		s.Log.Error("Error Generating Request for delete playlist", err)
	}
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		s.Log.Error("Error Sending Request for delete playlist", err)
	}
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		s.Log.Info(string(body))
	}
}

func (s *Spotify) generateURISongs(songs []string) []string {
	var songStructs []string
	for _, song := range songs {
		songStructs = append(songStructs, fmt.Sprintf("{\"uri\":\"spotify:track:%v\"}", song))
	}
	return songStructs
}
