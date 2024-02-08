package main

import (
	"log"
	"os"

	"github.com/gempir/go-twitch-irc/v4"
)

const (
	channel_name = "mr_mvaldes"
)

func main() {
	log.Println("Starting bot...")

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
		if message.Message == "!github" {
			client.Say(channel_name, "Github - https://links.mvaldes.dev/gh")
		}
		if message.Message == "!commands" {
			client.Say(channel_name, "Commands - !dotfiles, !socials, !github")
		}
	})

	client.OnUserNoticeMessage(func(message twitch.UserNoticeMessage) {
		client.Say(channel_name, "User notice message")
    log.Println(message)
	})

	client.Join(channel_name)
	err := client.Connect()
	if err != nil {
		log.Fatal(err)
	}
}
