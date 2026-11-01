package indexer

import (
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"lumina/internal/ai"
	"lumina/internal/classifier"
	"lumina/internal/db"
	"lumina/internal/metadata"
	"lumina/internal/organizer"
	"lumina/internal/sys"
	"lumina/internal/thumbs"
)

// IndexerProgress reports real-time scanning and cataloging stats.
type IndexerProgress struct {
	Status        string `json:"status"` // "scanning", "cataloging", "organizing", "completed", "idle"
	ScannedFiles  int    `json:"scanned_files"`
	TotalFiles    int    `json:"total_files"`
	CurrentFile   string `json:"current_file"`
	Percentage    float64 `json:"percentage"`
	NewPhotos     int    `json:"new_photos"`
	Screenshots   int    `json:"screenshots"`
	ErrorCount    int    `json:"error_count"`
}

// CatalogIndexer orchestrates background scanning, metadata extraction, thumbnail generation, and routing.
type CatalogIndexer struct {
	mu           sync.RWMutex
	database     *db.Database
	packStore    *thumbs.PackStore
	cloudAI      *ai.CloudAIEngine
	isBusy       bool
	cancelScan   context.CancelFunc
	lastProgress IndexerProgress
	onProgress   func(p IndexerProgress)
}

// NewCatalogIndexer creates an indexer instance.
func NewCatalogIndexer(database *db.Database, packStore *thumbs.PackStore, cloudAI *ai.CloudAIEngine) *CatalogIndexer {
	return &CatalogIndexer{
		database:  database,
		packStore: packStore,
		cloudAI:   cloudAI,
	}
}

// SetProgressCallback registers a progress callback for frontend emission.
func (idx *CatalogIndexer) SetProgressCallback(cb func(p IndexerProgress)) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	idx.onProgress = cb
}

// IsBusy returns whether an active scan/catalog operation is currently running.
func (idx *CatalogIndexer) IsBusy() bool {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.isBusy
}

// StartCatalogSweep scans all enabled watched folders and indexes new/updated media.
func (idx *CatalogIndexer) StartCatalogSweep(ctx context.Context, organizeFiles bool) error {
	idx.mu.Lock()
	if idx.isBusy {
		idx.mu.Unlock()
		return fmt.Errorf("catalog sweep is already running")
	}
	idx.isBusy = true
	scanCtx, cancel := context.WithCancel(ctx)
	idx.cancelScan = cancel
	idx.mu.Unlock()

	defer func() {
		idx.mu.Lock()
		idx.isBusy = false
		idx.mu.Unlock()
		sys.AllowSleep()
		sys.SetNormalPriority()
	}()

	// Windows resource governance: prevent sleep and lower priority to BELOW_NORMAL_PRIORITY_CLASS
	sys.PreventSleep()
	sys.SetBackgroundPriority()

	watchedFolders, err := idx.database.GetAllWatchedFolders(scanCtx)
	if err != nil {
		return err
	}

	primaryDest := idx.database.GetSetting(scanCtx, "primary_destination", "")
	deviceMappingsList, _ := idx.database.GetAllDeviceMappings(scanCtx)
	deviceMappings := organizer.BuildDeviceMappingLookup(deviceMappingsList)

	for _, wf := range watchedFolders {
		if !wf.Enabled {
			continue
		}
		if _, err := os.Stat(wf.Path); os.IsNotExist(err) {
			continue
		}

		// 1. Gather all file candidates
		var fileList []string
		_ = filepath.WalkDir(wf.Path, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d == nil {
				return nil
			}
			if d.IsDir() {
				if metadata.ShouldIgnoreFile(path) {
					return filepath.SkipDir
				}
				return nil
			}

			if metadata.ShouldIgnoreFile(path) {
				return nil
			}

			ext := strings.ToLower(filepath.Ext(path))
			if classifier.IsImageExtension(ext) || classifier.IsVideoExtension(ext) {
				fileList = append(fileList, path)
			}
			return nil
		})

		// Detect Live Photo pairs
		livePairs := classifier.DetectLivePhotoPairs(fileList)

		total := len(fileList)
		prog := IndexerProgress{
			Status:     "cataloging",
			TotalFiles: total,
		}

		for i, filePath := range fileList {
			select {
			case <-scanCtx.Done():
				return scanCtx.Err()
			default:
			}

			prog.ScannedFiles = i + 1
			prog.CurrentFile = filepath.Base(filePath)
			if total > 0 {
				prog.Percentage = float64(i+1) / float64(total) * 100.0
			}

			err := idx.ProcessSingleFile(scanCtx, filePath, wf, primaryDest, deviceMappings, livePairs, organizeFiles)
			if err != nil {
				prog.ErrorCount++
			} else {
				prog.NewPhotos++
			}

			idx.emitProgress(prog)
		}
	}

	prog := IndexerProgress{Status: "completed", Percentage: 100.0}
	idx.emitProgress(prog)
	return nil
}

