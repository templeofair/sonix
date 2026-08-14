package repository

import (
	"context"
	"database/sql"
	"log"
	"time"
)

// PageRef is a document page used by the extraction pipeline.
type PageRef struct {
	Index       int
	StoragePath string
}

// ExtractionRepository persists extraction status, progress, and text.
type ExtractionRepository interface {
	SetDocumentProcessing(ctx context.Context, docID int64) error
	GetDocumentStatus(ctx context.Context, docID int64) (string, error)
	ResetExtraction(ctx context.Context, docID int64) (updated bool, err error)
	ResetStuckExtractions(ctx context.Context) (int64, error)
	MarkFailed(ctx context.Context, docID int64, reason string) error
	MarkPartial(ctx context.Context, docID int64, reason string) error
	MarkReady(ctx context.Context, docID int64) error
	SetProgress(ctx context.Context, docID int64, done, total int) error
	ClearProgress(ctx context.Context, docID int64) error
	LoadPages(ctx context.Context, docID int64) ([]PageRef, error)
	SaveOriginalText(ctx context.Context, docID int64, original, engineID string) error
	SaveMetadata(ctx context.Context, docID int64, summary, fullTextEnglish, documentDate, promptVersion string) error
	SaveRawResponse(ctx context.Context, docID int64, raw string) error
	SaveWallMs(ctx context.Context, docID int64, wallMs int64) error
	RefreshFTS(ctx context.Context, docID int64, summary, original, english string)
	IsProcessing(ctx context.Context, docID int64) bool
}

// SQLiteExtractionRepository implements ExtractionRepository.
type SQLiteExtractionRepository struct {
	db *sql.DB
}

// NewSQLiteExtractionRepository returns an ExtractionRepository backed by db.
func NewSQLiteExtractionRepository(db *sql.DB) *SQLiteExtractionRepository {
	return &SQLiteExtractionRepository{db: db}
}

func (r *SQLiteExtractionRepository) SetDocumentProcessing(ctx context.Context, docID int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE documents SET status = 'processing', updated_at = ? WHERE id = ?`,
		time.Now().UTC().Format(time.RFC3339), docID)
	return err
}

func (r *SQLiteExtractionRepository) GetDocumentStatus(ctx context.Context, docID int64) (string, error) {
	var status string
	err := r.db.QueryRowContext(ctx, `SELECT status FROM documents WHERE id = ?`, docID).Scan(&status)
	if err == sql.ErrNoRows {
		return "", ErrNotFound
	}
	return status, err
}

func (r *SQLiteExtractionRepository) ResetExtraction(ctx context.Context, docID int64) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE documents SET status = 'pending', updated_at = ?, extraction_error = NULL,
			extraction_pages_done = NULL, extraction_pages_total = NULL
		 WHERE id = ? AND status IN ('processing', 'failed', 'partial')`,
		time.Now().UTC().Format(time.RFC3339), docID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *SQLiteExtractionRepository) ResetStuckExtractions(ctx context.Context) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE documents SET status = 'failed', updated_at = ?, extraction_error = ?,
			extraction_pages_done = NULL, extraction_pages_total = NULL WHERE status = 'processing'`,
		time.Now().UTC().Format(time.RFC3339), "Extraction interrupted (server restarted).")
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

func (r *SQLiteExtractionRepository) MarkFailed(ctx context.Context, docID int64, reason string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx,
		`UPDATE documents SET status = 'failed', updated_at = ?, extraction_error = ?,
			extraction_pages_done = NULL, extraction_pages_total = NULL WHERE id = ? AND status = 'processing'`,
		now, reason, docID)
	return err
}

// MarkPartial records that original text was saved but translation/summary failed.
func (r *SQLiteExtractionRepository) MarkPartial(ctx context.Context, docID int64, reason string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx,
		`UPDATE documents SET status = 'partial', updated_at = ?, extraction_error = ?,
			extraction_pages_done = NULL, extraction_pages_total = NULL WHERE id = ? AND status = 'processing'`,
		now, reason, docID)
	return err
}

func (r *SQLiteExtractionRepository) MarkReady(ctx context.Context, docID int64) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx,
		`UPDATE documents SET status = 'ready', updated_at = ?, extraction_error = NULL,
			extraction_pages_done = NULL, extraction_pages_total = NULL WHERE id = ?`,
		now, docID)
	return err
}

func (r *SQLiteExtractionRepository) SetProgress(ctx context.Context, docID int64, done, total int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE documents SET extraction_pages_done = ?, extraction_pages_total = ?, updated_at = ? WHERE id = ?`,
		done, total, time.Now().UTC().Format(time.RFC3339), docID)
	return err
}

