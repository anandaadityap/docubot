package service_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/supernand/docubot/backend/internal/ai"
	"github.com/supernand/docubot/backend/internal/database"
	"github.com/supernand/docubot/backend/internal/models"
	"github.com/supernand/docubot/backend/internal/repository"
	"github.com/supernand/docubot/backend/internal/service"
)

func newTestDocService(t *testing.T, embedder ai.Embedder) (*service.DocumentService, *service.AuthService, string) {
	t.Helper()
	dir := t.TempDir()
	uploadDir := filepath.Join(dir, "uploads")
	db, err := database.Open(filepath.Join(dir, "test.db"), uploadDir)
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if embedder == nil {
		embedder = ai.NewStubEmbedder()
	}
	docs := service.NewDocumentService(
		repository.NewDocumentRepo(db),
		repository.NewChunkRepo(db),
		embedder,
		uploadDir,
	)
	auth := service.NewAuthService(repository.NewUserRepo(db), "test-secret")
	return docs, auth, uploadDir
}

func registerUser(t *testing.T, auth *service.AuthService, email string) int64 {
	t.Helper()
	u, err := auth.Register(context.Background(), "Test", email, "secret123")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	return u.ID
}

func storedFilename(id int64, name string) string {
	return fmt.Sprintf("%d_%s", id, name)
}

func TestDocumentUploadAndProcess_Ready(t *testing.T) {
	svc, auth, uploadDir := newTestDocService(t, nil)
	ctx := context.Background()
	userID := registerUser(t, auth, "doc@example.com")

	samplePath := filepath.Join("..", "..", "testdata", "manual-pengguna.md")
	raw, err := os.ReadFile(samplePath)
	if err != nil {
		t.Fatalf("read sample: %v", err)
	}

	doc, err := svc.Upload(ctx, userID, "manual-pengguna.md", int64(len(raw)), bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	if doc.Status != models.DocumentStatusPending {
		t.Fatalf("status = %s, want pending", doc.Status)
	}

	path := filepath.Join(uploadDir, storedFilename(doc.ID, "manual-pengguna.md"))
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("stored file missing: %v", err)
	}

	if err := svc.Process(ctx, doc.ID); err != nil {
		t.Fatalf("process: %v", err)
	}

	detail, err := svc.Get(ctx, userID, doc.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if detail.Status != models.DocumentStatusReady {
		t.Fatalf("status = %s error=%s", detail.Status, detail.ErrorMsg)
	}
	if detail.ChunkCount < 1 || len(detail.Chunks) < 1 {
		t.Fatalf("chunk_count=%d chunks=%d", detail.ChunkCount, len(detail.Chunks))
	}
	if detail.Chunks[0].Content == "" {
		t.Fatal("empty chunk content")
	}
}

func TestDocumentUpload_InvalidExt(t *testing.T) {
	svc, auth, _ := newTestDocService(t, nil)
	userID := registerUser(t, auth, "ext@example.com")
	_, err := svc.Upload(context.Background(), userID, "virus.exe", 10, strings.NewReader("hello"))
	if !errors.Is(err, service.ErrValidation) {
		t.Fatalf("err = %v, want ErrValidation", err)
	}
}

func TestDocumentUpload_TooLarge(t *testing.T) {
	svc, auth, _ := newTestDocService(t, nil)
	userID := registerUser(t, auth, "big@example.com")
	size := int64(5<<20) + 1
	_, err := svc.Upload(context.Background(), userID, "big.md", size, strings.NewReader("x"))
	if !errors.Is(err, service.ErrValidation) {
		t.Fatalf("err = %v, want ErrValidation", err)
	}
}

func TestDocumentProcess_EmptyFailed(t *testing.T) {
	svc, auth, _ := newTestDocService(t, nil)
	ctx := context.Background()
	userID := registerUser(t, auth, "empty@example.com")

	doc, err := svc.Upload(ctx, userID, "empty.md", 0, strings.NewReader("   \n\n  "))
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	err = svc.Process(ctx, doc.ID)
	if err == nil {
		t.Fatal("expected process error")
	}
	detail, err := svc.Get(ctx, userID, doc.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if detail.Status != models.DocumentStatusFailed {
		t.Fatalf("status = %s, want failed", detail.Status)
	}
	if detail.ErrorMsg == "" {
		t.Fatal("expected error_msg")
	}
}

func TestDocumentProcess_EmbedErrorFailed(t *testing.T) {
	stub := &ai.StubEmbedder{FailWith: errors.New("boom")}
	svc, auth, _ := newTestDocService(t, stub)
	ctx := context.Background()
	userID := registerUser(t, auth, "embedfail@example.com")

	doc, err := svc.Upload(ctx, userID, "ok.md", 11, strings.NewReader("hello world"))
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	_ = svc.Process(ctx, doc.ID)
	detail, err := svc.Get(ctx, userID, doc.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if detail.Status != models.DocumentStatusFailed {
		t.Fatalf("status = %s, want failed", detail.Status)
	}
	if !strings.Contains(detail.ErrorMsg, "embed") {
		t.Fatalf("error_msg = %q", detail.ErrorMsg)
	}
}

func TestDocumentDelete_CascadeAndFile(t *testing.T) {
	svc, auth, uploadDir := newTestDocService(t, nil)
	ctx := context.Background()
	userID := registerUser(t, auth, "del@example.com")

	doc, err := svc.Upload(ctx, userID, "del.md", 12, strings.NewReader("delete me pls"))
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	if err := svc.Process(ctx, doc.ID); err != nil {
		t.Fatalf("process: %v", err)
	}
	path := filepath.Join(uploadDir, storedFilename(doc.ID, "del.md"))
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file missing before delete: %v", err)
	}

	if err := svc.Delete(ctx, userID, doc.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := svc.Get(ctx, userID, doc.ID); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("get after delete: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("file still on disk: %v", err)
	}
}

func TestDocument_OtherUserNotFound(t *testing.T) {
	svc, auth, _ := newTestDocService(t, nil)
	ctx := context.Background()
	owner := registerUser(t, auth, "owner@example.com")
	other := registerUser(t, auth, "other@example.com")

	doc, err := svc.Upload(ctx, owner, "priv.md", 10, strings.NewReader("secret doc"))
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	if _, err := svc.Get(ctx, other, doc.ID); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("get: %v", err)
	}
	if err := svc.Delete(ctx, other, doc.ID); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("delete: %v", err)
	}
}

func TestDocumentReprocess(t *testing.T) {
	svc, auth, _ := newTestDocService(t, nil)
	ctx := context.Background()
	userID := registerUser(t, auth, "re@example.com")

	doc, err := svc.Upload(ctx, userID, "re.md", 20, strings.NewReader("first version content here"))
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	if err := svc.Process(ctx, doc.ID); err != nil {
		t.Fatalf("process: %v", err)
	}
	before, _ := svc.Get(ctx, userID, doc.ID)

	re, err := svc.Reprocess(ctx, userID, doc.ID)
	if err != nil {
		t.Fatalf("reprocess: %v", err)
	}
	if re.Status != models.DocumentStatusProcessing {
		t.Fatalf("status = %s", re.Status)
	}
	if err := svc.Process(ctx, doc.ID); err != nil {
		t.Fatalf("process again: %v", err)
	}
	after, err := svc.Get(ctx, userID, doc.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if after.Status != models.DocumentStatusReady {
		t.Fatalf("status = %s", after.Status)
	}
	if after.ChunkCount < 1 || before.ChunkCount < 1 {
		t.Fatalf("chunks before=%d after=%d", before.ChunkCount, after.ChunkCount)
	}
}
