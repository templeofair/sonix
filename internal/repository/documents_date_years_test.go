package repository_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/templeofair/sonix/internal/database"
	"github.com/templeofair/sonix/internal/repository"
)

// seedDateBuckets inserts documents covering every letter-date bucket:
// two in 2024, one in 2023, and three undated (NULL date, empty date, no extraction row).
func seedDateBuckets(t *testing.T) *repository.SQLiteDocumentRepository {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "dates.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	ctx := context.Background()

	_, err = db.ExecContext(ctx,
		`INSERT INTO documents (id, title, status, created_at, updated_at) VALUES
		 (1, 'Dated2024a', 'ready', '2025-01-01T00:00:00Z', '2025-01-01T00:00:00Z'),
		 (2, 'Dated2024b', 'ready', '2025-01-02T00:00:00Z', '2025-01-02T00:00:00Z'),
		 (3, 'Dated2023',  'ready', '2025-01-03T00:00:00Z', '2025-01-03T00:00:00Z'),
		 (4, 'NullDate',   'ready', '2025-01-04T00:00:00Z', '2025-01-04T00:00:00Z'),
		 (5, 'EmptyDate',  'ready', '2025-01-05T00:00:00Z', '2025-01-05T00:00:00Z'),
		 (6, 'NoRow',      'pending', '2025-01-06T00:00:00Z', '2025-01-06T00:00:00Z')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx,
		`INSERT INTO extractions (document_id, tags, summary, document_date) VALUES
		 (1, '[]', 'alpha letter', '2024-12-01'),
		 (2, '[]', 'beta letter',  '2024-01-15'),
		 (3, '[]', 'gamma letter', '2023-06-01'),
		 (4, '[]', 'delta letter', NULL),
		 (5, '[]', 'epsilon letter', '')`)
	if err != nil {
		t.Fatal(err)
	}
	// Mirror what the extraction path writes, so q= has something to match.
	_, err = db.ExecContext(ctx,
		`INSERT INTO extractions_fts (document_id, summary, full_text_original, full_text_english) VALUES
		 (1, 'alpha letter', '', ''),
		 (2, 'beta letter', '', ''),
		 (3, 'gamma letter', '', ''),
		 (4, 'delta letter', '', ''),
		 (5, 'epsilon letter', '', '')`)
	if err != nil {
		t.Fatal(err)
	}
	return repository.NewSQLiteDocumentRepository(db)
}

func TestDocumentRepository_DocumentDateYears(t *testing.T) {
	repo := seedDateBuckets(t)
	ctx := context.Background()

	years, undated, err := repo.DocumentDateYears(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(years) != 2 {
		t.Fatalf("years = %#v", years)
	}
	if years[0].Year != "2024" || years[0].Count != 2 {
		t.Fatalf("first bucket = %#v", years[0])
	}
	if years[1].Year != "2023" || years[1].Count != 1 {
		t.Fatalf("second bucket = %#v", years[1])
	}
	// NULL date, empty date, and no extraction row all count as undated.
	if undated != 3 {
		t.Fatalf("undated = %d", undated)
	}

	// Every document lands in exactly one bucket.
	total, err := repo.Count(ctx, repository.DocumentListFilter{})
	if err != nil {
		t.Fatal(err)
	}
	var sum int64
	for _, y := range years {
		sum += y.Count
	}
	if sum+undated != total {
		t.Fatalf("bucket sum %d + undated %d != total %d", sum, undated, total)
	}
}

func TestDocumentRepository_DocumentDateYearsEmptyDB(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "empty.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	repo := repository.NewSQLiteDocumentRepository(db)

	years, undated, err := repo.DocumentDateYears(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(years) != 0 || undated != 0 {
		t.Fatalf("years = %#v undated = %d", years, undated)
	}
}

func TestDocumentRepository_ListUndated(t *testing.T) {
	repo := seedDateBuckets(t)
	ctx := context.Background()

	t.Run("undated bucket", func(t *testing.T) {
		list, err := repo.List(ctx, repository.DocumentListFilter{Undated: true, Limit: 50})
		if err != nil {
			t.Fatal(err)
		}
		if len(list) != 3 {
			t.Fatalf("len = %d (%#v)", len(list), list)
		}
		// Existing default sort: created_at DESC.
		if list[0].Title.String != "NoRow" || list[1].Title.String != "EmptyDate" || list[2].Title.String != "NullDate" {
			t.Fatalf("order = %q %q %q", list[0].Title.String, list[1].Title.String, list[2].Title.String)
		}
		n, err := repo.Count(ctx, repository.DocumentListFilter{Undated: true})
		if err != nil {
			t.Fatal(err)
		}
		if n != 3 {
			t.Fatalf("count = %d", n)
		}
	})

	t.Run("undated with q", func(t *testing.T) {
		list, err := repo.List(ctx, repository.DocumentListFilter{Undated: true, Q: "delta", Limit: 50})
		if err != nil {
			t.Fatal(err)
		}
		if len(list) != 1 || list[0].Title.String != "NullDate" {
			t.Fatalf("undated+q = %#v", list)
		}
		// A dated letter stays out of the bucket even when the search matches it.
		list, err = repo.List(ctx, repository.DocumentListFilter{Undated: true, Q: "alpha", Limit: 50})
		if err != nil {
			t.Fatal(err)
		}
		if len(list) != 0 {
			t.Fatalf("dated match leaked into undated: %#v", list)
		}
		// Without the flag the same search still finds it.
		list, err = repo.List(ctx, repository.DocumentListFilter{Q: "alpha", Limit: 50})
		if err != nil {
			t.Fatal(err)
		}
		if len(list) != 1 || list[0].Title.String != "Dated2024a" {
			t.Fatalf("q without undated = %#v", list)
		}
	})

	t.Run("undated wins over document_date range", func(t *testing.T) {
		list, err := repo.List(ctx, repository.DocumentListFilter{
			Undated:          true,
			DocumentDateFrom: "2024-01-01",
			DocumentDateTo:   "2024-12-31",
			Limit:            50,
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(list) != 3 {
			t.Fatalf("len = %d", len(list))
		}
	})

	t.Run("absent param lists everything", func(t *testing.T) {
		list, err := repo.List(ctx, repository.DocumentListFilter{Limit: 50})
		if err != nil {
			t.Fatal(err)
		}
		if len(list) != 6 {
			t.Fatalf("len = %d", len(list))
		}
	})
}
