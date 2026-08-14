package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/templeofair/sonix/internal/repository"
)

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	filter := repository.DocumentListFilter{
		Q:                strings.TrimSpace(r.URL.Query().Get("q")),
		Tag:              strings.TrimSpace(r.URL.Query().Get("tag")),
		Year:             strings.TrimSpace(r.URL.Query().Get("year")),
		DocumentDateFrom: r.URL.Query().Get("document_date_from"),
		DocumentDateTo:   r.URL.Query().Get("document_date_to"),
		CreatedFrom:      r.URL.Query().Get("created_from"),
		CreatedTo:        r.URL.Query().Get("created_to"),
		Status:           strings.TrimSpace(r.URL.Query().Get("status")),
	}
	docIDs, err := s.export.ListMatching(r.Context(), filter)
	if err != nil {
		writeSearchOrServerError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=\"documents-"+time.Now().Format("2006-01-02")+".zip\"")
	_ = s.export.WriteZip(r.Context(), w, docIDs)
}
