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
	"go.opentelemetry.io/otel/attribute"
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
	twitchUserExpiration = 14400 // 4 hours in seconds
	twitchAppExpiration  = 14400
	spotifyExpiration    = 3600

	// API Endpoints
	twitchTokenURL = "https://id.twitch.tv/oauth2/token"
	twitchValidURL = "https://id.twitch.tv/oauth2/validate"
	tokenURL       = "https://accounts.spotify.com/api/token"
)

var (
	errMissingTokenOrID = errors.New("token or client ID not found in environment")
	errSpotifyNoToken   = errors.New("failed to produce a new token")
	errInvalidRequest   = errors.New("failed to create HTTP request")
	errHTTPRequest      = errors.New("HTTP request failed")
	errResponseParsing  = errors.New("failed to parse response")
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
	cacheService := cache.NewCacheService()
	httpClient := &http.Client{Timeout: requestTimeout}
	return &SecretService{Log: logger, Cache: cacheService, httpClient: httpClient}
}

// GetEnvironmentVariable retrieves an environment variable and validates it exists and is not empty.
// Returns a clear error message indicating which variable is missing and its purpose.
// Used for: TWITCH_CLIENT_ID, TWITCH_CLIENT_SECRET, SPOTIFY_CLIENT_ID, SPOTIFY_CLIENT_SECRET, ADMIN_TOKEN, TWITCH_REFRESH_TOKEN
func (s *SecretService) GetEnvironmentVariable(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		err := fmt.Errorf("required environment variable '%s' is missing or empty - cannot proceed", key)
		s.Log.Error(err.Error(), err)
		return "", err
	}
	return value, nil
}

// InitSecrets initializes the secrets by loading tokens.
// App tokens (TWITCH_APP_TOKEN) are auto-generated via client credentials.
// User tokens (TWITCH_USER_TOKEN) are generated from TWITCH_REFRESH_TOKEN and expire every 4 hours. They are automatically refreshed by the background goroutine before expiry.
func (s *SecretService) InitSecrets() {
	ctx := context.Background()
	_, span := telemetry.StartSpan(ctx, "secrets.init_secrets")
	defer span.End()

	// Twitch User Token - generated from refresh token, expires every 4 hours
	if _, err := s.Cache.GetToken("TWITCH_USER_TOKEN"); err != nil {
		// Try to load from environment variable first (for initial startup)
		userTokenFromEnv := os.Getenv(twitchUserToken)
		if userTokenFromEnv != "" {
			// Store the environment variable value in Redis with 4-hour TTL
			s.Log.Info("[SOURCE: ENV VAR] TWITCH_USER_TOKEN loaded from environment variable")
			if err := s.Cache.StoreToken(cache.Token{
				Key:        "TWITCH_USER_TOKEN",
				Value:      userTokenFromEnv,
				Expiration: time.Duration(twitchUserExpiration) * time.Second,
			}); err != nil {
				s.Log.Error("Failed to store TWITCH_USER_TOKEN in Redis:", err)
			}
		} else {
			// No env var, try to generate from refresh token
			s.Log.Info("[SOURCE: GENERATED] TWITCH_USER_TOKEN not in environment, generating from refresh token")
			newToken, expiresIn, err := s.RefreshUserToken()
			if err != nil {
				s.Log.Error("Failed to generate TWITCH_USER_TOKEN from refresh token - initial token may not have been provided:", err)
			} else {
				if err := s.Cache.StoreToken(cache.Token{
					Key:        "TWITCH_USER_TOKEN",
					Value:      newToken,
					Expiration: time.Duration(expiresIn) * time.Second,
				}); err != nil {
					s.Log.Error("Failed to store TWITCH_USER_TOKEN in Redis:", err)
				}
			}
		}
	}

	// Twitch App Token (refresh token grant)
	if _, err := s.Cache.GetToken("TWITCH_APP_TOKEN"); err != nil {
		s.Log.Info("[SOURCE: GENERATED] TWITCH_APP_TOKEN not in cache, generating from refresh token")
		newToken, expiresIn, err := s.RefreshAppToken()
		if err != nil {
			s.Log.Error("Failed to generate TWITCH_APP_TOKEN - check if TWITCH_REFRESH_TOKEN is set and token generation succeeded:", err)
		} else {
			if err := s.Cache.StoreToken(cache.Token{
				Key:        "TWITCH_APP_TOKEN",
				Value:      newToken,
				Expiration: time.Duration(expiresIn) * time.Second,
			}); err != nil {
				s.Log.Error("Failed to store TWITCH_APP_TOKEN in Redis:", err)
			}
		}
	}

	// Spotify Token
	if _, err := s.Cache.GetToken("SPOTIFY_TOKEN"); err != nil {
		s.Log.Info("[SOURCE: GENERATED] SPOTIFY_TOKEN not in cache, generating from refresh token")
		newSpotifyToken, err := s.GetSpotifyToken()
		if err != nil {
			s.Log.Error("Failed to generate SPOTIFY_TOKEN - check if SPOTIFY_REFRESH_TOKEN and credentials are set:", err)
		} else {
			if err := s.Cache.StoreToken(cache.Token{
				Key:        "SPOTIFY_TOKEN",
				Value:      newSpotifyToken,
				Expiration: time.Duration(spotifyExpiration) * time.Second,
			}); err != nil {
				s.Log.Error("Failed to store SPOTIFY_TOKEN in Redis:", err)
			}
		}
	}
}