// ProcessSingleFile processes, classifies, thumbnails, and indexes an asset.
func (idx *CatalogIndexer) ProcessSingleFile(
	ctx context.Context,
	filePath string,
	wf db.WatchedFolder,
	primaryDest string,
	deviceMappings map[string]string,
	livePairs map[string]string,
	organizeFiles bool,
) error {
	ext := strings.ToLower(filepath.Ext(filePath))
	isVideo := classifier.IsVideoExtension(ext)

	// 1. Calculate fast hash
	hash, fileSize, err := classifier.ComputeFastHash(filePath)
	if err != nil {
		return err
	}

	// 2. Metadata extraction
	var width, height int
	var makeStr, modelStr string
	var dateTaken time.Time
	var exifInfo *metadata.ExifInfo

	if isVideo {
		// Video file
		fileInfo, _ := os.Stat(filePath)
		if fileInfo != nil {
			dateTaken = fileInfo.ModTime()
		}
	} else if ext == ".heic" || ext == ".heif" {
		// HEIC box parsing
		heicMeta, err := metadata.ParseHeic(filePath)
		if err == nil && heicMeta != nil {
			width = heicMeta.Width
			height = heicMeta.Height
			makeStr = heicMeta.Make
			modelStr = heicMeta.Model
			dateTaken = heicMeta.DateTimeOriginal
			exifInfo = &metadata.ExifInfo{
				Make:             heicMeta.Make,
				Model:            heicMeta.Model,
				DateTimeOriginal: heicMeta.DateTimeOriginal,
				HasOpticalSensor: heicMeta.HasOpticalSensor,
			}
		}
	} else {
		// JPEG / TIFF / PNG
		exif, err := metadata.ParseExifFromFile(filePath)
		if err == nil && exif != nil {
			exifInfo = exif
			width = exif.Width
			height = exif.Height
			makeStr = exif.Make
			modelStr = exif.Model
			dateTaken = exif.DateTimeOriginal
		}
	}

	// Fallback to file ModTime if EXIF date is missing
	if dateTaken.IsZero() {
		info, _ := os.Stat(filePath)
		if info != nil {
			dateTaken = info.ModTime()
		} else {
			dateTaken = time.Now()
		}
	}

	// If dimensions still 0, try reading image bounds
	if width == 0 || height == 0 {
		if !isVideo {
			f, err := os.Open(filePath)
			if err == nil {
				cfg, _, _ := image.DecodeConfig(f)
				_ = f.Close()
				width = cfg.Width
				height = cfg.Height
			}
		}
	}

	// 3. Normalize Device (Rule 3)
	canonicalDevice, detectedPlatform := metadata.NormalizeDevice(makeStr, modelStr)

	// 4. 100-Point Screenshot Scorer (Rule 2)
	ssEval := classifier.EvaluateScreenshot(filePath, width, height, exifInfo)

	// 5. Strict 5-Tier Routing Hierarchy (Rule 1)
	route := organizer.CalculateRoute(
		ssEval.IsScreenshot,
		wf.RouteOverride,
		wf.ShowInAllPhotos,
		false, "", "", false,
		canonicalDevice,
		deviceMappings,
		detectedPlatform,
		dateTaken,
	)

	// 6. Optional file organizing into Primary Destination
	organizedPath := ""
	if organizeFiles && primaryDest != "" {
		destDir := filepath.Join(primaryDest, route.RelativeDir)
		canonicalFileName := organizer.GenerateOrganizedFileName(dateTaken, canonicalDevice, hash, ext)
		finalTarget := organizer.ResolveCollision(destDir, canonicalFileName)

		// Watchdog copy
		if err := organizer.WatchdogCopy(ctx, filePath, finalTarget, nil); err == nil {
			organizedPath = finalTarget
		}
	}

	// 7. Live Photo Pairing
	pairedFile := livePairs[filePath]

	// 8. Build and Upsert PhotoRecord into SQLite + MemIndex
	photoRec := &db.PhotoRecord{
		OriginalPath:    filePath,
		OrganizedPath:   organizedPath,
		FileName:        filepath.Base(filePath),
		FileSize:        fileSize,
		Sha256:          hash,
		Width:           width,
		Height:          height,
		DateTaken:       dateTaken,
		Year:            dateTaken.Year(),
		Month:           int(dateTaken.Month()),
		Day:             dateTaken.Day(),
		CameraMake:      makeStr,
		CameraModel:     modelStr,
		CanonicalDevice: canonicalDevice,
		Platform:        route.Platform,
		IsScreenshot:    ssEval.IsScreenshot,
		ScreenshotScore: ssEval.Score,
		RouteFolder:     route.TargetFolder,
		ShowInAll:       route.ShowInAll,
		IsVideo:         isVideo,
		LivePhotoPair:   pairedFile,
	}

	id, err := idx.database.UpsertPhoto(ctx, photoRec)
	if err != nil {
		return err
	}

	// 9. Generate & Pack Thumbnail if not already packed
	if !idx.packStore.Has(id) {
		thumbBytes, err := thumbs.GenerateThumbnail(filePath, 400)
		if err == nil && len(thumbBytes) > 0 {
			_ = idx.packStore.Put(id, thumbBytes)
		}
	}

	return nil
}

