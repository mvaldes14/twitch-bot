// Package secrets handles all interactions with secrets
package secrets

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/mvaldes14/twitch-bot/pkgs/cache"
	"github.com/mvaldes14/twitch-bot/pkgs/telemetry"
)

const (
	twitchRefreshToken = "TWITCH_REFRESH_TOKEN"
	twitchAppToken     = "TWITCH_APP_TOKEN"
	twitchUserToken    = "TWITCH_USER_TOKEN"
	twitchClientID     = "TWITCH_CLIENT_ID"
	twitchSecret       = "TWITCH_CLIENT_SECRET"
	requestTimeout     = 30 * time.Second

	// API Endpoints
	twitchTokenURL = "https://id.twitch.tv/oauth2/token"
	twitchValidURL = "https://id.twitch.tv/oauth2/validate"
)

var (
	errDopplerSaveSecret   = errors.New("Failed to store secret in Doppler")
	errDopplerMissingToken = errors.New("Doppler token not found in environment")
	errDopplerAPIErr       = errors.New("Error received from Doppler API")
	errMissingTokenOrID    = errors.New("Token or Client ID not found in environment")
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

// BuildSecretHeaders Returns the secrets from env variables to build headers for requests
func (s *SecretService) BuildSecretHeaders() (RequestHeader, error) {
	token := os.Getenv(twitchAppToken)
	clientID := os.Getenv(twitchClientID)
	if token == "" || clientID == "" {
		s.Log.Error("Token or Client ID not found in environment", errDopplerMissingToken)
		return RequestHeader{}, errDopplerMissingToken
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
		s.Log.Error("Failed to make request generating user token", err)
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
	httpReq, err := http.NewRequest(req.Method, req.URL, bytes.NewBuffer([]byte(req.Payload)))
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
	if resp.StatusCode != http.StatusOK {
		s.Log.Error("Error received from Doppler API", errDopplerAPIErr)
		return errDopplerAPIErr
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, target)
}
