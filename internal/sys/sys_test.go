package sys

import (
	"os"
	"testing"
)

func TestSysGovernanceAndBenchmark(t *testing.T) {
	// 1. Worker count check
	workers := GetOptimalWorkerCount()
	if workers < 2 {
		t.Errorf("Expected at least 2 workers, got %d", workers)
	}

	// 2. Power and priority calls (must not crash)
	PreventSleep()
	AllowSleep()
	SetBackgroundPriority()
	SetNormalPriority()

	// 3. Drive benchmark
	tempDir, err := os.MkdirTemp("", "lumina_sys_bench_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	res := BenchmarkDrive(tempDir)
	if res.Error != "" {
		t.Errorf("Benchmark error: %s", res.Error)
	}
	if res.WriteMBps <= 0 {
		t.Errorf("Expected positive write throughput, got %.2f MB/s", res.WriteMBps)
	}
	if res.ReadMBps <= 0 {
		t.Errorf("Expected positive read throughput, got %.2f MB/s", res.ReadMBps)
	}

	// 4. Drive comparison
	p, m, fastest := CompareDrives(tempDir, "")
	if fastest != tempDir || !p.IsFastest {
		t.Errorf("Expected primary drive to be fastest when no mirror provided: got %s, %+v, %+v", fastest, p, m)
	}

	// 5. Test Drive & Common Folder Detection
	drives := GetDetectedDrives()
	if len(drives) == 0 {
		t.Logf("No drives detected or running non-windows environment")
	} else {
		t.Logf("Detected %d drives, first: %+v", len(drives), drives[0])
	}

	folders := GetCommonMediaFolders()
	t.Logf("Detected %d common media folders", len(folders))
}
