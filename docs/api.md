# HTTP API

Base path `/api` except `GET /health`.

**Auth:** every `/api` route except `POST /api/login` and `POST /api/logout` requires the session cookie (`HttpOnly`, `SameSite=Strict`). `GET /health` is unauthenticated.

**Tenancy:** documents are **not** scoped per user. Any valid session can read, modify, and delete any letter.

JSON unless noted. Errors are short plain-text messages (no internal traces).

| Method | Path | Auth | Notes |
|--------|------|------|--------|
| GET | `/health` | no | `ok` |
| POST | `/api/login` | no | Body `{"username","password"}`. Sets cookie. `{"ok":"true"}`. 401 / 429 |
| POST | `/api/logout` | cookie optional | Clears cookie |
| GET | `/api/me` | yes | `{"username"}` |
| GET | `/api/settings` | yes | Ollama URL/models, inbox flags, printer IP |
| PUT | `/api/settings` | yes | Same fields; URL and printer IP validated |
| POST | `/api/settings/ollama/test` | yes | Connectivity check |
| POST | `/api/settings/printer/test` | yes | Uses **saved** printer IP only; rate-limited |
| GET | `/api/export` | yes | Zip stream of the library |
| GET | `/api/documents` | yes | List; query: `q`, tags, date, `undated`, pagination |
| POST | `/api/documents` | yes | `{"title"}` → `201` `{"id"}` |
| GET | `/api/documents/years` | yes | Created-at year buckets |
| GET | `/api/documents/tags` | yes | Tag list |
| GET | `/api/documents/document-date-years` | yes | Letter-date years + `undated_count` |
| GET | `/api/documents/{id}` | yes | Detail + pages + extraction |
| DELETE | `/api/documents/{id}` | yes | |
| PUT | `/api/documents/{id}/title` | yes | |
| PUT | `/api/documents/{id}/tags` | yes | |
| PUT | `/api/documents/{id}/document_date` | yes | ISO date or empty |
| POST | `/api/documents/{id}/pages` | yes | Multipart images/PDF; cap `DOCUMENT_MAX_PAGES` |
| GET | `/api/documents/{id}/pages/{pageIndex}/image` | yes | Original page |
| GET | `/api/documents/{id}/pages/{pageIndex}/thumbnail` | yes | JPEG thumb |
| POST | `/api/documents/{id}/pages/{pageIndex}/rotate` | yes | |
| GET | `/api/documents/{id}/text` | yes | Stored text |
| POST | `/api/documents/{id}/extract` | yes | `{"use_ocr": false}` (optional `ignore_ocr`). 429 if busy |
| GET | `/api/documents/{id}/status` | yes | Job status |
| POST | `/api/documents/{id}/reset-extraction` | yes | Cancel / reset |

List/detail JSON field names match the SPA (`id`, `title`, `status`, `document_date`, `page_count`, `extraction`, …). Env and extract toggles: [configuration.md](configuration.md).
