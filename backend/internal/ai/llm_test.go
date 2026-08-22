package ai_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supernand/docubot/backend/internal/ai"
)

func TestStubLLM_UnknownWithoutContext(t *testing.T) {
	llm := &ai.StubLLM{}
	var got strings.Builder
	_, err := llm.ChatStream(context.Background(), ai.ChatRequest{
		System: "sys",
		User:   "Konteks:\n(tidak ada cuplikan relevan)\n\nPertanyaan: siapa raja mars?",
	}, func(tok string) error {
		got.WriteString(tok)
		return nil
	})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	if !strings.Contains(strings.ToLower(got.String()), "tidak") {
		t.Fatalf("got %q", got.String())
	}
}

func TestOpenAIChat_StreamsTokens(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("path %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			t.Errorf("auth %s", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fl := w.(http.Flusher)
		chunks := []map[string]any{
			{"choices": []map[string]any{{"delta": map[string]any{"content": "Halo"}}}},
			{"choices": []map[string]any{{"delta": map[string]any{"content": " dunia"}}}},
			{"choices": []map[string]any{{"delta": map[string]any{}}}, "usage": map[string]any{
				"prompt_tokens": 10, "completion_tokens": 2, "total_tokens": 12,
			}},
		}
		for _, c := range chunks {
			b, _ := json.Marshal(c)
			_, _ = w.Write([]byte("data: " + string(b) + "\n\n"))
			fl.Flush()
		}
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer srv.Close()

	client := ai.NewOpenAIChat("sk-test", srv.URL, "deepseek-chat")
	var got strings.Builder
	usage, err := client.ChatStream(context.Background(), ai.ChatRequest{
		System: "s", User: "u", Temperature: 0.2, MaxTokens: 50,
	}, func(tok string) error {
		got.WriteString(tok)
		return nil
	})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	if got.String() != "Halo dunia" {
		t.Fatalf("got %q", got.String())
	}
	if usage.TotalTokens != 12 {
		t.Fatalf("usage %+v", usage)
	}
}

func TestStubLLM_VagueQuestionListsTopics(t *testing.T) {
	content := `# FAQ Support

## Jam operasional & kontak
Chat bot aktif 24 jam.

## Cara reset password
Buka Settings lalu Security.

## Paket & harga
Starter Rp 99.000.
`
	_, user := ai.BuildRAGPrompt("mau tanya mengenai toko", []ai.ScoredItem{
		{VectorItem: ai.VectorItem{Filename: "faq-toko-kita.md", Content: content}},
	})
	llm := &ai.StubLLM{}
	var got strings.Builder
	_, err := llm.ChatStream(context.Background(), ai.ChatRequest{User: user}, func(tok string) error {
		got.WriteString(tok)
		return nil
	})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	out := got.String()
	if strings.Contains(out, "# FAQ") || strings.Contains(out, "## ") {
		t.Fatalf("dumped raw markdown: %q", out)
	}
	if !strings.Contains(strings.ToLower(out), "topik") {
		t.Fatalf("expected topic list, got %q", out)
	}
	if strings.Count(out, " ") > 80 {
		t.Fatalf("answer too long for vague question: %q", out)
	}
}

func TestStubLLM_SpecificQuestionExtractsSection(t *testing.T) {
	content := `# FAQ

## Cara reset password
Buka **Settings** lalu Security. Klik Reset password.

## Paket & harga
Starter Rp 99.000 per bulan.
`
	_, user := ai.BuildRAGPrompt("gimana cara reset password?", []ai.ScoredItem{
		{VectorItem: ai.VectorItem{Filename: "faq.md", Content: content}},
	})
	llm := &ai.StubLLM{}
	var got strings.Builder
	_, err := llm.ChatStream(context.Background(), ai.ChatRequest{User: user}, func(tok string) error {
		got.WriteString(tok)
		return nil
	})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	out := strings.ToLower(got.String())
	if !strings.Contains(out, "settings") {
		t.Fatalf("expected reset section, got %q", got.String())
	}
	if strings.Contains(out, "99.000") {
		t.Fatalf("leaked unrelated harga section: %q", got.String())
	}
}

func TestBuildRAGPrompt_EmptyHits(t *testing.T) {
	sys, user := ai.BuildRAGPrompt("apa kabar?", nil)
	if sys == "" {
		t.Fatal("empty system")
	}
	if !strings.Contains(user, "(tidak ada cuplikan relevan)") {
		t.Fatalf("user = %s", user)
	}
	if !strings.Contains(user, "apa kabar?") {
		t.Fatal("missing question")
	}
}
