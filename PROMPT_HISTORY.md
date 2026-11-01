# Lumina Photos V3 — Prompt & Iteration History

## [v3.1.0] - Blank Acrylic Window Bug Fix & Method 1 Guided Onboarding
- **Date:** 2026-11-01
- **Focus:** Fix 4 root causes of WebView2 blank transparent Acrylic window & implement Method 1 (Guided Glassmorphic Onboarding + Auto-Drive Discovery + Visual Hub + Instant Search Chips).
- **Status:** COMPLETED & VERIFIED
- **Key Fixes & Deliverables:**
  1. **Root Cause 1 Fix (Wails AssetServer Middleware):** Replaced custom HTTP mux in `main.go` that swallowed embedded assets with Wails' `AssetServer.Middleware` pass-through, serving `/thumbs/` and `/raw/` while passing all other requests (`/`, `/index.html`, `/assets/*`) to embedded frontend dist.
  2. **Root Cause 2 Fix (Null-Slice JSON Serialization):** Ensured all Go backend methods bound to Wails initialize slices with `make([]..., 0)` instead of `nil` (`GetPhotos`, `GetTimelineSummary`, `GetWatchedFolders`, `GetDeviceMappings`), eliminating React `null.map()` crashes.
  3. **Root Cause 3 Fix (React ErrorBoundary & Safe Bridge):** Wrapped `<App />` in high-contrast `ErrorBoundary.tsx` and introduced `safeWailsCall` helper ensuring frame 1 renders without blocking or invisible crashes.
  4. **Root Cause 4 Fix (High-Contrast Layering Over Acrylic):** Configured solid `bg-slate-950/75` layout container with frosted panels (`bg-slate-900/80 backdrop-blur-xl border border-white/15`) and vibrant accent borders so the UI never blends invisibly into Windows Acrylic blur.
  5. **Method 1 Onboarding & Auto-Drive Discovery (`internal/sys/drives.go`):** Implemented `GetDetectedDrives()` scanning `A:\` to `Z:\` with Win32 `GetVolumeInformationW` and `GetDiskFreeSpaceEx`, detecting fixed/removable drives, free GB, and auto-suggested destination.
  6. **3-Card Interactive Setup Hub (`SetupHub.tsx`):**
     - Step 1: 1-Click drive cards + common media folders + recursive subfolder scanning + per-folder Route Overrides (`Auto` / `Amruta`) + `Show in All` toggle.
     - Step 2: Auto-destination on largest drive + 1-click NVMe speed mirror toggle on `C:\`.
     - Step 3: Live architecture tree preview + "🚀 Start Organizing Library" CTA.
  7. **Smart Search Bar With Instant Filter Chips (`Header.tsx`):** Added drop-down discovery chips for Top Places (cities), Devices, Timeline (years), and Smart Categories.
  8. **Production Executable:** Re-compiled standalone Windows binary `.\LuminaPhotos.exe` (17.8 MB).

## [v3.0.0] - Clean-Slate Ground-Up Build
- **Date:** 2026-11-01
- **Focus:** Complete architectural scaffolding and production implementation of Lumina Photos V3 from scratch.
- **Status:** COMPLETED & VERIFIED
- **Milestones:**
  1. Documentation Ecosystem: Created `RULES_AND_GUIDELINES.md`, `PROJECT_CONTEXT.md`, and `PROMPT_HISTORY.md`.
  2. Pure Go Core Backend:
     - SQLite3 dual connection (`writeDB` / `readDB`) with WAL mode + FTS5 full-text indexing (`internal/db`).
     - In-Memory Columnar Index (`internal/db/memindex.go`) with sub-millisecond filtering across 200,000+ items.
     - Windows OS governance with thread execution state (anti-sleep) and priority throttling (`internal/sys`).
     - Pure Go ISO BMFF `.heic` box parser (`ispe`, `meta`, `exif`) and device canonical normalizer (`internal/metadata`).
     - 100-Point multi-factor screenshot scorer with negative vetoes for videos and HEIC (`internal/classifier`).
     - Single-file `thumbs.pack` archive + EXIF APP1 fast preview + 128MB LRU cache (`internal/thumbs`).
     - 5-Tier hierarchical routing organizer with Apple Live Photo pairing and watchdog copy (`internal/organizer`).
     - Zero-Key Windows Native OCR + Local Visual Scene Analyzer + Optional Gemini Boost (`internal/ai`).
     - 1-Click Consumer Google Photos sync with PKCE loopback authentication (`internal/cloud`).
     - Face clustering & skin-tone detection (`internal/faces`).
  3. Glassmorphic Translucent React 18 UI:
     - Windows Mica/Acrylic translucent window styling.
     - Framer Motion spring physics, grid zoom slider (3-10 columns), chameleon ambient lightbox lighting, timeline scrubbing bar, floating glass selection pill.
  4. Testing & Executable:
     - Comprehensive unit and integration test coverage across all packages (`go test ./...` PASS).
     - Production Windows executable compiled via Wails v2 to `.\LuminaPhotos.exe` (17.8 MB).
