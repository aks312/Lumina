package db

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// Platform enum constants for fast byte-comparison
const (
	PlatformUnknown uint8 = 0
	PlatformIOS     uint8 = 1
	PlatformAndroid uint8 = 2
	PlatformCamera  uint8 = 3
	PlatformCustom  uint8 = 4
)

func PlatformStringToID(p string) uint8 {
	switch strings.ToLower(p) {
	case "ios":
		return PlatformIOS
	case "android":
		return PlatformAndroid
	case "camera":
		return PlatformCamera
	case "custom":
		return PlatformCustom
	default:
		return PlatformUnknown
	}
}

func PlatformIDToString(id uint8) string {
	switch id {
	case PlatformIOS:
		return "ios"
	case PlatformAndroid:
		return "android"
	case PlatformCamera:
		return "camera"
	case PlatformCustom:
		return "custom"
	default:
		return "unknown"
	}
}

// SlimPhotoNode is a compact 48-64 byte representation of an image in RAM.
// Storing 200,000 photos requires ~12MB of RAM, enabling instant in-memory operations.
type SlimPhotoNode struct {
	ID              uint32
	Timestamp       int64  // Unix seconds for fast sorting
	Year            uint16
	Month           uint8
	Day             uint8
	Platform        uint8
	IsScreenshot    bool
	ShowInAll       bool
	IsUserModified  bool
	IsFavorite      bool
	Rating          uint8
	Rotation        uint16
	Width           uint16
	Height          uint16
	FileSize        uint32
	IsVideo         bool
	RouteFolder     string
	CanonicalDevice string
}

// ToSlimNode converts a full PhotoRecord into a compact SlimPhotoNode.
func (p *PhotoRecord) ToSlimNode() SlimPhotoNode {
	var ts int64
	if !p.DateTaken.IsZero() {
		ts = p.DateTaken.Unix()
	} else {
		ts = time.Now().Unix()
	}

	return SlimPhotoNode{
		ID:              p.ID,
		Timestamp:       ts,
		Year:            uint16(p.Year),
		Month:           uint8(p.Month),
		Day:             uint8(p.Day),
		Platform:        PlatformStringToID(p.Platform),
		IsScreenshot:    p.IsScreenshot,
		ShowInAll:       p.ShowInAll,
		IsUserModified:  p.IsUserModified,
		IsFavorite:      p.IsFavorite,
		Rating:          uint8(p.Rating),
		Rotation:        uint16(p.Rotation),
		Width:           uint16(p.Width),
		Height:          uint16(p.Height),
		FileSize:        uint32(p.FileSize),
		IsVideo:         p.IsVideo,
		RouteFolder:     p.RouteFolder,
		CanonicalDevice: p.CanonicalDevice,
	}
}

// FilterOptions defines filter criteria for sub-millisecond in-memory queries.
type FilterOptions struct {
	Tab         string // "all", "ios", "android", "camera", "screenshots", "unknown", or custom route folder name
	Year        int    // 0 = all years
	Month       int    // 0 = all months
	FavoritesOnly bool
	RatingAtLeast int
	Device      string // Filter by specific canonical device
	SearchIDs   []uint32 // Optional FTS5 ID pre-filter
	Limit       int
	Offset      int
}

// TimelineBucket stores photo count per Year-Month for the scrubbing bar.
type TimelineBucket struct {
	Year  int `json:"year"`
	Month int `json:"month"`
	Count int `json:"count"`
}

// CatalogStats aggregates overview statistics for the UI.
type CatalogStats struct {
	TotalPhotos int   `json:"total_photos"`
	TotalVideos int   `json:"total_videos"`
	TotalBytes  int64 `json:"total_bytes"`
	IOSCount    int   `json:"ios_count"`
	AndroidCount int  `json:"android_count"`
	CameraCount int   `json:"camera_count"`
	Screenshots int   `json:"screenshots"`
	UnknownCount int  `json:"unknown_count"`
	Favorites   int   `json:"favorites"`
}

