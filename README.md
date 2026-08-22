# DocuBot

AI support bot with a knowledge base (RAG) — Go (Gin) + React + SQLite + DeepSeek/OpenAI.

Live target: `https://chatbot.supernand.tech` (Docker Compose + host nginx).

## What it does

Visitors ask questions on the public chat page. The bot retrieves relevant chunks from uploaded documents, streams an answer (SSE), and cites sources `[1] [2]`. Admins upload `.md`/`.txt`, review conversations, and watch 14-day analytics.

## Quick start

```bash
cp .env.example .env
# Dummy keys are OK locally: StubEmbedder + StubLLM still reach status=ready and answer extractively.

docker compose up --build
# Frontend: http://127.0.0.1:3000
# Health:   http://127.0.0.1:3000/healthz
```

1. Open `/register`, create the admin account (the **first** account owns the public bot).
2. Upload `backend/testdata/manual-pengguna.md` on **Dokumen**.
3. Wait until status is `ready`, then chat on `/`.

### Local development (without Docker)

**Backend**

Install Air once (`go install github.com/air-verse/air@latest`), then:

```bash
cd backend
# PowerShell
$env:DATABASE_PATH="..\data\docubot.db"
$env:UPLOAD_DIR="..\data\uploads"
air
```

Air watches `.go` files under `cmd/` and `internal/` and rebuilds the API automatically. Without live reload: `go run ./cmd/server`.

If `EMBED_API_KEY` / `LLM_API_KEY` are empty, the server uses deterministic stub embeddings and an extractive StubLLM (no external API).

**Frontend**

```bash
cd frontend
npm install
npm run dev
# Vite proxies /api and /healthz → localhost:8080
```

## Tests

```bash
cd backend && go test ./...
```

On some Windows setups, WDAC blocks `handler.test.exe` in the Go cache. Compile with another name:

```bash
cd backend
go test -o tmp/htest.exe ./internal/handler
```

## Auth API

```bash
curl -s -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"name":"Nanda","email":"nanda@example.com","password":"secret123"}'

curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"nanda@example.com","password":"secret123"}'
```

## Documents API

All routes need `Authorization: Bearer <token>`.

```bash
curl -s -X POST http://localhost:8080/api/v1/documents \
  -H "Authorization: Bearer <token>" \
  -F "file=@backend/testdata/manual-pengguna.md"
```

## Chat API (public, SSE)

```bash
curl -N -X POST http://localhost:8080/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{"message":"Gimana cara reset password?"}'
```

Events: `sources` → `token`* → `done` (or `inactive` / `error`). Rate limit: 10 requests/minute/IP.

Public bot profile (no JWT): `GET /api/v1/bot`.

## Admin API

- `GET/PUT /api/v1/settings`
- `GET /api/v1/conversations?page=1&limit=20`
- `GET /api/v1/conversations/:id`
- `GET /api/v1/analytics/overview`
- `GET /api/v1/analytics/top-questions?limit=10`

## Benchmark

With the API running (after a document is `ready`):

```bash
# Git Bash / Linux / macOS
BASE_URL=http://127.0.0.1:8080 ./scripts/bench.sh

# PowerShell
.\scripts\bench.ps1 -BaseUrl http://127.0.0.1:8080
```

The script prints wall time, server `latency_ms`, token usage, and an estimated USD cost using DeepSeek-chat list prices (override `OUT_PER_M` / `-OutPerMillion`). Record a real run in this README after deploy.

**Sample local run (StubLLM, 22 Agu 2026):** wall time and cost depend on your machine; with real DeepSeek keys, target time-to-first-token &lt; 2s.

## Deploy (VPS)

1. `docker compose up -d --build`
2. DNS A record `chatbot.supernand.tech` → VPS IP
3. Host nginx reverse-proxy to `127.0.0.1:3000` + Certbot SSL
4. Set `JWT_SECRET`, `LLM_API_KEY`, `EMBED_API_KEY`, and `CORS_ORIGINS` in `.env`

Example host nginx (SSE buffering off):

```nginx
server {
    server_name chatbot.supernand.tech;
    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_buffering off;
        proxy_read_timeout 120s;
    }
}
```

## Layout

- `backend/` — Go API (Gin + SQLite + RAG)
- `frontend/` — React (Vite + Tailwind)
- `data/` — SQLite DB + uploads (runtime, gitignored)
- `scripts/` — benchmark
- `misc/` — BRD/PRD

## Stack

Go, Gin, SQLite, React, Vite, Tailwind, Recharts, DeepSeek (chat) / OpenAI-compatible embeddings, Docker.
