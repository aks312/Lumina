package metadata

import (
	"path/filepath"
	"regexp"
	"strings"
)

// Artifact rejection patterns: reject source thumbnails, cache, Synology @eaDir, and 10-digit temp files
var (
	reThumbnailArtifact = regexp.MustCompile(`(?i)(\.thumbnails|\.cache|@eaDir|\.trash|\.synology|thumb[_-]|preview[_-])`)
	reTenDigitJpg       = regexp.MustCompile(`^\d{10}\.jpg$`)
	reHiddenFile        = regexp.MustCompile(`^\.[^.]+`)
)

// ShouldIgnoreFile checks if a file is an unwanted system/cache artifact.
func ShouldIgnoreFile(filePath string) bool {
	base := filepath.Base(filePath)
	if base == "" || strings.HasPrefix(base, ".") {
		return true
	}

	// Reject 10-digit temp thumbnails like 1672538192.jpg
	if reTenDigitJpg.MatchString(base) {
		return true
	}

	// Reject thumbs.pack or lumina databases
	if strings.HasPrefix(base, "thumbs.pack") || strings.HasPrefix(base, "lumina_v3.db") {
		return true
	}

	// Check path components for .cache, .thumbnails, @eaDir
	normalized := filepath.ToSlash(filePath)
	parts := strings.Split(normalized, "/")
	for _, p := range parts {
		if reThumbnailArtifact.MatchString(p) {
			return true
		}
	}

	return false
}

// NormalizeDevice converts raw hardware IDs and messy EXIF strings into clean canonical names.
func NormalizeDevice(makeStr string, modelStr string) (canonical string, platform string) {
	makeClean := strings.TrimSpace(makeStr)
	modelClean := strings.TrimSpace(modelStr)

	// Never allow device_model = "Screenshot" (Rule 3)
	if strings.EqualFold(modelClean, "screenshot") {
		modelClean = ""
	}

	combined := strings.TrimSpace(makeClean + " " + modelClean)
	combinedLower := strings.ToLower(combined)

	// If no make or model at all, return Unknown
	if combined == "" {
		return "Unknown", "unknown"
	}

	// 1. Apple iOS Family
	if strings.Contains(combinedLower, "apple") || strings.Contains(combinedLower, "iphone") || strings.Contains(combinedLower, "ipad") {
		return normalizeAppleDevice(combined), "ios"
	}

	// 2. Android Hardware Mapping
	if isAndroidFamily(combinedLower) {
		return normalizeAndroidDevice(makeClean, modelClean, combined), "android"
	}

	// 3. Dedicated Camera Manufacturers
	if isCameraFamily(combinedLower) {
		return normalizeCameraDevice(makeClean, modelClean), "camera"
	}

	// Fallback: clean up whatever make/model was provided
	cleaned := cleanIdentifier(combined)
	if cleaned == "" {
		return "Unknown", "unknown"
	}
	return cleaned, "unknown"
}

func isAndroidFamily(str string) bool {
	androidKeywords := []string{
		"samsung", "google", "pixel", "xiaomi", "redmi", "poco", "oneplus",
		"oppo", "vivo", "iqoo", "realme", "asus", "motorola", "moto",
		"sony xperia", "huawei", "honor", "nothing", "sm-s", "sm-g", "sm-a",
		"sm-f", "sm-n", "cph", "v2", "m2", "2201", "2304", "in20", "le21",
	}
	for _, kw := range androidKeywords {
		if strings.Contains(str, kw) {
			return true
		}
	}
	return false
}

func isCameraFamily(str string) bool {
	cameraKeywords := []string{
		"canon", "nikon", "sony", "fujifilm", "fuji", "panasonic",
		"lumix", "olympus", "om digital", "leica", "hasselblad",
		"pentax", "ricoh", "gopro", "dji", "phase one", "sigma",
	}
	for _, kw := range cameraKeywords {
		if strings.Contains(str, kw) {
			// Sony phones (Xperia) are Android, cameras are Camera
			if kw == "sony" && strings.Contains(str, "xperia") {
				return false
			}
			return true
		}
	}
	return false
}

