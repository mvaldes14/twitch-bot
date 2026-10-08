package routes

import (
	"reflect"
	"testing"
	"time"
)

func TestSocialRemindersEnabledOnlyWhenTrue(t *testing.T) {
	t.Setenv(socialReminderEnabledEnv, "")
	if socialRemindersEnabled() {
		t.Fatal("empty SOCIAL_REMINDER_ENABLED should disable reminders")
	}

	t.Setenv(socialReminderEnabledEnv, "false")
	if socialRemindersEnabled() {
		t.Fatal("false SOCIAL_REMINDER_ENABLED should disable reminders")
	}

	t.Setenv(socialReminderEnabledEnv, "true")
	if !socialRemindersEnabled() {
		t.Fatal("true SOCIAL_REMINDER_ENABLED should enable reminders")
	}

	t.Setenv(socialReminderEnabledEnv, "TRUE")
	if !socialRemindersEnabled() {
		t.Fatal("SOCIAL_REMINDER_ENABLED should be case-insensitive")
	}
}

func TestSocialReminderInterval(t *testing.T) {
	t.Setenv(socialReminderIntervalEnv, "45m")
	if got := socialReminderInterval(); got != 45*time.Minute {
		t.Fatalf("interval = %s, want 45m", got)
	}

	t.Setenv(socialReminderIntervalEnv, "garbage")
	if got := socialReminderInterval(); got != defaultSocialReminderInterval {
		t.Fatalf("invalid interval = %s, want default %s", got, defaultSocialReminderInterval)
	}

	t.Setenv(socialReminderIntervalEnv, "5m")
	if got := socialReminderInterval(); got != defaultSocialReminderInterval {
		t.Fatalf("too-short interval = %s, want default %s", got, defaultSocialReminderInterval)
	}
}

func TestSocialReminderMessages(t *testing.T) {
	t.Setenv(socialReminderMessagesEnv, "one | two| |three ")
	want := []string{"one", "two", "three"}
	if got := socialReminderMessages(); !reflect.DeepEqual(got, want) {
		t.Fatalf("messages = %#v, want %#v", got, want)
	}

	t.Setenv(socialReminderMessagesEnv, " | ")
	if got := socialReminderMessages(); !reflect.DeepEqual(got, defaultSocialReminderMessages) {
		t.Fatalf("blank messages should fall back to defaults")
	}
}
