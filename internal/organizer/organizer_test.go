package organizer

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func Test5TierRoutingHierarchy(t *testing.T) {
	date := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	deviceMappings := map[string]string{
		"Samsung_Galaxy_S23": "Akash",
	}

	// 1. Priority 1: Screenshot Routing
	// Must route to Screenshots/2026/2026-03, ShowInAll=false, RuleApplied=1
	r1 := CalculateRoute(
		true, // isScreenshot
		"Amruta", true, // watched override (must be IGNORED by screenshot!)
		false, "", "", false,
		"Samsung_Galaxy_S23", deviceMappings,
		"android", date,
	)
	if r1.RuleApplied != 1 || r1.TargetFolder != "Screenshots" || r1.ShowInAll != false {
		t.Errorf("Priority 1 Violation: Screenshot must route to Screenshots and be hidden. Got: %+v", r1)
	}

	// 2. Priority 2: Watched Source Folder Override
	// Must route to Amruta/2026/2026-03, RuleApplied=2
	r2 := CalculateRoute(
		false,
		"Amruta", false, // showInAll=false for this watched folder
		false, "", "", false,
		"Samsung_Galaxy_S23", deviceMappings,
		"android", date,
	)
	if r2.RuleApplied != 2 || r2.TargetFolder != "Amruta" || r2.ShowInAll != false {
		t.Errorf("Priority 2 Violation: Watched override failed. Got: %+v", r2)
	}

	// 3. Priority 3: User Manual Override
	// Must route to CustomFolder, RuleApplied=3
	r3 := CalculateRoute(
		false,
		"", true,
		true, "CustomTrip", "custom", true, // isUserModified=true
		"Samsung_Galaxy_S23", deviceMappings,
		"android", date,
	)
	if r3.RuleApplied != 3 || r3.TargetFolder != "CustomTrip" {
		t.Errorf("Priority 3 Violation: User modification override failed. Got: %+v", r3)
	}

	// 4. Priority 4: Device-to-Folder Mapping
	// Must route to Akash/2026/2026-03, RuleApplied=4
	r4 := CalculateRoute(
		false,
		"", true,
		false, "", "", false,
		"Samsung_Galaxy_S23", deviceMappings,
		"android", date,
	)
	if r4.RuleApplied != 4 || r4.TargetFolder != "Akash" {
		t.Errorf("Priority 4 Violation: Device mapping failed. Got: %+v", r4)
	}

	// 5. Priority 5: Platform Fallback
	r5Ios := CalculateRoute(
		false,
		"", true,
		false, "", "", false,
		"Apple_iPhone_15_Pro", nil,
		"ios", date,
	)
	if r5Ios.RuleApplied != 5 || r5Ios.TargetFolder != "iOS" {
		t.Errorf("Priority 5 iOS fallback failed. Got: %+v", r5Ios)
	}

	r5Camera := CalculateRoute(
		false,
		"", true,
		false, "", "", false,
		"Sony_Alpha", nil,
		"camera", date,
	)
	if r5Camera.RuleApplied != 5 || r5Camera.TargetFolder != "Camera" {
		t.Errorf("Priority 5 Camera fallback failed. Got: %+v", r5Camera)
	}
}

func TestOrganizedFileNameAndWatchdogCopy(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lumina_organizer_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	date := time.Date(2026, 3, 1, 14, 30, 0, 0, time.UTC)
	fileName := GenerateOrganizedFileName(date, "Apple_iPhone_15_Pro", "abcdef123456", ".jpg")
	expectedName := "20260301_143000_Apple_iPhone_15_Pro_abcdef12.jpg"
	if fileName != expectedName {
		t.Errorf("GenerateOrganizedFileName mismatch: got %s, want %s", fileName, expectedName)
	}

	// Test WatchdogCopy
	srcFile := filepath.Join(tempDir, "source.txt")
	destFile := filepath.Join(tempDir, "sub", "dest.txt")
	testContent := []byte("Hello Lumina Photos V3 Organizer")
	if err := os.WriteFile(srcFile, testContent, 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	progressReported := false
	err = WatchdogCopy(context.Background(), srcFile, destFile, func(p CopyProgress) {
		progressReported = true
	})
	if err != nil {
		t.Fatalf("WatchdogCopy failed: %v", err)
	}
	if !progressReported {
		t.Errorf("Expected copy progress to be reported")
	}

	readBack, err := os.ReadFile(destFile)
	if err != nil || string(readBack) != string(testContent) {
		t.Errorf("Copy content mismatch")
	}

	// Test Collision Resolver
	resolved := ResolveCollision(filepath.Dir(destFile), "dest.txt")
	if filepath.Base(resolved) != "dest_1.txt" {
		t.Errorf("Expected dest_1.txt, got %s", filepath.Base(resolved))
	}
}
