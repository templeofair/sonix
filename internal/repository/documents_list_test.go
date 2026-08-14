package repository_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/templeofair/sonix/internal/database"
	"github.com/templeofair/sonix/internal/repository"
)

func TestDocumentRepository_ListAndYears(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "docs.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	_, err = db.ExecContext(context.Background(),
		`INSERT INTO documents (title, status, created_at, updated_at) VALUES
		 ('A', 'pending', '2024-06-01T00:00:00Z', '2024-06-01T00:00:00Z'),
		 ('B', 'failed', '2023-03-01T00:00:00Z', '2023-03-01T00:00:00Z'),
		 ('C', 'ready', '2024-08-01T00:00:00Z', '2024-08-01T00:00:00Z')`)
	if err != nil {
		t.Fatal(err)
	}

	repo := repository.NewSQLiteDocumentRepository(db)
	years, err := repo.Years(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(years) != 2 || years[0] != "2024" || years[1] != "2023" {
		t.Fatalf("years = %#v", years)
	}

	list, err := repo.List(context.Background(), repository.DocumentListFilter{
		Status: "pending,failed",
		Limit:  100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("list len = %d", len(list))
	}
	// Default order: created_at DESC → A (2024-06) before B (2023-03)
	if list[0].Title.String != "A" || list[1].Title.String != "B" {
		t.Fatalf("default order titles = %q, %q", list[0].Title.String, list[1].Title.String)
	}

	ids, err := repo.ListIDs(context.Background(), repository.DocumentListFilter{
		CreatedFrom: "2024-01-01",
		CreatedTo:   "2024-12-31T23:59:59Z",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 {
		t.Fatalf("ids len = %d", len(ids))
	}
}

func TestEscapeFTS_viaListEmpty(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "fts.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	repo := repository.NewSQLiteDocumentRepository(db)
	for _, q := range []string{`hello "world"`, `AND OR NOT ( * ^`, `!!!`} {
		list, err := repo.List(context.Background(), repository.DocumentListFilter{Q: q, Limit: 10})
		if err != nil {
			t.Fatalf("q=%q: %v", q, err)
		}
		if len(list) != 0 {
			t.Fatalf("q=%q want empty got %d", q, len(list))
		}
	}
}

func TestDocumentRepository_ListSortTotalPageCount(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "sort.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	ctx := context.Background()

	_, err = db.ExecContext(ctx,
		`INSERT INTO documents (id, title, status, created_at, updated_at) VALUES
		 (1, 'OldCreate', 'ready', '2023-01-01T00:00:00Z', '2023-01-01T00:00:00Z'),
		 (2, 'NewCreate', 'ready', '2025-01-01T00:00:00Z', '2025-01-01T00:00:00Z'),
		 (3, 'MidCreate', 'ready', '2024-01-01T00:00:00Z', '2024-01-01T00:00:00Z')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx,
		`INSERT INTO extractions (document_id, tags, category, document_date) VALUES
		 (1, '[]', 'bank', '2024-12-01'),
		 (2, '[]', 'tax', '2023-06-01'),
		 (3, '[]', 'bank', NULL)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx,
		`INSERT INTO document_pages (document_id, storage_path, page_index, content_type) VALUES
		 (1, '1/page_0.jpg', 0, 'image/jpeg'),
		 (1, '1/page_1.jpg', 1, 'image/jpeg'),
		 (2, '2/page_0.jpg', 0, 'image/jpeg')`)
	if err != nil {
		t.Fatal(err)
	}

	repo := repository.NewSQLiteDocumentRepository(db)

	t.Run("default created_desc", func(t *testing.T) {
		list, err := repo.List(ctx, repository.DocumentListFilter{Limit: 50})
		if err != nil {
			t.Fatal(err)
		}
		if len(list) != 3 {
			t.Fatalf("len=%d", len(list))
		}
		if list[0].Title.String != "NewCreate" || list[1].Title.String != "MidCreate" || list[2].Title.String != "OldCreate" {
			t.Fatalf("order = %q %q %q", list[0].Title.String, list[1].Title.String, list[2].Title.String)
		}
		if list[0].PageCount != 1 || list[2].PageCount != 2 {
			t.Fatalf("page counts New=%d Old=%d", list[0].PageCount, list[2].PageCount)
		}
	})

	t.Run("date_desc", func(t *testing.T) {
		list, err := repo.List(ctx, repository.DocumentListFilter{Sort: "date_desc", Limit: 50})
		if err != nil {
			t.Fatal(err)
		}
		// dated: 2024-12-01 (OldCreate), 2023-06-01 (NewCreate), then null MidCreate
		if list[0].Title.String != "OldCreate" || list[1].Title.String != "NewCreate" || list[2].Title.String != "MidCreate" {
			t.Fatalf("date_desc = %q %q %q", list[0].Title.String, list[1].Title.String, list[2].Title.String)
		}
	})

	t.Run("date_asc", func(t *testing.T) {
		list, err := repo.List(ctx, repository.DocumentListFilter{Sort: "date_asc", Limit: 50})
		if err != nil {
			t.Fatal(err)
		}
		if list[0].Title.String != "NewCreate" || list[1].Title.String != "OldCreate" || list[2].Title.String != "MidCreate" {
			t.Fatalf("date_asc = %q %q %q", list[0].Title.String, list[1].Title.String, list[2].Title.String)
		}
	})

	t.Run("unknown sort falls back to created_desc", func(t *testing.T) {
		list, err := repo.List(ctx, repository.DocumentListFilter{Sort: "nope", Limit: 50})
		if err != nil {
			t.Fatal(err)
		}
		if list[0].Title.String != "NewCreate" {
			t.Fatalf("fallback first = %q", list[0].Title.String)
		}
	})

	t.Run("total matches filter", func(t *testing.T) {
		n, err := repo.Count(ctx, repository.DocumentListFilter{Status: "ready"})
		if err != nil {
			t.Fatal(err)
		}
		if n != 3 {
			t.Fatalf("count ready = %d", n)
		}
		list, err := repo.List(ctx, repository.DocumentListFilter{Status: "ready", Limit: 1})
		if err != nil {
			t.Fatal(err)
		}
		if len(list) != 1 {
			t.Fatalf("limited list len = %d", len(list))
		}
	})
}
