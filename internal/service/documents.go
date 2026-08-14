package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/templeofair/sonix/internal/repository"
	xdraw "golang.org/x/image/draw"
)

var (
	// ErrDocumentNotFound is returned when a document id does not exist.
	ErrDocumentNotFound = errors.New("document not found")
	// ErrNoExtraction is returned when updating date without an extractions row.
	ErrNoExtraction = errors.New("no extraction for document")
	// ErrInvalidTextLang is returned when lang is not original or english.
	ErrInvalidTextLang = errors.New("lang must be original or english")
	// ErrUnsupportedContentType is returned for page uploads with a disallowed MIME type.
	ErrUnsupportedContentType = errors.New("unsupported file type")
	// ErrPDFConversion is returned when pdftoppm fails or produces no pages.
	ErrPDFConversion = errors.New("PDF conversion failed")
	// ErrTooManyPages is returned when a letter would exceed DOCUMENT_MAX_PAGES.
	ErrTooManyPages = errors.New("too many pages")
	// ErrInvalidRotateDegrees is returned when rotate degrees are not 90, 180, or 270.
	ErrInvalidRotateDegrees = errors.New("degrees must be 90, 180, or 270")
)

// ThumbnailWidth is the fixed long-edge width for list thumbnails.
const ThumbnailWidth = 320

// DocumentListItem is a list API row.
type DocumentListItem struct {
	ID                 int64
	Title              *string
	Status             string
	CreatedAt          time.Time
	UpdatedAt          time.Time
	DocumentDate       *string
	PageCount          int
	ThumbnailAvailable bool
}

// DocumentListResult is the list payload with total matching rows.
type DocumentListResult struct {
	Documents []DocumentListItem
	Total     int64
}

// PageInfo is page metadata for detail responses.
type PageInfo struct {
	PageIndex   int
	ContentType string
}

// ExtractionInfo is extraction metadata for detail responses.
type ExtractionInfo struct {
	Tags             []string
	Summary          string
	DocumentDate     *string
	ExtractedAt      time.Time
	EngineID         string
	PromptVersion    string
	ExtractionWallMs *int64
	FullTextOriginal *string
	FullTextEnglish  *string
}

// DocumentDetail is the GET /documents/{id} payload (without JSON tags — handlers encode).
type DocumentDetail struct {
	ID                   int64
	Title                *string
	Status               string
	CreatedAt            time.Time
	UpdatedAt            time.Time
	ExtractionError      *string
	ExtractionPagesDone  *int
	ExtractionPagesTotal *int
	Pages                []PageInfo
	Extraction           *ExtractionInfo
	PageCount            int
	ThumbnailAvailable   bool
}

// DocumentService orchestrates document CRUD and related reads.
type DocumentService struct {
	docs        repository.DocumentRepository
	uploadsPath string
	thumbsPath  string
	pages       PageStorage
	pageLimit   int
}

// DefaultDocumentMaxPages is used when DOCUMENT_MAX_PAGES is unset.
const DefaultDocumentMaxPages = 50

// NewDocumentService wires a document repository and optional filesystem roots for thumbnails.
func NewDocumentService(docs repository.DocumentRepository, uploadsPath, thumbsPath string) *DocumentService {
	return &DocumentService{
		docs:        docs,
		uploadsPath: uploadsPath,
		thumbsPath:  thumbsPath,
		pages:       &LocalPageStorage{UploadsPath: uploadsPath, MaxPages: DefaultDocumentMaxPages, PDFTimeout: DefaultPDFConvertTimeout},
		pageLimit:   DefaultDocumentMaxPages,
	}
}

// SetLimits applies page and PDF-conversion caps from config. Zero keeps defaults.
func (s *DocumentService) SetLimits(maxPages int, pdfTimeout time.Duration) {
	if maxPages > 0 {
		s.pageLimit = maxPages
	}
	if ps, ok := s.pages.(*LocalPageStorage); ok {
		ps.MaxPages = s.pageLimit
		if pdfTimeout > 0 {
			ps.PDFTimeout = pdfTimeout
		}
	}
}

