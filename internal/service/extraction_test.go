package service_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/templeofair/sonix/internal/config"
	"github.com/templeofair/sonix/internal/database"
	"github.com/templeofair/sonix/internal/ocr"
	"github.com/templeofair/sonix/internal/ollama"
	"github.com/templeofair/sonix/internal/repository"
	"github.com/templeofair/sonix/internal/service"
)

// Tests exercise the persistence side of the extraction pipeline against a
// real SQLite database. They intentionally avoid Ollama — the goal is to
// lock in the SQL contract for engine_id and prompt_version.

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func newTestExtractionService(t *testing.T, cfg *config.Config) (*service.ExtractionService, *sql.DB) {
	t.Helper()
	db := newTestDB(t)
	if cfg == nil {
		cfg = &config.Config{
			OllamaBaseURL: "http://127.0.0.1:11434",
			OllamaVision:  "gemma3:latest",
			OllamaText:    "gemma3:latest",
		}
	}
	settings := service.NewSettingsService(repository.NewSQLiteSettingsRepository(db), cfg)
	ocrProvider := ocr.NewTesseractOptions(ocr.TesseractOptions{
		Lang: cfg.OCRLang,
		DPI:  cfg.OCRDPI,
		PSM:  cfg.OCRPSM,
	})
	svc := service.NewExtractionService(
		repository.NewSQLiteExtractionRepository(db),
		settings,
		ocrProvider,
		cfg,
		t.TempDir(),
	)
	return svc, db
}

