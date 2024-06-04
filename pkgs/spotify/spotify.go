// Package spotify interacts with spotify
package spotify

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/mvaldes14/twitch-bot/pkgs/types"
)

const (
	tokenURL          = "https://accounts.spotify.com/api/token"
	nextURL           = "https://api.spotify.com/v1/me/player/next"              // POST
	currentURL        = "https://api.spotify.com/v1/me/player/currently-playing" // GET
	playlistID        = "72Cwey4JPR3DV3cdUS72xG"
	playlistURL       = "https://api.spotify.com/v1/playlists/" // +id GET
	getPlaylistURL    = "https://api.spotify.com/v1/playlists/" // +id/tracks GET
	deletePlaylistURL = "https://api.spotify.com/v1/playlists/" // +id/tracks DELETE
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

func parseSong(url string) string {
	splitURL := strings.Split(url, "/")
	trackID := splitURL[len(splitURL)-1]
	splitTrackID := strings.Split(trackID, "?")
	trackID = splitTrackID[0]
	return trackID
}

// AddToPlaylist includes a song to the playlist
func AddToPlaylist(token string, song string) {
	if validateURL(song) {
		addPlaylistURL := fmt.Sprintf("https://api.spotify.com/v1/playlists/%v/tracks", playlistID)
		songID := parseSong(song)
		// position := getPlaylist(token)
		// body := fmt.Sprintf("{\"uris\":[\"spotify:track:%v\"], \"position\":\"%v\"}", songID, position-1)
		body := fmt.Sprintf("{\"uris\":[\"spotify:track:%v\"]}", songID)
		req, err := http.NewRequest("POST", addPlaylistURL, bytes.NewBuffer([]byte(body)))
		if err != nil {
			log.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		res, err := client.Do(req)
		if err != nil {
			log.Fatal(err)
		}
		log.Println(res.StatusCode)
	}
}

func validateURL(url string) bool {
	if strings.Contains(url, "https://open.spotify.com/") {
		return true
	}
	return false
}

func getPlaylist(token string) int {
	req, err := http.NewRequest("GET", playlistURL+playlistID, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	body, err := io.ReadAll(res.Body)
	var playlist types.SpotifyPlaylistResponse
	json.Unmarshal(body, &playlist)
	return playlist.Tracks.Total
}

func getSongsPlaylist(playlistID, token string) []string {
	req, err := http.NewRequest("GET", getPlaylistURL+playlistID+"/tracks?", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		log.Fatal(err)
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	var playstResponse types.SpotifyPlaylistItemList
	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}
	json.Unmarshal(body, &playstResponse)

	var songIDs []string
	for _, item := range playstResponse.Items {
		songIDs = append(songIDs, item.Track.ID)
	}
	return songIDs
}

// DeleteSongPlaylist wipes the playlist to start fresh
func DeleteSongPlaylist(token string) {
	songs := getSongsPlaylist(playlistID, token)
	formatSongs := generateURISongs(songs)
	body := fmt.Sprintf("{\"tracks\":[%v]}", strings.Join(formatSongs, ","))
	req, err := http.NewRequest("DELETE", deletePlaylistURL+playlistID+"tracks", bytes.NewBuffer([]byte(body)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		log.Fatal(err)
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			log.Fatal(err)
		}
		log.Println(string(body))
	}
}

func generateURISongs(songs []string) []string {
	var songStructs []string
	for _, song := range songs {
		songStructs = append(songStructs, fmt.Sprintf("{\"uri\":\"spotify:track:%v\"}", song))
	}
	return songStructs
}