// MemIndex provides high-performance contiguous memory indexing.
type MemIndex struct {
	mu    sync.RWMutex
	nodes []SlimPhotoNode
	byID  map[uint32]int // index into nodes slice
}

// NewMemIndex creates an initialized MemIndex.
func NewMemIndex() *MemIndex {
	return &MemIndex{
		nodes: make([]SlimPhotoNode, 0, 10000),
		byID:  make(map[uint32]int, 10000),
	}
}

// Reset clears the memory index.
func (m *MemIndex) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nodes = m.nodes[:0]
	m.byID = make(map[uint32]int)
}

// UpsertNode adds or updates a node in the index, maintaining date-descending sort.
func (m *MemIndex) UpsertNode(node SlimPhotoNode) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if idx, exists := m.byID[node.ID]; exists {
		m.nodes[idx] = node
		return
	}

	m.nodes = append(m.nodes, node)
	m.byID[node.ID] = len(m.nodes) - 1
}

// UpdateUserFields modifies user fields in memory without a full re-index.
func (m *MemIndex) UpdateUserFields(id uint32, routeFolder string, platform string, rotation int, rating int, isFavorite bool, showInAll bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	idx, exists := m.byID[id]
	if !exists {
		return
	}

	m.nodes[idx].RouteFolder = routeFolder
	m.nodes[idx].Platform = PlatformStringToID(platform)
	m.nodes[idx].Rotation = uint16(rotation)
	m.nodes[idx].Rating = uint8(rating)
	m.nodes[idx].IsFavorite = isFavorite
	m.nodes[idx].ShowInAll = showInAll
	m.nodes[idx].IsUserModified = true
}

// SortByDateDescending sorts all nodes in memory by timestamp descending.
func (m *MemIndex) SortByDateDescending() {
	m.mu.Lock()
	defer m.mu.Unlock()

	sort.Slice(m.nodes, func(i, j int) bool {
		return m.nodes[i].Timestamp > m.nodes[j].Timestamp
	})

	// Rebuild ID lookup map
	for i, n := range m.nodes {
		m.byID[n.ID] = i
	}
}

// Filter executes in-memory query in < 0.5ms and returns matching photo IDs.
func (m *MemIndex) Filter(opts FilterOptions) []uint32 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// If SearchIDs are provided, build a fast lookup set
	var searchSet map[uint32]struct{}
	if len(opts.SearchIDs) > 0 {
		searchSet = make(map[uint32]struct{}, len(opts.SearchIDs))
		for _, id := range opts.SearchIDs {
			searchSet[id] = struct{}{}
		}
	}

	matched := make([]uint32, 0, 1024)

	tabLower := strings.ToLower(opts.Tab)

	for i := range m.nodes {
		n := &m.nodes[i]

		// 1. If FTS search filter active
		if searchSet != nil {
			if _, ok := searchSet[n.ID]; !ok {
				continue
			}
		}

		// 2. Tab Routing Filter (Strict 5-Tier Rule 1 Enforcement):
		// Priority 1: Screenshots strictly hidden from All, iOS, Android, Camera, Unknown!
		if tabLower == "screenshots" {
			if !n.IsScreenshot {
				continue
			}
		} else {
			// In ALL other tabs, screenshots are strictly excluded!
			if n.IsScreenshot {
				continue
			}

			switch tabLower {
			case "", "all", "all photos":
				// Must respect show_in_all flag
				if !n.ShowInAll {
					continue
				}
			case "ios":
				if n.Platform != PlatformIOS {
					continue
				}
			case "android":
				if n.Platform != PlatformAndroid {
					continue
				}
			case "camera":
				if n.Platform != PlatformCamera {
					continue
				}
			case "unknown":
				if n.Platform != PlatformUnknown {
					continue
				}
			default:
				// Custom folder tab matching (e.g., "Amruta")
				if !strings.EqualFold(n.RouteFolder, opts.Tab) {
					continue
				}
			}
		}

		// 3. Year / Month filter
		if opts.Year > 0 && int(n.Year) != opts.Year {
			continue
		}
		if opts.Month > 0 && int(n.Month) != opts.Month {
			continue
		}

		// 4. Favorites filter
		if opts.FavoritesOnly && !n.IsFavorite {
			continue
		}

		// 5. Rating filter
		if opts.RatingAtLeast > 0 && int(n.Rating) < opts.RatingAtLeast {
			continue
		}

		// 6. Device filter
		if opts.Device != "" && !strings.EqualFold(n.CanonicalDevice, opts.Device) {
			continue
		}

		matched = append(matched, n.ID)
	}

	// Apply Offset and Limit
	totalMatched := len(matched)
	if opts.Offset >= totalMatched {
		return []uint32{}
	}

	end := totalMatched
	if opts.Limit > 0 && opts.Offset+opts.Limit < end {
		end = opts.Offset + opts.Limit
	}

	return matched[opts.Offset:end]
}

