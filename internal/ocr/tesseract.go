package ocr

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Defaults for German letter OCR on phone captures.
const (
	DefaultLang = "deu+eng"
	DefaultDPI  = 300
	// DefaultPSM is Tesseract page segmentation mode 1 (auto + OSD orientation).
	// Phone captures often arrive rotated; without OSD they produce silent garbage.
	DefaultPSM = "1"
)

// Tesseract runs the `tesseract` CLI: one process per page image.
type Tesseract struct {
	Lang string
	// DPI is an optional fixed --dpi override (OCR_DPI). Zero means estimate
	// per image from pixel height assuming A4 (see ResolveDPI).
	DPI int
	PSM string
}

// TesseractOptions configures language, DPI hint, and page segmentation mode.
type TesseractOptions struct {
	Lang string
	DPI  int
	PSM  string
}

// NewTesseract returns Tesseract with project defaults (deu+eng, 300 DPI, PSM 1).
func NewTesseract() *Tesseract {
	return NewTesseractOptions(TesseractOptions{})
}

// NewTesseractOptions returns a configured Tesseract backend.
func NewTesseractOptions(opts TesseractOptions) *Tesseract {
	lang := strings.TrimSpace(opts.Lang)
	if lang == "" {
		lang = DefaultLang
	}
	psm := strings.TrimSpace(opts.PSM)
	if psm == "" {
		psm = DefaultPSM
	}
	// DPI 0 = auto (estimated per page). Positive values force --dpi.
	return &Tesseract{Lang: lang, DPI: opts.DPI, PSM: psm}
}

// EngineID reports the language actually used for provenance on new extractions rows.
func (t *Tesseract) EngineID() string {
	lang := strings.TrimSpace(t.Lang)
	if lang == "" {
		lang = DefaultLang
	}
	return "tesseract:" + lang
}

// buildArgs constructs the tesseract CLI argv (excluding the binary name).
// When t.DPI <= 0, --dpi is estimated from the image's pixel size (A4 assumption).
func (t *Tesseract) buildArgs(imagePath, lang string) []string {
	if strings.TrimSpace(lang) == "" {
		lang = t.Lang
	}
	if strings.TrimSpace(lang) == "" {
		lang = DefaultLang
	}
	dpi := ResolveDPI(imagePath, t.DPI)
	psm := strings.TrimSpace(t.PSM)
	if psm == "" {
		psm = DefaultPSM
	}
	return []string{
		imagePath, "stdout",
		"-l", lang,
		"--dpi", strconv.Itoa(dpi),
		"--psm", psm,
	}
}

// ExtractText runs Tesseract on imagePath and returns stdout text.
// Lang overrides the instance default when non-empty.
func (t *Tesseract) ExtractText(ctx context.Context, imagePath, lang string) (string, error) {
	args := t.buildArgs(imagePath, lang)
	cmd := exec.CommandContext(ctx, "tesseract", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg != "" {
			return "", fmt.Errorf("%w: %s", err, msg)
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// ListInstalledLangs runs `tesseract --list-langs` and returns language codes.
func ListInstalledLangs(ctx context.Context) (map[string]bool, error) {
	cmd := exec.CommandContext(ctx, "tesseract", "--list-langs")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("tesseract --list-langs: %w: %s", err, strings.TrimSpace(string(out)))
	}
	available := make(map[string]bool)
	for _, line := range strings.Split(string(out), "\n") {
		code := strings.TrimSpace(line)
		if code == "" || strings.HasPrefix(strings.ToLower(code), "list of") {
			continue
		}
		available[code] = true
	}
	return available, nil
}

// ResolveLang checks that each component of a Tesseract lang string (e.g. "deu+eng")
// is installed. Missing components are dropped; if none remain, falls back to "eng"
// when available. Returns the resolved lang and an optional warning for the operator.
func ResolveLang(requested string, available map[string]bool) (resolved string, warning string) {
	requested = strings.TrimSpace(requested)
	if requested == "" {
		requested = DefaultLang
	}
	parts := strings.Split(requested, "+")
	ok := make([]string, 0, len(parts))
	missing := make([]string, 0)
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if available == nil || available[p] {
			ok = append(ok, p)
			continue
		}
		missing = append(missing, p)
	}
	if len(ok) > 0 {
		resolved = strings.Join(ok, "+")
		if len(missing) > 0 {
			warning = fmt.Sprintf("OCR_LANG %q missing installed data for %s; using %q", requested, strings.Join(missing, ", "), resolved)
		}
		return resolved, warning
	}
	if available == nil || available["eng"] {
		return "eng", fmt.Sprintf("OCR_LANG %q has no installed language packs; falling back to eng", requested)
	}
	// Last resort: keep the request so ExtractText surfaces a clear Tesseract error.
	return requested, fmt.Sprintf("OCR_LANG %q has no installed language packs and eng is unavailable", requested)
}
