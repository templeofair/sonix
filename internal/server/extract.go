package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/templeofair/sonix/internal/service"
)

func (s *Server) handleExtract(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	docIDStr := r.PathValue("id")
	if docIDStr == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	docID, err := strconv.ParseInt(docIDStr, 10, 64)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	// Default path is LLM vision; OCR is opt-in via {"use_ocr": true}.
	// Older frontends may still send {"ignore_ocr": true} — treat that as a
	// request for the *default* path (i.e. LLM vision) and keep accepting it
	// so queued requests from a not-yet-reloaded tab don't regress to OCR.
	var body struct {
		UseOCR    bool `json:"use_ocr"`
		IgnoreOCR bool `json:"ignore_ocr"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}
	useOCR := body.UseOCR
	if body.IgnoreOCR {
		useOCR = false
	}

	if err := s.extraction.Start(r.Context(), docID, useOCR); err != nil {
		if errors.Is(err, service.ErrExtractionBusy) {
			http.Error(w, "extraction already running", http.StatusTooManyRequests)
			return
		}
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, map[string]string{"status": "processing"})
}

func (s *Server) handleExtractStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	docIDStr := r.PathValue("id")
	if docIDStr == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	docID, err := strconv.ParseInt(docIDStr, 10, 64)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	status, err := s.extraction.Status(r.Context(), docID)
	if err != nil {
		if errors.Is(err, service.ErrExtractionNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, map[string]string{"status": status})
}

func (s *Server) handleResetExtraction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	docIDStr := r.PathValue("id")
	if docIDStr == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	docID, err := strconv.ParseInt(docIDStr, 10, 64)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err := s.extraction.Reset(r.Context(), docID); err != nil {
		if errors.Is(err, service.ErrExtractionNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, map[string]string{"status": "pending"})
}
