package server_test

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// testUploadsPath is set by setupDocServer for path assertions.
var testUploadsPath string

// tinyPNG is a valid 1×1 PNG used for upload tests.
func tinyPNG(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func multipartBody(t *testing.T, fieldName, filename, contentType string, data []byte) (*bytes.Buffer, string) {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreatePart(map[string][]string{
		"Content-Disposition": {`form-data; name="` + fieldName + `"; filename="` + filename + `"`},
		"Content-Type":        {contentType},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return &body, w.FormDataContentType()
}

func TestUploadPages_Image(t *testing.T) {
	srv, db := setupDocServer(t)
	h := srv.Handler()
	cookie := loginCookie(t, h)

	_, err := db.Exec(`INSERT INTO documents (id, title, status) VALUES (7, 'Img', 'pending')`)
	if err != nil {
		t.Fatal(err)
	}

	pngBytes := tinyPNG(t)
	body, ct := multipartBody(t, "files", "page.png", "image/png", pngBytes)
	req := httptest.NewRequest(http.MethodPost, "/api/documents/7/pages", body)
	req.Header.Set("Content-Type", ct)
	req.Header.Set("Cookie", cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["ok"] != true || resp["document_id"].(float64) != 7 {
		t.Fatalf("resp %#v", resp)
	}

	var n int
	var storagePath, contentType string
	err = db.QueryRow(`SELECT COUNT(*), storage_path, content_type FROM document_pages WHERE document_id = 7`).
		Scan(&n, &storagePath, &contentType)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 || contentType != "image/png" {
		t.Fatalf("pages n=%d ct=%s path=%s", n, contentType, storagePath)
	}
	full := filepath.Join(testUploadsPath, storagePath)
	if _, err := os.Stat(full); err != nil {
		t.Fatalf("stored file missing: %v", err)
	}
}

func TestUploadPages_PDF(t *testing.T) {
	if _, err := exec.LookPath("pdftoppm"); err != nil {
		t.Skip("pdftoppm not installed")
	}
	srv, db := setupDocServer(t)
	h := srv.Handler()
	cookie := loginCookie(t, h)

	_, err := db.Exec(`INSERT INTO documents (id, title, status) VALUES (8, 'Pdf', 'pending')`)
	if err != nil {
		t.Fatal(err)
	}

	pdf := []byte(`%PDF-1.1
1 0 obj<< /Type /Catalog /Pages 2 0 R >>endobj
2 0 obj<< /Type /Pages /Kids [3 0 R] /Count 1 >>endobj
3 0 obj<< /Type /Page /Parent 2 0 R /MediaBox [0 0 200 200] /Contents 4 0 R /Resources<< /Font<< /F1 5 0 R >> >> >>endobj
4 0 obj<< /Length 44 >>stream
BT /F1 24 Tf 50 100 Td (Hi) Tj ET
endstream
endobj
5 0 obj<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>endobj
xref
0 6
0000000000 65535 f 
0000000009 00000 n 
0000000058 00000 n 
0000000115 00000 n 
0000000266 00000 n 
0000000360 00000 n 
trailer<< /Size 6 /Root 1 0 R >>
startxref
441
%%EOF
`)

	body, ct := multipartBody(t, "files", "doc.pdf", "application/pdf", pdf)
	req := httptest.NewRequest(http.MethodPost, "/api/documents/8/pages", body)
	req.Header.Set("Content-Type", ct)
	req.Header.Set("Cookie", cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}

	var n int
	err = db.QueryRow(`SELECT COUNT(*) FROM document_pages WHERE document_id = 8 AND content_type = 'image/png'`).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	if n < 1 {
		t.Fatalf("expected >=1 PNG page from PDF, got %d", n)
	}
}

func TestUploadPages_UnsupportedType(t *testing.T) {
	srv, db := setupDocServer(t)
	h := srv.Handler()
	cookie := loginCookie(t, h)
	_, err := db.Exec(`INSERT INTO documents (id, title, status) VALUES (9, 'X', 'pending')`)
	if err != nil {
		t.Fatal(err)
	}
	body, ct := multipartBody(t, "files", "x.txt", "text/plain", []byte("nope"))
	req := httptest.NewRequest(http.MethodPost, "/api/documents/9/pages", body)
	req.Header.Set("Content-Type", ct)
	req.Header.Set("Cookie", cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestUploadPages_NotFound(t *testing.T) {
	srv, _ := setupDocServer(t)
	h := srv.Handler()
	cookie := loginCookie(t, h)
	body, ct := multipartBody(t, "files", "page.png", "image/png", tinyPNG(t))
	req := httptest.NewRequest(http.MethodPost, "/api/documents/404/pages", body)
	req.Header.Set("Content-Type", ct)
	req.Header.Set("Cookie", cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d", rec.Code)
	}
}
