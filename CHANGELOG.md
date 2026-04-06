# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
