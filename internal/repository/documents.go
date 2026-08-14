package repository

import (
	"context"
	"database/sql"
	"strings"
	"time"
	"unicode"
)

// DocumentListFilter holds query params for listing/exporting documents.
type DocumentListFilter struct {
	Q                string
	Tag              string // comma-separated; match if any tag present (OR)
	Year             string // comma-separated upload years; strftime IN (OR)
	DocumentDateFrom string
	DocumentDateTo   string
	CreatedFrom      string
	CreatedTo        string
	Status           string // comma-separated, e.g. "pending,failed"
	Sort             string // created_desc (default), date_desc, date_asc
	Undated          bool   // restrict to documents with no letter date
	Limit            int
	Offset           int
}

// DocumentListRow is one row from a list query.
type DocumentListRow struct {
	ID           int64
	Title        sql.NullString
	Status       string
	CreatedAt    string // RFC3339 from DB
	UpdatedAt    string
	DocumentDate sql.NullString
	PageCount    int
}

// DocumentDateYear is one letter-date year with the number of documents in it.
type DocumentDateYear struct {
	Year  string
	Count int64
}

// DocumentIDRow is id + created_at for export.
type DocumentIDRow struct {
	ID        int64
	CreatedAt string
}

// DocumentPageRow is a page metadata row (no file bytes).
type DocumentPageRow struct {
	PageIndex   int
	ContentType string
	StoragePath string
}

// DocumentCore is the documents table row used by GetDetail.
type DocumentCore struct {
	ID                   int64
	Title                sql.NullString
	Status               string
	CreatedAt            string
	UpdatedAt            string
	ExtractionError      sql.NullString
	ExtractionPagesDone  sql.NullInt64
	ExtractionPagesTotal sql.NullInt64
}

// ExtractionDetail is the extractions row used by GetDetail / export.
type ExtractionDetail struct {
	TagsJSON         string
	Summary          string
	DocumentDate     string
	ExtractedAt      string
	EngineID         string
	PromptVersion    string
	ExtractionWallMs sql.NullInt64
	FullTextOriginal string
	FullTextEnglish  string
}

// DocumentRepository lists and mutates documents.
type DocumentRepository interface {
	List(ctx context.Context, f DocumentListFilter) ([]DocumentListRow, error)
	Count(ctx context.Context, f DocumentListFilter) (int64, error)
	ListIDs(ctx context.Context, f DocumentListFilter) ([]DocumentIDRow, error)
	Years(ctx context.Context) ([]string, error)
	Tags(ctx context.Context) ([]string, error)
	DocumentDateYears(ctx context.Context) (years []DocumentDateYear, undated int64, err error)
	Create(ctx context.Context, title interface{}) (int64, error)
	UpdateTitle(ctx context.Context, docID int64, title interface{}) (updated bool, err error)
	GetCore(ctx context.Context, docID int64) (*DocumentCore, error)
	ListPages(ctx context.Context, docID int64) ([]DocumentPageRow, error)
	GetExtraction(ctx context.Context, docID int64) (*ExtractionDetail, error)
	UpsertTags(ctx context.Context, docID int64, tagsJSON string) error
	UpdateDocumentDate(ctx context.Context, docID int64, documentDate interface{}) (updated bool, err error)
	Delete(ctx context.Context, docID int64) (deleted bool, err error)
	Exists(ctx context.Context, docID int64) (bool, error)
	MaxPageIndex(ctx context.Context, docID int64) (int, error)
	InsertPage(ctx context.Context, docID int64, storagePath string, pageIndex int, contentType string) error
	GetPage(ctx context.Context, docID int64, pageIndex int) (storagePath, contentType string, err error)
	UpdatePageContentType(ctx context.Context, docID int64, pageIndex int, contentType string) error
	GetText(ctx context.Context, docID int64, lang string) (string, error)
}

