package secrets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
)

const (
	PROJECT_NAME  = "bots"
	CONFIG_NAME   = "tokens"
	USER_TOKEN    = "TWITCH_USER_TOKEN"
	REFRESH_TOKEN = "TWITCH_REFRESH_TOKEN"
	TWITCH_TOKEN  = "TWITCH_TOKEN"
	CLIENT_ID     = "TWITCH_CLIENT_ID"
	CLIENT_SECRET = "TWITCH_CLIENT_SECRET"
	DOPPLER_TOKEN = "DOPPLER_TOKEN"

	// API Endpoints
	TWITCH_TOKEN_URL = "https://id.twitch.tv/oauth2/token"
	DOPPLER_API_URL  = "https://api.doppler.com/v3/configs/config/secrets"
)

// SecretManager interface defines the contract for managing secrets
type SecretManager interface {
	// GetUserToken retrieves the user token from environment
	GetUserToken() string
	// BuildSecretHeaders builds headers for Twitch API requests
	BuildSecretHeaders() RequestHeader
	// GenerateNewToken generates a new token using refresh token
	GenerateNewToken() TwitchRefreshResponse
	// StoreNewTokens stores new tokens in Doppler
	StoreNewTokens(tokens TwitchRefreshResponse) bool
}

// SecretService implements SecretManager interface
type SecretService struct {
	Logger *slog.Logger
}

// NewSecretService creates a new instance of SecretService
func NewSecretService(logger *slog.Logger) SecretManager {
	return &SecretService{
		Logger: logger,
	}
}

// GetUserToken retrieves the user token from environment
func (s *SecretService) GetUserToken() string {
	return os.Getenv(USER_TOKEN)
}

// BuildSecretHeaders Returns the secrets from env variables to build headers for requests
func (s *SecretService) BuildSecretHeaders() RequestHeader {
	token := os.Getenv(TWITCH_TOKEN)
	clientID := os.Getenv(CLIENT_ID)
	return RequestHeader{
		Token:    token,
		ClientID: clientID,
	}
}

// GenerateNewToken creates a new token by using the existing refresh token
func (s *SecretService) GenerateNewToken() TwitchRefreshResponse {
	twitchId := os.Getenv(CLIENT_ID)
	twitchSecret := os.Getenv(CLIENT_SECRET)
	twitchRefreshToken := os.Getenv(REFRESH_TOKEN)
	payload := fmt.Sprintf("grant_type=refresh_token&refresh_token=%v&client_id=%v&client_secret=%v", twitchRefreshToken, twitchId, twitchSecret)
	headers := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}
	req := RequestJson{
		Method:  "POST",
		URL:     TWITCH_TOKEN_URL,
		Payload: payload,
		Headers: headers,
	}
	var response TwitchRefreshResponse
	if err := s.MakeRequestMarshallJson(&req, &response); err != nil {
		s.Logger.Error("Error", "Making Execution", req.URL, "Failed:", err)
	}
	return response
}

// MakeRequestMarshallJson makes a request and marshals the response into the target interface
func (s *SecretService) MakeRequestMarshallJson(req *RequestJson, target interface{}) error {
	httpReq, err := http.NewRequest(req.Method, req.URL, bytes.NewBuffer([]byte(req.Payload)))
	if err != nil {
		return err
	}
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}
	client := &http.Client{}
	s.Logger.Info("Sending request")
	resp, err := client.Do(httpReq)
	if err != nil {
		s.Logger.Error("Error", "Sending request:", err)
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
	dopplerToken := os.Getenv(DOPPLER_TOKEN)
	headers := map[string]string{
		"Accept":        "application/json",
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + dopplerToken,
	}

	payload := fmt.Sprintf(`{
		"project": "%v",
		"config": "%v",
    "secrets": {"TWITCH_USER_TOKEN": "%v", "TWITCH_REFRESH_TOKEN": "%v"}
	}`, PROJECT_NAME, CONFIG_NAME, tokens.AccessToken, tokens.RefreshToken)
	req := RequestJson{
		Method:  "POST",
		URL:     DOPPLER_API_URL,
		Payload: payload,
		Headers: headers,
	}
	s.Logger.Info("Storing new tokens in doppler")
	var response DopplerSecretUpdate
	if err := s.MakeRequestMarshallJson(&req, &response); err != nil {
		s.Logger.Error("Error", "Making Execution", req.URL, "Failed:", err)
		return false
	}
	return response.Success
}
