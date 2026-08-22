package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/supernand/docubot/backend/internal/ai"
	"github.com/supernand/docubot/backend/internal/models"
	"github.com/supernand/docubot/backend/internal/repository"
)

const (
	maxUploadBytes  = 5 << 20 // 5 MiB
	chunkPreviewLen = 200
)

var (
	// ErrNotFound is returned when a document is missing or not owned by the user.
	ErrNotFound = errors.New("not found")
)

// DocumentService handles upload, ingest pipeline, list, delete, and reprocess.
type DocumentService struct {
	docs      repository.DocumentRepository
	chunks    repository.ChunkRepository
	embedder  ai.Embedder
	uploadDir string
}

// NewDocumentService constructs a DocumentService.
func NewDocumentService(
	docs repository.DocumentRepository,
	chunks repository.ChunkRepository,
	embedder ai.Embedder,
	uploadDir string,
) *DocumentService {
	return &DocumentService{
		docs:      docs,
		chunks:    chunks,
		embedder:  embedder,
		uploadDir: uploadDir,
	}
}

// Upload validates the file, persists it, and inserts a pending document row.
// Caller should invoke Process asynchronously after a successful Upload.
func (s *DocumentService) Upload(ctx context.Context, userID int64, originalName string, size int64, r io.Reader) (*models.Document, error) {
	fileType, err := validateUpload(originalName, size)
	if err != nil {
		return nil, err
	}
	safeName := sanitizeFilename(originalName)
	if safeName == "" {
		return nil, fmt.Errorf("%w: invalid filename", ErrValidation)
	}

	doc, err := s.docs.Create(ctx, userID, safeName, fileType, size)
	if err != nil {
		return nil, fmt.Errorf("create document: %w", err)
	}

	path := s.storedPath(doc.ID, safeName)
	if err := writeFileLimited(path, r, maxUploadBytes); err != nil {
		_ = s.docs.DeleteForUser(ctx, doc.ID, userID)
		if errors.Is(err, errTooLarge) {
			return nil, fmt.Errorf("%w: file exceeds 5 MB", ErrValidation)
		}
		return nil, fmt.Errorf("save file: %w", err)
	}

	// Refresh in case size from header was 0 (chunked); keep reported size from Create.
	return doc, nil
}

