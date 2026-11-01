package thumbs

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	xdraw "golang.org/x/image/draw"

	"lumina/internal/sys"
)

var (
	ffmpegOnce     sync.Once
	ffmpegPath     string
	hasFfmpegChecked bool
)

func getFFmpegPath() string {
	ffmpegOnce.Do(func() {
		if p, err := exec.LookPath("ffmpeg"); err == nil {
			ffmpegPath = p
		}
		hasFfmpegChecked = true
	})
	return ffmpegPath
}

// GenerateVideoThumbnail extracts or synthesizes a visual thumbnail for any video.
// Rule 5 Compliant: NEVER invokes powershell.exe or any visible subprocess.
// 1. Pure-Go sidecar (.thm / .jpg) & embedded MP4/MOV cover atom scanner (0 processes).
// 2. Silent FFmpeg frame extractor (only if installed on PATH, with CREATE_NO_WINDOW).
// 3. Guaranteed Pure-Go stylized cinema poster fallback (<1ms, 0 processes).
func GenerateVideoThumbnail(filePath string, maxDim int) ([]byte, error) {
	if maxDim <= 0 {
		maxDim = 400
	}

	// Stage 1a: Check for camera sidecar thumbnail file (.thm, .jpg)
	if sidecarData, ok := findSidecarThumbnail(filePath, maxDim); ok {
		return sidecarData, nil
	}

	// Stage 1b: Scan first 2MB and last 2MB for embedded MP4/MOV JPEG preview atoms
	if frameBytes, err := extractEmbeddedVideoCover(filePath, maxDim); err == nil && len(frameBytes) > 1000 {
		return frameBytes, nil
	}

	// Stage 2: If ffmpeg is available in PATH, use silent command with CREATE_NO_WINDOW
	if getFFmpegPath() != "" {
		if frameBytes, err := extractFFmpegVideoFrame(filePath, maxDim); err == nil && len(frameBytes) > 1000 {
			return frameBytes, nil
		}
	}

	// Stage 3: Guaranteed 100% Pure-Go Cinema Glass Poster (<1ms, ZERO external processes)
	return GenerateStylizedVideoPoster(filePath, maxDim), nil
}

// findSidecarThumbnail checks for companion files like VID_0001.THM or VID_0001.JPG created by cameras.
func findSidecarThumbnail(videoPath string, maxDim int) ([]byte, bool) {
	dir := filepath.Dir(videoPath)
	ext := filepath.Ext(videoPath)
	base := strings.TrimSuffix(filepath.Base(videoPath), ext)

	candidateExts := []string{".thm", ".THM", ".jpg", ".JPG", ".jpeg", ".JPEG"}
	for _, cExt := range candidateExts {
		sidecarPath := filepath.Join(dir, base+cExt)
		if info, err := os.Stat(sidecarPath); err == nil && !info.IsDir() && info.Size() > 500 && info.Size() < 50*1024*1024 {
			f, err := os.Open(sidecarPath)
			if err == nil {
				img, _, err := image.Decode(f)
				_ = f.Close()
				if err == nil && img != nil {
					return scaleAndEncodeJpeg(img, maxDim)
				}
			}
		}
	}
	return nil, false
}

