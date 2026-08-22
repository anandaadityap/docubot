package models

import "time"

// Document statuses for the ingest pipeline.
const (
	DocumentStatusPending    = "pending"
	DocumentStatusProcessing = "processing"
	DocumentStatusReady      = "ready"
	DocumentStatusFailed     = "failed"
)

// Document is a knowledge-base file owned by an admin user.
type Document struct {
	ID         int64     `json:"id"`
	UserID     int64     `json:"-"`
	Filename   string    `json:"filename"`
	FileType   string    `json:"file_type"`
	SizeBytes  int64     `json:"size_bytes"`
	Status     string    `json:"status"`
	ErrorMsg   string    `json:"error_msg,omitempty"`
	ChunkCount int       `json:"chunk_count"`
	EmbedModel string    `json:"embed_model,omitempty"`
	EmbedDim   int       `json:"embed_dim,omitempty"`
	CreatedAt  time.Time `json:"created_at,omitempty"`
	UpdatedAt  time.Time `json:"updated_at,omitempty"`
}
