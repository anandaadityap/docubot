# Status & Gap Analysis — DocuBot (setelah MVP)

| Field | Value |
|---|---|
| **Tipe dokumen** | Status implementasi + gap analysis (teknikal & flow user) |
| **Acuan utama** | [`misc/BRD-PRD-docubot-ai-support-bot.md`](./BRD-PRD-docubot-ai-support-bot.md) — BRD + PRD v1.2 (21 Agustus 2026) |
| **Versi** | 1.0 |
| **Tanggal** | 22 Agustus 2026 |
| **Konteks** | Produk MVP sudah jalan dan dinilai cukup oke. Dokumen ini mencatat **apa yang sudah terpenuhi**, **celah yang masih ada**, dan **prioritas perbaikan** — tanpa mengubah ruang lingkup BRD. |

---

## Cara memakai dokumen ini

Dokumen ini **bukan pengganti BRD/PRD**. Ruang lingkup, persona, FR/NFR, skema DB, dan API tetap merujuk ke [`BRD-PRD-docubot-ai-support-bot.md`](./BRD-PRD-docubot-ai-support-bot.md).

Dokumen ini adalah **perkembangan berdasarkan BRD/PRD tersebut**: snapshot setelah milestone M1–M8 diimplementasikan di repo. Gunakan untuk:

1. Konteks sesi coding berikutnya (apa yang sudah ada, apa yang sengaja belum).
2. Memilih perbaikan tanpa mengulang audit dari nol.
3. Menjaga anti-scope-creep: ide di luar MVP tetap di §17 Roadmap BRD, bukan masuk daftar gap di sini kecuali disebut eksplisit.

**Aturan:** gap di bawah ini adalah utang terhadap MVP yang sudah di-scope di BRD, atau kualitas demo yang merusak janji produk. Bukan undangan menambah PDF widget, multi-tenant, WhatsApp, dsb.

---

## 1. Ringkasan eksekutif

DocuBot sudah menjadi produk fullstack yang bisa didemo: admin register/login, upload `.md`/`.txt` → chunk → embed → `ready`, chat publik SSE dengan kutipan, dashboard (dokumen, percakapan, analitik 14 hari, setelan), Docker Compose, test backend, README.

Yang masih mengganggu **kualitas chat sebagai support bot** dan **keamanan demo live** lebih penting daripada fitur baru. Celah terbesar:

| # | Celah | Dampak |
|---|---|---|
| 1 | Register terbuka + owner = user pertama | Demo live bisa di-hijack; admin ke-2 tidak menggerakkan bot publik |
| 2 | LLM tidak menerima riwayat percakapan | Follow-up ("itu berapa harganya?") gagal |
| 3 | Chip sumber hanya muncul jika model menulis `[1]` | Janji "kutipan sumber" mudah putus |
| 4 | `conversation_id` integer berurutan di endpoint publik | Percakapan pengunjung bisa ditempeli orang lain |

---

## 2. Status vs BRD (yang sudah terpenuhi)

Referensi FR/NFR/US ada di BRD §5–§6 dan acceptance §3.3.

### 2.1 Fungsional — sudah ada

| Area BRD | Status di kode | Catatan |
|---|---|---|
| FR-01–04 Auth (register, login, JWT 24 jam, `/auth/me`) | Ada | Password bcrypt, middleware Bearer |
| FR-05–10 Upload `.md`/`.txt` 5 MB, pipeline `pending → processing → ready \| failed`, list, hapus cascade | Ada | Proses di goroutine setelah upload |
| FR-07 Chunk ~800 token, overlap 10%, batas paragraf | Ada | `internal/ai/chunker.go` |
| FR-08 Embed + simpan vektor JSON di SQLite | Ada | OpenAI `text-embedding-3-small` atau StubEmbedder jika key kosong |
| FR-11 Reprocess | Ada | UI tombol "Proses ulang" |
| FR-13–18 Chat SSE `sources → token → done`, cosine top-k + min_score, prompt "hanya dari konteks", simpan conversation/messages, bot off tanpa LLM | Ada | Event tambahan `inactive` dan `error` |
| FR-19 Latency & token usage disimpan | Ada di DB | **Belum ditampilkan** di analytics (lihat gap A11) |
| FR-20–23 Admin: dokumen, percakapan + pagination, analytics chart, setelan | Ada | |
| FR-24–25 Chat publik + tautan Admin Login | Ada | Juga `/register` |
| NFR-03 Key LLM hanya di server | Ada | |
| NFR-05 Rate limit chat 10/menit/IP | Ada | Hanya `/chat`, bukan register/login |
| NFR-09 Docker Compose + `.env.example` | Ada | |
| NFR-12 `go test ./...`, struktur folder, README | Ada | Frontend tanpa tes |

### 2.2 Di luar MVP (jangan dikerjakan sebagai "gap")

Sudah sengaja ditunda di BRD §3.2 dan §17:

- PDF penuh (FR-12 P1) — MVP hanya `.md`/`.txt`
- Embed widget `<script>`, WhatsApp/Telegram, multi-tenant, billing
- Multi-bahasa UI, voice, LLM lokal sebagai default

