package ai

import (
	"strings"
)

const (
	// DefaultTargetTokens is the approximate chunk size (BRD FR-07).
	DefaultTargetTokens = 800
	// DefaultOverlapRatio is 10% overlap between consecutive chunks.
	DefaultOverlapRatio = 0.10
)

// TextChunk is a slice of source text with an estimated token count.
type TextChunk struct {
	Content    string
	TokenCount int
	Position   int
}

// Chunk splits text into ~targetTokens chunks with overlap, preferring paragraph
// boundaries and never cutting mid-word. Empty/whitespace-only input yields nil.
func Chunk(text string) []TextChunk {
	return ChunkWithOptions(text, DefaultTargetTokens, DefaultOverlapRatio)
}

// ChunkWithOptions is like Chunk but with explicit target and overlap ratio.
func ChunkWithOptions(text string, targetTokens int, overlapRatio float64) []TextChunk {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if targetTokens < 1 {
		targetTokens = DefaultTargetTokens
	}
	if overlapRatio < 0 {
		overlapRatio = 0
	}
	if overlapRatio > 0.5 {
		overlapRatio = 0.5
	}
	overlapTokens := int(float64(targetTokens) * overlapRatio)
	if overlapTokens < 1 && overlapRatio > 0 {
		overlapTokens = 1
	}

	units := flattenUnits(text, targetTokens)
	if len(units) == 0 {
		return nil
	}

	var chunks []TextChunk
	i := 0
	for i < len(units) {
		start := i
		tok := 0
		for i < len(units) {
			uTok := EstimateTokens(units[i])
			if tok > 0 && tok+uTok > targetTokens {
				break
			}
			tok += uTok
			i++
			if tok >= targetTokens {
				break
			}
		}
		if i == start {
			// Single oversized unit (should be rare after splitOversized).
			i = start + 1
		}
		end := i
		content := strings.TrimSpace(strings.Join(units[start:end], "\n\n"))
		chunks = append(chunks, TextChunk{
			Content:    content,
			TokenCount: EstimateTokens(content),
			Position:   len(chunks),
		})
		if end >= len(units) {
			break
		}
		// Next window starts earlier for overlap, but must still advance.
		next := end
		if overlapTokens > 0 {
			back := end - 1
			acc := 0
			for back > start {
				acc += EstimateTokens(units[back])
				if acc >= overlapTokens {
					break
				}
				back--
			}
			next = back
			if next <= start {
				next = start + 1
			}
		}
		i = next
	}

	return chunks
}

func flattenUnits(text string, targetTokens int) []string {
	var units []string
	for _, p := range splitParagraphs(text) {
		units = append(units, splitOversized(p, targetTokens)...)
	}
	return units
}

// EstimateTokens approximates token count as max(1, runeCount/4) for non-empty text.
func EstimateTokens(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	n := len([]rune(s))
	t := n / 4
	if t < 1 {
		return 1
	}
	return t
}

func splitParagraphs(text string) []string {
	parts := strings.Split(text, "\n\n")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// splitOversized breaks a paragraph that exceeds targetTokens at word boundaries.
func splitOversized(paragraph string, targetTokens int) []string {
	if EstimateTokens(paragraph) <= targetTokens {
		return []string{paragraph}
	}

	words := strings.Fields(paragraph)
	if len(words) == 0 {
		return nil
	}

	var out []string
	var buf []string
	for _, w := range words {
		trial := w
		if len(buf) > 0 {
			trial = strings.Join(buf, " ") + " " + w
		}
		if len(buf) > 0 && EstimateTokens(trial) > targetTokens {
			out = append(out, strings.Join(buf, " "))
			buf = []string{w}
			continue
		}
		buf = append(buf, w)
	}
	if len(buf) > 0 {
		out = append(out, strings.Join(buf, " "))
	}
	return out
}
