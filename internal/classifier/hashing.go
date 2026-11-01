package classifier

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	sparseThreshold = 2 * 1024 * 1024 // 2 MB
	sparseChunkSize = 65536           // 64 KB
)

// ComputeFastHash calculates SHA-256 for small files, and sparse 3-point SHA-256 for large files/videos.
// Allows indexing 50GB 4K videos in < 10ms without reading gigabytes of data.
func ComputeFastHash(filePath string) (string, int64, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return "", 0, err
	}
	fileSize := info.Size()

	f, err := os.Open(filePath)
	if err != nil {
		return "", fileSize, err
	}
	defer f.Close()

	hasher := sha256.New()

	if fileSize <= sparseThreshold {
		// Full hash for smaller images
		if _, err := io.Copy(hasher, f); err != nil {
			return "", fileSize, err
		}
	} else {
		// Sparse 3-point hash: First 64KB + Middle 64KB + Last 64KB + FileSize
		// 1. First 64KB
		firstBuf := make([]byte, sparseChunkSize)
		n1, _ := io.ReadFull(f, firstBuf)
		hasher.Write(firstBuf[:n1])

		// 2. Middle 64KB
		midOffset := (fileSize / 2) - (sparseChunkSize / 2)
		if midOffset > 0 {
			if _, err := f.Seek(midOffset, io.SeekStart); err == nil {
				midBuf := make([]byte, sparseChunkSize)
				n2, _ := io.ReadFull(f, midBuf)
				hasher.Write(midBuf[:n2])
			}
		}

		// 3. Last 64KB
		endOffset := fileSize - sparseChunkSize
		if endOffset > 0 {
			if _, err := f.Seek(endOffset, io.SeekStart); err == nil {
				endBuf := make([]byte, sparseChunkSize)
				n3, _ := io.ReadFull(f, endBuf)
				hasher.Write(endBuf[:n3])
			}
		}

		// Append file size into hash salt
		hasher.Write([]byte(fmt.Sprintf(":size=%d", fileSize)))
	}

	return hex.EncodeToString(hasher.Sum(nil)), fileSize, nil
}

// LivePhotoPair links an Apple HEIC image with its companion .MOV video.
type LivePhotoPair struct {
	PhotoPath string `json:"photo_path"`
	VideoPath string `json:"video_path"`
	BaseName  string `json:"base_name"`
}

// DetectLivePhotoPairs scans a list of file paths in a directory and pairs Apple Live Photos (.HEIC + .MOV).
func DetectLivePhotoPairs(filePaths []string) map[string]string {
	// Maps photoPath -> paired videoPath and videoPath -> paired photoPath
	pairs := make(map[string]string)

	byBase := make(map[string]map[string]string) // dir+basename -> ext -> fullPath

	for _, p := range filePaths {
		dir := filepath.Dir(p)
		ext := strings.ToLower(filepath.Ext(p))
		baseWithoutExt := strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))
		key := filepath.Join(dir, strings.ToLower(baseWithoutExt))

		if byBase[key] == nil {
			byBase[key] = make(map[string]string)
		}
		byBase[key][ext] = p
	}

	for _, exts := range byBase {
		heicPath, hasHeic := exts[".heic"]
		movPath, hasMov := exts[".mov"]
		if hasHeic && hasMov {
			pairs[heicPath] = movPath
			pairs[movPath] = heicPath
		}
	}

	return pairs
}
