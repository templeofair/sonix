package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config holds application configuration from environment.
type Config struct {
	SessionSecret string
	DatabasePath  string
	DataDir       string // for uploads and SQLite if path is relative
	OllamaBaseURL string
	OllamaVision  string
	OllamaText    string
	ServerAddr    string
	// OCREngine selects the page-OCR implementation when use_ocr is true (e.g. "tesseract").
	// From OCR_ENGINE; empty means tesseract.
	OCREngine string
	// OCRLang is the Tesseract -l value (e.g. "deu+eng"). From OCR_LANG; empty → deu+eng.
	OCRLang string
	// OCRDPI forces Tesseract --dpi when > 0 (from OCR_DPI). Zero = estimate per
	// page from pixel size assuming A4 (see ocr.ResolveDPI).
	OCRDPI int
	// OCRPSM is Tesseract --psm. From OCR_PSM; empty → "1" (auto + OSD).
	OCRPSM string
	// ExtractionJobTimeoutMin is the wall-clock budget for one document job
	// (vision + text). From EXTRACTION_JOB_TIMEOUT_MINUTES; 0 → 60.
	ExtractionJobTimeoutMin int
	// ExtractionMaxConcurrent is how many document jobs may run at once.
	// From EXTRACTION_MAX_CONCURRENT; 0 → 1.
	ExtractionMaxConcurrent int
	// DocumentMaxPages caps pages on one letter (upload + PDF conversion).
	// From DOCUMENT_MAX_PAGES; 0 → 50.
	DocumentMaxPages int
	// PDFConvertTimeoutSec is the pdftoppm budget. From PDF_CONVERT_TIMEOUT_SECONDS; 0 → 120.
	PDFConvertTimeoutSec int
	// ExtractionPipeline selects the text/metadata path: "v2" (page-wise
	// translate + summary + document_date) or "v1" (legacy single ExtractMetadata).
	// From EXTRACTION_PIPELINE; empty → "v2".
	ExtractionPipeline string
	// ImportInboxDir is the hardware/scan drop folder. Empty → DATA_DIR/inbox.
	ImportInboxDir string
	// ImportInboxEnabledDefault is the env default when Settings has no override.
	// From IMPORT_INBOX_ENABLED; empty/false → off until Settings enables it.
	ImportInboxEnabledDefault bool
	// ImportAutoExtractDefault starts extraction after inbox import when Settings
	// has no override. From IMPORT_AUTO_EXTRACT; empty → true.
	ImportAutoExtractDefault bool
}

// Load reads config from environment. Caller should validate (e.g. SessionSecret required).
func Load() *Config {
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "data"
	}
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = filepath.Join(dataDir, "sonix.db")
	}
	ollama := os.Getenv("OLLAMA_BASE_URL")
	if ollama == "" {
		ollama = "http://localhost:11434"
	}
	vision := os.Getenv("OLLAMA_VISION_MODEL")
	if vision == "" {
		vision = "llava"
	}
	text := os.Getenv("OLLAMA_TEXT_MODEL")
	if text == "" {
		text = "llama3.2"
	}
	addr := os.Getenv("SERVER_ADDR")
	if addr == "" {
		// 9080 avoids Open WebUI / other apps that commonly bind 8080; Ollama stays on 11434.
		addr = ":9080"
	}
	ocrDPI := 0
	if v := strings.TrimSpace(os.Getenv("OCR_DPI")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			ocrDPI = n
		}
	}
	jobTimeoutMin := envInt("EXTRACTION_JOB_TIMEOUT_MINUTES", 0)
	pipeline := strings.ToLower(strings.TrimSpace(os.Getenv("EXTRACTION_PIPELINE")))
	if pipeline != "v1" {
		pipeline = "v2"
	}
	inboxDir := strings.TrimSpace(os.Getenv("IMPORT_INBOX_DIR"))
	if inboxDir == "" {
		inboxDir = filepath.Join(dataDir, "inbox")
	}
	importEnabled := envBool(os.Getenv("IMPORT_INBOX_ENABLED"), false)
	autoExtract := envBoolDefaultTrue(os.Getenv("IMPORT_AUTO_EXTRACT"))
	return &Config{
		SessionSecret:             os.Getenv("SESSION_SECRET"),
		DatabasePath:              dbPath,
		DataDir:                   dataDir,
		OllamaBaseURL:             ollama,
		OllamaVision:              vision,
		OllamaText:                text,
		ServerAddr:                addr,
		OCREngine:                 os.Getenv("OCR_ENGINE"),
		OCRLang:                   strings.TrimSpace(os.Getenv("OCR_LANG")),
		OCRDPI:                    ocrDPI,
		OCRPSM:                    strings.TrimSpace(os.Getenv("OCR_PSM")),
		ExtractionJobTimeoutMin:   jobTimeoutMin,
		ExtractionMaxConcurrent:   envInt("EXTRACTION_MAX_CONCURRENT", 1),
		DocumentMaxPages:          envInt("DOCUMENT_MAX_PAGES", 50),
		PDFConvertTimeoutSec:      envInt("PDF_CONVERT_TIMEOUT_SECONDS", 120),
		ExtractionPipeline:        pipeline,
		ImportInboxDir:            inboxDir,
		ImportInboxEnabledDefault: importEnabled,
		ImportAutoExtractDefault:  autoExtract,
	}
}

func envInt(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

func envBool(v string, def bool) bool {
	v = strings.TrimSpace(strings.ToLower(v))
	if v == "" {
		return def
	}
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func envBoolDefaultTrue(v string) bool {
	v = strings.TrimSpace(strings.ToLower(v))
	if v == "" {
		return true
	}
	return v == "1" || v == "true" || v == "yes" || v == "on"
}
