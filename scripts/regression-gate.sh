#!/usr/bin/env bash
# Automated regression gate: Go tests + vet; optional web production build via Docker.
# Usage: from repo root — ./scripts/regression-gate.sh
# Env: SKIP_WEB_BUILD=1  — skip the Node/Vite step (CI without Docker or quick backend-only check).
# Env: SKIP_WEB_TEST=1   — run web build only (skip Vitest) after npm ci.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "== go test ./... =="
go test ./...

echo "== go vet ./... =="
go vet ./...

if [ "${SKIP_WEB_BUILD:-}" = "1" ]; then
  echo "== SKIP web build (SKIP_WEB_BUILD=1) =="
  exit 0
fi

if command -v docker >/dev/null 2>&1; then
  echo "== web: npm ci + production build (node:20-alpine) =="
  if [ "${SKIP_WEB_TEST:-}" = "1" ]; then
    docker run --rm -v "$ROOT/web:/app" -w /app node:20-alpine sh -c "npm ci && npm run build"
  else
    echo "== web: npm run test (Vitest) =="
    docker run --rm -v "$ROOT/web:/app" -w /app node:20-alpine sh -c "npm ci && npm run test && npm run build"
  fi
else
  echo "WARN: docker not found — skipping web build. Install Docker or run manually:" >&2
  echo "  cd web && npm ci && npm run build" >&2
  echo "See docs/regression-gates.md" >&2
  exit 1
fi

echo "OK — all regression gate steps passed."
