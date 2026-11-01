package faces

import (
	"image"
	"image/color"
	"testing"
)

func TestDetectFaceRegions(t *testing.T) {
	// Create image with a skin-tone oval/rectangle representing a face candidate
	img := image.NewRGBA(image.Rect(0, 0, 300, 300))
	skinColor := color.RGBA{R: 210, G: 160, B: 120, A: 255}

	// Draw skin-tone patch at (100, 80) of size (80, 100)
	for y := 80; y < 180; y++ {
		for x := 100; x < 180; x++ {
			img.Set(x, y, skinColor)
		}
	}

	regions := DetectFaceRegions(img)
	if len(regions) == 0 {
		t.Fatalf("Expected to detect at least 1 face candidate region, got 0")
	}

	reg := regions[0]
	if reg.Width < 50 || reg.Height < 70 {
		t.Errorf("Unexpected face bounds: %+v", reg)
	}
}
