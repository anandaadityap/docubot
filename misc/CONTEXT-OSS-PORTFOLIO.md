# Context — kemasan OSS & portofolio (setelah V2)

| Field | Value |
|---|---|
| **Tipe dokumen** | Context sesi berikutnya: kondisi produk + pekerjaan kemasan (bukan fitur baru) |
| **Versi** | 1.2 |
| **Tanggal** | 23 Agustus 2026 |
| **Status** | Kemasan P0/P1 di-repo sebagian selesai; demo live + i18n masih nanti |
| **Produk** | DocuBot (Go + React + SQLite + RAG), self-host 1 akun = 1 bot |

> **Bukan pengganti** BRD, STATUS-GAPS, atau PLAN-V2. File ini menjawab satu pertanyaan: *produk V2 sudah ada di kode; apa yang masih kurang agar repo layak open source dan item portofolio?*

---

## Cara memakai dokumen ini

Baca berurutan. Jangan mulai dari file ini jika Anda belum tahu apa itu DocuBot.

| # | File | Dipakai untuk | Jangan dipakai untuk |
|---|---|---|---|
| 1 | [`BRD-PRD-docubot-ai-support-bot.md`](./BRD-PRD-docubot-ai-support-bot.md) | Ruang lingkup RAG, persona, FR/NFR, skema/API **MVP**, prinsip desain | Status milestone (§11 masih menulis M4–M8 pending — **basi**). Target penghasilan / strategi bid Upwork (§2, §18) **jangan** jadi copy GitHub publik |
| 2 | [`STATUS-GAPS-docubot-ai-support-bot.md`](./STATUS-GAPS-docubot-ai-support-bot.md) | Gap MVP P0–P2 yang **sudah ditutup**, peta file RAG/SSE | Antrian fitur pasca-V2 (dokumen ini beku di 22 Agu, sebelum slug/landing) |
| 3 | [`PLAN-V2-docubot-selfhost-embed.md`](./PLAN-V2-docubot-selfhost-embed.md) | Keputusan produk V2 + DoD kode. **Paling akurat untuk perilaku aplikasi** | Pekerjaan kemasan OSS (CI, screenshot, saring `misc/`) — itu tugas file ini |
| 4 | **Dokumen ini** | P0–P2 kemasan; non-goal; DoD “siap publik / siap portofolio” | Menambah FR produk (PDF, WhatsApp, N-bot, npm widget) |
| 5 | [`README.md`](../README.md) | Permukaan publik clone/run/embed | Spesifikasi lengkap (tetap di BRD/PLAN-V2) |
| 6 | [`LICENSE`](../LICENSE) | MIT | — |
| 7 | [`example/embed-dummy.html`](./example/embed-dummy.html) | Bukti iframe lokal (ganti `SLUG`) | Desain halaman klien |
| 8 | `frontend/public/samples/faq-contoh.md` | FAQ demo ingest (copy publik di app) | — |

**Aturan konflik**

- Fitur produk / slug / iframe / non-goal V2 → ikuti PLAN-V2.
- RAG, auth, dokumen, SSE, analytics, Docker → ikuti BRD (perilaku), STATUS-GAPS (apa yang sudah ditutup).
- Status “apa yang dikerjakan sekarang” untuk OSS/portofolio → **file ini**.
- BRD §11 (tabel milestone) dan STATUS-GAPS §5 **bukan** backlog aktif.

---

## Status papan (23 Agu 2026)

Bahasa: **produk (landing, admin, chat, error) = Indonesia saja.** Open source nanti butuh i18n (minimal `id` + `en`); itu **K17 — nanti**, bukan sprint sekarang. Jangan campur string Inggris di UI.

### Selesai di repo

| ID | Item | Bukti |
|---|---|---|
| K4 | Gitignore WAL/dist/tmp | `.gitignore` |
| K5 | `misc/` opsi B | README tidak menautkan angka penghasilan; [`README.md`](./README.md) di folder ini |
| K6 | README etalase | [`README.md`](../README.md) |
| K7 | CI | `.github/workflows/ci.yml` |
| K8 | Frontend README | `frontend/README.md` |
| K9 | Pin Go 1.26 | `backend/go.mod` + `backend/Dockerfile` |
| K10 | Landing 3 langkah + kutipan | `frontend/src/pages/LandingPage.tsx` |
| K11 | Model keamanan tertulis | [`SECURITY.md`](../SECURITY.md) + README |
| K12 | Identitas bot satu jalur | Sudah di kode V2 (BotService); jangan tambah penulis `settings` ketiga |
| K13 | Contributing + security | [`CONTRIBUTING.md`](../CONTRIBUTING.md), [`SECURITY.md`](../SECURITY.md) |
| K14 | Changelog 0.2.0 | [`CHANGELOG.md`](../CHANGELOG.md) — **tag git `v0.2.0` belum** |

