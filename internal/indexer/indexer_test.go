package indexer

import (
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"lumina/internal/ai"
	"lumina/internal/db"
	"lumina/internal/thumbs"
)

func TestCatalogSweepAndRepairActions(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lumina_indexer_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbDir := filepath.Join(tempDir, "data")
	database, err := db.Open(dbDir)
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer database.Close()

	packStore, err := thumbs.NewPackStore(dbDir)
	if err != nil {
		t.Fatalf("Failed to create PackStore: %v", err)
	}
	defer packStore.Close()

	cloudAI := ai.NewCloudAIEngine("")
	idx := NewCatalogIndexer(database, packStore, cloudAI)

	// Create a source watched folder
	srcFolder := filepath.Join(tempDir, "SourcePhotos")
	_ = os.MkdirAll(srcFolder, 0755)

	// 1. Create a regular photo
	regularPhotoPath := filepath.Join(srcFolder, "CameraPhoto.jpg")
	f1, _ := os.Create(regularPhotoPath)
	img1 := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img1.Set(x, y, color.RGBA{R: 200, G: 100, B: 50, A: 255})
		}
	}
	_ = jpeg.Encode(f1, img1, &jpeg.Options{Quality: 90})
	_ = f1.Close()

	// 2. Create a screenshot PNG (1080x2400)
	screenshotPath := filepath.Join(srcFolder, "Screenshot_20260301.png")
	f2, _ := os.Create(screenshotPath)
	img2 := image.NewRGBA(image.Rect(0, 0, 100, 100))
	_ = png.Encode(f2, img2)
	_ = f2.Close()

	// Add watched folder with route override "Family"
	ctx := context.Background()
	_ = database.AddWatchedFolder(ctx, srcFolder, "Family", true)

	// Run catalog sweep
	err = idx.StartCatalogSweep(ctx, false)
	if err != nil {
		t.Fatalf("StartCatalogSweep failed: %v", err)
	}

	// Verify catalog items
	stats := database.MemIndex.GetStats()
	if stats.TotalPhotos < 2 {
		t.Errorf("Expected at least 2 indexed photos, got %d", stats.TotalPhotos)
	}
	if stats.Screenshots != 1 {
		t.Errorf("Expected 1 screenshot identified, got %d", stats.Screenshots)
	}

	// Verify Rule 1: Screenshot hidden from "All Photos" tab
	allPhotos := database.MemIndex.Filter(db.FilterOptions{Tab: "all"})
	if len(allPhotos) != 1 {
		t.Errorf("Rule 1 Violation: 'All Photos' should contain only 1 photo (non-screenshot), got %d", len(allPhotos))
	}

	// Test ReVerifyScreenshots
	if err := idx.ReVerifyScreenshots(ctx); err != nil {
		t.Errorf("ReVerifyScreenshots failed: %v", err)
	}
}
