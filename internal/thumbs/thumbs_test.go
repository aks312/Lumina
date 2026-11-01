package thumbs

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
)

func TestPackStoreAndLRU(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lumina_thumbs_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	ps, err := NewPackStore(tempDir)
	if err != nil {
		t.Fatalf("NewPackStore failed: %v", err)
	}
	defer ps.Close()

	// Put 3 sample thumbnails
	dummyThumb1 := []byte("fake_jpeg_data_1")
	dummyThumb2 := []byte("fake_jpeg_data_2")

	if err := ps.Put(101, dummyThumb1); err != nil {
		t.Fatalf("Put 101 failed: %v", err)
	}
	if err := ps.Put(102, dummyThumb2); err != nil {
		t.Fatalf("Put 102 failed: %v", err)
	}

	if !ps.Has(101) || !ps.Has(102) || ps.Has(999) {
		t.Errorf("PackStore.Has returned incorrect existence")
	}

	// Read from LRU / disk
	res1, err := ps.Get(101)
	if err != nil || string(res1) != "fake_jpeg_data_1" {
		t.Errorf("Get 101 mismatch: got %s, err %v", string(res1), err)
	}

	// Close and reopen to verify index persistence
	_ = ps.Close()

	ps2, err := NewPackStore(tempDir)
	if err != nil {
		t.Fatalf("Reopening PackStore failed: %v", err)
	}
	defer ps2.Close()

	if !ps2.Has(101) || !ps2.Has(102) {
		t.Errorf("Index persistence failed across re-opens")
	}

	res2, err := ps2.Get(102)
	if err != nil || string(res2) != "fake_jpeg_data_2" {
		t.Errorf("Get 102 from persisted index failed: %s, %v", string(res2), err)
	}
}

func TestGenerateThumbnailAndRotate(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lumina_gen_thumb_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test JPEG image
	testImg := image.NewRGBA(image.Rect(0, 0, 800, 600))
	for y := 0; y < 600; y++ {
		for x := 0; x < 800; x++ {
			testImg.Set(x, y, color.RGBA{R: uint8(x % 255), G: uint8(y % 255), B: 120, A: 255})
		}
	}

	imgPath := filepath.Join(tempDir, "sample.jpg")
	f, err := os.Create(imgPath)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	if err := jpeg.Encode(f, testImg, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("Failed to encode jpeg: %v", err)
	}
	_ = f.Close()

	thumbBytes, err := GenerateThumbnail(imgPath, 200)
	if err != nil || len(thumbBytes) == 0 {
		t.Fatalf("GenerateThumbnail failed: len=%d, err=%v", len(thumbBytes), err)
	}

	decodedThumb, _, err := image.Decode(bytes.NewReader(thumbBytes))
	if err != nil {
		t.Fatalf("Failed to decode generated thumbnail: %v", err)
	}
	b := decodedThumb.Bounds()
	if b.Dx() > 200 || b.Dy() > 200 {
		t.Errorf("Expected thumbnail max dim <= 200, got %dx%d", b.Dx(), b.Dy())
	}

	// Test 90-degree rotation
	rotated := RotateImage(decodedThumb, 90)
	if rotated.Bounds().Dx() != b.Dy() || rotated.Bounds().Dy() != b.Dx() {
		t.Errorf("Expected rotated bounds %dx%d, got %dx%d", b.Dy(), b.Dx(), rotated.Bounds().Dx(), rotated.Bounds().Dy())
	}
}