// SQLiteDocumentRepository implements DocumentRepository.
type SQLiteDocumentRepository struct {
	db *sql.DB
}

// NewSQLiteDocumentRepository returns a DocumentRepository backed by db.
func NewSQLiteDocumentRepository(db *sql.DB) *SQLiteDocumentRepository {
	return &SQLiteDocumentRepository{db: db}
}

// ftsMatchArg turns a user search string into an FTS5 MATCH expression.
// Tokens are quoted so operators like AND/OR/NEAR cannot produce a syntax error (HTTP 500).
// An empty return means “no searchable tokens”; callers should match nothing, not everything.
func ftsMatchArg(q string) string {
	tokens := ftsTokens(q)
	if len(tokens) == 0 {
		return ""
	}
	parts := make([]string, len(tokens))
	for i, tok := range tokens {
		parts[i] = `"` + strings.ReplaceAll(tok, `"`, `""`) + `"`
	}
	return strings.Join(parts, " AND ")
}

func ftsTokens(s string) []string {
	var tokens []string
	var b strings.Builder
	flush := func() {
		if b.Len() == 0 {
			return
		}
		tokens = append(tokens, b.String())
		b.Reset()
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			b.WriteRune(r)
			continue
		}
		flush()
	}
	flush()
	return tokens
}

// splitCSV trims and drops empty segments from a comma-separated filter param.
func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// buildListQuery builds FROM/WHERE/args for list and export.
// includeTagsInFTS matches list behaviour (FTS UNION tags LIKE); export historically matched FTS only.
// forCount omits ORDER BY / LIMIT so callers can SELECT COUNT(*).
func buildListQuery(f DocumentListFilter, includeTagsInFTS bool, selectCols string, withLimit bool, forCount bool) (query string, args []any) {
	where := []string{"1=1"}
	if f.Status != "" {
		cleaned := splitCSV(f.Status)
		if len(cleaned) > 0 {
			placeholders := make([]string, len(cleaned))
			for i := range cleaned {
				placeholders[i] = "?"
				args = append(args, cleaned[i])
			}
			where = append(where, "d.status IN ("+strings.Join(placeholders, ",")+")")
		}
	}
	if years := splitCSV(f.Year); len(years) > 0 {
		placeholders := make([]string, len(years))
		for i := range years {
			placeholders[i] = "?"
			args = append(args, years[i])
		}
		where = append(where, "strftime('%Y', d.created_at) IN ("+strings.Join(placeholders, ",")+")")
	}
	if f.CreatedFrom != "" {
		where = append(where, "d.created_at >= ?")
		args = append(args, f.CreatedFrom)
	}
	if f.CreatedTo != "" {
		where = append(where, "d.created_at <= ?")
		args = append(args, f.CreatedTo)
	}

	needsExtractionJoin := f.Tag != "" || f.DocumentDateFrom != "" || f.DocumentDateTo != "" || f.Q != "" || f.Undated
	alwaysJoinExtractions := includeTagsInFTS // list path uses includeTagsInFTS=true
	fromClause := "FROM documents d"
	if alwaysJoinExtractions || needsExtractionJoin {
		fromClause = "FROM documents d LEFT JOIN extractions e ON e.document_id = d.id"
	}
	if tags := splitCSV(f.Tag); len(tags) > 0 {
		placeholders := make([]string, len(tags))
		for i := range tags {
			placeholders[i] = "?"
			args = append(args, tags[i])
		}
		where = append(where, "json_valid(e.tags) AND EXISTS (SELECT 1 FROM json_each(e.tags) WHERE value IN ("+strings.Join(placeholders, ",")+"))")
	}
	// Undated and a document_date range are contradictory: no NULL/'' date can satisfy
	// a range comparison. When both arrive, Undated wins and the range is ignored, so a
	// stale range param cannot silently empty the No date bucket.
	if f.Undated {
		where = append(where, "(e.document_date IS NULL OR e.document_date = '')")
	} else {
		if f.DocumentDateFrom != "" {
			where = append(where, "e.document_date >= ?")
			args = append(args, f.DocumentDateFrom)
		}
		if f.DocumentDateTo != "" {
			where = append(where, "e.document_date <= ?")
			args = append(args, f.DocumentDateTo)
		}
	}
	if f.Q != "" {
		match := ftsMatchArg(f.Q)
		if match == "" {
			where = append(where, "0")
		} else if includeTagsInFTS {
			fromClause = "FROM documents d INNER JOIN (SELECT document_id FROM extractions_fts WHERE extractions_fts MATCH ? UNION SELECT document_id FROM extractions WHERE tags LIKE ?) f ON f.document_id = d.id LEFT JOIN extractions e ON e.document_id = d.id"
			args = append([]any{match, "%" + f.Q + "%"}, args...)
		} else {
			fromClause = "FROM documents d INNER JOIN (SELECT document_id FROM extractions_fts WHERE extractions_fts MATCH ?) f ON f.document_id = d.id LEFT JOIN extractions e ON e.document_id = d.id"
			args = append([]any{match}, args...)
		}
	}

	query = `SELECT ` + selectCols + ` ` + fromClause + ` WHERE ` + strings.Join(where, " AND ")
	if forCount {
		return query, args
	}

	query += listOrderClause(f.Sort, withLimit)
	if withLimit {
		limit := f.Limit
		if limit <= 0 || limit > 500 {
			limit = 50
		}
		offset := f.Offset
		if offset < 0 {
			offset = 0
		}
		query += ` LIMIT ? OFFSET ?`
		args = append(args, limit, offset)
	}
	return query, args
}

