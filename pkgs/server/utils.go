package server

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/mvaldes14/twitch-bot/pkgs/types"
)

func BuildHeaders() types.RequestHeader {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	}
	token := os.Getenv("TWITCH_TOKEN")
	clientID := os.Getenv("TWITCH_CLIENT_ID")
	return types.RequestHeader{
		Token:    token,
		ClientID: clientID,
	}
}
