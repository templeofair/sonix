package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// PageStorage persists uploaded page images and PDF→PNG conversion under the uploads root.
type PageStorage interface {
	StoreImage(docID int64, pageIndex int, contentType string, r io.Reader) (relPath string, err error)
	StorePDF(ctx context.Context, docID int64, startIndex, maxNewPages int, r io.Reader) (relPaths []string, err error)
}

// LocalPageStorage writes under UploadsPath using the filesystem and pdftoppm.
type LocalPageStorage struct {
	UploadsPath string
	// ConvertPDF defaults to pdftoppm conversion; override in tests.
	ConvertPDF func(ctx context.Context, pdfPath, docDir string, startIndex, maxNewPages int) ([]string, error)
	MaxPages   int
	PDFTimeout time.Duration
}

// DefaultPDFConvertTimeout is used when PDF_CONVERT_TIMEOUT_SECONDS is unset.
const DefaultPDFConvertTimeout = 120 * time.Second

var allowedUploadTypes = map[string]bool{
	"image/jpeg":      true,
	"image/png":       true,
	"image/webp":      true,
	"application/pdf": true,
}

// IsAllowedUploadType reports whether contentType is accepted for page upload.
func IsAllowedUploadType(contentType string) bool {
	return allowedUploadTypes[contentType]
}

func (s *LocalPageStorage) convertPDF() func(ctx context.Context, pdfPath, docDir string, startIndex, maxNewPages int) ([]string, error) {
	if s.ConvertPDF != nil {
		return s.ConvertPDF
	}
	timeout := s.PDFTimeout
	if timeout <= 0 {
		timeout = DefaultPDFConvertTimeout
	}
	return func(ctx context.Context, pdfPath, docDir string, startIndex, maxNewPages int) ([]string, error) {
		return convertPDFToImages(ctx, pdfPath, docDir, startIndex, maxNewPages, timeout)
	}
}

func (s *LocalPageStorage) StoreImage(docID int64, pageIndex int, contentType string, r io.Reader) (string, error) {
	docDir := filepath.Join(s.UploadsPath, strconv.FormatInt(docID, 10))
	if err := os.MkdirAll(docDir, 0700); err != nil {
		return "", err
	}
	ext := ".jpg"
	if strings.HasPrefix(contentType, "image/png") {
		ext = ".png"
	} else if strings.HasPrefix(contentType, "image/webp") {
		ext = ".webp"
	}
	storagePath := filepath.Join(docDir, fmt.Sprintf("page_%d%s", pageIndex, ext))
	dst, err := os.Create(storagePath)
	if err != nil {
		return "", err
	}
	_, err = io.Copy(dst, r)
	closeErr := dst.Close()
	if err != nil {
		return "", err
	}
	if closeErr != nil {
		return "", closeErr
	}
	relPath := filepath.Join(strconv.FormatInt(docID, 10), filepath.Base(storagePath))
	return relPath, nil
}

func (s *LocalPageStorage) StorePDF(ctx context.Context, docID int64, startIndex, maxNewPages int, r io.Reader) ([]string, error) {
	docDir := filepath.Join(s.UploadsPath, strconv.FormatInt(docID, 10))
	if err := os.MkdirAll(docDir, 0700); err != nil {
		return nil, err
	}
	tmpF, err := os.CreateTemp("", "sonix-*.pdf")
	if err != nil {
		return nil, err
	}
	tmpName := tmpF.Name()
	_, err = io.Copy(tmpF, r)
	closeErr := tmpF.Close()
	if err != nil {
		os.Remove(tmpName)
		return nil, err
	}
	if closeErr != nil {
		os.Remove(tmpName)
		return nil, closeErr
	}
	pagePaths, err := s.convertPDF()(ctx, tmpName, docDir, startIndex, maxNewPages)
	os.Remove(tmpName)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPDFConversion, err)
	}
	return pagePaths, nil
}

// convertPDFToImages uses pdftoppm (poppler-utils) to convert a PDF to PNG images.
// Behaviour matches the former server helper (byte-identical naming and DPI).
func convertPDFToImages(ctx context.Context, pdfPath, docDir string, startIndex, maxNewPages int, timeout time.Duration) ([]string, error) {
	if timeout <= 0 {
		timeout = DefaultPDFConvertTimeout
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	baseName := fmt.Sprintf("page_%d", startIndex)
	args := []string{"-png", "-r", "150"}
	if maxNewPages > 0 {
		args = append(args, "-l", strconv.Itoa(maxNewPages))
	}
	args = append(args, pdfPath, baseName)
	cmd := exec.CommandContext(runCtx, "pdftoppm", args...)
	cmd.Dir = docDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, string(out))
	}
	docID := filepath.Base(docDir)
	var relPaths []string
	for i := 1; ; i++ {
		name := fmt.Sprintf("%s-%d.png", baseName, i)
		full := filepath.Join(docDir, name)
		if _, err := os.Stat(full); os.IsNotExist(err) {
			break
		}
		relPaths = append(relPaths, filepath.Join(docID, name))
	}
	if len(relPaths) == 0 {
		return nil, fmt.Errorf("no pages produced")
	}
	return relPaths, nil
}