### Belum — kerjakan berikutnya (butuh VPS / rekaman / kunci API)

| ID | Item | Catatan |
|---|---|---|
| **K1** | Demo live | Deploy `chatbot.supernand.tech` (atau URL final) |
| **K2** | Screenshot + rekaman 60–90 dtk | Alur daftar → unggah FAQ → tes → iframe dummy |
| **K3** | Bench DeepSeek nyata | Satu baris angka di README, bukan StubLLM |
| K14b | Tag `v0.2.0` | Setelah commit kemasan ini |

### Nanti — jangan dikerjakan sekarang

| ID | Item | Catatan |
|---|---|---|
| **K17** | i18n (`id` + `en`) | Sekarang **hanya Indonesia**. Nanti: file locale, UI EN untuk visitor OSS/Upwork. Jangan string hardcoded campur bahasa. |
| K15 | README/landing dwibahasa | Tergantung K17. README GitHub boleh tetap EN sebagai clone surface; UI tetap ID sampai K17. |
| K16 | Tes frontend | Hanya jika regresi UI sering (SSE / slug 404) |

---

## 1. Kebenaran produk (23 Agu 2026)

DocuBot adalah **aplikasi self-host**, bukan SaaS multi-tenant dan bukan library RAG.

Alur yang sudah jalan di kode:

1. Register admin pertama → baris `users` + `settings` + `bots` (slug) dalam transaksi yang sama.
2. Unggah `.md` / `.txt` → chunk → embed → `ready`.
3. Chat publik `POST /api/v1/b/:slug/chat` (SSE + kutipan). Slug salah = 404, **bukan** fallback user #1.
4. `/` = landing. `/b/:slug` = chat pelanggan (tanpa Admin Login). `?embed=1` untuk iframe.
5. Admin **Pasang**: URL publik + snippet iframe + playground (`channel=playground`).
6. Docker Compose, tes backend, LICENSE MIT, README self-host + embed.

Pengecualian sadar `GetOldest` / bot tertua: **hanya** `GET /api/v1/demo` untuk tombol “Coba demo” di landing — tidak boleh dipakai menjawab chat.

Sisa DoD V2 (PLAN-V2 §3.2): **rekaman/screenshot alur daftar → unggah FAQ → tes → salin snippet → chat dari HTML dummy** belum wajib tercatat. Deploy live `chatbot.supernand.tech` dan angka bench DeepSeek nyata di README juga belum.

`UserRepo.First()` masih ada di repository tetapi **bukan** path produksi chat. Jangan dihidupkan lagi.

---

## 2. Verdict (jangan dilebihkan di copy publik)

| Lensa | Nilai | Artinya |
|---|---|---|
| Produk MVP + V2 | Kuat | RAG + slug + iframe + admin utuh |
| Repo OSS | Inti kemasan ada | CI + README etalase + SECURITY/CONTRIBUTING. Belum: demo live, screenshot, bench DeepSeek, i18n |
| Portofolio Upwork | Inti ada, kemasan belum | Cerita “self-host + embed” valid; tanpa demo live + screenshot + angka, klien tidak bisa verifikasi |

Klaim yang **boleh** setelah P0 selesai: self-host, Docker satu perintah, chat berkutipan, iframe ke `/b/{slug}`.

Klaim yang **belum boleh**: “live production”, “TTFT < 2s DeepSeek” (NFR-01), “siap di-fork siapa pun tanpa gesekan”.

---

## 3. Antrian kerja kemasan

Kerjakan **P0 → P1 → P2**. Jangan sisipkan fitur dari §5.

### 3.1 P0 — wajib sebelum repo publik / item Upwork

