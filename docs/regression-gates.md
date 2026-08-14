# Regression gates (Sonix)

This document defines **what must pass** before merge or release, and how it is **enforced**. Large-scale ML evaluation harnesses are out of scope per project direction; gates combine **automated checks** + **manual smoke** + **operator judgment** for extraction quality.

## Tier 1 — Automated (required)

| Check | Command | Notes |
|--------|---------|--------|
| Go tests | `go test ./...` | Locks SQL contracts, ollama helpers, extraction wiring, OCR factory. |
| Go vet | `go vet ./...` | Static analysis. |
| Web production build | `npm ci && npm run build` in `web/` | Typecheck + Vite. Use Docker if Node is not local (see script). |

**Single entrypoint:** [`scripts/regression-gate.sh`](../scripts/regression-gate.sh) runs Tier 1 end-to-end when Docker is available. Use `SKIP_WEB_BUILD=1 ./scripts/regression-gate.sh` for backend-only verification.

## Tier 2 — Manual smoke (required for UI-facing changes)

Follow [smoke-checklist.md](smoke-checklist.md) against a running server (local or Compose). Covers auth, shell, lists, add flows, document detail, search, settings.

## Tier 3 — Quality / performance (operator / release)

These are **not** automated in-repo today; track them when changing prompts, models, or infrastructure:

| Area | What to watch | How |
|------|----------------|-----|
| Extraction success | Documents reaching `ready` vs `failed` for representative scans | Logs + DB / UI |
| Latency | Time from extract click to ready on typical docs | Rough timing; optional log timestamps |
| Summary / date | Plausible summaries, ISO dates when model cooperates | Spot-check documents; `prompt_version` / `engine_id` on `extractions` for attribution |
| Search | FTS returns expected hits for known strings | Manual search from My letters (search box + filters) |

Tighten thresholds when you have a fixed **golden set** of documents and a team agreement; until then, regressions are caught by tests + smoke + spot checks.

## Phased rollout (already in product)

- **Frontend:** Unified **four-destination shell** (My letters / Explore / Scan letters / Settings); no `newUX` / `VITE_NEW_UX` runtime branching in the app (legacy localStorage key `sonix_feature_newUx` is harmless if present).
- **Extraction:** default LLM vision (`/api/chat` per page, then metadata); OCR opt-in; pipeline strategy logged per document (`two_phase_ocr` | `two_phase_vision`). See [README.md](../README.md) extraction section.

## Troubleshooting

- **`npm run build` / `tsc` errors in `web/`:** Run `npm ci` in `web/` so `@types/react` and friends are installed. The project intentionally does **not** ship stub `react.d.ts` files in `src/` (those broke strict checking when real types were present).

## Related

- [architecture.md](architecture.md) — stack and layering.
- [extraction.md](extraction.md) — extraction flow reference.
