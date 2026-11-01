package ai

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	_ "golang.org/x/image/webp"

	"lumina/internal/metadata"
	"lumina/internal/sys"
)

// SceneAnalysisResult stores generated tags and scene classification.
type SceneAnalysisResult struct {
	Tags        []string `json:"tags"`
	Description string   `json:"description"`
	OcrText     string   `json:"ocr_text"`
	FaceCount   int      `json:"face_count"`
	IsDocument  bool     `json:"is_document"`
	Sharpness   float64  `json:"sharpness"`
	Luminance   float64  `json:"luminance"`
}

// AnalyzeLocalPhoto performs 100% offline, zero-key visual scene analysis and native Windows OCR.
func AnalyzeLocalPhoto(ctx context.Context, filePath string, exif *metadata.ExifInfo, runOcr bool) (*SceneAnalysisResult, error) {
	res := &SceneAnalysisResult{
		Tags: make([]string, 0, 8),
	}

	// 1. Decode image for visual analysis
	f, err := os.Open(filePath)
	if err == nil {
		img, _, decErr := image.Decode(f)
		_ = f.Close()
		if decErr == nil && img != nil {
			analyzeVisualProperties(img, res)
		}
	}

	// 2. Add EXIF time-of-day tags
	if exif != nil && !exif.DateTimeOriginal.IsZero() {
		hour := exif.DateTimeOriginal.Hour()
		if hour >= 5 && hour < 11 {
			res.Tags = append(res.Tags, "morning")
		} else if hour >= 11 && hour < 17 {
			res.Tags = append(res.Tags, "daylight")
		} else if hour >= 17 && hour < 20 {
			res.Tags = append(res.Tags, "golden_hour", "evening")
		} else {
			res.Tags = append(res.Tags, "night")
		}

		if exif.Make != "" {
			res.Tags = append(res.Tags, strings.ToLower(exif.Make))
		}
	}

	// 3. Native Windows WinRT OCR if requested and file exists
	if runOcr && runtime.GOOS == "windows" {
		ocrText, ocrErr := RunWindowsNativeOcr(ctx, filePath)
		if ocrErr == nil && len(ocrText) > 0 {
			res.OcrText = strings.TrimSpace(ocrText)
			// Deduce document/receipt from OCR content
			lowerOcr := strings.ToLower(res.OcrText)
			if strings.Contains(lowerOcr, "total") || strings.Contains(lowerOcr, "receipt") ||
				strings.Contains(lowerOcr, "invoice") || strings.Contains(lowerOcr, "amount") ||
				strings.Contains(lowerOcr, "$") || strings.Contains(lowerOcr, "tax") {
				res.Tags = append(res.Tags, "receipt", "document")
				res.IsDocument = true
			} else if len(res.OcrText) > 60 {
				res.Tags = append(res.Tags, "document", "text")
				res.IsDocument = true
			}
		}
	}

	// Build summary description
	res.Tags = deduplicateTags(res.Tags)
	res.Description = fmt.Sprintf("Scene with %s", strings.Join(res.Tags, ", "))

	return res, nil
}

