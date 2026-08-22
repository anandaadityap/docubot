package ai

import (
	"context"
	"strings"
	"time"
)

// ChatRequest is an LLM completion request (non-RAG fields only).
type ChatRequest struct {
	System      string
	History     []ChatTurn
	User        string
	Temperature float64
	MaxTokens   int
}

// TokenUsage is prompt + completion token counts from the provider.
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// LLMProvider streams chat completions.
type LLMProvider interface {
	ChatStream(ctx context.Context, req ChatRequest, onToken func(token string) error) (TokenUsage, error)
}

// StubLLM answers without an external API: extractive quote if context exists,
// otherwise an honest "tidak tahu". Used when LLM_API_KEY is empty and in tests.
type StubLLM struct {
	// Reply, if set, is streamed instead of the default extractive answer.
	Reply string
	// FailWith, if set, makes ChatStream return this error.
	FailWith error
	// Delay between tokens (tests leave this 0).
	TokenDelay time.Duration
}

// ChatStream implements LLMProvider.
func (s *StubLLM) ChatStream(ctx context.Context, req ChatRequest, onToken func(token string) error) (TokenUsage, error) {
	if err := ctx.Err(); err != nil {
		return TokenUsage{}, err
	}
	if s.FailWith != nil {
		return TokenUsage{}, s.FailWith
	}

	text := strings.TrimSpace(s.Reply)
	if text == "" {
		text = stubReply(req.User)
	}

	words := strings.Fields(text)
	var built strings.Builder
	for i, w := range words {
		if err := ctx.Err(); err != nil {
			return TokenUsage{}, err
		}
		piece := w
		if i > 0 {
			piece = " " + w
		}
		built.WriteString(piece)
		if err := onToken(piece); err != nil {
			return TokenUsage{}, err
		}
		if s.TokenDelay > 0 {
			select {
			case <-ctx.Done():
				return TokenUsage{}, ctx.Err()
			case <-time.After(s.TokenDelay):
			}
		}
	}

	promptEst := EstimateTokens(req.System) + EstimateTokens(req.User)
	for _, h := range req.History {
		promptEst += EstimateTokens(h.Content)
	}
	compEst := EstimateTokens(built.String())
	return TokenUsage{
		PromptTokens:     promptEst,
		CompletionTokens: compEst,
		TotalTokens:      promptEst + compEst,
	}, nil
}

func stubReply(userPrompt string) string {
	question, snippets := parseRAGUserPrompt(userPrompt)
	if len(snippets) == 0 {
		return "Maaf, saya tidak menemukan informasi itu di dokumen knowledge base. Silakan tanya hal lain yang ada di panduan."
	}

	best := snippets[0]
	sections := splitMDSections(best)
	matched := matchSections(question, sections)

	if isVagueQuestion(question) || len(matched) == 0 {
		topics := sectionTitles(sections)
		if len(topics) == 0 {
			preview := stripMarkdown(best)
			preview = truncateRunes(preview, 280)
			return "Saya menemukan ini di dokumen: " + preview + " [1]"
		}
		return "Saya bisa bantu dari knowledge base toko. Topik yang tersedia: " +
			strings.Join(topics, "; ") +
			". Coba tanya lebih spesifik, misalnya harga, refund, atau cara reset password. [1]"
	}

	var b strings.Builder
	for i, s := range matched {
		if i >= 2 {
			break
		}
		if s.title != "" {
			b.WriteString(s.title)
			b.WriteString(": ")
		}
		b.WriteString(stripMarkdown(s.body))
		if i == 0 {
			b.WriteString(" ")
		}
	}
	out := strings.TrimSpace(collapseSpace(b.String()))
	out = truncateRunes(out, 600)
	return out + " [1]"
}

func parseRAGUserPrompt(userPrompt string) (question string, snippets []string) {
	qIdx := strings.LastIndex(userPrompt, "Pertanyaan:")
	body := userPrompt
	if qIdx >= 0 {
		question = strings.TrimSpace(userPrompt[qIdx+len("Pertanyaan:"):])
		body = userPrompt[:qIdx]
	}
	const marker = "Konteks:"
	if i := strings.Index(body, marker); i >= 0 {
		body = body[i+len(marker):]
	}
	body = strings.TrimSpace(body)
	if body == "" || strings.Contains(body, "(tidak ada cuplikan relevan)") {
		return question, nil
	}

	parts := splitSnippetBlocks(body)
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// Drop leading "[1] (file.md):"
		if strings.HasPrefix(p, "[") {
			if j := strings.Index(p, "):"); j >= 0 {
				p = strings.TrimSpace(p[j+2:])
			}
		}
		if p != "" {
			snippets = append(snippets, p)
		}
	}
	return question, snippets
}

