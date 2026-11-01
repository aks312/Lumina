package sys

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// DriveBenchmarkResult stores write and read throughput metrics in MB/s.
type DriveBenchmarkResult struct {
	DrivePath  string  `json:"drive_path"`
	WriteMBps  float64 `json:"write_mbps"`
	ReadMBps   float64 `json:"read_mbps"`
	TotalScore float64 `json:"total_score"` // weighted composite (60% read, 40% write)
	IsFastest  bool    `json:"is_fastest"`
	Error      string  `json:"error,omitempty"`
}

// BenchmarkDrive measures I/O throughput in MB/s by writing and reading a sample block.
// Uses a 16MB buffer to accurately measure NVMe vs HDD performance while finishing in < 100ms.
func BenchmarkDrive(driveDir string) DriveBenchmarkResult {
	res := DriveBenchmarkResult{DrivePath: driveDir}
	if driveDir == "" {
		res.Error = "empty drive path"
		return res
	}

	if err := os.MkdirAll(driveDir, 0755); err != nil {
		res.Error = err.Error()
		return res
	}

	testFile := filepath.Join(driveDir, fmt.Sprintf(".lumina_bench_%d.tmp", time.Now().UnixNano()))
	defer os.Remove(testFile)

	const blockSize = 16 * 1024 * 1024 // 16 MB
	buf := make([]byte, blockSize)
	_, _ = rand.Read(buf[:1024]) // seed header with random bytes

	// 1. Measure Write Throughput
	startWrite := time.Now()
	f, err := os.OpenFile(testFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		res.Error = fmt.Sprintf("write open failed: %v", err)
		return res
	}

	written, err := f.Write(buf)
	if err != nil {
		_ = f.Close()
		res.Error = fmt.Sprintf("write failed: %v", err)
		return res
	}
	_ = f.Sync() // flush OS cache to disk
	_ = f.Close()
	writeDur := time.Since(startWrite)
	if writeDur > 0 {
		res.WriteMBps = (float64(written) / (1024 * 1024)) / writeDur.Seconds()
	}

	// 2. Measure Read Throughput
	startRead := time.Now()
	fRead, err := os.Open(testFile)
	if err != nil {
		res.Error = fmt.Sprintf("read open failed: %v", err)
		return res
	}

	readBuf := make([]byte, blockSize)
	readBytes, err := fRead.Read(readBuf)
	_ = fRead.Close()
	if err != nil && readBytes == 0 {
		res.Error = fmt.Sprintf("read failed: %v", err)
		return res
	}
	readDur := time.Since(startRead)
	if readDur > 0 {
		res.ReadMBps = (float64(readBytes) / (1024 * 1024)) / readDur.Seconds()
	}

	// Calculate weighted score (reading is 70% of viewer priority)
	res.TotalScore = (res.ReadMBps * 0.70) + (res.WriteMBps * 0.30)
	return res
}

// CompareDrives benchmarks Primary and Secondary Mirror drives and returns optimal drive selection.
func CompareDrives(primaryPath string, mirrorPath string) (primary DriveBenchmarkResult, mirror DriveBenchmarkResult, fastestPath string) {
	primary = BenchmarkDrive(primaryPath)

	if mirrorPath != "" && mirrorPath != primaryPath {
		mirror = BenchmarkDrive(mirrorPath)

		if mirror.Error == "" && mirror.TotalScore > primary.TotalScore {
			mirror.IsFastest = true
			primary.IsFastest = false
			fastestPath = mirrorPath
			return primary, mirror, fastestPath
		}
	}

	primary.IsFastest = true
	fastestPath = primaryPath
	return primary, mirror, fastestPath
}
