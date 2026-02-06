// Package subscriptions handles all subscribe events on twitch
package subscriptions

import "time"

// SubscriptionType represents a Twitch subscription type
type SubscriptionType struct {
	Name    string
	Version string
	Type    string
}

// SubscriptionData represents a single subscription from Twitch
type SubscriptionData struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	Type      string `json:"type"`
	Version   string `json:"version"`
	Condition struct {
		BroadcasterUserID string `json:"broadcaster_user_id"`
		UserID            string `json:"user_id"`
	} `json:"condition"`
	CreatedAt time.Time `json:"created_at"`
	Transport struct {
		Method   string `json:"method"`
		Callback string `json:"callback"`
	} `json:"transport"`
	Cost int `json:"cost"`
}

// ValidateSubscription represents the response from Twitch subscription validation
type ValidateSubscription struct {
	Data         []SubscriptionData `json:"data"`
	Total        int                `json:"total"`
	MaxTotalCost int                `json:"max_total_cost"`
	TotalCost    int                `json:"total_cost"`
}

// EventLog represents a log entry for an event
type EventLog struct {
	Username  string    `json:"username"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"@timestamp"`
	Type      string    `json:"type"`
}

// ChatMessage represents a chat message event
type ChatMessage struct {
	BroadcasterID string `json:"broadcaster_id"`
	SenderID      string `json:"sender_id"`
	Message       string `json:"message"`
}

// SubscribeEvent represents a subscription event
type SubscribeEvent struct {
	Challenge    string `json:"challenge"`
	Subscription struct {
		ID        string `json:"id"`
		Status    string `json:"status"`
		Type      string `json:"type"`
		Version   string `json:"version"`
		Cost      int    `json:"cost"`
		Condition struct {
			BroadcasterUserID string `json:"broadcaster_user_id"`
		} `json:"condition"`
		Transport struct {
			Method   string `json:"method"`
			Callback string `json:"callback"`
		} `json:"transport"`
		CreatedAt time.Time `json:"created_at"`
	} `json:"subscription"`
}

// ChatMessageEvent represents a chat message event from Twitch
type ChatMessageEvent struct {
	Challenge    string `json:"challenge"`
	Subscription struct {
		ID        string `json:"id"`
		Status    string `json:"status"`
		Type      string `json:"type"`
		Version   string `json:"version"`
		Condition struct {
			BroadcasterUserID string `json:"broadcaster_user_id"`
			UserID            string `json:"user_id"`
		} `json:"condition"`
		Transport struct {
			Method    string `json:"method"`
			SessionID string `json:"session_id"`
		} `json:"transport"`
		CreatedAt time.Time `json:"created_at"`
		Cost      int       `json:"cost"`
	} `json:"subscription"`
	Event struct {
		BroadcasterUserID    string `json:"broadcaster_user_id"`
		BroadcasterUserLogin string `json:"broadcaster_user_login"`
		BroadcasterUserName  string `json:"broadcaster_user_name"`
		ChatterUserID        string `json:"chatter_user_id"`
		ChatterUserLogin     string `json:"chatter_user_login"`
		ChatterUserName      string `json:"chatter_user_name"`
		MessageID            string `json:"message_id"`
		Message              struct {
			Text      string `json:"text"`
			Fragments []struct {
				Type      string `json:"type"`
				Text      string `json:"text"`
				Cheermote struct {
					Prefix string `json:"prefix"`
					Bits   int    `json:"bits"`
					Tier   int    `json:"tier"`
				} `json:"cheermote"`
			} `json:"fragments"`
		} `json:"message"`
		Color       string    `json:"color"`
		Badges []struct {
			SetID string `json:"set_id"`
			ID    string `json:"id"`
			Info  string `json:"info"`
		} `json:"badges"`
		MessageType string    `json:"message_type"`
		SentAt      time.Time `json:"sent_at"`
	} `json:"event"`
}

