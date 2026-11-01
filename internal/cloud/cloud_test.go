package cloud

import (
	"context"
	"os"
	"testing"

	"lumina/internal/db"
)

func TestPKCEGeneration(t *testing.T) {
	v, c, err := GeneratePKCE()
	if err != nil {
		t.Fatalf("GeneratePKCE failed: %v", err)
	}
	if len(v) < 32 {
		t.Errorf("Expected verifier length >= 32, got %d", len(v))
	}
	if len(c) < 32 {
		t.Errorf("Expected challenge length >= 32, got %d", len(c))
	}
	if v == c {
		t.Errorf("Verifier and Challenge should differ (Challenge is SHA256 hashed)")
	}
}

func TestGooglePhotosManagerInit(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lumina_cloud_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	database, err := db.Open(tempDir)
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer database.Close()

	mgr := NewGooglePhotosSyncManager(database)
	if mgr.IsConnected() {
		t.Errorf("Expected not connected initially")
	}

	// Test cloud registry deduplication check
	ctx := context.Background()
	testHash := "hash_sync_test_123"
	if database.IsSyncedToGPhotos(ctx, testHash) {
		t.Errorf("Expected hash to not be synced initially")
	}

	_ = database.MarkSyncedToGPhotos(ctx, testHash, "remote_id_1", "Samsung_Galaxy_S23")
	if !database.IsSyncedToGPhotos(ctx, testHash) {
		t.Errorf("Expected hash to be marked as synced")
	}
}
