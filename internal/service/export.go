package service

import (
	"archive/zip"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/templeofair/sonix/internal/repository"
)

// ExportService builds filtered document zip archives.
type ExportService struct {
	docs        *DocumentService
	uploadsPath string
}

// NewExportService wires document access and uploads path.
func NewExportService(docs *DocumentService, uploadsPath string) *ExportService {
	return &ExportService{docs: docs, uploadsPath: uploadsPath}
}

// ListMatching returns document ids for the export filter (call before writing headers).
func (s *ExportService) ListMatching(ctx context.Context, filter repository.DocumentListFilter) ([]repository.DocumentIDRow, error) {
	return s.docs.ListIDs(ctx, filter)
}

// WriteZip streams a zip of the given documents to w.
func (s *ExportService) WriteZip(ctx context.Context, w io.Writer, docIDs []repository.DocumentIDRow) error {
	zw := zip.NewWriter(w)
	defer zw.Close()

	for _, doc := range docIDs {
		t, _ := time.Parse(time.RFC3339, doc.CreatedAt)
		year := t.Format("2006")
		month := t.Format("01")
		prefix := year + "/" + month + "/letter_" + strconv.FormatInt(doc.ID, 10)
		if err := s.addDocumentToZip(ctx, zw, doc.ID, prefix); err != nil {
			continue
		}
	}
	return nil
}

func (s *ExportService) addDocumentToZip(ctx context.Context, zw *zip.Writer, docID int64, prefix string) error {
	pages, err := s.docs.ListPageFiles(ctx, docID)
	if err != nil {
		return err
	}
	pageNum := 1
	for _, p := range pages {
		fullPath := filepath.Join(s.uploadsPath, p.StoragePath)
		src, err := os.Open(fullPath)
		if err != nil {
			return err
		}
		name := prefix + "/page_" + strconv.Itoa(pageNum) + filepath.Ext(p.StoragePath)
		f, err := zw.Create(name)
		if err != nil {
			src.Close()
			return err
		}
		_, copyErr := io.Copy(f, src)
		src.Close()
		if copyErr != nil {
			return copyErr
		}
		pageNum++
	}

	ext, err := s.docs.GetExtractionExport(ctx, docID)
	if err != nil {
		return nil
	}
	var tags []string
	_ = json.Unmarshal([]byte(ext.TagsJSON), &tags)
	meta := map[string]any{"tags": tags, "summary": ext.Summary}
	metaBytes, _ := json.MarshalIndent(meta, "", "  ")
	f, _ := zw.Create(prefix + "/metadata.json")
	_, _ = f.Write(metaBytes)
	f2, _ := zw.Create(prefix + "/original.txt")
	_, _ = f2.Write([]byte(ext.FullTextOriginal))
	f3, _ := zw.Create(prefix + "/translated.txt")
	_, _ = f3.Write([]byte(ext.FullTextEnglish))
	return nil
}
