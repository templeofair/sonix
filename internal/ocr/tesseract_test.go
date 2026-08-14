package ocr

import (
	"strings"
	"testing"

	"github.com/templeofair/sonix/internal/config"
)

func TestTesseract_EngineID(t *testing.T) {
	if got := NewTesseract().EngineID(); got != "tesseract:deu+eng" {
		t.Fatalf("EngineID() = %q, want tesseract:deu+eng", got)
	}
	custom := NewTesseractOptions(TesseractOptions{Lang: "deu"})
	if got := custom.EngineID(); got != "tesseract:deu" {
		t.Fatalf("EngineID() = %q, want tesseract:deu", got)
	}
}

func TestTesseract_buildArgs(t *testing.T) {
	tess := NewTesseractOptions(TesseractOptions{Lang: "deu+eng", DPI: 300, PSM: "1"})
	args := tess.buildArgs("/tmp/page.jpg", "")
	joined := strings.Join(args, " ")
	wantParts := []string{"/tmp/page.jpg", "stdout", "-l", "deu+eng", "--dpi", "300", "--psm", "1"}
	for i, part := range wantParts {
		if i >= len(args) || args[i] != part {
			t.Fatalf("buildArgs = %v\njoined = %q\nwant prefix %v", args, joined, wantParts)
		}
	}
}

func TestTesseract_buildArgs_LangOverride(t *testing.T) {
	tess := NewTesseractOptions(TesseractOptions{Lang: "deu+eng"})
	args := tess.buildArgs("/img.jpg", "eng")
	if args[3] != "eng" {
		t.Fatalf("lang override = %q, want eng; args=%v", args[3], args)
	}
}

func TestResolveLang(t *testing.T) {
	available := map[string]bool{"deu": true, "eng": true, "osd": true}

	got, warn := ResolveLang("deu+eng", available)
	if got != "deu+eng" || warn != "" {
		t.Fatalf("ResolveLang(deu+eng) = %q, %q", got, warn)
	}

	got, warn = ResolveLang("deu+fra", available)
	if got != "deu" || warn == "" {
		t.Fatalf("ResolveLang(deu+fra) = %q, %q; want deu with warning", got, warn)
	}

	got, warn = ResolveLang("fra+spa", available)
	if got != "eng" || warn == "" {
		t.Fatalf("ResolveLang(missing) = %q, %q; want eng fallback", got, warn)
	}

	got, warn = ResolveLang("", available)
	if got != DefaultLang {
		t.Fatalf("ResolveLang(empty) = %q, want %q (warn=%q)", got, DefaultLang, warn)
	}
}

func TestNewProviderFromConfig_Default(t *testing.T) {
	p, err := NewProviderFromConfig(nil)
	if err != nil {
		t.Fatal(err)
	}
	tess, ok := p.(*Tesseract)
	if !ok {
		t.Fatalf("want *Tesseract, got %T", p)
	}
	if tess.Lang != DefaultLang {
		t.Fatalf("Lang = %q, want %q", tess.Lang, DefaultLang)
	}
}

func TestNewProviderFromConfig_TesseractExplicit(t *testing.T) {
	p, err := NewProviderFromConfig(&config.Config{
		OCREngine: "tesseract",
		OCRLang:   "deu",
		OCRDPI:    300,
		OCRPSM:    "6",
	})
	if err != nil {
		t.Fatal(err)
	}
	tess, ok := p.(*Tesseract)
	if !ok {
		t.Fatalf("want *Tesseract, got %T", p)
	}
	// Without tesseract binary, lang stays as configured (deu).
	if tess.Lang != "deu" && tess.Lang != "eng" {
		t.Fatalf("Lang = %q, want deu (or eng fallback if langs listed without deu)", tess.Lang)
	}
	if tess.DPI != 300 {
		t.Fatalf("DPI = %d, want 300", tess.DPI)
	}
	if tess.PSM != "6" {
		t.Fatalf("PSM = %q, want 6", tess.PSM)
	}
}

func TestNewProviderFromConfig_UnknownEngine(t *testing.T) {
	_, err := NewProviderFromConfig(&config.Config{OCREngine: "paddle"})
	if err == nil {
		t.Fatal("expected error for unknown OCR_ENGINE")
	}
}
