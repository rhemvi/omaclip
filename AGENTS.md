# Omaclip — Project Instructions

## Overview

A Wails desktop clipboard manager that tracks clipboard history and syncs across multiple computers on the local network.

## Tech Stack

- **Backend**: Go (Wails v3)
- **Frontend**: Vue 3 + Vite + Pinia
- **CSS**: TailwindCSS only — no other CSS frameworks, no custom CSS classes outside of Tailwind utilities
- **Peer Discovery**: mDNS via `hashicorp/mdns` (`_omaclip._tcp` service type)
- **Sync Transport**: HTTPS with TLS — a CA cert is derived from the passphrase and used to sign a leaf cert; peers validate against this CA (no `InsecureSkipVerify`)

## Features

1. **Clipboard history** — continuously poll/monitor the system clipboard and store a history of copied items (text and PNG images)
2. **History UI** — browse and re-copy from clipboard history
3. **Multi-machine sync** — peers discover each other via mDNS and pull each other's clipboard over HTTPS
4. **Omarchy theming** — automatically reads the active Omarchy color theme and applies it to the UI; colors update live when the theme changes
5. **Remote clipboard disable** — `--remote-clipboards-disable` flag skips networking entirely for single-machine use

## Clipboard History

- In-memory only (no persistence across restarts)
- Default max: 50 items, configurable via `OMACLIP_CLIPBOARD_MAX_HISTORY`
- Items can be pinned to the top of the list (local tab only, in-memory only); default max: 3 pins, configurable via `OMACLIP_CLIPBOARD_MAX_PINNED`. Exceeding the cap auto-unpins the oldest pin (FIFO).
- Supports text and images (max 25 MB per image); content type is either `"text"` or `"image"`
- Images are stored in their original format as base64 in `ClipboardEntry.ImageData`; `ImageMimeType` carries the original MIME type. Non-PNG images are converted to PNG on copy-out where supported.
- `ClipboardEntry.Checksum` stores SHA-256 of exact text bytes or raw stored image bytes at ingestion; image hashing also detects local clipboard changes.
- Wayland-aware: uses `wl-paste`/`wl-copy` if available; falls back through xclip → xsel → macOS (`osascript`+`pbpaste` both required)

## Multi-Machine Sync (mDNS + HTTPS)

- Each instance starts an HTTPS server on all IPv4 interfaces (`0.0.0.0`); the port is configurable and defaults to a random OS-assigned port
- TLS: a CA cert is derived from the passphrase key bytes and used to sign a leaf cert; peers validate against this CA — no `InsecureSkipVerify`
- mDNS advertises `version=2` and `ph=<first 16 hex chars of Argon2id passphrase hash>`; discovery requires both matching protocol and passphrase. All machines must upgrade together; no legacy fallback, including manual peers.
- All endpoints require `X-Omaclip-Pass`, validated with `subtle.ConstantTimeCompare`.
- `GET /api/clipboard/checksum` returns the precomputed advertised-history fingerprint. The monitor rebuilds it after history changes from ordered IDs, content checksums, types, MIME types, and timestamps, limited to the configured shared window and excluding rejected images.
- On change, `GET /api/clipboard` returns a payload-free manifest with its own fingerprint; `GET /api/clipboard/{id}/content` returns exact text or raw image bytes only when missing from the receiver's per-peer cache.
- Payload identity is `(contentType, checksum)`; receivers verify hashes and publish only complete snapshots. Failures retain the previous visible snapshot and successful pending downloads; later polls can supersede pending snapshots without sender-side retention.
- Cache state retains only the visible/latest pending snapshots and is released when a peer disappears.
- Peers expire after ~3 missed browse cycles (~6s)

## Passphrase

