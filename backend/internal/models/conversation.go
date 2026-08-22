package models

import "time"

// Conversation is a public chat session owned by the bot's admin user.
type Conversation struct {
	ID           int64     `json:"id"`
	PublicID     string    `json:"public_id,omitempty"`
	UserID       int64     `json:"-"`
	Title        string    `json:"title"`
	MessageCount int       `json:"message_count,omitempty"`
	CreatedAt    time.Time `json:"created_at,omitempty"`
	UpdatedAt    time.Time `json:"updated_at,omitempty"`
}
