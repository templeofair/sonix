package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/templeofair/sonix/internal/config"
	"github.com/templeofair/sonix/internal/repository"
)

const defaultOllamaPort = "11434"

// ErrInvalidPrinterIP is returned when Settings printer IP fails validation.
var ErrInvalidPrinterIP = errors.New("invalid printer IP")

// ErrInvalidOllamaURL is returned when an Ollama base URL is not allowed.
var ErrInvalidOllamaURL = errors.New("invalid Ollama URL")

// SettingsView is the GET /settings JSON payload.
type SettingsView struct {
	OllamaBaseURL      string
	OllamaBaseURLRaw   string
	OllamaModel        string // effective vision model
	OllamaModelRaw     string
	OllamaTextModel    string // effective text model
	OllamaTextModelRaw string

	ImportInboxEnabled  bool
	ImportAutoExtract   bool
	ImportExtractUseOCR bool
	ImportLastFile      string
	ImportLastError     string
	ImportLastAt        string
	HPPrinterIP         string
}

// SettingsUpdateResult is the PUT /settings JSON payload.
type SettingsUpdateResult struct {
	OllamaBaseURL       string
	OllamaModel         string
	OllamaTextModel     string
	ImportInboxEnabled  bool
	ImportAutoExtract   bool
	ImportExtractUseOCR bool
	HPPrinterIP         string
}

// OllamaTestResult is the POST /settings/ollama/test JSON payload.
type OllamaTestResult struct {
	OK    bool
	Error string
}

// PrinterTestResult is the POST /settings/printer/test JSON payload.
type PrinterTestResult struct {
	OK    bool
	Error string
}

// SettingsService resolves and persists Ollama settings.
type SettingsService struct {
	repo repository.SettingsRepository
	cfg  *config.Config
}

// NewSettingsService wires settings repository and config fallbacks.
func NewSettingsService(repo repository.SettingsRepository, cfg *config.Config) *SettingsService {
	return &SettingsService{repo: repo, cfg: cfg}
}

// EffectiveBaseURL returns settings URL if set, else config.
func (s *SettingsService) EffectiveBaseURL(ctx context.Context) string {
	stored, err := s.repo.Get(ctx, repository.SettingsKeyOllamaBaseURL)
	if err == nil && strings.TrimSpace(stored) != "" {
		return NormalizeOllamaURL(stored)
	}
	if s.cfg == nil {
		return ""
	}
	return s.cfg.OllamaBaseURL
}

// EffectiveVisionModel returns the model used for page scanning (vision OCR).
func (s *SettingsService) EffectiveVisionModel(ctx context.Context) string {
	stored, err := s.repo.Get(ctx, repository.SettingsKeyOllamaModel)
	if err == nil && strings.TrimSpace(stored) != "" {
		return strings.TrimSpace(stored)
	}
	if s.cfg != nil && s.cfg.OllamaVision != "" {
		return s.cfg.OllamaVision
	}
	return "gemma3:latest"
}

// EffectiveTextModel returns the model used for translation, summary, and date.
// When the text setting is empty, a Settings vision model (if any) is reused so
// existing single-model installs keep working; otherwise env OLLAMA_TEXT_MODEL
// and finally the vision model are used.
func (s *SettingsService) EffectiveTextModel(ctx context.Context) string {
	stored, err := s.repo.Get(ctx, repository.SettingsKeyOllamaTextModel)
	if err == nil && strings.TrimSpace(stored) != "" {
		return strings.TrimSpace(stored)
	}
	visionStored, _ := s.getRaw(ctx, repository.SettingsKeyOllamaModel)
	if strings.TrimSpace(visionStored) != "" {
		return strings.TrimSpace(visionStored)
	}
	if s.cfg != nil && s.cfg.OllamaText != "" {
		return s.cfg.OllamaText
	}
	return s.EffectiveVisionModel(ctx)
}

// EffectiveModel is kept for older callers; it is the vision model.
func (s *SettingsService) EffectiveModel(ctx context.Context) string {
	return s.EffectiveVisionModel(ctx)
}