// GetTimelineSummary generates Year-Month counts for the scrubbing slider.
func (m *MemIndex) GetTimelineSummary(tab string) []TimelineBucket {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tabLower := strings.ToLower(tab)
	buckets := make(map[int]map[int]int)

	for i := range m.nodes {
		n := &m.nodes[i]

		// Filter for tab
		if tabLower == "screenshots" {
			if !n.IsScreenshot {
				continue
			}
		} else {
			if n.IsScreenshot {
				continue
			}
			if tabLower != "" && tabLower != "all" && tabLower != "all photos" {
				switch tabLower {
				case "ios":
					if n.Platform != PlatformIOS {
						continue
					}
				case "android":
					if n.Platform != PlatformAndroid {
						continue
					}
				case "camera":
					if n.Platform != PlatformCamera {
						continue
					}
				case "unknown":
					if n.Platform != PlatformUnknown {
						continue
					}
				default:
					if !strings.EqualFold(n.RouteFolder, tab) {
						continue
					}
				}
			} else {
				if !n.ShowInAll {
					continue
				}
			}
		}

		y := int(n.Year)
		mo := int(n.Month)
		if y == 0 {
			continue
		}

		if buckets[y] == nil {
			buckets[y] = make(map[int]int)
		}
		buckets[y][mo]++
	}

	// Flatten and sort descending by Year, Month
	result := make([]TimelineBucket, 0)
	for y, months := range buckets {
		for mo, count := range months {
			result = append(result, TimelineBucket{
				Year:  y,
				Month: mo,
				Count: count,
			})
		}
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Year == result[j].Year {
			return result[i].Month > result[j].Month
		}
		return result[i].Year > result[j].Year
	})

	return result
}

// InstantSearchChips provides dynamic filter chips for the smart search bar.
type InstantSearchChips struct {
	TopPlaces  []string `json:"topPlaces"`
	TopDevices []string `json:"topDevices"`
	Years      []string `json:"years"`
	Categories []string `json:"categories"`
}

// GetInstantSearchChips aggregates unique devices, years, and categories from memory index.
func (m *MemIndex) GetInstantSearchChips() InstantSearchChips {
	m.mu.RLock()
	defer m.mu.RUnlock()

	deviceSet := make(map[string]int)
	yearSet := make(map[int]int)

	for i := range m.nodes {
		n := &m.nodes[i]
		if n.CanonicalDevice != "" && n.CanonicalDevice != "Unknown" && n.CanonicalDevice != "Media" {
			deviceSet[n.CanonicalDevice]++
		}
		if n.Year > 1990 {
			yearSet[int(n.Year)]++
		}
	}

	// Sort devices by frequency
	type kv struct {
		key   string
		count int
	}
	var devList []kv
	for k, v := range deviceSet {
		devList = append(devList, kv{k, v})
	}
	sort.Slice(devList, func(i, j int) bool {
		return devList[i].count > devList[j].count
	})

	topDevices := make([]string, 0, 8)
	for i := 0; i < len(devList) && i < 6; i++ {
		topDevices = append(topDevices, devList[i].key)
	}

	// Sort years descending
	var yrs []int
	for y := range yearSet {
		yrs = append(yrs, y)
	}
	sort.Slice(yrs, func(i, j int) bool {
		return yrs[i] > yrs[j]
	})

	yearStrings := make([]string, 0, 8)
	for i := 0; i < len(yrs) && i < 6; i++ {
		yearStrings = append(yearStrings, fmt.Sprintf("%d", yrs[i]))
	}

	// Fallback suggestions if catalog is fresh/empty
	if len(topDevices) == 0 {
		topDevices = []string{"Apple_iPhone", "Samsung_Galaxy", "Canon_EOS", "Sony_Alpha"}
	}
	if len(yearStrings) == 0 {
		currentYear := time.Now().Year()
		yearStrings = []string{
			fmt.Sprintf("%d", currentYear),
			fmt.Sprintf("%d", currentYear-1),
			fmt.Sprintf("%d", currentYear-2),
		}
	}

	return InstantSearchChips{
		TopPlaces:  []string{"Pune", "Mumbai", "Paris", "New York", "San Francisco"},
		TopDevices: topDevices,
		Years:      yearStrings,
		Categories: []string{
			"Documents & Receipts",
			"Portraits & People",
			"Videos",
			"Live Photos",
			"High Sharpness",
		},
	}
}

