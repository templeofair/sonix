package service_test

import (
	"testing"

	"github.com/templeofair/sonix/internal/service"
)

func TestPublicExtractionMessage(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want string
	}{
		{"", "Extraction failed. Check Ollama in Settings, then retry."},
		{
			`LLM vision: ollama 400 Bad Request: {"error":{"message":"request (4452 tokens) exceeds the available context size (4096 tokens)","type":"exceed_context_size_error"}}`,
			"This page is too large for the AI model’s context. Try OCR, or retry extraction.",
		},
		{
			`Get "http://192.168.1.9:11434/api/chat": connection refused`,
			"Could not reach Ollama. Check the URL in Settings.",
		},
		{"context deadline exceeded", "Ollama timed out. Try again, or use a smaller model."},
		{"ollama 404: model 'llava' not found", "Ollama model not found. Check the model name in Settings."},
		{"no pages", "no pages"},
		{"Ollama URL is not allowed", "Ollama URL is not allowed"},
		{"save original text: open /app/data/uploads/12/page_0.png: no such file", "Extraction failed. Check Ollama in Settings, then retry."},
	}
	for _, tc := range cases {
		got := service.PublicExtractionMessage(tc.in)
		if got != tc.want {
			t.Errorf("PublicExtractionMessage(%q)\n got %q\nwant %q", tc.in, got, tc.want)
		}
		if got != service.PublicExtractionMessage(got) {
			t.Errorf("not idempotent: %q -> %q", got, service.PublicExtractionMessage(got))
		}
	}
}

func TestPublicImportMessage(t *testing.T) {
	t.Parallel()
	if got := service.PublicImportMessage("open /app/data/inbox/scan.pdf: permission denied"); got != "Import failed" {
		t.Fatalf("got %q", got)
	}
	if got := service.PublicImportMessage("file too large (max 50 MB)"); got != "file too large (max 50 MB)" {
		t.Fatalf("got %q", got)
	}
}
