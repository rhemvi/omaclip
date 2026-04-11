# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.3.2] - 2026-04-11

### Added

- WebP, BMP, and TIFF image decoding support via `golang.org/x/image`; these formats are now correctly converted to PNG on copy-out
- Rejected image entries are now highlighted on keyboard navigation and mouse hover, matching the selection style of regular entries

### Fixed

- Copy-out now falls back to the original MIME type when PNG conversion is not supported (e.g. AVIF, HEIC) instead of incorrectly labelling the data as `image/png`
- `toPNG` now logs a warning when image format decoding fails instead of silently returning unconverted data

## [0.3.1] - 2026-04-10

### Fixed

- Remote sync endpoint now returns up to `MaxHistory` valid entries even when rejected image entries are present in history; previously rejected entries consumed slots in the limit, causing fewer entries to be returned to peers

## [0.3.0] - 2026-04-10

### Changed

- Images are now stored in their original format (JPEG, PNG, etc.) instead of being normalised to PNG on ingestion, significantly reducing memory usage and eliminating the ~7s conversion delay for large images
- On copy-out, non-PNG images are converted to PNG for clipboard compatibility using fast compression (`png.BestSpeed`); PNG images are returned as-is with no conversion
- Replaced single `OMACLIP_CLIPBOARD_MAX_IMAGE_MB` with two separate limits:
  - `OMACLIP_CLIPBOARD_MAX_PNG_IMAGE_MB` (default 5) for PNG images from the compositor (screenshots, browser copies)
  - `OMACLIP_CLIPBOARD_MAX_NON_PNG_IMAGE_MB` (default 2) for compressed formats like JPEG from file managers or macOS clipboard

### Added

- Rejected images now appear as warning entries in the clipboard history so the user knows why an image was not stored
- Rejected image entries are not copyable (keyboard shortcuts skip them) and are excluded from remote sync

### Fixed

- Sync endpoint now serves the actual image MIME type instead of hardcoding `image/png`
- Peer fetcher now populates `ImageMimeType` from the HTTP response `Content-Type` header
- Re-copying an image from omaclip no longer triggers a spurious "exceeds size limit" warning

## [0.2.1] - 2026-04-10

### Changed

- Refactored clipboard type parsing to use `strings.SplitSeq` iterator instead of `strings.Split`, avoiding intermediate slice allocation
- Max image size extracted from a hardcoded constant to the configuration struct, making it configurable via `OMACLIP_CLIPBOARD_MAX_IMAGE_MB` (default lowered from 25 MB to 5 MB)
- Log a warning when an image is rejected for exceeding the size limit
- Updated Go dependencies to latest versions

## [0.2.0] - 2026-04-10

### Added

