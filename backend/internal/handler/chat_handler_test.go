package handler_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/supernand/docubot/backend/internal/ai"
	"github.com/supernand/docubot/backend/internal/database"
	"github.com/supernand/docubot/backend/internal/handler"
	"github.com/supernand/docubot/backend/internal/middleware"
	"github.com/supernand/docubot/backend/internal/repository"
	"github.com/supernand/docubot/backend/internal/service"
)

func setupFullRouter(t *testing.T, llm ai.LLMProvider) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dir := t.TempDir()
	uploadDir := filepath.Join(dir, "uploads")
	db, err := database.Open(filepath.Join(dir, "test.db"), uploadDir)
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	secret := "test-secret"
	users := repository.NewUserRepo(db)
	authSvc := service.NewAuthService(users, secret)
	authH := handler.NewAuthHandler(authSvc)
	embedder := ai.NewStubEmbedder()
	if llm == nil {
		llm = &ai.StubLLM{Reply: "Buka Settings lalu Security. [1]"}
	}

	docSvc := service.NewDocumentService(repository.NewDocumentRepo(db), repository.NewChunkRepo(db), embedder, uploadDir)
	docH := handler.NewDocumentHandler(docSvc)

	chatSvc := service.NewChatService(
		users,
		repository.NewChunkRepo(db),
		repository.NewConversationRepo(db),
		repository.NewMessageRepo(db),
		repository.NewSettingsRepo(db),
		embedder,
		llm,
	)
	settingsH := handler.NewSettingsHandler(service.NewSettingsService(users, repository.NewSettingsRepo(db)))
	convoH := handler.NewConversationHandler(service.NewConversationService(
		repository.NewConversationRepo(db),
		repository.NewMessageRepo(db),
	))
	analyticsH := handler.NewAnalyticsHandler(service.NewAnalyticsService(repository.NewAnalyticsRepo(db)))

	r := gin.New()
	v1 := r.Group("/api/v1")
	v1.POST("/auth/register", authH.Register)
	v1.POST("/auth/login", authH.Login)
	v1.GET("/bot", settingsH.PublicBot)
	v1.POST("/chat", handler.NewChatHandler(chatSvc).Chat)

	authed := v1.Group("")
	authed.Use(middleware.Auth(secret))
	authed.POST("/documents", docH.Upload)
	authed.GET("/documents", docH.List)
	authed.GET("/documents/:id", docH.Get)
	authed.GET("/conversations", convoH.List)
	authed.GET("/conversations/:id", convoH.Get)
	authed.GET("/analytics/overview", analyticsH.Overview)
	authed.GET("/analytics/top-questions", analyticsH.TopQuestions)
	authed.GET("/settings", settingsH.Get)
	authed.PUT("/settings", settingsH.Put)
	return r
}

func parseSSE(body string) map[string][]json.RawMessage {
	out := map[string][]json.RawMessage{}
	sc := bufio.NewScanner(strings.NewReader(body))
	var event string
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "event:") {
			event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}
		if strings.HasPrefix(line, "data:") && event != "" {
			raw := json.RawMessage(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
			out[event] = append(out[event], raw)
			event = ""
		}
	}
	return out
}

func readSampleManual(t *testing.T) []byte {
	t.Helper()
	candidates := []string{
		filepath.Join("..", "..", "testdata", "manual-pengguna.md"),
		filepath.Join("testdata", "manual-pengguna.md"),
	}
	for _, p := range candidates {
		raw, err := os.ReadFile(p)
		if err == nil {
			return raw
		}
	}
	t.Fatal("testdata/manual-pengguna.md not found")
	return nil
}

