package ocr

import (
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
)

// a4LongInches is the long edge of ISO A4 in inches (297 mm).
const a4LongInches = 297.0 / 25.4

// EstimateDPIFromA4 estimates scan/capture DPI assuming the image is a full A4 page.
// Uses the longer pixel edge as A4's long side. Result is clamped to [72, 600].
func EstimateDPIFromA4(width, height int) int {
	if width < 1 || height < 1 {
		return DefaultDPI
	}
	long := width
	if height > long {
		long = height
	}
	dpi := int(math.Round(float64(long) / a4LongInches))
	if dpi < 72 {
		return 72
	}
	if dpi > 600 {
		return 600
	}
	return dpi
}

// ImageSize returns pixel dimensions of an image file, or an error if undecodable.
func ImageSize(path string) (width, height int, err error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0, err
	}
	return cfg.Width, cfg.Height, nil
}

// ResolveDPI picks the --dpi value for a page image.
// When configuredDPI > 0 it wins (operator override via OCR_DPI).
// Otherwise DPI is estimated from pixel size assuming A4; DefaultDPI on decode failure.
func ResolveDPI(imagePath string, configuredDPI int) int {
	if configuredDPI > 0 {
		return configuredDPI
	}
	w, h, err := ImageSize(imagePath)
	if err != nil {
		return DefaultDPI
	}
	return EstimateDPIFromA4(w, h)
}
