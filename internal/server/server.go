package server

import (
	"database/sql"
	"io/fs"
	"net/http"
	"time"

	"github.com/templeofair/sonix/internal/auth"
	"github.com/templeofair/sonix/internal/config"
	"github.com/templeofair/sonix/internal/handler"
	"github.com/templeofair/sonix/internal/ocr"
	"github.com/templeofair/sonix/internal/repository"
	"github.com/templeofair/sonix/internal/service"
)

// Options holds dependencies for the server.
type Options struct {
	Config      *config.Config
	DB          *sql.DB
	UploadsPath string
	ThumbsPath  string
	StaticFS    fs.FS // optional: embed of frontend build for SPA
	// OCRProvider used when extraction requests OCR (use_ocr). If nil, New uses Tesseract.
	OCRProvider ocr.Provider
	// Auth is optional; if nil, New builds AuthService from DB.
	Auth *service.AuthService
	// Documents is optional; if nil, New builds DocumentService from DB.
	Documents *service.DocumentService
	// Extraction is optional; if nil, New builds ExtractionService from DB.
	Extraction *service.ExtractionService
	// Settings is optional; if nil, New builds SettingsService from DB.
	Settings *service.SettingsService
	// Export is optional; if nil, New builds ExportService from Documents.
	Export *service.ExportService
}

// Server is the HTTP server.
type Server struct {
	opts       Options
	mux        *http.ServeMux
	auth       *handler.Auth
	documents  *service.DocumentService
	extraction *service.ExtractionService
	settings   *service.SettingsService
	export     *service.ExportService
	printerLim *printerLimiter
}

// New builds a new Server.
func New(opts Options) *Server {
	if opts.OCRProvider == nil {
		opts.OCRProvider = ocr.NewTesseract()
	}
	authSvc := opts.Auth
	if authSvc == nil && opts.DB != nil {
		authSvc = service.NewAuthService(
			repository.NewSQLiteUserRepository(opts.DB),
			repository.NewSQLiteSessionRepository(opts.DB),
		)
	}

	docs := opts.Documents
	if docs == nil && opts.DB != nil {
		docs = service.NewDocumentService(repository.NewSQLiteDocumentRepository(opts.DB), opts.UploadsPath, opts.ThumbsPath)
	}
	if docs != nil && opts.Config != nil {
		docs.SetLimits(opts.Config.DocumentMaxPages, time.Duration(opts.Config.PDFConvertTimeoutSec)*time.Second)
	}

	settings := opts.Settings
	if settings == nil && opts.DB != nil {
		settings = service.NewSettingsService(
			repository.NewSQLiteSettingsRepository(opts.DB),
			opts.Config,
		)
	}

	extraction := opts.Extraction
	if extraction == nil && opts.DB != nil {
		extraction = service.NewExtractionService(
			repository.NewSQLiteExtractionRepository(opts.DB),
			settings,
			opts.OCRProvider,
			opts.Config,
			opts.UploadsPath,
		)
	}

	exportSvc := opts.Export
	if exportSvc == nil && docs != nil {
		exportSvc = service.NewExportService(docs, opts.UploadsPath)
	}

	s := &Server{
		opts:       opts,
		mux:        http.NewServeMux(),
		auth:       handler.NewAuth(authSvc, mustCookieMAC(opts.Config)),
		documents:  docs,
		extraction: extraction,
		settings:   settings,
		export:     exportSvc,
		printerLim: newPrinterLimiter(),
	}
	s.routes()
	return s
}

func mustCookieMAC(cfg *config.Config) *auth.CookieMAC {
	secret := ""
	if cfg != nil {
		secret = cfg.SessionSecret
	}
	mac, err := auth.NewCookieMAC(secret)
	if err != nil {
		panic("SESSION_SECRET: " + err.Error())
	}
	return mac
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	s.mux.Handle("/api/", http.StripPrefix("/api", s.apiHandler()))
	if s.opts.StaticFS != nil {
		s.mux.Handle("/", SPAHandler(s.opts.StaticFS))
	}
}

func (s *Server) apiHandler() http.Handler {
	mux := http.NewServeMux()
	auth := s.auth
	mux.HandleFunc("POST /login", auth.Login)
	mux.HandleFunc("POST /logout", auth.Logout)
	mux.HandleFunc("GET /me", auth.RequireAuth(auth.Me))
	mux.HandleFunc("GET /settings", auth.RequireAuth(s.handleGetSettings))
	mux.HandleFunc("PUT /settings", auth.RequireAuth(s.handlePutSettings))
	mux.HandleFunc("GET /export", auth.RequireAuth(s.handleExport))
	mux.HandleFunc("GET /documents/years", auth.RequireAuth(s.handleGetDocumentsYears))
	mux.HandleFunc("GET /documents/tags", auth.RequireAuth(s.handleGetDocumentsTags))
	mux.HandleFunc("GET /documents/document-date-years", auth.RequireAuth(s.handleGetDocumentDateYears))
	mux.HandleFunc("GET /documents/{id}/pages/{pageIndex}/image", auth.RequireAuth(s.handleGetPageImage))
	mux.HandleFunc("GET /documents/{id}/pages/{pageIndex}/thumbnail", auth.RequireAuth(s.handleGetPageThumbnail))
	mux.HandleFunc("POST /documents/{id}/pages/{pageIndex}/rotate", auth.RequireAuth(s.handleRotatePage))
	mux.HandleFunc("GET /documents/{id}/text", auth.RequireAuth(s.handleGetDocumentText))
	mux.HandleFunc("POST /documents/{id}/extract", auth.RequireAuth(s.handleExtract))
	mux.HandleFunc("POST /documents/{id}/reset-extraction", auth.RequireAuth(s.handleResetExtraction))
	mux.HandleFunc("GET /documents/{id}/status", auth.RequireAuth(s.handleExtractStatus))
	mux.HandleFunc("PUT /documents/{id}/title", auth.RequireAuth(s.handlePutDocumentTitle))
	mux.HandleFunc("PUT /documents/{id}/tags", auth.RequireAuth(s.handlePutDocumentTags))
	mux.HandleFunc("PUT /documents/{id}/document_date", auth.RequireAuth(s.handlePutDocumentDate))
	mux.HandleFunc("POST /documents/{id}/pages", auth.RequireAuth(s.handleUploadPages))
	mux.HandleFunc("GET /documents/{id}", auth.RequireAuth(s.handleGetDocument))
	mux.HandleFunc("DELETE /documents/{id}", auth.RequireAuth(s.handleDeleteDocument))
	mux.HandleFunc("GET /documents", auth.RequireAuth(s.handleListDocuments))
	mux.HandleFunc("POST /documents", auth.RequireAuth(s.handleCreateDocument))
	mux.HandleFunc("POST /settings/ollama/test", auth.RequireAuth(s.handleOllamaTest))
	mux.HandleFunc("POST /settings/printer/test", auth.RequireAuth(s.handlePrinterTest))
	return mux
}

// Handler returns the HTTP handler (for use with custom listeners).
func (s *Server) Handler() http.Handler {
	return securityHeaders(s.mux)
}

// ListenAndServe listens on the configured address.
func (s *Server) ListenAndServe() error {
	return http.ListenAndServe(s.opts.Config.ServerAddr, s.Handler())
}