// listOrderClause preserves historical defaults:
//   - list (withLimit): created_at DESC unless sort=date_*
//   - export ListIDs (!withLimit): created_at ASC (unchanged)
func listOrderClause(sort string, withLimit bool) string {
	if !withLimit {
		return " ORDER BY d.created_at"
	}
	switch strings.ToLower(strings.TrimSpace(sort)) {
	case "date_desc":
		return " ORDER BY CASE WHEN e.document_date IS NULL OR e.document_date = '' THEN 1 ELSE 0 END, e.document_date DESC, d.created_at DESC"
	case "date_asc":
		return " ORDER BY CASE WHEN e.document_date IS NULL OR e.document_date = '' THEN 1 ELSE 0 END, e.document_date ASC, d.created_at DESC"
	default:
		return " ORDER BY d.created_at DESC"
	}
}

func (r *SQLiteDocumentRepository) List(ctx context.Context, f DocumentListFilter) ([]DocumentListRow, error) {
	const cols = `d.id, d.title, d.status, d.created_at, d.updated_at, e.document_date,
		(SELECT COUNT(*) FROM document_pages dp WHERE dp.document_id = d.id) AS page_count`
	query, args := buildListQuery(f, true, cols, true, false)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []DocumentListRow
	for rows.Next() {
		var d DocumentListRow
		if err := rows.Scan(&d.ID, &d.Title, &d.Status, &d.CreatedAt, &d.UpdatedAt, &d.DocumentDate, &d.PageCount); err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	if list == nil {
		list = []DocumentListRow{}
	}
	return list, rows.Err()
}

func (r *SQLiteDocumentRepository) Count(ctx context.Context, f DocumentListFilter) (int64, error) {
	query, args := buildListQuery(f, true, `COUNT(*)`, false, true)
	var n int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&n)
	return n, err
}

