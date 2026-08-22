package ai

import (
	"math"
	"sort"
)

// VectorItem is a chunk (or any text unit) with an embedding, used for retrieval.
type VectorItem struct {
	ID         int64
	DocumentID int64
	Filename   string
	Content    string
	Embedding  []float32
}

// ScoredItem is a VectorItem with a cosine similarity score.
type ScoredItem struct {
	VectorItem
	Score float64
}

// Cosine returns cosine similarity of a and b.
// Identical non-zero vectors → 1; orthogonal → 0; empty or mismatched dims → 0.
func Cosine(a, b []float32) float64 {
	n := len(a)
	if n == 0 || n != len(b) {
		return 0
	}
	var dot, na, nb float64
	for i := 0; i < n; i++ {
		fa := float64(a[i])
		fb := float64(b[i])
		dot += fa * fb
		na += fa * fa
		nb += fb * fb
	}
	if na < 1e-18 || nb < 1e-18 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

// TopK returns up to k items with cosine(query, item) >= minScore, highest first.
func TopK(query []float32, items []VectorItem, k int, minScore float64) []ScoredItem {
	if k < 1 || len(query) == 0 || len(items) == 0 {
		return nil
	}
	var hits []ScoredItem
	for _, it := range items {
		score := Cosine(query, it.Embedding)
		if score < minScore {
			continue
		}
		hits = append(hits, ScoredItem{VectorItem: it, Score: score})
	}
	sort.SliceStable(hits, func(i, j int) bool {
		return hits[i].Score > hits[j].Score
	})
	if len(hits) > k {
		hits = hits[:k]
	}
	return hits
}
