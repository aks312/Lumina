//go:build windows

package sys

import (
	"context"
	"os/exec"
	"syscall"
)

const CREATE_NO_WINDOW = 0x08000000

// SilentCommand creates an exec.Cmd guaranteed NEVER to open a console/terminal window on Windows.
func SilentCommand(name string, arg ...string) *exec.Cmd {
	cmd := exec.Command(name, arg...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: CREATE_NO_WINDOW,
	}
	return cmd
}

// SilentCommandContext creates a context-aware exec.Cmd guaranteed NEVER to open a console window.
func SilentCommandContext(ctx context.Context, name string, arg ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, arg...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: CREATE_NO_WINDOW,
	}
	return cmd
}
