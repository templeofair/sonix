package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/templeofair/sonix/internal/service"
)

func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	view, err := s.settings.Get(r.Context())
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, map[string]any{
		"ollama_base_url":        view.OllamaBaseURL,
		"ollama_base_url_raw":    view.OllamaBaseURLRaw,
		"ollama_model":           view.OllamaModel,
		"ollama_model_raw":       view.OllamaModelRaw,
		"ollama_text_model":      view.OllamaTextModel,
		"ollama_text_model_raw":  view.OllamaTextModelRaw,
		"import_inbox_enabled":   view.ImportInboxEnabled,
		"import_auto_extract":    view.ImportAutoExtract,
		"import_extract_use_ocr": view.ImportExtractUseOCR,
		"import_last_file":       view.ImportLastFile,
		"import_last_error":      view.ImportLastError,
		"import_last_at":         view.ImportLastAt,
		"hp_printer_ip":          view.HPPrinterIP,
	})
}

func (s *Server) handlePutSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		OllamaBaseURL       string  `json:"ollama_base_url"`
		OllamaModel         string  `json:"ollama_model"`
		OllamaTextModel     string  `json:"ollama_text_model"`
		ImportInboxEnabled  *bool   `json:"import_inbox_enabled"`
		ImportAutoExtract   *bool   `json:"import_auto_extract"`
		ImportExtractUseOCR *bool   `json:"import_extract_use_ocr"`
		HPPrinterIP         *string `json:"hp_printer_ip"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	result, err := s.settings.Update(r.Context(), body.OllamaBaseURL, body.OllamaModel, body.OllamaTextModel, body.ImportInboxEnabled, body.ImportAutoExtract, body.ImportExtractUseOCR, body.HPPrinterIP)
	if err != nil {
		if errors.Is(err, service.ErrInvalidPrinterIP) || errors.Is(err, service.ErrInvalidOllamaURL) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, map[string]any{
		"ollama_base_url":        result.OllamaBaseURL,
		"ollama_model":           result.OllamaModel,
		"ollama_text_model":      result.OllamaTextModel,
		"import_inbox_enabled":   result.ImportInboxEnabled,
		"import_auto_extract":    result.ImportAutoExtract,
		"import_extract_use_ocr": result.ImportExtractUseOCR,
		"hp_printer_ip":          result.HPPrinterIP,
	})
}

func (s *Server) handleOllamaTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	result := s.settings.TestOllama(r.Context())
	w.Header().Set("Content-Type", "application/json")
	if !result.OK {
		writeJSON(w, map[string]any{"ok": false, "error": result.Error})
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

func (s *Server) handlePrinterTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.printerLim.allow(printerClientIP(r.RemoteAddr)) {
		http.Error(w, "too many printer tests", http.StatusTooManyRequests)
		return
	}
	result := s.settings.TestPrinter(r.Context())
	w.Header().Set("Content-Type", "application/json")
	if !result.OK {
		writeJSON(w, map[string]any{"ok": false, "error": result.Error})
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}
