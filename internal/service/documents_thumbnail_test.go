package service_test

import (
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"

	"github.com/templeofair/sonix/internal/database"
	"github.com/templeofair/sonix/internal/repository"
	"github.com/templeofair/sonix/internal/service"
)

func TestDocumentService_EnsureThumbnail(t *testing.T) {
	dir := t.TempDir()
	db, err := database.Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	uploads := filepath.Join(dir, "uploads")
	thumbs := filepath.Join(dir, "thumbs")
	_ = os.MkdirAll(filepath.Join(uploads, "1"), 0700)

	// 800x600 source JPEG
	srcPath := filepath.Join(uploads, "1", "page_0.jpg")
	if err := writeTestJPEG(srcPath, 800, 600); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	_, err = db.ExecContext(ctx, `INSERT INTO documents (id, status) VALUES (1, 'ready')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx,
		`INSERT INTO document_pages (document_id, storage_path, page_index, content_type) VALUES (1, '1/page_0.jpg', 0, 'image/jpeg')`)
	if err != nil {
		t.Fatal(err)
	}

	svc := service.NewDocumentService(repository.NewSQLiteDocumentRepository(db), uploads, thumbs)
	thumb, err := svc.EnsureThumbnail(ctx, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !thumb.FromCache || thumb.ContentType != "image/jpeg" {
		t.Fatalf("thumb = %#v", thumb)
	}
	f, err := os.Open(thumb.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := jpeg.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != service.ThumbnailWidth {
		t.Fatalf("width = %d, want %d", img.Bounds().Dx(), service.ThumbnailWidth)
	}

	// Second call hits cache
	thumb2, err := svc.EnsureThumbnail(ctx, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if thumb2.Path != thumb.Path {
		t.Fatalf("cache path changed: %s vs %s", thumb.Path, thumb2.Path)
	}
}

func TestDocumentService_EnsureThumbnail_FallbackOnBadImage(t *testing.T) {
	dir := t.TempDir()
	db, err := database.Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	uploads := filepath.Join(dir, "uploads")
	thumbs := filepath.Join(dir, "thumbs")
	_ = os.MkdirAll(filepath.Join(uploads, "2"), 0700)
	srcPath := filepath.Join(uploads, "2", "page_0.jpg")
	if err := os.WriteFile(srcPath, []byte("not-an-image"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	_, _ = db.ExecContext(ctx, `INSERT INTO documents (id, status) VALUES (2, 'ready')`)
	_, _ = db.ExecContext(ctx,
		`INSERT INTO document_pages (document_id, storage_path, page_index, content_type) VALUES (2, '2/page_0.jpg', 0, 'image/jpeg')`)

	svc := service.NewDocumentService(repository.NewSQLiteDocumentRepository(db), uploads, thumbs)
	thumb, err := svc.EnsureThumbnail(ctx, 2, 0)
	if err != nil {
		t.Fatal(err)
	}
	if thumb.FromCache {
		t.Fatal("expected full-image fallback, not cache")
	}
	if thumb.Path != srcPath {
		t.Fatalf("path = %s, want %s", thumb.Path, srcPath)
	}
}

func TestDocumentService_RotatePage(t *testing.T) {
	dir := t.TempDir()
	db, err := database.Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	uploads := filepath.Join(dir, "uploads")
	thumbs := filepath.Join(dir, "thumbs")
	_ = os.MkdirAll(filepath.Join(uploads, "1"), 0700)
	srcPath := filepath.Join(uploads, "1", "page_0.jpg")
	if err := writeTestJPEG(srcPath, 40, 20); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	_, err = db.ExecContext(ctx, `INSERT INTO documents (id, status) VALUES (1, 'ready')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx,
		`INSERT INTO document_pages (document_id, storage_path, page_index, content_type) VALUES (1, '1/page_0.jpg', 0, 'image/jpeg')`)
	if err != nil {
		t.Fatal(err)
	}

	svc := service.NewDocumentService(repository.NewSQLiteDocumentRepository(db), uploads, thumbs)
	if _, err := svc.EnsureThumbnail(ctx, 1, 0); err != nil {
		t.Fatal(err)
	}
	cachePath := filepath.Join(thumbs, "1", "0.jpg")
	if _, err := os.Stat(cachePath); err != nil {
		t.Fatalf("thumb cache missing: %v", err)
	}

	if err := svc.RotatePage(ctx, 1, 0, 90); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	img, err := jpeg.Decode(f)
	_ = f.Close()
	if err != nil {
		t.Fatal(err)
	}
	b := img.Bounds()
	if b.Dx() != 20 || b.Dy() != 40 {
		t.Fatalf("size after 90° = %dx%d, want 20x40", b.Dx(), b.Dy())
	}
	if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
		t.Fatal("expected thumb cache removed after rotate")
	}
	if err := svc.RotatePage(ctx, 1, 0, 45); err == nil {
		t.Fatal("expected invalid degrees error")
	}
}

func writeTestJPEG(path string, w, h int) error {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 100, A: 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return jpeg.Encode(f, img, &jpeg.Options{Quality: 90})
}
