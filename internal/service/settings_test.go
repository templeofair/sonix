package service

import (
	"context"
	"testing"

	"github.com/templeofair/sonix/internal/config"
	"github.com/templeofair/sonix/internal/repository"
)

type memSettings struct {
	m map[string]string
}

func (m *memSettings) Get(_ context.Context, key string) (string, error) {
	v, ok := m.m[key]
	if !ok {
		return "", repository.ErrNotFound
	}
	return v, nil
}

func (m *memSettings) Set(_ context.Context, key, value string) error {
	m.m[key] = value
	return nil
}

func (m *memSettings) Delete(_ context.Context, key string) error {
	delete(m.m, key)
	return nil
}

func TestEffectiveModels_SplitAndFallback(t *testing.T) {
	cfg := &config.Config{
		OllamaBaseURL: "http://localhost:11434",
		OllamaVision:  "env-vision",
		OllamaText:    "env-text",
	}
	repo := &memSettings{m: map[string]string{}}
	s := NewSettingsService(repo, cfg)
	ctx := context.Background()

	if got := s.EffectiveVisionModel(ctx); got != "env-vision" {
		t.Fatalf("vision from env = %q", got)
	}
	if got := s.EffectiveTextModel(ctx); got != "env-text" {
		t.Fatalf("text from env = %q, want env-text when no settings", got)
	}

	_ = repo.Set(ctx, repository.SettingsKeyOllamaModel, "Keyvan/german-ocr-turbo")
	if got := s.EffectiveVisionModel(ctx); got != "Keyvan/german-ocr-turbo" {
		t.Fatalf("vision from settings = %q", got)
	}
	// Single-model Settings install: text reuses vision.
	if got := s.EffectiveTextModel(ctx); got != "Keyvan/german-ocr-turbo" {
		t.Fatalf("text fallback to vision = %q", got)
	}

	_ = repo.Set(ctx, repository.SettingsKeyOllamaTextModel, "Keyvan/german-text-3.1")
	if got := s.EffectiveTextModel(ctx); got != "Keyvan/german-text-3.1" {
		t.Fatalf("text from settings = %q", got)
	}
}

func TestValidateOllamaURL(t *testing.T) {
	ok := []string{
		"http://localhost:11434",
		"http://127.0.0.1:11434",
		"http://host.docker.internal:11434",
		"http://192.168.1.10:11434",
		"https://ollama.example.com:11434",
		"localhost:11434",
	}
	for _, u := range ok {
		if err := ValidateOllamaURL(u); err != nil {
			t.Errorf("%s: %v", u, err)
		}
	}
	bad := []string{
		"",
		"file:///etc/passwd",
		"http://user:pass@localhost:11434",
		"http://169.254.169.254/",
		"http://metadata.google.internal/",
		"ftp://localhost:11434",
	}
	for _, u := range bad {
		if err := ValidateOllamaURL(u); err == nil {
			t.Errorf("%s: expected error", u)
		}
	}
}

func TestTestPrinter_UsesSavedIPOnly(t *testing.T) {
	repo := &memSettings{m: map[string]string{}}
	s := NewSettingsService(repo, &config.Config{})
	ctx := context.Background()
	res := s.TestPrinter(ctx)
	if res.OK || res.Error != "Enter a printer IP first" {
		t.Fatalf("empty saved: %+v", res)
	}
	_ = repo.Set(ctx, repository.SettingsKeyHPPrinterIP, "not-an-ip")
	res = s.TestPrinter(ctx)
	if res.OK || res.Error != "Invalid printer IP" {
		t.Fatalf("invalid saved: %+v", res)
	}
}

func TestValidatePrinterIP(t *testing.T) {
	if err := validatePrinterIP(""); err != nil {
		t.Fatal(err)
	}
	if err := validatePrinterIP("192.168.1.50"); err != nil {
		t.Fatal(err)
	}
	if err := validatePrinterIP("not a ip"); err == nil {
		t.Fatal("expected error")
	}
	if err := validatePrinterIP("printer.local"); err == nil {
		t.Fatal("hostname should be rejected; IPv4 only")
	}
}

func TestOllamaModelPresent(t *testing.T) {
	names := []string{"Keyvan/german-ocr-turbo:latest", "Keyvan/german-text-3.1:latest"}
	if !ollamaModelPresent(names, "Keyvan/german-ocr-turbo") {
		t.Error("should match without :latest")
	}
	if !ollamaModelPresent(names, "Keyvan/german-ocr-turbo:latest") {
		t.Error("should match full name")
	}
	if ollamaModelPresent(names, "missing-model") {
		t.Error("missing model should be absent")
	}
}
