# Contributing

DocuBot is a **self-host app** (one admin, one bot). Packaging: [`misc/CONTEXT-OSS-PORTFOLIO.md`](./misc/CONTEXT-OSS-PORTFOLIO.md). Product behavior: [`misc/PLAN-V2-docubot-selfhost-embed.md`](./misc/PLAN-V2-docubot-selfhost-embed.md).

## Run locally

```bash
cp .env.example .env
# Placeholder LLM_API_KEY / EMBED_API_KEY → StubLLM + stub embeddings.
```

Backend (`cd backend`): `go run ./cmd/server` or Air. Frontend (`cd frontend`): `npm install && npm run dev`.

Or `docker compose up --build` from the repo root.

## Tests

```bash
cd backend && go test ./...
```

Do not call live LLM APIs in tests. Frontend tests are not required unless you are fixing a UI regression (SSE parse / slug 404).

## PRs

- Keep PRs small. Packaging work must not add PDF ingest, WhatsApp, N-bots, or an npm widget.
- Do not commit `.env`, `data/`, `frontend/dist/`, `backend/tmp/`, or SQLite WAL files.
- Do not copy income targets or bid strategy from `misc/BRD-*.md` into public README copy.
