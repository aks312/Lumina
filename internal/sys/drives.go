package sys

import (
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// DetectedDriveInfo stores details of an identified storage volume on Windows.
type DetectedDriveInfo struct {
	DriveLetter string  `json:"driveLetter"` // e.g. "C:\\", "D:\\"
	VolumeLabel string  `json:"volumeLabel"` // e.g. "NVMe SSD", "EOS_DIGITAL"
	DriveType   string  `json:"driveType"`   // "Fixed SSD/HDD" | "Removable SD/USB" | "Network Drive"
	TotalGB     float64 `json:"totalGB"`
	FreeGB      float64 `json:"freeGB"`
	Recommended bool    `json:"recommended"` // true for largest drive or camera SD card
}

// CommonFolderInfo stores user standard media folders (Pictures, Desktop, Downloads)
type CommonFolderInfo struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Icon string `json:"icon"` // "pictures", "desktop", "downloads", "camera"
}

// GetDetectedDrives scans drive letters (A:\ to Z:\) and returns all active volumes with capacities.
func GetDetectedDrives() []DetectedDriveInfo {
	var drives []DetectedDriveInfo
	if runtime.GOOS != "windows" {
		return drives
	}

	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	procGetLogicalDrives := kernel32.NewProc("GetLogicalDrives")
	procGetDriveType := kernel32.NewProc("GetDriveTypeW")
	procGetVolumeInformation := kernel32.NewProc("GetVolumeInformationW")

	bitmask, _, _ := procGetLogicalDrives.Call()
	if bitmask == 0 {
		return drives
	}

	var maxFreeGB float64
	bestIdx := -1

	for i := 0; i < 26; i++ {
		if (bitmask & (1 << i)) != 0 {
			letter := string(rune('A'+i)) + ":\\"
			letterPtr, err := syscall.UTF16PtrFromString(letter)
			if err != nil {
				continue
			}

			dt, _, _ := procGetDriveType.Call(uintptr(unsafe.Pointer(letterPtr)))
			driveTypeStr := "Unknown"
			switch dt {
			case 2:
				driveTypeStr = "Removable SD/USB"
			case 3:
				driveTypeStr = "Fixed SSD/HDD"
			case 4:
				driveTypeStr = "Network Drive"
			default:
				// Skip CD-ROM or RAM disks
				continue
			}

			// Volume Name
			var volNameBuf [260]uint16
			var fsNameBuf [260]uint16
			_, _, _ = procGetVolumeInformation.Call(
				uintptr(unsafe.Pointer(letterPtr)),
				uintptr(unsafe.Pointer(&volNameBuf[0])),
				uintptr(len(volNameBuf)),
				0, 0, 0,
				uintptr(unsafe.Pointer(&fsNameBuf[0])),
				uintptr(len(fsNameBuf)),
			)
			volLabel := syscall.UTF16ToString(volNameBuf[:])
			if volLabel == "" {
				if dt == 2 {
					volLabel = "Removable Drive"
				} else {
					volLabel = "Local Disk"
				}
			}

			// Free & Total Space
			freeBytes, totalBytes, err := GetAvailableDiskSpace(letter)
			if err != nil || totalBytes == 0 {
				continue
			}

			totalGB := float64(totalBytes) / (1024 * 1024 * 1024)
			freeGB := float64(freeBytes) / (1024 * 1024 * 1024)

			driveInfo := DetectedDriveInfo{
				DriveLetter: letter,
				VolumeLabel: volLabel,
				DriveType:   driveTypeStr,
				TotalGB:     round1(totalGB),
				FreeGB:      round1(freeGB),
				Recommended: false,
			}

			drives = append(drives, driveInfo)
			idx := len(drives) - 1

			// Prefer non-system drive with largest free space, or largest drive overall
			if freeGB > maxFreeGB {
				maxFreeGB = freeGB
				bestIdx = idx
			}
		}
	}

	if bestIdx >= 0 && bestIdx < len(drives) {
		drives[bestIdx].Recommended = true
	}

	return drives
}

// GetCommonMediaFolders discovers standard Windows user folders (Pictures, Desktop, Downloads, Camera DCIM).
func GetCommonMediaFolders() []CommonFolderInfo {
	var folders []CommonFolderInfo

	userProfile := os.Getenv("USERPROFILE")
	if userProfile == "" {
		return folders
	}

	candidates := []struct {
		name string
		sub  string
		icon string
	}{
		{"Pictures", "Pictures", "pictures"},
		{"Desktop", "Desktop", "desktop"},
		{"Downloads", "Downloads", "downloads"},
	}

	for _, c := range candidates {
		fullPath := filepath.Join(userProfile, c.sub)
		if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
			folders = append(folders, CommonFolderInfo{
				Name: c.name,
				Path: fullPath,
				Icon: c.icon,
			})
		}
	}

	// Check if any removable drives have a DCIM folder (e.g., Camera SD cards)
	drives := GetDetectedDrives()
	for _, d := range drives {
		if d.DriveType == "Removable SD/USB" {
			dcim := filepath.Join(d.DriveLetter, "DCIM")
			if info, err := os.Stat(dcim); err == nil && info.IsDir() {
				folders = append(folders, CommonFolderInfo{
					Name: d.VolumeLabel + " (DCIM)",
					Path: dcim,
					Icon: "camera",
				})
			}
		}
	}

	return folders
}

func round1(val float64) float64 {
	return float64(int(val*10+0.5)) / 10
}
