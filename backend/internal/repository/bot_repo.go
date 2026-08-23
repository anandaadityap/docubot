package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/supernand/docubot/backend/internal/models"
	"github.com/supernand/docubot/backend/internal/util"
)

// BotRepository persists public bot identity (1:1 with users).
type BotRepository interface {
	GetBySlug(ctx context.Context, slug string) (*models.Bot, error)
	GetByUserID(ctx context.Context, userID int64) (*models.Bot, error)
	GetOldest(ctx context.Context) (*models.Bot, error)
	Update(ctx context.Context, b *models.Bot) error
	SlugTaken(ctx context.Context, slug string, excludeUserID int64) (bool, error)
}

// BotRepo is the SQLite implementation of BotRepository.
type BotRepo struct {
	db *sql.DB
}

// NewBotRepo returns a SQLite-backed BotRepository.
func NewBotRepo(db *sql.DB) *BotRepo {
	return &BotRepo{db: db}
}

const botSelect = `
		SELECT id, user_id, slug, name, welcome_message, active, created_at, updated_at
		FROM bots`

// GetBySlug loads a bot by its public slug.
func (r *BotRepo) GetBySlug(ctx context.Context, slug string) (*models.Bot, error) {
	row := r.db.QueryRowContext(ctx, botSelect+` WHERE slug = ?`, slug)
	return scanBot(row)
}

// GetByUserID loads the bot owned by userID.
func (r *BotRepo) GetByUserID(ctx context.Context, userID int64) (*models.Bot, error) {
	row := r.db.QueryRowContext(ctx, botSelect+` WHERE user_id = ?`, userID)
	return scanBot(row)
}

// GetOldest returns the bot with the smallest user_id (landing demo pointer only).
func (r *BotRepo) GetOldest(ctx context.Context) (*models.Bot, error) {
	row := r.db.QueryRowContext(ctx, botSelect+` ORDER BY user_id ASC LIMIT 1`)
	return scanBot(row)
}

// Update replaces identity fields for a bot. Unique slug collisions return ErrSlugTaken.
func (r *BotRepo) Update(ctx context.Context, b *models.Bot) error {
	active := 0
	if b.Active {
		active = 1
	}
	res, err := r.db.ExecContext(ctx, `
		UPDATE bots SET
			slug = ?, name = ?, welcome_message = ?, active = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ?`,
		b.Slug, b.Name, b.WelcomeMessage, active, b.UserID,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrSlugTaken
		}
		return fmt.Errorf("update bot: %w", err)
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

// SlugTaken reports whether slug is used by a different user.
func (r *BotRepo) SlugTaken(ctx context.Context, slug string, excludeUserID int64) (bool, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM bots WHERE slug = ? AND user_id != ?`,
		slug, excludeUserID,
	).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("slug taken: %w", err)
	}
	return n > 0, nil
}

func scanBot(row *sql.Row) (*models.Bot, error) {
	var b models.Bot
	var active int
	var createdAt, updatedAt string
	err := row.Scan(&b.ID, &b.UserID, &b.Slug, &b.Name, &b.WelcomeMessage, &active, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan bot: %w", err)
	}
	b.Active = active != 0
	b.CreatedAt = parseSQLiteTime(createdAt)
	b.UpdatedAt = parseSQLiteTime(updatedAt)
	return &b, nil
}

type slugCounter interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func insertBotTx(ctx context.Context, tx slugCounter, userID int64, slugSeed, botName, welcome string, active bool) error {
	botName = strings.TrimSpace(botName)
	if botName == "" {
		botName = "DocuBot"
	}
	welcome = strings.TrimSpace(welcome)
	if welcome == "" {
		welcome = "Halo! Ada yang bisa saya bantu?"
	}
	slug := util.AllocateSlug(slugSeed, userID, func(s string) bool {
		var n int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM bots WHERE slug = ?`, s).Scan(&n); err != nil {
			return true
		}
		return n > 0
	})
	act := 0
	if active {
		act = 1
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO bots (user_id, slug, name, welcome_message, active)
		VALUES (?, ?, ?, ?, ?)`,
		userID, slug, botName, welcome, act,
	)
	if err != nil {
		return fmt.Errorf("insert bot: %w", err)
	}
	return nil
}
