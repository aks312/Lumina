package faces

import (
	"image"
	"math"
)

// FaceRegion represents a detected face bounding box with confidence score.
type FaceRegion struct {
	X          int     `json:"x"`
	Y          int     `json:"y"`
	Width      int     `json:"width"`
	Height     int     `json:"height"`
	Confidence float64 `json:"confidence"`
	SkinHue    float64 `json:"skin_hue"`
}

// DetectFaceRegions scans an image for candidate face bounding boxes using skin-tone geometry analysis.
func DetectFaceRegions(img image.Image) []FaceRegion {
	if img == nil {
		return nil
	}

	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w < 60 || h < 60 {
		return nil
	}

	// Grid sampling: step size of 4 pixels
	step := 4
	gridW := w / step
	gridH := h / step

	skinGrid := make([][]bool, gridH)
	for y := 0; y < gridH; y++ {
		skinGrid[y] = make([]bool, gridW)
		for x := 0; x < gridW; x++ {
			px := b.Min.X + x*step
			py := b.Min.Y + y*step
			r32, g32, b32, _ := img.At(px, py).RGBA()
			r := r32 >> 8
			g := g32 >> 8
			bl := b32 >> 8

			if isSkinRGB(r, g, bl) {
				skinGrid[y][x] = true
			}
		}
	}

	// Connected-component analysis on skinGrid
	visited := make([][]bool, gridH)
	for y := 0; y < gridH; y++ {
		visited[y] = make([]bool, gridW)
	}

	var regions []FaceRegion

	for y := 0; y < gridH; y++ {
		for x := 0; x < gridW; x++ {
			if skinGrid[y][x] && !visited[y][x] {
				minX, maxX := x, x
				minY, maxY := y, y
				pixelCount := 0

				// BFS flood fill
				queue := [][2]int{{x, y}}
				visited[y][x] = true

				for len(queue) > 0 {
					curr := queue[0]
					queue = queue[1:]
					cx, cy := curr[0], curr[1]
					pixelCount++

					if cx < minX {
						minX = cx
					}
					if cx > maxX {
						maxX = cx
					}
					if cy < minY {
						minY = cy
					}
					if cy > maxY {
						maxY = cy
					}

					// 4-neighborhood
					neighbors := [][2]int{
						{cx + 1, cy}, {cx - 1, cy},
						{cx, cy + 1}, {cx, cy - 1},
					}
					for _, nb := range neighbors {
						nx, ny := nb[0], nb[1]
						if nx >= 0 && nx < gridW && ny >= 0 && ny < gridH {
							if skinGrid[ny][nx] && !visited[ny][nx] {
								visited[ny][nx] = true
								queue = append(queue, [2]int{nx, ny})
							}
						}
					}
				}

				// Check aspect ratio and size: human faces typically have aspect ratio 1.0 to 1.8
				boxW := (maxX - minX + 1) * step
				boxH := (maxY - minY + 1) * step

				if boxW >= 30 && boxH >= 35 && pixelCount > 40 {
					aspect := float64(boxH) / float64(boxW)
					if aspect >= 0.8 && aspect <= 2.2 {
						density := float64(pixelCount) / float64((boxW/step)*(boxH/step))
						if density >= 0.25 {
							regions = append(regions, FaceRegion{
								X:          minX * step,
								Y:          minY * step,
								Width:      boxW,
								Height:     boxH,
								Confidence: math.Min(1.0, density+0.3),
							})
						}
					}
				}
			}
		}
	}

	return regions
}

func isSkinRGB(r, g, b uint32) bool {
	return r > 95 && g > 40 && b > 20 &&
		(r > g) && (r > b) &&
		(int(r)-int(g) > 15) &&
		(math.Abs(float64(r)-float64(g)) > 15)
}
