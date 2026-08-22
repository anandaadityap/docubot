package ai_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supernand/docubot/backend/internal/ai"
)

func TestStubEmbedder_Deterministic(t *testing.T) {
	s := ai.NewStubEmbedder()
	ctx := context.Background()
	a, err := s.Embed(ctx, []string{"hello", "world"})
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	b, err := s.Embed(ctx, []string{"hello", "world"})
	if err != nil {
		t.Fatalf("embed2: %v", err)
	}
	if len(a) != 2 || len(b) != 2 {
		t.Fatalf("len a=%d b=%d", len(a), len(b))
	}
	if len(a[0]) == 0 {
		t.Fatal("empty vector")
	}
	for i := range a[0] {
		if a[0][i] != b[0][i] {
			t.Fatalf("not deterministic at %d", i)
		}
	}
	// Different texts → different vectors.
	same := true
	for i := range a[0] {
		if a[0][i] != a[1][i] {
			same = false
			break
		}
	}
	if same {
		t.Fatal("different texts produced identical vectors")
	}
}

func TestStubEmbedder_FailWith(t *testing.T) {
	s := &ai.StubEmbedder{FailWith: ai.ErrEmbedderUnavailable}
	_, err := s.Embed(context.Background(), []string{"x"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOpenAIEmbedder_BatchAndAuth(t *testing.T) {
	var calls int
	var gotAuth string
	var inputs [][]string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		gotAuth = r.Header.Get("Authorization")
		if r.URL.Path != "/embeddings" {
			t.Errorf("path = %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Model string   `json:"model"`
			Input []string `json:"input"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			t.Errorf("unmarshal: %v", err)
			http.Error(w, "bad", 400)
			return
		}
		inputs = append(inputs, req.Input)

		type item struct {
			Embedding []float32 `json:"embedding"`
			Index     int       `json:"index"`
		}
		data := make([]item, len(req.Input))
		for i := range req.Input {
			data[i] = item{Index: i, Embedding: []float32{float32(i), 0.5}}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
	}))
	defer srv.Close()

	e := ai.NewOpenAIEmbedder("sk-test", srv.URL, "text-embedding-3-small")
	// Force small batch via embedding 40 texts — but batch size is 32 internal.
	texts := make([]string, 40)
	for i := range texts {
		texts[i] = strings.Repeat("t", i+1)
	}
	vecs, err := e.Embed(context.Background(), texts)
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	if gotAuth != "Bearer sk-test" {
		t.Fatalf("auth = %q", gotAuth)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2 (batches of 32+8)", calls)
	}
	if len(inputs[0]) != 32 || len(inputs[1]) != 8 {
		t.Fatalf("batch sizes = %d, %d", len(inputs[0]), len(inputs[1]))
	}
	if len(vecs) != 40 {
		t.Fatalf("vecs = %d", len(vecs))
	}
	if len(vecs[0]) != 2 {
		t.Fatalf("dims = %d", len(vecs[0]))
	}
}

func TestOpenAIEmbedder_MissingKey(t *testing.T) {
	e := ai.NewOpenAIEmbedder("", "http://example.com/v1", "m")
	_, err := e.Embed(context.Background(), []string{"a"})
	if err == nil {
		t.Fatal("expected error")
	}
}
