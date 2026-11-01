# Lumina Photos V3 — Project Context & Architecture Blueprint

## System Overview
Lumina Photos V3 is a high-performance Windows desktop photo organizer, 120fps hardware-accelerated photo/video viewer, and zero-key local AI asset manager.

- **Backend:** Go 1.23+ with Wails v2.17+
- **Database:** Pure Go SQLite3 (`modernc.org/sqlite`) with WAL mode, FTS5 full-text search, and dual-connection architecture (`writeDB` with MaxOpenConns=1 for atomic serialization; `readDB` with MaxOpenConns=NumCPU for non-blocking concurrent queries).
- **In-Memory Index:** `memindex.go` holding contiguous `[]SlimPhotoNode` in memory for `<0.5ms` tab switches, filtering, and timeline scrubbing across 25,000–200,000+ photos. Includes O(1) in-memory fast path resolution (`FastLookup`) enabling 0ms raw media and thumbnail serving with zero SQLite locking.
- **Thumbnail Store:** Single `thumbs.pack` archive + offset index + EXIF APP1 fast thumbnail extractor + 128MB LRU cache to eliminate NTFS file descriptor storms. Features a pure-Go 3-stage visual video thumbnail pipeline (`video_thumb.go`: camera sidecars + embedded MP4 cover atoms -> silent FFmpeg with `CREATE_NO_WINDOW` -> pure-Go cinema poster fallback; zero PowerShell subprocesses).
- **Silent Subprocesses (Rule 5):** `internal/sys/exec_windows.go` providing `sys.SilentCommand` and `sys.SilentCommandContext` with `CREATE_NO_WINDOW (0x08000000)` and `HideWindow: true`, completely preventing terminal window flashes or keyboard focus theft across the entire app.
- **Metadata Engine:** Pure Go binary ISO BMFF parser for `.heic` box inspection (`ftyp`, `meta`, `ispe`, `exif`) and native EXIF TIFF parser for camera metadata.
- **Screenshot Classifier:** 100-point multi-factor heuristic engine with strict vetoes, digital UI scanline uniformity, and viewport resolution matching.
- **Local AI & OCR:** Windows Native WinRT OCR engine via zero-setup offline runner + Local Visual Scene Analyzer (dominant colors, luminance, skin-tone face detection, foliage/sky gradient, text density).
- **Cloud Sync:** 1-click Google Photos OAuth with local loopback callback (`127.0.0.1:8989`), batch uploader (50 photos/chunk), album routing, dedicated Google Photos glassmorphic view (`GooglePhotosView.tsx`), and tag/caption embedding in EXIF.
- **Frontend & Bridge:** React 18 + TypeScript + TailwindCSS + Framer Motion spring physics + Lucide Icons. Translucent Acrylic/Mica glassmorphism with high-contrast panel layering, Chameleon ambient lighting, 2-layer progressive Lightbox (0ms thumbnail base + high-res master cross-fade), timeline scrubbing with Month/Year filtering pills, floating action pill, ErrorBoundary, safe Wails runtime wrapper, Method 1 3-card Setup Hub with automatic drive detection, People/Faces view (`PeopleFacesView.tsx`), and instant discovery search chips.
- **Wails Asset Server:** Custom middleware pass-through (`AssetServer.Middleware`) intercepting `/thumb/`, `/thumbs/`, and `/raw/` with range requests (`http.ServeContent`, `Accept-Ranges: bytes`), non-blocking RAM path lookups, and clean socket closures while cleanly delegating all other routes (`/`, `/index.html`, `/assets/*`) to the embedded React frontend.

---

## Database Schema (SQLite3 + WAL)
- `photos`: Core asset records (paths, hashes, dimensions, timestamps, routing, tags, rotation, rating, favorite).
- `photos_fts`: FTS5 virtual table for lightning search across filenames, tags, OCR text, camera, and device names.
- `watched_folders`: Monitored source directories with route overrides, subfolder recurrence, and `show_in_all_photos` flag.
- `device_folder_mappings`: Device canonical name to custom folder/person mappings.
- `app_settings`: Key-value configuration for primary destination, mirror destination, fast drive, cloud credentials, AI keys.
- `gphotos_cloud_registry`: Sha256 hashes of items successfully synced to Google Photos to prevent re-uploads.
- `face_clusters`: Detected human faces, bounding boxes, and assigned person labels.

---

## Active Subsystems & Directory Tree
```
Lumina/
├── RULES_AND_GUIDELINES.md
├── PROJECT_CONTEXT.md
├── PROMPT_HISTORY.md
├── wails.json
├── main.go
├── app.go
├── internal/
│   ├── db/              # SQLite3 + In-Memory Columnar Index
│   ├── sys/             # Windows OS governance, priority, power state, NVMe benchmark
│   ├── metadata/        # ISO BMFF HEIC parser, EXIF reader, device normalizer
│   ├── classifier/      # 100-pt Screenshot scorer & Live Photo pairing
│   ├── thumbs/          # thumbs.pack single archive + EXIF APP1 preview + LRU cache
│   ├── organizer/       # 5-Tier routing engine, collision resolver, fast copy
│   ├── indexer/         # File scanner, watchdog, large video hashing (<10ms)
│   ├── ai/              # Zero-Key Windows OCR, local visual scene analyzer, Gemini boost
│   ├── cloud/           # 1-Click Google Photos PKCE sync & album manager
│   └── faces/           # Color/Skin-tone clustering & face region bounding
└── frontend/            # Glassmorphic React 18 + TS + Tailwind + Framer Motion
```
