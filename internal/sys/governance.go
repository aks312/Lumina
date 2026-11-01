package sys

import (
	"log"
	"runtime"
	"syscall"

	"golang.org/x/sys/windows"
)

const (
	// Windows Execution State flags for power management
	ES_CONTINUOUS       = 0x80000000
	ES_SYSTEM_REQUIRED  = 0x00000001
	ES_AWAYMODE_REQUIRED = 0x00000040

	// Windows Process Priority Classes
	NORMAL_PRIORITY_CLASS       = 0x00000020
	BELOW_NORMAL_PRIORITY_CLASS = 0x00004000
	IDLE_PRIORITY_CLASS         = 0x00000040
)

var (
	kernel32               = windows.NewLazySystemDLL("kernel32.dll")
	procSetThreadExecState = kernel32.NewProc("SetThreadExecutionState")
)

// GetOptimalWorkerCount calculates max(2, runtime.NumCPU()/2) to leave headroom for 120fps UI.
func GetOptimalWorkerCount() int {
	cpus := runtime.NumCPU()
	workers := cpus / 2
	if workers < 2 {
		workers = 2
	}
	return workers
}

// PreventSleep signals Windows that active scanning or uploading is in progress,
// preventing system standby or monitor-off sleep.
func PreventSleep() {
	if runtime.GOOS != "windows" {
		return
	}
	r, _, err := procSetThreadExecState.Call(uintptr(ES_CONTINUOUS | ES_SYSTEM_REQUIRED))
	if r == 0 {
		log.Printf("[sys] SetThreadExecutionState failed: %v", err)
	}
}

// AllowSleep restores default Windows power management sleep behavior.
func AllowSleep() {
	if runtime.GOOS != "windows" {
		return
	}
	procSetThreadExecState.Call(uintptr(ES_CONTINUOUS))
}

// SetBackgroundPriority lowers process priority to BELOW_NORMAL_PRIORITY_CLASS during heavy scan/indexing.
func SetBackgroundPriority() {
	if runtime.GOOS != "windows" {
		return
	}
	h, err := windows.GetCurrentProcess()
	if err != nil {
		return
	}
	_ = windows.SetPriorityClass(h, BELOW_NORMAL_PRIORITY_CLASS)
}

// SetNormalPriority restores process priority to NORMAL_PRIORITY_CLASS when background work finishes.
func SetNormalPriority() {
	if runtime.GOOS != "windows" {
		return
	}
	h, err := windows.GetCurrentProcess()
	if err != nil {
		return
	}
	_ = windows.SetPriorityClass(h, NORMAL_PRIORITY_CLASS)
}

// GetAvailableDiskSpace returns free and total bytes for a given path.
func GetAvailableDiskSpace(path string) (freeBytes uint64, totalBytes uint64, err error) {
	if runtime.GOOS != "windows" {
		return 0, 0, nil
	}
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, 0, err
	}
	var freeBytesAvailable, totalNumberOfBytes, totalNumberOfFreeBytes uint64
	err = windows.GetDiskFreeSpaceEx(pathPtr, &freeBytesAvailable, &totalNumberOfBytes, &totalNumberOfFreeBytes)
	if err != nil {
		return 0, 0, err
	}
	return freeBytesAvailable, totalNumberOfBytes, nil
}
