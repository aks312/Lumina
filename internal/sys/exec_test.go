package sys

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestRule5ZeroRawExecCommands scans all Go files in the codebase and fails if raw
// exec.Command or exec.CommandContext is called outside of internal/sys/exec_*.go.
func TestRule5ZeroRawExecCommands(t *testing.T) {
	// Locate project root by looking for go.mod
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current dir: %v", err)
	}

	rootDir := ""
	curr := dir
	for i := 0; i < 5; i++ {
		if _, err := os.Stat(filepath.Join(curr, "go.mod")); err == nil {
			rootDir = curr
			break
		}
		curr = filepath.Dir(curr)
	}

	if rootDir == "" {
		t.Fatalf("Could not locate project root containing go.mod")
	}

	var violations []string

	err = filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "node_modules" || name == "dist" || name == "build" {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(d.Name(), ".go") {
			return nil
		}

		baseName := d.Name()
		if baseName == "exec_windows.go" || baseName == "exec_other.go" || baseName == "exec_test.go" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		str := string(content)
		if strings.Contains(str, "exec.Command(") || strings.Contains(str, "exec.CommandContext(") {
			rel, _ := filepath.Rel(rootDir, path)
			violations = append(violations, rel)
		}

		return nil
	})

	if err != nil {
		t.Fatalf("Error walking files: %v", err)
	}

	if len(violations) > 0 {
		t.Fatalf("Rule 5 Violation: Found raw exec.Command in %v. Must use sys.SilentCommand/SilentCommandContext instead!", violations)
	}
}

// TestSilentCommandFlags verifies that SilentCommand sets HideWindow and CREATE_NO_WINDOW on Windows.
func TestSilentCommandFlags(t *testing.T) {
	cmd := SilentCommand("cmd.exe", "/c", "echo", "lumina")
	if cmd == nil {
		t.Fatalf("SilentCommand returned nil cmd")
	}

	if runtime.GOOS == "windows" {
		if cmd.SysProcAttr == nil {
			t.Fatalf("Expected cmd.SysProcAttr to be configured on Windows")
		}
		if !cmd.SysProcAttr.HideWindow {
			t.Errorf("Expected HideWindow to be true")
		}
		if cmd.SysProcAttr.CreationFlags&CREATE_NO_WINDOW == 0 {
			t.Errorf("Expected CREATE_NO_WINDOW (0x08000000) flag to be set")
		}
	}

	ctx := context.Background()
	ctxCmd := SilentCommandContext(ctx, "cmd.exe", "/c", "echo", "lumina")
	if ctxCmd == nil {
		t.Fatalf("SilentCommandContext returned nil cmd")
	}

	if runtime.GOOS == "windows" {
		if ctxCmd.SysProcAttr == nil {
			t.Fatalf("Expected ctxCmd.SysProcAttr to be configured on Windows")
		}
		if !ctxCmd.SysProcAttr.HideWindow {
			t.Errorf("Expected ctxCmd HideWindow to be true")
		}
		if ctxCmd.SysProcAttr.CreationFlags&CREATE_NO_WINDOW == 0 {
			t.Errorf("Expected ctxCmd CREATE_NO_WINDOW (0x08000000) flag to be set")
		}
	}
}
