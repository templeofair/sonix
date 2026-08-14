package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetDocumentDateYearsAndUndatedList(t *testing.T) {
	srv, db := setupDocServer(t)
	h := srv.Handler()
	cookie := loginCookie(t, h)

	if _, err := db.Exec(`INSERT INTO documents (id, title, status, created_at, updated_at) VALUES
		 (1, 'Dated2024', 'ready', '2025-01-01T00:00:00Z', '2025-01-01T00:00:00Z'),
		 (2, 'Dated2023', 'ready', '2025-01-02T00:00:00Z', '2025-01-02T00:00:00Z'),
		 (3, 'EmptyDate', 'ready', '2025-01-03T00:00:00Z', '2025-01-03T00:00:00Z'),
		 (4, 'NoRow', 'pending', '2025-01-04T00:00:00Z', '2025-01-04T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO extractions (document_id, tags, document_date) VALUES
		 (1, '[]', '2024-05-01'),
		 (2, '[]', '2023-05-01'),
		 (3, '[]', '')`); err != nil {
		t.Fatal(err)
	}

	get := func(url string, withCookie bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, url, nil)
		if withCookie {
			req.Header.Set("Cookie", cookie)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	t.Run("requires auth", func(t *testing.T) {
		if rec := get("/api/documents/document-date-years", false); rec.Code != http.StatusUnauthorized {
			t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("year buckets and undated count", func(t *testing.T) {
		rec := get("/api/documents/document-date-years", true)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
		}
		var resp struct {
			Years []struct {
				Year  string `json:"year"`
				Count int64  `json:"count"`
			} `json:"years"`
			UndatedCount int64 `json:"undated_count"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}
		if len(resp.Years) != 2 {
			t.Fatalf("years = %#v", resp.Years)
		}
		if resp.Years[0].Year != "2024" || resp.Years[0].Count != 1 {
			t.Fatalf("first = %#v", resp.Years[0])
		}
		if resp.Years[1].Year != "2023" || resp.Years[1].Count != 1 {
			t.Fatalf("second = %#v", resp.Years[1])
		}
		if resp.UndatedCount != 2 {
			t.Fatalf("undated_count = %d", resp.UndatedCount)
		}
	})

	t.Run("existing years endpoint untouched", func(t *testing.T) {
		rec := get("/api/documents/years", true)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
		}
		var resp struct {
			Years []string `json:"years"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}
		// Still grouped on the import year, which is 2025 for every seeded document.
		if len(resp.Years) != 1 || resp.Years[0] != "2025" {
			t.Fatalf("years = %#v", resp.Years)
		}
	})

	listTotal := func(t *testing.T, url string) (int, float64) {
		t.Helper()
		rec := get(url, true)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
		}
		var resp struct {
			Documents []map[string]any `json:"documents"`
			Total     float64          `json:"total"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}
		return len(resp.Documents), resp.Total
	}

	t.Run("undated param filters the list", func(t *testing.T) {
		for _, v := range []string{"1", "true", "TRUE"} {
			n, total := listTotal(t, "/api/documents?undated="+v)
			if n != 2 || total != 2 {
				t.Fatalf("undated=%s → %d docs, total %v", v, n, total)
			}
		}
	})

	t.Run("falsy or absent param lists everything", func(t *testing.T) {
		for _, url := range []string{"/api/documents", "/api/documents?undated=", "/api/documents?undated=0", "/api/documents?undated=false"} {
			n, total := listTotal(t, url)
			if n != 4 || total != 4 {
				t.Fatalf("%s → %d docs, total %v", url, n, total)
			}
		}
	})
}