- `--peers-mdns-interface` flag to bind mDNS to a specific network interface, useful for multi-homed hosts (contributed by [@K53N0](https://github.com/K53N0))
- Broader image type support: clipboard now detects images copied from file managers (PNG, JPEG, GIF, BMP, WebP, TIFF) in addition to raw PNG (contributed by [@K53N0](https://github.com/K53N0))
- mDNS registration now logs the advertised IPs for easier network debugging (contributed by [@ntavelis](https://github.com/ntavelis))

### Changed

- Replaced `hashicorp/mdns` with `grandcat/zeroconf` for more reliable peer discovery (contributed by [@K53N0](https://github.com/K53N0))
- Refactored `startNetworking` error handling: the function now returns an error instead of calling `os.Exit` directly, keeping exit logic centralised in the startup function (contributed by [@ntavelis](https://github.com/ntavelis))
- Simplified mDNS browse logic: IPv6 fallback removed (IPv4-only traffic enforced); peers with no IPv4 address are skipped rather than stored with an empty address (contributed by [@ntavelis](https://github.com/ntavelis))
- Updated Go dependencies to latest versions (contributed by [@ntavelis](https://github.com/ntavelis))
- macOS clipboard backend consolidated: dropped the text-only pbpaste fallback; Darwin now requires both `osascript` and `pbpaste`, combining full image and text support in a single `DarwinClipboard` implementation (contributed by [@ntavelis](https://github.com/ntavelis))
- Clipboard images are now normalised to PNG at ingestion using `image.Decode` + `png.Encode`, ensuring consistent format across all backends and correct behaviour when pasting into browsers (contributed by [@ntavelis](https://github.com/ntavelis))
- `ClipboardEntry` now carries `imageMimeType` so the frontend renders images with the correct MIME type in data URLs (contributed by [@ntavelis](https://github.com/ntavelis))
- `SetImage` across all clipboard backends now accepts a `mimeType` parameter and advertises the correct type to the clipboard (contributed by [@ntavelis](https://github.com/ntavelis))
- Shared clipboard utilities (`isImageFile`, `clipboardTypes`, `parseClipboardTypes`) extracted to a common file used by all backends (contributed by [@ntavelis](https://github.com/ntavelis))

### Fixed

- macOS clipboard text detection corrected; cross-platform image file copy support improved (contributed by [@K53N0](https://github.com/K53N0))
- Sentinel errors added for mDNS registration and browse failures for cleaner error handling (contributed by [@K53N0](https://github.com/K53N0))

## [0.1.4] - 2026-04-07

### Fixed

- AUR PKGBUILD now uses versioned filenames for architecture-specific binaries to prevent stale yay cache across upgrades

## [0.1.3] - 2026-04-07

### Fixed

- AUR package checksums and `pkgrel` corrected

## [0.1.2] - 2026-04-07

### Added

- AUR package `omaclip-bin` for easy installation on Arch Linux

### Fixed

- webkit2gtk package names corrected in install instructions for Arch Linux and Alpine

## [0.1.1] - 2026-04-06

### Fixed

- VCS short hash now correctly embedded in the binary version string (`--version` now shows `vX.Y.Z+shortHash` as intended)

## [0.1.0] - 2026-04-06

### Changed

- Renamed all internal references from `clipmaster` to `omaclip` (env vars, config path, mDNS service type, HTTP headers)
- Updated GitHub repository URL to `rhemvi/omaclip`

## [0.0.6] - 2026-04-06

### Changed

- Project renamed from Omaclip to Omaclip

## [0.0.5] - 2026-04-05

### Fixed

- Corrected help text for `--clipboard-poll-interval` flag

## [0.0.4] - 2026-04-05

### Changed

- Updated Go dependencies to latest minor/patch versions
- App version is now embedded in the binary as `vX.Y.Z+shortHash` (or `+shortHash.dirty` for unclean builds)

## [0.0.3] - 2026-04-05

### Added

- Log selected clipboard backend at startup
- Known limitations section in README documenting GNOME/Mutter Wayland focus flicker

### Changed

- Default clipboard poll interval from 500ms to 2s
- Default remote clipboards poll interval from 1s to 2s
- Darwin osascript backend is now a top-level case in the selection order, preferred over pbpaste

### Fixed

- Clipboard monitor now falls back to polling when `wl-paste --watch` exits immediately on compositors without `wlr-data-control` support (e.g. GNOME/Mutter)

## [0.0.2] - 2026-04-05

### Changed

- Wayland clipboard monitoring now uses `wl-paste --watch` instead of polling, with automatic fallback to polling if the watcher fails to start

### Fixed

- Passphrase setup screen now shows the actual config file path instead of a hardcoded default

## [0.0.1] - 2026-04-04

This is the initial release.

A desktop clipboard manager for Omarchy, works in Linux and macOS. It tracks your clipboard's history, lets you browse and copy items and is designed for a keyboard-first workflows.

When you run it on multiple machines peers will form a secure mesh, where they will auto-discover each other on the local network and share their clipboards.

It works on Linux and macOS, but it loves Omarchy, hot-reloading its color scheme the moment your OS theme changes.

### Added

- In-memory clipboard history text and images (PNG), up to 50 items (configurable)
- Keyboard navigation with shortcuts for quick copying (Ctrl+1..9)
- Expandable entries for viewing long text or larger image previews
- Live Omarchy theme support, colors update automatically when you switch themes
- Secure multi-machine sync, peers discover each other via mDNS and sync over HTTPS with certificate validation and a shared passphrase; only machines with the same passphrase can connect
