# Omaclip

A desktop clipboard manager for Omarchy, works in Linux and macOS. It tracks your
clipboard's history, lets you browse and copy items and is designed for a
keyboard-first workflow.

<https://github.com/user-attachments/assets/434b6ff6-21e1-459b-833f-7f63c5a9ee88>

When you run it on multiple machines peers will form a secure mesh, where they will auto-discover
each other on the local network and share their clipboards.

It works on Linux and macOS, but it loves [Omarchy](https://omarchy.org),
hot-reloading its color scheme the moment your OS theme changes.

- In-memory clipboard history text and images (PNG/JPEG), up to 50 items
  (configurable)
- Keyboard navigation with shortcuts for quick copying (Ctrl+1..9)
- Expandable entries for viewing long text or larger image previews
- Image file copy support: copying an image file from a file manager
  (Finder, Nautilus, etc.) captures the actual image, not just the filename
- Live Omarchy theme support, colors update automatically when you switch
  themes
- Secure multi-machine sync, peers discover each other via mDNS and sync
  over HTTPS with certificate validation and a shared passphrase; only
  machines with the same passphrase can connect
- Optional mDNS interface binding for multi-NIC setups

## Installation

### Option 1 — AUR (recommended for Arch / Omarchy)

```bash
yay -S omaclip-bin
```

### Option 2 — One-liner

The install script detects your OS, architecture, and package manager,
installs the required dependencies, and places the binary in `/usr/local/bin`.

```bash
curl -fsSL https://raw.githubusercontent.com/rhemvi/omaclip/master/install.sh | sh
```

### Option 3 — Manual installation

#### Linux

Install the runtime dependencies for your distro, then download and install
the binary.

##### Debian / Ubuntu

```bash
sudo apt install libgtk-3-0 libwebkit2gtk-4.1-0 xclip
curl -fsSL https://github.com/rhemvi/omaclip/releases/latest/download/omaclip-linux-amd64 -o omaclip
sudo install -m 755 omaclip /usr/local/bin/omaclip
```

##### Arch Linux

```bash
sudo pacman -S --needed gtk3 webkit2gtk-4.1 wl-clipboard
curl -fsSL https://github.com/rhemvi/omaclip/releases/latest/download/omaclip-linux-amd64 -o omaclip
sudo install -m 755 omaclip /usr/local/bin/omaclip
```

##### Fedora / RHEL

```bash
sudo dnf install gtk3 webkit2gtk4.1 wl-clipboard
curl -fsSL https://github.com/rhemvi/omaclip/releases/latest/download/omaclip-linux-amd64 -o omaclip
sudo install -m 755 omaclip /usr/local/bin/omaclip
```

##### openSUSE

```bash
sudo zypper install libgtk-3-0 libwebkit2gtk-4_1-0 xclip
curl -fsSL https://github.com/rhemvi/omaclip/releases/latest/download/omaclip-linux-amd64 -o omaclip
sudo install -m 755 omaclip /usr/local/bin/omaclip
```

> For ARM64 machines replace `omaclip-linux-amd64` with
> `omaclip-linux-arm64`.

#### macOS

No extra dependencies needed, macOS already ships with WebKit.

```bash
# Intel
curl -fsSL https://github.com/rhemvi/omaclip/releases/latest/download/omaclip-darwin-amd64 -o omaclip
sudo install -m 755 omaclip /usr/local/bin/omaclip

# Apple Silicon (M1/M2/M3)
curl -fsSL https://github.com/rhemvi/omaclip/releases/latest/download/omaclip-darwin-arm64 -o omaclip
sudo install -m 755 omaclip /usr/local/bin/omaclip
```

## Configuration

Omaclip can be configured via CLI flags or environment variables. Run
`omaclip --help` to see all options.

### Passphrase

On first launch, omaclip will prompt for a passphrase used to secure peer
sync. It is saved to `~/.config/omaclip/config.json`. All machines must
share the same passphrase to discover and sync with each other.

### mDNS interface binding

By default, mDNS peer discovery broadcasts on all network interfaces. On
machines with multiple NICs (e.g. WiFi + Ethernet + Docker bridge), you can
bind to a specific interface:

```bash
# CLI flag
omaclip --peers-mdns-interface en0

# Environment variable
export OMACLIP_PEERS_MDNS_INTERFACE=en0
```

Common interface names: `en0` (macOS WiFi), `wlan0` (Linux WiFi),
`eth0` (Linux Ethernet).

To make it permanent, add the export to your shell profile (`~/.zshrc`,
`~/.bashrc`, etc.).

## Live Development

To run in live development mode, run `wails dev` in the project directory.
This will run a Vite development server that will provide very fast hot reload
of your frontend changes. If you want to develop in a browser and have access
to your Go methods, there is also a dev server that runs on
<http://localhost:34115>. Connect to this in your browser, and you can call your
Go code from devtools.

## Building

To build a redistributable, production ready package, run:

```bash
task app:build
```

## Known Limitations

### Clipboard monitoring on GNOME/Mutter (Wayland)

On desktops that use GNOME's Mutter compositor (Fedora, Ubuntu, Pop!_OS, etc.),
clipboard monitoring may cause brief focus flicker. This is a Wayland security
restriction: the standard clipboard protocol only delivers content to the
focused window, so `wl-paste` must briefly acquire focus on each poll cycle.

Compositors built on wlroots (Hyprland, Sway, etc.) are not affected because
they support the `wlr-data-control` protocol, which allows background clipboard
access without focus changes. On these compositors, omaclip uses event-driven
watching with `wl-paste --watch` and no polling is needed.

X11 sessions and macOS are also unaffected.
