# Perkembangan & Plan V2 — DocuBot: self-host + identitas bot + embed

| Field | Value |
|---|---|
| **Tipe dokumen** | Catatan perkembangan setelah MVP + rencana implementasi V2 |
| **Versi** | 1.1 |
| **Tanggal** | 23 Agustus 2026 |
| **Status** | Implemented di kode (lihat §3.2; rekaman demo masih opsional) |
| **Produk** | DocuBot (Go + React + SQLite + RAG) |

---

## Cara memakai dokumen ini

Dokumen ini **bukan pengganti** BRD/PRD dan **bukan pengganti** gap analysis MVP. Baca berurutan:

1. [`BRD-PRD-docubot-ai-support-bot.md`](./BRD-PRD-docubot-ai-support-bot.md) — ruang lingkup MVP, persona, FR/NFR, skema awal, API awal, prinsip desain. **Masih berlaku** untuk RAG, auth, dokumen, SSE, analytics, deploy Docker.
2. [`STATUS-GAPS-docubot-ai-support-bot.md`](./STATUS-GAPS-docubot-ai-support-bot.md) — apa yang sudah di-ship vs BRD, gap P0–P2 yang **sudah ditutup**, peta file. Anggap MVP **selesai**.
3. **Dokumen ini** — keputusan produk setelah MVP, dan plan V2 yang harus diimplementasi berikutnya.

Contoh FAQ untuk tes/demo tetap di [`example/faq-toko-kita.md`](./example/faq-toko-kita.md) dan `frontend/public/samples/faq-contoh.md`.

**Aturan:** jika dokumen ini bertentangan dengan BRD §3.2 / §17 atau STATUS-GAPS §2.2 / §5 tentang widget embed, **dokumen ini yang diikuti untuk V2**. Multi-tenant, library/SDK, PDF, WhatsApp, billing tetap ditunda (lihat §4 Non-goal).

---

## Daftar isi

