// Package commands responds to chat events
package commands

import (
	"github.com/mvaldes14/twitch-bot/pkgs/types"
)

const (
	url    = "https://api.twitch.tv/helix/chat/messages"
	userID = "1792311"
)

// ParseMessage Parses the incoming messages from stream
func ParseMessage(msg types.ChatMessageEvent) {
	switch msg.Event.Message.Text {
	case "!commands":
		SendMessage("!github, !dotfiles")
	case "!github":
		SendMessage("https://links.mvaldes.dev/gh")
	case "!dotfiles":
		SendMessage("https://links.mvaldes.dev/dotfiles")
	case "!test":
		SendMessage("Test Me")
	}
}