// FollowEvent represents a follow event from Twitch
type FollowEvent struct {
	Subscription struct {
		ID        string `json:"id"`
		Type      string `json:"type"`
		Version   string `json:"version"`
		Status    string `json:"status"`
		Cost      int    `json:"cost"`
		Condition struct {
			BroadcasterUserID string `json:"broadcaster_user_id"`
			ModeratorUserID   string `json:"moderator_user_id"`
		} `json:"condition"`
		Transport struct {
			Method   string `json:"method"`
			Callback string `json:"callback"`
		} `json:"transport"`
		CreatedAt time.Time `json:"created_at"`
	} `json:"subscription"`
	Event struct {
		UserID               string    `json:"user_id"`
		UserLogin            string    `json:"user_login"`
		UserName             string    `json:"user_name"`
		BroadcasterUserID    string    `json:"broadcaster_user_id"`
		BroadcasterUserLogin string    `json:"broadcaster_user_login"`
		BroadcasterUserName  string    `json:"broadcaster_user_name"`
		FollowedAt           time.Time `json:"followed_at"`
	} `json:"event"`
}

// CheerEvent represents a cheer event from Twitch
type CheerEvent struct {
	Subscription struct {
		ID        string `json:"id"`
		Type      string `json:"type"`
		Version   string `json:"version"`
		Status    string `json:"status"`
		Cost      int    `json:"cost"`
		Condition struct {
			BroadcasterUserID string `json:"broadcaster_user_id"`
		} `json:"condition"`
		Transport struct {
			Method   string `json:"method"`
			Callback string `json:"callback"`
		} `json:"transport"`
		CreatedAt time.Time `json:"created_at"`
	} `json:"subscription"`
	Event struct {
		IsAnonymous          bool   `json:"is_anonymous"`
		UserID               string `json:"user_id"`
		UserLogin            string `json:"user_login"`
		UserName             string `json:"user_name"`
		BroadcasterUserID    string `json:"broadcaster_user_id"`
		BroadcasterUserLogin string `json:"broadcaster_user_login"`
		BroadcasterUserName  string `json:"broadcaster_user_name"`
		Message              string `json:"message"`
		Bits                 int    `json:"bits"`
	} `json:"event"`
}

// RewardEvent represents a reward redemption event from Twitch
type RewardEvent struct {
	Subscription struct {
		ID        string `json:"id"`
		Type      string `json:"type"`
		Version   string `json:"version"`
		Status    string `json:"status"`
		Cost      int    `json:"cost"`
		Condition struct {
			BroadcasterUserID string `json:"broadcaster_user_id"`
			RewardID          string `json:"reward_id"`
		} `json:"condition"`
		Transport struct {
			Method   string `json:"method"`
			Callback string `json:"callback"`
		} `json:"transport"`
		CreatedAt time.Time `json:"created_at"`
	} `json:"subscription"`
	Event struct {
		ID                   string `json:"id"`
		BroadcasterUserID    string `json:"broadcaster_user_id"`
		BroadcasterUserLogin string `json:"broadcaster_user_login"`
		BroadcasterUserName  string `json:"broadcaster_user_name"`
		UserID               string `json:"user_id"`
		UserLogin            string `json:"user_login"`
		UserName             string `json:"user_name"`
		UserInput            string `json:"user_input"`
		Status               string `json:"status"`
		Reward               struct {
			ID     string `json:"id"`
			Title  string `json:"title"`
			Cost   int    `json:"cost"`
			Prompt string `json:"prompt"`
		} `json:"reward"`
		RedeemedAt time.Time `json:"redeemed_at"`
	} `json:"event"`
}

// SubscriptionEvent a response event from Twitch
type SubscriptionEvent struct {
	Subscription struct {
		ID        string `json:"id"`
		Type      string `json:"type"`
		Version   string `json:"version"`
		Status    string `json:"status"`
		Cost      int    `json:"cost"`
		Condition struct {
			BroadcasterUserID string `json:"broadcaster_user_id"`
		} `json:"condition"`
		Transport struct {
			Method   string `json:"method"`
			Callback string `json:"callback"`
		} `json:"transport"`
		CreatedAt time.Time `json:"created_at"`
	} `json:"subscription"`
	Event struct {
		UserID               string `json:"user_id"`
		UserLogin            string `json:"user_login"`
		UserName             string `json:"user_name"`
		BroadcasterUserID    string `json:"broadcaster_user_id"`
		BroadcasterUserLogin string `json:"broadcaster_user_login"`
		BroadcasterUserName  string `json:"broadcaster_user_name"`
		Tier                 string `json:"tier"`
		IsGift               bool   `json:"is_gift"`
	} `json:"event"`
}
