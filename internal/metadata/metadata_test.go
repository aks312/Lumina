package metadata

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestDeviceNormalizer(t *testing.T) {
	tests := []struct {
		makeStr      string
		modelStr     string
		expectedDev  string
		expectedPlat string
	}{
		{"Apple", "iPhone 15 Pro", "Apple_iPhone_15_Pro", "ios"},
		{"Apple", "iPhone14,2", "Apple_iPhone_13_Pro", "ios"},
		{"Samsung", "SM-S911B", "Samsung_Galaxy_S23", "android"},
		{"samsung", "dm1q", "Samsung_Galaxy_S23", "android"},
		{"Google", "Pixel 5 (redfin)", "Google_Pixel_5", "android"},
		{"ASUS", "ASUS_Z01RD", "ASUS_ZenFone_5Z", "android"},
		{"iQOO", "I2019", "iQOO_9_SE", "android"},
		{"Canon", "EOS R5", "Canon_EOS_R5", "camera"},
		{"Sony", "ILCE-7M4", "Sony_ILCE_7M4", "camera"},
		{"DJI", "FC3582", "DJI_FC3582", "camera"},
		{"", "", "Unknown", "unknown"},
		{"Unknown", "Screenshot", "Unknown", "unknown"}, // Rule 3: Never set device_model = "Screenshot"
	}

	for _, tt := range tests {
		dev, plat := NormalizeDevice(tt.makeStr, tt.modelStr)
		if dev != tt.expectedDev || plat != tt.expectedPlat {
			t.Errorf("NormalizeDevice(%q, %q) = (%q, %q); want (%q, %q)",
				tt.makeStr, tt.modelStr, dev, plat, tt.expectedDev, tt.expectedPlat)
		}
	}
}

func TestArtifactRejection(t *testing.T) {
	artifacts := []string{
		"C:/Photos/.thumbnails/img.jpg",
		"C:/Photos/.cache/abc.jpg",
		"C:/Photos/@eaDir/synology_thumb.jpg",
		"C:/Photos/1672538192.jpg",
		"C:/Photos/.hidden_file.jpg",
	}

	for _, a := range artifacts {
		if !ShouldIgnoreFile(a) {
			t.Errorf("Expected ShouldIgnoreFile(%q) to be true", a)
		}
	}

	validPhotos := []string{
		"C:/Photos/IMG_1024.JPG",
		"C:/Photos/2026-03-01_Akash.heic",
		"C:/Photos/DSC_0042.NEF",
	}

	for _, v := range validPhotos {
		if ShouldIgnoreFile(v) {
			t.Errorf("Expected ShouldIgnoreFile(%q) to be false", v)
		}
	}
}

func TestHeicParserWithSyntheticIspe(t *testing.T) {
	buf := new(bytes.Buffer)

	// 1. ftyp box
	ftyp := []byte("ftypheic\x00\x00\x00\x00mif1heic")
	_ = binary.Write(buf, binary.BigEndian, uint32(len(ftyp)+4))
	buf.Write(ftyp)

	// 2. ispe box
	// box size: 4 bytes, type: "ispe", version/flags: 4 bytes, width: 4032, height: 3024
	_ = binary.Write(buf, binary.BigEndian, uint32(20))
	buf.WriteString("ispe")
	buf.Write([]byte{0x00, 0x00, 0x00, 0x00}) // version 0, flags 0
	_ = binary.Write(buf, binary.BigEndian, uint32(4032))
	_ = binary.Write(buf, binary.BigEndian, uint32(3024))

	meta, err := ParseHeicFromBytes(buf.Bytes())
	if err != nil {
		t.Fatalf("ParseHeicFromBytes failed: %v", err)
	}
	if meta.Width != 4032 || meta.Height != 3024 {
		t.Errorf("Expected dimensions 4032x3024, got %dx%d", meta.Width, meta.Height)
	}
}

func TestExifOpticalSensorDetection(t *testing.T) {
	// Build a valid synthetic Little-Endian TIFF with ExposureTime (0x829A) and ISO (0x8827)
	buf := new(bytes.Buffer)
	// II (Little Endian) + 42 + first IFD offset (8)
	buf.Write([]byte{'I', 'I', 0x2A, 0x00, 0x08, 0x00, 0x00, 0x00})

	// IFD entries count = 2
	_ = binary.Write(buf, binary.LittleEndian, uint16(2))

	// Entry 1: ISO (0x8827), SHORT (3), count 1, value 100
	_ = binary.Write(buf, binary.LittleEndian, uint16(0x8827))
	_ = binary.Write(buf, binary.LittleEndian, uint16(3))
	_ = binary.Write(buf, binary.LittleEndian, uint32(1))
	_ = binary.Write(buf, binary.LittleEndian, uint16(100))
	_ = binary.Write(buf, binary.LittleEndian, uint16(0)) // pad to 4 bytes

	// Entry 2: FocalLength (0x920A), RATIONAL (5), count 1, offset 38
	_ = binary.Write(buf, binary.LittleEndian, uint16(0x920A))
	_ = binary.Write(buf, binary.LittleEndian, uint16(5))
	_ = binary.Write(buf, binary.LittleEndian, uint32(1))
	_ = binary.Write(buf, binary.LittleEndian, uint32(38)) // offset to rational

	// Next IFD = 0
	_ = binary.Write(buf, binary.LittleEndian, uint32(0))

	// Offset 38: Rational 240 / 10 = 24mm
	_ = binary.Write(buf, binary.LittleEndian, uint32(240))
	_ = binary.Write(buf, binary.LittleEndian, uint32(10))

	info, err := ParseExifFromBytes(buf.Bytes())
	if err != nil {
		t.Fatalf("ParseExifFromBytes failed: %v", err)
	}

	if !info.HasOpticalSensor {
		t.Errorf("Expected HasOpticalSensor to be true (ISO & FocalLength present)")
	}
	if info.ISO != 100 {
		t.Errorf("Expected ISO 100, got %d", info.ISO)
	}
	if info.FocalLength != 24.0 {
		t.Errorf("Expected FocalLength 24.0, got %.1f", info.FocalLength)
	}
}
