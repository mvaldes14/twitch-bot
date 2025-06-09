package secrets

// RequestJson represents a JSON HTTP request
type RequestJson struct {
	Method  string
	URL     string
	Payload string
	Headers map[string]string
}

// RequestHeader represents the headers needed for Twitch API requests
type RequestHeader struct {
	Token    string
	ClientID string
}

// TwitchRefreshResponse represents the response from Twitch token refresh
type TwitchRefreshResponse struct {
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	Scope        []string `json:"scope"`
	TokenType    string   `json:"token_type"`
}

// TwitchUserTokenResponse represents the response from getting a new user token
type TwitchUserTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

// DopplerSecretUpdate represents the response from Doppler API
type DopplerSecretUpdate struct {
	Messages []string `json:"messages"`
	Data     struct {
		Name string `json:"name"`
	} `json:"data"`
	Success bool `json:"success"`
}
