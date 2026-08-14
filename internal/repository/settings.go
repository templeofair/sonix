package repository

import (
	"context"
	"database/sql"
)

// Settings keys for Ollama configuration and auto-import scans (inbox).
const (
	SettingsKeyOllamaBaseURL   = "ollama_base_url"
	SettingsKeyOllamaModel     = "ollama_model"      // vision (page scanning)
	SettingsKeyOllamaTextModel = "ollama_text_model" // translation / summary / date

	SettingsKeyImportInboxEnabled  = "import_inbox_enabled"   // "true" / "false"
	SettingsKeyImportAutoExtract   = "import_auto_extract"    // "true" / "false"
	SettingsKeyImportExtractUseOCR = "import_extract_use_ocr" // "true" / "false"; default true when unset
	SettingsKeyImportLastFile      = "import_last_file"
	SettingsKeyImportLastError     = "import_last_error"
	SettingsKeyImportLastAt        = "import_last_at" // RFC3339
	SettingsKeyHPPrinterIP         = "hp_printer_ip"
)

// SettingsRepository reads and writes key/value settings.
type SettingsRepository interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
	Delete(ctx context.Context, key string) error
}

// SQLiteSettingsRepository implements SettingsRepository.
type SQLiteSettingsRepository struct {
	db *sql.DB
}

// NewSQLiteSettingsRepository returns a SettingsRepository backed by db.
func NewSQLiteSettingsRepository(db *sql.DB) *SQLiteSettingsRepository {
	return &SQLiteSettingsRepository{db: db}
}

func (r *SQLiteSettingsRepository) Get(ctx context.Context, key string) (string, error) {
	var value string
	err := r.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return value, nil
}

func (r *SQLiteSettingsRepository) Set(ctx context.Context, key, value string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value)
	return err
}

func (r *SQLiteSettingsRepository) Delete(ctx context.Context, key string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM settings WHERE key = ?`, key)
	return err
}