func insertDoc(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	res, err := db.ExecContext(context.Background(),
		`INSERT INTO documents (status) VALUES ('processing')`)
	if err != nil {
		t.Fatalf("insert document: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

type extractionRow struct {
	engineID         sql.NullString
	promptVersion    sql.NullString
	summary          sql.NullString
	fullTextOriginal sql.NullString
	fullTextEnglish  sql.NullString
	documentDate     sql.NullString
}

func readExtraction(t *testing.T, db *sql.DB, docID int64) extractionRow {
	t.Helper()
	var r extractionRow
	err := db.QueryRowContext(context.Background(),
		`SELECT engine_id, prompt_version, summary, full_text_original, full_text_english, document_date
		   FROM extractions WHERE document_id = ?`, docID).Scan(
		&r.engineID, &r.promptVersion, &r.summary, &r.fullTextOriginal, &r.fullTextEnglish, &r.documentDate)
	if err != nil {
		t.Fatalf("read extractions row: %v", err)
	}
	return r
}

func TestSaveOriginalText_PersistsEngineID(t *testing.T) {
	s, db := newTestExtractionService(t, nil)
	docID := insertDoc(t, db)

	if err := s.SaveOriginalText(context.Background(), docID, "hello world", "vision:unified-vision-v1"); err != nil {
		t.Fatalf("saveOriginalText: %v", err)
	}

	row := readExtraction(t, db, docID)
	if !row.engineID.Valid || row.engineID.String != "vision:unified-vision-v1" {
		t.Fatalf("engine_id = %v, want %q", row.engineID, "vision:unified-vision-v1")
	}
	if row.promptVersion.Valid {
		t.Fatalf("prompt_version should be NULL after only saveOriginalText, got %q", row.promptVersion.String)
	}
	if !row.fullTextOriginal.Valid || row.fullTextOriginal.String != "hello world" {
		t.Fatalf("full_text_original = %v, want %q", row.fullTextOriginal, "hello world")
	}
}

func TestSaveMetadata_StampsPromptVersionAndKeepsEngineID(t *testing.T) {
	s, db := newTestExtractionService(t, nil)
	docID := insertDoc(t, db)

	if err := s.SaveOriginalText(context.Background(), docID, "Rechnung 1", "vision:unified-vision-v1"); err != nil {
		t.Fatalf("saveOriginalText: %v", err)
	}
	if err := s.SaveMetadata(context.Background(), docID, "summary text", "english text", "2024-12-19", "metadata-v12"); err != nil {
		t.Fatalf("saveMetadata: %v", err)
	}

	row := readExtraction(t, db, docID)
	if !row.engineID.Valid || row.engineID.String != "vision:unified-vision-v1" {
		t.Fatalf("engine_id was lost: %v", row.engineID)
	}
	if !row.promptVersion.Valid || row.promptVersion.String != "metadata-v12" {
		t.Fatalf("prompt_version = %v, want %q", row.promptVersion, "metadata-v12")
	}
	if row.summary.String != "summary text" || row.fullTextEnglish.String != "english text" || row.documentDate.String != "2024-12-19" {
		t.Fatalf("payload fields not persisted: summary=%v english=%v date=%v",
			row.summary, row.fullTextEnglish, row.documentDate)
	}
}

func TestSaveOriginalText_ReusesRowOnReextract(t *testing.T) {
	s, db := newTestExtractionService(t, nil)
	docID := insertDoc(t, db)

	if err := s.SaveOriginalText(context.Background(), docID, "old text", "tesseract:eng"); err != nil {
		t.Fatalf("first save: %v", err)
	}
	if err := s.SaveOriginalText(context.Background(), docID, "new text", "vision:unified-vision-v1"); err != nil {
		t.Fatalf("re-extract save: %v", err)
	}

	row := readExtraction(t, db, docID)
	if row.engineID.String != "vision:unified-vision-v1" {
		t.Fatalf("engine_id not updated on re-extract: %q", row.engineID.String)
	}
	if row.fullTextOriginal.String != "new text" {
		t.Fatalf("full_text_original not updated: %q", row.fullTextOriginal.String)
	}
}

func TestSaveExtractionWallMs_PersistsWallMs(t *testing.T) {
	s, db := newTestExtractionService(t, nil)
	docID := insertDoc(t, db)

	if err := s.SaveOriginalText(context.Background(), docID, "x", "vision:unified-vision-v1"); err != nil {
		t.Fatalf("saveOriginalText: %v", err)
	}
	if err := s.SaveExtractionWallMs(context.Background(), docID, 42_000); err != nil {
		t.Fatalf("saveExtractionWallMs: %v", err)
	}
	var wall sql.NullInt64
	err := db.QueryRowContext(context.Background(),
		`SELECT extraction_wall_ms FROM extractions WHERE document_id = ?`, docID).Scan(&wall)
	if err != nil {
		t.Fatalf("read extraction_wall_ms: %v", err)
	}
	if !wall.Valid || wall.Int64 != 42_000 {
		t.Fatalf("extraction_wall_ms = %v, want 42000", wall)
	}
}

func TestReset_AcceptsPartialStatus(t *testing.T) {
	s, db := newTestExtractionService(t, nil)
	docID := insertDoc(t, db)
	ctx := context.Background()
	if err := s.SaveOriginalText(ctx, docID, "German text", "vision:unified-vision-v1"); err != nil {
		t.Fatalf("saveOriginalText: %v", err)
	}
	repo := repository.NewSQLiteExtractionRepository(db)
	if err := repo.MarkPartial(ctx, docID, "LLM metadata: boom"); err != nil {
		t.Fatalf("MarkPartial: %v", err)
	}
	var status, extrErr string
	if err := db.QueryRowContext(ctx, `SELECT status, extraction_error FROM documents WHERE id = ?`, docID).
		Scan(&status, &extrErr); err != nil {
		t.Fatalf("read status: %v", err)
	}
	if status != "partial" || extrErr == "" {
		t.Fatalf("status=%q err=%q, want partial with reason", status, extrErr)
	}
	if err := s.Reset(ctx, docID); err != nil {
		t.Fatalf("Reset: %v", err)
	}
	if err := db.QueryRowContext(ctx, `SELECT status FROM documents WHERE id = ?`, docID).Scan(&status); err != nil {
		t.Fatalf("read after reset: %v", err)
	}
	if status != "pending" {
		t.Fatalf("status after reset = %q, want pending", status)
	}
}

func TestExtractor_EngineIDs(t *testing.T) {
	if got := ocr.NewTesseract().EngineID(); got != "tesseract:deu+eng" {
		t.Errorf("tesseract EngineID = %q, want %q", got, "tesseract:deu+eng")
	}

	cases := []struct {
		model string
		want  string
	}{
		{"Keyvan/german-ocr:latest", "vision:unified-vision-v1"},
		{"keyvan/german-ocr", "vision:unified-vision-v1"},
		{"Keyvan/german-ocr-turbo:latest", "vision:unified-vision-v1"},
		{"llama3.2", "vision:unified-vision-v1"},
		{"gemma3:latest", "vision:unified-vision-v1"},
	}
	for _, tc := range cases {
		client := ollama.NewClient("http://localhost:11434", tc.model, tc.model)
		got := "vision:" + client.PageProfileName()
		if got != tc.want {
			t.Errorf("vision engineID for model %q = %q, want %q", tc.model, got, tc.want)
		}
	}
}

func TestMetadataPromptVersion_IsExportedAndStable(t *testing.T) {
	if ollama.MetadataPromptVersion == "" {
		t.Fatal("ollama.MetadataPromptVersion is empty")
	}
	if ollama.MetadataPromptVersion != "metadata-v12" {
		t.Fatalf("MetadataPromptVersion changed to %q — bump intentionally and update this test",
			ollama.MetadataPromptVersion)
	}
}

func TestPickPipeline_OCR(t *testing.T) {
	s, _ := newTestExtractionService(t, &config.Config{
		OllamaBaseURL: "http://127.0.0.1:11434",
		OllamaVision:  "gemma3:latest",
		OllamaText:    "gemma3:latest",
	})
	strategy, engineID, visionModel, _, isOCR := s.PickPipeline(context.Background(), 1, true, 3)
	if strategy != "two_phase_ocr" {
		t.Fatalf("strategy = %q, want %q", strategy, "two_phase_ocr")
	}
	if !isOCR {
		t.Fatal("expected OCR extractor")
	}
	if visionModel != "" {
		t.Fatalf("visionModel = %q, want empty for OCR path", visionModel)
	}
	if engineID != "tesseract:deu+eng" {
		t.Fatalf("engineID = %q, want tesseract:deu+eng", engineID)
	}
}

func TestPickPipeline_OCRLangFromConfig(t *testing.T) {
	s, _ := newTestExtractionService(t, &config.Config{
		OllamaBaseURL: "http://127.0.0.1:11434",
		OllamaVision:  "gemma3:latest",
		OllamaText:    "gemma3:latest",
		OCRLang:       "deu",
	})
	_, engineID, _, _, isOCR := s.PickPipeline(context.Background(), 1, true, 1)
	if !isOCR {
		t.Fatal("expected OCR extractor")
	}
	if engineID != "tesseract:deu" {
		t.Fatalf("engineID = %q, want tesseract:deu (from OCR_LANG)", engineID)
	}
}

func TestPickPipeline_Vision(t *testing.T) {
	s, _ := newTestExtractionService(t, &config.Config{
		OllamaBaseURL: "http://127.0.0.1:11434",
		OllamaVision:  "llama3.2",
		OllamaText:    "llama3.2",
	})
	strategy, engineID, visionModel, textModel, isOCR := s.PickPipeline(context.Background(), 42, false, 1)
	if strategy != "two_phase_vision" {
		t.Fatalf("strategy = %q, want %q", strategy, "two_phase_vision")
	}
	if isOCR {
		t.Fatal("expected vision extractor")
	}
	if visionModel != "llama3.2" || textModel != "llama3.2" {
		t.Fatalf("visionModel=%q textModel=%q", visionModel, textModel)
	}
	if engineID != "vision:"+ollama.UnifiedVisionProfileName {
		t.Fatalf("engineID = %q, want vision:%s", engineID, ollama.UnifiedVisionProfileName)
	}
}
