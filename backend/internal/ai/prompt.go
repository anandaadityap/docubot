package ai

import (
	"fmt"
	"strings"
)

const systemPrompt = `Kamu adalah asisten support yang menjawab berdasarkan dokumen knowledge base.
Aturan wajib:
- Jawab HANYA dari konteks yang diberikan. Jangan memakai pengetahuan di luar itu.
- Jangan menempel atau menyalin seluruh dokumen. Ringkas jadi jawaban percakapan yang membantu.
- Sertakan kutipan sumber sebagai [1], [2], dst sesuai nomor cuplikan yang kamu pakai.
- Jika pertanyaan terlalu umum, tawarkan 3–5 topik yang ada di konteks dan minta pengguna lebih spesifik.
- Jika jawaban tidak ada di konteks, katakan jujur bahwa kamu tidak tahu. Jangan mengarang.
- Jawab dalam bahasa yang sama dengan pertanyaan pengguna.
- Jawaban singkat, jelas, dan membantu. Jangan menampilkan sintaks markdown mentah (#, |, ---) kecuali perlu daftar pendek.`

// BuildRAGPrompt returns system + user messages for the LLM.
func BuildRAGPrompt(question string, hits []ScoredItem) (system, user string) {
	var b strings.Builder
	b.WriteString("Konteks:\n")
	if len(hits) == 0 {
		b.WriteString("(tidak ada cuplikan relevan)\n")
	} else {
		for i, h := range hits {
			fmt.Fprintf(&b, "[%d] (%s): %s\n\n", i+1, h.Filename, strings.TrimSpace(h.Content))
		}
	}
	b.WriteString("Pertanyaan: ")
	b.WriteString(strings.TrimSpace(question))
	return systemPrompt, b.String()
}
