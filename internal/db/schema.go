package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// initSchema sets up tables, indexes, triggers, and FTS5 virtual tables.
func (d *Database) initSchema() error {
	d.writeMu.Lock()
	defer d.writeMu.Unlock()

	queries := []string{
		// Base photos table
		`CREATE TABLE IF NOT EXISTS photos (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			original_path TEXT NOT NULL UNIQUE,
			organized_path TEXT DEFAULT '',
			file_name TEXT NOT NULL,
			file_size INTEGER NOT NULL,
			sha256 TEXT NOT NULL,
			width INTEGER DEFAULT 0,
			height INTEGER DEFAULT 0,
			date_taken DATETIME NOT NULL,
			year INTEGER NOT NULL,
			month INTEGER NOT NULL,
			day INTEGER NOT NULL,
			camera_make TEXT DEFAULT '',
			camera_model TEXT DEFAULT '',
			canonical_device TEXT DEFAULT '',
			platform TEXT DEFAULT 'unknown',
			is_screenshot INTEGER DEFAULT 0,
			screenshot_score INTEGER DEFAULT 0,
			route_folder TEXT DEFAULT '',
			show_in_all INTEGER DEFAULT 1,
			is_user_modified INTEGER DEFAULT 0,
			rotation INTEGER DEFAULT 0,
			rating INTEGER DEFAULT 0,
			is_favorite INTEGER DEFAULT 0,
			tags TEXT DEFAULT '',
			ocr_text TEXT DEFAULT '',
			ai_description TEXT DEFAULT '',
			face_count INTEGER DEFAULT 0,
			is_video INTEGER DEFAULT 0,
			duration_sec REAL DEFAULT 0,
			live_photo_pair TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,

		// Fast filter indexes
		`CREATE INDEX IF NOT EXISTS idx_photos_sha256 ON photos(sha256);`,
		`CREATE INDEX IF NOT EXISTS idx_photos_date ON photos(year, month, day, date_taken);`,
		`CREATE INDEX IF NOT EXISTS idx_photos_platform ON photos(platform);`,
		`CREATE INDEX IF NOT EXISTS idx_photos_screenshot ON photos(is_screenshot);`,
		`CREATE INDEX IF NOT EXISTS idx_photos_route_folder ON photos(route_folder);`,
		`CREATE INDEX IF NOT EXISTS idx_photos_show_in_all ON photos(show_in_all);`,
		`CREATE INDEX IF NOT EXISTS idx_photos_favorite ON photos(is_favorite);`,

		// FTS5 Full-Text Search virtual table
		`CREATE VIRTUAL TABLE IF NOT EXISTS photos_fts USING fts5(
			file_name,
			camera_make,
			camera_model,
			canonical_device,
			tags,
			ocr_text,
			ai_description,
			content='photos',
			content_rowid='id'
		);`,

		// Triggers to keep FTS5 synchronized with photos
		`CREATE TRIGGER IF NOT EXISTS photos_ai AFTER INSERT ON photos BEGIN
			INSERT INTO photos_fts(rowid, file_name, camera_make, camera_model, canonical_device, tags, ocr_text, ai_description)
			VALUES (new.id, new.file_name, new.camera_make, new.camera_model, new.canonical_device, new.tags, new.ocr_text, new.ai_description);
		END;`,

		`CREATE TRIGGER IF NOT EXISTS photos_ad AFTER DELETE ON photos BEGIN
			INSERT INTO photos_fts(photos_fts, rowid, file_name, camera_make, camera_model, canonical_device, tags, ocr_text, ai_description)
			VALUES ('delete', old.id, old.file_name, old.camera_make, old.camera_model, old.canonical_device, old.tags, old.ocr_text, old.ai_description);
		END;`,

		`CREATE TRIGGER IF NOT EXISTS photos_au AFTER UPDATE ON photos BEGIN
			INSERT INTO photos_fts(photos_fts, rowid, file_name, camera_make, camera_model, canonical_device, tags, ocr_text, ai_description)
			VALUES ('delete', old.id, old.file_name, old.camera_make, old.camera_model, old.canonical_device, old.tags, old.ocr_text, old.ai_description);
			INSERT INTO photos_fts(rowid, file_name, camera_make, camera_model, canonical_device, tags, ocr_text, ai_description)
			VALUES (new.id, new.file_name, new.camera_make, new.camera_model, new.canonical_device, new.tags, new.ocr_text, new.ai_description);
		END;`,

		// Watched folders
		`CREATE TABLE IF NOT EXISTS watched_folders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			path TEXT NOT NULL UNIQUE,
			route_override TEXT DEFAULT '',
			show_in_all_photos INTEGER DEFAULT 1,
			enabled INTEGER DEFAULT 1,
			last_scanned_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,

		// Device to custom folder mappings
		`CREATE TABLE IF NOT EXISTS device_folder_mappings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			device_name TEXT NOT NULL UNIQUE,
			custom_folder TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,

		// App settings (key-value)
		`CREATE TABLE IF NOT EXISTS app_settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,

		// Google Photos upload registry
		`CREATE TABLE IF NOT EXISTS gphotos_cloud_registry (
			sha256 TEXT PRIMARY KEY,
			remote_id TEXT DEFAULT '',
			album_name TEXT DEFAULT '',
			uploaded_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,

		// Face clusters
		`CREATE TABLE IF NOT EXISTS face_clusters (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			person_name TEXT NOT NULL,
			cover_photo INTEGER DEFAULT 0,
			face_count INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
	}

	for _, q := range queries {
		if _, err := d.writeDB.Exec(q); err != nil {
			return fmt.Errorf("schema migration failed on query:\n%s\nerror: %w", q, err)
		}
	}

	return nil
}

// UpsertPhoto inserts or updates a photo record in SQLite.
func (d *Database) UpsertPhoto(ctx context.Context, p *PhotoRecord) (uint32, error) {
	d.writeMu.Lock()
	defer d.writeMu.Unlock()

	query := `INSERT INTO photos (
		original_path, organized_path, file_name, file_size, sha256,
		width, height, date_taken, year, month, day,
		camera_make, camera_model, canonical_device, platform,
		is_screenshot, screenshot_score, route_folder, show_in_all,
		is_user_modified, rotation, rating, is_favorite,
		tags, ocr_text, ai_description, face_count,
		is_video, duration_sec, live_photo_pair, created_at, updated_at
	) VALUES (
		?, ?, ?, ?, ?,
		?, ?, ?, ?, ?, ?,
		?, ?, ?, ?,
		?, ?, ?, ?,
		?, ?, ?, ?,
		?, ?, ?, ?,
		?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
	) ON CONFLICT(original_path) DO UPDATE SET
		organized_path = excluded.organized_path,
		file_name = excluded.file_name,
		file_size = excluded.file_size,
		sha256 = excluded.sha256,
		width = CASE WHEN photos.is_user_modified = 1 THEN photos.width ELSE excluded.width END,
		height = CASE WHEN photos.is_user_modified = 1 THEN photos.height ELSE excluded.height END,
		date_taken = CASE WHEN photos.is_user_modified = 1 THEN photos.date_taken ELSE excluded.date_taken END,
		year = CASE WHEN photos.is_user_modified = 1 THEN photos.year ELSE excluded.year END,
		month = CASE WHEN photos.is_user_modified = 1 THEN photos.month ELSE excluded.month END,
		day = CASE WHEN photos.is_user_modified = 1 THEN photos.day ELSE excluded.day END,
		camera_make = excluded.camera_make,
		camera_model = excluded.camera_model,
		canonical_device = excluded.canonical_device,
		platform = CASE WHEN photos.is_user_modified = 1 THEN photos.platform ELSE excluded.platform END,
		is_screenshot = CASE WHEN photos.is_user_modified = 1 THEN photos.is_screenshot ELSE excluded.is_screenshot END,
		screenshot_score = excluded.screenshot_score,
		route_folder = CASE WHEN photos.is_user_modified = 1 THEN photos.route_folder ELSE excluded.route_folder END,
		show_in_all = CASE WHEN photos.is_user_modified = 1 THEN photos.show_in_all ELSE excluded.show_in_all END,
		tags = excluded.tags,
		ocr_text = excluded.ocr_text,
		ai_description = excluded.ai_description,
		face_count = excluded.face_count,
		is_video = excluded.is_video,
		duration_sec = excluded.duration_sec,
		live_photo_pair = excluded.live_photo_pair,
		updated_at = CURRENT_TIMESTAMP
	RETURNING id;`

	var id uint32
	err := d.writeDB.QueryRowContext(ctx, query,
		p.OriginalPath, p.OrganizedPath, p.FileName, p.FileSize, p.Sha256,
		p.Width, p.Height, p.DateTaken, p.Year, p.Month, p.Day,
		p.CameraMake, p.CameraModel, p.CanonicalDevice, p.Platform,
		p.IsScreenshot, p.ScreenshotScore, p.RouteFolder, p.ShowInAll,
		p.IsUserModified, p.Rotation, p.Rating, p.IsFavorite,
		p.Tags, p.OcrText, p.AiDescription, p.FaceCount,
		p.IsVideo, p.DurationSec, p.LivePhotoPair,
	).Scan(&id)

	if err != nil {
		return 0, err
	}

	p.ID = id

	// Update in-memory index
	d.MemIndex.UpsertNode(p.ToSlimNode())

	return id, nil
}

// GetPhotoByID fetches a single photo by its database ID.
func (d *Database) GetPhotoByID(ctx context.Context, id uint32) (*PhotoRecord, error) {
	query := `SELECT 
		id, original_path, organized_path, file_name, file_size, sha256,
		width, height, date_taken, year, month, day,
		camera_make, camera_model, canonical_device, platform,
		is_screenshot, screenshot_score, route_folder, show_in_all,
		is_user_modified, rotation, rating, is_favorite,
		tags, ocr_text, ai_description, face_count,
		is_video, duration_sec, live_photo_pair, created_at, updated_at
	FROM photos WHERE id = ?;`

	row := d.readDB.QueryRowContext(ctx, query, id)
	return scanPhotoRecord(row)
}

// GetPhotoByOriginalPath retrieves a photo record by original path.
func (d *Database) GetPhotoByOriginalPath(ctx context.Context, path string) (*PhotoRecord, error) {
	query := `SELECT 
		id, original_path, organized_path, file_name, file_size, sha256,
		width, height, date_taken, year, month, day,
		camera_make, camera_model, canonical_device, platform,
		is_screenshot, screenshot_score, route_folder, show_in_all,
		is_user_modified, rotation, rating, is_favorite,
		tags, ocr_text, ai_description, face_count,
		is_video, duration_sec, live_photo_pair, created_at, updated_at
	FROM photos WHERE original_path = ?;`

	row := d.readDB.QueryRowContext(ctx, query, path)
	return scanPhotoRecord(row)
}

// GetPhotoByHash retrieves a photo record by sha256 hash.
func (d *Database) GetPhotoByHash(ctx context.Context, hash string) (*PhotoRecord, error) {
	query := `SELECT 
		id, original_path, organized_path, file_name, file_size, sha256,
		width, height, date_taken, year, month, day,
		camera_make, camera_model, canonical_device, platform,
		is_screenshot, screenshot_score, route_folder, show_in_all,
		is_user_modified, rotation, rating, is_favorite,
		tags, ocr_text, ai_description, face_count,
		is_video, duration_sec, live_photo_pair, created_at, updated_at
	FROM photos WHERE sha256 = ? LIMIT 1;`

	row := d.readDB.QueryRowContext(ctx, query, hash)
	return scanPhotoRecord(row)
}

// UpdatePhotoUserFields updates user-editable fields and sets is_user_modified = 1.
func (d *Database) UpdatePhotoUserFields(ctx context.Context, id uint32, routeFolder string, platform string, rotation int, rating int, isFavorite bool, showInAll bool) error {
	d.writeMu.Lock()
	defer d.writeMu.Unlock()

	query := `UPDATE photos SET 
		route_folder = ?,
		platform = ?,
		rotation = ?,
		rating = ?,
		is_favorite = ?,
		show_in_all = ?,
		is_user_modified = 1,
		updated_at = CURRENT_TIMESTAMP
	WHERE id = ?;`

	_, err := d.writeDB.ExecContext(ctx, query, routeFolder, platform, rotation, rating, isFavorite, showInAll, id)
	if err != nil {
		return err
	}

	// Update mem index
	d.MemIndex.UpdateUserFields(id, routeFolder, platform, rotation, rating, isFavorite, showInAll)
	return nil
}

// UpdatePhotoAiAndOcr updates tags, OCR text, and AI description.
func (d *Database) UpdatePhotoAiAndOcr(ctx context.Context, id uint32, tags string, ocrText string, aiDesc string) error {
	d.writeMu.Lock()
	defer d.writeMu.Unlock()

	query := `UPDATE photos SET 
		tags = ?,
		ocr_text = ?,
		ai_description = ?,
		updated_at = CURRENT_TIMESTAMP
	WHERE id = ?;`

	_, err := d.writeDB.ExecContext(ctx, query, tags, ocrText, aiDesc, id)
	return err
}

// SearchPhotosFTS searches for matching photo IDs using FTS5 match queries.
func (d *Database) SearchPhotosFTS(ctx context.Context, queryStr string, limit int) ([]uint32, error) {
	if limit <= 0 {
		limit = 100
	}
	// Sanitize and format for FTS5
	cleanQuery := fmt.Sprintf(`"%s"*`, queryStr)
	query := `SELECT rowid FROM photos_fts WHERE photos_fts MATCH ? ORDER BY rank LIMIT ?;`

	rows, err := d.readDB.QueryContext(ctx, query, cleanQuery, limit)
	if err != nil {
		// Fallback to LIKE if FTS expression has syntax errors
		likeQuery := "%" + queryStr + "%"
		fbQuery := `SELECT id FROM photos WHERE file_name LIKE ? OR tags LIKE ? OR ocr_text LIKE ? OR ai_description LIKE ? LIMIT ?;`
		rows, err = d.readDB.QueryContext(ctx, fbQuery, likeQuery, likeQuery, likeQuery, likeQuery, limit)
		if err != nil {
			return nil, err
		}
	}
	defer rows.Close()

	ids := make([]uint32, 0)
	for rows.Next() {
		var id uint32
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanPhotoRecord(row rowScanner) (*PhotoRecord, error) {
	var p PhotoRecord
	var dt string
	var ca, ua string
	var isSS, showAll, isMod, isFav, isVid int

	err := row.Scan(
		&p.ID, &p.OriginalPath, &p.OrganizedPath, &p.FileName, &p.FileSize, &p.Sha256,
		&p.Width, &p.Height, &dt, &p.Year, &p.Month, &p.Day,
		&p.CameraMake, &p.CameraModel, &p.CanonicalDevice, &p.Platform,
		&isSS, &p.ScreenshotScore, &p.RouteFolder, &showAll,
		&isMod, &p.Rotation, &p.Rating, &isFav,
		&p.Tags, &p.OcrText, &p.AiDescription, &p.FaceCount,
		&isVid, &p.DurationSec, &p.LivePhotoPair, &ca, &ua,
	)
	if err != nil {
		return nil, err
	}

	p.IsScreenshot = isSS == 1
	p.ShowInAll = showAll == 1
	p.IsUserModified = isMod == 1
	p.IsFavorite = isFav == 1
	p.IsVideo = isVid == 1

	if parsed, err := time.Parse(time.RFC3339, dt); err == nil {
		p.DateTaken = parsed
	} else if parsed, err := time.Parse("2006-01-02 15:04:05", dt); err == nil {
		p.DateTaken = parsed
	} else if parsed, err := time.Parse("2006-01-02T15:04:05", dt); err == nil {
		p.DateTaken = parsed
	} else {
		p.DateTaken = time.Now()
	}

	return &p, nil
}

// Watched Folder Operations

func (d *Database) GetAllWatchedFolders(ctx context.Context) ([]WatchedFolder, error) {
	rows, err := d.readDB.QueryContext(ctx, `SELECT id, path, route_override, show_in_all_photos, enabled, last_scanned_at, created_at FROM watched_folders ORDER BY id ASC;`)
	if err != nil {
		return make([]WatchedFolder, 0), err
	}
	defer rows.Close()

	result := make([]WatchedFolder, 0)
	for rows.Next() {
		var wf WatchedFolder
		var showInAll, enabled int
		var lastScan sql.NullString
		var ca string

		if err := rows.Scan(&wf.ID, &wf.Path, &wf.RouteOverride, &showInAll, &enabled, &lastScan, &ca); err != nil {
			return nil, err
		}
		wf.ShowInAllPhotos = showInAll == 1
		wf.Enabled = enabled == 1
		if lastScan.Valid {
			wf.LastScannedAt, _ = time.Parse("2006-01-02 15:04:05", lastScan.String)
		}
		result = append(result, wf)
	}
	return result, nil
}

func (d *Database) AddWatchedFolder(ctx context.Context, path string, routeOverride string, showInAll bool) error {
	d.writeMu.Lock()
	defer d.writeMu.Unlock()

	showAllInt := 0
	if showInAll {
		showAllInt = 1
	}

	_, err := d.writeDB.ExecContext(ctx, `INSERT INTO watched_folders (path, route_override, show_in_all_photos, enabled)
		VALUES (?, ?, ?, 1)
		ON CONFLICT(path) DO UPDATE SET
			route_override = excluded.route_override,
			show_in_all_photos = excluded.show_in_all_photos,
			enabled = 1;`, path, routeOverride, showAllInt)
	return err
}

func (d *Database) RemoveWatchedFolder(ctx context.Context, id int) error {
	d.writeMu.Lock()
	defer d.writeMu.Unlock()

	_, err := d.writeDB.ExecContext(ctx, `DELETE FROM watched_folders WHERE id = ?;`, id)
	return err
}

// Device Mapping Operations

func (d *Database) GetAllDeviceMappings(ctx context.Context) ([]DeviceMapping, error) {
	rows, err := d.readDB.QueryContext(ctx, `SELECT id, device_name, custom_folder, created_at FROM device_folder_mappings ORDER BY device_name ASC;`)
	if err != nil {
		return make([]DeviceMapping, 0), err
	}
	defer rows.Close()

	list := make([]DeviceMapping, 0)
	for rows.Next() {
		var m DeviceMapping
		var ca string
		if err := rows.Scan(&m.ID, &m.DeviceName, &m.CustomFolder, &ca); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, nil
}

func (d *Database) SetDeviceMapping(ctx context.Context, deviceName string, customFolder string) error {
	d.writeMu.Lock()
	defer d.writeMu.Unlock()

	_, err := d.writeDB.ExecContext(ctx, `INSERT INTO device_folder_mappings (device_name, custom_folder)
		VALUES (?, ?)
		ON CONFLICT(device_name) DO UPDATE SET custom_folder = excluded.custom_folder;`, deviceName, customFolder)
	return err
}

func (d *Database) DeleteDeviceMapping(ctx context.Context, deviceName string) error {
	d.writeMu.Lock()
	defer d.writeMu.Unlock()

	_, err := d.writeDB.ExecContext(ctx, `DELETE FROM device_folder_mappings WHERE device_name = ?;`, deviceName)
	return err
}

// App Settings Operations

func (d *Database) GetSetting(ctx context.Context, key string, defaultValue string) string {
	var val string
	err := d.readDB.QueryRowContext(ctx, `SELECT value FROM app_settings WHERE key = ?;`, key).Scan(&val)
	if err != nil {
		return defaultValue
	}
	return val
}

func (d *Database) SetSetting(ctx context.Context, key string, val string) error {
	d.writeMu.Lock()
	defer d.writeMu.Unlock()

	_, err := d.writeDB.ExecContext(ctx, `INSERT INTO app_settings (key, value, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP;`, key, val)
	return err
}

// Google Photos Cloud Registry Operations

func (d *Database) IsSyncedToGPhotos(ctx context.Context, sha256 string) bool {
	var count int
	_ = d.readDB.QueryRowContext(ctx, `SELECT count(*) FROM gphotos_cloud_registry WHERE sha256 = ?;`, sha256).Scan(&count)
	return count > 0
}

func (d *Database) MarkSyncedToGPhotos(ctx context.Context, sha256 string, remoteID string, albumName string) error {
	d.writeMu.Lock()
	defer d.writeMu.Unlock()

	_, err := d.writeDB.ExecContext(ctx, `INSERT INTO gphotos_cloud_registry (sha256, remote_id, album_name, uploaded_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(sha256) DO UPDATE SET remote_id = excluded.remote_id, album_name = excluded.album_name;`,
		sha256, remoteID, albumName)
	return err
}

// FactoryReset wipes database tables for clean-slate testing or catalog reset.
func (d *Database) FactoryReset(ctx context.Context) error {
	d.writeMu.Lock()
	defer d.writeMu.Unlock()

	tables := []string{"photos", "watched_folders", "device_folder_mappings", "gphotos_cloud_registry", "face_clusters"}
	for _, t := range tables {
		if _, err := d.writeDB.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s;", t)); err != nil {
			return err
		}
	}
	d.MemIndex.Reset()
	return nil
}
