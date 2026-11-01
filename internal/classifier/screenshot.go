package classifier

import (
	"image"
	"path/filepath"
	"regexp"
	"strings"

	"lumina/internal/metadata"
)

var (
	reDeterministicScreenshot = regexp.MustCompile(`(?i)(screenshot|screen_shot|screen-shot|screen_capture|screencap|snipping|screenclip)`)
	reIosPngScreenshot        = regexp.MustCompile(`^IMG_\d{4}\.PNG$`)
)

// Known exact viewport resolutions (both portrait and landscape)
var knownScreenResolutions = map[[2]int]bool{
	{1080, 1920}: true, {1920, 1080}: true,
	{1080, 2246}: true, {2246, 1080}: true,
	{1080, 2340}: true, {2340, 1080}: true,
	{1080, 2400}: true, {2400, 1080}: true,
	{1440, 2560}: true, {2560, 1440}: true,
	{1440, 3120}: true, {3120, 1440}: true,
	{1440, 3200}: true, {3200, 1440}: true,
	{1170, 2532}: true, {2532, 1170}: true,
	{1284, 2778}: true, {2778, 1284}: true,
	{1179, 2556}: true, {2556, 1179}: true,
	{1290, 2796}: true, {2796, 1290}: true,
	{828, 1792}:  true, {1792, 828}:  true,
	{1125, 2436}: true, {2436, 1125}: true,
	{750, 1334}:  true, {1334, 750}:  true,
	{1920, 1200}: true, {1200, 1920}: true,
	{2560, 1600}: true, {1600, 2560}: true,
	{3840, 2160}: true, {2160, 3840}: true,
}

// ScreenshotEvaluation contains score and diagnosis.
type ScreenshotEvaluation struct {
	IsScreenshot bool   `json:"is_screenshot"`
	Score        int    `json:"score"`
	Reason       string `json:"reason"`
}

// EvaluateScreenshot scores a photo using the 100-point multi-factor heuristic.
func EvaluateScreenshot(filePath string, width int, height int, exif *metadata.ExifInfo) ScreenshotEvaluation {
	ext := strings.ToLower(filepath.Ext(filePath))
	baseName := filepath.Base(filePath)

	// RULE 2: UNBREAKABLE VIDEO VETO (-1000 pts)
	if IsVideoExtension(ext) {
		return ScreenshotEvaluation{
			IsScreenshot: false,
			Score:        -1000,
			Reason:       "VETO: Video file can never be a screenshot",
		}
	}

	// RULE 2: UNBREAKABLE HEIC / RAW CAMERA VETO (-1000 pts)
	// HEIC/RAW can NEVER be classified as screenshots unless the original filename explicitly contains "screenshot"
	isHeicOrRaw := ext == ".heic" || ext == ".heif" || ext == ".cr2" || ext == ".cr3" || ext == ".nef" || ext == ".arw" || ext == ".dng" || ext == ".rw2" || ext == ".orf"
	if isHeicOrRaw {
		if !reDeterministicScreenshot.MatchString(baseName) {
			return ScreenshotEvaluation{
				IsScreenshot: false,
				Score:        -1000,
				Reason:       "VETO: HEIC/RAW camera file without explicit 'screenshot' in filename",
			}
		}
	}

	// RULE 2: OPTICAL SENSOR EXIF VETO (-1000 pts)
	if exif != nil && exif.HasOpticalSensor {
		return ScreenshotEvaluation{
			IsScreenshot: false,
			Score:        -1000,
			Reason:       "VETO: Physical optical camera sensor EXIF detected (ExposureTime/FNumber/ISO/FocalLength)",
		}
	}

	score := 0
	var reasons []string

	// 1. Deterministic filename check (+100 pts)
	if reDeterministicScreenshot.MatchString(baseName) || reDeterministicScreenshot.MatchString(filePath) {
		score += 100
		reasons = append(reasons, "deterministic_filename_match")
	}

	// iOS PNG naming format: IMG_1234.PNG (+100 pts)
	if ext == ".png" && reIosPngScreenshot.MatchString(baseName) {
		score += 100
		reasons = append(reasons, "ios_png_screenshot_pattern")
	}

	// EXIF Software / OriginalFileName check
	if exif != nil {
		softLower := strings.ToLower(exif.Software)
		if strings.Contains(softLower, "screenshot") || strings.Contains(softLower, "snipping") || strings.Contains(softLower, "screenclip") {
			score += 100
			reasons = append(reasons, "exif_software_screenshot")
		}
	}

	// 2. Viewport Resolution Match (+35 pts)
	if width > 0 && height > 0 {
		key := [2]int{width, height}
		if knownScreenResolutions[key] {
			score += 35
			reasons = append(reasons, "exact_viewport_resolution_match")
		}
	}

	// 3. PNG Format Bias without Camera EXIF (+15 pts)
	if ext == ".png" && (exif == nil || (exif.Make == "" && exif.Model == "")) {
		score += 15
		reasons = append(reasons, "png_without_camera_make")
	}

	// Threshold requirement: >= 75 pts to be classified as screenshot
	isSS := score >= 75

	return ScreenshotEvaluation{
		IsScreenshot: isSS,
		Score:        score,
		Reason:       strings.Join(reasons, "; "),
	}
}

