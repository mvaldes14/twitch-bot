// package secrets handles all interactions with secrets
package secrets

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/mvaldes14/twitch-bot/pkgs/telemetry"
)

const (
	projectName   = "bots"
	configName    = "tokens"
	userToken     = "TWITCH_USER_TOKEN"
	refreshToken  = "TWITCH_REFRESH_TOKEN"
	twitchToken   = "TWITCH_TOKEN"
	clientID      = "TWITCH_CLIENT_ID"
	dopplerSecret = "TWITCH_CLIENT_SECRET"
	dopplerToken  = "DOPPLER_TOKEN"

	// API Endpoints
	twitchTokenURL = "https://id.twitch.tv/oauth2/token"
	dopplerAPIURL  = "https://api.doppler.com/v3/configs/config/secrets"
)

// SecretService implements SecretManager interface
type SecretService struct {
	Log *telemetry.CustomLogger
}

// NewSecretService creates a new instance of SecretService
func NewSecretService() *SecretService {
	logger := telemetry.NewLogger("secrets")
	return &SecretService{Log: logger}
}

// GetUserToken retrieves the user token from environment
func (s *SecretService) GetUserToken() string {
	return os.Getenv(userToken)
}

// BuildSecretHeaders Returns the secrets from env variables to build headers for requests
func (s *SecretService) BuildSecretHeaders() (RequestHeader, error) {
	token := os.Getenv(twitchToken)
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

// GenerateNewToken creates a new token by using the existing refresh token
func (s *SecretService) GenerateNewToken() TwitchRefreshResponse {
	twitchID := os.Getenv(clientID)
	twitchSecret := os.Getenv(dopplerSecret)
	twitchRefreshToken := os.Getenv(refreshToken)
	payload := fmt.Sprintf("grant_type=refresh_token&refresh_token=%v&client_id=%v&client_secret=%v", twitchRefreshToken, twitchID, twitchSecret)
	headers := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}
	req := RequestJson{
		Method:  "POST",
		URL:     twitchTokenURL,
		Payload: payload,
		Headers: headers,
	}
	var response TwitchRefreshResponse
	if err := s.MakeRequestMarshallJSON(&req, &response); err != nil {
		s.Log.Error("Failed to make request", err)
	}
	return response
}

// MakeRequestMarshallJSON makes a request and marshals the response into the target interface
func (s *SecretService) MakeRequestMarshallJSON(req *RequestJson, target any) error {
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
		s.Log.Error("Error Sending request to doppler:", err)
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, target)
}

// StoreNewTokens stores new tokens in Doppler
func (s *SecretService) StoreNewTokens(tokens TwitchRefreshResponse) bool {
	dopplerToken := os.Getenv(dopplerToken)
	headers := map[string]string{
		"Accept":        "application/json",
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + dopplerToken,
	}

	payload := fmt.Sprintf(`{
		"project": "%v",
		"config": "%v",
    "secrets": {"TWITCH_USER_TOKEN": "%v", "TWITCH_REFRESH_TOKEN": "%v"}
	}`, projectName, configName, tokens.AccessToken, tokens.RefreshToken)
	req := RequestJson{
		Method:  "POST",
		URL:     dopplerAPIURL,
		Payload: payload,
		Headers: headers,
	}
	s.Log.Info("Storing new tokens in doppler")
	var response DopplerSecretUpdate
	if err := s.MakeRequestMarshallJSON(&req, &response); err != nil {
		s.Log.Error("Failed to send request", err)
		return false
	}
	return response.Success
}
