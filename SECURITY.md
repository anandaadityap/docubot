# Security

## Secrets

- Put API keys and `JWT_SECRET` only in server env (`.env` is gitignored). Never send them to the browser.
- Production refuses a default `JWT_SECRET`. Use a long random value.
- Do not commit `.env`, database files, or upload directories.

## Auth and chat

- Admin JWT is stored in **localStorage** (XSS on the admin origin can steal it). Treat the admin UI like a trusted origin.
- Public chat is unauthenticated, rate-limited per IP (chat 10/min, auth 5/min).
- Default register is first-user-only; keep `REGISTER_MODE=first-only` (or `closed`) on a public VPS.

## Embed

Iframe embed is **same-origin to DocuBot**. The shop site does not call the API directly, so shop CORS is not required.

This version does **not** set `Content-Security-Policy: frame-ancestors` per bot. Any site can iframe `/b/{slug}?embed=1` if your reverse proxy does not send `X-Frame-Options: DENY` / `SAMEORIGIN`. Do not add those headers if you need embed. An origin allowlist is a later, optional hardening step.

## Reports

Open a private GitHub security advisory if the repository is public, or email the maintainer listed on the GitHub profile. Do not file a public issue for unreleased vulnerabilities.
