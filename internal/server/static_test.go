package server

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestSPAHandlerServesWebManifest(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html": &fstest.MapFile{
			Data: []byte("<!doctype html><title>Sonix</title>"),
		},
		"manifest.webmanifest": &fstest.MapFile{
			Data: []byte(`{"name":"Sonix","display":"standalone"}`),
		},
		"icons/icon-192.png": &fstest.MapFile{
			Data: []byte("fake-png"),
		},
	}

	h := SPAHandler(fsys)

	req := httptest.NewRequest(http.MethodGet, "/manifest.webmanifest", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "application/manifest+json") {
		t.Fatalf("Content-Type = %q, want application/manifest+json", ct)
	}
	if body := rec.Body.String(); !strings.Contains(body, `"name":"Sonix"`) {
		t.Fatalf("body = %q, want manifest JSON", body)
	}
}

func TestSPAHandlerFallsBackToIndex(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html": &fstest.MapFile{
			Data: []byte("<!doctype html><title>fallback</title>"),
		},
	}
	h := SPAHandler(fsys)

	req := httptest.NewRequest(http.MethodGet, "/documents/2", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("Content-Type = %q, want text/html", ct)
	}
	if !strings.Contains(rec.Body.String(), "fallback") {
		t.Fatalf("body missing index fallback: %q", rec.Body.String())
	}
}

func TestSPAHandlerServesExistingAsset(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("index")},
		"icons/icon-192.png": &fstest.MapFile{
			Data: []byte("png-bytes"),
		},
	}
	h := SPAHandler(fs.FS(fsys))

	req := httptest.NewRequest(http.MethodGet, "/icons/icon-192.png", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if rec.Body.String() != "png-bytes" {
		t.Fatalf("body = %q, want png-bytes", rec.Body.String())
	}
}
