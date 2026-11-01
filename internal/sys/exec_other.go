//go:build !windows

package sys

import (
	"context"
	"os/exec"
)

// SilentCommand creates an exec.Cmd on non-Windows platforms.
func SilentCommand(name string, arg ...string) *exec.Cmd {
	return exec.Command(name, arg...)
}

// SilentCommandContext creates a context-aware exec.Cmd on non-Windows platforms.
func SilentCommandContext(ctx context.Context, name string, arg ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, arg...)
}
