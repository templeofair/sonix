package server

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// SPAHandler serves files from fsys and falls back to index.html for paths that are not files.
// This avoids http.FileServer's built-in redirect of "/index.html" → "/" which
// causes infinite 301 loops for client-side routes like /documents/2.
func SPAHandler(fsys fs.FS) http.Handler {
	root, _ := fs.Sub(fsys, ".")
	fileServer := http.FileServer(http.FS(root))

	// Pre-read index.html once at startup for the SPA fallback.
	indexHTML, _ := fs.ReadFile(root, "index.html")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqPath := strings.TrimPrefix(r.URL.Path, "/")
		if reqPath == "" {
			reqPath = "index.html"
		}
		f, err := root.Open(reqPath)
		if err == nil {
			defer f.Close()
			if stat, _ := f.Stat(); stat != nil && !stat.IsDir() {
				setStaticContentType(w, reqPath)
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		// Fallback: serve index.html directly for SPA routing.
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(indexHTML)
	})
}

// setStaticContentType ensures MIME types Go's mime package may miss (notably .webmanifest).
func setStaticContentType(w http.ResponseWriter, reqPath string) {
	switch strings.ToLower(path.Ext(reqPath)) {
	case ".webmanifest":
		w.Header().Set("Content-Type", "application/manifest+json")
	}
}
