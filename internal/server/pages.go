package server

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/templeofair/sonix/internal/service"
)

const maxUploadSize = 50 << 20 // 50 MB per request

func (s *Server) handleUploadPages(w http.ResponseWriter, r *http.Request) {
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

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		http.Error(w, "request too large or invalid multipart", http.StatusBadRequest)
		return
	}
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		files = r.MultipartForm.File["file"]
	}
	if len(files) == 0 {
		http.Error(w, "no files in request", http.StatusBadRequest)
		return
	}

	uploads := make([]service.PageUpload, 0, len(files))
	for _, fh := range files {
		fh := fh
		uploads = append(uploads, service.PageUpload{
			ContentType: fh.Header.Get("Content-Type"),
			Open: func() (io.ReadCloser, error) {
				return fh.Open()
			},
		})
	}

	err = s.documents.UploadPages(r.Context(), docID, uploads)
	if err != nil {
		if errors.Is(err, service.ErrDocumentNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, service.ErrUnsupportedContentType) {
			http.Error(w, service.ErrUnsupportedContentType.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, service.ErrPDFConversion) {
			log.Printf("upload: PDF conversion: %v", err)
			http.Error(w, service.ErrPDFConversion.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, service.ErrTooManyPages) {
			http.Error(w, service.ErrTooManyPages.Error(), http.StatusBadRequest)
			return
		}
		if err.Error() == "no files in request" {
			http.Error(w, "no files in request", http.StatusBadRequest)
			return
		}
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, map[string]any{"ok": true, "document_id": docID})
}

func (s *Server) handleGetPageImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	docIDStr := r.PathValue("id")
	pageIndexStr := r.PathValue("pageIndex")
	if docIDStr == "" || pageIndexStr == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	docID, err := strconv.ParseInt(docIDStr, 10, 64)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	pageIndex, err := strconv.Atoi(pageIndexStr)
	if err != nil || pageIndex < 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	storagePath, _, err := s.documents.GetPage(r.Context(), docID, pageIndex)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	fullPath := filepath.Join(s.opts.UploadsPath, storagePath)
	absUploads, _ := filepath.Abs(s.opts.UploadsPath)
	absFull, _ := filepath.Abs(fullPath)
	if !service.PathInside(absUploads, absFull) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	http.ServeFile(w, r, fullPath)
}

func (s *Server) handleGetPageThumbnail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	docIDStr := r.PathValue("id")
	pageIndexStr := r.PathValue("pageIndex")
	if docIDStr == "" || pageIndexStr == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	docID, err := strconv.ParseInt(docIDStr, 10, 64)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	pageIndex, err := strconv.Atoi(pageIndexStr)
	if err != nil || pageIndex < 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	thumb, err := s.documents.EnsureThumbnail(r.Context(), docID, pageIndex)
	if err != nil {
		if errors.Is(err, service.ErrDocumentNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	if thumb.FromCache {
		w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "private, max-age=60")
	}
	if thumb.ContentType != "" {
		w.Header().Set("Content-Type", thumb.ContentType)
	}
	http.ServeFile(w, r, thumb.Path)
}

func (s *Server) handleRotatePage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	docIDStr := r.PathValue("id")
	pageIndexStr := r.PathValue("pageIndex")
	if docIDStr == "" || pageIndexStr == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	docID, err := strconv.ParseInt(docIDStr, 10, 64)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	pageIndex, err := strconv.Atoi(pageIndexStr)
	if err != nil || pageIndex < 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	var body struct {
		Degrees int `json:"degrees"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	err = s.documents.RotatePage(r.Context(), docID, pageIndex, body.Degrees)
	if err != nil {
		if errors.Is(err, service.ErrDocumentNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, service.ErrInvalidRotateDegrees) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, map[string]any{"ok": true})
}

func (s *Server) handleGetDocumentText(w http.ResponseWriter, r *http.Request) {
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
	lang := r.URL.Query().Get("lang")
	text, err := s.documents.GetText(r.Context(), docID, lang)
	if err != nil {
		if errors.Is(err, service.ErrInvalidTextLang) {
			http.Error(w, "lang must be original or english", http.StatusBadRequest)
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(text))
}

func writeJSON(w http.ResponseWriter, v any) {
	json.NewEncoder(w).Encode(v)
}
