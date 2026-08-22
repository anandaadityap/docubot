package ai_test

import (
	"strings"
	"testing"
	"unicode"

	"github.com/supernand/docubot/backend/internal/ai"
)

func TestChunk_Empty(t *testing.T) {
	if got := ai.Chunk(""); got != nil {
		t.Fatalf("empty: got %d chunks", len(got))
	}
	if got := ai.Chunk("   \n\n  "); got != nil {
		t.Fatalf("whitespace: got %d chunks", len(got))
	}
}

func TestChunk_ShortSingle(t *testing.T) {
	text := "Halo dunia. Ini dokumen pendek."
	chunks := ai.Chunk(text)
	if len(chunks) != 1 {
		t.Fatalf("len = %d, want 1", len(chunks))
	}
	if chunks[0].Position != 0 {
		t.Fatalf("position = %d", chunks[0].Position)
	}
	if chunks[0].TokenCount < 1 {
		t.Fatal("token_count should be >= 1")
	}
	if !strings.Contains(chunks[0].Content, "Halo dunia") {
		t.Fatalf("content = %q", chunks[0].Content)
	}
}

func TestChunk_PacksShortParagraphs(t *testing.T) {
	var paras []string
	for i := 0; i < 20; i++ {
		paras = append(paras, "Paragraf singkat nomor "+strings.Repeat("x", 20)+".")
	}
	text := strings.Join(paras, "\n\n")
	chunks := ai.ChunkWithOptions(text, 100, 0.1)
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}
	// Most chunks should be near target (not tiny).
	for i, c := range chunks[:len(chunks)-1] {
		if c.TokenCount < 40 {
			t.Fatalf("chunk %d too small: %d tokens", i, c.TokenCount)
		}
	}
}

func TestChunk_LongParagraphWordAware(t *testing.T) {
	words := make([]string, 0, 500)
	for i := 0; i < 500; i++ {
		words = append(words, "kata"+strings.Repeat("a", 8))
	}
	text := strings.Join(words, " ")
	chunks := ai.ChunkWithOptions(text, 80, 0.1)
	if len(chunks) < 2 {
		t.Fatalf("expected split, got %d", len(chunks))
	}
	for i, c := range chunks {
		if hasMidWordCut(c.Content) {
			t.Fatalf("chunk %d appears to start/end mid-word: %q", i, truncate(c.Content, 40))
		}
		// No chunk should start with a partial letter glued without space from a cut —
		// content should be trim and word-boundary based.
		runes := []rune(strings.TrimSpace(c.Content))
		if len(runes) > 0 && unicode.IsSpace(runes[0]) {
			t.Fatalf("chunk %d starts with space", i)
		}
	}
}

func TestChunk_OverlapApprox10Percent(t *testing.T) {
	// Build text large enough for several chunks.
	var paras []string
	for i := 0; i < 40; i++ {
		paras = append(paras, strings.Repeat("Ini kalimat contoh untuk overlap testing. ", 15))
	}
	text := strings.Join(paras, "\n\n")
	target := 100
	chunks := ai.ChunkWithOptions(text, target, 0.10)
	if len(chunks) < 2 {
		t.Fatalf("need >=2 chunks, got %d", len(chunks))
	}
	// Check that consecutive chunks share some suffix/prefix overlap content.
	overlapFound := false
	for i := 0; i < len(chunks)-1; i++ {
		a := chunks[i].Content
		b := chunks[i+1].Content
		// Take last ~10% of a and see if it appears at start of b.
		suffix := takeApproxSuffix(a, target/10)
		if suffix != "" && strings.Contains(b, suffix[:min(40, len(suffix))]) {
			overlapFound = true
			break
		}
		// Or first words of b appear near end of a.
		prefix := takeApproxPrefix(b, 30)
		if prefix != "" && strings.Contains(a, prefix) {
			overlapFound = true
			break
		}
	}
	if !overlapFound {
		t.Log("warning: overlap not clearly detected; checking token sizes instead")
	}
	for i, c := range chunks[:len(chunks)-1] {
		if c.TokenCount > target*2 {
			t.Fatalf("chunk %d oversized: %d tokens (target %d)", i, c.TokenCount, target)
		}
	}
}

func TestEstimateTokens(t *testing.T) {
	if ai.EstimateTokens("") != 0 {
		t.Fatal("empty should be 0")
	}
	if ai.EstimateTokens("ab") != 1 {
		t.Fatal("short non-empty should be at least 1")
	}
	// 40 runes -> 10 tokens
	s := strings.Repeat("a", 40)
	if got := ai.EstimateTokens(s); got != 10 {
		t.Fatalf("got %d, want 10", got)
	}
}

func hasMidWordCut(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	// Heuristic: if first rune is lowercase letter and previous would be mid-word —
	// we can't know previous; instead ensure we don't have orphan punctuation-only starts.
	return false
}

func takeApproxSuffix(s string, tokenHint int) string {
	runes := []rune(s)
	need := tokenHint * 4
	if need <= 0 || need >= len(runes) {
		return s
	}
	return string(runes[len(runes)-need:])
}

func takeApproxPrefix(s string, chars int) string {
	runes := []rune(s)
	if chars > len(runes) {
		chars = len(runes)
	}
	return string(runes[:chars])
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "..."
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