func TestChatSSE_SourcesTokenDone(t *testing.T) {
	r := setupFullRouter(t, nil)
	token := loginToken(t, r, "chat@example.com")

	raw := readSampleManual(t)
	uw := uploadMultipart(r, token, "file", "manual-pengguna.md", raw)
	if uw.Code != http.StatusCreated {
		t.Fatalf("upload %d %s", uw.Code, uw.Body.String())
	}
	var created struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(uw.Body.Bytes(), &created)
	waitReady(t, r, token, created.Data.ID)

	w := postJSON(r, "/api/v1/chat", map[string]any{
		"message": "Gimana cara reset password?",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/event-stream") {
		t.Fatalf("content-type = %s", ct)
	}
	ev := parseSSE(w.Body.String())
	if len(ev["sources"]) != 1 {
		t.Fatalf("sources events = %d body=%s", len(ev["sources"]), w.Body.String())
	}
	if len(ev["token"]) == 0 {
		t.Fatalf("no tokens: %s", w.Body.String())
	}
	if len(ev["done"]) != 1 {
		t.Fatalf("done events = %d", len(ev["done"]))
	}
	var done struct {
		ConversationID int64 `json:"conversation_id"`
		MessageID      int64 `json:"message_id"`
	}
	if err := json.Unmarshal(ev["done"][0], &done); err != nil {
		t.Fatalf("done json: %v", err)
	}
	if done.ConversationID == 0 || done.MessageID == 0 {
		t.Fatalf("done payload %+v", done)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations?page=1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	cw := httptest.NewRecorder()
	r.ServeHTTP(cw, req)
	if cw.Code != http.StatusOK {
		t.Fatalf("list conversations %d %s", cw.Code, cw.Body.String())
	}
}

func TestChat_EmptyMessage_JSON(t *testing.T) {
	r := setupFullRouter(t, nil)
	_ = loginToken(t, r, "empty@example.com")
	w := postJSON(r, "/api/v1/chat", map[string]any{"message": "  "})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
}

func TestPublicBotAndSettings(t *testing.T) {
	r := setupFullRouter(t, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/bot", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("bot before register: %d", w.Code)
	}
	var before struct {
		Data struct {
			Configured bool `json:"configured"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &before)
	if before.Data.Configured {
		t.Fatal("expected unconfigured")
	}

	token := loginToken(t, r, "set@example.com")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	sw := httptest.NewRecorder()
	r.ServeHTTP(sw, req)
	if sw.Code != http.StatusOK {
		t.Fatalf("get settings %d %s", sw.Code, sw.Body.String())
	}

	body := map[string]any{
		"bot_name": "TokoBot", "welcome_message": "Halo toko!", "bot_active": true,
		"temperature": 0.2, "max_tokens": 200, "top_k": 3, "min_score": 0.4,
	}
	b, _ := json.Marshal(body)
	preq := httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewReader(b))
	preq.Header.Set("Content-Type", "application/json")
	preq.Header.Set("Authorization", "Bearer "+token)
	pw := httptest.NewRecorder()
	r.ServeHTTP(pw, preq)
	if pw.Code != http.StatusOK {
		t.Fatalf("put settings %d %s", pw.Code, pw.Body.String())
	}

	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/api/v1/bot", nil))
	var after struct {
		Data struct {
			BotName    string `json:"bot_name"`
			Configured bool   `json:"configured"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w2.Body.Bytes(), &after)
	if after.Data.BotName != "TokoBot" || !after.Data.Configured {
		t.Fatalf("public bot %+v", after.Data)
	}
}

func TestAnalyticsOverview(t *testing.T) {
	r := setupFullRouter(t, nil)
	token := loginToken(t, r, "an@example.com")
	raw := readSampleManual(t)
	uw := uploadMultipart(r, token, "file", "manual-pengguna.md", raw)
	if uw.Code != http.StatusCreated {
		t.Fatalf("upload %d %s", uw.Code, uw.Body.String())
	}
	var created struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(uw.Body.Bytes(), &created)
	waitReady(t, r, token, created.Data.ID)
	_ = postJSON(r, "/api/v1/chat", map[string]any{"message": "berapa harga?"})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/overview", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("overview %d %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			TotalConversations int `json:"total_conversations"`
			Daily              []struct {
				Date  string `json:"date"`
				Chats int    `json:"chats"`
			} `json:"daily"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json: %v", err)
	}
	if resp.Data.TotalConversations < 1 {
		t.Fatalf("conversations = %d", resp.Data.TotalConversations)
	}
	if len(resp.Data.Daily) != 14 {
		t.Fatalf("daily len = %d", len(resp.Data.Daily))
	}

	tq := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/top-questions?limit=10", nil)
	tq.Header.Set("Authorization", "Bearer "+token)
	tw := httptest.NewRecorder()
	r.ServeHTTP(tw, tq)
	if tw.Code != http.StatusOK {
		t.Fatalf("top %d %s", tw.Code, tw.Body.String())
	}
}