// AnalyzeImageScanlines performs fast sample row/column uniformity & status-bar contrast checks.
// Returns points (+0 to +50) based on digital UI characteristics.
func AnalyzeImageScanlines(img image.Image) int {
	if img == nil {
		return 0
	}
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w < 100 || h < 100 {
		return 0
	}

	points := 0

	// Check top status bar (rows 2 to 10): UI status bars often have high flat-color uniformity
	topRowY := bounds.Min.Y + 5
	firstR, firstG, firstB, _ := img.At(bounds.Min.X+20, topRowY).RGBA()
	flatPixels := 0
	samples := 50
	step := max(1, w/samples)

	for i := 0; i < samples; i++ {
		x := bounds.Min.X + i*step
		r, g, b, _ := img.At(x, topRowY).RGBA()
		diff := absDiff(r, firstR) + absDiff(g, firstG) + absDiff(b, firstB)
		if diff < 1500 { // within ~2% color variation
			flatPixels++
		}
	}

	// If >80% of top status-bar pixels are flat solid color (+25 pts)
	if float64(flatPixels)/float64(samples) > 0.80 {
		points += 25
	}

	// Digital UI high-contrast bimodal edge check:
	// Digital text on screens has sharp transitions between black/white pixels without optical blur
	centerRowY := bounds.Min.Y + (h / 2)
	sharpTransitions := 0
	prevLum := uint32(0)

	for i := 0; i < samples; i++ {
		x := bounds.Min.X + i*step
		r, g, b, _ := img.At(x, centerRowY).RGBA()
		lum := (r*299 + g*587 + b*114) / 1000
		if i > 0 {
			lumDiff := absDiff(lum, prevLum)
			if lumDiff > 30000 { // stark bimodal transition
				sharpTransitions++
			}
		}
		prevLum = lum
	}

	if sharpTransitions >= 8 {
		points += 25
	}

	return points
}

// IsVideoExtension checks if a file extension belongs to video formats.
func IsVideoExtension(ext string) bool {
	switch strings.ToLower(ext) {
	case ".mp4", ".mov", ".mkv", ".avi", ".m4v", ".3gp", ".wmv", ".webm", ".flv", ".ts":
		return true
	default:
		return false
	}
}

// IsImageExtension checks if a file extension is a supported photo format.
func IsImageExtension(ext string) bool {
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg", ".png", ".heic", ".heif", ".webp", ".tiff", ".tif",
		".dng", ".cr2", ".cr3", ".nef", ".arw", ".rw2", ".orf":
		return true
	default:
		return false
	}
}

func absDiff(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
}