// Get returns effective and raw settings for the API.
func (s *SettingsService) Get(ctx context.Context) (*SettingsView, error) {
	urlStored, _ := s.getRaw(ctx, repository.SettingsKeyOllamaBaseURL)
	modelStored, _ := s.getRaw(ctx, repository.SettingsKeyOllamaModel)
	textStored, _ := s.getRaw(ctx, repository.SettingsKeyOllamaTextModel)
	lastFile, _ := s.getRaw(ctx, repository.SettingsKeyImportLastFile)
	lastErr, _ := s.getRaw(ctx, repository.SettingsKeyImportLastError)
	lastAt, _ := s.getRaw(ctx, repository.SettingsKeyImportLastAt)
	printerIP, _ := s.getRaw(ctx, repository.SettingsKeyHPPrinterIP)

	effectiveURL := ""
	if s.cfg != nil {
		effectiveURL = s.cfg.OllamaBaseURL
	}
	if strings.TrimSpace(urlStored) != "" {
		effectiveURL = NormalizeOllamaURL(urlStored)
	}

	return &SettingsView{
		OllamaBaseURL:       effectiveURL,
		OllamaBaseURLRaw:    strings.TrimSpace(urlStored),
		OllamaModel:         s.EffectiveVisionModel(ctx),
		OllamaModelRaw:      strings.TrimSpace(modelStored),
		OllamaTextModel:     s.EffectiveTextModel(ctx),
		OllamaTextModelRaw:  strings.TrimSpace(textStored),
		ImportInboxEnabled:  s.ImportInboxEnabled(ctx),
		ImportAutoExtract:   s.ImportAutoExtract(ctx),
		ImportExtractUseOCR: s.ImportExtractUseOCR(ctx),
		ImportLastFile:      lastFile,
		ImportLastError:     PublicImportMessage(lastErr),
		ImportLastAt:        lastAt,
		HPPrinterIP:         strings.TrimSpace(printerIP),
	}, nil
}

// Update persists Ollama and optional inbox import toggles. Import fields and
// printer IP are only written when the corresponding pointer is non-nil.
func (s *SettingsService) Update(ctx context.Context, ollamaBaseURL, ollamaModel, ollamaTextModel string, importEnabled, importAutoExtract, importExtractUseOCR *bool, hpPrinterIP *string) (*SettingsUpdateResult, error) {
	rawURL := strings.TrimSpace(ollamaBaseURL)
	if rawURL == "" {
		_ = s.repo.Delete(ctx, repository.SettingsKeyOllamaBaseURL)
	} else {
		if err := ValidateOllamaURL(rawURL); err != nil {
			return nil, err
		}
		if err := s.repo.Set(ctx, repository.SettingsKeyOllamaBaseURL, rawURL); err != nil {
			return nil, err
		}
	}

	rawModel := strings.TrimSpace(ollamaModel)
	if rawModel == "" {
		_ = s.repo.Delete(ctx, repository.SettingsKeyOllamaModel)
	} else if err := s.repo.Set(ctx, repository.SettingsKeyOllamaModel, rawModel); err != nil {
		return nil, err
	}

	rawText := strings.TrimSpace(ollamaTextModel)
	if rawText == "" {
		_ = s.repo.Delete(ctx, repository.SettingsKeyOllamaTextModel)
	} else if err := s.repo.Set(ctx, repository.SettingsKeyOllamaTextModel, rawText); err != nil {
		return nil, err
	}

	if importEnabled != nil {
		v := "false"
		if *importEnabled {
			v = "true"
		}
		if err := s.repo.Set(ctx, repository.SettingsKeyImportInboxEnabled, v); err != nil {
			return nil, err
		}
	}
	if importAutoExtract != nil {
		v := "false"
		if *importAutoExtract {
			v = "true"
		}
		if err := s.repo.Set(ctx, repository.SettingsKeyImportAutoExtract, v); err != nil {
			return nil, err
		}
	}
	if importExtractUseOCR != nil {
		v := "false"
		if *importExtractUseOCR {
			v = "true"
		}
		if err := s.repo.Set(ctx, repository.SettingsKeyImportExtractUseOCR, v); err != nil {
			return nil, err
		}
	}
	if hpPrinterIP != nil {
		ip := strings.TrimSpace(*hpPrinterIP)
		if err := validatePrinterIP(ip); err != nil {
			return nil, err
		}
		if ip == "" {
			_ = s.repo.Delete(ctx, repository.SettingsKeyHPPrinterIP)
		} else if err := s.repo.Set(ctx, repository.SettingsKeyHPPrinterIP, ip); err != nil {
			return nil, err
		}
		if err := s.writeHPPrinterIPFile(ip); err != nil {
			return nil, err
		}
	}

	effectiveURL := ""
	if s.cfg != nil {
		effectiveURL = s.cfg.OllamaBaseURL
	}
	if rawURL != "" {
		effectiveURL = NormalizeOllamaURL(rawURL)
	}
	printerIP, _ := s.getRaw(ctx, repository.SettingsKeyHPPrinterIP)
	return &SettingsUpdateResult{
		OllamaBaseURL:       effectiveURL,
		OllamaModel:         s.EffectiveVisionModel(ctx),
		OllamaTextModel:     s.EffectiveTextModel(ctx),
		ImportInboxEnabled:  s.ImportInboxEnabled(ctx),
		ImportAutoExtract:   s.ImportAutoExtract(ctx),
		ImportExtractUseOCR: s.ImportExtractUseOCR(ctx),
		HPPrinterIP:         strings.TrimSpace(printerIP),
	}, nil
}