func (r *SQLiteExtractionRepository) ClearProgress(ctx context.Context, docID int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE documents SET extraction_pages_done = NULL, extraction_pages_total = NULL, updated_at = ? WHERE id = ?`,
		time.Now().UTC().Format(time.RFC3339), docID)
	return err
}

func (r *SQLiteExtractionRepository) LoadPages(ctx context.Context, docID int64) ([]PageRef, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT page_index, storage_path FROM document_pages WHERE document_id = ? ORDER BY page_index`, docID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var pages []PageRef
	for rows.Next() {
		var p PageRef
		if err := rows.Scan(&p.Index, &p.StoragePath); err != nil {
			return nil, err
		}
		pages = append(pages, p)
	}
	return pages, rows.Err()
}

func (r *SQLiteExtractionRepository) SaveOriginalText(ctx context.Context, docID int64, original, engineID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO extractions (document_id, tags, category, summary, full_text_original, full_text_english, document_date, extracted_at, engine_id)
		VALUES (?, '[]', '', '', ?, '', NULL, ?, ?)
		ON CONFLICT(document_id) DO UPDATE SET
			full_text_original = excluded.full_text_original,
			extracted_at = excluded.extracted_at,
			engine_id = excluded.engine_id
	`, docID, original, now, engineID)
	return err
}

func (r *SQLiteExtractionRepository) SaveMetadata(ctx context.Context, docID int64, summary, fullTextEnglish, documentDate, promptVersion string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `
		UPDATE extractions SET summary = ?, full_text_english = ?, document_date = ?, category = NULL, extracted_at = ?, prompt_version = ?
		WHERE document_id = ?
	`, summary, fullTextEnglish, nullIfEmpty(documentDate), now, promptVersion, docID)
	return err
}

// SaveRawResponse stores the model output that failed to parse. The row is
// created by SaveOriginalText before the metadata call runs, so an UPDATE is
// enough; if the run failed earlier than that there is nothing to attach to and
// the no-op is correct.
func (r *SQLiteExtractionRepository) SaveRawResponse(ctx context.Context, docID int64, raw string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE extractions SET raw_response = ? WHERE document_id = ?`, nullIfEmpty(raw), docID)
	return err
}

func (r *SQLiteExtractionRepository) SaveWallMs(ctx context.Context, docID int64, wallMs int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE extractions SET extraction_wall_ms = ? WHERE document_id = ?`, wallMs, docID)
	return err
}

func (r *SQLiteExtractionRepository) RefreshFTS(ctx context.Context, docID int64, summary, original, english string) {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM extractions_fts WHERE document_id = ?`, docID); err != nil {
		log.Printf("extraction: doc_id=%d FTS delete failed: %v", docID, err)
	}
	if _, err := r.db.ExecContext(ctx,
		`INSERT INTO extractions_fts (document_id, summary, full_text_original, full_text_english) VALUES (?, ?, ?, ?)`,
		docID, summary, original, english); err != nil {
		log.Printf("extraction: doc_id=%d FTS insert failed: %v", docID, err)
	}
}

func (r *SQLiteExtractionRepository) IsProcessing(ctx context.Context, docID int64) bool {
	var status string
	if err := r.db.QueryRowContext(ctx, `SELECT status FROM documents WHERE id = ?`, docID).Scan(&status); err != nil {
		return false
	}
	return status == "processing"
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
