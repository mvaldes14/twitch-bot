// Package secrets handles all interactions with secrets
package secrets

import (
	"bytes"
	"context"
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

	"github.com/mvaldes14/twitch-bot/pkgs/cache"
	"github.com/mvaldes14/twitch-bot/pkgs/telemetry"
)

const (
	twitchRefreshToken   = "TWITCH_REFRESH_TOKEN"
	twitchAppToken       = "TWITCH_APP_TOKEN"
	twitchUserToken      = "TWITCH_USER_TOKEN"
	twitchClientID       = "TWITCH_CLIENT_ID"
	twitchSecret         = "TWITCH_CLIENT_SECRET"
	spotifyRefreshToken  = "SPOTIFY_REFRESH_TOKEN"
	spotifyClientID      = "SPOTIFY_CLIENT_ID"
	spotifyClientSecret  = "SPOTIFY_CLIENT_SECRET"
	requestTimeout       = 30 * time.Second
	twitchUserExpiration = 5259487
	twitchAppExpiration  = 14400
	spotifyExpiration    = 3600

	// API Endpoints
	twitchTokenURL = "https://id.twitch.tv/oauth2/token"
	twitchValidURL = "https://id.twitch.tv/oauth2/validate"
	tokenURL       = "https://accounts.spotify.com/api/token"
)

var (
	errMissingTokenOrID      = errors.New("token or client ID not found in environment")
	errSpotifyMissingSecrets = errors.New("missing credentials from environment")
	errSpotifyNoToken        = errors.New("failed to produce a new token")
	errInvalidRequest        = errors.New("failed to create HTTP request")
	errHTTPRequest           = errors.New("HTTP request failed")
	errResponseParsing       = errors.New("failed to parse response")
)

// SecretService implements SecretManager interface
type SecretService struct {
	Log        *telemetry.CustomLogger
	Cache      *cache.Service
	httpClient *http.Client
}

// NewSecretService creates a new instance of SecretService
func NewSecretService() *SecretService {
	logger := telemetry.NewLogger("secrets")
	cache := cache.NewCacheService()
	httpClient := &http.Client{Timeout: requestTimeout}
	return &SecretService{Log: logger, Cache: cache, httpClient: httpClient}
}

// InitSecrets initializes the secrets by checking the cache and generating new tokens if necessary
func (s *SecretService) InitSecrets() {
	twitchUToken, err := s.Cache.GetToken("TWITCH_USER_TOKEN")
	if err == nil {
		os.Setenv("TWITCH_USER_TOKEN", twitchUToken)
	} else {
		newTwitchUserToken, err := s.GenerateUserToken()
		if err != nil {
			s.Log.Error("ERROR:", err)
		}
		if err := s.Cache.StoreToken(cache.Token{
			Key:        "TWITCH_USER_TOKEN",
			Value:      newTwitchUserToken,
			Expiration: time.Duration(twitchUserExpiration) * time.Second,
		}); err != nil {
			s.Log.Error("Failed to store twitch user token", err)
		}
	}

	twitchAToken, err := s.Cache.GetToken("TWITCH_APP_TOKEN")
	if err == nil {
		os.Setenv("TWITCH_APP_TOKEN", twitchAToken)
	} else {
		newTwitchAppToken, err := s.RefreshAppToken()
		if err != nil {
			s.Log.Error("ERROR:", err)
		}
		if err := s.Cache.StoreToken(cache.Token{
			Key:        "TWITCH_APP_TOKEN",
			Value:      newTwitchAppToken,
			Expiration: time.Duration(twitchAppExpiration) * time.Second,
		}); err != nil {
			s.Log.Error("Failed to store twitch app token", err)
		}
	}

	spotifyTk, err := s.Cache.GetToken("SPOTIFY_TOKEN")
	if err == nil {
		os.Setenv("SPOTIFY_TOKEN", spotifyTk)
	} else {
		newSpotifyToken, err := s.GetSpotifyToken()
		if err != nil {
			s.Log.Error("ERROR:", err)
		}
		if err := s.Cache.StoreToken(cache.Token{
			Key:        "SPOTIFY_TOKEN",
			Value:      newSpotifyToken,
			Expiration: time.Duration(spotifyExpiration) * time.Second,
		}); err != nil {
			s.Log.Error("Failed to store spotify token", err)
		}
	}
}

// BuildSecretHeaders Returns the secrets from env variables to build headers for requests
func (s *SecretService) BuildSecretHeaders() (RequestHeader, error) {
	token := os.Getenv(twitchAppToken)
	clientID := os.Getenv(twitchClientID)
	if token == "" || clientID == "" {
		s.Log.Error("Missing Twitch token or Client ID in environment", errMissingTokenOrID)
		return RequestHeader{}, errMissingTokenOrID
	}
	return RequestHeader{
		Token:    token,
		ClientID: clientID,
	}, nil
}

