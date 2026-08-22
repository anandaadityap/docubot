package models

import "time"

const (
	RoleUser = "user"
	RoleBot  = "bot"
)

// Source is a cited knowledge-base snippet attached to a bot message.
type Source struct {
	DocID    int64   `json:"doc_id"`
	Filename string  `json:"filename"`
	Snippet  string  `json:"snippet"`
	Score    float64 `json:"score"`
}

// Message is one turn in a conversation.
type Message struct {
	ID             int64     `json:"id"`
	ConversationID int64     `json:"conversation_id"`
	Role           string    `json:"role"`
	Content        string    `json:"content"`
	Sources        []Source  `json:"sources,omitempty"`
	LatencyMS      *int      `json:"latency_ms,omitempty"`
	TokenUsage     *int      `json:"token_usage,omitempty"`
	CreatedAt      time.Time `json:"created_at,omitempty"`
}
