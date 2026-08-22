package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/supernand/docubot/backend/internal/models"
)

// SettingsRepository persists bot settings.
type SettingsRepository interface {
	GetByUserID(ctx context.Context, userID int64) (*models.Settings, error)
	Update(ctx context.Context, s *models.Settings) error
}

// SettingsRepo is the SQLite implementation.
type SettingsRepo struct {
	db *sql.DB
}

// NewSettingsRepo returns a SQLite-backed SettingsRepository.
func NewSettingsRepo(db *sql.DB) *SettingsRepo {
	return &SettingsRepo{db: db}
}

// GetByUserID loads settings for a user.
func (r *SettingsRepo) GetByUserID(ctx context.Context, userID int64) (*models.Settings, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT user_id, bot_name, welcome_message, bot_active, temperature, max_tokens, top_k, min_score,
		       created_at, updated_at
		FROM settings WHERE user_id = ?`, userID)
	return scanSettings(row)
}

// Update replaces settings fields for user_id.
func (r *SettingsRepo) Update(ctx context.Context, s *models.Settings) error {
	active := 0
	if s.BotActive {
		active = 1
	}
	res, err := r.db.ExecContext(ctx, `
		UPDATE settings SET
			bot_name = ?, welcome_message = ?, bot_active = ?,
			temperature = ?, max_tokens = ?, top_k = ?, min_score = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ?`,
		s.BotName, s.WelcomeMessage, active, s.Temperature, s.MaxTokens, s.TopK, s.MinScore, s.UserID,
	)
	if err != nil {
		return fmt.Errorf("update settings: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func scanSettings(row *sql.Row) (*models.Settings, error) {
	var s models.Settings
	var active int
	var createdAt, updatedAt string
	err := row.Scan(
		&s.UserID, &s.BotName, &s.WelcomeMessage, &active,
		&s.Temperature, &s.MaxTokens, &s.TopK, &s.MinScore,
		&createdAt, &updatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan settings: %w", err)
	}
	s.BotActive = active != 0
	s.CreatedAt = parseSQLiteTime(createdAt)
	s.UpdatedAt = parseSQLiteTime(updatedAt)
	return &s, nil
}