// extractEmbeddedVideoCover scans the first 2MB and last 2MB of an MP4/MOV container
// for embedded JPEG preview markers (covr / meta / udta / pict atoms).
func extractEmbeddedVideoCover(filePath string, maxDim int) ([]byte, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	fileSize := stat.Size()
	chunkSize := int64(2 * 1024 * 1024) // 2 MB

	// 1. Scan first 2MB of video header
	headerLen := chunkSize
	if headerLen > fileSize {
		headerLen = fileSize
	}
	headBuf := make([]byte, headerLen)
	if _, err := io.ReadFull(f, headBuf); err == nil || err == io.ErrUnexpectedEOF {
		if thumb, ok := searchJpegInBytes(headBuf, maxDim); ok {
			return thumb, nil
		}
	}

	// 2. Scan last 2MB (where moov/udta/meta atoms frequently reside in optimized MP4s)
	if fileSize > chunkSize*2 {
		tailLen := chunkSize
		tailBuf := make([]byte, tailLen)
		if _, err := f.Seek(fileSize-tailLen, io.SeekStart); err == nil {
			if _, err := io.ReadFull(f, tailBuf); err == nil || err == io.ErrUnexpectedEOF {
				if thumb, ok := searchJpegInBytes(tailBuf, maxDim); ok {
					return thumb, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("no embedded cover atom found")
}

// searchJpegInBytes finds JPEG SOI/EOI markers within a buffer and decodes them.
func searchJpegInBytes(data []byte, maxDim int) ([]byte, bool) {
	soiIdx := 0
	for {
		idx := bytes.Index(data[soiIdx:], []byte{0xFF, 0xD8, 0xFF})
		if idx == -1 {
			break
		}
		actualSOI := soiIdx + idx
		if actualSOI+500 >= len(data) {
			break
		}

		// Look for EOI marker
		eoiIdx := bytes.Index(data[actualSOI+300:], []byte{0xFF, 0xD9})
		if eoiIdx != -1 {
			fullLen := 300 + eoiIdx + 2
			jpegCandidate := data[actualSOI : actualSOI+fullLen]

			if img, _, err := image.Decode(bytes.NewReader(jpegCandidate)); err == nil && img != nil {
				if scaled, ok := scaleAndEncodeJpeg(img, maxDim); ok {
					return scaled, true
				}
			}
		}

		soiIdx = actualSOI + 3
	}
	return nil, false
}

// extractFFmpegVideoFrame extracts a single frame using ffmpeg with sys.SilentCommandContext (CREATE_NO_WINDOW).
func extractFFmpegVideoFrame(filePath string, maxDim int) ([]byte, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, err
	}

	tmpJpg := filepath.Join(os.TempDir(), fmt.Sprintf("lumina_vthumb_%d.jpg", time.Now().UnixNano()))
	defer os.Remove(tmpJpg)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Guarantee NO console window is spawned (Rule 5)
	cmd := sys.SilentCommandContext(
		ctx,
		"ffmpeg",
		"-ss", "00:00:01",
		"-i", absPath,
		"-vframes", "1",
		"-q:v", "3",
		"-vf", fmt.Sprintf("scale=%d:-1", maxDim),
		"-y", tmpJpg,
	)

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(tmpJpg)
	if err != nil || len(data) < 1000 {
		return nil, fmt.Errorf("failed to read ffmpeg frame output")
	}

	return data, nil
}

func scaleAndEncodeJpeg(img image.Image, maxDim int) ([]byte, bool) {
	bounds := img.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()
	if srcW <= 0 || srcH <= 0 {
		return nil, false
	}

	dstW, dstH := scaleProportional(srcW, srcH, maxDim)
	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	xdraw.BiLinear.Scale(dst, dst.Bounds(), img, bounds, xdraw.Over, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 82}); err == nil && buf.Len() > 0 {
		return buf.Bytes(), true
	}
	return nil, false
}

// GenerateStylizedVideoPoster creates a high-contrast dark cinema frame poster with a play button graphic.
// 100% Pure Go (<1ms, ZERO external processes, ZERO terminal windows, ZERO 404s).
func GenerateStylizedVideoPoster(filePath string, maxDim int) []byte {
	w, h := 400, 300
	if maxDim > 0 && maxDim < 400 {
		w = maxDim
		h = int(float64(maxDim) * 0.75)
		if h < 100 {
			h = 100
		}
	}

	dst := image.NewRGBA(image.Rect(0, 0, w, h))

	// 1. Draw rich slate-blue gradient background
	for y := 0; y < h; y++ {
		progress := float64(y) / float64(h)
		r := uint8(15 + progress*15)
		g := uint8(23 + progress*20)
		b := uint8(42 + progress*25)
		for x := 0; x < w; x++ {
			dst.Set(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}

	// 2. Draw subtle cinema film-strip sprocket holes along top and bottom
	sprocketColor := color.RGBA{R: 255, G: 255, B: 255, A: 35}
	for x := 15; x < w-20; x += 30 {
		for sy := 6; sy < 16; sy++ {
			for sx := 0; sx < 14; sx++ {
				dst.Set(x+sx, sy, sprocketColor)
				dst.Set(x+sx, h-16+sy-6, sprocketColor)
			}
		}
	}

	// 3. Draw glowing circular badge in the center
	centerX, centerY := w/2, h/2
	circleRadius := 38
	circleRadiusSq := circleRadius * circleRadius
	glowColor := color.RGBA{R: 56, G: 189, B: 248, A: 220}   // Sky 400
	innerCircle := color.RGBA{R: 12, G: 74, B: 110, A: 240} // Sky 950

	for dy := -circleRadius; dy <= circleRadius; dy++ {
		for dx := -circleRadius; dx <= circleRadius; dx++ {
			distSq := dx*dx + dy*dy
			if distSq <= circleRadiusSq {
				px := centerX + dx
				py := centerY + dy
				if distSq >= circleRadiusSq-100 {
					dst.Set(px, py, glowColor)
				} else {
					dst.Set(px, py, innerCircle)
				}
			}
		}
	}

	// 4. Draw crisp white play triangle inside the center circle
	triColor := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	triLeft := centerX - 8
	triRight := centerX + 15
	triTop := centerY - 14
	triBottom := centerY + 14

	for py := triTop; py <= triBottom; py++ {
		progress := float64(py-triTop) / float64(triBottom-triTop)
		var spanRight int
		if progress <= 0.5 {
			spanRight = triLeft + int(float64(triRight-triLeft)*(progress*2.0))
		} else {
			spanRight = triLeft + int(float64(triRight-triLeft)*((1.0-progress)*2.0))
		}
		for px := triLeft; px <= spanRight; px++ {
			dst.Set(px, py, triColor)
		}
	}

	// 5. Draw "VIDEO" tag bar in bottom left
	ext := strings.ToUpper(strings.TrimPrefix(filepath.Ext(filePath), "."))
	if ext == "" {
		ext = "VIDEO"
	}
	badgeBg := color.RGBA{R: 2, G: 132, B: 199, A: 230} // Sky 600
	for by := h - 40; by < h-22; by++ {
		for bx := 18; bx < 85; bx++ {
			dst.Set(bx, by, badgeBg)
		}
	}

	buf := new(bytes.Buffer)
	_ = jpeg.Encode(buf, dst, &jpeg.Options{Quality: 85})
	return buf.Bytes()
}
