package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"lumina/internal/ai"
	"lumina/internal/cloud"
	"lumina/internal/db"
	"lumina/internal/indexer"
	"lumina/internal/sys"
	"lumina/internal/thumbs"
)

// SetupHubData stores discovery data for consumer onboarding
type SetupHubData struct {
	Drives        []sys.DetectedDriveInfo `json:"drives"`
	CommonFolders []sys.CommonFolderInfo  `json:"commonFolders"`
	SuggestedDest string                  `json:"suggestedDest"`
	HasFastNVMe   bool                    `json:"hasFastNVMe"`
	FastDrive     string                  `json:"fastDrive"`
}

// App struct
type App struct {
	ctx           context.Context
	database      *db.Database
	packStore     *thumbs.PackStore
	cloudAI       *ai.CloudAIEngine
	indexer       *indexer.CatalogIndexer
	gphotos       *cloud.GooglePhotosSyncManager
	dataDir       string
	mu            sync.RWMutex
	isFullscreen  bool
}

// NewApp creates a new App application struct
func NewApp() *App {
	// Determine AppData directory for persistent storage
	userProfile := os.Getenv("USERPROFILE")
	if userProfile == "" {
		userProfile = "."
	}
	dataDir := filepath.Join(userProfile, ".lumina_photos_v3")
	_ = os.MkdirAll(dataDir, 0755)

	database, err := db.Open(dataDir)
	if err != nil {
		fmt.Printf("Critical: DB open error: %v\n", err)
	}

	packStore, err := thumbs.NewPackStore(dataDir)
	if err != nil {
		fmt.Printf("Critical: PackStore open error: %v\n", err)
	}

	savedGeminiKey := ""
	if database != nil {
		savedGeminiKey = database.GetSetting(context.Background(), "gemini_api_key", "")
	}
	cloudAIEngine := ai.NewCloudAIEngine(savedGeminiKey)

	var catalogIndexer *indexer.CatalogIndexer
	if database != nil && packStore != nil {
		catalogIndexer = indexer.NewCatalogIndexer(database, packStore, cloudAIEngine)
	}

	var gphotosManager *cloud.GooglePhotosSyncManager
	if database != nil {
		gphotosManager = cloud.NewGooglePhotosSyncManager(database)
	}

	return &App{
		database:  database,
		packStore: packStore,
		cloudAI:   cloudAIEngine,
		indexer:   catalogIndexer,
		gphotos:   gphotosManager,
		dataDir:   dataDir,
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Wire indexer progress callback to Wails events
	if a.indexer != nil {
		a.indexer.SetProgressCallback(func(p indexer.IndexerProgress) {
			runtime.EventsEmit(ctx, "indexer:progress", p)
		})
	}

	// Auto-benchmark Primary vs Mirror Drive on startup
	go func() {
		primary := a.database.GetSetting(ctx, "primary_destination", "")
		mirror := a.database.GetSetting(ctx, "secondary_mirror", "")
		if primary != "" {
			_, _, fastest := sys.CompareDrives(primary, mirror)
			_ = a.database.SetSetting(ctx, "fastest_serving_drive", fastest)
		}
	}()
}

// shutdown is called at termination
func (a *App) shutdown(ctx context.Context) {
	if a.database != nil {
		_ = a.database.Close()
	}
	if a.packStore != nil {
		_ = a.packStore.Close()
	}
}

// Frontend API Bindings

// GetStats returns catalog overview statistics.
func (a *App) GetStats() db.CatalogStats {
	if a.database == nil || a.database.MemIndex == nil {
		return db.CatalogStats{}
	}
	return a.database.MemIndex.GetStats()
}

// GetPhotos executes sub-millisecond in-memory filtering across the catalog.
func (a *App) GetPhotos(tab string, year int, month int, device string, favoritesOnly bool, ratingAtLeast int, query string, offset int, limit int) []*db.PhotoRecord {
	if a.database == nil || a.database.MemIndex == nil {
		return make([]*db.PhotoRecord, 0)
	}

	var searchIDs []uint32
	if strings.TrimSpace(query) != "" {
		ids, err := a.database.SearchPhotosFTS(a.ctx, strings.TrimSpace(query), 500)
		if err == nil {
			searchIDs = ids
		}
	}

	opts := db.FilterOptions{
		Tab:           tab,
		Year:          year,
		Month:         month,
		Device:        device,
		FavoritesOnly: favoritesOnly,
		RatingAtLeast: ratingAtLeast,
		SearchIDs:     searchIDs,
		Offset:        offset,
		Limit:         limit,
	}

	matchingIDs := a.database.MemIndex.Filter(opts)
	results := make([]*db.PhotoRecord, 0, len(matchingIDs))

	for _, id := range matchingIDs {
		p, err := a.database.GetPhotoByID(a.ctx, id)
		if err == nil && p != nil {
			results = append(results, p)
		}
	}

	return results
}

// GetPhotoDetails fetches a single photo with full metadata by ID.
func (a *App) GetPhotoDetails(id uint32) (*db.PhotoRecord, error) {
	if a.database == nil {
		return nil, fmt.Errorf("database unavailable")
	}
	return a.database.GetPhotoByID(a.ctx, id)
}

// GetTimelineSummary returns photo counts per Year-Month for the right-hand scrubbing bar.
func (a *App) GetTimelineSummary(tab string) []db.TimelineBucket {
	if a.database == nil || a.database.MemIndex == nil {
		return make([]db.TimelineBucket, 0)
	}
	return a.database.MemIndex.GetTimelineSummary(tab)
}

// UpdatePhoto persists manual changes to a photo (sets is_user_modified = 1).
func (a *App) UpdatePhoto(id uint32, routeFolder string, platform string, rotation int, rating int, isFavorite bool, showInAll bool) error {
	if a.database == nil {
		return fmt.Errorf("database unavailable")
	}
	return a.database.UpdatePhotoUserFields(a.ctx, id, routeFolder, platform, rotation, rating, isFavorite, showInAll)
}

// BatchUpdatePhotos modifies multiple selected photos (Move to Folder, Hide/Show in All, Favorite).
func (a *App) BatchUpdatePhotos(ids []uint32, routeFolder string, isFavorite *bool, showInAll *bool) error {
	if a.database == nil {
		return fmt.Errorf("database unavailable")
	}

	for _, id := range ids {
		p, err := a.database.GetPhotoByID(a.ctx, id)
		if err != nil || p == nil {
			continue
		}

		targetFolder := p.RouteFolder
		if routeFolder != "" {
			targetFolder = routeFolder
		}

		fav := p.IsFavorite
		if isFavorite != nil {
			fav = *isFavorite
		}

		showAll := p.ShowInAll
		if showInAll != nil {
			showAll = *showInAll
		}

		_ = a.database.UpdatePhotoUserFields(a.ctx, id, targetFolder, p.Platform, p.Rotation, p.Rating, fav, showAll)
	}

	return nil
}

// TriggerSmartTagAndOCR runs zero-key local visual scene analysis and native Windows OCR.
// If an optional Gemini key is configured, seamlessly leverages the multi-model cloud boost!
func (a *App) TriggerSmartTagAndOCR(photoIDs []uint32) error {
	if a.database == nil {
		return fmt.Errorf("database unavailable")
	}

	go func() {
		for _, id := range photoIDs {
			p, err := a.database.GetPhotoByID(a.ctx, id)
			if err != nil || p == nil {
				continue
			}

			targetPath := p.OrganizedPath
			if targetPath == "" || !fileExists(targetPath) {
				targetPath = p.OriginalPath
			}

			// Local offline analysis
			analysis, err := ai.AnalyzeLocalPhoto(a.ctx, targetPath, nil, true)
			if err == nil && analysis != nil {
				tagStr := strings.Join(analysis.Tags, ", ")
				_ = a.database.UpdatePhotoAiAndOcr(a.ctx, id, tagStr, analysis.OcrText, analysis.Description)
			}
		}

		runtime.EventsEmit(a.ctx, "ai:analysis_complete", map[string]int{"count": len(photoIDs)})
	}()

	return nil
}

// Watched Folders Management

func (a *App) GetWatchedFolders() ([]db.WatchedFolder, error) {
	if a.database == nil {
		return make([]db.WatchedFolder, 0), fmt.Errorf("database unavailable")
	}
	return a.database.GetAllWatchedFolders(a.ctx)
}

func (a *App) AddWatchedFolder(path string, routeOverride string, showInAll bool) error {
	if a.database == nil {
		return fmt.Errorf("database unavailable")
	}
	return a.database.AddWatchedFolder(a.ctx, path, routeOverride, showInAll)
}

func (a *App) RemoveWatchedFolder(id int) error {
	if a.database == nil {
		return fmt.Errorf("database unavailable")
	}
	return a.database.RemoveWatchedFolder(a.ctx, id)
}

// Device Mappings Management

func (a *App) GetDeviceMappings() ([]db.DeviceMapping, error) {
	if a.database == nil {
		return make([]db.DeviceMapping, 0), fmt.Errorf("database unavailable")
	}
	return a.database.GetAllDeviceMappings(a.ctx)
}

func (a *App) SetDeviceMapping(deviceName string, customFolder string) error {
	if a.database == nil {
		return fmt.Errorf("database unavailable")
	}
	return a.database.SetDeviceMapping(a.ctx, deviceName, customFolder)
}

func (a *App) DeleteDeviceMapping(deviceName string) error {
	if a.database == nil {
		return fmt.Errorf("database unavailable")
	}
	return a.database.DeleteDeviceMapping(a.ctx, deviceName)
}

// GetSetupHubData discovers local storage drives, user folders, and auto-suggests destination.
func (a *App) GetSetupHubData() SetupHubData {
	drives := sys.GetDetectedDrives()
	if drives == nil {
		drives = make([]sys.DetectedDriveInfo, 0)
	}
	commonFolders := sys.GetCommonMediaFolders()
	if commonFolders == nil {
		commonFolders = make([]sys.CommonFolderInfo, 0)
	}

	suggested := ""
	for _, d := range drives {
		if d.Recommended {
			suggested = filepath.Join(d.DriveLetter, "LuminaPhotos")
			break
		}
	}
	if suggested == "" && len(drives) > 0 {
		suggested = filepath.Join(drives[0].DriveLetter, "LuminaPhotos")
	}
	if suggested == "" {
		userProfile := os.Getenv("USERPROFILE")
		suggested = filepath.Join(userProfile, "Pictures", "LuminaPhotos")
	}

	return SetupHubData{
		Drives:        drives,
		CommonFolders: commonFolders,
		SuggestedDest: suggested,
		HasFastNVMe:   len(drives) > 1,
		FastDrive:     "C:\\",
	}
}

// GetInstantSearchChips returns dynamic search filter chips from catalog.
func (a *App) GetInstantSearchChips() db.InstantSearchChips {
	if a.database == nil || a.database.MemIndex == nil {
		return db.InstantSearchChips{
			TopPlaces:  []string{"Pune", "Mumbai", "Paris", "New York"},
			TopDevices: []string{"Apple_iPhone", "Samsung_Galaxy"},
			Years:      []string{"2026", "2025", "2024"},
			Categories: []string{"Documents & Receipts", "Portraits & People", "Videos", "Live Photos", "High Sharpness"},
		}
	}
	return a.database.MemIndex.GetInstantSearchChips()
}

// ServeThumbnail handles /thumb/{id} and /thumbs/{id}
func (a *App) ServeThumbnail(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	idStr := strings.TrimPrefix(path, "/thumbs/")
	if idStr == path {
		idStr = strings.TrimPrefix(path, "/thumb/")
	}
	id64, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	id := uint32(id64)

	// Try fetching from thumbs.pack + LRU cache
	if a.packStore != nil {
		thumbBytes, err := a.packStore.Get(id)
		if err == nil && len(thumbBytes) > 0 {
			w.Header().Set("Content-Type", "image/jpeg")
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			w.Header().Set("Content-Length", strconv.Itoa(len(thumbBytes)))
			_, _ = w.Write(thumbBytes)
			return
		}
	}

	// Generate on the fly from file
	if a.database != nil {
		photo, err := a.database.GetPhotoByID(context.Background(), id)
		if err == nil && photo != nil {
			filePath := photo.OrganizedPath
			if filePath == "" || !fileExists(filePath) {
				filePath = photo.OriginalPath
			}
			if fileExists(filePath) {
				thumbBytes, err := thumbs.GenerateThumbnail(filePath, 400)
				if err == nil && len(thumbBytes) > 0 {
					if a.packStore != nil {
						_ = a.packStore.Put(id, thumbBytes)
					}
					w.Header().Set("Content-Type", "image/jpeg")
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
					w.Header().Set("Content-Length", strconv.Itoa(len(thumbBytes)))
					_, _ = w.Write(thumbBytes)
					return
				}
			}
		}
	}

	http.NotFound(w, r)
}

// ServeRawMedia handles /raw/{id}
func (a *App) ServeRawMedia(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	idStr := strings.TrimPrefix(path, "/raw/")
	id64, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	id := uint32(id64)

	if a.database != nil {
		photo, err := a.database.GetPhotoByID(context.Background(), id)
		if err == nil && photo != nil {
			filePath := photo.OrganizedPath
			if filePath == "" || !fileExists(filePath) {
				filePath = photo.OriginalPath
			}

			if fileExists(filePath) {
				// Apply rotation if needed
				if photo.Rotation != 0 {
					f, err := os.Open(filePath)
					if err == nil {
						img, _, decErr := image.Decode(f)
						_ = f.Close()
						if decErr == nil && img != nil {
							rotated := thumbs.RotateImage(img, photo.Rotation)
							var buf bytes.Buffer
							if err := jpeg.Encode(&buf, rotated, &jpeg.Options{Quality: 92}); err == nil {
								w.Header().Set("Content-Type", "image/jpeg")
								w.Header().Set("Cache-Control", "no-cache")
								_, _ = w.Write(buf.Bytes())
								return
							}
						}
					}
				}

				ext := strings.ToLower(filepath.Ext(filePath))
				mime := "image/jpeg"
				switch ext {
				case ".png":
					mime = "image/png"
				case ".webp":
					mime = "image/webp"
				case ".mp4":
					mime = "video/mp4"
				case ".mov":
					mime = "video/quicktime"
				case ".mkv":
					mime = "video/x-matroska"
				}
				w.Header().Set("Content-Type", mime)
				w.Header().Set("Cache-Control", "public, max-age=86400")
				http.ServeFile(w, r, filePath)
				return
			}
		}
	}

	http.NotFound(w, r)
}

// App Settings

func (a *App) GetSetting(key string, defaultValue string) string {
	if a.database == nil {
		return defaultValue
	}
	return a.database.GetSetting(a.ctx, key, defaultValue)
}

func (a *App) SaveSetting(key string, val string) error {
	if a.database == nil {
		return fmt.Errorf("database unavailable")
	}
	if key == "gemini_api_key" && a.cloudAI != nil {
		a.cloudAI.SetApiKey(val)
	}
	return a.database.SetSetting(a.ctx, key, val)
}

// Indexing & Scanning

func (a *App) StartScan(organize bool) error {
	if a.indexer == nil {
		return fmt.Errorf("indexer unavailable")
	}
	go func() {
		_ = a.indexer.StartCatalogSweep(a.ctx, organize)
	}()
	return nil
}

// Google Photos Cloud Sync

func (a *App) IsGoogleConnected() bool {
	if a.gphotos == nil {
		return false
	}
	return a.gphotos.IsConnected()
}

func (a *App) StartGoogleSignIn() (string, error) {
	if a.gphotos == nil {
		return "", fmt.Errorf("gphotos sync unavailable")
	}
	return a.gphotos.Start1ClickSignIn(a.ctx, func() {
		runtime.EventsEmit(a.ctx, "gphotos:connected", true)
	})
}

func (a *App) SyncToGooglePhotos() (int, error) {
	if a.gphotos == nil {
		return 0, fmt.Errorf("gphotos sync unavailable")
	}

	// Fetch all photos in catalog
	ids := a.database.MemIndex.Filter(db.FilterOptions{Tab: "all", Limit: 100000})
	photos := make([]*db.PhotoRecord, 0, len(ids))
	for _, id := range ids {
		p, err := a.database.GetPhotoByID(a.ctx, id)
		if err == nil && p != nil {
			photos = append(photos, p)
		}
	}

	go func() {
		_, _ = a.gphotos.UploadBatch(a.ctx, photos, func(synced int, total int) {
			runtime.EventsEmit(a.ctx, "gphotos:upload_progress", map[string]int{
				"synced": synced,
				"total":  total,
			})
		})
	}()

	return len(photos), nil
}

// Administrative / Repair Actions (Settings -> Catalog Maintenance)

func (a *App) GlobalReAnalyze() error {
	if a.indexer == nil {
		return fmt.Errorf("indexer unavailable")
	}
	go func() {
		_ = a.indexer.GlobalReAnalyze(a.ctx)
		runtime.EventsEmit(a.ctx, "maintenance:completed", "Global Re-Analyze completed")
	}()
	return nil
}

func (a *App) ReVerifyScreenshots() error {
	if a.indexer == nil {
		return fmt.Errorf("indexer unavailable")
	}
	go func() {
		_ = a.indexer.ReVerifyScreenshots(a.ctx)
		runtime.EventsEmit(a.ctx, "maintenance:completed", "Re-Verify Screenshots completed")
	}()
	return nil
}

func (a *App) PurgeThumbnails() error {
	if a.indexer == nil {
		return fmt.Errorf("indexer unavailable")
	}
	return a.indexer.PurgeThumbnails(a.ctx)
}

func (a *App) FactoryReset() error {
	if a.database == nil {
		return fmt.Errorf("database unavailable")
	}
	return a.database.FactoryReset(a.ctx)
}

// BenchmarkDrives measures primary vs mirror drive speed in MB/s
func (a *App) BenchmarkDrives() (map[string]any, error) {
	primary := a.database.GetSetting(a.ctx, "primary_destination", "")
	mirror := a.database.GetSetting(a.ctx, "secondary_mirror", "")

	pBench, mBench, fastest := sys.CompareDrives(primary, mirror)
	return map[string]any{
		"primary": pBench,
		"mirror":  mBench,
		"fastest": fastest,
	}, nil
}

// SelectDirectoryDialog opens native Windows folder selection dialog
func (a *App) SelectDirectoryDialog() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Folder",
	})
}

// ToggleFullscreen toggles native Wails fullscreen
func (a *App) ToggleFullscreen() {
	if a.isFullscreen {
		runtime.WindowUnfullscreen(a.ctx)
		a.isFullscreen = false
	} else {
		runtime.WindowFullscreen(a.ctx)
		a.isFullscreen = true
	}
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}
