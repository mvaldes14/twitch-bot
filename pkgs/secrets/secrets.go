package secrets

import (
	"fmt"
	"os"

	"github.com/mvaldes14/twitch-bot/pkgs/types"
)

const (
	PROJECT_NAME  = "bots"
	CONFIG_NAME   = "tokens"
	USER_TOKEN    = "TWITCH_USER_TOKEN"
	REFRESH_TOKEN = "TWITCH_REFRESH_TOKEN"
)

func GetUserToken() string {
	return os.Getenv(USER_TOKEN)
}

// GenerateNewToken creates a new token by using the existing refresh token
func GenerateNewToken() types.TwitchRefreshResponse {
	twitchId := os.Getenv("TWITCH_CLIENT_ID")
	twitchSecret := os.Getenv("TWITCH_CLIENT_SECRET")
	twitchRefreshToken := os.Getenv("TWITCH_REFRESH_TOKEN")
	payload := fmt.Sprintf("grant_type=refresh_token&refresh_token=%v&client_id=%v&client_secret=%v", twitchRefreshToken, twitchId, twitchSecret)
	headers := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}
	req := types.RequestJson{
		Method:  "POST",
		URL:     "https://id.twitch.tv/oauth2/token",
		Payload: payload,
		Headers: headers,
	}
	var response types.TwitchRefreshResponse
	if err := MakeRequestMarshallJson(&req, &response); err != nil {
		logger.Error("Error", "Making Execution", req.URL, "Failed:", err)
	}
	return response
}

func StoreNewTokens(tokens types.TwitchRefreshResponse) bool {
	dopplerToken := os.Getenv("DOPPLER_TOKEN")
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
	req := types.RequestJson{
		Method:  "POST",
		URL:     "https://api.doppler.com/v3/configs/config/secrets",
		Payload: payload,
		Headers: headers,
	}
	logger.Info("Storing new tokens in doppler")
	var response types.DopplerSecretUpdate
	if err := MakeRequestMarshallJson(&req, &response); err != nil {
		logger.Error("Error", "Making Execution", req.URL, "Failed:", err)
	}
	return response.Success
}
