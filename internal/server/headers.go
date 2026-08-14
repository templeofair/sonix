package server

import (
	"net/http"
	"strings"
)

// securityCSP is a same-origin policy that still allows camera/blob previews.
const securityCSP = "default-src 'self'; img-src 'self' blob: data:; media-src 'self' blob:; style-src 'self' 'unsafe-inline'; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'; object-src 'none'"

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Content-Security-Policy", securityCSP)
		next.ServeHTTP(w, r)
	})
}

func writeSearchOrServerError(w http.ResponseWriter, err error) {
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "fts5") {
		http.Error(w, "invalid search query", http.StatusBadRequest)
		return
	}
	http.Error(w, "server error", http.StatusInternalServerError)
}
