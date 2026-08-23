# Prompt Hermes — deploy DocuBot live (VPS)

Tempel blok **Prompt untuk Hermes** di bawah ke agent di VPS. File ini salinan yang sama agar setelah `git clone` / `git pull` agent bisa merujuk ulang tanpa mengulang chat.

Jangan menaruh API key, password SSH, atau `JWT_SECRET` di chat. Kunci hanya di `.env` di server (tidak di-commit).

---

## Prompt untuk Hermes

```text
Kamu agent di VPS. Tugas: bawa DocuBot sampai LIVE dan bench nyata. Jangan menambah fitur produk.

Repo: https://github.com/anandaadityap/docubot.git
Branch: main
Produk: self-host 1 akun = 1 bot (Go + React + SQLite + RAG). Bukan SaaS, bukan npm widget.

============================================================
FASE 0 — repo
============================================================
Clone atau pull latest main. Kerja di direktori repo. Jangan commit .env, data/, dist/, tmp/.

============================================================
FASE 1 — BACA DULU (wajib berurutan, jangan skip, jangan coding dulu)
============================================================
Baca file berikut SATU PER SATU, selesai satu baru lanjut. Catat konflik: dokumen mana yang menang.

1) misc/README.md
   Peta folder misc. Bukan pitch publik.

2) misc/BRD-PRD-docubot-ai-support-bot.md
   Pakai: RAG, auth, FR/NFR, skema/API MVP, prinsip, deploy Docker.
   Jangan pakai: tabel milestone §11 (basi). Jangan salin target penghasilan / strategi bid Upwork ke README publik.

3) misc/STATUS-GAPS-docubot-ai-support-bot.md
   Pakai: gap MVP yang SUDAH ditutup, peta file RAG/SSE.
   Jangan pakai: sebagai backlog fitur baru. §5 bukan antrian aktif.

4) misc/PLAN-V2-docubot-selfhost-embed.md
   Pakai: perilaku V2 (slug, landing /, chat /b/:slug, iframe, nginx jangan X-Frame-Options DENY/SAMEORIGIN).
   Jika bentrok dengan BRD soal embed: PLAN-V2 yang diikuti.
   Jangan: PDF, WhatsApp, N-bot, npm widget, i18n sekarang.

5) misc/CONTEXT-OSS-PORTFOLIO.md  ← STATUS KERJA SEKARANG
   Ini sumber kebenaran “apa yang dikerjakan”.
   Selesai di repo: K4–K14 (CI, README, SECURITY, landing, dll.).
   Tugasmu HANYA yang masih terbuka di papan:
     K1 demo live
     K2 screenshot/rekaman (jika bisa di server; kalau tidak, siapkan URL + checklist untuk manusia)
     K3 bench DeepSeek nyata (bukan StubLLM) + satu baris angka di README
     K14b tag v0.2.0 opsional setelah live stabil
   JANGAN: K17 i18n, K15 dwibahasa UI, K16 tes frontend, fitur di §5 Non-goal.
   Bahasa UI tetap Indonesia. Jangan campur string Inggris di UI.

6) README.md (root) + SECURITY.md + .env.example + docker-compose.yml
   Perintah deploy, healthz, env, iframe, bench scripts.

7) misc/example/embed-dummy.html
   Tes iframe lokal/produksi (ganti SLUG). Jangan baca contoh FAQ toko di misc/example kecuali perlu ingest tes (pakai frontend/public/samples/faq-contoh.md).

Setelah FASE 1, balas ringkas: (a) domain yang akan dipakai, (b) apakah Docker/nginx/certbot sudah ada, (c) rencana 8–12 langkah. Baru FASE 2.

============================================================
FASE 2 — implementasi (K1 → K3)
============================================================
Aturan:
- Kunci LLM/embed/JWT HANYA di .env di VPS. Jangan echo secret ke log chat.
- Production: APP_ENV=production, JWT_SECRET kuat (bukan change-me-please). App menolak JWT default di production.
- REGISTER_MODE=first-only kecuali owner minta lain.
- CORS_ORIGINS harus berisi origin HTTPS publik (https://DOMAIN).
- LLM: DeepSeek (LLM_BASE_URL + LLM_API_KEY + LLM_MODEL=deepseek-chat) sesuai .env.example.
- Embed: OpenAI-compatible `/embeddings`. Live memakai OpenRouter:
  EMBED_BASE_URL=https://openrouter.ai/api/v1
  EMBED_MODEL=nvidia/nemotron-3-embed-1b:free
  EMBED_API_KEY dari owner (sk-or-..., jangan echo). Setelah ganti embed model/dim, POST /api/v1/documents/{id}/reprocess sampai ready. Tanpa key nyata server pakai stub 64-d — bench K3 tidak valid.
- docker compose: frontend bind 127.0.0.1:3000. Nginx host reverse-proxy ke situ + SSL.
- SSE: proxy_buffering off; proxy_read_timeout ≥ 75s.
- JANGAN set X-Frame-Options DENY/SAMEORIGIN atau CSP frame-ancestors 'none' pada / atau /b/ (iframe harus jalan).
- Health: http://127.0.0.1:3000/healthz dan/atau backend :8080/healthz.
- Volume data/ untuk SQLite + uploads.

K1 selesai jika:
- HTTPS buka landing /
- /b/{slug} chat dari KB (setelah register admin pertama + unggah sample FAQ sampai ready)
- iframe (embed=1) tidak kosong/diblokir
- README menyebut URL live yang benar (ganti “not claimed live yet”)

K3 selesai jika:
- scripts/bench.sh atau bench.ps1 dijalankan ke BASE_URL API live (bukan StubLLM)
- satu baris README: ttft_ms / latency / token / estimasi USD, tanggal, model DeepSeek
- Target NFR: TTFT < 2s setelah dokumen ready (catat angka aktual meski di atas target)

K2: rekam atau screenshot jika lingkungan memungkinkan. Jika agent tidak bisa rekam GUI, tulis checklist manual 6 langkah + URL; jangan memblokir K1/K3.

K14b: git tag v0.2.0 hanya jika owner setuju dan working tree bersih; GitHub Release boleh ditunda sampai K2 ada.

Commit: boleh update README (URL + bench). Jangan commit .env. Jangan force-push. Jangan rewrite history.

============================================================
FASE 3 — laporan
============================================================
Kirim ke owner:
- URL HTTPS landing + contoh /b/{slug}
- Apakah iframe OK
- Angka bench (ttft_ms, latency, tokens, USD)
- File yang diubah
- Yang belum (K2 jika tidak bisa rekam)
- Perintah yang owner harus jalankan sendiri jika ada (DNS, key yang belum diisi)

STOP setelah K1+K3 (dan K2 sebisanya). Tidak ada fitur baru.
```
