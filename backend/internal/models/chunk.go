package models

// Chunk is a text slice of a document with an optional embedding vector.
type Chunk struct {
	ID         int64     `json:"id,omitempty"`
	DocumentID int64     `json:"document_id,omitempty"`
	Position   int       `json:"position"`
	Content    string    `json:"content"`
	TokenCount int       `json:"token_count"`
	Embedding  []float32 `json:"-"`
	EmbedModel string    `json:"embed_model,omitempty"`
	EmbedDim   int       `json:"embed_dim,omitempty"`
}