func (r *SQLiteDocumentRepository) ListIDs(ctx context.Context, f DocumentListFilter) ([]DocumentIDRow, error) {
	// Preserve historical export FTS behaviour (MATCH only, no tags UNION).
	query, args := buildListQuery(f, false, `d.id, d.created_at`, false, false)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []DocumentIDRow
	for rows.Next() {
		var d DocumentIDRow
		if err := rows.Scan(&d.ID, &d.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	if list == nil {
		list = []DocumentIDRow{}
	}
	return list, rows.Err()
}

func (r *SQLiteDocumentRepository) Years(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT DISTINCT strftime('%Y', created_at) AS year FROM documents ORDER BY year DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var years []string
	for rows.Next() {
		var y string
		if err := rows.Scan(&y); err != nil {
			return nil, err
		}
		years = append(years, y)
	}
	if years == nil {
		years = []string{}
	}
	return years, rows.Err()
}

// Tags returns distinct manual tag strings across extractions, sorted ascending.
func (r *SQLiteDocumentRepository) Tags(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT je.value
		  FROM extractions e, json_each(e.tags) AS je
		 WHERE json_valid(e.tags) AND typeof(je.value) = 'text' AND je.value != ''
		 ORDER BY je.value COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tags []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	if tags == nil {
		tags = []string{}
	}
	return tags, rows.Err()
}

// DocumentDateYears groups documents by the year of the letter's own date, newest first,
// and counts the ones with no letter date at all (no extraction row, NULL, or empty).
// The two buckets partition the documents table: sum(counts) + undated == COUNT(*).
func (r *SQLiteDocumentRepository) DocumentDateYears(ctx context.Context) ([]DocumentDateYear, int64, error) {
	// substr fallback keeps the partition exact: strftime returns NULL for a stored date
	// that is not a valid ISO timestamp, and such a row is still not undated.
	rows, err := r.db.QueryContext(ctx,
		`SELECT COALESCE(strftime('%Y', e.document_date), substr(e.document_date, 1, 4)) AS year, COUNT(*) AS count
		   FROM documents d LEFT JOIN extractions e ON e.document_id = d.id
		  WHERE e.document_date IS NOT NULL AND e.document_date != ''
		  GROUP BY year ORDER BY year DESC`)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var years []DocumentDateYear
	for rows.Next() {
		var y DocumentDateYear
		if err := rows.Scan(&y.Year, &y.Count); err != nil {
			return nil, 0, err
		}
		years = append(years, y)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if years == nil {
		years = []DocumentDateYear{}
	}

	var undated int64
	err = r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM documents d LEFT JOIN extractions e ON e.document_id = d.id
		  WHERE e.document_date IS NULL OR e.document_date = ''`).Scan(&undated)
	if err != nil {
		return nil, 0, err
	}
	return years, undated, nil
}

func (r *SQLiteDocumentRepository) Create(ctx context.Context, title interface{}) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO documents (title, status) VALUES (?, 'pending')`, title)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *SQLiteDocumentRepository) UpdateTitle(ctx context.Context, docID int64, title interface{}) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE documents SET title = ?, updated_at = ? WHERE id = ?`,
		title, time.Now().UTC().Format(time.RFC3339), docID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *SQLiteDocumentRepository) GetCore(ctx context.Context, docID int64) (*DocumentCore, error) {
	var d DocumentCore
	err := r.db.QueryRowContext(ctx,
		`SELECT id, title, status, created_at, updated_at, extraction_error, extraction_pages_done, extraction_pages_total
		   FROM documents WHERE id = ?`, docID).Scan(
		&d.ID, &d.Title, &d.Status, &d.CreatedAt, &d.UpdatedAt,
		&d.ExtractionError, &d.ExtractionPagesDone, &d.ExtractionPagesTotal)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *SQLiteDocumentRepository) ListPages(ctx context.Context, docID int64) ([]DocumentPageRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT page_index, content_type, storage_path FROM document_pages WHERE document_id = ? ORDER BY page_index`, docID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []DocumentPageRow
	for rows.Next() {
		var p DocumentPageRow
		if err := rows.Scan(&p.PageIndex, &p.ContentType, &p.StoragePath); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	if list == nil {
		list = []DocumentPageRow{}
	}
	return list, rows.Err()
}

func (r *SQLiteDocumentRepository) GetExtraction(ctx context.Context, docID int64) (*ExtractionDetail, error) {
	var e ExtractionDetail
	err := r.db.QueryRowContext(ctx,
		`SELECT tags, COALESCE(summary,''), COALESCE(document_date,''), COALESCE(extracted_at,''),
		        COALESCE(engine_id,''), COALESCE(prompt_version,''), extraction_wall_ms,
		        COALESCE(full_text_original,''), COALESCE(full_text_english,'')
		   FROM extractions WHERE document_id = ?`, docID).Scan(
		&e.TagsJSON, &e.Summary, &e.DocumentDate, &e.ExtractedAt,
		&e.EngineID, &e.PromptVersion, &e.ExtractionWallMs,
		&e.FullTextOriginal, &e.FullTextEnglish)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *SQLiteDocumentRepository) UpsertTags(ctx context.Context, docID int64, tagsJSON string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO extractions (document_id, tags) VALUES (?, ?)
		 ON CONFLICT(document_id) DO UPDATE SET tags = excluded.tags`,
		docID, tagsJSON)
	return err
}

func (r *SQLiteDocumentRepository) UpdateDocumentDate(ctx context.Context, docID int64, documentDate interface{}) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE extractions SET document_date = ? WHERE document_id = ?`, documentDate, docID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *SQLiteDocumentRepository) Delete(ctx context.Context, docID int64) (bool, error) {
	_, _ = r.db.ExecContext(ctx, `DELETE FROM extractions_fts WHERE document_id = ?`, docID)
	res, err := r.db.ExecContext(ctx, `DELETE FROM documents WHERE id = ?`, docID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *SQLiteDocumentRepository) Exists(ctx context.Context, docID int64) (bool, error) {
	var exists int
	err := r.db.QueryRowContext(ctx, `SELECT 1 FROM documents WHERE id = ?`, docID).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *SQLiteDocumentRepository) MaxPageIndex(ctx context.Context, docID int64) (int, error) {
	var maxIndex int
	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(page_index), -1) FROM document_pages WHERE document_id = ?`, docID).Scan(&maxIndex)
	return maxIndex, err
}

func (r *SQLiteDocumentRepository) InsertPage(ctx context.Context, docID int64, storagePath string, pageIndex int, contentType string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO document_pages (document_id, storage_path, page_index, content_type) VALUES (?, ?, ?, ?)`,
		docID, storagePath, pageIndex, contentType)
	return err
}

func (r *SQLiteDocumentRepository) GetPage(ctx context.Context, docID int64, pageIndex int) (string, string, error) {
	var storagePath, contentType string
	err := r.db.QueryRowContext(ctx,
		`SELECT storage_path, content_type FROM document_pages WHERE document_id = ? AND page_index = ?`,
		docID, pageIndex).Scan(&storagePath, &contentType)
	if err == sql.ErrNoRows {
		return "", "", ErrNotFound
	}
	return storagePath, contentType, err
}

func (r *SQLiteDocumentRepository) UpdatePageContentType(ctx context.Context, docID int64, pageIndex int, contentType string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE document_pages SET content_type = ? WHERE document_id = ? AND page_index = ?`,
		contentType, docID, pageIndex)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *SQLiteDocumentRepository) GetText(ctx context.Context, docID int64, lang string) (string, error) {
	var text string
	var err error
	if lang == "original" {
		err = r.db.QueryRowContext(ctx, `SELECT full_text_original FROM extractions WHERE document_id = ?`, docID).Scan(&text)
	} else {
		err = r.db.QueryRowContext(ctx, `SELECT full_text_english FROM extractions WHERE document_id = ?`, docID).Scan(&text)
	}
	if err == sql.ErrNoRows {
		return "", ErrNotFound
	}
	return text, err
}

// ParseTimeRFC3339 parses DB timestamps; zero time on failure.
func ParseTimeRFC3339(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}
