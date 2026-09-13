# Omaclip

A desktop clipboard manager for Omarchy that works on Linux and macOS. It tracks
your clipboard history, lets you browse and copy items, and is designed for a
keyboard-first workflow. Run it on multiple machines and they form a secure mesh,
automatically discovering each other and sharing their clipboards.

<https://github.com/user-attachments/assets/0fe02cc8-3e18-4d15-a089-144030f26b49>

It works on Linux and macOS, but it loves [Omarchy](https://omarchy.org),
hot-reloading its color scheme the moment your OS theme changes.

- In-memory clipboard history text and images (PNG/JPEG), up to 50 items
  (configurable)
- Keyboard navigation with shortcuts for quick copying (Ctrl+1..9)
- Expandable entries for viewing long text or larger image previews
- Image file copy support: copying an image file from a file manager
  (Finder, Nautilus, etc.) captures the actual image, not just the filename
- Live theme loading for Omarchy 3 and 4; colors update automatically when you
  switch themes
- Secure multi-machine sync, peers discover each other via mDNS and sync
  over HTTPS with certificate validation and a shared passphrase; only
  machines with the same passphrase can connect
- Checksum check first, before a sync from remote peer, no wasteful network traffic
- Optional mDNS interface binding for multi-NIC setups
- Manual peer configuration and fixed sync server port for VPN/routed networks

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

## Omarchy shell plugin

Omarchy users can install the [Omarchy Omaclip QML plugin](https://github.com/rhemvi/omarchy-omaclip)
to start Omaclip automatically in a Hyprland special workspace and register a
copy hook that closes the workspace and pastes the copied entry into the focused
window at the cursor location.

## Configuration

Omaclip can be configured via CLI flags or environment variables. Run
`omaclip --help` to see all options.

### Passphrase

On first launch, omaclip will prompt for a passphrase used to secure peer
sync. It is saved by default to `~/.config/omaclip/config.json`. All machines must
share the same passphrase to discover and sync with each other.

### Copy hook

`--copy-hook` runs a shell command after Omaclip successfully writes a selected
local or remote entry to the system clipboard.

To verify the hook is running, show a desktop notification after copying an
entry:

```bash
omaclip --copy-hook='notify-send "Omaclip" "Copy hook triggered"'
```

For example, if Omaclip is open on a Hyprland special workspace named
`scratchpad`, this closes the workspace, restores focus to the previous
window, and pastes the selected entry with `wtype`:

```bash
omaclip --copy-hook='hyprctl dispatch "hl.dsp.workspace.toggle_special(\"scratchpad\")" && sleep 0.2 && wtype -M ctrl -M shift -P v -p v -m shift -m ctrl'
```

### mDNS interface binding

By default, mDNS peer discovery uses IPv4 multicast on all eligible network
interfaces. On machines with multiple NICs (e.g. WiFi + Ethernet), it is
recommended to set `--peers-mdns-interface` so both mDNS advertising and
browsing use the intended local network interface.

This is useful for local mDNS networks. VPNs such as Tailscale generally do not
carry mDNS multicast between machines, so use manual peers for those setups.

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

### Manual peers and fixed sync port

By default, omaclip uses mDNS to discover peers on the local network and starts
its sync HTTPS server on a random OS-assigned port. For VPN or routed networks
such as Tailscale, mDNS may not cross network boundaries and the random port can
change after restart. In that setup, run each machine with a fixed sync server
port and manually list the other peers by Tailscale IPv4 address:

```bash
# On this machine, listen on a stable port and pull from davel over Tailscale
omaclip --sync-server-port=36742 \
  --peers-list=davel@100.101.102.103:36742
```

Use the same passphrase on every machine. The `name@` prefix is optional and is
only used as the display name; the address must be an IP and port. To specify
multiple peers, separate them with semicolons and quote the value in your shell:
`--peers-list='davel@100.101.102.103:36742;alice@100.104.105.106:36742'`.
When `--peers-list` is set, mDNS discovery is skipped entirely.

Environment variable equivalent:

```bash
export OMACLIP_SYNC_SERVER_PORT=36742
export OMACLIP_PEERS_LIST=davel@100.101.102.103:36742
```

Run this on every machine that should participate in the mesh. Each machine
should use its own fixed `--sync-server-port` and list the other machines with
their respective Tailscale IPs in `--peers-list`; once both sides point at each
other, their clipboards sync between them.

For example, on `alice`:

```bash
omaclip --sync-server-port=36742 \
  --peers-list=davel@100.101.102.103:36742
```

And on `davel`:

```bash
omaclip --sync-server-port=36742 \
  --peers-list=alice@100.104.105.106:36742
```

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
