package ocr

import (
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEstimateDPIFromA4(t *testing.T) {
	// 3508×2480 ≈ A4 at 300 DPI (long edge 3508)
	got := EstimateDPIFromA4(2480, 3508)
	if got < 295 || got > 305 {
		t.Fatalf("EstimateDPIFromA4(2480,3508)=%d, want ~300", got)
	}
	// 4K phone landscape long edge 3840 → ~328 DPI
	got = EstimateDPIFromA4(3840, 2160)
	if got < 320 || got > 340 {
		t.Fatalf("EstimateDPIFromA4(3840,2160)=%d, want ~328", got)
	}
	if EstimateDPIFromA4(0, 0) != DefaultDPI {
		t.Fatalf("invalid size should fall back to DefaultDPI")
	}
}

func TestResolveDPI_OverrideWins(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "page.jpg")
	writeGrayJPEG(t, path, 1000, 1400)
	if got := ResolveDPI(path, 300); got != 300 {
		t.Fatalf("override = %d, want 300", got)
	}
	got := ResolveDPI(path, 0)
	// 1400 long edge ≈ 120 DPI
	if got < 110 || got > 130 {
		t.Fatalf("auto DPI = %d, want ~120", got)
	}
}

func TestTesseract_buildArgs_AutoDPI(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "page.jpg")
	writeGrayJPEG(t, path, 2480, 3508) // ~300 DPI A4
	tess := NewTesseractOptions(TesseractOptions{Lang: "deu+eng", DPI: 0, PSM: "1"})
	args := tess.buildArgs(path, "")
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--dpi 300") && !strings.Contains(joined, "--dpi 299") && !strings.Contains(joined, "--dpi 301") {
		t.Fatalf("auto dpi args = %v", args)
	}
}

func writeGrayJPEG(t *testing.T, path string, w, h int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.White)
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 80}); err != nil {
		t.Fatal(err)
	}
}
