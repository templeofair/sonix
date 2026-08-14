# Configuration (env, Settings, extract toggles)

**Who this is for:** Anyone changing behavior via API, environment, or Settings.

**In one sentence:** This table lists every switch that changes how Sonix extracts documents, talks to Ollama, or shapes document list/detail payloads.

---

## Request body (`POST /api/documents/:id/extract`)

| Toggle | Type | Default | Plain English | Technical | LLM impact (rough) |
|--------|------|---------|---------------|-----------|---------------------|
| `use_ocr` | JSON bool | `false` | When **true**, use **Tesseract** for page text instead of the vision model. | Sets `useOCR` in `handleExtract`; passed to `ExtractionService.Start` / `RunExtraction`. | **Lower** vision load; **one** text call for metadata (and maybe translate-only retry). No per-page vision. |
| `ignore_ocr` | JSON bool | `false` | **Compatibility:** if `true`, forces `use_ocr` off so the client gets the **default LLM vision** path. | `if body.IgnoreOCR { useOCR = false }` | Same as not sending `use_ocr`. |

**Grep:** `use_ocr`, `IgnoreOCR`, `useOCR` in [`internal/server/extract.go`](../internal/server/extract.go); pipeline in [`internal/service/extraction.go`](../internal/service/extraction.go).

---

## Environment variables (backend)

