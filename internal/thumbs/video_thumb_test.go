package thumbs

import (
	"bytes"
	"image"
	"testing"
)

func TestGenerateVideoThumbnailGuaranteed(t *testing.T) {
	fakeVideoPath := "C:/Users/User/Videos/test_clip.mp4"
	thumbBytes, err := GenerateThumbnail(fakeVideoPath, 400)
	if err != nil {
		t.Fatalf("GenerateThumbnail for video returned error: %v", err)
	}

	if len(thumbBytes) < 1000 {
		t.Fatalf("Expected video thumbnail to be at least 1000 bytes, got %d", len(thumbBytes))
	}

	// Verify it's a valid JPEG
	if thumbBytes[0] != 0xFF || thumbBytes[1] != 0xD8 {
		t.Fatalf("Video thumbnail does not start with JPEG SOI marker: %x %x", thumbBytes[0], thumbBytes[1])
	}

	// Verify decoding
	img, format, err := image.Decode(bytes.NewReader(thumbBytes))
	if err != nil {
		t.Fatalf("Failed to decode generated video thumbnail: %v", err)
	}

	if format != "jpeg" {
		t.Fatalf("Expected jpeg format, got %s", format)
	}

	bounds := img.Bounds()
	if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		t.Fatalf("Invalid image bounds: %v", bounds)
	}
}

func TestStylizedVideoPoster(t *testing.T) {
	poster := GenerateStylizedVideoPoster("my_clip.mov", 400)
	if len(poster) < 1000 {
		t.Fatalf("Generated poster too small: %d bytes", len(poster))
	}

	img, _, err := image.Decode(bytes.NewReader(poster))
	if err != nil {
		t.Fatalf("Failed to decode poster: %v", err)
	}

	if img.Bounds().Dx() != 400 || img.Bounds().Dy() != 300 {
		t.Fatalf("Unexpected poster bounds: %v", img.Bounds())
	}
}