1. [Ringkasan perkembangan](#1-ringkasan-perkembangan)
2. [Keputusan V2](#2-keputusan-v2)
3. [Tujuan & sukses](#3-tujuan--kriteria-sukses)
4. [Non-goal](#4-non-goal--jangan-dikerjakan-di-v2)
5. [Keadaan kode sekarang](#5-keadaan-kode-sekarang-baseline)
6. [Target V2](#6-target-v2)
7. [Model data](#7-model-data)
8. [API](#8-spesifikasi-api)
9. [Frontend](#9-frontend--ux)
10. [Paket kerja](#10-paket-kerja-urutan-wajib)
11. [Migrasi & kompatibilitas](#11-migrasi--kompatibilitas)
12. [Aturan slug](#12-aturan-slug)
13. [Embed iframe](#13-embed-iframe)
14. [Tes](#14-tes-wajib)
15. [Risiko](#15-risiko)
16. [Setelah V2](#16-setelah-v2-bukan-sekarang)
17. [Riwayat](#17-riwayat-dokumen)

---

## 1. Ringkasan perkembangan

### 1.1 Yang sudah terjadi (MVP)

Sesuai BRD v1.2 dan STATUS-GAPS v1.1:

- Auth JWT, register first-only, upload `.md`/`.txt`, chunk + embed, chat SSE + kutipan, sesi UUID, dashboard dokumen/percakapan/analitik/setelan, Docker, tes backend.
- Gap alur MVP (memori chat, persist tab, daftar sumber, register lock, dsb.) **sudah ditutup**.

Produk yang di-ship: **satu instance demo**. Chat publik di `/` milik user terdaftar pertama (`users.First()`). Tidak ada slug, tidak ada saluran pasang di situs lain.

### 1.2 Mengapa MVP terasa kurang

Bukan karena RAG rusak. Karena tiga pekerjaan bertabrakan di satu URL:

| Siapa | Yang mereka dapat di `/` | Yang mereka butuhkan |
|---|---|---|
| Visitor / klien Upwork | Widget chat + Admin Login | Landing: apa ini, coba demo, cara self-host |
| Owner bot | Tidak ada tautan/snippet pasang | Wizard, tes, salin tautan + iframe |
| Pelanggan bisnis | Chat di domain DocuBot, ada jejak admin | Chat di situs mereka, tanpa login admin |

Skema sudah punya `user_id` di dokumen/percakapan/setelan (kelihatan tenant-shaped). Permukaan publik mengabaikannya.

### 1.3 Ide yang ditolak sebagai langkah berikutnya

Dicatat supaya sesi implementasi tidak mengulang debat:

| Ide | Putusan | Alasan singkat |
|---|---|---|
| Multi-tenant / org SaaS | Tolak di V2 | Chat tetap `users.First()` tanpa slug; tenant tanpa embed = N dashboard kosong |
| Library Go / npm component | Tolak sebagai fokus | Repo adalah aplikasi; hanya `internal/ai` yang layak di-import; pasar RAG Go sudah ramai |
| Plugin SDK “drop-in di sistem user” | Tolak bentuk lebar | Yang dibutuhkan: iframe ke halaman chat ber-slug |

Bentuk yang dipilih: **aplikasi OSS yang di-self-host** (clone / `docker compose up`), satu akun = satu bot, bisa di-embed lewat iframe.

---

## 2. Keputusan V2

1. **Bentuk produk:** aplikasi self-host (MIT), bukan hosted SaaS, bukan library.
2. **Identitas:** bot punya `slug` publik. Semua chat dan profil publik memakai slug, bukan user pertama.
3. **Kardinalitas V2:** 1 user = 1 bot. Dokumen, percakapan, setelan RAG tetap di `user_id`. Jangan tambah `bot_id` ke `documents` / `conversations` di V2 (itu persiapan N bot, di luar slice).
4. **Permukaan:** `/` = landing. `/b/:slug` = chat pelanggan. Admin terpisah.
5. **Pasang:** snippet iframe ke `/b/:slug` (atau `?embed=1`). Bukan npm, bukan `<script>` widget yang `fetch` lintas origin.
6. **Setelah deploy + rekaman demo: berhenti menambah fitur** sampai ada kebutuhan klien nyata.

---

## 3. Tujuan & kriteria sukses

### 3.1 Tujuan bisnis (tidak berubah dari BRD §2, disesuaikan bentuk)

- Portfolio Upwork: cerita “support bot RAG yang di-self-host dan di-embed ke situs klien”, bukan “chat di VPS saya”.
- Berguna: owner bisa unggah dokumen, tes, salin iframe, tempel di HTML lain, chat jalan.

### 3.2 Definition of Done — V2 selesai jika semua ini benar

- [x] `POST /api/v1/b/:slug/chat` menolak slug yang tidak ada (bukan fallback ke user #1).
- [x] `GET /api/v1/bots/:slug` mengembalikan profil bot itu, termasuk `has_ready_kb`.
- [x] Tidak ada path produksi yang memanggil `users.First()` untuk menjawab chat. (Pengecualian sadar: `GET /api/v1/demo` boleh menunjuk bot tertua untuk tombol “Coba demo” di landing — lihat §8.3.)
- [x] `/` adalah landing, bukan `ChatWindow` penuh.
- [x] `/b/:slug` chat tanpa tautan Admin Login.
- [x] Admin `/admin/install` menampilkan URL publik + snippet iframe siap salin.
- [x] File HTML dummy (boleh `misc/example/embed-dummy.html`) memuat iframe itu dan chat memakai knowledge base bot tersebut.
- [x] Register membuat baris `bots` dalam transaksi yang sama dengan `users` + `settings`.
- [x] DB lama (user tanpa bot) di-backfill saat migrate.
- [x] `go test ./...` hijau; tes baru di §14 lulus.
- [x] `LICENSE` MIT; README menjelaskan self-host + embed.
- [ ] Satu rekaman/screenshot alur: daftar → unggah FAQ → tes → salin snippet → chat dari HTML dummy.

### 3.3 Di luar DoD V2 (boleh jelek / belum ada)

Landing tidak perlu desain marketing kelas SaaS. Playground admin boleh sederhana (lihat paket C). Origin allowlist per bot **tidak wajib** (iframe same-page API, lihat §13). PDF tidak wajib.

---

## 4. Non-goal — jangan dikerjakan di V2

- Multi-tenant, org, undangan tim, billing.
- N bot per akun.
- `bot_id` di tabel `documents` / `conversations` / `messages`.
- Paket npm / SDK React / `go get` library publik (`pkg/rag`).
- PDF, URL ingest, WhatsApp/Telegram, dark mode, i18n.
- Ganti SQLite / pgvector.
- Rewrite frontend atau ganti Gin.
- Thumbs up/down, escalate, jawab manual (kecuali kolom `channel` playground di paket C — itu boleh).

Kalau suatu PR V2 menyentuh item di atas, pecah PR atau tolak dulu.

---

## 5. Keadaan kode sekarang (baseline)

Peta file STATUS-GAPS §6 masih benar. Tambahan yang **harus diubah** di V2:

| Perilaku | Lokasi | Masalah |
|---|---|---|
| Owner chat = user pertama | `service/chat_service.go` `users.First()` | Tidak ada identitas bot |
| Profil publik = setelan user pertama | `settings_service.GetPublic` | Sama |
| Chat tanpa slug | `POST /api/v1/chat`, `frontend/src/api/chat.ts` | Tidak bisa bedakan bot |
| Homepage = chat | `App.tsx` `/` → `PublicChatPage` → `ChatWindow` | Landing tidak ada |
| Admin Login di chat | `ChatWindow.tsx` header | Tidak pantas untuk pelanggan / iframe |
| User + settings tanpa bot | `user_repo.Create` | Belum ada tabel bots |
| Session chat 1 kunci | `sessionStorage` `docubot_public_chat` | Tabrakan jika nanti ada 2 slug di 1 browser |

Yang **jangan dirombak:** pipeline dokumen, chunker, embedder, cosine search, prompt RAG, SSE emitter, rate limit, JWT, analitik per `user_id`.

---

## 6. Target V2

```
Pengunjung          Owner                      Situs klien
    |                 |                              |
    v                 v                              v
   / landing     /admin/*                      iframe
    |              |  dokumen, pasang, tes           |
    |              v                                 v
    +---- /b/{slug}  <---- POST /api/v1/b/{slug}/chat --+
                |
                v
         bot.user_id → documents/chunks/settings (tetap)
```

`ChatService.Chat` menerima `Slug`, resolve `bots` → `user_id`, lalu retrieval seperti sekarang (`ListReadyWithEmbeddingsForUser`).

---

## 7. Model data

### 7.1 Tabel baru `bots`

Tambah di `backend/internal/database/migrate.go` (pola yang sama: `CREATE TABLE IF NOT EXISTS` + backfill).

```sql
CREATE TABLE IF NOT EXISTS bots (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id          INTEGER NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    slug             TEXT NOT NULL UNIQUE,
    name             TEXT NOT NULL DEFAULT 'DocuBot',
    welcome_message  TEXT NOT NULL DEFAULT 'Halo! Ada yang bisa saya bantu?',
    active           INTEGER NOT NULL DEFAULT 1,
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_bots_slug ON bots(slug);
CREATE UNIQUE INDEX IF NOT EXISTS idx_bots_user ON bots(user_id);
```

Satu user satu bot (`user_id UNIQUE`).

### 7.2 Relasi V2 (sengaja tidak diubah)

```
users 1───1 bots
users 1───n documents 1───n chunks
users 1───n conversations 1───n messages
users 1───1 settings     -- hanya knobs RAG
```

**Sumber kebenaran nama/welcome/active:** kolom di `bots`.  
**Sumber kebenaran RAG:** `settings.temperature`, `max_tokens`, `top_k`, `min_score`.

`settings.bot_name`, `welcome_message`, `bot_active` **tetap ada** di DB lama agar migrasi aman. Setelah backfill:

- Baca publik & admin identity dari `bots`.
- `PUT /settings` yang masih mengirim `bot_name` / `welcome_message` / `bot_active` **menulis ke `bots`** (dan boleh mirror ke `settings` supaya baris lama tidak menyesatkan). Jangan biarkan dua sumber yang bisa drift tanpa dual-write.

### 7.3 Backfill (wajib di `Migrate`)

Untuk setiap `users` yang belum punya `bots`:

1. Ambil `settings` jika ada; jika tidak, pakai default.
2. `slug` = slugify(`settings.bot_name` atau nama user atau `"bot"`). Jika kosong/tabrakan, `bot-{id}` lalu `bot-{id}-{n}`.
3. Insert `bots` dari nama/welcome/active settings.

Setelah backfill, setiap user punya tepat satu bot.

### 7.4 Kolom opsional paket C: `conversations.channel`

```sql
-- additive, default public
ALTER TABLE conversations ADD COLUMN channel TEXT NOT NULL DEFAULT 'public';
-- nilai: public | playground
```

Analitik dan daftar percakapan default **hanya `public`**. Playground admin memakai `channel=playground` agar log pelanggan tidak tercampur. Jika paket C ditunda, skip kolom ini dan tes admin memakai `/b/:slug` di tab baru (kurang ideal, masih DoD-able jika Install ada).

**Rekomendasi:** kerjakan kolom `channel` di paket C, bukan A, supaya A tetap kecil.

---

## 8. Spesifikasi API

Base path tetap `/api/v1`. Format error tetap `{ "error": { "code", "message" } }` (BRD §9).

### 8.1 Publik — profil bot

**`GET /api/v1/bots/:slug`**

- 200:

```json
{
  "data": {
    "slug": "toko-kita",
    "bot_name": "Asisten Toko Kita",
    "welcome_message": "Halo! Ada yang bisa saya bantu?",
    "bot_active": true,
    "configured": true,
    "has_ready_kb": true
  }
}
```

- 404 `NOT_FOUND` — slug tidak ada. Jangan kembalikan bot lain.
- `configured` di V2 hampir selalu `true` jika baris bot ada. Tetap kirim untuk kompatibilitas UI.

`register_open` **jangan** ditaruh di profil bot pelanggan (itu urusan landing/register). Landing memakai `GET /api/v1/auth/register-status` seperti sekarang.

### 8.2 Publik — chat SSE

**`POST /api/v1/b/:slug/chat`**

Body (sama seperti sekarang, plus opsional channel):

```json
{
  "conversation_id": "uuid-atau-null",
  "message": "Gimana cara reset password?",
  "channel": "public"
}
```

- `channel` default `public`. Nilai diizinkan: `public`, `playground`. `playground` boleh dipanggil tanpa JWT di V2 (disalahgunakan = log terpisah; cukup untuk slice ini). Jangan dokumentasikan `playground` di README publik.
- Rate limit: tetap 10/menit/IP (middleware yang sama).
- Event SSE: tetap `sources` → `token`* → `done` | `inactive` | `error`.
- 404 jika slug tidak ada (boleh SSE `error` code `NOT_FOUND` setelah headers stream, **atau** HTTP 404 JSON sebelum stream — pilih **HTTP 404 JSON** jika slug invalid, supaya klien embed mudah. Jika slug valid lalu bot nonaktif: stream `inactive` seperti sekarang).
- Retrieval + persist memakai `bot.user_id`.

**`POST /api/v1/chat` (lama)**

Jangan silently fallback ke user #1.

- Opsi A (disarankan): HTTP 400 `GONE` / `VALIDATION_ERROR` message: `gunakan POST /api/v1/b/{slug}/chat`.
- Tes handler lama yang memanggil `/chat` **harus diubah** ke path baru.

Jangan biarkan kedua path hidup dengan semantik berbeda.

### 8.3 Publik — pointer demo untuk landing

**`GET /api/v1/demo`**

Hanya untuk tombol “Coba demo” di `/`:

```json
{
  "data": {
    "slug": "docubot",
    "bot_name": "DocuBot",
    "has_ready_kb": true,
    "configured": true
  }
}
```

- Implementasi: bot dengan `id` / `user_id` terkecil yang ada. Jika tidak ada user: `{ "configured": false }` (200, bukan 404) agar landing tidak error.
- Ini satu-satunya tempat `ORDER BY user_id ASC LIMIT 1` masih sah, dan **bukan** untuk menjawab pertanyaan.

### 8.4 Admin (JWT)

Tetap: documents, conversations, analytics, settings, auth/me.

**Baru:**

**`GET /api/v1/admin/bot`** — bot milik user JWT.

```json
{
  "data": {
    "slug": "toko-kita",
    "name": "Asisten Toko Kita",
    "welcome_message": "...",
    "active": true,
    "public_path": "/b/toko-kita",
    "embed_path": "/b/toko-kita?embed=1"
  }
}
```

Base URL publik **tidak** di-hardcode di backend. Frontend menyusun URL dari `window.location.origin` + `public_path`.

**`PUT /api/v1/admin/bot`**

```json
{
  "slug": "toko-kita",
  "name": "Asisten Toko Kita",
  "welcome_message": "Halo!",
  "active": true
}
```

- Validasi slug §12. Jika slug berubah, embed lama putus — UI Install harus menulis peringatan.
- Dual-write identity ke `settings` (nama/welcome/active) agar `GET /settings` lama tidak dusta.

**`PUT /api/v1/settings`**

Tetap menerima body lama (BRD §9.6 + frontend `SettingsPage`). Handler:

1. Update knobs RAG di `settings`.
2. Update identity di `bots` (+ mirror settings).

**`GET /api/v1/settings`**

Join: knobs dari `settings`, `bot_name` / `welcome_message` / `bot_active` dari `bots`. Frontend SettingsPage bisa tetap.

**`GET /api/v1/bot` (lama, tanpa slug)**

Hapus atau 400 “gunakan GET /api/v1/bots/:slug”. Landing dan ChatWindow tidak boleh memakai ini.

### 8.5 Register

`POST /auth/register` tidak berubah bentuk JSON. Di dalam `UserRepo.Create` (transaksi yang sudah insert `settings`): insert `bots` dengan slug unik dari `name` user (slugify). Default name bot = nama user atau `"DocuBot"` jika nama kosong.

---

## 9. Frontend & UX

### 9.1 Rute

| Path | Halaman | Auth |
|---|---|---|
| `/` | Landing baru | tidak |
| `/b/:slug` | Chat pelanggan (`ChatWindow`) | tidak |
| `/login` `/register` | tetap | tidak |
| `/admin/documents` dll. | tetap | JWT |
| `/admin/install` | Pasang: URL + iframe snippet | JWT |
| `/admin/setup` | Onboarding sekali (opsional jika Install+Documents cukup) | JWT |

Catch-all `*` → `/` (bukan chat).

### 9.2 Landing (`/`) — konten minimum

Bukan chat penuh. Cukup:

1. Nama produk + 2–3 kalimat: bot support dari dokumen, self-host, embed iframe.
2. Tombol **Coba demo** → `/b/{slug}` dari `GET /api/v1/demo` (disabled + teks jika `configured: false`).
3. Tombol **Masuk admin** / **Daftar** (daftar hanya jika register-status open).
4. Blok “Pasang”: contoh iframe 5 baris (slug placeholder `your-bot`).
5. Satu baris stack: Go, React, SQLite, RAG.

Jangan taruh Admin Login di dalam widget demo. Demo = navigasi ke `/b/slug`.

### 9.3 `ChatWindow`

Props baru, contoh:

```ts
type ChatWindowProps = {
  slug: string
  variant: 'page' | 'embed'
}
```

- `botApi.public(slug)` → `GET /api/v1/bots/:slug`.
- `streamChat(slug, message, conversationId, ...)` → `POST /api/v1/b/:slug/chat`.
- `sessionStorage` key: `docubot_public_chat:${slug}`.
- Header: nama bot, status, tombol Chat baru. **Tanpa** Admin Login.
- `variant=embed` (`?embed=1`): header lebih rapat, `html/body` tetap full height (iframe).
- 404 slug: pesan “Bot tidak ditemukan”, tanpa form chat.

Jangan fetch `/api/v1/bot` lagi.

### 9.4 Admin — Install

Nav sidebar: **Pasang** (`/admin/install`).

Isi:

- Slug (editable + Save lewat `PUT /admin/bot`).
- URL publik (readonly, tombol salin): `{origin}/b/{slug}`.
- Snippet iframe (readonly, tombol salin), misalnya:

```html
<iframe
  src="https://CONTOH/b/SLUG?embed=1"
  title="DocuBot"
  style="width:100%;height:640px;border:0;border-radius:12px"
  loading="lazy"
></iframe>
```

Frontend mengganti `https://CONTOH` dengan `window.location.origin`.

- Peringatan: ganti slug = putus embed lama.
- Tautan “Buka chat publik” (tab baru).
- Jika paket C pakai playground: kotak chat mini di halaman yang sama, `channel: "playground"`.

### 9.5 Onboarding setelah daftar

Minimum yang memenuhi DoD:

1. Register → login (sudah ada) → **redirect ke `/admin/install`** (bukan documents), dengan banner: “1) unggah dokumen 2) tes 3) salin snippet”.
2. Empty state Documents tetap 3 langkah (sudah ada) + tautan ke Install.

Wizard `/admin/setup` terpisah **opsional**. Jangan blokir V2 jika Install + Documents empty state sudah jelas.

### 9.6 Settings

Tetap. Tambah teks kecil: “Slug dan tautan pasang ada di halaman Pasang.” Jangan duplikasi editor slug di dua tempat.

---

## 10. Paket kerja (urutan wajib)

Kerjakan **A → B → C → D**. Jangan mulai UI landing sebelum chat ber-slug (A) hijau di tes.

### Paket A — Identitas bot (backend)

**Hasil:** slug di DB; chat dan profil publik tidak memakai `users.First()`.

**File utama**

- `backend/internal/database/migrate.go`
- `backend/internal/models/bot.go` (baru)
- `backend/internal/repository/bot_repo.go` (baru) + tes
- `backend/internal/repository/user_repo.go` — Create: insert bot di TX yang sama
- `backend/internal/service/bot_service.go` (baru) — GetBySlug, GetByUser, Update, slugify
- `backend/internal/service/chat_service.go` — `ChatInput.Slug` wajib; resolve bot; error `ErrBotNotFound`
- `backend/internal/service/settings_service.go` — GetPublic(slug); Get/Update join bots; hapus GetPublic tanpa slug
- `backend/internal/handler/bot_handler.go` (baru) — public get, demo, admin get/put
- `backend/internal/handler/chat_handler.go` — path param slug
- `backend/internal/handler/settings_handler.go` — drop `GET /bot` lama
- `backend/cmd/server/main.go` — wiring
- Tes: `chat_service_test.go`, `chat_handler_test.go`, `bot_repo` / `bot_service` baru, `user_repo` create → bot ada

**Langkah**

1. DDL + backfill.
2. Repo: `GetBySlug`, `GetByUserID`, `Create`, `Update`. Unique slug collision → error yang di-map service ke slug-`-{n}`.
3. Ubah `NewChatService` agar menerima `BotRepository` (boleh tetap terima `UserRepository` jika masih perlu; **jangan** panggil `First` di `Chat`).
4. Handler chat baca `:slug`.
5. Router:

```
GET  /api/v1/bots/:slug
GET  /api/v1/demo
POST /api/v1/b/:slug/chat
GET  /api/v1/admin/bot     (JWT)
PUT  /api/v1/admin/bot     (JWT)
```

Hapus `GET /bot` dan `POST /chat` lama (atau 400, §8.2).

6. Tes §14 A.

**DoD A:** `go test ./...` hijau; curl `POST /api/v1/b/slug-salah/chat` tidak menjawab sebagai bot lain; register → row bots ada.

Frontend boleh masih rusak sampai paket B. Jangan merge ke main tanpa B jika demo live harus tetap jalan — kerjakan A+B dalam satu cabang jika perlu.

### Paket B — Permukaan publik (frontend + API client)

**Hasil:** `/` landing; `/b/:slug` chat.

**File utama**

- `frontend/src/App.tsx`
- `frontend/src/pages/LandingPage.tsx` (baru)
- `frontend/src/pages/PublicChatPage.tsx` — baca `useParams().slug`
- `frontend/src/components/chat/ChatWindow.tsx`
- `frontend/src/api/chat.ts` — path baru
- `frontend/src/api/types.ts` — `slug` di PublicBot

**Langkah**

1. Client API sesuai §8.1–8.2.
2. `PublicChatPage` + `ChatWindow` props slug; hilangkan Admin Login.
3. Landing §9.2.
4. Storage key per slug.
5. Manual: register → upload sample → buka `/b/{slug}` → tanya isi FAQ.

**DoD B:** `/` bukan bubble chat; slug salah = empty state; slug benar = SSE jalan.

### Paket C — Go live admin

**Hasil:** owner bisa salin iframe tanpa baca README.

**File utama**

- `frontend/src/pages/admin/InstallPage.tsx` (baru)
- `frontend/src/pages/admin/AdminLayout.tsx` — nav Pasang
- `frontend/src/api/admin.ts` — `adminBotApi`
- `frontend/src/pages/LoginPage.tsx` — setelah register/login, `Navigate` ke `/admin/install`
- (Opsional) playground di InstallPage + migrasi `conversations.channel`
- `backend/.../conversation_repo.go` — Create terima channel; ListByUser filter default public
- Analytics: `WHERE channel = 'public'` (atau `channel IS NULL OR channel = 'public'` setelah backfill)

**DoD C:** salin snippet dari UI, tempel ke `misc/example/embed-dummy.html`, buka file/static server, chat jalan. Playground tidak muncul di daftar Percakapan default jika kolom channel dikerjakan.

### Paket D — Kemas & berhenti

**File utama**

- `LICENSE` (MIT, nama pemegang hak sesuai repo)
- `README.md` — Quick start self-host; embed iframe; sebut slug; hapus kesan “hanya demo di `/`”
- Dokumen ini: centang DoD §3.2; versi 1.1 “implemented” setelah merge
- BRD: jangan rewrite total. Tambah 1 paragraf di BRD §17 atau catatan versi 1.3: “V2 self-host + slug + iframe — lihat PLAN-V2…” (PR terpisah boleh)
- Deploy VPS seperti README yang sudah ada; pastikan host nginx **tidak** memasang `X-Frame-Options: DENY` / `SAMEORIGIN` untuk `/b/` (lihat §13)

**DoD D:** README + dummy embed + tes hijau + DoD §3.2. **Stop fitur.**

---

## 11. Migrasi & kompatibilitas

| Skenario | Perilaku |
|---|---|
| DB kosong | Register → user + settings + bot |
| DB MVP (user + settings, tanpa bots) | `Migrate` backfill 1 bot / user |
| Frontend lama masih `POST /chat` | 400 dengan pesan path baru (putus sengaja) |
| Bookmark `/` sebagai chat | Landing; user admin memakai `/b/slug` |
| Dua user (invite/open) | Masing-masing bot + slug; chat publik tidak campur karena slug |

Tidak ada migrasi embedding. Tidak ada perubahan chunk.

---

## 12. Aturan slug

- Normalisasi: trim, lowercase, spasi → `-`, buang karakter selain `a-z0-9-`, collapse `--`.
- Panjang 3–48 setelah normalisasi.
- Tidak boleh: `admin`, `api`, `login`, `register`, `b`, `embed`, `demo`, `healthz`, `assets`, `static`.
- Unik global.
- Generate awal: slugify(nama bot atau nama user); jika hasil `< 3` atau reserved → `bot`; jika tabrakan → `bot-{userID}` lalu suffix `-2`, `-3`.
- Update slug: sama validasinya; 409 jika diambil user lain.

Tes unit slugify: `"Toko Kita!"` → `toko-kita`; `"---"` → fallback.

---

## 13. Embed iframe

Iframe memuat **halaman** DocuBot (`/b/slug?embed=1`). JavaScript di dalam iframe memanggil `/api/v1/...` **same-origin** ke host DocuBot. **CORS klien tidak diperlukan** untuk mode ini. Jangan bangun script widget `fetch` dari origin toko di V2.

Yang bisa memblokir:

- `X-Frame-Options: DENY` / `SAMEORIGIN` di nginx host atau `frontend/nginx.conf`
- CSP `frame-ancestors 'none'`

V2: **jangan set** header itu untuk lokasi SPA. `frontend/nginx.conf` saat ini tidak set X-Frame-Options — biarkan. Dokumentasikan di README deploy: jangan tambah DENY.

Allowlist origin per bot = P3 setelah V2 (perlu `Content-Security-Policy: frame-ancestors ...` dinamis, lebih rumit).

File bantu: `misc/example/embed-dummy.html` (paket C/D) — HTML statis, iframe `src` diisi origin local (`http://127.0.0.1:3000/b/SLUG?embed=1`).

Tinggi iframe default 640px; lebar 100%. ChatWindow sudah `h-full` — pastikan rute embed: `#root` / parent iframe 100% tinggi (`html, body, #root { height: 100% }` sudah ada di `index.css`).

---

## 14. Tes wajib

### 14.1 Backend (paket A, jangan diskip)

| Kasus | Harapan |
|---|---|
| Register | 1 row `bots`, slug valid unik |
| Backfill migrate | user lama tanpa bot → 1 bot |
| `GET /bots/tidak-ada` | 404 |
| Chat slug A, dokumen user A | jawaban dari KB A |
| Chat slug B (user kedua, jika tes buat 2 user via invite/open) | tidak memakai chunk user A |
| Chat slug salah | 404, tidak memanggil LLM |
| `GET /demo` tanpa user | 200 `configured: false` |
| `GET /demo` dengan user | slug bot tertua |
| `POST /chat` lama | 400 (jika opsi A) |
| Bot `active=0` | event `inactive`, tanpa LLM |
| Dual-write settings | PUT settings ubah nama → GET bots slug menampilkan nama baru |

Perbarui `setupFullRouter` / `newChatStack`: inject `BotRepo`; tes HTTP memakai `/api/v1/b/{slug}/chat`.

### 14.2 Manual (paket B–D)

Checklist pengganti E2E (frontend tes tetap P2, STATUS-GAPS T11):

1. Register first admin.
2. Upload `faq-contoh.md` / testdata, status Ready.
3. `/` landing, coba demo → `/b/{slug}`.
4. Tanya yang ada di dokumen → sumber tampil.
5. `/admin/install` salin iframe → dummy HTML → pertanyaan sama.
6. Refresh tab chat: sesi persist (per slug).
7. Mobile: landing + chat `/b/:slug`.

---

## 15. Risiko

| Risiko | Mitigasi |
|---|---|
| Dual-write `settings` vs `bots` drift | Semua tulis identity lewat BotService; GET settings join bots |
| `GET /demo` = First() disalahartikan | Komentar di handler + tes bahwa Chat tidak memanggil First |
| Putus API `/chat` untuk bookmark/demo lama | README + 400 yang jelas; V2 belum janji stabil API publik |
| Iframe diblokir nginx host | Cek header di paket D; dokumentasikan |
| Scope creep landing | Konten minimum §9.2, bukan situs marketing |
| Playground tanpa JWT disalahgunakan | Channel terpisah; rate limit tetap; README tidak sebutkan |

---

## 16. Setelah V2 (bukan sekarang)

Urutan **hanya jika** V2 live dan ada permintaan nyata:

1. PDF teks / tempel teks (knowledge nyata).
2. N bot per akun → baru saat itu `bot_id` di documents & conversations.
3. `frame-ancestors` / origin allowlist.
4. Sinyal kualitas percakapan (thumbs, unanswered).
5. Ekstrak `internal/ai` ke `pkg/rag` **jika** ada yang benar-benar import.
6. Hosted multi-tenant, npm widget, channel WhatsApp — paling akhir.

Jangan masuk backlog sprint V2.

---

## 17. Riwayat dokumen

| Versi | Tanggal | Isi |
|---|---|---|
| 1.0 | 23 Agu 2026 | Perkembangan pasca-MVP + plan implementasi V2 (self-host, slug, landing, iframe). Acuan: BRD v1.2, STATUS-GAPS v1.1. Kode belum diubah. |
| 1.1 | 23 Agu 2026 | Implemented: tabel `bots`, chat `/b/:slug`, landing, Pasang + iframe, LICENSE MIT. DoD §3.2 dicentang kecuali rekaman demo. |

Setelah implementasi merge: naikkan ke **1.1**, centang §3.2, catat commit/hash singkat.

---

*Implementasi mengikuti paket A→D di §10. Jangan mulai dari UI atau LICENSE sebelum A membuat chat ber-slug.*