| Variable | Default | Plain English | Technical | LLM impact |
|----------|---------|---------------|-----------|------------|
| `OLLAMA_BASE_URL` | `http://localhost:11434` | Where the Ollama HTTP API lives. http(s) only; credentials and link-local/metadata addresses are rejected. Localhost, Docker host, and LAN IPs are allowed. | Overridable from app **Settings** (DB); validated by `ValidateOllamaURL`; `ExtractionService.buildOllamaClient` reads Settings then env. | N/A (connectivity). |
| `OLLAMA_VISION_MODEL` | `llava` | Model used for **image** requests (and same slot as text unless you split in config). | Passed to `ollama.NewClient` as both vision and text model names from config. | Larger / slower models increase latency per vision call. |
| `OLLAMA_TEXT_MODEL` | `llama3.2` | Same as vision in practice; drives **chat** JSON for metadata and translate-only retry. | See `Config.OllamaText` in [`internal/config/config.go`](../internal/config/config.go). | Metadata and translate steps scale with model size. |
| `OLLAMA_TIMEOUT_MINUTES` | `15` | HTTP client timeout for each Ollama request. | [`internal/ollama/client.go`](../internal/ollama/client.go) `NewClient`. | Prevents hung jobs; too low causes false failures on slow CPUs. |
| `EXTRACTION_JOB_TIMEOUT_MINUTES` | `60` | Wall-clock budget for **one document** job (all pages + text pipeline). | [`ExtractionService.Start`](../internal/service/extraction.go) wraps the job in `context.WithTimeout`. | Independent of per-call `OLLAMA_TIMEOUT_MINUTES`; raise for long multi-page letters. |
| `EXTRACTION_MAX_CONCURRENT` | `1` | How many document extraction jobs may run at once. Further Extract clicks get HTTP 429 until a slot is free. | [`ExtractionService.Start`](../internal/service/extraction.go) | Keeps Ollama from running several letters at once on a small CPU. |
| `DOCUMENT_MAX_PAGES` | `50` | Max pages on one letter (image uploads and PDF conversion). | [`DocumentService.UploadPages`](../internal/service/documents.go) | Caps vision/OCR cost per letter. |
| `PDF_CONVERT_TIMEOUT_SECONDS` | `120` | Timeout for `pdftoppm` when turning a PDF into page images. | [`convertPDFToImages`](../internal/service/page_storage.go) | No LLM. |
| `EXTRACTION_PIPELINE` | `v2` | Text/metadata path. `v2` = page-wise translate + summary + document date; `v1` = legacy single `ExtractMetadata` call. | [`ExtractionService.useLegacyPipeline`](../internal/service/extraction.go). | Instant A/B rollback if v2 misbehaves. |
| `OLLAMA_VISION_MAX_EDGE` | `2048` | Max long-edge px for images sent to vision (stored uploads unchanged). Default ~175 DPI on A4 (was 1536 / ~130 DPI). | [`ImageToBase64ForVision`](../internal/ollama/image.go) — Catmull-Rom downscale. | Lower = fewer tokens / faster; too low hurts small-print OCR-via-vision. |
| `OLLAMA_VISION_ENDPOINT` | `/api/chat` | Which endpoint per-page vision posts to. Only `/api/generate` is accepted as an override. | [`visionEndpoint`](../internal/ollama/options.go). | `/api/chat` applies the model's renderer template; on `/api/generate` a `RENDERER`-based instruct model can continue the document instead of transcribing it ([ollama#14793](https://github.com/ollama/ollama/issues/14793)). Override exists to A/B the two. |
| `OLLAMA_NUM_CTX` | `8192` | `options.num_ctx` for **vision** calls (avoids inheriting a model's 4096). | [`visionOptions`](../internal/ollama/options.go). | Higher uses more memory; pair with downscale. |
| `OLLAMA_NUM_PREDICT_VISION` | `4096` | Output cap for one page transcription. | [`visionOptions`](../internal/ollama/options.go). | Turbo's Modelfile says 2048, which truncates dense pages. Truncation now logs `done_reason=length` instead of passing silently. |
| `OLLAMA_NUM_CTX_TEXT` | `32768` | **Ceiling** for text-call `num_ctx`; the actual value is sized from input length. | [`textNumCtx`](../internal/ollama/options.go). | Raise for long multi-page documents; costs KV memory. |
| `OLLAMA_NUM_CTX_TEXT_FLOOR` | `8192` | Floor for text-call `num_ctx`. | [`textNumCtx`](../internal/ollama/options.go). | Keeps short documents off a model's 4096 default. |
| `OLLAMA_NUM_PREDICT_TEXT` | `32768` | Ceiling for text-call `num_predict` (sized from input). | [`textNumPredict`](../internal/ollama/options.go). | A translation is roughly as long as its source, so this is a runaway guard, not a trim. |
| `OLLAMA_REPEAT_PENALTY` | `1.1` | Repetition penalty for every call. | [`baseOptions`](../internal/ollama/options.go). | Turbo ships 1.5, which cannot prevent page-length repeats (the penalty window is `repeat_last_n`, 64 by default) but does make a model paraphrase legitimately repeated wording. |
| `OLLAMA_NUM_THREAD` | (unset → Ollama decides) | CPU threads per request. | [`baseOptions`](../internal/ollama/options.go). | Ollama defaults to performance cores only ([ollama#6264](https://github.com/ollama/ollama/pull/6264)), so a 14-core hybrid CPU sits near 22%. That is often correct — generation is memory-bandwidth bound — so sweep before raising. |
| `OCR_ENGINE` | (empty → tesseract) | Which OCR **provider** when `use_ocr` is true. | `Config.OCREngine`; `ocr.NewProviderFromConfig` / tesseract default. | No LLM. |
| `OCR_LANG` | `deu+eng` | Tesseract language(s) for German letters that may contain English product names. | `Config.OCRLang`; passed as `-l`; startup validates against `tesseract --list-langs` and falls back (e.g. to `eng`) if packs are missing. New `engine_id` values look like `tesseract:deu+eng`. | No LLM. |
| `OCR_DPI` | *(auto)* | When unset/`0`, `--dpi` is estimated from the page’s long pixel edge assuming A4. Set a positive value to force a fixed hint. | `Config.OCRDPI` → [`ResolveDPI`](../internal/ocr/dpi.go). | No LLM. |
| `OCR_PSM` | `1` | Page segmentation mode. `1` = auto + OSD (orientation), important for rotated phone captures. | `Config.OCRPSM`; passed as `--psm`. Try `3` (default auto) or `6` (single block) when measuring. | No LLM. |
| `SESSION_SECRET` | *(required, no default)* | HMAC key for the session cookie. Generate with `openssl rand -hex 32`. Placeholders like `change-me-in-production` are rejected at startup. Changing it signs out everyone. | [`auth.CookieMAC`](../internal/auth/cookie.go), [`cmd/server/main.go`](../cmd/server/main.go) | N/A |
| `SEED_USERNAME` / `SEED_PASSWORD` | empty (no seed) | First-run login only. Creates the user if the users table is empty. Password must be at least 12 characters when seeding. Changing these later does **not** update an existing user. Set both or neither. | [`SeedUserIfEmpty`](../internal/service/auth.go), [`cmd/server/main.go`](../cmd/server/main.go) | N/A |
| `DATA_DIR`, `DATABASE_PATH`, `SERVER_ADDR`, `HTTPS_ADDR` | HTTP default `:9080`, HTTPS default `:9443` | Paths and bind addresses. Defaults avoid 8080 and Ollama (11434). | [`internal/config/config.go`](../internal/config/config.go), [`cmd/server/main.go`](../cmd/server/main.go) | N/A |
| `IMPORT_INBOX_DIR` | `$DATA_DIR/inbox` | Folder watched for hardware/manual scan drops. | [`InboxImporter`](../internal/service/import_inbox.go) | N/A |
| `IMPORT_INBOX_ENABLED` | `false` | Env default for inbox import; Settings can override. | `Config.ImportInboxEnabledDefault`; `SettingsService.ImportInboxEnabled` | May start extract jobs if auto-extract on |
| `IMPORT_AUTO_EXTRACT` | `true` | Start extraction after a successful inbox import. | `SettingsService.ImportAutoExtract` | Same as manual Extract |

**Grep:** `OLLAMA_`, `OCR_ENGINE`, `OCR_LANG`, `OCR_DPI`, `OCR_PSM`, `EXTRACTION_`, `DOCUMENT_MAX_PAGES`, `PDF_CONVERT_`, `IMPORT_INBOX` in `internal/config`, `internal/ocr`, `internal/service`, `internal/ollama`.

---

## Document list and detail (query parameters)

Additive list/detail options used by the library UI. Defaults preserve previous behaviour.

| Param / field | Where | Default | Plain English | Technical |
|---------------|-------|---------|---------------|-----------|
| `sort` | `GET /api/documents` | `created_desc` | Order of results. | `created_desc` (upload time, legacy), `date_desc` / `date_asc` (extraction `document_date`). Unknown values fall back to `created_desc`. |
| `status` | `GET /api/documents` | omitted | Filter by one or more statuses (OR). | Comma-separated → `d.status IN (...)`. Empty = no status filter. |
| `tag` | `GET /api/documents` | omitted | Filter by one or more manual tags (OR). | Comma-separated; `json_each(tags)` value `IN (...)`. |
| `year` | `GET /api/documents` | omitted | Filter by upload year(s) (OR). | Comma-separated → `strftime('%Y', created_at) IN (...)`. Distinct from letter-date Explore years. |
| Document tags list | `GET /api/documents/tags` | — | Distinct tags for the library Tags combobox. | `{ "tags": ["bank", "invoice", …] }` sorted case-insensitive. |
| `undated` | `GET /api/documents` | omitted / `0` | Restrict to letters with no extracted/entered document date (Explore **No date**). | `undated=1` → `document_date` NULL or `''`. Absent or `0` leaves the list query unchanged. |
| Document-date years | `GET /api/documents/document-date-years` | — | Year folders for Explore (letter date, not import date). | `{ "years": [{ "year": "2024", "count": 37 }, …], "undated_count": 4 }`; grouped on `strftime('%Y', document_date)`, newest first. Separate from legacy `GET /api/documents/years` (import year). |
| `total` | list JSON | — | How many rows match the filters (for Load more). | Sibling of `documents[]`; not a query param. |
| `page_count` | list/detail JSON | — | Number of pages on the document. | From `COUNT(*)` on `document_pages`. |
| `thumbnail_available` | list/detail JSON | — | Whether page 0 can be thumbnailed. | True when at least one page exists. |
| Thumbnail route | `GET /api/documents/{id}/pages/{pageIndex}/thumbnail` | — | Small JPEG for cards/grids. | On-demand resize to **320px** width; cached under `DATA_DIR/thumbs/{id}/{index}.jpg`; falls back to full image on failure. |
| Rotate page | `POST /api/documents/{id}/pages/{pageIndex}/rotate` | `{ "degrees": 90 \| 180 \| 270 }` | Clockwise rewrite of the stored page image. | Invalidates thumb cache; body `degrees` required. |
| `include=text` | `GET /api/documents/{id}` | omitted | Include OCR/full text on the detail payload. | When set, `extraction.full_text_original` / `full_text_english` are present. Default response omits them (same shape as before for text). |

**Grep:** `Sort`, `PageCount`, `IncludeText`, `EnsureThumbnail` in `internal/repository`, `internal/service`, `internal/server`.

---

## App Settings (database)

| Setting | Plain English | Technical |
|---------|---------------|-----------|
| Ollama server URL | User can point Sonix at another host without editing env. | Validated (`ValidateOllamaURL`); read in `ExtractionService.buildOllamaClient` via SettingsService; falls back to `OLLAMA_BASE_URL`. |
| Model for scanning pages | Vision model for page images. | Settings key `ollama_model` → `EffectiveVisionModel`; falls back to `OLLAMA_VISION_MODEL`. |
| Model for translation and summary | Text model for English translation, summary, and document date. | Settings key `ollama_text_model` → `EffectiveTextModel`. Empty → reuse the scanning model (existing installs keep working). Otherwise `OLLAMA_TEXT_MODEL`. |
| Test connection | Probes Ollama and checks both configured models exist in `/api/tags`. | `SettingsService.TestOllama`. |
| Enable auto-import | Watch `$DATA_DIR/inbox` for scanner/manual drops. | Settings key `import_inbox_enabled`; Settings label **Auto-import scans**; see [auto-import-scans.md](auto-import-scans.md). |
| Extract after import | Start extraction when an inbox file becomes a letter (only when auto-import is on). | Settings key `import_auto_extract`. |
| Extraction mode (auto-import) | When extract-after is on: **OCR (Tesseract)** or **LLM vision**. Cleared when extract-after is off. Also the suggested default on document Extract / Re-process when the letter has no prior `engine_id`. | Settings key `import_extract_use_ocr`; inbox `Extract.Start(..., useOCR)`. |
| Printer IP | OfficeJet LAN address for the HP helper. | Settings key `hp_printer_ip`; file `$DATA_DIR/hp-scan/printer_ip`. |
| Test printer | Checks the **saved** printer IP on the LAN (save first). 4 tests per IP per minute. Request body is ignored. | `POST /api/settings/printer/test` — TCP :80 then `/eSCL/ScannerStatus` / `/Scan/Status`. |

---

## Cross-links

- Architecture and timing: [extraction.md](extraction.md)
- Auto-import scans / HP helper: [auto-import-scans.md](auto-import-scans.md)
- Example HTTP bodies per branch: [extraction-requests.md](extraction-requests.md)
- Tuning under load: [performance.md](performance.md)
- Env template: [`.env.example`](../.env.example)
