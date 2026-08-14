package ollama

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"os"
	"strconv"

	xdraw "golang.org/x/image/draw"
)

// VisionMaxLongEdge is the max width/height (px) sent to Ollama vision.
// ~2048 on A4 is roughly 175 DPI (vs ~130 DPI at 1536). Stored uploads are unchanged.
// Override with OLLAMA_VISION_MAX_EDGE (512–4096).
const VisionMaxLongEdge = 2048

// VisionJPEGQuality is the JPEG quality used when re-encoding for vision.
const VisionJPEGQuality = 85

// ImageToBase64 reads a file and returns base64-encoded string (no resize).
func ImageToBase64(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// ImageToBase64ForVision returns a JPEG base64 payload sized for Ollama vision.
// Images whose long edge exceeds VisionMaxLongEdge (or OLLAMA_VISION_MAX_EDGE)
// are downscaled with Catmull-Rom (not nearest-neighbour) so thin strokes and
// umlaut dots survive. Decode failures fall back to the raw file bytes.
func ImageToBase64ForVision(path string) (string, error) {
	maxEdge := VisionMaxLongEdge
	if v := os.Getenv("OLLAMA_VISION_MAX_EDGE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 512 && n <= 4096 {
			maxEdge = n
		}
	}

	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	src, _, err := image.Decode(f)
	if err != nil {
		return ImageToBase64(path)
	}

	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()
	if sw <= 0 || sh <= 0 {
		return "", fmt.Errorf("invalid image size")
	}

	nw, nh := sw, sh
	long := sw
	if sh > long {
		long = sh
	}
	if long > maxEdge {
		if sw >= sh {
			nw = maxEdge
			nh = sh * maxEdge / sw
		} else {
			nh = maxEdge
			nw = sw * maxEdge / sh
		}
		if nw < 1 {
			nw = 1
		}
		if nh < 1 {
			nh = 1
		}
	}

	var outImg image.Image = src
	if nw != sw || nh != sh {
		dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
		// Catmull-Rom preserves thin strokes far better than point sampling.
		xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, b, xdraw.Over, nil)
		outImg = dst
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, outImg, &jpeg.Options{Quality: VisionJPEGQuality}); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}
