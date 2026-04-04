# Image Support — Implementation Plan

## Current State

The app is text-only. All clipboard backends return `(string, error)`, the `ClipboardEntry.Content` field is a `string`, and the frontend renders with `{{ entry.content }}`.

- Copying a **file from a file browser** typically shows the file URI/path as text
- Copying a **pure image** (screenshot, image editor) is silently ignored — no crash, no error

## Changes Required by Layer

### 1. Data Model (`business/clipboard/model.go`)

The core bottleneck — `Content` is a `string`. Add:

- `ContentType string` field (e.g., `"text/plain"`, `"image/png"`)
- `ImageData []byte` field (or base64 string) for binary content

### 2. Foundation Clipboard Backends (`foundation/clipboard/`)

The `Reader` interface only has `GetText() (string, error)`. Add:

- A way to **detect** what's on the clipboard (text vs image)
- A `GetImage() ([]byte, string, error)` method (bytes + MIME type)
- Platform-specific implementations:
  - **Wayland**: `wl-paste --type image/png` (already supports MIME targeting)
  - **xclip**: `xclip -selection clipboard -t image/png -o`
  - **xsel**: doesn't support images well — may need xclip fallback
  - **macOS**: `osascript` or `pngpaste`
- Extend the `Writer` interface for writing images back

### 3. Monitor (`business/clipboard/monitor.go`)

- `poll()` currently only calls `GetText()` — needs to first check content type, then read accordingly
- `addEntry()` and `CopyItem()` need to handle both types
- `GenericClipboard` (Wails runtime writer) has no image API — rely on platform CLI tools for writing too

### 4. Sync Transport

- **HTTP handler** (`app/handlers/clipboard.go`): JSON-encodes entries directly. Images as base64 in JSON works but is ~33% larger. Alternative: separate `GET /api/clipboard/{id}/image` endpoint for binary
- **Peer fetcher** (`business/peersclipsync/peersclipsync.go`): Decodes into `[]ClipboardEntry` — needs to handle the new fields
- Bandwidth consideration: syncing images across peers is much heavier than text

### 5. Frontend

- **`ClipboardItem.vue`**: Currently `{{ entry.content }}` — needs conditional rendering: `<img>` tag for images, `<p>` for text
- **Stores** (`clipboard.js`, `remote.js`): `copyItem` and `copyRemoteItem` assume string content — need image-aware copy methods
- **Wails bindings** (`app/app.go`): `CopyRemoteItem(content string)` needs an image variant

### 6. Wails TypeScript Bindings

Auto-generated from Go structs — rebuild after model changes.

## Effort Breakdown

| Layer | Effort |
|---|---|
| Model + interfaces | Small — struct and interface changes |
| Platform backends (4 files) | Medium — each needs image detection + read/write |
| Monitor polling logic | Small-medium |
| Sync transport | Medium — decide on base64-in-JSON vs separate endpoint |
| Frontend rendering | Small — conditional `<img>` vs `<p>` |
| Frontend copy-back | Medium — writing images to clipboard from frontend |

The **platform backends** and **sync transport** are the hardest parts. The rest is plumbing the new content type through existing code.
