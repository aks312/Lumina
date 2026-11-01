package classifier

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"lumina/internal/metadata"
)

func TestScreenshotVetoes(t *testing.T) {
	// 1. Video file Veto (Rule 2)
	videoEval := EvaluateScreenshot("C:/Videos/vacation.mp4", 1920, 1080, nil)
	if videoEval.IsScreenshot || videoEval.Score != -1000 {
		t.Errorf("Rule 2 Violation: Videos must be vetoed (-1000). Got: %+v", videoEval)
	}

	// 2. HEIC file without "screenshot" Veto (Rule 2)
	heicEval := EvaluateScreenshot("C:/Photos/IMG_5678.HEIC", 4032, 3024, nil)
	if heicEval.IsScreenshot || heicEval.Score != -1000 {
		t.Errorf("Rule 2 Violation: HEIC camera photo must be vetoed (-1000). Got: %+v", heicEval)
	}

	// 3. Optical sensor EXIF Veto (Rule 2)
	exifWithOptics := &metadata.ExifInfo{
		Make:             "Apple",
		Model:            "iPhone 15 Pro",
		HasOpticalSensor: true,
		ExposureTime:     "1/120",
		FNumber:          1.78,
		ISO:              80,
	}
	opticalEval := EvaluateScreenshot("C:/Photos/IMG_1000.JPG", 1179, 2556, exifWithOptics)
	if opticalEval.IsScreenshot || opticalEval.Score != -1000 {
		t.Errorf("Rule 2 Violation: Photo with optical EXIF must be vetoed (-1000). Got: %+v", opticalEval)
	}

	// 4. Deterministic Screenshot Match (+100 pts)
	ssEval := EvaluateScreenshot("C:/Captures/Screenshot_2026-03-01.png", 1080, 2400, nil)
	if !ssEval.IsScreenshot || ssEval.Score < 100 {
		t.Errorf("Expected deterministic screenshot to score >= 100. Got: %+v", ssEval)
	}

	// 5. iOS PNG Pattern Match (+100 pts)
	iosPngEval := EvaluateScreenshot("C:/DCIM/IMG_0123.PNG", 1170, 2532, nil)
	if !iosPngEval.IsScreenshot || iosPngEval.Score < 100 {
		t.Errorf("Expected iOS PNG screenshot to qualify. Got: %+v", iosPngEval)
	}
}

func TestAnalyzeImageScanlines(t *testing.T) {
	// Create synthetic image with flat status bar
	img := image.NewRGBA(image.Rect(0, 0, 300, 600))
	// Flat top bar (e.g. solid black status bar)
	for y := 0; y < 30; y++ {
		for x := 0; x < 300; x++ {
			img.Set(x, y, color.Black)
		}
	}
	// Alternate high-contrast bimodal center
	for x := 0; x < 300; x++ {
		if (x/10)%2 == 0 {
			img.Set(x, 300, color.White)
		} else {
			img.Set(x, 300, color.Black)
		}
	}

	points := AnalyzeImageScanlines(img)
	if points < 25 {
		t.Errorf("Expected >= 25 points from flat status bar and bimodal edges, got %d", points)
	}
}

func TestSparseHashingAndLivePhotoPairing(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lumina_classifier_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create 3MB dummy file to trigger sparse 3-point hashing
	largeFile := filepath.Join(tempDir, "sample_video.mov")
	f, err := os.Create(largeFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	buf := make([]byte, 1024*1024)
	for i := 0; i < 3; i++ {
		_, _ = f.Write(buf)
	}
	_ = f.Close()

	hash, size, err := ComputeFastHash(largeFile)
	if err != nil || hash == "" || size != 3*1024*1024 {
		t.Errorf("ComputeFastHash failed: hash=%s, size=%d, err=%v", hash, size, err)
	}

	// Test Apple Live Photo pairing
	heicPath := filepath.Join(tempDir, "IMG_9999.HEIC")
	movPath := filepath.Join(tempDir, "IMG_9999.MOV")
	_ = os.WriteFile(heicPath, []byte("dummy heic"), 0644)
	_ = os.WriteFile(movPath, []byte("dummy mov"), 0644)

	pairs := DetectLivePhotoPairs([]string{heicPath, movPath, largeFile})
	if pairs[heicPath] != movPath || pairs[movPath] != heicPath {
		t.Errorf("DetectLivePhotoPairs failed to pair Live Photo: %+v", pairs)
	}
}
