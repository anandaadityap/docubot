# DocuBot web

React (Vite + Tailwind) UI: landing, public chat `/b/:slug`, admin.

Root [README](../README.md) has clone, Docker, and API docs.

```bash
npm install
npm run dev
```

Vite proxies `/api` and `/healthz` to `http://localhost:8080`. The production image serves `dist/` via nginx (`nginx.conf`) and proxies `/api/` plus `/healthz` to the Go service.
