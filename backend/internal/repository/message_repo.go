package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/supernand/docubot/backend/internal/models"
)

// MessageRepository persists chat messages.
type MessageRepository interface {
	Create(ctx context.Context, m *models.Message) (*models.Message, error)
	ListByConversation(ctx context.Context, conversationID int64) ([]models.Message, error)
}

// MessageRepo is the SQLite implementation.
type MessageRepo struct {
	db *sql.DB
}

// NewMessageRepo returns a SQLite-backed MessageRepository.
func NewMessageRepo(db *sql.DB) *MessageRepo {
	return &MessageRepo{db: db}
}

// Create inserts a message and returns it with ID.
func (r *MessageRepo) Create(ctx context.Context, m *models.Message) (*models.Message, error) {
	var sources any
	if len(m.Sources) > 0 {
		b, err := json.Marshal(m.Sources)
		if err != nil {
			return nil, fmt.Errorf("marshal sources: %w", err)
		}
		sources = string(b)
	}

	res, err := r.db.ExecContext(ctx, `
		INSERT INTO messages (conversation_id, role, content, sources, latency_ms, token_usage)
		VALUES (?, ?, ?, ?, ?, ?)`,
		m.ConversationID, m.Role, m.Content, sources, m.LatencyMS, m.TokenUsage,
	)
	if err != nil {
		return nil, fmt.Errorf("insert message: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("last insert id: %w", err)
	}

	if _, err := r.db.ExecContext(ctx,
		`UPDATE conversations SET updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		m.ConversationID,
	); err != nil {
		return nil, fmt.Errorf("touch conversation: %w", err)
	}

	row := r.db.QueryRowContext(ctx, `
		SELECT id, conversation_id, role, content, sources, latency_ms, token_usage, created_at
		FROM messages WHERE id = ?`, id)
	return scanMessage(row)
}

// ListByConversation returns messages in chronological order.
func (r *MessageRepo) ListByConversation(ctx context.Context, conversationID int64) ([]models.Message, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, conversation_id, role, content, sources, latency_ms, token_usage, created_at
		FROM messages WHERE conversation_id = ? ORDER BY id ASC`,
		conversationID,
	)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	defer rows.Close()

	var out []models.Message
	for rows.Next() {
		m, err := scanMessageRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list messages rows: %w", err)
	}
	if out == nil {
		out = []models.Message{}
	}
	return out, nil
}

func scanMessage(row *sql.Row) (*models.Message, error) {
	return scanMessageRow(row)
}

func scanMessageRow(row scannable) (*models.Message, error) {
	var m models.Message
	var sources sql.NullString
	var latency, tokens sql.NullInt64
	var createdAt string
	err := row.Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content, &sources, &latency, &tokens, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("scan message: %w", err)
	}
	if sources.Valid && sources.String != "" {
		if err := json.Unmarshal([]byte(sources.String), &m.Sources); err != nil {
			return nil, fmt.Errorf("unmarshal sources: %w", err)
		}
	}
	if latency.Valid {
		v := int(latency.Int64)
		m.LatencyMS = &v
	}
	if tokens.Valid {
		v := int(tokens.Int64)
		m.TokenUsage = &v
	}
	m.CreatedAt = parseSQLiteTime(createdAt)
	return &m, nil
}
