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

// InitSecrets initializes the secrets by checking the cache and generating new tokens if necessary.
// Tokens are stored in Redis and read from there on demand — no os.Setenv needed.
func (s *SecretService) InitSecrets() {
	// Twitch User Token (client credentials grant)
	if _, err := s.Cache.GetToken("TWITCH_USER_TOKEN"); err != nil {
		newToken, expiresIn, err := s.GenerateUserToken()
		if err != nil {
			s.Log.Error("Failed to generate twitch user token:", err)
		} else {
			if err := s.Cache.StoreToken(cache.Token{
				Key:        "TWITCH_USER_TOKEN",
				Value:      newToken,
				Expiration: time.Duration(expiresIn) * time.Second,
			}); err != nil {
				s.Log.Error("Failed to store twitch user token", err)
			}
		}
	}

	// Twitch App Token (refresh token grant)
	if _, err := s.Cache.GetToken("TWITCH_APP_TOKEN"); err != nil {
		newToken, expiresIn, err := s.RefreshAppToken()
		if err != nil {
			s.Log.Error("Failed to refresh twitch app token:", err)
		} else {
			if err := s.Cache.StoreToken(cache.Token{
				Key:        "TWITCH_APP_TOKEN",
				Value:      newToken,
				Expiration: time.Duration(expiresIn) * time.Second,
			}); err != nil {
				s.Log.Error("Failed to store twitch app token", err)
			}
		}
	}

	// Spotify Token
	if _, err := s.Cache.GetToken("SPOTIFY_TOKEN"); err != nil {
		newSpotifyToken, err := s.GetSpotifyToken()
		if err != nil {
			s.Log.Error("Failed to get spotify token:", err)
		} else {
			if err := s.Cache.StoreToken(cache.Token{
				Key:        "SPOTIFY_TOKEN",
				Value:      newSpotifyToken,
				Expiration: time.Duration(spotifyExpiration) * time.Second,
			}); err != nil {
				s.Log.Error("Failed to store spotify token", err)
			}
		}
	}
}

// BuildSecretHeaders reads the app token from Redis cache and returns headers for Twitch API requests
func (s *SecretService) BuildSecretHeaders() (RequestHeader, error) {
	token, err := s.Cache.GetToken(twitchAppToken)
	if err != nil || token == "" {
		s.Log.Error("Missing Twitch app token in cache", errMissingTokenOrID)
		return RequestHeader{}, errMissingTokenOrID
	}
	clientID := os.Getenv(twitchClientID)
	if clientID == "" {
		s.Log.Error("Missing Twitch Client ID in environment", errMissingTokenOrID)
		return RequestHeader{}, errMissingTokenOrID
	}
	return RequestHeader{
		Token:    token,
		ClientID: clientID,
	}, nil
}

// GetUserToken reads the user token from Redis cache
func (s *SecretService) GetUserToken() (string, error) {
	token, err := s.Cache.GetToken(twitchUserToken)
	if err != nil || token == "" {
		return "", fmt.Errorf("missing twitch user token in cache: %w", err)
	}
	return token, nil
}

// GenerateUserToken acquires a new token that is valid for 2 months
func (s *SecretService) GenerateUserToken() (string, int, error) {
	s.Log.Info("Generating new twitch user token")
	twitchID := os.Getenv(twitchClientID)
	twitchSecretVal := os.Getenv(twitchSecret)
	if twitchID == "" || twitchSecretVal == "" {
		return "", 0, errMissingTokenOrID
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
		return "", 0, fmt.Errorf("generate user token request failed: %w", err)
	}
	if response.AccessToken == "" {
		return "", 0, fmt.Errorf("generate user token returned empty access token")
	}

	expiresIn := response.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = twitchUserExpiration
	}

	return response.AccessToken, expiresIn, nil
}

// RefreshAppToken uses the refresh token to get a new one.
// It persists the rotated refresh token to Redis and returns the access token and its TTL.
func (s *SecretService) RefreshAppToken() (string, int, error) {
	twitchID := os.Getenv(twitchClientID)
	twitchSecretVal := os.Getenv(twitchSecret)

	// Read refresh token from Redis first, fall back to env var
	twitchRefreshTk, err := s.Cache.GetToken(twitchRefreshToken)
	if err != nil || twitchRefreshTk == "" {
		twitchRefreshTk = os.Getenv(twitchRefreshToken)
	}

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
		return "", 0, fmt.Errorf("refresh app token request failed: %w", err)
	}
	if response.AccessToken == "" {
		return "", 0, fmt.Errorf("refresh app token returned empty access token")
	}

	// Persist the rotated refresh token so subsequent refreshes use the new one
	if response.RefreshToken != "" {
		if err := s.Cache.StoreToken(cache.Token{
			Key:   twitchRefreshToken,
			Value: response.RefreshToken,
			// Refresh tokens don't expire on their own, use a long TTL
			Expiration: 0,
		}); err != nil {
			s.Log.Error("Failed to store rotated refresh token", err)
		}
	}

	expiresIn := response.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = twitchAppExpiration
	}

	return response.AccessToken, expiresIn, nil
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
		return true
	}
	return false
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