// ImportInboxEnabled reports whether the inbox watcher should consume files.
func (s *SettingsService) ImportInboxEnabled(ctx context.Context) bool {
	stored, err := s.getRaw(ctx, repository.SettingsKeyImportInboxEnabled)
	if err == nil && strings.TrimSpace(stored) != "" {
		return parseBoolSetting(stored)
	}
	if s.cfg != nil {
		return s.cfg.ImportInboxEnabledDefault
	}
	return false
}

// ImportAutoExtract reports whether inbox imports should start extraction.
func (s *SettingsService) ImportAutoExtract(ctx context.Context) bool {
	stored, err := s.getRaw(ctx, repository.SettingsKeyImportAutoExtract)
	if err == nil && strings.TrimSpace(stored) != "" {
		return parseBoolSetting(stored)
	}
	if s.cfg != nil {
		return s.cfg.ImportAutoExtractDefault
	}
	return true
}

// ImportExtractUseOCR reports whether auto-import extraction should use Tesseract.
// Default true (OCR) when unset.
func (s *SettingsService) ImportExtractUseOCR(ctx context.Context) bool {
	stored, err := s.getRaw(ctx, repository.SettingsKeyImportExtractUseOCR)
	if err == nil && strings.TrimSpace(stored) != "" {
		return parseBoolSetting(stored)
	}
	return true
}

// RecordImportResult stores the last inbox import outcome for Settings display.
func (s *SettingsService) RecordImportResult(ctx context.Context, filename, errMsg string) error {
	if s == nil || s.repo == nil {
		return nil
	}
	_ = s.repo.Set(ctx, repository.SettingsKeyImportLastFile, filename)
	_ = s.repo.Set(ctx, repository.SettingsKeyImportLastAt, time.Now().UTC().Format(time.RFC3339))
	if errMsg == "" {
		_ = s.repo.Delete(ctx, repository.SettingsKeyImportLastError)
	} else {
		_ = s.repo.Set(ctx, repository.SettingsKeyImportLastError, errMsg)
	}
	return nil
}

// SyncHPPrinterIPFile writes the stored printer IP to the shared volume for the helper.
// Safe to call at startup.
func (s *SettingsService) SyncHPPrinterIPFile(ctx context.Context) error {
	ip, _ := s.getRaw(ctx, repository.SettingsKeyHPPrinterIP)
	return s.writeHPPrinterIPFile(strings.TrimSpace(ip))
}