func splitSnippetBlocks(body string) []string {
	var out []string
	start := 0
	for i := 0; i < len(body); i++ {
		if i > 0 && body[i] == '[' && (body[i-1] == '\n') {
			out = append(out, body[start:i])
			start = i
		}
	}
	out = append(out, body[start:])
	return out
}

type mdSection struct {
	title string
	body  string
}

func splitMDSections(text string) []mdSection {
	lines := strings.Split(text, "\n")
	var out []mdSection
	var cur mdSection
	flush := func() {
		cur.body = strings.TrimSpace(cur.body)
		if cur.title != "" || cur.body != "" {
			out = append(out, cur)
		}
		cur = mdSection{}
	}
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "## ") {
			flush()
			cur.title = strings.TrimSpace(strings.TrimPrefix(trim, "## "))
			continue
		}
		if strings.HasPrefix(trim, "# ") && cur.title == "" && strings.TrimSpace(cur.body) == "" {
			cur.title = strings.TrimSpace(strings.TrimPrefix(trim, "# "))
			continue
		}
		if trim == "---" {
			continue
		}
		cur.body += line + "\n"
	}
	flush()
	return out
}

func sectionTitles(sections []mdSection) []string {
	var out []string
	for _, s := range sections {
		t := strings.TrimSpace(s.title)
		if t == "" || strings.HasPrefix(strings.ToLower(t), "faq") {
			continue
		}
		out = append(out, t)
		if len(out) >= 8 {
			break
		}
	}
	return out
}

var stubStop = map[string]struct{}{
	"mau": {}, "tanya": {}, "mengenai": {}, "tentang": {}, "ya": {},
	"yuk": {}, "halo": {}, "hai": {}, "hi": {}, "info": {}, "informasi": {},
	"toko": {}, "shop": {}, "bantu": {}, "bantuan": {}, "bisa": {}, "saya": {},
	"aku": {}, "ada": {}, "yang": {}, "untuk": {}, "apa": {}, "kah": {},
	"tolong": {}, "minta": {}, "dong": {}, "nih": {}, "ini": {}, "itu": {},
	"dengan": {}, "dari": {}, "ke": {}, "di": {}, "dan": {}, "atau": {},
}

func tokenize(s string) []string {
	s = strings.ToLower(s)
	var b strings.Builder
	var out []string
	flush := func() {
		w := b.String()
		b.Reset()
		if w == "" {
			return
		}
		if _, skip := stubStop[w]; skip {
			return
		}
		if len([]rune(w)) < 3 {
			return
		}
		out = append(out, w)
	}
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			continue
		}
		flush()
	}
	flush()
	return out
}

func isVagueQuestion(q string) bool {
	return len(tokenize(q)) == 0
}

func matchSections(question string, sections []mdSection) []mdSection {
	qtoks := tokenize(question)
	if len(qtoks) == 0 {
		return nil
	}
	type scored struct {
		s     mdSection
		score int
	}
	var hits []scored
	for _, sec := range sections {
		blob := strings.ToLower(sec.title + " " + sec.body)
		n := 0
		for _, t := range qtoks {
			if strings.Contains(blob, t) {
				n++
			}
		}
		if n == 0 {
			continue
		}
		hits = append(hits, scored{s: sec, score: n})
	}
	if len(hits) == 0 {
		return nil
	}
	best := hits[0]
	for _, h := range hits[1:] {
		if h.score > best.score {
			best = h
		}
	}
	out := []mdSection{best.s}
	for _, h := range hits {
		if h.s.title != best.s.title && h.score == best.score {
			out = append(out, h.s)
			break
		}
	}
	return out
}

func stripMarkdown(s string) string {
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "__", "")
	s = strings.ReplaceAll(s, "`", "")
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "|") {
			cells := strings.Split(trim, "|")
			var keep []string
			for _, c := range cells {
				c = strings.TrimSpace(c)
				if c == "" || strings.Trim(c, "-: ") == "" {
					continue
				}
				keep = append(keep, c)
			}
			if len(keep) > 0 {
				lines = append(lines, strings.Join(keep, " — "))
			}
			continue
		}
		if strings.HasPrefix(trim, "#") {
			trim = strings.TrimSpace(strings.TrimLeft(trim, "#"))
		}
		if strings.HasPrefix(trim, "- ") {
			trim = "• " + strings.TrimSpace(trim[2:])
		}
		if trim != "" {
			lines = append(lines, trim)
		}
	}
	return collapseSpace(strings.Join(lines, " "))
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

func collapseSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

var _ LLMProvider = (*StubLLM)(nil)
