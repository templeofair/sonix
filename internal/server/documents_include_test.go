package server_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/templeofair/sonix/internal/auth"
	"github.com/templeofair/sonix/internal/config"
	"github.com/templeofair/sonix/internal/database"
	"github.com/templeofair/sonix/internal/repository"
	"github.com/templeofair/sonix/internal/server"
	"github.com/templeofair/sonix/internal/service"
)

func setupDocServer(t *testing.T) (*server.Server, *sql.DB) {
	t.Helper()
	dir := t.TempDir()
	db, err := database.Open(filepath.Join(dir, "s.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	uploads := filepath.Join(dir, "uploads")
	thumbs := filepath.Join(dir, "thumbs")
	testUploadsPath = uploads
	t.Cleanup(func() { testUploadsPath = "" })
	cfg := &config.Config{SessionSecret: "test-secret-at-least-16", ServerAddr: ":0", DataDir: dir}
	docs := service.NewDocumentService(repository.NewSQLiteDocumentRepository(db), uploads, thumbs)
	auth := service.NewAuthService(
		repository.NewSQLiteUserRepository(db),
		repository.NewSQLiteSessionRepository(db),
	)
	if err := auth.SeedUserIfEmpty(context.Background(), "admin", "test-password-1"); err != nil {
		t.Fatal(err)
	}
	srv := server.New(server.Options{
		Config:      cfg,
		DB:          db,
		UploadsPath: uploads,
		ThumbsPath:  thumbs,
		Documents:   docs,
		Auth:        auth,
		Settings:    service.NewSettingsService(repository.NewSQLiteSettingsRepository(db), cfg),
	})
	return srv, db
}

func loginCookie(t *testing.T, h http.Handler) string {
	t.Helper()
	body := strings.NewReader(`{"username":"admin","password":"test-password-1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/login", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login status %d body %s", rec.Code, rec.Body.String())
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == auth.CookieName {
			return c.Name + "=" + c.Value
		}
	}
	t.Fatal("no session cookie")
	return ""
}

func TestGetDocument_IncludeTextOptIn(t *testing.T) {
	srv, db := setupDocServer(t)
	h := srv.Handler()
	cookie := loginCookie(t, h)

	_, err := db.Exec(`INSERT INTO documents (id, title, status) VALUES (9, 'Doc', 'ready')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO extractions (document_id, tags, summary, full_text_original, full_text_english)
		VALUES (9, '[]', 'sum', 'ORIGINAL TEXT', 'ENGLISH TEXT')`)
	if err != nil {
		t.Fatal(err)
	}

	get := func(include string) map[string]any {
		url := "/api/documents/9"
		if include != "" {
			url += "?include=" + include
		}
		req := httptest.NewRequest(http.MethodGet, url, nil)
		req.Header.Set("Cookie", cookie)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
		}
		var m map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
			t.Fatal(err)
		}
		return m
	}

	defaultResp := get("")
	ext, _ := defaultResp["extraction"].(map[string]any)
	if ext == nil {
		t.Fatal("missing extraction")
	}
	if _, ok := ext["full_text_original"]; ok {
		t.Fatal("default response must omit full_text_original")
	}
	if _, ok := ext["full_text_english"]; ok {
		t.Fatal("default response must omit full_text_english")
	}
	if defaultResp["page_count"] == nil {
		t.Fatal("page_count should be present")
	}

	withText := get("text")
	ext2, _ := withText["extraction"].(map[string]any)
	if ext2["full_text_original"] != "ORIGINAL TEXT" {
		t.Fatalf("full_text_original = %#v", ext2["full_text_original"])
	}
	if ext2["full_text_english"] != "ENGLISH TEXT" {
		t.Fatalf("full_text_english = %#v", ext2["full_text_english"])
	}
}
