package ai

import (
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
	"time"

	"lumina/internal/metadata"
)

func TestLocalVisualSceneAnalyzer(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lumina_ai_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 1. Create a nature/foliage test image (dominant green)
	foliageImg := image.NewRGBA(image.Rect(0, 0, 200, 200))
	greenColor := color.RGBA{R: 30, G: 160, B: 40, A: 255}
	for y := 0; y < 200; y++ {
		for x := 0; x < 200; x++ {
			foliageImg.Set(x, y, greenColor)
		}
	}

	foliagePath := filepath.Join(tempDir, "foliage.jpg")
	f1, _ := os.Create(foliagePath)
	_ = jpeg.Encode(f1, foliageImg, &jpeg.Options{Quality: 85})
	_ = f1.Close()

	res1, err := AnalyzeLocalPhoto(context.Background(), foliagePath, nil, false)
	if err != nil {
		t.Fatalf("AnalyzeLocalPhoto failed: %v", err)
	}

	hasNatureTag := false
	for _, tag := range res1.Tags {
		if tag == "nature" || tag == "foliage" {
			hasNatureTag = true
			break
		}
	}
	if !hasNatureTag {
		t.Errorf("Expected 'nature' or 'foliage' tag for green image, got tags: %v", res1.Tags)
	}

	// 2. Test EXIF Time-of-Day tag generation
	exifNight := &metadata.ExifInfo{
		DateTimeOriginal: time.Date(2026, 3, 1, 23, 15, 0, 0, time.UTC),
		Make:             "Apple",
	}
	res2, err := AnalyzeLocalPhoto(context.Background(), foliagePath, exifNight, false)
	if err != nil {
		t.Fatalf("AnalyzeLocalPhoto with EXIF failed: %v", err)
	}

	hasNightTag := false
	for _, tag := range res2.Tags {
		if tag == "night" {
			hasNightTag = true
			break
		}
	}
	if !hasNightTag {
		t.Errorf("Expected 'night' tag for 23:15 photo, got tags: %v", res2.Tags)
	}
}

func TestCloudAIEngineDefaults(t *testing.T) {
	engine := NewCloudAIEngine("")
	if engine.HasKey() {
		t.Errorf("Expected HasKey to be false when no key is set")
	}

	engine.SetApiKey("AIzaSyDummyKeyTest123456789")
	if !engine.HasKey() {
		t.Errorf("Expected HasKey to be true after setting key")
	}
}