func (s *DocumentService) pageCap() int {
	if s.pageLimit > 0 {
		return s.pageLimit
	}
	return DefaultDocumentMaxPages
}

// PageUpload is one multipart file accepted by UploadPages.
type PageUpload struct {
	ContentType string
	Open        func() (io.ReadCloser, error)
}

// Create inserts a pending document and returns its id.
func (s *DocumentService) Create(ctx context.Context, title string) (int64, error) {
	var titleParam interface{}
	if strings.TrimSpace(title) != "" {
		titleParam = strings.TrimSpace(title)
	}
	return s.docs.Create(ctx, titleParam)
}

// UpdateTitle sets the document title.
func (s *DocumentService) UpdateTitle(ctx context.Context, docID int64, title string) (string, error) {
	var titleParam interface{}
	trimmed := strings.TrimSpace(title)
	if trimmed != "" {
		titleParam = trimmed
	}
	ok, err := s.docs.UpdateTitle(ctx, docID, titleParam)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", ErrDocumentNotFound
	}
	return trimmed, nil
}

// List returns filtered documents and the total matching count.
func (s *DocumentService) List(ctx context.Context, f repository.DocumentListFilter) (*DocumentListResult, error) {
	rows, err := s.docs.List(ctx, f)
	if err != nil {
		return nil, err
	}
	total, err := s.docs.Count(ctx, f)
	if err != nil {
		return nil, err
	}
	list := make([]DocumentListItem, 0, len(rows))
	for _, row := range rows {
		d := DocumentListItem{
			ID:                 row.ID,
			Status:             row.Status,
			CreatedAt:          repository.ParseTimeRFC3339(row.CreatedAt),
			UpdatedAt:          repository.ParseTimeRFC3339(row.UpdatedAt),
			PageCount:          row.PageCount,
			ThumbnailAvailable: row.PageCount > 0,
		}
		if row.Title.Valid {
			d.Title = &row.Title.String
		}
		if row.DocumentDate.Valid && row.DocumentDate.String != "" {
			d.DocumentDate = &row.DocumentDate.String
		}
		list = append(list, d)
	}
	return &DocumentListResult{Documents: list, Total: total}, nil
}

// Years returns distinct created_at years.
func (s *DocumentService) Years(ctx context.Context) ([]string, error) {
	return s.docs.Years(ctx)
}

// Tags returns distinct manual tags across the library.
func (s *DocumentService) Tags(ctx context.Context) ([]string, error) {
	return s.docs.Tags(ctx)
}

// DocumentDateYears returns per-year counts by the letter's own date, newest first,
// plus the number of documents with no letter date.
func (s *DocumentService) DocumentDateYears(ctx context.Context) ([]repository.DocumentDateYear, int64, error) {
	return s.docs.DocumentDateYears(ctx)
}

// GetOptions controls optional detail fields.
type GetOptions struct {
	IncludeText bool
}

