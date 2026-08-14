package database

import (
	"database/sql"
	"embed"
	"fmt"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaFS embed.FS

// Open opens a SQLite database at path and runs migrations (schema).
func Open(path string) (*sql.DB, error) {
	// PRAGMAs in the DSN apply to every pooled connection. Exec-only PRAGMAs
	// only affect the connection that ran them, which left later pool
	// connections without busy_timeout / WAL under MaxOpenConns > 1.
	dsn := fmt.Sprintf(
		"file:%s?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)",
		filepath.ToSlash(path),
	)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	// WAL allows concurrent readers with one writer. A pool of 1 wedged the
	// whole API (including login) when a long extraction or an interrupted
	// query left the sole connection unusable. modernc.org/sqlite ≥1.36
	// discards interrupted connections via IsValid (SQLITE_INTERRUPT).
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	if err := runSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	// Additive migrations. Each statement is best-effort: on a fresh database
	// the column is already in CREATE TABLE; on an existing database the
	// ALTER adds it. SQLite returns "duplicate column" when it already
	// exists, which we discard so startup stays idempotent.
	_, _ = db.Exec("ALTER TABLE documents ADD COLUMN extraction_error TEXT")
	_, _ = db.Exec("ALTER TABLE extractions ADD COLUMN engine_id TEXT")
	_, _ = db.Exec("ALTER TABLE extractions ADD COLUMN prompt_version TEXT")
	_, _ = db.Exec("ALTER TABLE documents ADD COLUMN extraction_pages_done INTEGER")
	_, _ = db.Exec("ALTER TABLE documents ADD COLUMN extraction_pages_total INTEGER")
	_, _ = db.Exec("ALTER TABLE extractions ADD COLUMN extraction_wall_ms INTEGER")
	_, _ = db.Exec("CREATE INDEX IF NOT EXISTS idx_extractions_document_date ON extractions(document_date)")
	_, _ = db.Exec("CREATE INDEX IF NOT EXISTS idx_extractions_category ON extractions(category)")
	return db, nil
}

func runSchema(db *sql.DB) error {
	data, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("read schema: %w", err)
	}
	// Run each statement (split by semicolon). Skip only if segment is empty or
	// entirely comment/whitespace; segments that start with a comment but contain
	// SQL (e.g. "-- Sessions\nCREATE TABLE...") must still be executed.
	stmts := strings.Split(string(data), ";")
	for _, s := range stmts {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		// Skip only if the first non-empty, non-comment line doesn't look like SQL
		trimmed := dropLeadingCommentsAndBlanks(s)
		if trimmed == "" {
			continue
		}
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("exec schema: %w\nstmt: %s", err, s)
		}
	}
	return nil
}

// dropLeadingCommentsAndBlanks removes leading lines that are empty or start with "--".
// Used to detect segments that are entirely comments (should be skipped).
func dropLeadingCommentsAndBlanks(s string) string {
	for s != "" {
		line := s
		if i := strings.Index(s, "\n"); i >= 0 {
			line = s[:i]
			s = s[i+1:]
		} else {
			s = ""
		}
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "--") {
			return line + "\n" + s // at least one SQL line remains
		}
	}
	return ""
}
