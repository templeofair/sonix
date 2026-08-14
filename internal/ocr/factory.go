package ocr

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/templeofair/sonix/internal/config"
)

// NewProviderFromConfig selects an OCR implementation from cfg (OCR_ENGINE).
// Supported values: empty, "tesseract". Other engines return an error until implemented.
// For Tesseract, applies OCR_LANG / OCR_DPI / OCR_PSM and validates language packs when
// `tesseract --list-langs` is available (skipped with a log line if the binary is missing).
func NewProviderFromConfig(cfg *config.Config) (Provider, error) {
	if cfg == nil {
		return NewTesseract(), nil
	}
	switch strings.ToLower(strings.TrimSpace(cfg.OCREngine)) {
	case "", "tesseract":
		opts := TesseractOptions{
			Lang: cfg.OCRLang,
			DPI:  cfg.OCRDPI,
			PSM:  cfg.OCRPSM,
		}
		t := NewTesseractOptions(opts)
		available, err := ListInstalledLangs(context.Background())
		if err != nil {
			// Common in unit tests / hosts without Tesseract; keep configured lang.
			log.Printf("ocr: could not list Tesseract languages (%v); using OCR_LANG=%q", err, t.Lang)
			return t, nil
		}
		resolved, warning := ResolveLang(t.Lang, available)
		if warning != "" {
			log.Printf("ocr: %s", warning)
		}
		t.Lang = resolved
		return t, nil
	default:
		return nil, fmt.Errorf("ocr: unknown OCR_ENGINE %q (supported: tesseract)", cfg.OCREngine)
	}
}