// BuildSecretHeaders reads the app token from Redis cache and returns headers for Twitch API requests.
// Validates that the token exists in cache and that the client ID environment variable is set before returning, fails early if missing.
func (s *SecretService) BuildSecretHeaders() (RequestHeader, error) {
	ctx := context.Background()
	_, span := telemetry.StartSpan(ctx, "secrets.build_headers")
	defer span.End()

	// Validate token exists in cache before proceeding
	token, err := s.Cache.GetToken(twitchAppToken)
	if err != nil || token == "" {
		cacheMissingErr := fmt.Errorf("TWITCH_APP_TOKEN not found in Redis cache - required for Twitch API requests. Check if token generation succeeded during startup: %w", err)
		s.Log.Error("Cannot build headers - TWITCH_APP_TOKEN missing from cache", cacheMissingErr)
		telemetry.RecordError(span, cacheMissingErr)
		return RequestHeader{}, cacheMissingErr
	}

	// Validate client ID environment variable exists
	clientID, err := s.GetEnvironmentVariable(twitchClientID)
	if err != nil {
		envErr := fmt.Errorf("TWITCH_CLIENT_ID not found in environment - required for Twitch API requests: %w", err)
		s.Log.Error("Cannot build headers - TWITCH_CLIENT_ID missing from environment", envErr)
		telemetry.RecordError(span, envErr)
		return RequestHeader{}, envErr
	}

	return RequestHeader{
		Token:    token,
		ClientID: clientID,
	}, nil
}

// GetUserToken reads the user token from Redis cache.
// Returns a specific error if the token is missing, which would prevent user-scoped Twitch API operations.
func (s *SecretService) GetUserToken() (string, error) {
	ctx := context.Background()
	_, span := telemetry.StartSpan(ctx, "secrets.get_user_token")
	defer span.End()

	token, err := s.Cache.GetToken(twitchUserToken)
	if err != nil || token == "" {
		tokenErr := fmt.Errorf("TWITCH_USER_TOKEN not found in Redis cache - required for Twitch API user-scoped operations. Token expires every 4 hours. If missing, the background renewal goroutine will automatically generate a new one. If still missing after refresh, the TWITCH_REFRESH_TOKEN may be invalid or revoked: %w", err)
		s.Log.Error("TWITCH_USER_TOKEN missing from cache", tokenErr)
		telemetry.RecordError(span, tokenErr)
		return "", tokenErr
	}
	return token, nil
}

// RefreshAppToken generates a new app access token using client credentials.
// App access tokens are required for Twitch EventSub webhook subscriptions.
func (s *SecretService) RefreshAppToken() (string, int, error) {
	ctx := context.Background()
	_, span := telemetry.StartExternalSpan(ctx, "twitch.refresh_app_token", "twitch", "refresh_app_token")
	defer span.End()

	twitchID := os.Getenv(twitchClientID)
	twitchSecretVal := os.Getenv(twitchSecret)

	if twitchID == "" || twitchSecretVal == "" {
		missingErr := fmt.Errorf("TWITCH_CLIENT_ID or TWITCH_CLIENT_SECRET not set")
		s.Log.Error("Cannot generate app token - credentials missing", missingErr)
		telemetry.RecordError(span, missingErr)
		telemetry.IncrementTokenRefreshTotal(ctx, "app", "error")
		return "", 0, missingErr
	}

	payload := fmt.Sprintf("grant_type=client_credentials&client_id=%v&client_secret=%v", twitchID, twitchSecretVal)
	req := RequestJSON{
		Method:  "POST",
		URL:     twitchTokenURL,
		Payload: payload,
		Headers: map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
	}
	var response TwitchRefreshResponse
	if err := s.MakeRequestMarshallJSON(req, &response); err != nil {
		s.Log.Error("Failed to make request for app token", err)
		telemetry.RecordError(span, err)
		telemetry.IncrementTokenRefreshTotal(ctx, "app", "error")
		return "", 0, fmt.Errorf("app token request failed: %w", err)
	}
	if response.AccessToken == "" {
		emptyErr := fmt.Errorf("app token request returned empty access token")
		telemetry.RecordError(span, emptyErr)
		telemetry.IncrementTokenRefreshTotal(ctx, "app", "error")
		return "", 0, emptyErr
	}

	expiresIn := response.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = twitchAppExpiration
	}

	telemetry.AddSpanAttributes(span, attribute.Int("token.expires_in", expiresIn))
	telemetry.IncrementTokenRefreshTotal(ctx, "app", "success")
	telemetry.RecordTokenTTL(ctx, "app", float64(expiresIn))
	return response.AccessToken, expiresIn, nil
}

