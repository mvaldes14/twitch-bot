package routes

import (
	"context"
	"os"
	"strings"
	"time"
)

const (
	socialReminderEnabledEnv  = "SOCIAL_REMINDER_ENABLED"
	socialReminderIntervalEnv = "SOCIAL_REMINDER_INTERVAL"
	socialReminderMessagesEnv = "SOCIAL_REMINDER_MESSAGES"

	defaultSocialReminderInterval = 30 * time.Minute
	minSocialReminderInterval     = 10 * time.Minute
)

var defaultSocialReminderMessages = []string{
	"Si te gusta este contenido, también subo videos a YouTube sobre DevOps, homelab, k8s y tooling: https://umami.mvaldes.dev/q/youtube",
	"¿Quieres seguir la conversación o compartir dudas de DevOps/homelab? Únete al Discord: https://umami.mvaldes.dev/q/discord",
	"Si quieres seguir el contenido fuera de Twitch: Twitter/X https://umami.mvaldes.dev/q/twitter · YouTube https://umami.mvaldes.dev/q/youtube · Discord https://umami.mvaldes.dev/q/discord",
}

func socialRemindersEnabled() bool {
	return strings.EqualFold(os.Getenv(socialReminderEnabledEnv), "true")
}

func socialReminderInterval() time.Duration {
	value := strings.TrimSpace(os.Getenv(socialReminderIntervalEnv))
	if value == "" {
		return defaultSocialReminderInterval
	}

	interval, err := time.ParseDuration(value)
	if err != nil || interval < minSocialReminderInterval {
		return defaultSocialReminderInterval
	}
	return interval
}

func socialReminderMessages() []string {
	value := strings.TrimSpace(os.Getenv(socialReminderMessagesEnv))
	if value == "" {
		return defaultSocialReminderMessages
	}

	parts := strings.Split(value, "|")
	messages := make([]string, 0, len(parts))
	for _, part := range parts {
		message := strings.TrimSpace(part)
		if message != "" {
			messages = append(messages, message)
		}
	}
	if len(messages) == 0 {
		return defaultSocialReminderMessages
	}
	return messages
}

func (rt *Router) startSocialReminders() {
	if !socialRemindersEnabled() {
		rt.Log.Info("Social reminders disabled")
		return
	}

	messages := socialReminderMessages()
	interval := socialReminderInterval()

	rt.reminderMu.Lock()
	if rt.reminderCancel != nil {
		rt.reminderMu.Unlock()
		rt.Log.Info("Social reminders already running")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	rt.reminderCancel = cancel
	rt.reminderMu.Unlock()

	rt.Log.Info("Social reminders started")
	go rt.runSocialReminders(ctx, interval, messages)
}

func (rt *Router) runSocialReminders(ctx context.Context, interval time.Duration, messages []string) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	idx := 0
	for {
		select {
		case <-ctx.Done():
			rt.Log.Info("Social reminders stopped")
			return
		case <-ticker.C:
			message := messages[idx%len(messages)]
			idx++

			sendCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
			if err := rt.Actions.SendMessage(sendCtx, message); err != nil {
				rt.Log.Error("Failed to send social reminder", err)
			}
			cancel()
		}
	}
}

func (rt *Router) stopSocialReminders() {
	rt.reminderMu.Lock()
	cancel := rt.reminderCancel
	rt.reminderCancel = nil
	rt.reminderMu.Unlock()

	if cancel == nil {
		return
	}
	cancel()
}
