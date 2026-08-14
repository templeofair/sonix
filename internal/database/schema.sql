-- Users: id, username (unique), password_hash, created_at
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

-- Sessions: session_id (primary), user_id, expires_at
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);

-- Documents: id, title (optional), status, created_at, updated_at, extraction_error (last failure reason)
CREATE TABLE IF NOT EXISTS documents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT,
    status TEXT NOT NULL DEFAULT 'pending',
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    extraction_error TEXT,
    extraction_pages_done INTEGER,
    extraction_pages_total INTEGER
);
CREATE INDEX IF NOT EXISTS idx_documents_status_created ON documents(status, created_at);

-- Document pages: id, document_id, storage_path, page_index, content_type, created_at
CREATE TABLE IF NOT EXISTS document_pages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    document_id INTEGER NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    storage_path TEXT NOT NULL,
    page_index INTEGER NOT NULL,
    content_type TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
CREATE INDEX IF NOT EXISTS idx_document_pages_document_id ON document_pages(document_id);

-- Extractions: id, document_id, tags (JSON), category, summary, full_text_original, full_text_english,
-- document_date, raw_response, extracted_at, engine_id, prompt_version.
--   engine_id      identifies the text-extraction engine that produced full_text_original
--                  (e.g. "tesseract:eng", "vision:german-ocr-v1", "vision:generic-vision-json-v1").
--   prompt_version identifies the prompt set used for the metadata row (summary / full_text_english /
--                  document_date), e.g. "metadata-v8", legacy "metadata-v7",
--                  "german-ocr-two-phase-translate-v1". Bumped when prompt rules change (see ollama/prompts.go and extract.go).
-- Both columns are nullable so legacy rows from before this migration stay readable.
CREATE TABLE IF NOT EXISTS extractions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    document_id INTEGER NOT NULL REFERENCES documents(id) ON DELETE CASCADE UNIQUE,
    tags TEXT NOT NULL DEFAULT '[]',
    category TEXT,
    summary TEXT,
    full_text_original TEXT,
    full_text_english TEXT,
    document_date TEXT,
    raw_response TEXT,
    extracted_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    engine_id TEXT,
    prompt_version TEXT,
    extraction_wall_ms INTEGER
);
CREATE INDEX IF NOT EXISTS idx_extractions_document_id ON extractions(document_id);
CREATE INDEX IF NOT EXISTS idx_extractions_document_date ON extractions(document_date);
CREATE INDEX IF NOT EXISTS idx_extractions_category ON extractions(category);

-- App settings (e.g. ollama_base_url), key-value store
CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

-- FTS5 for full-text search, populated from application when extractions are inserted/updated.
CREATE VIRTUAL TABLE IF NOT EXISTS extractions_fts USING fts5(
    document_id UNINDEXED,
    summary,
    full_text_original,
    full_text_english
);
