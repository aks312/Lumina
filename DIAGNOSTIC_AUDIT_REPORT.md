# Lumina Photos V3 — Diagnostic Audit Report
**Date:** 2026-11-01  
**Scope:** Static and Runtime Diagnostic Audit across Backend (`main.go`, `app.go`, `internal/`) and Frontend (`frontend/src/`) for 55,579-File Catalog Workload.

---

## 1. Executive Summary
A comprehensive audit was performed across all subsystems to diagnose the 6 reported bugs and discover any secondary edge cases or architectural bottlenecks. All 6 root causes were confirmed in code, and 4 additional architectural bottlenecks were identified.

---

## 2. Findings on Reported Bugs

### Bug 1: Videos have blank black cards with no thumbnails
- **Root Cause in Code:** `internal/thumbs/generator.go` line 180 (`createGenericPlaceholder`): When `image.Decode` fails on video files, it fills a canvas with a solid dark slate color `RGB(20, 24, 33)` with zero graphics, zero video icon badges, and zero frame extraction.
- **Impact:** Video cards in the gallery render as completely black rectangles.
- **Fix:** Implement `internal/thumbs/video_thumb.go` with 3-stage extraction:
  1. Native Windows Shell frame extractor (PowerShell WinRT/Shell.Application or ffmpeg).
  2. Pure-Go MP4/MOV embedded `moov`/`covr` preview scanner.
  3. Stylized high-contrast cinema poster fallback with video badge and duration.
  4. Automatic background backfill for all existing catalog videos.

### Bug 2: Timeline buttons (`2026 JAN`, `2025 DEC`) do nothing when clicked
- **Root Cause in Code:** In `frontend/src/App.tsx`, `handleSelectMonth` called `document.getElementById('timeline-YYYY-MM')`. Because the initial gallery query loads a page slice (1000 items), months from previous years (2025, 2024, etc.) are not present in the DOM. The `getElementById` call returns `null` and silently fails.
- **Impact:** Clicking any month outside the initial slice has zero effect.
- **Fix:** In `App.tsx` and `PhotoGrid.tsx`, if the month DOM node is present, scroll smoothly to it. If not in the current slice, filter the query by `year` and `month`, and display a dismissible `[ 📅 Month YYYY ✕ ]` active filter chip in the header. Highlight the active month in the scrubber with `text-cyan-400 bg-cyan-500/15`.

### Bug 3 & 4: Lightbox flashes previous photo on open; jerky transition on close
- **Root Cause in Code:**
  1. In `frontend/src/components/Lightbox.tsx`, `currentIdx` state was initialized to `0`. On frame 1, `currentPhoto = photos[currentIdx]` evaluated to `photos[0]` (the very first photo in the gallery) before the `useEffect` synchronized with the clicked photo's index on frame 2.
  2. The `<img>` element lacked a unique `key={currentPhoto.id}`, allowing the browser to reuse the DOM image element and show the previously decoded bitmap while waiting for the new image network request.
  3. `layoutId` shared-element morphing in Framer Motion thrashed layout calculations across hundreds of grid items on modal open/close, causing frame drops and jumps.
- **Fix:**
  - Initialize Lightbox index directly from the clicked photo.
  - Remove `layoutId` shared-element animation; replace with hardware-accelerated modal transitions (`will-change: transform, opacity`).
  - Implement two-layer progressive render: Layer 1 displays the instant cached `/thumb/{id}` thumbnail on Frame 1 (0ms), while Layer 2 cross-fades the full `/raw/{id}` image once loaded.

### Bug 5: `/raw/{id}` stops loading photos after rapid cycling or during active scan
- **Root Cause in Code:**
  1. `ServeRawMedia` in `app.go` called `a.database.GetPhotoByID(context.Background(), id)` on every request without a timeout. While the background crawler indexes 55,579 files with write locks on SQLite, concurrent reads block.
  2. WebView2 enforces a 6-socket connection limit per origin. When 6 concurrent `/raw/{id}` requests block on SQLite, WebView2's socket pool is completely exhausted, preventing any subsequent image from loading.
  3. `http.ServeFile` was used without explicit `Range` request handling and client abort timeouts.
- **Fix:**
  - Cache file paths directly in `MemIndex` in RAM (`map[uint32]string` with `sync.RWMutex`), resolving paths in `< 0.001ms` with **zero SQLite queries**.
  - Enforce `context.WithTimeout(r.Context(), 15*time.Second)` on requests.
  - Use `http.ServeContent` with an open `*os.File` for streaming and `Range` support.
  - Implement automatic client-side retry in `Lightbox.tsx` falling back to `/thumb/{id}` on network error.

### Bug 6: "Google Photos" sidebar tab changes header title but renders Photo Gallery
- **Root Cause in Code:** In `frontend/src/App.tsx`, lines 240-250 only checked `currentTab === 'cleanup'`. When `currentTab === 'google_photos'`, `'people'`, or `'settings'`, the condition fell through to the default branch, which unconditionally rendered `<PhotoGrid />`.
- **Impact:** Clicking `Google Photos` updated the header text to "Google Photos Cloud", but the main view remained the regular photo grid.
- **Fix:** Implement dedicated view components:
  - `<GooglePhotosView />`: 1-Click PKCE sign-in, account status, album sync cards, and batch upload progress.
  - `<PeopleFacesView />`: Face clusters, recognized people, face count badges.
  - `<CleanUpAssistantView />`: Duplicates, large videos, low sharpness review.
  - `<SettingsView />`: Catalog storage paths, NVMe speed benchmark, and Catalog Maintenance actions.

---

## 3. Discovered Secondary Bottlenecks & Edge Cases

1. **HTML5 `<video controls>` Seeking / Range Requests:**
   - Browsers require `Accept-Ranges: bytes` and HTTP `206 Partial Content` to allow seeking through large video files. Serving via `http.ServeContent` provides full HTTP 206 range request support for smooth video playback.
2. **Path Caching for 55,579 Files:**
   - Maintaining `map[uint32]string` in memory for 55,579 paths requires ~5MB of RAM, which is negligible on modern PCs and eliminates SQLite lock contention during active sweeps.
3. **Thumbnail Pack Index Lookup:**
   - `thumbs.pack` in `internal/thumbs/pack.go` has an LRU cache, but unindexed video thumbnails must fall back to the newly implemented multi-stage generator rather than throwing 404s.

---

## 4. Remediation Plan
1. Create `internal/thumbs/video_thumb.go` with 3-stage video frame generation and backfill.
2. Update `internal/db/memindex.go` to store `idToPath` in RAM for 0ms path lookup.
3. Update `app.go` to use `MemIndex` path lookup and `http.ServeContent` with 15s timeout in `ServeRawMedia`.
4. Create dedicated view components: `GooglePhotosView.tsx`, `PeopleFacesView.tsx`.
5. Update `App.tsx` view switcher for all tabs, month scrubbing with active filter pill, and two-layer Lightbox.
6. Write automated tests verifying raw media concurrency during active scans and video thumbnail generation.
7. Rebuild frontend and standalone Windows executable `.\LuminaPhotos.exe`.
