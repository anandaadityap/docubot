package service_test

import (
	"bytes"
	"context"
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

type capturingEmitter struct {
	sources  []models.Source
	tokens   strings.Builder
	inactive string
	doneID   string
	msgID    int64
}

func (e *capturingEmitter) Sources(s []models.Source) error { e.sources = s; return nil }
func (e *capturingEmitter) Token(c string) error            { e.tokens.WriteString(c); return nil }
func (e *capturingEmitter) Inactive(m string) error         { e.inactive = m; return nil }
func (e *capturingEmitter) Done(cid string, mid int64, _ int, _ int64) error {
	e.doneID = cid
	e.msgID = mid
	return nil
}

func newChatStack(t *testing.T, llm ai.LLMProvider) (*service.ChatService, *service.DocumentService, *service.SettingsService, int64, string) {
	t.Helper()
	dir := t.TempDir()
	uploadDir := filepath.Join(dir, "uploads")
	db, err := database.Open(filepath.Join(dir, "test.db"), uploadDir)
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	embedder := ai.NewStubEmbedder()
	users := repository.NewUserRepo(db)
	bots := repository.NewBotRepo(db)
	auth := service.NewAuthService(users, "test-secret")
	u, err := auth.Register(context.Background(), "Admin", "admin@example.com", "secret123")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	bot, err := bots.GetByUserID(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("bot: %v", err)
	}

	docs := service.NewDocumentService(repository.NewDocumentRepo(db), repository.NewChunkRepo(db), embedder, uploadDir)
	chat := service.NewChatService(
		bots,
		repository.NewDocumentRepo(db),
		repository.NewChunkRepo(db),
		repository.NewConversationRepo(db),
		repository.NewMessageRepo(db),
		repository.NewSettingsRepo(db),
		embedder,
		llm,
	)
	settings := service.NewSettingsService(repository.NewSettingsRepo(db), bots, repository.NewDocumentRepo(db))
	return chat, docs, settings, u.ID, bot.Slug
}

func uploadReady(t *testing.T, docs *service.DocumentService, userID int64) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "manual-pengguna.md"))
	if err != nil {
		t.Fatalf("sample: %v", err)
	}
	doc, err := docs.Upload(context.Background(), userID, "manual-pengguna.md", int64(len(raw)), bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	if err := docs.Process(context.Background(), doc.ID); err != nil {
		t.Fatalf("process: %v", err)
	}
}

func TestChat_EmptyMessage(t *testing.T) {
	chat, _, _, _, slug := newChatStack(t, &ai.StubLLM{})
	err := chat.Chat(context.Background(), service.ChatInput{Slug: slug, Message: "  "}, &capturingEmitter{})
	if err == nil || !strings.Contains(err.Error(), "required") {
		t.Fatalf("err = %v", err)
	}
}