func normalizeAppleDevice(combined string) string {
	mapping := map[string]string{
		"iphone10,3": "Apple_iPhone_X",
		"iphone10,6": "Apple_iPhone_X",
		"iphone11,2": "Apple_iPhone_XS",
		"iphone11,4": "Apple_iPhone_XS_Max",
		"iphone11,6": "Apple_iPhone_XS_Max",
		"iphone11,8": "Apple_iPhone_XR",
		"iphone12,1": "Apple_iPhone_11",
		"iphone12,3": "Apple_iPhone_11_Pro",
		"iphone12,5": "Apple_iPhone_11_Pro_Max",
		"iphone12,8": "Apple_iPhone_SE_2nd_Gen",
		"iphone13,1": "Apple_iPhone_12_mini",
		"iphone13,2": "Apple_iPhone_12",
		"iphone13,3": "Apple_iPhone_12_Pro",
		"iphone13,4": "Apple_iPhone_12_Pro_Max",
		"iphone14,2": "Apple_iPhone_13_Pro",
		"iphone14,3": "Apple_iPhone_13_Pro_Max",
		"iphone14,4": "Apple_iPhone_13_mini",
		"iphone14,5": "Apple_iPhone_13",
		"iphone14,6": "Apple_iPhone_SE_3rd_Gen",
		"iphone14,7": "Apple_iPhone_14",
		"iphone14,8": "Apple_iPhone_14_Plus",
		"iphone15,2": "Apple_iPhone_14_Pro",
		"iphone15,3": "Apple_iPhone_14_Pro_Max",
		"iphone15,4": "Apple_iPhone_15",
		"iphone15,5": "Apple_iPhone_15_Plus",
		"iphone16,1": "Apple_iPhone_15_Pro",
		"iphone16,2": "Apple_iPhone_15_Pro_Max",
		"iphone17,1": "Apple_iPhone_16_Pro",
		"iphone17,2": "Apple_iPhone_16_Pro_Max",
		"iphone17,3": "Apple_iPhone_16",
		"iphone17,4": "Apple_iPhone_16_Plus",
	}

	lower := strings.ToLower(combined)
	for code, name := range mapping {
		if strings.Contains(lower, code) {
			return name
		}
	}

	// Clean standard iPhone/iPad names
	clean := cleanIdentifier(combined)
	if !strings.HasPrefix(strings.ToLower(clean), "apple") {
		clean = "Apple_" + clean
	}
	return clean
}

func normalizeAndroidDevice(makeStr string, modelStr string, combined string) string {
	lower := strings.ToLower(combined)

	// Samsung normalization
	if strings.Contains(lower, "samsung") || strings.Contains(lower, "galaxy") || strings.Contains(lower, "sm-") {
		if strings.Contains(lower, "sm-s911") || strings.Contains(lower, "dm1q") || strings.Contains(lower, "s23") && !strings.Contains(lower, "ultra") && !strings.Contains(lower, "+") {
			return "Samsung_Galaxy_S23"
		}
		if strings.Contains(lower, "sm-s916") || strings.Contains(lower, "s23+") || strings.Contains(lower, "s23 plus") {
			return "Samsung_Galaxy_S23_Plus"
		}
		if strings.Contains(lower, "sm-s918") || strings.Contains(lower, "dm3q") || strings.Contains(lower, "s23 ultra") {
			return "Samsung_Galaxy_S23_Ultra"
		}
		if strings.Contains(lower, "sm-s921") || strings.Contains(lower, "s24") && !strings.Contains(lower, "ultra") {
			return "Samsung_Galaxy_S24"
		}
		if strings.Contains(lower, "sm-s928") || strings.Contains(lower, "s24 ultra") {
			return "Samsung_Galaxy_S24_Ultra"
		}
		if strings.Contains(lower, "sm-g998") || strings.Contains(lower, "s21 ultra") {
			return "Samsung_Galaxy_S21_Ultra"
		}
	}

	// Asus ZenFone
	if strings.Contains(lower, "z01rd") || strings.Contains(lower, "zenfone 5z") {
		return "ASUS_ZenFone_5Z"
	}

	// Google Pixel
	if strings.Contains(lower, "redfin") || strings.Contains(lower, "pixel 5") {
		return "Google_Pixel_5"
	}
	if strings.Contains(lower, "pixel 6 pro") {
		return "Google_Pixel_6_Pro"
	}
	if strings.Contains(lower, "pixel 7 pro") {
		return "Google_Pixel_7_Pro"
	}
	if strings.Contains(lower, "pixel 8 pro") {
		return "Google_Pixel_8_Pro"
	}
	if strings.Contains(lower, "pixel 9 pro") {
		return "Google_Pixel_9_Pro"
	}

	// iQOO
	if strings.Contains(lower, "i2019") || strings.Contains(lower, "iqoo 9 se") {
		return "iQOO_9_SE"
	}

	// Format make and model
	mk := cleanIdentifier(makeStr)
	md := cleanIdentifier(modelStr)
	if mk == "" {
		mk = "Android"
	}
	if md == "" {
		return mk
	}
	if strings.HasPrefix(strings.ToLower(md), strings.ToLower(mk)) {
		return md
	}
	return mk + "_" + md
}

func normalizeCameraDevice(makeStr string, modelStr string) string {
	mk := cleanIdentifier(makeStr)
	md := cleanIdentifier(modelStr)
	if strings.HasPrefix(strings.ToLower(md), strings.ToLower(mk)) {
		return md
	}
	if mk == "" {
		return md
	}
	if md == "" {
		return mk
	}
	return mk + "_" + md
}

func cleanIdentifier(s string) string {
	// Replace spaces, slashes, colons, hyphens with underscore
	re := regexp.MustCompile(`[^\w]+`)
	res := re.ReplaceAllString(s, "_")
	return strings.Trim(res, "_")
}
