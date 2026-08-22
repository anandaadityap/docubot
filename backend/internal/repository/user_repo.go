package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/supernand/docubot/backend/internal/models"
)

// UserRepository persists admin users.
type UserRepository interface {
	Create(ctx context.Context, email, passwordHash, name string) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByID(ctx context.Context, id int64) (*models.User, error)
	First(ctx context.Context) (*models.User, error)
}

// UserRepo is the SQLite implementation of UserRepository.
type UserRepo struct {
	db *sql.DB
}

// NewUserRepo returns a SQLite-backed UserRepository.
func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

// Create inserts a user and a default settings row in one transaction.
func (r *UserRepo) Create(ctx context.Context, email, passwordHash, name string) (*models.User, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx,
		`INSERT INTO users (email, password_hash, name) VALUES (?, ?, ?)`,
		email, passwordHash, name,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEmailTaken
		}
		return nil, fmt.Errorf("insert user: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("last insert id: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `INSERT INTO settings (user_id) VALUES (?)`, id); err != nil {
		return nil, fmt.Errorf("insert settings: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return r.GetByID(ctx, id)
}

// GetByEmail loads a user by email (exact match; caller should normalize).
func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, name, created_at FROM users WHERE email = ?`,
		email,
	)
	return scanUser(row)
}

// GetByID loads a user by primary key.
func (r *UserRepo) GetByID(ctx context.Context, id int64) (*models.User, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, name, created_at FROM users WHERE id = ?`,
		id,
	)
	return scanUser(row)
}

// First returns the earliest registered admin (single-tenant owner).
func (r *UserRepo) First(ctx context.Context) (*models.User, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, name, created_at FROM users ORDER BY id ASC LIMIT 1`,
	)
	return scanUser(row)
}

func scanUser(row *sql.Row) (*models.User, error) {
	var u models.User
	var createdAt string
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan user: %w", err)
	}
	u.CreatedAt = parseSQLiteTime(createdAt)
	return &u, nil
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique") || strings.Contains(msg, "constraint failed")
}
