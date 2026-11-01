package organizer

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"lumina/internal/db"
)

// RouteDecision contains the calculated destination path and route properties
type RouteDecision struct {
	TargetFolder string // "Screenshots", "Amruta", "iOS", "Android", "Camera", "Unknown"
	RelativeDir  string // "{TargetFolder}/{YYYY}/{YYYY-MM}"
	ShowInAll    bool
	Platform     string // "ios", "android", "camera", "unknown", "custom"
	RuleApplied  int    // 1 to 5
}

// CalculateRoute applies the strict 5-tier routing hierarchy (Rule 1).
func CalculateRoute(
	isScreenshot bool,
	watchedRouteOverride string,
	watchedShowInAll bool,
	isUserModified bool,
	existingRouteFolder string,
	existingPlatform string,
	existingShowInAll bool,
	canonicalDevice string,
	deviceMappings map[string]string,
	detectedPlatform string,
	dateTaken time.Time,
) RouteDecision {
	year := dateTaken.Year()
	if year == 0 {
		year = time.Now().Year()
	}
	monthStr := fmt.Sprintf("%04d-%02d", year, int(dateTaken.Month()))

	// PRIORITY 1: Screenshots (Rule 1)
	if isScreenshot {
		relDir := filepath.Join("Screenshots", fmt.Sprintf("%04d", year), monthStr)
		return RouteDecision{
			TargetFolder: "Screenshots",
			RelativeDir:  relDir,
			ShowInAll:    false, // STRICTLY hidden from All Photos & Platform tabs
			Platform:     detectedPlatform,
			RuleApplied:  1,
		}
	}

	// PRIORITY 2: Watched Source Folder Route Override (Rule 1)
	if strings.TrimSpace(watchedRouteOverride) != "" {
		folderName := strings.TrimSpace(watchedRouteOverride)
		relDir := filepath.Join(folderName, fmt.Sprintf("%04d", year), monthStr)
		return RouteDecision{
			TargetFolder: folderName,
			RelativeDir:  relDir,
			ShowInAll:    watchedShowInAll,
			Platform:     "custom",
			RuleApplied:  2,
		}
	}

	// PRIORITY 3: Manual User Photo Override (Rule 1: background sweeps NEVER overwrite user edits)
	if isUserModified && strings.TrimSpace(existingRouteFolder) != "" {
		relDir := filepath.Join(existingRouteFolder, fmt.Sprintf("%04d", year), monthStr)
		return RouteDecision{
			TargetFolder: existingRouteFolder,
			RelativeDir:  relDir,
			ShowInAll:    existingShowInAll,
			Platform:     existingPlatform,
			RuleApplied:  3,
		}
	}

	// PRIORITY 4: Device-to-Folder Mapping (Rule 1)
	if len(deviceMappings) > 0 && canonicalDevice != "" {
		for dev, customFolder := range deviceMappings {
			if strings.EqualFold(dev, canonicalDevice) && strings.TrimSpace(customFolder) != "" {
				folderName := strings.TrimSpace(customFolder)
				relDir := filepath.Join(folderName, fmt.Sprintf("%04d", year), monthStr)
				return RouteDecision{
					TargetFolder: folderName,
					RelativeDir:  relDir,
					ShowInAll:    true,
					Platform:     "custom",
					RuleApplied:  4,
				}
			}
		}
	}

	// PRIORITY 5: Evidence-Only Platform Fallback (Rule 1 & Rule 3)
	targetFolder := "Unknown"
	plat := "unknown"

	switch strings.ToLower(detectedPlatform) {
	case "ios":
		targetFolder = "iOS"
		plat = "ios"
	case "android":
		targetFolder = "Android"
		plat = "android"
	case "camera":
		targetFolder = "Camera"
		plat = "camera"
	default:
		targetFolder = "Unknown"
		plat = "unknown"
	}

	relDir := filepath.Join(targetFolder, fmt.Sprintf("%04d", year), monthStr)
	return RouteDecision{
		TargetFolder: targetFolder,
		RelativeDir:  relDir,
		ShowInAll:    true,
		Platform:     plat,
		RuleApplied:  5,
	}
}

// GenerateOrganizedFileName creates canonical non-colliding filename:
// Format: {YYYYMMDD}_{HHMMSS}_{CanonicalDevice}_{Hash[:8]}.{ext}
func GenerateOrganizedFileName(dateTaken time.Time, canonicalDevice string, hash string, origExt string) string {
	ext := strings.ToLower(origExt)
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}

	if dateTaken.IsZero() {
		dateTaken = time.Now()
	}

	datePart := dateTaken.Format("20060102_150405")

	devPart := canonicalDevice
	if devPart == "" || devPart == "Unknown" {
		devPart = "Media"
	}

	shortHash := hash
	if len(shortHash) > 8 {
		shortHash = shortHash[:8]
	}
	if shortHash == "" {
		shortHash = "00000000"
	}

	return fmt.Sprintf("%s_%s_%s%s", datePart, devPart, shortHash, ext)
}

// BuildDeviceMappingLookup converts database device mappings to a map.
func BuildDeviceMappingLookup(mappings []db.DeviceMapping) map[string]string {
	m := make(map[string]string, len(mappings))
	for _, mapping := range mappings {
		m[mapping.DeviceName] = mapping.CustomFolder
	}
	return m
}
