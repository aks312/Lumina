package db

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDatabaseAndMemIndex(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lumina_db_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	db, err := Open(tempDir)
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// 1. Test Upsert Photo
	now := time.Now()
	p1 := &PhotoRecord{
		OriginalPath:    filepath.Join(tempDir, "IMG_0001.JPG"),
		FileName:        "IMG_0001.JPG",
		FileSize:        2048576,
		Sha256:          "hash_photo_1",
		Width:           4032,
		Height:          3024,
		DateTaken:       now,
		Year:            now.Year(),
		Month:           int(now.Month()),
		Day:             now.Day(),
		CameraMake:      "Apple",
		CameraModel:     "iPhone 15 Pro",
		CanonicalDevice: "Apple_iPhone_15_Pro",
		Platform:        "ios",
		IsScreenshot:    false,
		ScreenshotScore: 0,
		RouteFolder:     "iOS",
		ShowInAll:       true,
		Tags:            "nature,sunset,outdoor",
		OcrText:         "Grand Canyon National Park",
		AiDescription:   "A breathtaking sunset over mountains",
	}

	id1, err := db.UpsertPhoto(ctx, p1)
	if err != nil {
		t.Fatalf("UpsertPhoto p1 failed: %v", err)
	}
	if id1 == 0 {
		t.Fatalf("Expected non-zero ID for p1")
	}

	// 2. Test Screenshot Photo (Priority 1: Must be hidden from All Photos and iOS tabs)
	p2 := &PhotoRecord{
		OriginalPath:    filepath.Join(tempDir, "Screenshot_2026.png"),
		FileName:        "Screenshot_2026.png",
		FileSize:        512000,
		Sha256:          "hash_screenshot_2",
		Width:           1080,
		Height:          2400,
		DateTaken:       now.Add(-time.Hour),
		Year:            now.Year(),
		Month:           int(now.Month()),
		Day:             now.Day(),
		CanonicalDevice: "Samsung_Galaxy_S23",
		Platform:        "android",
		IsScreenshot:    true,
		ScreenshotScore: 100,
		RouteFolder:     "Screenshots",
		ShowInAll:       false,
		Tags:            "screen_capture,receipt",
		OcrText:         "Payment Successful $45.00",
	}

	id2, err := db.UpsertPhoto(ctx, p2)
	if err != nil {
		t.Fatalf("UpsertPhoto p2 failed: %v", err)
	}

	// 3. Test MemIndex Filtering (Rule 1 Compliance)
	// All photos tab -> should have p1, but NOT p2 (screenshot)
	allResults := db.MemIndex.Filter(FilterOptions{Tab: "all"})
	if len(allResults) != 1 || allResults[0] != id1 {
		t.Errorf("Rule 1 Violation: 'All Photos' tab must exclude screenshots. Got IDs: %v", allResults)
	}

	// iOS tab -> should have p1
	iosResults := db.MemIndex.Filter(FilterOptions{Tab: "ios"})
	if len(iosResults) != 1 || iosResults[0] != id1 {
		t.Errorf("iOS tab filter failed. Got IDs: %v", iosResults)
	}

	// Android tab -> p2 is android but is_screenshot=true, so android tab must EXCLUDE p2!
	androidResults := db.MemIndex.Filter(FilterOptions{Tab: "android"})
	if len(androidResults) != 0 {
		t.Errorf("Rule 1 Violation: Screenshots must NOT appear in Android tab! Got IDs: %v", androidResults)
	}

	// Screenshots tab -> should have ONLY p2
	screenshotResults := db.MemIndex.Filter(FilterOptions{Tab: "screenshots"})
	if len(screenshotResults) != 1 || screenshotResults[0] != id2 {
		t.Errorf("Screenshots tab should have p2. Got IDs: %v", screenshotResults)
	}

	// 4. Test FTS5 Full-Text Search
	ftsResults, err := db.SearchPhotosFTS(ctx, "Canyon", 10)
	if err != nil {
		t.Fatalf("FTS5 search failed: %v", err)
	}
	if len(ftsResults) == 0 || ftsResults[0] != id1 {
		t.Errorf("Expected FTS search to find Canyon in p1 OCR text. Got: %v", ftsResults)
	}

	// 5. Test Watched Folders
	testFolder := filepath.Join(tempDir, "PhoneSync")
	if err := db.AddWatchedFolder(ctx, testFolder, "Amruta", true); err != nil {
		t.Fatalf("AddWatchedFolder failed: %v", err)
	}
	folders, err := db.GetAllWatchedFolders(ctx)
	if err != nil || len(folders) != 1 {
		t.Fatalf("GetAllWatchedFolders failed: %v", err)
	}
	if folders[0].RouteOverride != "Amruta" {
		t.Errorf("Expected RouteOverride 'Amruta', got: %s", folders[0].RouteOverride)
	}

	// 6. Test Device Mappings
	if err := db.SetDeviceMapping(ctx, "Samsung_Galaxy_S23", "Akash"); err != nil {
		t.Fatalf("SetDeviceMapping failed: %v", err)
	}
	mappings, err := db.GetAllDeviceMappings(ctx)
	if err != nil || len(mappings) != 1 {
		t.Fatalf("GetAllDeviceMappings failed: %v", err)
	}
	if mappings[0].CustomFolder != "Akash" {
		t.Errorf("Expected CustomFolder 'Akash', got: %s", mappings[0].CustomFolder)
	}

	// 7. Test User Field Modification (Priority 3: is_user_modified = 1)
	if err := db.UpdatePhotoUserFields(ctx, id1, "PersonalFavorites", "custom", 90, 5, true, true); err != nil {
		t.Fatalf("UpdatePhotoUserFields failed: %v", err)
	}
	updated, err := db.GetPhotoByID(ctx, id1)
	if err != nil {
		t.Fatalf("GetPhotoByID failed: %v", err)
	}
	if !updated.IsUserModified || updated.Rotation != 90 || updated.Rating != 5 || updated.RouteFolder != "PersonalFavorites" {
		t.Errorf("User modifications not persisted properly: %+v", updated)
	}

	// 8. Test FastLookup in MemIndex (0ms RAM path resolution)
	path1, rot1, isVid1, ok1 := db.MemIndex.FastLookup(id1)
	if !ok1 || path1 != p1.OriginalPath || rot1 != 90 || isVid1 {
		t.Errorf("FastLookup p1 failed: ok=%v, path=%s, rot=%d, isVid=%v", ok1, path1, rot1, isVid1)
	}

	// Also verify re-warming preserves FastLookup
	if err := db.WarmMemIndex(); err != nil {
		t.Fatalf("WarmMemIndex failed: %v", err)
	}
	path2, _, _, ok2 := db.MemIndex.FastLookup(id1)
	if !ok2 || path2 != p1.OriginalPath {
		t.Errorf("FastLookup after WarmMemIndex failed: ok=%v, path=%s", ok2, path2)
	}
}