// refreshAndStoreAppToken refreshes the Twitch app token and stores it in Redis.
// Returns the new TTL in seconds.
func (s *SecretService) refreshAndStoreAppToken() (int, error) {
	newToken, expiresIn, err := s.RefreshAppToken()
	if err != nil {
		return 0, fmt.Errorf("failed to refresh app token: %w", err)
	}
	if err := s.Cache.StoreToken(cache.Token{
		Key:        twitchAppToken,
		Value:      newToken,
		Expiration: time.Duration(expiresIn) * time.Second,
	}); err != nil {
		return 0, fmt.Errorf("failed to store app token: %w", err)
	}
	s.Log.Info("Twitch app token refreshed, expires in:", expiresIn)
	return expiresIn, nil
}

// refreshAndStoreUserToken regenerates the Twitch user token and stores it in Redis.
// Returns the new TTL in seconds.
func (s *SecretService) refreshAndStoreUserToken() (int, error) {
	newToken, expiresIn, err := s.GenerateUserToken()
	if err != nil {
		return 0, fmt.Errorf("failed to generate user token: %w", err)
	}
	if err := s.Cache.StoreToken(cache.Token{
		Key:        twitchUserToken,
		Value:      newToken,
		Expiration: time.Duration(expiresIn) * time.Second,
	}); err != nil {
		return 0, fmt.Errorf("failed to store user token: %w", err)
	}
	s.Log.Info("Twitch user token refreshed, expires in:", expiresIn)
	return expiresIn, nil
}

// refreshAndStoreSpotifyToken refreshes the Spotify token and stores it in Redis.
func (s *SecretService) refreshAndStoreSpotifyToken() error {
	newToken, err := s.GetSpotifyToken()
	if err != nil {
		return fmt.Errorf("failed to refresh spotify token: %w", err)
	}
	if err := s.Cache.StoreToken(cache.Token{
		Key:        "SPOTIFY_TOKEN",
		Value:      newToken,
		Expiration: time.Duration(spotifyExpiration) * time.Second,
	}); err != nil {
		return fmt.Errorf("failed to store spotify token: %w", err)
	}
	s.Log.Info("Spotify token refreshed")
	return nil
}

// RefreshAppTokenAndStore refreshes the Twitch app token and stores it in Redis.
// Exported for use by other packages on 401 detection.
func (s *SecretService) RefreshAppTokenAndStore() error {
	_, err := s.refreshAndStoreAppToken()
	return err
}

// RefreshUserTokenAndStore regenerates the Twitch user token and stores it in Redis.
// Exported for use by other packages on 401 detection.
func (s *SecretService) RefreshUserTokenAndStore() error {
	_, err := s.refreshAndStoreUserToken()
	return err
}

// StartTokenRenewal launches a background goroutine that periodically validates
// and proactively refreshes tokens before they expire. It checks every renewalInterval
// and refreshes tokens when they fail validation or are nearing expiry.
// The goroutine exits when the provided context is cancelled.
func (s *SecretService) StartTokenRenewal(ctx context.Context) {
	const renewalInterval = 30 * time.Minute

	go func() {
		s.Log.Info("Token renewal goroutine started, checking every 30 minutes")
		ticker := time.NewTicker(renewalInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				s.Log.Info("Token renewal goroutine stopped")
				return
			case <-ticker.C:
				s.renewTokens()
			}
		}
	}()
}

// renewTokens validates each token and refreshes it if expired or missing.
func (s *SecretService) renewTokens() {
	// Twitch App Token — most critical, expires every ~4 hours
	appToken, err := s.Cache.GetToken(twitchAppToken)
	if err != nil || appToken == "" {
		s.Log.Info("Twitch app token missing from cache, refreshing")
		if _, err := s.refreshAndStoreAppToken(); err != nil {
			s.Log.Error("Background renewal: failed to refresh app token", err)
		}
	} else if !s.ValidateToken(appToken) {
		s.Log.Info("Twitch app token failed validation, refreshing")
		if _, err := s.refreshAndStoreAppToken(); err != nil {
			s.Log.Error("Background renewal: failed to refresh app token", err)
		}
	}

	// Twitch User Token — expires every ~60 days
	userToken, err := s.Cache.GetToken(twitchUserToken)
	if err != nil || userToken == "" {
		s.Log.Info("Twitch user token missing from cache, regenerating")
		if _, err := s.refreshAndStoreUserToken(); err != nil {
			s.Log.Error("Background renewal: failed to regenerate user token", err)
		}
	} else if !s.ValidateToken(userToken) {
		s.Log.Info("Twitch user token failed validation, regenerating")
		if _, err := s.refreshAndStoreUserToken(); err != nil {
			s.Log.Error("Background renewal: failed to regenerate user token", err)
		}
	}

	// Spotify Token — expires every hour
	if _, err := s.Cache.GetToken("SPOTIFY_TOKEN"); err != nil {
		s.Log.Info("Spotify token missing from cache, refreshing")
		if err := s.refreshAndStoreSpotifyToken(); err != nil {
			s.Log.Error("Background renewal: failed to refresh spotify token", err)
		}
	}
}