// GenerateUserToken acquires a new token that is valid for 2 months
func (s *SecretService) GenerateUserToken() (string, error) {
	s.Log.Info("Generating new twitch user token")
	twitchID := os.Getenv(twitchClientID)
	twitchSecretVal := os.Getenv(twitchSecret)
	if twitchID == "" || twitchSecretVal == "" {
		return "", errMissingTokenOrID
	}
	payload := fmt.Sprintf("client_id=%v&client_secret=%v&grant_type=client_credentials", twitchID, twitchSecretVal)
	headers := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}
	req := RequestJSON{
		Method:  "POST",
		URL:     twitchTokenURL,
		Payload: payload,
		Headers: headers,
	}
	var response TwitchUserTokenResponse
	if err := s.MakeRequestMarshallJSON(req, &response); err != nil {
		s.Log.Error("Failed to make request generating user token", err)
	}
	return response.AccessToken, nil
}

// RefreshAppToken uses the refresh token to get a new one
func (s *SecretService) RefreshAppToken() (string, error) {
	twitchID := os.Getenv(twitchClientID)
	twitchSecretVal := os.Getenv(twitchSecret)
	twitchRefreshTk := os.Getenv(twitchRefreshToken)
	payload := fmt.Sprintf("grant_type=refresh_token&refresh_token=%v&client_id=%v&client_secret=%v", twitchRefreshTk, twitchID, twitchSecretVal)
	req := RequestJSON{
		Method:  "POST",
		URL:     twitchTokenURL,
		Payload: payload,
		Headers: map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
	}
	var response TwitchRefreshResponse
	if err := s.MakeRequestMarshallJSON(req, &response); err != nil {
		s.Log.Error("Failed to make request refreshing token", err)
	}
	return response.AccessToken, nil
}

// ValidateToken checks if the token is still valid
func (s *SecretService) ValidateToken(token string) bool {
	var response TwitchValidResponse
	req := RequestJSON{
		Method:  "GET",
		URL:     twitchValidURL,
		Headers: map[string]string{"Authorization": "Bearer " + token},
		Payload: "",
	}
	if err := s.MakeRequestMarshallJSON(req, &response); err != nil {
		s.Log.Error("Failed to make request refreshing token", err)
	}
	if response.ExpiresIn > 0 {
		s.Log.Info("Token is valid, expires in: ", response.ExpiresIn)
		return false
	}
	return true
}

// MakeRequestMarshallJSON makes a request and marshals the response into the target interface
func (s *SecretService) MakeRequestMarshallJSON(req RequestJSON, target any) error {
	ctx := context.Background()
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.URL, bytes.NewBuffer([]byte(req.Payload)))
	if err != nil {
		return err
	}
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}
	s.Log.Info("Sending request to:", req.URL)
	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, target)
}

// GetSpotifyToken retrieves a new Spotify token using the refresh token
func (s *SecretService) GetSpotifyToken() (string, error) {
	refreshToken := os.Getenv(spotifyRefreshToken)
	clientID := os.Getenv(spotifyClientID)
	clientSecret := os.Getenv(spotifyClientSecret)

	if refreshToken == "" || clientID == "" || clientSecret == "" {
		s.Log.Error("Missing Spotify credentials in environment", errSpotifyMissingSecrets)
		return "", errSpotifyMissingSecrets
	}

	encodedToken := base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientSecret))
	params := url.Values{}
	params.Set("grant_type", "refresh_token")
	params.Set("refresh_token", refreshToken)

	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(params.Encode()))
	if err != nil {
		s.Log.Error("Error forming request for GetSpotifyToken", err)
		return "", errInvalidRequest
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+encodedToken)

	s.Log.Info("Requesting New Spotify token")
	res, err := s.httpClient.Do(req)
	if err != nil {
		s.Log.Error("Error sending request to get new token", err)
		return "", errHTTPRequest
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		s.Log.Error("Token request failed with status", fmt.Errorf("status: %d", res.StatusCode))
		return "", errSpotifyNoToken
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		s.Log.Error("Error reading token response body", err)
		return "", errResponseParsing
	}

	var t SpotifyTokenResponse
	if err = json.Unmarshal(body, &t); err != nil {
		s.Log.Error("Error unmarshalling token response", err)
		return "", errResponseParsing
	}

	if t.AccessToken == "" {
		s.Log.Error("Received empty access token", errSpotifyNoToken)
		return "", errSpotifyNoToken
	}

	return t.AccessToken, nil
}
