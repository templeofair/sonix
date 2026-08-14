package ollama

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
)

func writeTestJPEG(t *testing.T, path string, w, h int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 255), G: uint8(y % 255), B: 100, A: 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
}

func TestImageToBase64ForVisionDownscales(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "big.jpg")
	writeTestJPEG(t, path, 3840, 2160)

	b64, err := ImageToBase64ForVision(path)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatal(err)
	}
	img, err := jpeg.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	b := img.Bounds()
	long := b.Dx()
	if b.Dy() > long {
		long = b.Dy()
	}
	if long > VisionMaxLongEdge {
		t.Fatalf("long edge %d > %d", long, VisionMaxLongEdge)
	}
	if b.Dx() != VisionMaxLongEdge {
		t.Fatalf("width = %d, want %d (landscape)", b.Dx(), VisionMaxLongEdge)
	}
}

func TestImageToBase64ForVisionPassthroughSmall(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "small.jpg")
	writeTestJPEG(t, path, 800, 600)

	b64, err := ImageToBase64ForVision(path)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatal(err)
	}
	img, err := jpeg.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	b := img.Bounds()
	if b.Dx() != 800 || b.Dy() != 600 {
		t.Fatalf("size = %dx%d, want 800x600", b.Dx(), b.Dy())
	}
}

// Thin 1px strokes vanish under nearest-neighbour when the sample grid misses them.
// Catmull-Rom must leave some dark signal after a 2× downscale.
func TestImageToBase64ForVisionPreservesThinStroke(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stroke.jpg")
	const w, h = 400, 400
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.White)
		}
	}
	for x := 0; x < w; x++ {
		img.Set(x, 101, color.Black) // odd row → NN 2× downscale often misses
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	t.Setenv("OLLAMA_VISION_MAX_EDGE", "200")
	b64, err := ImageToBase64ForVision(path)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatal(err)
	}
	out, err := jpeg.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	ob := out.Bounds()
	dark := 0
	for y := ob.Min.Y; y < ob.Max.Y; y++ {
		for x := ob.Min.X; x < ob.Max.X; x++ {
			r, g, b, _ := out.At(x, y).RGBA()
			// JPEG + Catmull leave mid-gray; count anything clearly darker than paper.
			if r>>8 < 200 || g>>8 < 200 || b>>8 < 200 {
				dark++
			}
		}
	}
	if dark < 50 {
		t.Fatalf("thin stroke vanished after vision downscale (dark pixels=%d)", dark)
	}
}
