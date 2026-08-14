package main

import (
	"context"
	"crypto/tls"
	"database/sql"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/templeofair/sonix/internal/auth"
	"github.com/templeofair/sonix/internal/config"
	"github.com/templeofair/sonix/internal/database"
	"github.com/templeofair/sonix/internal/ocr"
	"github.com/templeofair/sonix/internal/repository"
	"github.com/templeofair/sonix/internal/selfcert"
	"github.com/templeofair/sonix/internal/server"
	"github.com/templeofair/sonix/internal/service"
)

func seedUserIfNeeded(db *sql.DB, _ *config.Config) error {
	username := os.Getenv("SEED_USERNAME")
	password := os.Getenv("SEED_PASSWORD")
	if (username == "") != (password == "") {
		log.Print("seed: set both SEED_USERNAME and SEED_PASSWORD, or neither")
	}
	authSvc := service.NewAuthService(
		repository.NewSQLiteUserRepository(db),
		repository.NewSQLiteSessionRepository(db),
	)
	return authSvc.SeedUserIfEmpty(context.Background(), username, password)
}

func main() {
	cfg := config.Load()
	if err := auth.ValidateSessionSecret(cfg.SessionSecret); err != nil {
		log.Fatalf("SESSION_SECRET: %v", err)
	}
	if err := os.MkdirAll(cfg.DataDir, 0700); err != nil {
		log.Fatalf("create data dir: %v", err)
	}
	uploadsDir := filepath.Join(cfg.DataDir, "uploads")
	if err := os.MkdirAll(uploadsDir, 0700); err != nil {
		log.Fatalf("create uploads dir: %v", err)
	}
	thumbsDir := filepath.Join(cfg.DataDir, "thumbs")
	if err := os.MkdirAll(thumbsDir, 0700); err != nil {
		log.Fatalf("create thumbs dir: %v", err)
	}

	db, err := database.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	if err := seedUserIfNeeded(db, cfg); err != nil {
		log.Fatalf("seed user: %v", err)
	}

	ocrProvider, err := ocr.NewProviderFromConfig(cfg)
	if err != nil {
		log.Fatalf("ocr: %v", err)
	}

	settingsSvc := service.NewSettingsService(
		repository.NewSQLiteSettingsRepository(db),
		cfg,
	)
	docSvc := service.NewDocumentService(repository.NewSQLiteDocumentRepository(db), uploadsDir, thumbsDir)
	extractionSvc := service.NewExtractionService(
		repository.NewSQLiteExtractionRepository(db),
		settingsSvc,
		ocrProvider,
		cfg,
		uploadsDir,
	)
	if n, err := extractionSvc.ResetStuckExtractions(context.Background()); err != nil {
		log.Fatalf("reset stuck extractions: %v", err)
	} else if n > 0 {
		log.Printf("Reset %d document(s) stuck in processing (server restarted)", n)
	}

	inboxDir := cfg.ImportInboxDir
	if inboxDir == "" {
		inboxDir = filepath.Join(cfg.DataDir, "inbox")
	}
	importer := service.NewInboxImporter(inboxDir, docSvc, extractionSvc, settingsSvc)
	if err := importer.EnsureDirs(); err != nil {
		log.Fatalf("inbox dirs: %v", err)
	}
	go importer.Run(context.Background())
	if err := settingsSvc.SyncHPPrinterIPFile(context.Background()); err != nil {
		log.Printf("hp-scan: sync printer IP file: %v", err)
	}

	opts := server.Options{
		Config:      cfg,
		DB:          db,
		UploadsPath: uploadsDir,
		ThumbsPath:  thumbsDir,
		OCRProvider: ocrProvider,
		Documents:   docSvc,
		Extraction:  extractionSvc,
		Settings:    settingsSvc,
		Export:      service.NewExportService(docSvc, uploadsDir),
	}
	if sfs := staticFS(); sfs != nil {
		opts.StaticFS = sfs
	}
	srv := server.New(opts)

	// Start HTTPS listener with self-signed cert (for mobile camera access)
	tlsDir := filepath.Join(cfg.DataDir, "tls")
	cert, tlsErr := selfcert.EnsureCert(tlsDir)
	if tlsErr != nil {
		log.Printf("WARNING: could not generate TLS cert: %v (HTTPS disabled)", tlsErr)
	} else {
		httpsAddr := os.Getenv("HTTPS_ADDR")
		if httpsAddr == "" {
			// 9443 avoids clashes with other HTTPS services; pairs with HTTP :9080.
			httpsAddr = ":9443"
		}
		go func() {
			tlsSrv := &http.Server{
				Addr:    httpsAddr,
				Handler: srv.Handler(),
				TLSConfig: &tls.Config{
					Certificates: []tls.Certificate{cert},
				},
			}
			log.Printf("HTTPS listening on %s (self-signed)", httpsAddr)
			if err := tlsSrv.ListenAndServeTLS("", ""); err != nil {
				log.Printf("HTTPS server error: %v", err)
			}
		}()
	}

	log.Printf("HTTP listening on %s", cfg.ServerAddr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server: %v", err)
	}
}
