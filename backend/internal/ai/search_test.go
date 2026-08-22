package ai_test

import (
	"math"
	"testing"

	"github.com/supernand/docubot/backend/internal/ai"
)

func TestCosine_Identical(t *testing.T) {
	v := []float32{0.3, 0.4, 0.0}
	got := ai.Cosine(v, v)
	if math.Abs(got-1) > 1e-6 {
		t.Fatalf("identical = %v, want 1", got)
	}
}

func TestCosine_Orthogonal(t *testing.T) {
	a := []float32{1, 0, 0}
	b := []float32{0, 1, 0}
	got := ai.Cosine(a, b)
	if math.Abs(got) > 1e-6 {
		t.Fatalf("orthogonal = %v, want 0", got)
	}
}

func TestCosine_MismatchedOrEmpty(t *testing.T) {
	if ai.Cosine(nil, []float32{1}) != 0 {
		t.Fatal("empty should be 0")
	}
	if ai.Cosine([]float32{1}, []float32{1, 2}) != 0 {
		t.Fatal("mismatch should be 0")
	}
}

func TestTopK_OrderAndMinScore(t *testing.T) {
	q := []float32{1, 0, 0}
	items := []ai.VectorItem{
		{ID: 1, Content: "low", Embedding: []float32{0.1, 0.9, 0}}, // ~0.11
		{ID: 2, Content: "best", Embedding: []float32{1, 0, 0}},    // 1.0
		{ID: 3, Content: "mid", Embedding: []float32{0.8, 0.2, 0}}, // ~0.97
		{ID: 4, Content: "zero", Embedding: []float32{0, 1, 0}},    // 0
	}
	hits := ai.TopK(q, items, 5, 0.3)
	if len(hits) != 2 {
		t.Fatalf("len = %d, want 2 (min_score 0.3)", len(hits))
	}
	if hits[0].ID != 2 || hits[1].ID != 3 {
		t.Fatalf("order ids = %d, %d want 2, 3", hits[0].ID, hits[1].ID)
	}
	if hits[0].Score < hits[1].Score {
		t.Fatalf("not descending: %v then %v", hits[0].Score, hits[1].Score)
	}

	top1 := ai.TopK(q, items, 1, 0)
	if len(top1) != 1 || top1[0].ID != 2 {
		t.Fatalf("top-1 = %+v", top1)
	}
}

func TestTopK_Empty(t *testing.T) {
	if ai.TopK(nil, []ai.VectorItem{{ID: 1}}, 5, 0) != nil {
		t.Fatal("nil query")
	}
	if ai.TopK([]float32{1}, nil, 5, 0) != nil {
		t.Fatal("nil items")
	}
	if ai.TopK([]float32{1}, []ai.VectorItem{{Embedding: []float32{1}}}, 0, 0) != nil {
		t.Fatal("k=0")
	}
}

func TestStubEmbedder_SimilarTextRanksHigher(t *testing.T) {
	s := ai.NewStubEmbedder()
	texts := []string{
		"Buka menu Settings lalu Security untuk reset password.",
		"Kebijakan refund penuh dalam 7 hari setelah pembelian.",
		"Harga paket Starter adalah Rp 99.000 per bulan.",
	}
	vecs, err := s.Embed(t.Context(), texts)
	if err != nil {
		t.Fatalf("embed chunks: %v", err)
	}
	q, err := s.Embed(t.Context(), []string{"gimana cara reset password?"})
	if err != nil {
		t.Fatalf("embed query: %v", err)
	}
	items := make([]ai.VectorItem, len(texts))
	for i := range texts {
		items[i] = ai.VectorItem{ID: int64(i + 1), Content: texts[i], Embedding: vecs[i]}
	}
	hits := ai.TopK(q[0], items, 1, 0)
	if len(hits) != 1 {
		t.Fatalf("hits = %d", len(hits))
	}
	if hits[0].ID != 1 {
		t.Fatalf("top hit id = %d content=%q, want reset-password chunk", hits[0].ID, hits[0].Content)
	}
}
