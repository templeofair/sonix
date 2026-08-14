package repository_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/templeofair/sonix/internal/auth"
	"github.com/templeofair/sonix/internal/database"
	"github.com/templeofair/sonix/internal/repository"
)

func TestSQLiteUserRepository_CRUD(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "u.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	repo := repository.NewSQLiteUserRepository(db)

	n, err := repo.Count(context.Background())
	if err != nil || n != 0 {
		t.Fatalf("count = %d %v", n, err)
	}
	hash, err := auth.HashPassword("pw")
	if err != nil {
		t.Fatal(err)
	}
	id, err := repo.Create(context.Background(), "sam", hash)
	if err != nil || id == 0 {
		t.Fatalf("create: %d %v", id, err)
	}
	u, err := repo.FindByUsername(context.Background(), "sam")
	if err != nil || u.ID != id || u.Username != "sam" {
		t.Fatalf("FindByUsername: %+v %v", u, err)
	}
	u2, err := repo.FindByID(context.Background(), id)
	if err != nil || u2.Username != "sam" {
		t.Fatalf("FindByID: %+v %v", u2, err)
	}
	_, err = repo.FindByUsername(context.Background(), "missing")
	if err != repository.ErrNotFound {
		t.Fatalf("want ErrNotFound got %v", err)
	}
}