// Get returns document detail including pages and extraction when present.
func (s *DocumentService) Get(ctx context.Context, docID int64, opts GetOptions) (*DocumentDetail, error) {
	core, err := s.docs.GetCore(ctx, docID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrDocumentNotFound
		}
		return nil, err
	}
	doc := &DocumentDetail{
		ID:        core.ID,
		Status:    core.Status,
		CreatedAt: repository.ParseTimeRFC3339(core.CreatedAt),
		UpdatedAt: repository.ParseTimeRFC3339(core.UpdatedAt),
		Pages:     []PageInfo{},
	}
	if core.Title.Valid {
		doc.Title = &core.Title.String
	}
	if core.ExtractionError.Valid && core.ExtractionError.String != "" {
		msg := PublicExtractionMessage(core.ExtractionError.String)
		doc.ExtractionError = &msg
	}
	if core.ExtractionPagesDone.Valid {
		v := int(core.ExtractionPagesDone.Int64)
		doc.ExtractionPagesDone = &v
	}
	if core.ExtractionPagesTotal.Valid {
		v := int(core.ExtractionPagesTotal.Int64)
		doc.ExtractionPagesTotal = &v
	}

	pages, err := s.docs.ListPages(ctx, docID)
	if err == nil {
		for _, p := range pages {
			doc.Pages = append(doc.Pages, PageInfo{PageIndex: p.PageIndex, ContentType: p.ContentType})
		}
	}
	doc.PageCount = len(doc.Pages)
	doc.ThumbnailAvailable = doc.PageCount > 0

	ext, err := s.docs.GetExtraction(ctx, docID)
	if err == nil {
		var tags []string
		_ = json.Unmarshal([]byte(ext.TagsJSON), &tags)
		if tags == nil {
			tags = []string{}
		}
		info := &ExtractionInfo{
			Tags:          tags,
			Summary:       ext.Summary,
			ExtractedAt:   repository.ParseTimeRFC3339(ext.ExtractedAt),
			EngineID:      ext.EngineID,
			PromptVersion: ext.PromptVersion,
		}
		if ext.ExtractionWallMs.Valid {
			v := ext.ExtractionWallMs.Int64
			info.ExtractionWallMs = &v
		}
		if ext.DocumentDate != "" {
			info.DocumentDate = &ext.DocumentDate
		}
		if opts.IncludeText {
			if ext.FullTextOriginal != "" {
				t := ext.FullTextOriginal
				info.FullTextOriginal = &t
			}
			if ext.FullTextEnglish != "" {
				t := ext.FullTextEnglish
				info.FullTextEnglish = &t
			}
		}
		doc.Extraction = info
	}

	return doc, nil
}

// ThumbnailResult is a path to serve for a page thumbnail (cached or full-size fallback).
type ThumbnailResult struct {
	Path        string
	ContentType string
	FromCache   bool
}

// EnsureThumbnail returns a JPEG thumbnail at ThumbnailWidth, caching under thumbsPath.
// On decode/resize failure it returns the original full image path (caller should still serve it).
func (s *DocumentService) EnsureThumbnail(ctx context.Context, docID int64, pageIndex int) (*ThumbnailResult, error) {
	storagePath, contentType, err := s.docs.GetPage(ctx, docID, pageIndex)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrDocumentNotFound
		}
		return nil, err
	}
	srcPath := filepath.Join(s.uploadsPath, storagePath)
	absUploads, _ := filepath.Abs(s.uploadsPath)
	absSrc, _ := filepath.Abs(srcPath)
	if !PathInside(absUploads, absSrc) {
		return nil, ErrDocumentNotFound
	}
	if _, err := os.Stat(srcPath); err != nil {
		return nil, ErrDocumentNotFound
	}

	cacheDir := filepath.Join(s.thumbsPath, fmt.Sprintf("%d", docID))
	cachePath := filepath.Join(cacheDir, fmt.Sprintf("%d.jpg", pageIndex))
	if st, err := os.Stat(cachePath); err == nil && !st.IsDir() {
		return &ThumbnailResult{Path: cachePath, ContentType: "image/jpeg", FromCache: true}, nil
	}

	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return &ThumbnailResult{Path: srcPath, ContentType: contentType}, nil
	}
	if err := writeResizedJPEG(srcPath, cachePath, ThumbnailWidth); err != nil {
		return &ThumbnailResult{Path: srcPath, ContentType: contentType}, nil
	}
	return &ThumbnailResult{Path: cachePath, ContentType: "image/jpeg", FromCache: true}, nil
}

