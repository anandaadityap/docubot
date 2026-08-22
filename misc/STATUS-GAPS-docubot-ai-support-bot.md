# Status & Gap Analysis — DocuBot (setelah MVP)

| Field | Value |
|---|---|
| **Tipe dokumen** | Status implementasi + gap analysis (teknikal & flow user) |
| **Acuan utama** | [`misc/BRD-PRD-docubot-ai-support-bot.md`](./BRD-PRD-docubot-ai-support-bot.md) — BRD + PRD v1.2 (21 Agustus 2026) |
| **Versi** | 1.1 |
| **Tanggal** | 22 Agustus 2026 |
| **Konteks** | Produk MVP sudah jalan. v1.1 mencatat gap P0–P2 yang **sudah ditutup** di kode (commit setelah snapshot MVP). |

---

## Cara memakai dokumen ini

Dokumen ini **bukan pengganti BRD/PRD**. Ruang lingkup, persona, FR/NFR, skema DB, dan API tetap merujuk ke [`BRD-PRD-docubot-ai-support-bot.md`](./BRD-PRD-docubot-ai-support-bot.md).

**Aturan:** gap di bawah yang masih terbuka adalah utang terhadap MVP yang sudah di-scope di BRD, atau kualitas demo. Bukan undangan menambah PDF widget, multi-tenant, WhatsApp, dsb.

---

## 1. Ringkasan eksekutif

DocuBot fullstack: admin register/login (kunci setelah user pertama), upload `.md`/`.txt` → chunk → embed → `ready`, chat publik SSE dengan kutipan + daftar sumber, riwayat percakapan ke LLM, sesi UUID + persist tab, dashboard (dokumen + cuplikan chunk, percakapan, analitik token/biaya, setelan dengan penjelasan), Docker Compose, test backend, README.

Celah terbesar di v1.0 (register terbuka, tanpa memori chat, kutipan mudah putus, `conversation_id` integer) sudah ditutup.

---

## 2. Status vs BRD (yang sudah terpenuhi)

Referensi FR/NFR/US ada di BRD §5–§6 dan acceptance §3.3.

### 2.1 Fungsional — sudah ada

| Area BRD | Status di kode | Catatan |
|---|---|---|
| FR-01–04 Auth (register, login, JWT 24 jam, `/auth/me`) | Ada | Register default **first-only**; `GET /auth/register-status`; invite opsional |
| FR-05–10 Upload `.md`/`.txt` 5 MB, pipeline, list, hapus cascade | Ada | Ingest timeout 3 menit; overlap reprocess ditolak |
| FR-07 Chunk ~800 token, overlap 10% | Ada | |
| FR-08 Embed + vektor JSON | Ada | Model + dimensi disimpan per chunk/dokumen |
| FR-11 Reprocess | Ada | |
| FR-13–18 Chat SSE, RAG, "hanya dari konteks", simpan conversation | Ada | Riwayat N turn ke LLM; short-circuit 0 KB / 0 hit; `conversation_id` UUID |
| FR-15 / US-09 Kutipan sumber | Ada | Daftar sumber di bubble meskipun model tidak menulis `[n]` |
| FR-19 Latency & token usage | Ada | Ditampilkan di analitik + estimasi USD |
| FR-20–23 Admin dashboard | Ada | Empty state 3 langkah + contoh FAQ; cuplikan chunk; setelan berpenjelasan; admin mobile |
| FR-24–25 Chat publik | Ada | Persist sesi di tab, tombol Chat baru, retry SSE, input disable saat bot off |
| NFR-03 Key LLM hanya di server | Ada | |
| NFR-05 Rate limit | Ada | Chat 10/menit/IP; login/register 5/menit/IP; GC limiter |
| NFR-09 Docker Compose + `.env.example` | Ada | |
| NFR-10 Mobile | Ada | Chat publik + sidebar admin hamburger |
| NFR-12 `go test ./...`, struktur, README | Ada | Bench mencatat TTFT |

### 2.2 Di luar MVP (jangan dikerjakan sebagai "gap")

Sudah sengaja ditunda di BRD §3.2 dan §17: PDF penuh, embed widget, WhatsApp/Telegram, multi-tenant, billing, i18n, voice, LLM lokal default.

---

## 3. Gap — flow user (tertutup di v1.1)