- Minimum 8 chars, maximum 128 chars, no leading/trailing whitespace
- Stored in `$HOME/.config/omaclip/config.json` (mode 0600) as `{ "passphrase": "..." }`
- Loaded at startup; if invalid, the app logs an error and exits immediately
- On first launch (no config), the UI shows a passphrase setup screen before enabling networking
- If `DisableRemoteClipboards` is true, passphrase is never required
- The passphrase is the single security boundary: it derives the TLS CA cert, is hashed (Argon2id) into mDNS TXT records for peer filtering, and is validated on every HTTP request

## Omarchy Theming

- Reads `$HOME/.config/omarchy/current/theme/colors.toml`; gracefully skips if missing
- Watches file for changes (fsnotify, 200ms debounce) and emits `theme:loaded` Wails event
- All UI colors come from Omarchy theme tokens via CSS custom properties (`--color-*`); no hardcoded colors

## App Behavior

- Normal window (no system tray)
- No global hotkey yet
- Wails events: `clipboard:new`, `remote:updated`, `theme:loaded`

## Architecture Notes

- `app/app.go` — main Wails service; exposes Go methods through generated service bindings
- `app/routes.go` — registers HTTP routes on the sync server
- `app/handlers/` — HTTP handlers (clipboard endpoint)
- `business/clipboard/` — clipboard monitor, history, reader/writer interfaces
- `business/passphrase/` — thread-safe passphrase store + validation
- `business/sync/` — HTTPS server wrapper
- `business/peersclipsync/` — periodic fetcher of remote peer clipboards
- `business/theme/` — Omarchy theme loader and file watcher
- `foundation/clipboard/` — platform clipboard backends (Wayland + Wails runtime)
- `foundation/config/` — JSON config file read/write
- `foundation/mdns/` — mDNS registration and peer discovery
- `foundation/tlscert/` — self-signed TLS certificate generation
- Frontend Vue stores (Pinia): `clipboard.js`, `remote.js`, `theme.js`, shared `navigation.js` composable

## CLI Flags

All configurable via environment variables (`OMACLIP_<FLAG>`) or command-line args:

| Flag | Default | Description |
|------|---------|-------------|
| `OMACLIP_DEBUG` | `false` | Enable debug-level logging |
| `OMACLIP_CLIPBOARD_MAX_HISTORY` | `50` | Max local clipboard entries |
| `OMACLIP_CLIPBOARD_MAX_PINNED` | `3` | Max number of clipboard items that can be pinned at the top of the list |
| `OMACLIP_CLIPBOARD_MAX_PNG_IMAGE_MB` | `5` | Max PNG image size (MB) to accept into clipboard history |
| `OMACLIP_CLIPBOARD_MAX_NON_PNG_IMAGE_MB` | `2` | Max non-PNG image size (MB) to accept (e.g. JPEG from file managers or macOS clipboard) |
| `OMACLIP_CLIPBOARD_POLL_INTERVAL` | `2s` | Local clipboard poll frequency (fallback when watching is not supported) |
| `OMACLIP_REMOTE_CLIPBOARDS_MAX_HISTORY` | `5` | Max local entries transmitted to remote peers |
| `OMACLIP_REMOTE_CLIPBOARDS_POLL_INTERVAL` | `2s` | Peer fetch frequency |
| `OMACLIP_REMOTE_CLIPBOARDS_DISABLE` | `false` | Disable remote sync entirely |
| `OMACLIP_PEERS_POLL_INTERVAL` | `2s` | mDNS browse frequency |
| `OMACLIP_PEERS_MDNS_INTERFACE` | `` | Bind mDNS discovery and advertisement to a specific network interface (e.g. `wlan0`) |
| `OMACLIP_SYNC_SERVER_PORT` | `0` | Port the clipboard sync HTTPS server listens on; `0` uses an OS-assigned port |
| `OMACLIP_THEME_COLOR_PATH` | `~/.config/omarchy/current/theme/colors.toml` | Omarchy theme file path |
| `OMACLIP_CONFIG_PATH` | `~/.config/omaclip/config.json` | Config file path |

## Code Style

- No inline comments on every line — comments only on top of functions
- No custom CSS — Tailwind utility classes only
- Keep Go packages small and focused
