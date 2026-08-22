package models

import "time"

// Settings is per-admin bot configuration (one row per user).
type Settings struct {
	UserID         int64     `json:"-"`
	BotName        string    `json:"bot_name"`
	WelcomeMessage string    `json:"welcome_message"`
	BotActive      bool      `json:"bot_active"`
	Temperature    float64   `json:"temperature"`
	MaxTokens      int       `json:"max_tokens"`
	TopK           int       `json:"top_k"`
	MinScore       float64   `json:"min_score"`
	CreatedAt      time.Time `json:"-"`
	UpdatedAt      time.Time `json:"updated_at,omitempty"`
}

// PublicBot is the subset of settings exposed to the public chat page.
type PublicBot struct {
	BotName        string `json:"bot_name"`
	WelcomeMessage string `json:"welcome_message"`
	BotActive      bool   `json:"bot_active"`
	Configured     bool   `json:"configured"`
	HasReadyKB     bool   `json:"has_ready_kb"`
	RegisterOpen   bool   `json:"register_open"`
}