// RefreshUserToken generates a new user token from the refresh token. User tokens expire every 4 hours and are auto-refreshed by the background goroutine.
func (s *SecretService) RefreshUserToken() (string, int, error) {
	ctx := context.Background()
	_, span := telemetry.StartExternalSpan(ctx, "twitch.refresh_user_token", "twitch", "refresh_user_token")
	defer span.End()

	twitchID := os.Getenv(twitchClientID)
	twitchSecretVal := os.Getenv(twitchSecret)

	// Read refresh token from Redis first, fall back to env var
	twitchRefreshTk, err := s.Cache.GetToken(twitchRefreshToken)
	if err != nil || twitchRefreshTk == "" {
		twitchRefreshTk = os.Getenv(twitchRefreshToken)
	}

	if twitchID == "" || twitchSecretVal == "" || twitchRefreshTk == "" {
		telemetry.RecordError(span, errMissingTokenOrID)
		telemetry.IncrementTokenRefreshTotal(ctx, "user", "error")
		return "", 0, errMissingTokenOrID
	}

	payload := fmt.Sprintf("client_id=%v&client_secret=%v&grant_type=refresh_token&refresh_token=%v", twitchID, twitchSecretVal, twitchRefreshTk)
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
		s.Log.Error("Failed to make request refreshing user token", err)
		telemetry.RecordError(span, err)
		telemetry.IncrementTokenRefreshTotal(ctx, "user", "error")
		return "", 0, fmt.Errorf("refresh user token request failed: %w", err)
	}
	if response.AccessToken == "" {
		emptyErr := fmt.Errorf("refresh user token returned empty access token")
		telemetry.RecordError(span, emptyErr)
		telemetry.IncrementTokenRefreshTotal(ctx, "user", "error")
		return "", 0, emptyErr
	}

	expiresIn := response.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = twitchUserExpiration
	}

	// Store new refresh token if provided in response
	if response.RefreshToken != "" {
		if err := s.Cache.StoreToken(cache.Token{
			Key:        twitchRefreshToken,
			Value:      response.RefreshToken,
			Expiration: 365 * 24 * time.Hour, // Long TTL for refresh token
		}); err != nil {
			s.Log.Info("Failed to store new refresh token in Redis:", err)
		}
	}

	telemetry.AddSpanAttributes(span, attribute.Int("token.expires_in", expiresIn))
	telemetry.IncrementTokenRefreshTotal(ctx, "user", "success")
	telemetry.RecordTokenTTL(ctx, "user", float64(expiresIn))
	return response.AccessToken, expiresIn, nil
}