func TestChat_BotInactive(t *testing.T) {
	chat, docs, settings, userID, slug := newChatStack(t, &ai.StubLLM{})
	uploadReady(t, docs, userID)
	cfg, err := settings.Get(context.Background(), userID)
	if err != nil {
		t.Fatalf("get settings: %v", err)
	}
	_, err = settings.Update(context.Background(), userID, service.UpdateInput{
		BotName: cfg.BotName, WelcomeMessage: cfg.WelcomeMessage, BotActive: false,
		Temperature: cfg.Temperature, MaxTokens: cfg.MaxTokens, TopK: cfg.TopK, MinScore: cfg.MinScore,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	em := &capturingEmitter{}
	if err := chat.Chat(context.Background(), service.ChatInput{Slug: slug, Message: "halo"}, em); err != nil {
		t.Fatalf("chat: %v", err)
	}
	if em.inactive == "" {
		t.Fatal("expected inactive event")
	}
	if em.tokens.Len() > 0 {
		t.Fatal("should not stream LLM tokens when inactive")
	}
	if em.doneID == "" {
		t.Fatal("expected done conversation id")
	}
}

func TestChat_NoContext_DoesNotHallucinate(t *testing.T) {
	chat, _, _, _, slug := newChatStack(t, &ai.StubLLM{})

	em := &capturingEmitter{}
	q := "Siapa presiden negara fiktif Atlantis tahun 2099?"
	if err := chat.Chat(context.Background(), service.ChatInput{Slug: slug, Message: q}, em); err != nil {
		t.Fatalf("chat: %v", err)
	}
	if len(em.sources) != 0 {
		t.Fatalf("expected no sources, got %+v", em.sources)
	}
	got := strings.ToLower(em.tokens.String())
	if !strings.Contains(got, "tidak") && !strings.Contains(got, "belum") {
		t.Fatalf("expected honest unknown / empty kb, got %q", em.tokens.String())
	}
}

func TestChat_RelevantSources(t *testing.T) {
	chat, docs, _, userID, slug := newChatStack(t, &ai.StubLLM{Reply: "Buka Settings lalu Security. [1]"})
	uploadReady(t, docs, userID)

	em := &capturingEmitter{}
	if err := chat.Chat(context.Background(), service.ChatInput{Slug: slug, Message: "Gimana cara reset password?"}, em); err != nil {
		t.Fatalf("chat: %v", err)
	}
	if len(em.sources) == 0 {
		t.Fatal("expected at least one source")
	}
	if !strings.Contains(strings.ToLower(em.sources[0].Filename), "manual") {
		t.Fatalf("filename = %s", em.sources[0].Filename)
	}
	if !strings.Contains(em.tokens.String(), "[1]") {
		t.Fatalf("answer = %q", em.tokens.String())
	}
	if em.msgID == 0 {
		t.Fatal("expected saved bot message id")
	}
}

type recordingLLM struct {
	last ai.ChatRequest
	ai.StubLLM
}

func (r *recordingLLM) ChatStream(ctx context.Context, req ai.ChatRequest, onToken func(string) error) (ai.TokenUsage, error) {
	r.last = req
	return r.StubLLM.ChatStream(ctx, req, onToken)
}

func TestChat_FollowUpSendsHistory(t *testing.T) {
	rec := &recordingLLM{StubLLM: ai.StubLLM{Reply: "Harga paket Starter Rp 99.000. [1]"}}
	chat, docs, _, userID, slug := newChatStack(t, rec)
	uploadReady(t, docs, userID)

	em1 := &capturingEmitter{}
	if err := chat.Chat(context.Background(), service.ChatInput{Slug: slug, Message: "Ada paket apa saja?"}, em1); err != nil {
		t.Fatalf("first: %v", err)
	}
	if em1.doneID == "" {
		t.Fatal("expected conversation public id")
	}

	em2 := &capturingEmitter{}
	cid := em1.doneID
	if err := chat.Chat(context.Background(), service.ChatInput{Slug: slug, Message: "itu berapa harganya?", ConversationID: &cid}, em2); err != nil {
		t.Fatalf("follow-up: %v", err)
	}
	if len(rec.last.History) < 2 {
		t.Fatalf("expected prior turns in history, got %d: %+v", len(rec.last.History), rec.last.History)
	}
}

func TestChat_ZeroHit_DoesNotCallLLM(t *testing.T) {
	rec := &recordingLLM{StubLLM: ai.StubLLM{Reply: "should not run"}}
	chat, docs, settings, userID, slug := newChatStack(t, rec)
	uploadReady(t, docs, userID)
	cfg, err := settings.Get(context.Background(), userID)
	if err != nil {
		t.Fatalf("settings: %v", err)
	}
	if _, err := settings.Update(context.Background(), userID, service.UpdateInput{
		BotName: cfg.BotName, WelcomeMessage: cfg.WelcomeMessage, BotActive: true,
		Temperature: cfg.Temperature, MaxTokens: cfg.MaxTokens, TopK: cfg.TopK, MinScore: 0.99,
	}); err != nil {
		t.Fatalf("update: %v", err)
	}

	em := &capturingEmitter{}
	if err := chat.Chat(context.Background(), service.ChatInput{Slug: slug, Message: "xyzzy-unrelated-foobar-999"}, em); err != nil {
		t.Fatalf("chat: %v", err)
	}
	if rec.last.User != "" {
		t.Fatalf("LLM should not be called on 0-hit, got user=%q", rec.last.User)
	}
	got := strings.ToLower(em.tokens.String())
	if !strings.Contains(got, "tidak") {
		t.Fatalf("expected local unknown, got %q", em.tokens.String())
	}
}

func TestChat_UnknownSlug_DoesNotCallLLM(t *testing.T) {
	rec := &recordingLLM{StubLLM: ai.StubLLM{Reply: "should not run"}}
	chat, _, _, _, _ := newChatStack(t, rec)
	em := &capturingEmitter{}
	err := chat.Chat(context.Background(), service.ChatInput{Slug: "tidak-ada", Message: "halo"}, em)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("err = %v", err)
	}
	if rec.last.User != "" {
		t.Fatal("LLM must not run for unknown slug")
	}
}

func TestChat_DoesNotLeakOtherUserChunks(t *testing.T) {
	dir := t.TempDir()
	uploadDir := filepath.Join(dir, "uploads")
	db, err := database.Open(filepath.Join(dir, "test.db"), uploadDir)
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	embedder := ai.NewStubEmbedder()
	users := repository.NewUserRepo(db)
	bots := repository.NewBotRepo(db)
	auth := service.NewAuthServiceWithPolicy(users, "test-secret", "open", "")
	a, err := auth.Register(context.Background(), "Alice", "a@example.com", "secret123")
	if err != nil {
		t.Fatalf("register a: %v", err)
	}
	b, err := auth.Register(context.Background(), "Bob", "b@example.com", "secret123")
	if err != nil {
		t.Fatalf("register b: %v", err)
	}
	botA, _ := bots.GetByUserID(context.Background(), a.ID)
	botB, _ := bots.GetByUserID(context.Background(), b.ID)

	docs := service.NewDocumentService(repository.NewDocumentRepo(db), repository.NewChunkRepo(db), embedder, uploadDir)
	uploadReady(t, docs, a.ID)
	raw := []byte("Jam buka toko adalah pukul delapan pagi.")
	docB, err := docs.Upload(context.Background(), b.ID, "jam-buka.txt", int64(len(raw)), strings.NewReader(string(raw)))
	if err != nil {
		t.Fatalf("upload b: %v", err)
	}
	if err := docs.Process(context.Background(), docB.ID); err != nil {
		t.Fatalf("process b: %v", err)
	}

	chat := service.NewChatService(
		bots,
		repository.NewDocumentRepo(db),
		repository.NewChunkRepo(db),
		repository.NewConversationRepo(db),
		repository.NewMessageRepo(db),
		repository.NewSettingsRepo(db),
		embedder,
		&ai.StubLLM{Reply: "ok [1]"},
	)

	emB := &capturingEmitter{}
	if err := chat.Chat(context.Background(), service.ChatInput{Slug: botB.Slug, Message: "Gimana cara reset password?"}, emB); err != nil {
		t.Fatalf("chat b: %v", err)
	}
	for _, src := range emB.sources {
		if strings.Contains(strings.ToLower(src.Filename), "manual") {
			t.Fatalf("bot B used A's document: %+v", emB.sources)
		}
	}

	emA := &capturingEmitter{}
	if err := chat.Chat(context.Background(), service.ChatInput{Slug: botA.Slug, Message: "Gimana cara reset password?"}, emA); err != nil {
		t.Fatalf("chat a: %v", err)
	}
	if len(emA.sources) == 0 {
		t.Fatal("bot A should retrieve from its own KB")
	}
}
