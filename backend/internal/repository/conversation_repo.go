package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/supernand/docubot/backend/internal/models"
)

// ConversationRepository persists chat sessions.
type ConversationRepository interface {
	Create(ctx context.Context, userID int64, title string) (*models.Conversation, error)
	GetByIDForUser(ctx context.Context, id, userID int64) (*models.Conversation, error)
	ListByUser(ctx context.Context, userID int64, page, limit int) ([]models.Conversation, int, error)
	TouchTitle(ctx context.Context, id int64, title string) error
}

// ConversationRepo is the SQLite implementation.
type ConversationRepo struct {
	db *sql.DB
}

// NewConversationRepo returns a SQLite-backed ConversationRepository.
func NewConversationRepo(db *sql.DB) *ConversationRepo {
	return &ConversationRepo{db: db}
}

// Create inserts a conversation.
func (r *ConversationRepo) Create(ctx context.Context, userID int64, title string) (*models.Conversation, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO conversations (user_id, title) VALUES (?, ?)`,
		userID, title,
	)
	if err != nil {
		return nil, fmt.Errorf("insert conversation: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("last insert id: %w", err)
	}
	return r.GetByIDForUser(ctx, id, userID)
}

// GetByIDForUser loads a conversation owned by userID.
func (r *ConversationRepo) GetByIDForUser(ctx context.Context, id, userID int64) (*models.Conversation, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT c.id, c.user_id, c.title, c.created_at, c.updated_at,
		       (SELECT COUNT(*) FROM messages m WHERE m.conversation_id = c.id)
		FROM conversations c
		WHERE c.id = ? AND c.user_id = ?`,
		id, userID,
	)
	return scanConversation(row)
}

// ListByUser returns paginated conversations (newest first) and the total count.
func (r *ConversationRepo) ListByUser(ctx context.Context, userID int64, page, limit int) ([]models.Conversation, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	var total int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM conversations WHERE user_id = ?`, userID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count conversations: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT c.id, c.user_id, c.title, c.created_at, c.updated_at,
		       (SELECT COUNT(*) FROM messages m WHERE m.conversation_id = c.id)
		FROM conversations c
		WHERE c.user_id = ?
		ORDER BY c.id DESC
		LIMIT ? OFFSET ?`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list conversations: %w", err)
	}
	defer rows.Close()

	var out []models.Conversation
	for rows.Next() {
		c, err := scanConversationRow(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("list conversations rows: %w", err)
	}
	if out == nil {
		out = []models.Conversation{}
	}
	return out, total, nil
}

// TouchTitle updates title and updated_at.
func (r *ConversationRepo) TouchTitle(ctx context.Context, id int64, title string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE conversations SET title = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		title, id,
	)
	if err != nil {
		return fmt.Errorf("touch conversation: %w", err)
	}
	return nil
}

func scanConversation(row *sql.Row) (*models.Conversation, error) {
	c, err := scanConversationRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return c, err
}

func scanConversationRow(row scannable) (*models.Conversation, error) {
	var c models.Conversation
	var createdAt, updatedAt string
	err := row.Scan(&c.ID, &c.UserID, &c.Title, &createdAt, &updatedAt, &c.MessageCount)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan conversation: %w", err)
	}
	c.CreatedAt = parseSQLiteTime(createdAt)
	c.UpdatedAt = parseSQLiteTime(updatedAt)
	return &c, nil
}
