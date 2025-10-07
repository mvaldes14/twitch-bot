// Package secrets handles all interactions with secrets
package secrets

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

	"github.com/mvaldes14/twitch-bot/pkgs/cache"
	"github.com/mvaldes14/twitch-bot/pkgs/service"
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

// TODO: Think of all the possible errors we can throw based on the service
var (
	errMissingTokenOrID      = errors.New("Token or Client ID not found in environment")
	errFailedToInit          = errors.New("Failed to initialize secrets, check environment variables")
	errSpotifyMissingSecrets = errors.New("Missing credentials from environment")
	errSpotifyNoToken        = errors.New("Failed to produce a new token")
	errInvalidURL            = errors.New("Invalid URL to add to Spotify Playlist")
	errInvalidRequest        = errors.New("Failed to create HTTP request")
	errHTTPRequest           = errors.New("HTTP request failed")
	errResponseParsing       = errors.New("Failed to parse response")
)

// Secret interface defines the methods to export and apply
type Secret interface {
	InitSecrets() error
	GetTwitchAppToken() error
	GetTwitchUserToken() error
	GetSpotifyToken() error
}

// SecretService implements SecretManager interface
type SecretService struct {
	Service *service.Service
	Cache   *cache.CacheService
}

// NewSecretService creates a new instance of SecretService
func NewSecretService() *SecretService {
	cache := cache.NewCacheService()
	service := service.NewService("notifications")
	return &SecretService{Service: service, Cache: cache}
}

// InitSecrets initializes the secrets by checking the cache and generating new tokens if necessary
func (s *SecretService) InitSecrets() {
	twitchUToken, err := s.Cache.GetToken("TWITCH_USER_TOKEN")
	if err == nil {
		os.Setenv("TWITCH_USER_TOKEN", twitchUToken)
	} else {
		twitchUserToken, err := s.GenerateUserToken()
		if err != nil {
			s.Service.Logger.Error(err)
		}
		s.Cache.StoreToken(cache.Token{
			Key:        "TWITCH_USER_TOKEN",
			Value:      twitchUserToken,
			Expiration: time.Duration(twitchUserExpiration) * time.Second,
		})
	}

	twitchAToken, err := s.Cache.GetToken("TWITCH_APP_TOKEN")
	if err == nil {
		os.Setenv("TWITCH_APP_TOKEN", twitchAToken)
	} else {
		twitchAppToken, err := s.RefreshAppToken()
		if err != nil {
			s.Service.Logger.Error(err)
		}
		s.Cache.StoreToken(cache.Token{
			Key:        "TWITCH_APP_TOKEN",
			Value:      twitchAppToken,
			Expiration: time.Duration(twitchAppExpiration) * time.Second,
		})
	}

	spotifyToken, err := s.Cache.GetToken("SPOTIFY_TOKEN")
	if err == nil {
		os.Setenv("SPOTIFY_TOKEN", spotifyToken)
	} else {
		spotifyToken, err := s.GetSpotifyToken()
		if err != nil {
			s.Service.Logger.Error(err)
		}
		s.Cache.StoreToken(cache.Token{
			Key:        "SPOTIFY_TOKEN",
			Value:      spotifyToken,
			Expiration: time.Duration(spotifyExpiration) * time.Second,
		})
	}
}

// BuildSecretHeaders Returns the secrets from env variables to build headers for requests
func (s *SecretService) BuildSecretHeaders() (RequestHeader, error) {
	token := os.Getenv(twitchAppToken)
	clientID := os.Getenv(twitchClientID)
	if token == "" || clientID == "" {
		s.Service.Logger.Error(errMissingTokenOrID)
		return RequestHeader{}, errMissingTokenOrID
	}
	return RequestHeader{
		Token:    token,
		ClientID: clientID,
	}, nil
}

// GenerateUserToken acquires a new token that is valid for 2 months
func (s *SecretService) GenerateUserToken() (string, error) {
	s.Service.Logger.Info("Generating new twitch user token")
	twitchID := os.Getenv(twitchClientID)
	twitchSecret := os.Getenv(twitchSecret)
	if twitchID == "" || twitchSecret == "" {
		return "", errMissingTokenOrID
	}
	payload := fmt.Sprintf("client_id=%v&client_secret=%v&grant_type=client_credentials", twitchID, twitchSecret)
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
		s.Service.Logger.Error(err)
	}
	return response.AccessToken, nil
}

// RefreshAppToken uses the refresh token to get a new one
func (s *SecretService) RefreshAppToken() (string, error) {
	twitchID := os.Getenv(twitchClientID)
	twitchSecret := os.Getenv(twitchSecret)
	twitchRefreshToken := os.Getenv(twitchRefreshToken)
	payload := fmt.Sprintf("grant_type=refresh_token&refresh_token=%v&client_id=%v&client_secret=%v", twitchRefreshToken, twitchID, twitchSecret)
	req := RequestJSON{
		Method:  "POST",
		URL:     twitchTokenURL,
		Payload: payload,
		Headers: map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
	}
	var response TwitchRefreshResponse
	if err := s.MakeRequestMarshallJSON(req, &response); err != nil {
		s.Service.Logger.Error(err)
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
		s.Service.Logger.Error(err)
	}
	if response.ExpiresIn > 0 {
		s.Service.Logger.Info(fmt.Sprintf("Token is valid, expires in: %v ", response.ExpiresIn))
		return false
	}
	return true
}

// MakeRequestMarshallJSON makes a request and marshals the response into the target interface
func (s *SecretService) MakeRequestMarshallJSON(req RequestJSON, target any) error {
	httpReq, err := http.NewRequest(req.Method, req.URL, bytes.NewBuffer([]byte(req.Payload)))
	if err != nil {
		return err
	}
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}
	s.Service.Logger.Info(fmt.Sprintf("Sending request to: %v", req.URL))
	resp, err := s.Service.Client.Do(httpReq)
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
		s.Service.Logger.Error(errSpotifyMissingSecrets)
		return "", errSpotifyMissingSecrets
	}

	encodedToken := base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientSecret))
	params := url.Values{}
	params.Set("grant_type", "refresh_token")
	params.Set("refresh_token", refreshToken)

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(params.Encode()))
	if err != nil {
		s.Service.Logger.Error(err)
		return "", errInvalidRequest
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+encodedToken)

	s.Service.Logger.Info("Requesting New Spotify token")
	res, err := s.Service.Client.Do(req)
	if err != nil {
		s.Service.Logger.Error(err)
		return "", errHTTPRequest
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		s.Service.Logger.Error(errSpotifyNoToken)
		return "", errSpotifyNoToken
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		s.Service.Logger.Error(err)
		return "", errResponseParsing
	}

	var t SpotifyTokenResponse
	if err = json.Unmarshal(body, &t); err != nil {
		s.Service.Logger.Error(err)
		return "", errResponseParsing
	}

	if t.AccessToken == "" {
		s.Service.Logger.Error(errSpotifyNoToken)
		return "", errSpotifyNoToken
	}

	return t.AccessToken, nil
}
