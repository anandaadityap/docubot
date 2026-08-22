package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/supernand/docubot/backend/internal/models"
)

// EmbeddedChunk is a chunk plus its parent document filename (for retrieval).
type EmbeddedChunk struct {
	ID         int64
	DocumentID int64
	Filename   string
	Position   int
	Content    string
	TokenCount int
	Embedding  []float32
	EmbedModel string
	EmbedDim   int
}

// ChunkRepository persists document chunks and embeddings.
type ChunkRepository interface {
	ReplaceForDocument(ctx context.Context, documentID int64, chunks []models.Chunk) error
	ListByDocument(ctx context.Context, documentID int64) ([]models.Chunk, error)
	ListReadyWithEmbeddingsForUser(ctx context.Context, userID int64) ([]EmbeddedChunk, error)
	DeleteByDocument(ctx context.Context, documentID int64) error
}

// ChunkRepo is the SQLite implementation of ChunkRepository.
type ChunkRepo struct {
	db *sql.DB
}

// NewChunkRepo returns a SQLite-backed ChunkRepository.
func NewChunkRepo(db *sql.DB) *ChunkRepo {
	return &ChunkRepo{db: db}
}

// ReplaceForDocument deletes existing chunks and inserts the new set in one transaction.
func (r *ChunkRepo) ReplaceForDocument(ctx context.Context, documentID int64, chunks []models.Chunk) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM chunks WHERE document_id = ?`, documentID); err != nil {
		return fmt.Errorf("delete chunks: %w", err)
	}

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO chunks (document_id, position, content, token_count, embedding, embed_model, embed_dim)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return fmt.Errorf("prepare insert: %w", err)
	}
	defer stmt.Close()

	for _, c := range chunks {
		embJSON, err := json.Marshal(c.Embedding)
		if err != nil {
			return fmt.Errorf("marshal embedding: %w", err)
		}
		if _, err := stmt.ExecContext(ctx, documentID, c.Position, c.Content, c.TokenCount, string(embJSON), c.EmbedModel, c.EmbedDim); err != nil {
			return fmt.Errorf("insert chunk: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// ListByDocument returns chunks ordered by position (content only; embeddings omitted for preview).
func (r *ChunkRepo) ListByDocument(ctx context.Context, documentID int64) ([]models.Chunk, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, document_id, position, content, token_count
		 FROM chunks WHERE document_id = ? ORDER BY position ASC`,
		documentID,
	)
	if err != nil {
		return nil, fmt.Errorf("list chunks: %w", err)
	}
	defer rows.Close()

	var out []models.Chunk
	for rows.Next() {
		var c models.Chunk
		if err := rows.Scan(&c.ID, &c.DocumentID, &c.Position, &c.Content, &c.TokenCount); err != nil {
			return nil, fmt.Errorf("scan chunk: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list chunks rows: %w", err)
	}
	if out == nil {
		out = []models.Chunk{}
	}
	return out, nil
}

// ListReadyWithEmbeddingsForUser returns all chunks of ready documents for a user.
func (r *ChunkRepo) ListReadyWithEmbeddingsForUser(ctx context.Context, userID int64) ([]EmbeddedChunk, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT c.id, c.document_id, d.filename, c.position, c.content, c.token_count, c.embedding,
		       COALESCE(c.embed_model, ''), COALESCE(c.embed_dim, 0)
		FROM chunks c
		INNER JOIN documents d ON d.id = c.document_id
		WHERE d.user_id = ? AND d.status = ?
		ORDER BY c.document_id, c.position`,
		userID, models.DocumentStatusReady,
	)
	if err != nil {
		return nil, fmt.Errorf("list ready chunks: %w", err)
	}
	defer rows.Close()

	var out []EmbeddedChunk
	for rows.Next() {
		var c EmbeddedChunk
		var embJSON string
		if err := rows.Scan(&c.ID, &c.DocumentID, &c.Filename, &c.Position, &c.Content, &c.TokenCount, &embJSON, &c.EmbedModel, &c.EmbedDim); err != nil {
			return nil, fmt.Errorf("scan embedded chunk: %w", err)
		}
		if err := json.Unmarshal([]byte(embJSON), &c.Embedding); err != nil {
			return nil, fmt.Errorf("unmarshal embedding for chunk %d: %w", c.ID, err)
		}
		if c.EmbedDim == 0 {
			c.EmbedDim = len(c.Embedding)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list ready chunks rows: %w", err)
	}
	if out == nil {
		out = []EmbeddedChunk{}
	}
	return out, nil
}

// DeleteByDocument removes all chunks for a document.
func (r *ChunkRepo) DeleteByDocument(ctx context.Context, documentID int64) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM chunks WHERE document_id = ?`, documentID); err != nil {
		return fmt.Errorf("delete chunks: %w", err)
	}
	return nil
}