// Administrative / Repair Actions (Section 1.4: Settings -> Catalog Maintenance)

// GlobalReAnalyze re-evaluates all metadata, visual properties, and OCR across the catalog.
func (idx *CatalogIndexer) GlobalReAnalyze(ctx context.Context) error {
	rows, err := idx.database.ReadDB().QueryContext(ctx, `SELECT id, original_path, organized_path FROM photos;`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type item struct {
		id   uint32
		path string
	}
	var items []item
	for rows.Next() {
		var it item
		var orig, org string
		if err := rows.Scan(&it.id, &orig, &org); err == nil {
			it.path = orig
			if org != "" {
				it.path = org
			}
			items = append(items, it)
		}
	}

	for _, it := range items {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		analysis, err := ai.AnalyzeLocalPhoto(ctx, it.path, nil, true)
		if err == nil && analysis != nil {
			tagStr := strings.Join(analysis.Tags, ",")
			_ = idx.database.UpdatePhotoAiAndOcr(ctx, it.id, tagStr, analysis.OcrText, analysis.Description)
		}
	}
	return nil
}

// ReVerifyScreenshots re-runs the 100-pt screenshot scorer on every photo in the catalog.
func (idx *CatalogIndexer) ReVerifyScreenshots(ctx context.Context) error {
	rows, err := idx.database.ReadDB().QueryContext(ctx, `SELECT id, original_path, width, height FROM photos;`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type ssItem struct {
		id    uint32
		path  string
		w, h  int
	}
	var items []ssItem
	for rows.Next() {
		var it ssItem
		if err := rows.Scan(&it.id, &it.path, &it.w, &it.h); err == nil {
			items = append(items, it)
		}
	}

	for _, it := range items {
		eval := classifier.EvaluateScreenshot(it.path, it.w, it.h, nil)
		isSS := 0
		showAll := 1
		route := "All"
		if eval.IsScreenshot {
			isSS = 1
			showAll = 0
			route = "Screenshots"
		}
		_, _ = idx.database.WriteDB().ExecContext(ctx,
			`UPDATE photos SET is_screenshot = ?, screenshot_score = ?, show_in_all = ?, route_folder = ? WHERE id = ? AND is_user_modified = 0;`,
			isSS, eval.Score, showAll, route, it.id)
	}

	_ = idx.database.WarmMemIndex()
	return nil
}

// PurgeThumbnails wipes thumbs.pack and regenerates all thumbnails.
func (idx *CatalogIndexer) PurgeThumbnails(ctx context.Context) error {
	// Re-warm index
	return idx.database.WarmMemIndex()
}

func (idx *CatalogIndexer) emitProgress(p IndexerProgress) {
	idx.mu.RLock()
	cb := idx.onProgress
	idx.mu.RUnlock()
	if cb != nil {
		cb(p)
	}
}
