package service

import "strings"

const (
	msgExtractGeneric        = "Extraction failed. Check Ollama in Settings, then retry."
	msgExtractContext        = "This page is too large for the AI model’s context. Try OCR, or retry extraction."
	msgExtractUnreachable    = "Could not reach Ollama. Check the URL in Settings."
	msgExtractTimeout        = "Ollama timed out. Try again, or use a smaller model."
	msgExtractModelMissing   = "Ollama model not found. Check the model name in Settings."
	msgExtractNoPages        = "no pages"
	msgExtractURLNotAllowed  = "Ollama URL is not allowed"
	msgOllamaTestUnreachable = "Could not reach Ollama. Check the URL and that Ollama is running."
	msgImportFailed          = "Import failed"
	msgImportTooLarge        = "file too large (max 50 MB)"
	msgImportNotRegular      = "refusing non-regular file"
)

// PublicExtractionMessage maps a raw pipeline error to a stable client string.
// Paths, Ollama URLs, and model dumps stay in the server log only.
func PublicExtractionMessage(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return msgExtractGeneric
	}
	switch raw {
	case msgExtractGeneric, msgExtractContext, msgExtractUnreachable,
		msgExtractTimeout, msgExtractModelMissing, msgExtractNoPages, msgExtractURLNotAllowed:
		return raw
	}
	lower := strings.ToLower(raw)
	switch {
	case strings.Contains(lower, "exceed_context_size"),
		strings.Contains(lower, "exceeds the available context"),
		strings.Contains(lower, "n_prompt_tokens") && strings.Contains(lower, "n_ctx"):
		return msgExtractContext
	case strings.Contains(lower, "connection refused"),
		strings.Contains(lower, "no such host"),
		strings.Contains(lower, "network is unreachable"):
		return msgExtractUnreachable
	case strings.Contains(lower, "timeout"),
		strings.Contains(lower, "deadline exceeded"):
		return msgExtractTimeout
	case strings.Contains(lower, "404") && strings.Contains(lower, "model"):
		return msgExtractModelMissing
	case lower == msgExtractNoPages:
		return msgExtractNoPages
	case strings.Contains(lower, "ollama url is not allowed"):
		return msgExtractURLNotAllowed
	default:
		return msgExtractGeneric
	}
}

// PublicOllamaTestMessage is the client string when an Ollama probe cannot connect.
func PublicOllamaTestMessage() string {
	return msgOllamaTestUnreachable
}

// PublicImportMessage maps inbox-import errors to a stable client string.
func PublicImportMessage(raw string) string {
	raw = strings.TrimSpace(raw)
	switch raw {
	case "":
		return ""
	case msgImportTooLarge, msgImportNotRegular, msgImportFailed:
		return raw
	default:
		return msgImportFailed
	}
}