| ID | Status |
|---|---|
| **U1** memori percakapan ke LLM | Tertutup — 6 pesan terakhir + query retrieval gabungan follow-up |
| **U2** refresh = sesi hilang | Tertutup — `sessionStorage` + tombol Chat baru |
| **U3** kutipan hanya jika `[n]` | Tertutup — daftar Sumber di bubble |
| **U4** status bot vs input | Tertutup — input disable saat off; header jujur; empty KB dijelaskan |
| **U5** SSE putus | Tertutup — abort 60s, Coba lagi |
| **U6** integer conversation_id | Tertutup — UUID `public_id` |
| **A1** register terbuka | Tertutup — first-only / invite / closed |
| **A2** onboarding path repo | Tertutup — 3 langkah + unduh contoh FAQ |
| **A3** preview chunk | Tertutup — tombol Cuplikan di Dokumen |
| **A4** setelan tanpa penjelasan | Tertutup |
| **A5** token/biaya analytics | Tertutup |
| **A6** admin mobile | Tertutup |
| **A7** UX 401 | Tertutup — redirect login `?reason=expired` |

---

## 4. Gap — teknikal (tertutup di v1.1, kecuali catatan)

| ID | Status |
|---|---|
| **T1** retrieval load-all | Dilunakkan: WAL + `MaxOpenConns(4)`. Cosine in-Go tetap pilihan MVP. |
| **T2** model embed tidak di-versi | Tertutup — `embed_model` / `embed_dim`; mismatch → pesan reprocess |
| **T3** user message sebelum LLM | Tertutup — persist user+bot bersama setelah jawaban (atau pesan error) |
| **T4** 0 hit tetap LLM | Tertutup — jawaban lokal |
| **T5** ingest tanpa timeout | Tertutup — 3 menit + lock overlap + reset `processing` saat startup |
| **T6** rate limit sempit | Tertutup — auth 5/menit + GC |
| **T7** JWT default | Tertutup — `APP_ENV=production` / `GIN_MODE=release` → fatal |
| **T8** validasi ekstensi saja | Sesuai MVP; bukan bug |
| **T9** timeout tidak selaras | Tertutup — LLM 60s, nginx/Vite ~75–90s, UI 60s |
| **T10** bench tanpa TTFT | Tertutup — `ttft_ms` di script; README menjelaskan Stub vs DeepSeek |
| **T11** tes frontend | Sengaja belum (antrian P2 tidak memaksa Vitest). Backend tes mencakup history, 0-hit, register lock, SSE UUID. |

---

## 5. Prioritas perbaikan (usulan)

Antrian P0–P2 di v1.0 sudah dikerjakan. Yang tersisa sadar:

- Angka bench **DeepSeek nyata** di README setelah deploy VPS (bukan StubLLM).
- Tes frontend (stream parse / SourceChip / 401) jika regresi UI mulai sering.
- Postgres + pgvector jika chunk mendekati 10k (NFR-02) — tetap lewat interface repository.

### Yang tidak masuk antrian ini

PDF, widget embed, multi-tenant, channel chat, dark mode, i18n — lihat BRD §17.

---

## 6. Peta file (untuk sesi berikutnya)

| Concern | Lokasi utama |
|---|---|
| RAG + persist chat | `backend/internal/service/chat_service.go` |
| Prompt | `backend/internal/ai/prompt.go` |
| SSE | `backend/internal/handler/chat_handler.go`, `frontend/src/api/chat.ts` |
| UI chat | `frontend/src/components/chat/ChatWindow.tsx`, `MessageBubble.tsx` |
| Owner bot publik | `user_repo.First`, `settings_service.GetPublic` |
| Register lock | `auth_service.go`, `REGISTER_MODE` |
| Ingest | `document_handler.startProcess` (timeout + lock) |
| Skema | `backend/internal/database/migrate.go` — kolom `public_id`, `embed_model`, `embed_dim` (migrasi sadar vs BRD §8) |

---

## 7. Riwayat dokumen

| Versi | Tanggal | Isi |
|---|---|---|
| 1.0 | 22 Agu 2026 | Snapshot setelah MVP di-repo. Tidak ada perubahan kode. |
| 1.1 | 22 Agu 2026 | Gap P0–P2 + T1–T10 ditutup di kode. Skema: `conversations.public_id`, embed versioning. |
