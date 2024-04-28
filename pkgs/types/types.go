package types

import "time"

type SubscriptionType struct {
	Name    string
	Version string
	Type    string
}

type RequestHeader struct {
	Token    string
	ClientID string
}

type ChatMessage struct {
	BroadcasterID string `json:"broadcaster_id"`
	SenderID      string `json:"sender_id"`
	Message       string `json:"message"`
}

type ValidateSubscription struct {
	Data []struct {
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
	} `json:"data"`
	Total        int `json:"total"`
	MaxTotalCost int `json:"max_total_cost"`
	TotalCost    int `json:"total_cost"`
}

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
				Type      string      `json:"type"`
				Text      string      `json:"text"`
				Cheermote interface{} `json:"cheermote"`
				Emote     interface{} `json:"emote"`
				Mention   interface{} `json:"mention"`
			} `json:"fragments"`
		} `json:"message"`
		Color  string `json:"color"`
		Badges []struct {
			SetID string `json:"set_id"`
			ID    string `json:"id"`
			Info  string `json:"info"`
		} `json:"badges"`
		MessageType                 string      `json:"message_type"`
		Cheer                       interface{} `json:"cheer"`
		Reply                       interface{} `json:"reply"`
		ChannelPointsCustomRewardID interface{} `json:"channel_points_custom_reward_id"`
	} `json:"event"`
}

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