func (s *SettingsService) writeHPPrinterIPFile(ip string) error {
	if s.cfg == nil || strings.TrimSpace(s.cfg.DataDir) == "" {
		return nil
	}
	dir := filepath.Join(s.cfg.DataDir, "hp-scan")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	path := filepath.Join(dir, "printer_ip")
	if ip == "" {
		_ = os.Remove(path)
		return nil
	}
	return os.WriteFile(path, []byte(ip+"\n"), 0600)
}

func validatePrinterIP(ip string) error {
	if ip == "" {
		return nil
	}
	parsed := net.ParseIP(ip)
	if parsed == nil || parsed.To4() == nil {
		return fmt.Errorf("%w: use an IPv4 address (e.g. 192.168.1.50)", ErrInvalidPrinterIP)
	}
	return nil
}

func parseBoolSetting(v string) bool {
	v = strings.TrimSpace(strings.ToLower(v))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// TestOllama probes GET {base}/api/tags and checks that the configured vision
// and text models are present (with or without a :tag suffix).
func (s *SettingsService) TestOllama(ctx context.Context) OllamaTestResult {
	baseURL := s.EffectiveBaseURL(ctx)
	if baseURL == "" {
		return OllamaTestResult{OK: false, Error: "Ollama URL not configured"}
	}
	if err := ValidateOllamaURL(baseURL); err != nil {
		return OllamaTestResult{OK: false, Error: "Ollama URL is not allowed"}
	}
	u := strings.TrimSuffix(baseURL, "/") + "/api/tags"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		log.Printf("ollama test: build request failed: %v", err)
		return OllamaTestResult{OK: false, Error: PublicOllamaTestMessage()}
	}
	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return errors.New("too many redirects")
			}
			if err := ValidateOllamaURL(req.URL.String()); err != nil {
				return err
			}
			return nil
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("ollama test: request failed: %v", err)
		return OllamaTestResult{OK: false, Error: PublicOllamaTestMessage()}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		log.Printf("ollama test: %s %s", resp.Status, string(body))
		return OllamaTestResult{OK: false, Error: "Ollama returned " + resp.Status}
	}

	var tags struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &tags); err != nil {
		return OllamaTestResult{OK: false, Error: "could not read Ollama model list"}
	}
	names := make([]string, 0, len(tags.Models))
	for _, m := range tags.Models {
		names = append(names, m.Name)
	}

	vision := s.EffectiveVisionModel(ctx)
	text := s.EffectiveTextModel(ctx)
	var missing []string
	if !ollamaModelPresent(names, vision) {
		missing = append(missing, "scanning model "+vision)
	}
	if text != vision && !ollamaModelPresent(names, text) {
		missing = append(missing, "translation model "+text)
	}
	if len(missing) > 0 {
		return OllamaTestResult{
			OK:    false,
			Error: "not found on Ollama: " + strings.Join(missing, "; ") + ". Pull them, then test again.",
		}
	}
	return OllamaTestResult{OK: true}
}

// TestPrinter checks that the saved OfficeJet (or similar) answers on the LAN.
// Probes TCP :80 then HP scan HTTP endpoints used by the scan helper.
// The request body is ignored; save Printer IP first, then test.
func (s *SettingsService) TestPrinter(ctx context.Context) PrinterTestResult {
	stored, _ := s.getRaw(ctx, repository.SettingsKeyHPPrinterIP)
	ip := strings.TrimSpace(stored)
	if ip == "" {
		return PrinterTestResult{OK: false, Error: "Enter a printer IP first"}
	}
	if err := validatePrinterIP(ip); err != nil {
		return PrinterTestResult{OK: false, Error: "Invalid printer IP"}
	}

	dialer := net.Dialer{Timeout: 3 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(ip, "80"))
	if err != nil {
		log.Printf("printer test: tcp %s:80 failed: %v", ip, err)
		return PrinterTestResult{OK: false, Error: "No response on port 80 — check Wi‑Fi and IP"}
	}
	_ = conn.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	paths := []string{
		"http://" + ip + "/eSCL/ScannerStatus",
		"http://" + ip + "/Scan/Status",
		"http://" + ip + ":8080/Scan/Status",
	}
	var lastStatus int
	for _, u := range paths {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			continue
		}
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		lastStatus = resp.StatusCode
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
		// Any HTTP response from a scan endpoint means the device is reachable.
		if resp.StatusCode > 0 && resp.StatusCode < 600 {
			if resp.StatusCode >= 200 && resp.StatusCode < 500 {
				return PrinterTestResult{OK: true}
			}
		}
	}
	if lastStatus > 0 {
		return PrinterTestResult{OK: true}
	}
	// Port 80 was open but no known scan URL answered — still treat as reachable host.
	return PrinterTestResult{OK: true}
}

