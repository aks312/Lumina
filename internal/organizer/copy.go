package organizer

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CopyProgress reports copy state for large files.
type CopyProgress struct {
	BytesCopied int64
	TotalBytes  int64
	Percentage  float64
}

// WatchdogCopy copies a file from src to dest with an activity watchdog.
// Even for 50GB files, as long as bytes are actively being transferred, it will never time out.
func WatchdogCopy(ctx context.Context, srcPath string, destPath string, onProgress func(p CopyProgress)) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}
	defer srcFile.Close()

	srcStat, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat source: %w", err)
	}
	totalBytes := srcStat.Size()

	// Ensure destination directory exists
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination dir: %w", err)
	}

	// Create temporary partial file to ensure atomic write
	tmpDestPath := destPath + ".lumina_part"
	destFile, err := os.OpenFile(tmpDestPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open destination: %w", err)
	}

	buf := make([]byte, 1024*1024) // 1 MB chunk buffer
	var bytesCopied int64
	lastActivity := time.Now()
	const maxInactivity = 60 * time.Second

	for {
		select {
		case <-ctx.Done():
			_ = destFile.Close()
			_ = os.Remove(tmpDestPath)
			return ctx.Err()
		default:
		}

		// Check watchdog: fail ONLY if zero bytes moved for 60 consecutive seconds
		if time.Since(lastActivity) > maxInactivity {
			_ = destFile.Close()
			_ = os.Remove(tmpDestPath)
			return fmt.Errorf("copy watchdog timeout: disk inactive for >60s")
		}

		n, readErr := srcFile.Read(buf)
		if n > 0 {
			lastActivity = time.Now()
			wN, writeErr := destFile.Write(buf[:n])
			if writeErr != nil {
				_ = destFile.Close()
				_ = os.Remove(tmpDestPath)
				return fmt.Errorf("failed to write destination chunk: %w", writeErr)
			}
			bytesCopied += int64(wN)
			if onProgress != nil && totalBytes > 0 {
				onProgress(CopyProgress{
					BytesCopied: bytesCopied,
					TotalBytes:  totalBytes,
					Percentage:  float64(bytesCopied) / float64(totalBytes) * 100.0,
				})
			}
		}

		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			_ = destFile.Close()
			_ = os.Remove(tmpDestPath)
			return fmt.Errorf("failed to read source chunk: %w", readErr)
		}
	}

	_ = destFile.Sync()
	_ = destFile.Close()

	// Atomically rename .lumina_part to final destination
	if err := os.Rename(tmpDestPath, destPath); err != nil {
		// If rename fails (e.g. cross-volume or existing), try remove + rename
		_ = os.Remove(destPath)
		if err2 := os.Rename(tmpDestPath, destPath); err2 != nil {
			_ = os.Remove(tmpDestPath)
			return fmt.Errorf("failed to rename part file to final dest: %w", err2)
		}
	}

	// Preserve modification time
	_ = os.Chtimes(destPath, srcStat.ModTime(), srcStat.ModTime())

	return nil
}

// ResolveCollision checks if target path exists and generates non-colliding path if needed.
func ResolveCollision(destDir string, fileName string) string {
	target := filepath.Join(destDir, fileName)
	if _, err := os.Stat(target); os.IsNotExist(err) {
		return target
	}

	ext := filepath.Ext(fileName)
	base := strings.TrimSuffix(fileName, ext)

	counter := 1
	for {
		newFileName := fmt.Sprintf("%s_%d%s", base, counter, ext)
		newTarget := filepath.Join(destDir, newFileName)
		if _, err := os.Stat(newTarget); os.IsNotExist(err) {
			return newTarget
		}
		counter++
	}
}
