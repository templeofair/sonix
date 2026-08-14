package service_test

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/jpeg"
	"io"
	"path/filepath"
	"testing"

	"github.com/templeofair/sonix/internal/database"
	"github.com/templeofair/sonix/internal/repository"
	"github.com/templeofair/sonix/internal/service"
)

func TestUploadPages_TooManyPages(t *testing.T) {
	dir := t.TempDir()
	db, err := database.Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	svc := service.NewDocumentService(repository.NewSQLiteDocumentRepository(db), filepath.Join(dir, "up"), filepath.Join(dir, "th"))
	svc.SetLimits(1, 0)
	ctx := context.Background()
	id, err := svc.Create(ctx, "cap")
	if err != nil {
		t.Fatal(err)
	}
	data := jpegBytes(t)
	open := func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(data)), nil }
	if err := svc.UploadPages(ctx, id, []service.PageUpload{{ContentType: "image/jpeg", Open: open}}); err != nil {
		t.Fatal(err)
	}
	err = svc.UploadPages(ctx, id, []service.PageUpload{{ContentType: "image/jpeg", Open: open}})
	if !errors.Is(err, service.ErrTooManyPages) {
		t.Fatalf("got %v, want ErrTooManyPages", err)
	}
}

func jpegBytes(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80}); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