// ollamaModelPresent matches "name" or "name:tag" against /api/tags names.
func ollamaModelPresent(names []string, want string) bool {
	want = strings.TrimSpace(want)
	if want == "" {
		return true
	}
	wantBase, _, _ := strings.Cut(want, ":")
	for _, n := range names {
		if n == want {
			return true
		}
		base, _, _ := strings.Cut(n, ":")
		if base == want || base == wantBase || n == wantBase {
			return true
		}
	}
	return false
}

func (s *SettingsService) getRaw(ctx context.Context, key string) (string, error) {
	v, err := s.repo.Get(ctx, key)
	if errors.Is(err, repository.ErrNotFound) {
		return "", nil
	}
	return v, err
}

// ValidateOllamaURL allows http(s) Ollama endpoints on LAN/localhost, and
// rejects credentials, non-HTTP schemes, and cloud-metadata / link-local targets.
func ValidateOllamaURL(in string) error {
	trimmed := strings.TrimSpace(in)
	if i := strings.Index(trimmed, "://"); i >= 0 {
		scheme := strings.ToLower(trimmed[:i])
		if scheme != "http" && scheme != "https" {
			return fmt.Errorf("%w: only http and https are allowed", ErrInvalidOllamaURL)
		}
	}
	normalized := NormalizeOllamaURL(trimmed)
	if normalized == "" {
		return fmt.Errorf("%w: URL is empty", ErrInvalidOllamaURL)
	}
	u, err := url.Parse(normalized)
	if err != nil || u.Host == "" {
		return fmt.Errorf("%w: use http://host:11434", ErrInvalidOllamaURL)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%w: only http and https are allowed", ErrInvalidOllamaURL)
	}
	if u.User != nil {
		return fmt.Errorf("%w: do not put credentials in the URL", ErrInvalidOllamaURL)
	}
	host := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	if host == "" {
		return fmt.Errorf("%w: missing host", ErrInvalidOllamaURL)
	}
	if host == "metadata.google.internal" || strings.HasSuffix(host, ".metadata.google.internal") {
		return fmt.Errorf("%w: that host is not allowed", ErrInvalidOllamaURL)
	}
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			return fmt.Errorf("%w: that address is not allowed", ErrInvalidOllamaURL)
		}
	}
	return nil
}

// NormalizeOllamaURL adds http:// if no scheme and :11434 if no port.
func NormalizeOllamaURL(in string) string {
	in = strings.TrimSpace(in)
	if in == "" {
		return ""
	}
	if !strings.HasPrefix(in, "http://") && !strings.HasPrefix(in, "https://") {
		in = "http://" + in
	}
	if strings.HasPrefix(in, "http://") {
		rest := in[7:]
		if rest == "" {
			return in
		}
		if !strings.Contains(rest, ":") {
			return in + ":" + defaultOllamaPort
		}
		lastColon := strings.LastIndex(rest, ":")
		portPart := rest[lastColon+1:]
		if portPart == "" || !isDigits(portPart) {
			return in + ":" + defaultOllamaPort
		}
	}
	if strings.HasPrefix(in, "https://") {
		rest := in[8:]
		if rest == "" {
			return in
		}
		if !strings.Contains(rest, ":") {
			return in + ":" + defaultOllamaPort
		}
		lastColon := strings.LastIndex(rest, ":")
		portPart := rest[lastColon+1:]
		if portPart == "" || !isDigits(portPart) {
			return in + ":" + defaultOllamaPort
		}
	}
	return in
}

func isDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}
