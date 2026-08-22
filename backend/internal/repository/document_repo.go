package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/supernand/docubot/backend/internal/models"
)

// DocumentRepository persists knowledge-base documents.
type DocumentRepository interface {
	Create(ctx context.Context, userID int64, filename, fileType string, sizeBytes int64) (*models.Document, error)
	GetByID(ctx context.Context, id int64) (*models.Document, error)
	GetByIDForUser(ctx context.Context, id, userID int64) (*models.Document, error)
	ListByUser(ctx context.Context, userID int64) ([]models.Document, error)
	UpdateStatus(ctx context.Context, id int64, status string) error
	SetReady(ctx context.Context, id int64, chunkCount int) error
	SetFailed(ctx context.Context, id int64, errMsg string) error
	DeleteForUser(ctx context.Context, id, userID int64) error
}

// DocumentRepo is the SQLite implementation of DocumentRepository.
type DocumentRepo struct {
	db *sql.DB
}

// NewDocumentRepo returns a SQLite-backed DocumentRepository.
func NewDocumentRepo(db *sql.DB) *DocumentRepo {
	return &DocumentRepo{db: db}
}

// Create inserts a document in pending status.
func (r *DocumentRepo) Create(ctx context.Context, userID int64, filename, fileType string, sizeBytes int64) (*models.Document, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO documents (user_id, filename, file_type, size_bytes, status)
		 VALUES (?, ?, ?, ?, ?)`,
		userID, filename, fileType, sizeBytes, models.DocumentStatusPending,
	)
	if err != nil {
		return nil, fmt.Errorf("insert document: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("last insert id: %w", err)
	}
	return r.GetByID(ctx, id)
}

// GetByID loads a document by primary key (any owner).
func (r *DocumentRepo) GetByID(ctx context.Context, id int64) (*models.Document, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, filename, file_type, size_bytes, status, COALESCE(error_msg, ''),
		        chunk_count, created_at, updated_at
		 FROM documents WHERE id = ?`,
		id,
	)
	return scanDocument(row)
}

// GetByIDForUser loads a document owned by userID.
func (r *DocumentRepo) GetByIDForUser(ctx context.Context, id, userID int64) (*models.Document, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, filename, file_type, size_bytes, status, COALESCE(error_msg, ''),
		        chunk_count, created_at, updated_at
		 FROM documents WHERE id = ? AND user_id = ?`,
		id, userID,
	)
	return scanDocument(row)
}

// ListByUser returns all documents for a user, newest first.
func (r *DocumentRepo) ListByUser(ctx context.Context, userID int64) ([]models.Document, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, filename, file_type, size_bytes, status, COALESCE(error_msg, ''),
		        chunk_count, created_at, updated_at
		 FROM documents WHERE user_id = ? ORDER BY id DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}
	defer rows.Close()

	var out []models.Document
	for rows.Next() {
		d, err := scanDocumentRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list documents rows: %w", err)
	}
	if out == nil {
		out = []models.Document{}
	}
	return out, nil
}

// UpdateStatus sets status and clears error_msg; bumps updated_at.
func (r *DocumentRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE documents SET status = ?, error_msg = NULL, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		status, id,
	)
	if err != nil {
		return fmt.Errorf("update status: %w", err)
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

// SetReady marks a document ready with chunk_count.
func (r *DocumentRepo) SetReady(ctx context.Context, id int64, chunkCount int) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE documents SET status = ?, chunk_count = ?, error_msg = NULL, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		models.DocumentStatusReady, chunkCount, id,
	)
	if err != nil {
		return fmt.Errorf("set ready: %w", err)
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

// SetFailed marks a document failed with an error message.
func (r *DocumentRepo) SetFailed(ctx context.Context, id int64, errMsg string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE documents SET status = ?, error_msg = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		models.DocumentStatusFailed, errMsg, id,
	)
	if err != nil {
		return fmt.Errorf("set failed: %w", err)
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

// DeleteForUser deletes a document owned by userID (chunks cascade via FK).
func (r *DocumentRepo) DeleteForUser(ctx context.Context, id, userID int64) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM documents WHERE id = ? AND user_id = ?`,
		id, userID,
	)
	if err != nil {
		return fmt.Errorf("delete document: %w", err)
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

func scanDocument(row *sql.Row) (*models.Document, error) {
	var d models.Document
	var createdAt, updatedAt string
	err := row.Scan(
		&d.ID, &d.UserID, &d.Filename, &d.FileType, &d.SizeBytes, &d.Status, &d.ErrorMsg,
		&d.ChunkCount, &createdAt, &updatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan document: %w", err)
	}
	d.CreatedAt = parseSQLiteTime(createdAt)
	d.UpdatedAt = parseSQLiteTime(updatedAt)
	return &d, nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanDocumentRow(row scannable) (*models.Document, error) {
	var d models.Document
	var createdAt, updatedAt string
	err := row.Scan(
		&d.ID, &d.UserID, &d.Filename, &d.FileType, &d.SizeBytes, &d.Status, &d.ErrorMsg,
		&d.ChunkCount, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan document: %w", err)
	}
	d.CreatedAt = parseSQLiteTime(createdAt)
	d.UpdatedAt = parseSQLiteTime(updatedAt)
	return &d, nil
}
