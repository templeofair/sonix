package database_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/templeofair/sonix/internal/database"
)

// Cancelled QueryContext used to poison the modernc pool (interrupted (9) on
// every later query). Driver IsValid (≥1.36) must discard that connection.
func TestOpen_RecoversAfterContextCancel(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "interrupt.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _ = db.ExecContext(ctx, `SELECT 1`)

	if _, err := db.ExecContext(context.Background(), `SELECT 1`); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "interrupted") {
			t.Fatalf("pool still poisoned after cancel: %v", err)
		}
		t.Fatalf("SELECT after cancel: %v", err)
	}
}