// Process runs extract → chunk → embed → ready|failed for a document by ID.
func (s *DocumentService) Process(ctx context.Context, documentID int64) error {
	doc, err := s.docs.GetByID(ctx, documentID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	return s.processDoc(ctx, doc)
}

func (s *DocumentService) processDoc(ctx context.Context, doc *models.Document) error {
	if err := s.docs.UpdateStatus(ctx, doc.ID, models.DocumentStatusProcessing); err != nil {
		return err
	}

	fail := func(msg string) error {
		_ = s.docs.SetFailed(ctx, doc.ID, msg)
		return fmt.Errorf("%s", msg)
	}

	path := s.storedPath(doc.ID, doc.Filename)
	raw, err := os.ReadFile(path)
	if err != nil {
		return fail(fmt.Sprintf("read file: %v", err))
	}
	text := strings.TrimSpace(string(raw))
	if text == "" {
		return fail("document is empty")
	}

	textChunks := ai.Chunk(text)
	if len(textChunks) == 0 {
		return fail("document is empty")
	}

	texts := make([]string, len(textChunks))
	for i, c := range textChunks {
		texts[i] = c.Content
	}

	vectors, err := s.embedder.Embed(ctx, texts)
	if err != nil {
		return fail(fmt.Sprintf("embed: %v", err))
	}
	if len(vectors) != len(textChunks) {
		return fail(fmt.Sprintf("embed: got %d vectors for %d chunks", len(vectors), len(textChunks)))
	}

	chunks := make([]models.Chunk, len(textChunks))
	for i, tc := range textChunks {
		chunks[i] = models.Chunk{
			DocumentID: doc.ID,
			Position:   tc.Position,
			Content:    tc.Content,
			TokenCount: tc.TokenCount,
			Embedding:  vectors[i],
		}
	}

	if err := s.chunks.ReplaceForDocument(ctx, doc.ID, chunks); err != nil {
		return fail(fmt.Sprintf("save chunks: %v", err))
	}
	if err := s.docs.SetReady(ctx, doc.ID, len(chunks)); err != nil {
		return fail(fmt.Sprintf("set ready: %v", err))
	}
	return nil
}

// List returns all documents for a user.
func (s *DocumentService) List(ctx context.Context, userID int64) ([]models.Document, error) {
	return s.docs.ListByUser(ctx, userID)
}

// DocumentDetail is a document plus preview chunks (truncated content, no embeddings).
type DocumentDetail struct {
	models.Document
	Chunks []ChunkPreview `json:"chunks"`
}

// ChunkPreview is a truncated chunk for admin preview.
type ChunkPreview struct {
	Position   int    `json:"position"`
	TokenCount int    `json:"token_count"`
	Content    string `json:"content"`
}

// Get returns a document owned by userID with chunk previews.
func (s *DocumentService) Get(ctx context.Context, userID, documentID int64) (*DocumentDetail, error) {
	doc, err := s.docs.GetByIDForUser(ctx, documentID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	list, err := s.chunks.ListByDocument(ctx, documentID)
	if err != nil {
		return nil, err
	}
	previews := make([]ChunkPreview, len(list))
	for i, c := range list {
		previews[i] = ChunkPreview{
			Position:   c.Position,
			TokenCount: c.TokenCount,
			Content:    truncateRunes(c.Content, chunkPreviewLen),
		}
	}
	return &DocumentDetail{Document: *doc, Chunks: previews}, nil
}

// Delete removes a document owned by userID, its chunks (FK cascade), and the file on disk.
func (s *DocumentService) Delete(ctx context.Context, userID, documentID int64) error {
	doc, err := s.docs.GetByIDForUser(ctx, documentID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	path := s.storedPath(doc.ID, doc.Filename)
	if err := s.docs.DeleteForUser(ctx, documentID, userID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	_ = os.Remove(path)
	return nil
}

// Reprocess clears chunks and re-runs the ingest pipeline for an owned document.
func (s *DocumentService) Reprocess(ctx context.Context, userID, documentID int64) (*models.Document, error) {
	doc, err := s.docs.GetByIDForUser(ctx, documentID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if err := s.chunks.DeleteByDocument(ctx, documentID); err != nil {
		return nil, err
	}
	if err := s.docs.UpdateStatus(ctx, documentID, models.DocumentStatusProcessing); err != nil {
		return nil, err
	}
	doc.Status = models.DocumentStatusProcessing
	doc.ChunkCount = 0
	doc.ErrorMsg = ""
	return doc, nil
}

func (s *DocumentService) storedPath(id int64, filename string) string {
	return filepath.Join(s.uploadDir, fmt.Sprintf("%d_%s", id, filename))
}

func validateUpload(originalName string, size int64) (fileType string, err error) {
	if size < 0 {
		return "", fmt.Errorf("%w: invalid file size", ErrValidation)
	}
	if size > maxUploadBytes {
		return "", fmt.Errorf("%w: file exceeds 5 MB", ErrValidation)
	}
	ext := strings.ToLower(filepath.Ext(originalName))
	switch ext {
	case ".md":
		return "md", nil
	case ".txt":
		return "txt", nil
	default:
		return "", fmt.Errorf("%w: only .md and .txt files are allowed", ErrValidation)
	}
}

func sanitizeFilename(name string) string {
	name = filepath.Base(name)
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return ""
	}
	var b strings.Builder
	for _, r := range name {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
			b.WriteRune(r)
		case r == '.' || r == '-' || r == '_' || r == ' ':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	out := strings.TrimSpace(b.String())
	if out == "" {
		return ""
	}
	// Cap length to keep paths reasonable.
	runes := []rune(out)
	if len(runes) > 180 {
		out = string(runes[:180])
	}
	return out
}

var errTooLarge = errors.New("file too large")

func writeFileLimited(path string, r io.Reader, limit int64) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	written, err := io.Copy(f, io.LimitReader(r, limit+1))
	if err != nil {
		_ = os.Remove(path)
		return err
	}
	if written > limit {
		_ = os.Remove(path)
		return errTooLarge
	}
	return nil
}

func truncateRunes(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}
