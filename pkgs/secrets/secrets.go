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

	"github.com/mvaldes14/twitch-bot/pkgs/cache"
	"github.com/mvaldes14/twitch-bot/pkgs/telemetry"
)

const (
	projectName     = "bots"
	configName      = "tokens"
	userToken       = "TWITCH_USER_TOKEN"
	refreshToken    = "TWITCH_REFRESH_TOKEN"
	twitchAppToken  = "TWITCH_APP_TOKEN"
	twitchUserToken = "TWITCH_USER_TOKEN"
	clientID        = "TWITCH_CLIENT_ID"
	secretID        = "TWITCH_CLIENT_SECRET"
	dopplerToken    = "DOPPLER_TOKEN"

	// API Endpoints
	twitchTokenURL = "https://id.twitch.tv/oauth2/token"
	dopplerAPIURL  = "https://api.doppler.com/v3/configs/config/secrets"
)

var (
	errDopplerSaveSecret   = errors.New("Failed to store secret in Doppler")
	errDopplerMissingToken = errors.New("Doppler token not found in environment")
	errDopplerAPIErr       = errors.New("Error received from Doppler API")
)

// SecretService implements SecretManager interface
type SecretService struct {
	Log   *telemetry.CustomLogger
	Cache *cache.Service
}

// NewSecretService creates a new instance of SecretService
func NewSecretService() *SecretService {
	logger := telemetry.NewLogger("secrets")
	cache := cache.NewCacheService()
	return &SecretService{Log: logger, Cache: cache}
}

// GetUserToken retrieves the user token from environment
func (s *SecretService) GetUserToken() string {
	return os.Getenv(userToken)
}

// BuildSecretHeaders Returns the secrets from env variables to build headers for requests
func (s *SecretService) BuildSecretHeaders() (RequestHeader, error) {
	token := os.Getenv(twitchAppToken)
	clientID := os.Getenv(clientID)
	if token == "" || clientID == "" {
		err := errors.New("Token or Client ID not found in environment")
		s.Log.Error("Token or Client ID not found in environment", err)
		return RequestHeader{}, err
	}
	return RequestHeader{
		Token:    token,
		ClientID: clientID,
	}, nil
}

// GenerateUserToken acquires a new token that is valid for 2 months
func (s *SecretService) GenerateUserToken() (TwitchUserTokenResponse, error) {
	s.Log.Info("Generating new user token")
	twitchID := os.Getenv(clientID)
	twitchSecret := os.Getenv(secretID)
	if twitchID == "" || twitchSecret == "" {
		s.Log.Error("Twitch ID or Secret not found in environment", errors.New("Twitch ID or Secret not found"))
		return TwitchUserTokenResponse{}, errors.New("Twitch ID or Secret not found in environment")
	}
	payload := fmt.Sprintf("client_id=%v&client_secret=%v&grant_type=client_credentials", twitchID, twitchSecret)
	headers := map[string]string{
		"Content-Type": "application/json",
	}
	req := RequestJSON{
		Method:  "POST",
		URL:     twitchTokenURL,
		Payload: payload,
		Headers: headers,
	}
	var response TwitchUserTokenResponse
	s.Cache.StoreToken(cache.Token{
		Key:        "TWITCH_USER_TOKEN",
		Value:      response.AccessToken,
		Expiration: response.ExpiresIn,
	})
	fmt.Println(response)
	if err := s.MakeRequestMarshallJSON(req, &response); err != nil {
		s.Log.Error("Failed to make request generating user token", err)
	}
	return response, nil
}

// RefreshAppToken uses the refresh token to get a new one
func (s *SecretService) RefreshAppToken() TwitchRefreshResponse {
	twitchID := os.Getenv(clientID)
	twitchSecret := os.Getenv(secretID)
	twitchRefreshToken := os.Getenv(refreshToken)
	payload := fmt.Sprintf("grant_type=refresh_token&refresh_token=%v&client_id=%v&client_secret=%v", twitchRefreshToken, twitchID, twitchSecret)
	headers := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}
	req := RequestJSON{
		Method:  "POST",
		URL:     twitchTokenURL,
		Payload: payload,
		Headers: headers,
	}
	var response TwitchRefreshResponse
	if err := s.MakeRequestMarshallJSON(req, &response); err != nil {
		s.Log.Error("Failed to make request refreshing token", err)
	}
	return response
}

// ValidateDopplerToken checks if the Doppler token is valid
func (s *SecretService) ValidateDopplerToken(token string) error {
	headers := map[string]string{
		"Accept":        "application/json",
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + token,
	}
	req := RequestJSON{
		Method:  "GET",
		URL:     dopplerAPIURL,
		Payload: "",
		Headers: headers,
	}
	var response DopplerSecretUpdate
	if err := s.MakeRequestMarshallJSON(req, &response); err != nil {
		s.Log.Error("Failed to send request to validate Doppler token", err)
		return errDopplerAPIErr
	}
	if !response.Success {
		return errDopplerAPIErr
	}
	return nil
}

// StoreNewTokens stores new tokens in Doppler
func (s *SecretService) StoreNewTokens(key, value string) error {
	dopplerToken := os.Getenv(dopplerToken)
	if err := s.ValidateDopplerToken(dopplerToken); err != nil {
		return errors.New("Doppler token is not valid")
	}
	headers := map[string]string{
		"Accept":        "application/json",
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + dopplerToken,
	}
	var payload string
	payload = fmt.Sprintf(`{
		"project": "%v",
		"config": "%v",
    "secrets": {"%v": "%v"}
	}`, projectName, configName, key, value)

	req := RequestJSON{
		Method:  "POST",
		URL:     dopplerAPIURL,
		Payload: payload,
		Headers: headers,
	}
	s.Log.Info("Storing new tokens in Doppler value: ", key)
	var response DopplerSecretUpdate
	if err := s.MakeRequestMarshallJSON(req, &response); err != nil {
		s.Log.Error("Failed to send request", err)
		return errDopplerSaveSecret
	}
	if !response.Success {
		return errDopplerSaveSecret
	}
	return nil
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
	client := &http.Client{}
	s.Log.Info("Sending request to doppler")
	resp, err := client.Do(httpReq)
	if err != nil {
		s.Log.Error("Error Sending request to doppler", err)
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
