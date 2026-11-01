package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// PhotoRecord represents a comprehensive photo/video asset in Lumina.
type PhotoRecord struct {
	ID              uint32    `json:"id"`
	OriginalPath    string    `json:"original_path"`
	OrganizedPath   string    `json:"organized_path"`
	FileName        string    `json:"file_name"`
	FileSize        int64     `json:"file_size"`
	Sha256          string    `json:"sha256"`
	Width           int       `json:"width"`
	Height          int       `json:"height"`
	DateTaken       time.Time `json:"date_taken"`
	Year            int       `json:"year"`
	Month           int       `json:"month"`
	Day             int       `json:"day"`
	CameraMake      string    `json:"camera_make"`
	CameraModel     string    `json:"camera_model"`
	CanonicalDevice string    `json:"canonical_device"`
	Platform        string    `json:"platform"` // ios, android, camera, unknown, custom
	IsScreenshot    bool      `json:"is_screenshot"`
	ScreenshotScore int       `json:"screenshot_score"`
	RouteFolder     string    `json:"route_folder"`
	ShowInAll       bool      `json:"show_in_all"`
	IsUserModified  bool      `json:"is_user_modified"`
	Rotation        int       `json:"rotation"` // 0, 90, 180, 270
	Rating          int       `json:"rating"`   // 0-5
	IsFavorite      bool      `json:"is_favorite"`
	Tags            string    `json:"tags"` // comma-separated or json
	OcrText         string    `json:"ocr_text"`
	AiDescription   string    `json:"ai_description"`
	FaceCount       int       `json:"face_count"`
	IsVideo         bool      `json:"is_video"`
	DurationSec     float64   `json:"duration_sec"`
	LivePhotoPair   string    `json:"live_photo_pair"` // path to paired .mov or .heic
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// WatchedFolder represents a directory monitored for changes.
type WatchedFolder struct {
	ID              int       `json:"id"`
	Path            string    `json:"path"`
	RouteOverride   string    `json:"route_override"`
	ShowInAllPhotos bool      `json:"show_in_all_photos"`
	Enabled         bool      `json:"enabled"`
	LastScannedAt   time.Time `json:"last_scanned_at"`
	CreatedAt       time.Time `json:"created_at"`
}

// DeviceMapping maps a canonical hardware device to a custom folder or person.
type DeviceMapping struct {
	ID           int       `json:"id"`
	DeviceName   string    `json:"device_name"`
	CustomFolder string    `json:"custom_folder"`
	CreatedAt    time.Time `json:"created_at"`
}

// AppSetting stores arbitrary configuration.
type AppSetting struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

// FaceCluster represents a recognized face identity.
type FaceCluster struct {
	ID         int       `json:"id"`
	PersonName string    `json:"person_name"`
	CoverPhoto uint32    `json:"cover_photo"`
	FaceCount  int       `json:"face_count"`
	CreatedAt  time.Time `json:"created_at"`
}

// Database encapsulates the dual SQLite connection architecture:
// writeDB: MaxOpenConns=1 for atomic serialized writes
// readDB: MaxOpenConns=NumCPU for high-concurrency non-blocking reads
type Database struct {
	writeDB  *sql.DB
	readDB   *sql.DB
	dbPath   string
	writeMu  sync.Mutex
	MemIndex *MemIndex
}

// Open initializes the SQLite database with WAL mode and returns the Database instance.
func Open(dbDir string) (*Database, error) {
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	dbPath := filepath.Join(dbDir, "lumina_v3.db")
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=cache_size(-65536)&_pragma=foreign_keys(ON)", filepath.ToSlash(dbPath))

	// Open write connection (strictly 1 connection to prevent lock contention)
	writeDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open writeDB: %w", err)
	}
	writeDB.SetMaxOpenConns(1)

	// Open read connection pool
	readDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		_ = writeDB.Close()
		return nil, fmt.Errorf("failed to open readDB: %w", err)
	}
	numCPU := runtime.NumCPU()
	if numCPU < 2 {
		numCPU = 2
	}
	readDB.SetMaxOpenConns(numCPU)
	readDB.SetMaxIdleConns(numCPU)

	db := &Database{
		writeDB:  writeDB,
		readDB:   readDB,
		dbPath:   dbPath,
		MemIndex: NewMemIndex(),
	}

	if err := db.initSchema(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	// Warm the in-memory index from DB
	if err := db.WarmMemIndex(); err != nil {
		// Non-fatal, but log/return
		fmt.Printf("Warning: WarmMemIndex returned %v\n", err)
	}

	return db, nil
}

// Close gracefully closes both write and read connections.
func (d *Database) Close() error {
	var err1, err2 error
	if d.writeDB != nil {
		err1 = d.writeDB.Close()
	}
	if d.readDB != nil {
		err2 = d.readDB.Close()
	}
	if err1 != nil {
		return err1
	}
	return err2
}

// WriteDB returns the write database handle.
func (d *Database) WriteDB() *sql.DB {
	return d.writeDB
}

// ReadDB returns the read database handle.
func (d *Database) ReadDB() *sql.DB {
	return d.readDB
}

// GetDBPath returns the underlying database file path.
func (d *Database) GetDBPath() string {
	return d.dbPath
}
