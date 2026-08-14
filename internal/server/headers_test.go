package server_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeaders(t *testing.T) {
	srv, _ := setupDocServer(t)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	h := rec.Header()
	if h.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q", h.Get("X-Content-Type-Options"))
	}
	if h.Get("X-Frame-Options") != "DENY" {
		t.Fatalf("X-Frame-Options = %q", h.Get("X-Frame-Options"))
	}
	if h.Get("Referrer-Policy") != "no-referrer" {
		t.Fatalf("Referrer-Policy = %q", h.Get("Referrer-Policy"))
	}
	if h.Get("Content-Security-Policy") == "" {
		t.Fatal("missing CSP")
	}
}