func writeResizedJPEG(srcPath, dstPath string, width int) error {
	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer f.Close()
	src, _, err := image.Decode(f)
	if err != nil {
		return err
	}
	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()
	if sw <= 0 || sh <= 0 {
		return fmt.Errorf("invalid image size")
	}
	nw := width
	if sw <= width {
		nw = sw
	}
	nh := sh * nw / sw
	if nh < 1 {
		nh = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, b, xdraw.Over, nil)
	tmp := dstPath + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if err := jpeg.Encode(out, dst, &jpeg.Options{Quality: 82}); err != nil {
		_ = out.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dstPath)
}

// UpdateTags upserts tags on the extractions row.
func (s *DocumentService) UpdateTags(ctx context.Context, docID int64, tags []string) ([]string, error) {
	if tags == nil {
		tags = []string{}
	}
	tagsJSON, _ := json.Marshal(tags)
	if err := s.docs.UpsertTags(ctx, docID, string(tagsJSON)); err != nil {
		return nil, err
	}
	return tags, nil
}

// UpdateDocumentDate sets or clears document_date on extractions.
func (s *DocumentService) UpdateDocumentDate(ctx context.Context, docID int64, documentDate *string) (interface{}, error) {
	var docDate interface{}
	if documentDate != nil && *documentDate != "" {
		docDate = *documentDate
	} else {
		docDate = nil
	}
	ok, err := s.docs.UpdateDocumentDate(ctx, docID, docDate)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNoExtraction
	}
	return docDate, nil
}

// Delete removes a document (and FTS row).
func (s *DocumentService) Delete(ctx context.Context, docID int64) error {
	ok, err := s.docs.Delete(ctx, docID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrDocumentNotFound
	}
	return nil
}

// Exists reports whether the document id exists.
func (s *DocumentService) Exists(ctx context.Context, docID int64) (bool, error) {
	return s.docs.Exists(ctx, docID)
}

// NextPageIndex returns max(page_index)+1 for uploads.
func (s *DocumentService) NextPageIndex(ctx context.Context, docID int64) (int, error) {
	maxIndex, err := s.docs.MaxPageIndex(ctx, docID)
	if err != nil {
		return 0, err
	}
	return maxIndex + 1, nil
}

// InsertPage records a page row.
func (s *DocumentService) InsertPage(ctx context.Context, docID int64, storagePath string, pageIndex int, contentType string) error {
	return s.docs.InsertPage(ctx, docID, storagePath, pageIndex, contentType)
}

// UploadPages stores uploaded images/PDFs via PageStorage and records document_pages rows.
func (s *DocumentService) UploadPages(ctx context.Context, docID int64, files []PageUpload) error {
	if len(files) == 0 {
		return fmt.Errorf("no files in request")
	}
	exists, err := s.docs.Exists(ctx, docID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrDocumentNotFound
	}
	nextIndex, err := s.NextPageIndex(ctx, docID)
	if err != nil {
		return err
	}
	for _, fh := range files {
		ct := fh.ContentType
		if !IsAllowedUploadType(ct) {
			return fmt.Errorf("%w: %s", ErrUnsupportedContentType, ct)
		}
		cap := s.pageCap()
		if nextIndex >= cap {
			return ErrTooManyPages
		}
		src, err := fh.Open()
		if err != nil {
			return err
		}
		if ct == "application/pdf" {
			pagePaths, err := s.pages.StorePDF(ctx, docID, nextIndex, cap-nextIndex, src)
			src.Close()
			if err != nil {
				return err
			}
			for j, relPath := range pagePaths {
				if err := s.docs.InsertPage(ctx, docID, relPath, nextIndex+j, "image/png"); err != nil {
					return err
				}
			}
			nextIndex += len(pagePaths)
			continue
		}
		relPath, err := s.pages.StoreImage(docID, nextIndex, ct, src)
		src.Close()
		if err != nil {
			return err
		}
		if err := s.docs.InsertPage(ctx, docID, relPath, nextIndex, ct); err != nil {
			return err
		}
		nextIndex++
	}
	return nil
}

