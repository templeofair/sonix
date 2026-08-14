# Architecture

Self-hosted **document scanner and analysis**: scan or upload pages, run OCR and/or Ollama vision, then produce searchable text, an English translation, a short summary, and a document date.

- **Backend:** Go (`github.com/templeofair/sonix`), stdlib `net/http`, SQLite via `modernc.org/sqlite` (no CGO), Argon2id passwords, HMAC-signed session cookies. One process serves REST `/api` and the SPA (`embed.FS`).
- **Frontend:** Vite + React 18 (TypeScript, React Router, Tailwind), built into the Go binary. Inter is self-hosted via `@fontsource/inter`.
- **Deploy:** Docker Compose; `sonix-data` volume; HTTP **9080** / HTTPS **9443**; optional hp-scan profile. Self-signed HTTPS for the phone camera. Operator detail: [README.md](../README.md), [deployment.md](deployment.md).

## Layers

**Handler → Service → Repository** on the Go side; **feature folders** on the React side. Public `/api` stays stable unless a change is explicit.

```
cmd/server          → composition root (config, DB, wire-up, listeners)
internal/handler    → HTTP (auth)
internal/service    → use cases (auth, documents, extraction, settings, export, inbox)
internal/repository → SQLite
internal/server     → routes + remaining HTTP handlers
internal/ocr        → OCR adapter
internal/ollama     → LLM client + prompts
web/src/features/*  → documents, settings, auth
web/src/shared/components → cross-feature UI
web/src/pages/*     → thin route shells
```

Primary UI destinations (order): **My letters** / **Explore** / **Scan** / **Settings**. Mobile Scan tab label is **Scan** (desktop sidebar: **Scan letters**). Route map: [information-architecture.md](information-architecture.md).

## Data

One SQLite file (`DATA_DIR/sonix.db`): users, sessions, documents, pages, extractions, settings; FTS5 for search. Images under `DATA_DIR/uploads`, thumbnails under `DATA_DIR/thumbs`, TLS under `DATA_DIR/tls`, inbox under `DATA_DIR/inbox`. Schema and small in-code migrations live in `internal/database` — no separate migrate tool.

Documents have **no `user_id`**. Any valid session can access every letter.

## Docs

| Doc | Role |
|-----|------|
| [README.md](../README.md) | Install and limitations |
| [configuration.md](configuration.md) | Env, Settings, extract body |
| [extraction.md](extraction.md) | OCR / Ollama pipeline |
| [deployment.md](deployment.md) | Compose, backup, host network |
| [api.md](api.md) | REST routes |
| [SECURITY.md](../SECURITY.md) | Threat model |