---

## 3. Gap — flow user

Persona: **Customer** (pengunjung `/`) dan **Admin/Owner** (bukan programmer). User stories: BRD §4.

### 3.1 Customer (chat publik)

| ID | Masalah | Perilaku sekarang | Kenapa merusak janji produk |
|---|---|---|---|
| **U1** | Tidak ada memori percakapan ke LLM | Setiap turn: embed pertanyaan + RAG + prompt sendirian. Riwayat tidak dikirim. | US-08: orang chat dengan follow-up. "itu" / "yang tadi" gagal. **Celah kualitas terbesar.** |
| **U2** | Refresh = sesi hilang | `conversation_id` hanya di state React. Tidak ada `sessionStorage`. Tidak ada tombol "Chat baru". | Reload tab = history UI hilang + conversation DB baru. Terasa seperti bug. |
| **U3** | Kutipan tidak selalu bisa diklik | Event `sources` ada, tapi `SourceChip` hanya render jika teks memuat `[n]`. Tidak ada daftar sumber di bawah bubble. | FR-15 / US-09: kepercayaan lewat sumber. Model yang lupa mengutip = sumber "hilang". |
| **U4** | Status bot vs input | Input tidak di-disable saat `bot_active = false`. Header bisa "Online" tanpa dokumen `ready`. | Pengunjung baru tahu setelah kirim: inactive / "tidak tahu", tanpa penjelasan KB kosong. US-11 kurang jelas. |
| **U5** | SSE putus tanpa recovery | Tidak ada abort client, timeout UI, atau "Coba lagi". | BRD §14 sudah mencatat risiko ini. Jawaban terpotong, kirim ulang = pesan baru. |
| **U6** | Sesi chat mudah ditebak | `conversation_id` integer (bukan UUID seperti API spec BRD §9.3). Chat publik. | Pengunjung A bisa `POST` dengan `conversation_id: 1` dan menempel ke sesi orang lain. |

### 3.2 Admin

| ID | Masalah | Perilaku sekarang | Kenapa merusak janji produk |
|---|---|---|---|
| **A1** | Register tidak pernah ditutup | Siapa pun `POST /auth/register`. Chat publik + `GET /bot` memakai **user id terkecil** (`users.First()`). | Live demo: orang pertama jadi owner bot. Admin ke-2 upload ke akun sendiri — **tidak muncul di `/`**. BRD §9.1: register terbuka untuk demo, production invite-only — belum ada saklar. |
| **A2** | Onboarding putus | Setelah daftar → Dokumen. Empty state menyebut path `backend/testdata/manual-pengguna.md`. | Persona owner bukan programmer (BRD §4.1). Tidak ada langkah "upload → tunggu ready → coba chat". |
| **A3** | Preview chunk tidak di UI | `GET /documents/:id` sudah mengembalikan cuplikan chunk. Halaman Dokumen tidak memakainya. | Sulit debug "kenapa bot salah jawab" tanpa buka API. |
| **A4** | Setelan RAG tanpa penjelasan | Temperature, top-k, min score, max tokens polos. | Owner non-teknis mudah merusak retrieval. |
| **A5** | Analitik tidak menutup loop biaya | Token usage sudah di `messages.token_usage`. Dashboard: total chat, pesan, avg latency, chart 14 hari, top 10. | FR-19 + risiko cost runaway (BRD §14) tidak terlihat di UI. |
| **A6** | Admin hampir tidak mobile | Sidebar tetap `w-56`, tabel dokumen tidak collapse. | NFR-10: mobile responsif. Chat publik relatif oke. |
| **A7** | JWT 24 jam, UX 401 senyap | Token `localStorage`. 401 menghapus token; user tetap di `/admin/*` sampai pindah route. | Tidak ada "sesi berakhir, login lagi". |

---

## 4. Gap — teknikal

Implementasi inti: `backend/internal/service/chat_service.go` (RAG), `document_handler.go` (ingest async), `frontend/src/components/chat/ChatWindow.tsx`.