| ID | Kerja | Definisi selesai | Acuan |
|---|---|---|---|
| **K1** | Demo live | `https://chatbot.supernand.tech` (atau URL final) buka landing, `/b/{slug}` menjawab dari KB, iframe tidak diblokir `X-Frame-Options` | README Deploy; PLAN-V2 §13; BRD NFR-11 |
| **K2** | Bukti visual | 3–4 screenshot: landing, chat + daftar sumber, admin Dokumen, admin Pasang. Satu rekaman 60–90 dtk: daftar → unggah sample FAQ → tes → salin iframe → chat dari [`embed-dummy.html`](./example/embed-dummy.html) | PLAN-V2 §3.2 checkbox terakhir; BRD §16 (media portofolio) |
| **K3** | Bench nyata | Satu baris README: TTFT / latency / token / estimasi USD dengan **DeepSeek**, bukan StubLLM. Script: `scripts/bench.sh` / `scripts/bench.ps1` | BRD NFR-01, §12.4; STATUS-GAPS T10 / §5 |
| **K4** | Git bersih | **Selesai (gitignore + commit kemasan).** Jangan commit `frontend/dist/`, `backend/tmp/`, `*.db-wal` / `*.db-shm` | `.gitignore` |
| **K5** | Saring `misc/` untuk GitHub publik | BRD memuat target penghasilan dan strategi bid. Opsi A: jangan push `misc/`. Opsi B: push tetapi README **tidak** menautkan angka pribadi; file ini + PLAN-V2 cukup sebagai peta. Pilih **sebelum** repo public | BRD §2, §18 |
| **K6** | README etalase | Urutan: problem → apa yang dibangun → hasil; link demo; screenshot; batas skala jujur (cosine in-Go, nyaman ≪ 10k chunk); cara embed iframe; `go test ./...` | README sekarang; BRD §16 pola SCARA |
| **K7** | CI minimal | GitHub Actions: `go test ./...` + `npm ci && npm run build` di `frontend/` | BRD NFR-12; klaim disiplin engineering |

### 3.2 P1 — clone & percaya

| ID | Kerja | Catatan |
|---|---|---|
| **K8** | Hapus/ganti `frontend/README.md` template Vite | Menyesatkan visitor yang membuka folder frontend dulu |
| **K9** | Pin Go yang bisa di-clone | `go.mod` `1.26.5` vs `Dockerfile` `golang:1.26-alpine` — samakan major; jangan paksa patch yang hanya ada di mesin author |
| **K10** | Landing sedikit lebih “bukti” | Tetap konten minimum PLAN-V2 §9.2. Boleh: 3 langkah + satu cuplikan “jawaban berkutipan”. **Bukan** situs marketing |
| **K11** | Dokumentasikan model keamanan | JWT di `localStorage`; iframe tanpa `frame-ancestors` allowlist (sadar, PLAN-V2 §13); `channel=playground` **jangan** dipromosikan di README. BRD NFR-03–06 |
| **K12** | Identitas bot satu jalur | Baca publik/admin dari `bots`. Dual-write ke `settings` hanya lewat BotService / SettingsService yang sudah ada. Jangan tambah penulis ketiga. PLAN-V2 §7.2, §15 |

### 3.3 P2 — setelah demo hidup

| ID | Kerja | Catatan |
|---|---|---|
| **K13** | `CONTRIBUTING.md` + `SECURITY.md` pendek | Cara stub LLM, tes, jangan commit `.env` |
| **K14** | Tag `v0.2.0` + CHANGELOG satu paragraf | V2 = slug + landing + iframe |
| **K15** | README/landing dwibahasa | Nanti, setelah K17. UI tetap ID sekarang. |
| **K16** | Tes frontend | Nanti. Kasus: parse SSE, slug 404. STATUS-GAPS T11 tetap sah ditunda |
| **K17** | i18n `id` + `en` | **Nanti.** Sekarang fokus Indonesia saja. Open source perlu EN nanti; jangan mulai ekstrak string di sprint ini. |

---

## 4. DoD kemasan (file ini selesai jika)

Centang saat mengerjakan; ini **bukan** DoD fitur V2 (itu PLAN-V2 §3.2).

- [ ] Demo URL di README bisa dicoba tanpa clone — **belum** (K1, butuh VPS)
- [ ] Screenshot atau rekaman ada di README (atau `docs/screenshots/`) — **belum** (K2)
- [ ] Satu angka bench DeepSeek tercatat — **belum** (K3, butuh API key + deploy)
- [x] `go test ./...` di CI — `.github/workflows/ci.yml` (23 Agu 2026)
- [x] Working tree: `.gitignore` menolak `frontend/dist/`, `backend/tmp/`, `*.db-wal` / `*.db-shm`; kemasan di-commit
- [x] Keputusan K5: **opsi B** — `misc/` boleh di repo; README publik tidak menautkan angka penghasilan; peta di [`README.md`](../README.md) + file ini
- [x] Copy README tidak mengklaim hosted SaaS / multi-tenant / widget npm

