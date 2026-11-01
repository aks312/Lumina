package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"lumina/internal/db"
	"lumina/internal/thumbs"
)

func TestRawMediaAndThumbConcurrencyDuringSweep(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lumina_app_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	database, err := db.Open(tempDir)
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}
	defer database.Close()

	packStore, err := thumbs.NewPackStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to open packStore: %v", err)
	}
	defer packStore.Close()

	app := &App{
		database:  database,
		packStore: packStore,
	}

	// Insert test photo and test video
	testPhotoPath := filepath.Join(tempDir, "sample_test.jpg")
	// Write minimal 1x1 JPEG
	fakeJpg := []byte{
		0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01,
		0x01, 0x01, 0x00, 0x48, 0x00, 0x48, 0x00, 0x00, 0xFF, 0xDB, 0x00, 0x43,
		0x00, 0x08, 0x06, 0x06, 0x07, 0x06, 0x05, 0x08, 0x07, 0x07, 0x07, 0x09,
		0x09, 0x08, 0x0A, 0x0C, 0x14, 0x0D, 0x0C, 0x0B, 0x0B, 0x0C, 0x19, 0x12,
		0x13, 0x0F, 0x14, 0x1D, 0x1A, 0x1F, 0x1E, 0x1D, 0x1A, 0x1C, 0x1C, 0x20,
		0x24, 0x2E, 0x27, 0x20, 0x22, 0x2C, 0x23, 0x1C, 0x1C, 0x28, 0x37, 0x29,
		0x2C, 0x30, 0x31, 0x34, 0x34, 0x34, 0x1F, 0x27, 0x39, 0x3D, 0x38, 0x32,
		0x3C, 0x2E, 0x33, 0x34, 0x32, 0xFF, 0xC0, 0x00, 0x0B, 0x08, 0x00, 0x01,
		0x00, 0x01, 0x01, 0x01, 0x11, 0x00, 0xFF, 0xC4, 0x00, 0x1F, 0x00, 0x00,
		0x01, 0x05, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0A, 0x0B, 0xFF, 0xDA, 0x00, 0x08, 0x01, 0x01, 0x00, 0x00, 0x3F,
		0x00, 0x7F, 0x00, 0xFF, 0xD9,
	}
	_ = os.WriteFile(testPhotoPath, fakeJpg, 0644)

	testVideoPath := filepath.Join(tempDir, "sample_clip.mp4")
	_ = os.WriteFile(testVideoPath, []byte("fake mp4 movie content"), 0644)

	ctx := context.Background()
	pID, err := database.UpsertPhoto(ctx, &db.PhotoRecord{
		OriginalPath: testPhotoPath,
		FileName:     "sample_test.jpg",
		FileSize:     int64(len(fakeJpg)),
		DateTaken:    time.Now(),
		Year:         2026,
		Month:        11,
		Day:          1,
		Platform:     "camera",
		ShowInAll:    true,
	})
	if err != nil {
		t.Fatalf("UpsertPhoto failed: %v", err)
	}

	vID, err := database.UpsertPhoto(ctx, &db.PhotoRecord{
		OriginalPath: testVideoPath,
		FileName:     "sample_clip.mp4",
		FileSize:     1024,
		DateTaken:    time.Now(),
		Year:         2026,
		Month:        11,
		Day:          1,
		Platform:     "camera",
		ShowInAll:    true,
		IsVideo:      true,
	})
	if err != nil {
		t.Fatalf("Upsert video failed: %v", err)
	}

	// Spin up 10 concurrent requests to /raw/{id} and /thumbs/{id}
	var wg sync.WaitGroup
	errCh := make(chan error, 30)

	for i := 0; i < 15; i++ {
		wg.Add(2)

		// Concurrent /raw/ request
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("GET", fmt.Sprintf("/raw/%d", pID), nil)
			rec := httptest.NewRecorder()
			app.ServeRawMedia(rec, req)

			if rec.Code != http.StatusOK {
				errCh <- fmt.Errorf("ServeRawMedia returned code %d", rec.Code)
			}
			if rec.Header().Get("Accept-Ranges") != "bytes" {
				errCh <- fmt.Errorf("ServeRawMedia missing Accept-Ranges header")
			}
		}()

		// Concurrent /thumbs/ request for video
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("GET", fmt.Sprintf("/thumbs/%d", vID), nil)
			rec := httptest.NewRecorder()
			app.ServeThumbnail(rec, req)

			if rec.Code != http.StatusOK {
				errCh <- fmt.Errorf("ServeThumbnail returned code %d", rec.Code)
			}
			if len(rec.Body.Bytes()) < 1000 {
				errCh <- fmt.Errorf("ServeThumbnail video returned too small: %d bytes", len(rec.Body.Bytes()))
			}
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("Concurrency test failed: %v", err)
	}
}
