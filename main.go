package main

import (
	"log"
	"os"

	"github.com/gempir/go-twitch-irc/v4"
	"github.com/joho/godotenv"
)

const (
	channel_name = "mr_mvaldes"
)

func main() {
	log.Println("Starting bot...")
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Could not read .env file")
	}

	token := os.Getenv("TWITCH_TOKEN")
	if token == "" {
		log.Fatal("TWITCH_TOKEN is empty")
	}
	log.Println("Loaded TOKEN file")
	log.Println("Connecting to Twitch...")
	client := twitch.NewClient(channel_name, token)

	client.OnPrivateMessage(func(message twitch.PrivateMessage) {
		if message.Message == "!dotfiles" {
			log.Println("Sending dotfiles link - requestor:", message.User.DisplayName)
			client.Say(channel_name, "Dotfiles - https://links.mvaldes.dev/dotfiles")
		}
		if message.Message == "!socials" {
			client.Say(channel_name, "Twitter - https://x.com/mr_mvaldes")
		}
	})

	client.OnUserNoticeMessage(func(message twitch.UserNoticeMessage) {
		client.Say(channel_name, "User notice message")
	})

	client.Join(channel_name)
	error := client.Connect()
	if err != nil {
		log.Fatal(error)
	}

}
