package repository_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/templeofair/sonix/internal/database"
	"github.com/templeofair/sonix/internal/repository"
)

func TestDocumentRepository_TagFilterExactMatch(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "tags.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	ctx := context.Background()

	_, err = db.ExecContext(ctx,
		`INSERT INTO documents (id, title, status, created_at, updated_at) VALUES
		 (1, 'Bank letter', 'ready', '2024-01-01T00:00:00Z', '2024-01-01T00:00:00Z'),
		 (2, 'Bankruptcy notice', 'ready', '2024-02-01T00:00:00Z', '2024-02-01T00:00:00Z'),
		 (3, 'Multi tag', 'ready', '2024-03-01T00:00:00Z', '2024-03-01T00:00:00Z')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx,
		`INSERT INTO extractions (document_id, tags, summary, full_text_original) VALUES
		 (1, '["bank","invoice"]', 'bank letter', 'paid from bank account'),
		 (2, '["bankruptcy"]', 'bankruptcy filing', 'chapter bankruptcy court'),
		 (3, '["bank","tax"]', 'tax from bank', 'bank transfer tax')`)
	if err != nil {
		t.Fatal(err)
	}

	repo := repository.NewSQLiteDocumentRepository(db)

	t.Run("tag filter bank is exact", func(t *testing.T) {
		list, err := repo.List(ctx, repository.DocumentListFilter{Tag: "bank", Limit: 50})
		if err != nil {
			t.Fatal(err)
		}
		if len(list) != 2 {
			t.Fatalf("tag=bank len=%d want 2", len(list))
		}
		got := map[string]bool{}
		for _, d := range list {
			got[d.Title.String] = true
		}
		if !got["Bank letter"] || !got["Multi tag"] || got["Bankruptcy notice"] {
			t.Fatalf("tag=bank titles = %#v", got)
		}
	})

	t.Run("tag filter bankruptcy still matches", func(t *testing.T) {
		list, err := repo.List(ctx, repository.DocumentListFilter{Tag: "bankruptcy", Limit: 50})
		if err != nil {
			t.Fatal(err)
		}
		if len(list) != 1 || list[0].Title.String != "Bankruptcy notice" {
			t.Fatalf("tag=bankruptcy = %#v", list)
		}
	})

	t.Run("free-text q still substring-matches tags", func(t *testing.T) {
		// Intentionally unchanged: q=bank must still find the bankruptcy-tagged doc via tags LIKE.
		list, err := repo.List(ctx, repository.DocumentListFilter{Q: "bank", Limit: 50})
		if err != nil {
			t.Fatal(err)
		}
		got := map[string]bool{}
		for _, d := range list {
			got[d.Title.String] = true
		}
		if !got["Bankruptcy notice"] {
			t.Fatalf("q=bank should still substring-match tag bankruptcy; got %#v", got)
		}
		if !got["Bank letter"] || !got["Multi tag"] {
			t.Fatalf("q=bank missing expected docs; got %#v", got)
		}
	})
}
