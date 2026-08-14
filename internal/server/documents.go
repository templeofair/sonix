package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/templeofair/sonix/internal/repository"
	"github.com/templeofair/sonix/internal/service"
)

type documentListItem struct {
	ID                 int64     `json:"id"`
	Title              *string   `json:"title,omitempty"`
	Status             string    `json:"status"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	DocumentDate       *string   `json:"document_date,omitempty"`
	PageCount          int       `json:"page_count"`
	ThumbnailAvailable bool      `json:"thumbnail_available"`
}

type documentDetail struct {
	ID                   int64           `json:"id"`
	Title                *string         `json:"title,omitempty"`
	Status               string          `json:"status"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
	ExtractionError      *string         `json:"extraction_error,omitempty"`
	ExtractionPagesDone  *int            `json:"extraction_pages_done,omitempty"`
	ExtractionPagesTotal *int            `json:"extraction_pages_total,omitempty"`
	Pages                []pageInfo      `json:"pages"`
	Extraction           *extractionInfo `json:"extraction,omitempty"`
	PageCount            int             `json:"page_count"`
	ThumbnailAvailable   bool            `json:"thumbnail_available"`
}

type pageInfo struct {
	PageIndex   int    `json:"page_index"`
	ContentType string `json:"content_type"`
}

type extractionInfo struct {
	Tags             []string  `json:"tags"`
	Summary          string    `json:"summary"`
	DocumentDate     *string   `json:"document_date,omitempty"`
	ExtractedAt      time.Time `json:"extracted_at"`
	EngineID         string    `json:"engine_id,omitempty"`
	PromptVersion    string    `json:"prompt_version,omitempty"`
	ExtractionWallMs *int64    `json:"extraction_wall_ms,omitempty"`
	FullTextOriginal *string   `json:"full_text_original,omitempty"`
	FullTextEnglish  *string   `json:"full_text_english,omitempty"`
}

func (s *Server) handleCreateDocument(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Title string `json:"title"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	id, err := s.documents.Create(r.Context(), body.Title)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]int64{"id": id})
}

func (s *Server) handlePutDocumentTitle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	idStr := r.PathValue("id")
	if idStr == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	docID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	var body struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	trimmed, err := s.documents.UpdateTitle(r.Context(), docID, body.Title)
	if err != nil {
		if errors.Is(err, service.ErrDocumentNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"title": trimmed})
}

func (s *Server) handleListDocuments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 0 {
		page = 0
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 500 {
		limit = 50
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
		Sort:             strings.TrimSpace(r.URL.Query().Get("sort")),
		Undated:          undatedParam(r.URL.Query().Get("undated")),
		Limit:            limit,
		Offset:           page * limit,
	}
	result, err := s.documents.List(r.Context(), filter)
	if err != nil {
		writeSearchOrServerError(w, err)
		return
	}
	list := make([]documentListItem, 0, len(result.Documents))
	for _, row := range result.Documents {
		list = append(list, documentListItem{
			ID:                 row.ID,
			Title:              row.Title,
			Status:             row.Status,
			CreatedAt:          row.CreatedAt,
			UpdatedAt:          row.UpdatedAt,
			DocumentDate:       row.DocumentDate,
			PageCount:          row.PageCount,
			ThumbnailAvailable: row.ThumbnailAvailable,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"documents": list, "total": result.Total})
}

// undatedParam reports whether ?undated asks for letters with no letter date.
// Anything else (absent, empty, "0", "false") keeps the historical list behaviour.
func undatedParam(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true":
		return true
	}
	return false
}

func (s *Server) handleGetDocumentsYears(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	years, err := s.documents.Years(r.Context())
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"years": years})
}

func (s *Server) handleGetDocumentsTags(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tags, err := s.documents.Tags(r.Context())
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"tags": tags})
}

type documentDateYear struct {
	Year  string `json:"year"`
	Count int64  `json:"count"`
}

type documentDateYearsResponse struct {
	Years        []documentDateYear `json:"years"`
	UndatedCount int64              `json:"undated_count"`
}

func (s *Server) handleGetDocumentDateYears(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	rows, undated, err := s.documents.DocumentDateYears(r.Context())
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	years := make([]documentDateYear, 0, len(rows))
	for _, row := range rows {
		years = append(years, documentDateYear{Year: row.Year, Count: row.Count})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(documentDateYearsResponse{Years: years, UndatedCount: undated})
}

func (s *Server) handleGetDocument(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	idStr := r.PathValue("id")
	if idStr == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	docID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	includeText := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("include")), "text")
	d, err := s.documents.Get(r.Context(), docID, service.GetOptions{IncludeText: includeText})
	if err != nil {
		if errors.Is(err, service.ErrDocumentNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	doc := documentDetail{
		ID:                   d.ID,
		Title:                d.Title,
		Status:               d.Status,
		CreatedAt:            d.CreatedAt,
		UpdatedAt:            d.UpdatedAt,
		ExtractionError:      d.ExtractionError,
		ExtractionPagesDone:  d.ExtractionPagesDone,
		ExtractionPagesTotal: d.ExtractionPagesTotal,
		Pages:                make([]pageInfo, 0, len(d.Pages)),
		PageCount:            d.PageCount,
		ThumbnailAvailable:   d.ThumbnailAvailable,
	}
	for _, p := range d.Pages {
		doc.Pages = append(doc.Pages, pageInfo{PageIndex: p.PageIndex, ContentType: p.ContentType})
	}
	if d.Extraction != nil {
		doc.Extraction = &extractionInfo{
			Tags:             d.Extraction.Tags,
			Summary:          d.Extraction.Summary,
			DocumentDate:     d.Extraction.DocumentDate,
			ExtractedAt:      d.Extraction.ExtractedAt,
			EngineID:         d.Extraction.EngineID,
			PromptVersion:    d.Extraction.PromptVersion,
			ExtractionWallMs: d.Extraction.ExtractionWallMs,
			FullTextOriginal: d.Extraction.FullTextOriginal,
			FullTextEnglish:  d.Extraction.FullTextEnglish,
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
}

func (s *Server) handlePutDocumentTags(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	idStr := r.PathValue("id")
	if idStr == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	docID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	var body struct {
		Tags []string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	tags, err := s.documents.UpdateTags(r.Context(), docID, body.Tags)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"tags": tags})
}

func (s *Server) handlePutDocumentDate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	idStr := r.PathValue("id")
	if idStr == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	docID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	var body struct {
		DocumentDate *string `json:"document_date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	docDate, err := s.documents.UpdateDocumentDate(r.Context(), docID, body.DocumentDate)
	if err != nil {
		if errors.Is(err, service.ErrNoExtraction) {
			http.Error(w, "no extraction for document", http.StatusNotFound)
			return
		}
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"document_date": docDate})
}

func (s *Server) handleDeleteDocument(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	idStr := r.PathValue("id")
	if idStr == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	docID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err := s.documents.Delete(r.Context(), docID); err != nil {
		if errors.Is(err, service.ErrDocumentNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