// analyzeVisualProperties inspects pixel luminance, color distribution, skin tones, foliage, and sky.
func analyzeVisualProperties(img image.Image, res *SceneAnalysisResult) {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= 0 || h <= 0 {
		return
	}

	// Subsample grid for ultra-fast <5ms analysis
	sampleStepX := max(1, w/60)
	sampleStepY := max(1, h/60)

	var totalLum float64
	var totalR, totalG, totalB float64
	var lumValues []float64

	skinPixels := 0
	greenFoliagePixels := 0
	skyPixels := 0
	totalSamples := 0

	topQuarterY := bounds.Min.Y + (h / 4)

	for y := bounds.Min.Y; y < bounds.Max.Y; y += sampleStepY {
		for x := bounds.Min.X; x < bounds.Max.X; x += sampleStepX {
			r32, g32, b32, _ := img.At(x, y).RGBA()
			r := float64(r32 >> 8)
			g := float64(g32 >> 8)
			b := float64(b32 >> 8)

			lum := 0.299*r + 0.587*g + 0.114*b
			totalLum += lum
			totalR += r
			totalG += g
			totalB += b
			lumValues = append(lumValues, lum)
			totalSamples++

			// 1. Skin-tone detection (standard RGB bounds for human skin)
			if r > 95 && g > 40 && b > 20 && (r-g) > 15 && r > b {
				skinPixels++
			}

			// 2. Foliage / Nature check (green dominance)
			if g > r+15 && g > b+15 && g > 50 {
				greenFoliagePixels++
			}

			// 3. Sky gradient check in top quarter (blue dominance)
			if y < topQuarterY && b > r+20 && b > g && b > 100 {
				skyPixels++
			}
		}
	}

	if totalSamples == 0 {
		return
	}

	avgLum := totalLum / float64(totalSamples)
	res.Luminance = avgLum

	// Calculate sharpness / standard deviation of contrast
	var lumVariance float64
	for _, l := range lumValues {
		diff := l - avgLum
		lumVariance += diff * diff
	}
	sharpness := math.Sqrt(lumVariance / float64(totalSamples))
	res.Sharpness = sharpness

	// Evaluate tags
	if avgLum < 50 {
		res.Tags = append(res.Tags, "night", "low_light")
	} else if avgLum > 195 {
		res.Tags = append(res.Tags, "bright", "sunny")
	}

	// Warm sunset check
	avgR := totalR / float64(totalSamples)
	avgB := totalB / float64(totalSamples)
	if avgR > avgB*1.4 && avgLum > 60 && avgLum < 170 {
		res.Tags = append(res.Tags, "sunset", "warm_tone")
	}

	// Foliage ratio
	foliageRatio := float64(greenFoliagePixels) / float64(totalSamples)
	if foliageRatio > 0.18 {
		res.Tags = append(res.Tags, "nature", "foliage", "outdoors")
	}

	// Sky ratio
	topSamples := totalSamples / 4
	if topSamples > 0 && float64(skyPixels)/float64(topSamples) > 0.30 {
		res.Tags = append(res.Tags, "sky", "outdoor")
	}

	// Skin tone / Portrait check
	skinRatio := float64(skinPixels) / float64(totalSamples)
	if skinRatio >= 0.05 && skinRatio <= 0.25 {
		res.Tags = append(res.Tags, "portrait", "person")
		res.FaceCount = 1
	} else if skinRatio > 0.25 {
		res.Tags = append(res.Tags, "group_photo", "people")
		res.FaceCount = 2
	}

	if sharpness > 60 {
		res.Tags = append(res.Tags, "high_sharpness", "vivid")
	}
}

// RunWindowsNativeOcr invokes Windows' built-in Windows.Media.Ocr.OcrEngine via PowerShell one-liner.
// Requires ZERO API keys, 100% offline, and works out-of-the-box on Windows 10/11.
func RunWindowsNativeOcr(ctx context.Context, filePath string) (string, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", err
	}

	// PowerShell script to invoke Windows.Media.Ocr.OcrEngine
	psScript := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Runtime.WindowsRuntime
$asTaskGeneric = ([System.WindowsRuntimeSystemExtensions].GetMethods() | Where-Object { $_.Name -eq 'AsTask' -and $_.GetParameters().Count -eq 1 -and $_.IsGenericMethod })[0]
Function Await($winRtTask, $resultType) {
    $asTask = $asTaskGeneric.MakeGenericMethod($resultType)
    $netTask = $asTask.Invoke($null, @($winRtTask))
    $netTask.Wait(5000) | Out-Null
    $netTask.Result
}
[Windows.Storage.StorageFile,Windows.Storage,ContentType=WindowsRuntime] | Out-Null
[Windows.Media.Ocr.OcrEngine,Windows.Media.Ocr,ContentType=WindowsRuntime] | Out-Null
[Windows.Graphics.Imaging.BitmapDecoder,Windows.Graphics.Imaging,ContentType=WindowsRuntime] | Out-Null

$file = Await ([Windows.Storage.StorageFile]::GetFileFromPathAsync('%s')) ([Windows.Storage.StorageFile])
$stream = Await ($file.OpenAsync([Windows.Storage.FileAccessMode]::Read)) ([Windows.Storage.Streams.IRandomAccessStream])
$decoder = Await ([Windows.Graphics.Imaging.BitmapDecoder]::CreateAsync($stream)) ([Windows.Graphics.Imaging.BitmapDecoder])
$bitmap = Await ($decoder.GetSoftwareBitmapAsync()) ([Windows.Graphics.Imaging.SoftwareBitmap])
$engine = [Windows.Media.Ocr.OcrEngine]::TryCreateFromUserProfileLanguages()
if (-not $engine) { $engine = [Windows.Media.Ocr.OcrEngine]::TryCreateFromLanguage([Windows.Globalization.Language]::new('en-US')) }
$res = Await ($engine.RecognizeAsync($bitmap)) ([Windows.Media.Ocr.OcrResult])
Write-Output $res.Text
`, strings.ReplaceAll(absPath, "'", "''"))

	cmdCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	// Guarantee no visible console window (Rule 5)
	cmd := sys.SilentCommandContext(cmdCtx, "powershell", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", psScript)
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return "", err
	}

	return strings.TrimSpace(out.String()), nil
}

func deduplicateTags(tags []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, t := range tags {
		clean := strings.ToLower(strings.TrimSpace(t))
		if clean != "" && !seen[clean] {
			seen[clean] = true
			result = append(result, clean)
		}
	}
	return result
}