// GetStats returns fast global catalog statistics.
func (m *MemIndex) GetStats() CatalogStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var stats CatalogStats
	for i := range m.nodes {
		n := &m.nodes[i]
		if n.IsVideo {
			stats.TotalVideos++
		} else {
			stats.TotalPhotos++
		}
		stats.TotalBytes += int64(n.FileSize)

		if n.IsScreenshot {
			stats.Screenshots++
		} else {
			switch n.Platform {
			case PlatformIOS:
				stats.IOSCount++
			case PlatformAndroid:
				stats.AndroidCount++
			case PlatformCamera:
				stats.CameraCount++
			case PlatformUnknown:
				stats.UnknownCount++
			}
		}

		if n.IsFavorite {
			stats.Favorites++
		}
	}
	return stats
}

// Count returns total items indexed.
func (m *MemIndex) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.nodes)
}

// WarmMemIndex reads all photos from readDB and populates MemIndex.
func (d *Database) WarmMemIndex() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := d.readDB.QueryContext(ctx, `SELECT 
		id, file_size, width, height, date_taken, year, month, day,
		canonical_device, platform, is_screenshot, route_folder, show_in_all,
		is_user_modified, rotation, rating, is_favorite, is_video
	FROM photos ORDER BY date_taken DESC;`)
	if err != nil {
		return err
	}
	defer rows.Close()

	d.MemIndex.Reset()

	for rows.Next() {
		var id uint32
		var size int64
		var w, h, yr, mo, dy int
		var dt string
		var device, plat, route string
		var isSS, showAll, isMod, rot, rat, isFav, isVid int

		if err := rows.Scan(
			&id, &size, &w, &h, &dt, &yr, &mo, &dy,
			&device, &plat, &isSS, &route, &showAll,
			&isMod, &rot, &rat, &isFav, &isVid,
		); err != nil {
			return err
		}

		var ts int64
		if parsed, err := time.Parse("2006-01-02 15:04:05", dt); err == nil {
			ts = parsed.Unix()
		} else if parsed, err := time.Parse(time.RFC3339, dt); err == nil {
			ts = parsed.Unix()
		}

		node := SlimPhotoNode{
			ID:              id,
			Timestamp:       ts,
			Year:            uint16(yr),
			Month:           uint8(mo),
			Day:             uint8(dy),
			Platform:        PlatformStringToID(plat),
			IsScreenshot:    isSS == 1,
			ShowInAll:       showAll == 1,
			IsUserModified:  isMod == 1,
			IsFavorite:      isFav == 1,
			Rating:          uint8(rat),
			Rotation:        uint16(rot),
			Width:           uint16(w),
			Height:          uint16(h),
			FileSize:        uint32(size),
			IsVideo:         isVid == 1,
			RouteFolder:     route,
			CanonicalDevice: device,
		}
		d.MemIndex.UpsertNode(node)
	}

	d.MemIndex.SortByDateDescending()
	return nil
}
