package ocr

import "context"

// Provider extracts UTF-8 plain text from a raster image on disk.
// Implementations: Tesseract CLI today; future engines (e.g. PaddleOCR)
// plug in here without changing the extraction pipeline contract.
type Provider interface {
	// ExtractText runs OCR on the image at imagePath. Lang is a Tesseract-style
	// code (e.g. "eng", "deu") where applicable; empty may use a default.
	ExtractText(ctx context.Context, imagePath, lang string) (string, error)
	// EngineID is persisted on extractions.engine_id when this provider is
	// chosen for page text (e.g. "tesseract:eng").
	EngineID() string
}