| ID | Masalah | Detail di kode | Risiko |
|---|---|---|---|
| **T1** | Retrieval load-all setiap chat | `ListReadyWithEmbeddingsForUser` + unmarshal JSON semua vektor, cosine in-Go. SQLite `MaxOpenConns(1)`, tidak ada WAL. | Cukup untuk demo kecil. NFR-02 (≤10k chunk &lt; 500 ms) rawan. Chat bisa ngantri di belakang ingest. |
| **T2** | Model embedding tidak di-versi | Stub 64 dim vs OpenAI 1536 dim. Tidak ada kolom model/dimensi di `chunks`. | Ganti `EMBED_API_KEY` / model tanpa reprocess → cosine tidak berarti, tanpa error jelas. |
| **T3** | User message disimpan sebelum LLM | `msgs.Create` user dulu, lalu retrieve + stream. Error LLM tidak rollback. | Log percakapan & analytics: pertanyaan tanpa jawaban bot. |
| **T4** | 0 hit tetap panggil LLM | `min_score` menyaring semua / belum ada dokumen `ready` → prompt "(tidak ada cuplikan relevan)" tetap ke DeepSeek. | Biaya + latency sia-sia (NFR-07). Bisa dijawab lokal. |
| **T5** | Ingest tanpa timeout / antrian | `startProcess` → goroutine `Process`. Tidak ada batas overlap reprocess. | Embed hang = status `processing` selamanya. |
| **T6** | Rate limit sempit | Hanya `POST /chat`. Register/login bebas. Limiter in-memory, key IP tidak di-GC. | Spam akun / brute password di demo publik. |
| **T7** | JWT secret default | `JWT_SECRET=change-me-please` hanya `log.Printf` warning; server tetap jalan. | VPS live dengan secret lemah. |
| **T8** | Validasi file hanya ekstensi | `.md` / `.txt` dari nama file, bukan isi. | Sesuai MVP; bukan bug PDF. |
| **T9** | Timeout tidak selaras | LLM HTTP 90s, frontend nginx `proxy_read_timeout` 120s, BRD §14 ~60s. | User menunggu lama tanpa feedback. |
| **T10** | Benchmark belum bukti portfolio | `scripts/bench.sh` ≈ wall time total, bukan TTFT. README masih angka StubLLM. | KPI NFR-01 / §15 belum tercatat di README setelah deploy. |
| **T11** | Tes frontend tidak ada | Backend: auth, chunker, search, chat SSE, inactive, "tidak tahu". | Regresi UI (stream parse, SourceChip, 401) tidak tertangkap. |

**Catatan arsitektur yang sengaja (bukan bug):** cosine in-Go + embedding JSON TEXT adalah pilihan MVP (BRD §7). Upgrade path tetap Postgres + pgvector via interface repository.

---

## 5. Prioritas perbaikan (usulan)

Kerjakan dari atas. Jangan campur dengan Roadmap §17 BRD.

| Prioritas | ID | Item | Alasan |
|---|---|---|---|
| **P0** | A1 | Kunci register (satu admin / invite-only / disable setelah user pertama) | Demo live tidak boleh di-hijack |
| **P0** | U1 | Kirim N turn terakhir ke LLM (window pendek, tetap RAG per pertanyaan) | Follow-up = cara orang memakai support bot |
| **P0** | U3 | Tampilkan daftar sumber di bubble meskipun model tidak menulis `[n]` | Janji kutipan sumber |
| **P1** | U6 | Sesi publik UUID / token opaq, bukan integer tebak-tebakan | Privacy percakapan |
| **P1** | U2 | Persist `conversation_id` + UI di tab; tombol chat baru | Refresh tidak mereset sesi |
| **P1** | T4, U4 | Short-circuit 0 dokumen / 0 hit; disable input saat bot off; empty-state jujur | Hemat biaya, pesan jujur |
| **P1** | A2 | Empty state admin + arahkan sample FAQ, tanpa path repo | Flow owner |
| **P2** | T2, T5 | Versi embedder + timeout ingest | Ketahanan |
| **P2** | A5, T10 | Token/biaya di analytics; angka bench nyata di README | Cerita portfolio |
| **P2** | A6, A4, A7, U5, T6, T7 | Mobile admin, penjelasan setelan, UX 401, retry SSE, rate limit auth, tolak JWT default di production | Polesan |

### Yang tidak masuk antrian ini

PDF, widget embed, multi-tenant, channel chat, dark mode, i18n — lihat BRD §17. Baru dikerjakan setelah MVP & portfolio selesai.

---

## 6. Peta file (untuk sesi berikutnya)

| Concern | Lokasi utama |
|---|---|
| RAG + persist chat | `backend/internal/service/chat_service.go` |
| Prompt "jangan mengarang" | `backend/internal/ai/prompt.go` |
| SSE | `backend/internal/handler/chat_handler.go`, `frontend/src/api/chat.ts` |
| UI chat | `frontend/src/components/chat/ChatWindow.tsx`, `MessageBubble.tsx` |
| Owner bot publik | `backend/internal/repository/user_repo.go` (`First`), `settings_service.go` (`GetPublic`) |
| Register | `backend/internal/handler/auth_handler.go`, `frontend/src/pages/LoginPage.tsx` |
| Ingest async | `backend/internal/handler/document_handler.go` (`startProcess`) |
| Retrieval | `backend/internal/ai/search.go`, `repository/chunk_repo.go` |
| Skema | `backend/internal/database/migrate.go` — harus tetap selaras BRD §8 kecuali ada migrasi sadar |

---

## 7. Riwayat dokumen

| Versi | Tanggal | Isi |
|---|---|---|
| 1.0 | 22 Agu 2026 | Snapshot pertama setelah MVP di-repo: status vs BRD v1.2, gap flow + teknikal, prioritas P0–P2. Tidak ada perubahan kode. |

Kalau BRD/PRD naik versi, update kolom acuan di header dan tambah baris di tabel ini. Gap yang sudah ditutup: pindahkan ke §2 (sudah terpenuhi) dan coret dari §5.