// GetPage returns storage path and content type for an image.
func (s *DocumentService) GetPage(ctx context.Context, docID int64, pageIndex int) (storagePath, contentType string, err error) {
	storagePath, contentType, err = s.docs.GetPage(ctx, docID, pageIndex)
	if errors.Is(err, repository.ErrNotFound) {
		return "", "", ErrDocumentNotFound
	}
	return storagePath, contentType, err
}

// RotatePage rewrites the stored page image rotated clockwise by degrees (90, 180, or 270)
// and invalidates the cached thumbnail for that page.
func (s *DocumentService) RotatePage(ctx context.Context, docID int64, pageIndex, degrees int) error {
	switch degrees {
	case 90, 180, 270:
	default:
		return ErrInvalidRotateDegrees
	}
	storagePath, _, err := s.docs.GetPage(ctx, docID, pageIndex)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrDocumentNotFound
		}
		return err
	}
	srcPath := filepath.Join(s.uploadsPath, storagePath)
	absUploads, _ := filepath.Abs(s.uploadsPath)
	absSrc, _ := filepath.Abs(srcPath)
	if !PathInside(absUploads, absSrc) {
		return ErrDocumentNotFound
	}
	f, err := os.Open(srcPath)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrDocumentNotFound
		}
		return err
	}
	src, _, err := image.Decode(f)
	_ = f.Close()
	if err != nil {
		return fmt.Errorf("decode page: %w", err)
	}
	rotated := rotateImageCW(src, degrees)
	tmp := srcPath + ".rot.tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if err := jpeg.Encode(out, rotated, &jpeg.Options{Quality: 92}); err != nil {
		_ = out.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, srcPath); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	_ = s.docs.UpdatePageContentType(ctx, docID, pageIndex, "image/jpeg")
	cachePath := filepath.Join(s.thumbsPath, fmt.Sprintf("%d", docID), fmt.Sprintf("%d.jpg", pageIndex))
	_ = os.Remove(cachePath)
	return nil
}

func rotateImageCW(src image.Image, degrees int) *image.RGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	var dst *image.RGBA
	switch degrees {
	case 90:
		dst = image.NewRGBA(image.Rect(0, 0, h, w))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				dst.Set(h-1-y, x, src.At(b.Min.X+x, b.Min.Y+y))
			}
		}
	case 180:
		dst = image.NewRGBA(image.Rect(0, 0, w, h))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				dst.Set(w-1-x, h-1-y, src.At(b.Min.X+x, b.Min.Y+y))
			}
		}
	case 270:
		dst = image.NewRGBA(image.Rect(0, 0, h, w))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				dst.Set(y, w-1-x, src.At(b.Min.X+x, b.Min.Y+y))
			}
		}
	default:
		dst = image.NewRGBA(image.Rect(0, 0, w, h))
		xdraw.Draw(dst, dst.Bounds(), src, b.Min, xdraw.Src)
	}
	return dst
}

// GetText returns original or english full text.
func (s *DocumentService) GetText(ctx context.Context, docID int64, lang string) (string, error) {
	if lang != "original" && lang != "english" {
		return "", ErrInvalidTextLang
	}
	text, err := s.docs.GetText(ctx, docID, lang)
	if errors.Is(err, repository.ErrNotFound) {
		return "", ErrDocumentNotFound
	}
	return text, err
}

// ListPageFiles returns pages with storage paths (for export).
func (s *DocumentService) ListPageFiles(ctx context.Context, docID int64) ([]repository.DocumentPageRow, error) {
	return s.docs.ListPages(ctx, docID)
}

// GetExtractionExport returns extraction texts/tags for zip export.
func (s *DocumentService) GetExtractionExport(ctx context.Context, docID int64) (*repository.ExtractionDetail, error) {
	ext, err := s.docs.GetExtraction(ctx, docID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrNoExtraction
	}
	return ext, err
}

// ListIDs returns document ids for export filtering.
func (s *DocumentService) ListIDs(ctx context.Context, f repository.DocumentListFilter) ([]repository.DocumentIDRow, error) {
	return s.docs.ListIDs(ctx, f)
}
