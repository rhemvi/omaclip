# Clipmaster

A desktop clipboard manager for Linux and macOS. It tracks your clipboard's
history, lets you browse and re-copy past items, and is designed for
keyboard-first workflows.

https://github.com/user-attachments/assets/434b6ff6-21e1-459b-833f-7f63c5a9ee88

Run it on multiple machines and they form a secure mesh — peers auto-discover
each other on the local network, no configuration needed. Every machine stays
in sync, and your clipboard is always where you need it.

It works on Linux and macOS, but it loves [Omarchy](https://omarchy.org) —
hot-reloading its color scheme the moment your OS theme changes, so it always
looks like it belongs.

- In-memory clipboard history — text and images (PNG), up to 50 items
  (configurable)
- Keyboard navigation with shortcuts for quick copying (Ctrl+1..9)
- Expandable entries for viewing long text or larger image previews
- Live Omarchy theme support — colors update automatically when you switch
  themes
- Secure multi-machine sync — peers discover each other via mDNS and sync
  over HTTPS with certificate validation and a shared passphrase; only
  machines with the same passphrase can connect

## Installation

### Option 1 — One-liner (recommended)

The install script detects your OS, architecture, and package manager,
installs the required dependencies, and places the binary in `/usr/local/bin`.

```bash
curl -fsSL https://raw.githubusercontent.com/ntavelis/clipmaster/main/install.sh | sh
```

### Option 2 — Manual installation

#### Linux

Install the runtime dependencies for your distro, then download and install
the binary.

##### Debian / Ubuntu

```bash
sudo apt install libgtk-3-0 libwebkit2gtk-4.0-37 wl-clipboard
curl -fsSL https://github.com/ntavelis/clipmaster/releases/latest/download/clipmaster-linux-amd64 -o clipmaster
sudo install -m 755 clipmaster /usr/local/bin/clipmaster
```

##### Arch Linux

```bash
sudo pacman -S --needed gtk3 webkit2gtk wl-clipboard
curl -fsSL https://github.com/ntavelis/clipmaster/releases/latest/download/clipmaster-linux-amd64 -o clipmaster
sudo install -m 755 clipmaster /usr/local/bin/clipmaster
```

##### Fedora / RHEL

```bash
sudo dnf install gtk3 webkit2gtk3 wl-clipboard
curl -fsSL https://github.com/ntavelis/clipmaster/releases/latest/download/clipmaster-linux-amd64 -o clipmaster
sudo install -m 755 clipmaster /usr/local/bin/clipmaster
```

##### openSUSE

```bash
sudo zypper install libgtk-3-0 libwebkit2gtk-4_0-37 wl-clipboard
curl -fsSL https://github.com/ntavelis/clipmaster/releases/latest/download/clipmaster-linux-amd64 -o clipmaster
sudo install -m 755 clipmaster /usr/local/bin/clipmaster
```

> For ARM64 machines replace `clipmaster-linux-amd64` with
> `clipmaster-linux-arm64`.

#### macOS

No extra dependencies needed — macOS ships with WebKit.

```bash
# Intel
curl -fsSL https://github.com/ntavelis/clipmaster/releases/latest/download/clipmaster-darwin-amd64 -o clipmaster
sudo install -m 755 clipmaster /usr/local/bin/clipmaster

# Apple Silicon (M1/M2/M3)
curl -fsSL https://github.com/ntavelis/clipmaster/releases/latest/download/clipmaster-darwin-arm64 -o clipmaster
sudo install -m 755 clipmaster /usr/local/bin/clipmaster
```

## Live Development

To run in live development mode, run `wails dev` in the project directory.
This will run a Vite development server that will provide very fast hot reload
of your frontend changes. If you want to develop in a browser and have access
to your Go methods, there is also a dev server that runs on
http://localhost:34115. Connect to this in your browser, and you can call your
Go code from devtools.

## Building

To build a redistributable, production mode package, run:

```bash
task app:build
```