// ValidateToken checks if the token is still valid
func (s *SecretService) ValidateToken(token string) bool {
	ctx := context.Background()
	_, span := telemetry.StartExternalSpan(ctx, "twitch.validate_token", "twitch", "validate_token")
	defer span.End()

	var response TwitchValidResponse
	req := RequestJSON{
		Method:  "GET",
		URL:     twitchValidURL,
		Headers: map[string]string{"Authorization": "Bearer " + token},
		Payload: "",
	}
	if err := s.MakeRequestMarshallJSON(req, &response); err != nil {
		s.Log.Error("Failed to make request validating token", err)
		telemetry.RecordError(span, err)
		return false
	}

	telemetry.AddSpanAttributes(span,
		attribute.Bool("token.valid", response.ExpiresIn > 0),
		attribute.Int("token.expires_in", response.ExpiresIn),
	)

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

// GetSpotifyToken retrieves a new Spotify token using the refresh token.
// Returns a specific error indicating which Spotify credentials are missing.
func (s *SecretService) GetSpotifyToken() (string, error) {
	ctx := context.Background()
	_, span := telemetry.StartExternalSpan(ctx, "spotify.refresh_token", "spotify", "refresh_token")
	defer span.End()

	refreshToken := os.Getenv(spotifyRefreshToken)
	clientID := os.Getenv(spotifyClientID)
	clientSecret := os.Getenv(spotifyClientSecret)

	// Build detailed error message indicating which credentials are missing
	var missingVars []string
	if refreshToken == "" {
		missingVars = append(missingVars, "SPOTIFY_REFRESH_TOKEN")
	}
	if clientID == "" {
		missingVars = append(missingVars, "SPOTIFY_CLIENT_ID")
	}
	if clientSecret == "" {
		missingVars = append(missingVars, "SPOTIFY_CLIENT_SECRET")
	}

	if len(missingVars) > 0 {
		missingErr := fmt.Errorf("missing Spotify credentials in environment: %v - required to refresh Spotify access tokens. Pass these as environment variables", missingVars)
		s.Log.Error(missingErr.Error(), missingErr)
		telemetry.RecordError(span, missingErr)
		return "", missingErr
	}

	encodedToken := base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientSecret))
	params := url.Values{}
	params.Set("grant_type", "refresh_token")
	params.Set("refresh_token", refreshToken)

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(params.Encode()))
	if err != nil {
		s.Log.Error("Error forming request for GetSpotifyToken", err)
		telemetry.RecordError(span, err)
		return "", errInvalidRequest
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+encodedToken)

	s.Log.Info("Requesting New Spotify token")
	res, err := s.httpClient.Do(req)
	if err != nil {
		s.Log.Error("Error sending request to get new token", err)
		telemetry.RecordError(span, err)
		return "", errHTTPRequest
	}
	defer res.Body.Close()

	telemetry.SetSpanStatus(span, res.StatusCode)

	if res.StatusCode != http.StatusOK {
		s.Log.Error("Token request failed with status", fmt.Errorf("status: %d", res.StatusCode))
		return "", errSpotifyNoToken
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		s.Log.Error("Error reading token response body", err)
		telemetry.RecordError(span, err)
		return "", errResponseParsing
	}

	var t SpotifyTokenResponse
	if err = json.Unmarshal(body, &t); err != nil {
		s.Log.Error("Error unmarshalling token response", err)
		telemetry.RecordError(span, err)
		return "", errResponseParsing
	}

	if t.AccessToken == "" {
		s.Log.Error("Received empty access token", errSpotifyNoToken)
		telemetry.RecordError(span, errSpotifyNoToken)
		telemetry.IncrementTokenRefreshTotal(ctx, "spotify", "error")
		return "", errSpotifyNoToken
	}

	telemetry.IncrementTokenRefreshTotal(ctx, "spotify", "success")
	return t.AccessToken, nil
}

// refreshAndStoreAppToken refreshes the Twitch app token and stores it in Redis.
func (s *SecretService) refreshAndStoreAppToken() error {
	ctx := context.Background()
	_, span := telemetry.StartSpan(ctx, "secrets.refresh_and_store_app_token")
	defer span.End()

	newToken, expiresIn, err := s.RefreshAppToken()
	if err != nil {
		telemetry.RecordError(span, err)
		return fmt.Errorf("failed to refresh app token: %w", err)
	}
	if err := s.Cache.StoreToken(cache.Token{
		Key:        twitchAppToken,
		Value:      newToken,
		Expiration: time.Duration(expiresIn) * time.Second,
	}); err != nil {
		telemetry.RecordError(span, err)
		return fmt.Errorf("failed to store app token: %w", err)
	}
	telemetry.AddSpanAttributes(span, attribute.Int("token.expires_in", expiresIn))
	s.Log.Info("Twitch app token refreshed, expires in:", expiresIn)
	return nil
}

// refreshAndStoreUserToken refreshes the Twitch user token and stores it in Redis.
func (s *SecretService) refreshAndStoreUserToken() error {
	ctx := context.Background()
	_, span := telemetry.StartSpan(ctx, "secrets.refresh_and_store_user_token")
	defer span.End()

	newToken, expiresIn, err := s.RefreshUserToken()
	if err != nil {
		telemetry.RecordError(span, err)
		return fmt.Errorf("failed to refresh user token: %w", err)
	}
	if err := s.Cache.StoreToken(cache.Token{
		Key:        twitchUserToken,
		Value:      newToken,
		Expiration: time.Duration(expiresIn) * time.Second,
	}); err != nil {
		telemetry.RecordError(span, err)
		return fmt.Errorf("failed to store user token: %w", err)
	}
	telemetry.AddSpanAttributes(span, attribute.Int("token.expires_in", expiresIn))
	s.Log.Info("Twitch user token refreshed, expires in:", expiresIn)
	return nil
}

