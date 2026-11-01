# Lumina Photos V3 — Rules & Architecture Guidelines

## Rule 1: Strict 5-Tier Routing Hierarchy
Every photo or video scanned and organized into the destination directory strictly adheres to the following descending priority order:

1. **Priority 1: Screenshots (`is_screenshot = 1`)**
   - Target Destination: `{Dest}/Screenshots/{YYYY}/{YYYY-MM}/`
   - Strict Visibility: **Strictly hidden** from `All Photos`, `iOS`, `Android`, `Camera`, and `Unknown` tabs. They only appear in the dedicated `📱 Screenshots` tab.
2. **Priority 2: Watched Source Folder Route Override**
   - E.g., if the user assigns a watched folder a custom route `Amruta`, **ALL** non-screenshot files from that folder route directly to `{Dest}/Amruta/{YYYY}/{YYYY-MM}/`.
   - Respects the folder's `show_in_all_photos` toggle (if 0, excluded from `All Photos` view).
3. **Priority 3: Manual User Photo Override (`is_user_modified = 1`)**
   - If a user manually changes a photo's destination folder, person, device, or rotation, set `is_user_modified = 1`.
   - Background re-scans and sweeps will **NEVER** overwrite manual user edits.
4. **Priority 4: Device-to-Folder Mapping**
   - Looked up in `device_folder_mappings` table (`device_name` -> `custom_folder`).
   - Assigns photos from any device (e.g. `Samsung_Galaxy_S23`) to a custom person/owner folder.
5. **Priority 5: Evidence-Only Platform Fallback**
   - If no higher rule matched, route based on verifiable hardware evidence:
     - `iOS/` -> Apple iPhone/iPad metadata detected.
     - `Android/` -> Google, Samsung, Xiaomi, OnePlus, Asus, Motorola, etc. detected.
     - `Camera/` -> Dedicated camera manufacturer (Canon, Nikon, Sony, Fujifilm, Olympus, Panasonic, Leica, GoPro, DJI, Hasselblad) detected.
     - `Unknown/` -> Zero verifiable device maker/model metadata.

---

## Rule 2: Unbreakable Video & HEIC Screenshot Veto
- **Videos** (`.mp4`, `.mov`, `.mkv`, `.avi`, `.m4v`, `.3gp`, `.wmv`) can **NEVER** be classified as screenshots under any circumstance (`Score = -1000`).
- **`.heic` camera files** can **NEVER** be classified as screenshots (`Score = -1000`) unless the original filename explicitly contains `"screenshot"`.

---

## Rule 3: Evidence-Only Device Naming — Zero Guessing
- **Never guess `iPhone`** merely because a file has a `.heic` extension (modern Android devices like Samsung Galaxy S23/S24 shoot HEIC natively).
- **Never guess `Camera`** merely from a 3:2 aspect ratio or `IMG_` prefix.
- **Never read `organized_filename`** when re-evaluating metadata (always read original source/file headers).
- **Never set `device_model = "Screenshot"`** — screenshots inherit the host OS/device if known, or remain empty/desktop.
- Canonical device names are strictly normalized (e.g., `Samsung_SM-S911B` -> `Samsung_Galaxy_S23`, `ASUS_Z01RD` -> `ASUS_ZenFone_5Z`, `Google_Pixel_5_(redfin)` -> `Google_Pixel_5`).

---

## Rule 4: Zero Silent Crashes & Memory Safety
- **Forbidden Packages:** NEVER use `github.com/SeriousBug/webp-go-pure` or `github.com/nfnt/resize` due to memory leaks, slice out-of-bounds panics, and CGo instability.
- **Image Processing:** Use standard library `image`, `image/jpeg`, `image/png`, and `golang.org/x/image/draw` with high-performance `BiLinear` or `CatmullRom` interpolation.
- **Memory Optimization:** Use `sync.Pool` for byte buffers and decompression scratchpads.
- **Fault Tolerance:** Call `debug.SetPanicOnFault(true)` and wrap worker goroutines in structured `recover()` handlers.
- **OS Governance:** Windows worker throttling using `BELOW_NORMAL_PRIORITY_CLASS` so user experience remains silky smooth at 120fps even during 100,000-photo cataloging.
- **Sleep Prevention:** Use Win32 `SetThreadExecutionState(ES_CONTINUOUS | ES_SYSTEM_REQUIRED)` during active scans and uploads so transfers are never severed by Windows entering sleep mode.

---

## Rule 5: Zero Visible Subprocess Windows & Pure-Go Fallbacks
- **Mandatory Silent Helpers:** Never call `os/exec.Command` or `exec.CommandContext` directly anywhere in the codebase. Always use `sys.SilentCommand` / `sys.SilentCommandContext` configured with `HideWindow: true` and `CreationFlags: 0x08000000 (CREATE_NO_WINDOW)`.
- **Zero Console Popping:** When running GUI binaries on Windows (`-H=windowsgui`), spawning unflagged child processes causes visible console/cmd windows to flash and steal keyboard focus. Every external process invocation must be silent and headless.
- **No Per-File Background Process Loops:** Never spawn `powershell.exe` in a per-file loop (e.g. for video thumbnailing, EXIF inspection, or drive detection). Use pure-Go scanners (`os.Open`, atomic parsing) and native Win32 DLL calls (`kernel32.dll`) which run in `< 1ms` with zero process overhead.
- **Automated Verification:** Verified by continuous unit test `internal/sys/TestRule5ZeroRawExecCommands` which fails the build if any raw `exec.Command` call is introduced outside `internal/sys/exec_*.go`.
