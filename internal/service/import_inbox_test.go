package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/templeofair/sonix/internal/database"
	"github.com/templeofair/sonix/internal/repository"
)

type memSettingsRepo struct {
	m map[string]string
}

func (r *memSettingsRepo) Get(_ context.Context, key string) (string, error) {
	v, ok := r.m[key]
	if !ok {
		return "", repository.ErrNotFound
	}
	return v, nil
}

func (r *memSettingsRepo) Set(_ context.Context, key, value string) error {
	if r.m == nil {
		r.m = map[string]string{}
	}
	r.m[key] = value
	return nil
}

func (r *memSettingsRepo) Delete(_ context.Context, key string) error {
	delete(r.m, key)
	return nil
}

func TestInboxImporter_importStableJPEG(t *testing.T) {
	dir := t.TempDir()
	uploads := filepath.Join(dir, "uploads")
	thumbs := filepath.Join(dir, "thumbs")
	inbox := filepath.Join(dir, "inbox")
	_ = os.MkdirAll(uploads, 0700)
	_ = os.MkdirAll(thumbs, 0700)

	db, err := database.Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	docs := NewDocumentService(repository.NewSQLiteDocumentRepository(db), uploads, thumbs)
	settings := NewSettingsService(&memSettingsRepo{m: map[string]string{
		repository.SettingsKeyImportInboxEnabled: "true",
		repository.SettingsKeyImportAutoExtract:  "false",
	}}, nil)

	imp := NewInboxImporter(inbox, docs, nil, settings)
	if err := imp.EnsureDirs(); err != nil {
		t.Fatal(err)
	}

	src := filepath.Join(inbox, "letter1.jpg")
	if err := os.WriteFile(src, []byte("\xff\xd8\xff\xd9fakejpeg"), 0600); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	if err := imp.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(src); err != nil {
		t.Fatalf("file should still be present after first poll: %v", err)
	}
	if err := imp.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatal("expected file moved after import")
	}
	processed := filepath.Join(inbox, "processed", "letter1.jpg")
	if _, err := os.Stat(processed); err != nil {
		t.Fatalf("expected processed file: %v", err)
	}

	list, err := docs.List(ctx, repository.DocumentListFilter{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if list == nil || len(list.Documents) != 1 {
		t.Fatalf("want 1 document, got %#v", list)
	}
	if list.Documents[0].Title == nil || *list.Documents[0].Title != "letter1" {
		t.Fatalf("title=%v", list.Documents[0].Title)
	}
}

func TestInboxImporter_skipWhileSizeChanging(t *testing.T) {
	dir := t.TempDir()
	inbox := filepath.Join(dir, "inbox")
	uploads := filepath.Join(dir, "uploads")
	thumbs := filepath.Join(dir, "thumbs")
	_ = os.MkdirAll(uploads, 0700)
	_ = os.MkdirAll(thumbs, 0700)

	db, err := database.Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	docs := NewDocumentService(repository.NewSQLiteDocumentRepository(db), uploads, thumbs)
	settings := NewSettingsService(&memSettingsRepo{m: map[string]string{
		repository.SettingsKeyImportInboxEnabled: "true",
		repository.SettingsKeyImportAutoExtract:  "false",
	}}, nil)
	imp := NewInboxImporter(inbox, docs, nil, settings)
	_ = imp.EnsureDirs()

	src := filepath.Join(inbox, "growing.jpg")
	_ = os.WriteFile(src, []byte("a"), 0600)
	ctx := context.Background()
	_ = imp.ScanOnce(ctx)
	_ = os.WriteFile(src, []byte("ab"), 0600)
	_ = imp.ScanOnce(ctx)
	if _, err := os.Stat(src); err != nil {
		t.Fatal("should not import while size changing")
	}
	list, err := docs.List(ctx, repository.DocumentListFilter{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Documents) != 0 {
		t.Fatalf("unexpected docs: %d", len(list.Documents))
	}
}

func TestInboxImporter_disabledSkips(t *testing.T) {
	// Run() checks enabled; ScanOnce itself does not — callers gate. Smoke the settings helpers.
	settings := NewSettingsService(&memSettingsRepo{m: map[string]string{}}, nil)
	if settings.ImportInboxEnabled(context.Background()) {
		t.Fatal("default should be disabled")
	}
	settings2 := NewSettingsService(&memSettingsRepo{m: map[string]string{
		repository.SettingsKeyImportInboxEnabled: "true",
	}}, nil)
	if !settings2.ImportInboxEnabled(context.Background()) {
		t.Fatal("expected enabled")
	}
}

func TestContentTypeForInboxFile(t *testing.T) {
	ct, ok := contentTypeForInboxFile("x.PDF")
	if !ok || ct != "application/pdf" {
		t.Fatalf("%v %v", ct, ok)
	}
	_, ok = contentTypeForInboxFile("x.txt")
	if ok {
		t.Fatal("txt should be rejected")
	}
}

func TestTitleFromFilename(t *testing.T) {
	if titleFromFilename("scan_01.pdf") != "scan_01" {
		t.Fatal(titleFromFilename("scan_01.pdf"))
	}
}

func TestInboxImporter_rejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	inbox := filepath.Join(dir, "inbox")
	uploads := filepath.Join(dir, "uploads")
	thumbs := filepath.Join(dir, "thumbs")
	outside := filepath.Join(dir, "outside")
	_ = os.MkdirAll(uploads, 0700)
	_ = os.MkdirAll(thumbs, 0700)
	_ = os.MkdirAll(outside, 0700)

	db, err := database.Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	docs := NewDocumentService(repository.NewSQLiteDocumentRepository(db), uploads, thumbs)
	settings := NewSettingsService(&memSettingsRepo{m: map[string]string{
		repository.SettingsKeyImportInboxEnabled: "true",
	}}, nil)
	imp := NewInboxImporter(inbox, docs, nil, settings)
	if err := imp.EnsureDirs(); err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(outside, "secret.jpg")
	if err := os.WriteFile(target, []byte("\xff\xd8\xff\xd9"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(inbox, "linked.jpg")
	if err := os.Symlink(target, link); err != nil {
		t.Skip("symlinks not supported in test environment")
	}

	ctx := context.Background()
	for range 2 {
		if err := imp.ScanOnce(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(link); err != nil {
		t.Fatal("symlink should remain in inbox")
	}
	list, err := docs.List(ctx, repository.DocumentListFilter{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Documents) != 0 {
		t.Fatalf("expected no import via symlink, got %d docs", len(list.Documents))
	}
}

func TestInboxImporter_rejectsOversizedFile(t *testing.T) {
	dir := t.TempDir()
	inbox := filepath.Join(dir, "inbox")
	uploads := filepath.Join(dir, "uploads")
	thumbs := filepath.Join(dir, "thumbs")
	_ = os.MkdirAll(uploads, 0700)
	_ = os.MkdirAll(thumbs, 0700)

	db, err := database.Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	docs := NewDocumentService(repository.NewSQLiteDocumentRepository(db), uploads, thumbs)
	settings := NewSettingsService(&memSettingsRepo{m: map[string]string{
		repository.SettingsKeyImportInboxEnabled: "true",
	}}, nil)
	imp := NewInboxImporter(inbox, docs, nil, settings)
	if err := imp.EnsureDirs(); err != nil {
		t.Fatal(err)
	}

	src := filepath.Join(inbox, "huge.pdf")
	payload := make([]byte, maxInboxImportBytes+1)
	copy(payload, []byte("%PDF-"))
	if err := os.WriteFile(src, payload, 0600); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	if err := imp.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if err := imp.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatal("oversized file should be moved out of inbox")
	}
	failed := filepath.Join(inbox, "failed", "huge.pdf")
	if _, err := os.Stat(failed); err != nil {
		t.Fatalf("expected failed/ copy: %v", err)
	}
	list, err := docs.List(ctx, repository.DocumentListFilter{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Documents) != 0 {
		t.Fatalf("expected no document for oversized file, got %d", len(list.Documents))
	}
}