Lihat **Status papan** di atas. Ringkas: P0 kode/docs kemasan **selesai kecuali K1–K3**. Bahasa UI **Indonesia**; i18n = K17 nanti.

Setelah DoD ini: **berhenti menambah fitur** sampai ada permintaan klien nyata (PLAN-V2 §2 poin 6, BRD §17).

---

## 5. Non-goal — jangan dikerjakan dari file ini

Sama rohnya dengan PLAN-V2 §4 dan §16, STATUS-GAPS §2.2 / §5, BRD §3.2 / §17:

- PDF penuh, ingest URL, WhatsApp / Telegram / Slack
- N bot per akun, `bot_id` di `documents` / `conversations`
- Multi-tenant, org, billing, hosted SaaS
- Paket npm / SDK React / `pkg/rag` publik
- pgvector / pindah Postgres “karena rapi”
- Dark mode, thumbs up/down
- i18n **sekarang** (itu K17 nanti; UI tetap Indonesia)
- Rewrite frontend atau ganti Gin
- Widget `<script>` lintas origin yang `fetch` ke API (V2 = iframe same-origin)

Jika suatu PR “kemasan” menyentuh item di atas, pecah PR atau tolak.

---

## 6. Peta file (kemasan + permukaan V2)

Melengkapi STATUS-GAPS §6 dan PLAN-V2 §5 / paket A–D.

| Concern | Lokasi |
|---|---|
| Landing | `frontend/src/pages/LandingPage.tsx` |
| Chat publik / embed | `frontend/src/pages/PublicChatPage.tsx`, `frontend/src/components/chat/ChatWindow.tsx` |
| Pasang + playground | `frontend/src/pages/admin/InstallPage.tsx` |
| API slug | `frontend/src/api/chat.ts`, `frontend/src/api/admin.ts` |
| Chat by slug | `backend/internal/service/chat_service.go`, `backend/internal/handler/chat_handler.go` |
| Bot + demo pointer | `backend/internal/service/bot_service.go`, `backend/internal/handler/bot_handler.go` |
| Slugify | `backend/internal/util/slug.go` |
| Skema `bots` + `conversations.channel` | `backend/internal/database/migrate.go` |
| Dummy iframe | `misc/example/embed-dummy.html` |
| Bench | `scripts/bench.sh`, `scripts/bench.ps1` |
| Compose / nginx iframe | `docker-compose.yml`, `frontend/nginx.conf` |

---

## 7. Copy portofolio (setelah P0)

Pakai pola BRD §16 (problem → action → result). Sesuaikan bentuk **self-host + iframe**, bukan “chat hanya di VPS saya”.

Jangan tempel target Rp/bulan atau rate bid ke GitHub.

Draft (edit angka setelah K3):

**Title:** Self-host AI support bot (RAG) — Go + React, embed via iframe

**Problem:** Small businesses repeat the same FAQ answers; they already have Markdown/TXT docs but no bot on their own site.

**Action:** Built a single-binary-friendly stack: Go (Gin) API, SQLite, chunk + embed + cosine retrieval, SSE answers with citations. React admin uploads docs, copies an iframe to `/b/{slug}`. Docker Compose one-command self-host.

**Result:** Common questions answered from the owner’s documents with visible sources. Embed does not need CORS on the shop origin (iframe is same-origin to DocuBot). Demo: [URL]. Tests: `go test ./...`.

---

## 8. Riwayat dokumen

| Versi | Tanggal | Isi |
|---|---|---|
| 1.0 | 23 Agu 2026 | Context kemasan OSS + portofolio. Acuan: BRD v1.2, STATUS-GAPS v1.1, PLAN-V2 v1.1. Tidak mengubah kode. |
| 1.1 | 23 Agu 2026 | Kemasan di-repo: CI, README etalase, SECURITY/CONTRIBUTING, landing bukti, gitignore WAL, `go 1.26`. K1–K3 masih terbuka. |
| 1.2 | 23 Agu 2026 | Papan status selesai vs nanti. Kebijakan bahasa: UI Indonesia sekarang; i18n `id`+`en` = K17 nanti. |

---

*Pekerjaan dari file ini adalah kemasan dan bukti, bukan sprint fitur. Setelah DoD §4, stop.*
