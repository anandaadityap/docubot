package models

import "time"

const (
	ChannelPublic     = "public"
	ChannelPlayground = "playground"
)

// Bot is the public identity of one admin (1 user = 1 bot).
type Bot struct {
	ID             int64     `json:"id"`
	UserID         int64     `json:"-"`
	Slug           string    `json:"slug"`
	Name           string    `json:"name"`
	WelcomeMessage string    `json:"welcome_message"`
	Active         bool      `json:"active"`
	CreatedAt      time.Time `json:"created_at,omitempty"`
	UpdatedAt      time.Time `json:"updated_at,omitempty"`
}

// PublicBot is the subset exposed at GET /api/v1/bots/:slug.
type PublicBot struct {
	Slug           string `json:"slug"`
	BotName        string `json:"bot_name"`
	WelcomeMessage string `json:"welcome_message"`
	BotActive      bool   `json:"bot_active"`
	Configured     bool   `json:"configured"`
	HasReadyKB     bool   `json:"has_ready_kb"`
}

// DemoBot is the landing-page pointer (GET /api/v1/demo).
type DemoBot struct {
	Slug       string `json:"slug,omitempty"`
	BotName    string `json:"bot_name,omitempty"`
	HasReadyKB bool   `json:"has_ready_kb"`
	Configured bool   `json:"configured"`
}

// AdminBot is GET/PUT /api/v1/admin/bot.
type AdminBot struct {
	Slug           string `json:"slug"`
	Name           string `json:"name"`
	WelcomeMessage string `json:"welcome_message"`
	Active         bool   `json:"active"`
	PublicPath     string `json:"public_path"`
	EmbedPath      string `json:"embed_path"`
}