// refreshAndStoreSpotifyToken refreshes the Spotify token and stores it in Redis.
func (s *SecretService) refreshAndStoreSpotifyToken() error {
	ctx := context.Background()
	_, span := telemetry.StartSpan(ctx, "secrets.refresh_and_store_spotify_token")
	defer span.End()

	newToken, err := s.GetSpotifyToken()
	if err != nil {
		telemetry.RecordError(span, err)
		return fmt.Errorf("failed to refresh spotify token: %w", err)
	}
	if err := s.Cache.StoreToken(cache.Token{
		Key:        "SPOTIFY_TOKEN",
		Value:      newToken,
		Expiration: time.Duration(spotifyExpiration) * time.Second,
	}); err != nil {
		telemetry.RecordError(span, err)
		return fmt.Errorf("failed to store spotify token: %w", err)
	}
	s.Log.Info("Spotify token refreshed")
	return nil
}

// RefreshAppTokenAndStore refreshes the Twitch app token and stores it in Redis.
// Exported for use by other packages on 401 detection.
func (s *SecretService) RefreshAppTokenAndStore() error {
	return s.refreshAndStoreAppToken()
}

// RefreshUserTokenAndStore refreshes the Twitch user token and stores it in Redis.
// Exported for use by other packages on 401 detection.
func (s *SecretService) RefreshUserTokenAndStore() error {
	return s.refreshAndStoreUserToken()
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
	ctx := context.Background()
	_, span := telemetry.StartSpan(ctx, "secrets.renew_tokens_cycle")
	defer span.End()

	// Twitch App Token — most critical, expires every ~4 hours
	appToken, err := s.Cache.GetToken(twitchAppToken)
	switch {
	case err != nil || appToken == "":
		s.Log.Info("Twitch app token missing from cache, refreshing")
		telemetry.AddSpanAttributes(span, attribute.String("app_token.action", "refresh_missing"))
		if err := s.refreshAndStoreAppToken(); err != nil {
			s.Log.Error("Background renewal: failed to refresh app token", err)
			telemetry.RecordError(span, err)
		}
	case !s.ValidateToken(appToken):
		s.Log.Info("Twitch app token failed validation, refreshing")
		telemetry.AddSpanAttributes(span, attribute.String("app_token.action", "refresh_invalid"))
		telemetry.IncrementTokenValidationTotal(ctx, "app", false)
		if err := s.refreshAndStoreAppToken(); err != nil {
			s.Log.Error("Background renewal: failed to refresh app token", err)
			telemetry.RecordError(span, err)
		}
	default:
		telemetry.AddSpanAttributes(span, attribute.String("app_token.action", "still_valid"))
		telemetry.IncrementTokenValidationTotal(ctx, "app", true)
	}

	// Twitch User Token — expires every 4 hours, proactively refresh
	userToken, err := s.Cache.GetToken(twitchUserToken)
	switch {
	case err != nil || userToken == "":
		s.Log.Info("Twitch user token missing from cache, generating new token")
		telemetry.AddSpanAttributes(span, attribute.String("user_token.action", "refresh_missing"))
		if err := s.refreshAndStoreUserToken(); err != nil {
			s.Log.Error("Background renewal: failed to generate user token", err)
			telemetry.RecordError(span, err)
		}
	case !s.ValidateToken(userToken):
		s.Log.Info("Twitch user token failed validation, generating new token")
		telemetry.AddSpanAttributes(span, attribute.String("user_token.action", "refresh_invalid"))
		telemetry.IncrementTokenValidationTotal(ctx, "user", false)
		if err := s.refreshAndStoreUserToken(); err != nil {
			s.Log.Error("Background renewal: failed to generate user token", err)
			telemetry.RecordError(span, err)
		}
	default:
		telemetry.AddSpanAttributes(span, attribute.String("user_token.action", "still_valid"))
		telemetry.IncrementTokenValidationTotal(ctx, "user", true)
	}

	// Spotify Token — expires every hour
	if _, err := s.Cache.GetToken("SPOTIFY_TOKEN"); err != nil {
		s.Log.Info("Spotify token missing from cache, refreshing")
		telemetry.AddSpanAttributes(span, attribute.String("spotify_token.action", "refresh_missing"))
		if err := s.refreshAndStoreSpotifyToken(); err != nil {
			s.Log.Error("Background renewal: failed to refresh spotify token", err)
			telemetry.RecordError(span, err)
		}
	} else {
		telemetry.AddSpanAttributes(span, attribute.String("spotify_token.action", "still_valid"))
	}
}
