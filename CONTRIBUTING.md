# Contributing

## Dev setup

- Go 1.22+
- Node 20+ (or Docker for the web build)
- Docker for a full stack

Backend:

```bash
SESSION_SECRET=$(openssl rand -hex 32) SEED_USERNAME=admin SEED_PASSWORD=changeme-at-least-12 go run ./cmd/server
```

Frontend (Vite proxies `/api` to `localhost:9080`):

```bash
cd web && npm ci && npm run dev
```

UI mock with fake data (DEV only): `http://localhost:5173/__ui`. It is not included in the production binary.

## Checks

```bash
./scripts/regression-gate.sh
# backend only:
SKIP_WEB_BUILD=1 ./scripts/regression-gate.sh
```

`go test ./...`, `go vet ./...`, and `npm ci && npm test && npm run build` in `web/` should stay green.

## Code layout

- Go: **Handler → Service → Repository**. `cmd/server` wires adapters. Do not put SQL in HTTP handlers.
- React: feature folders under `web/src/features/<name>/`. Pages stay thin.
- **Do not change `/api` JSON, status codes, or paths** unless the change is the point of the PR.

Commits: short imperative subject (what/why). No secrets, `.env`, or real letters.

See [docs/regression-gates.md](docs/regression-gates.md) and [docs/smoke-checklist.md](docs/smoke-checklist.md).
