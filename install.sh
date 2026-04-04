#!/usr/bin/env bash
set -euo pipefail

REPO="ntavelis/clipmaster"
BINARY="clipmaster"
INSTALL_DIR="/usr/local/bin"
DESKTOP_DIR="${HOME}/.local/share/applications"
ICON_DIR="${HOME}/.local/share/icons"
ICON_URL="https://github.com/${REPO}/releases/latest/download/appicon.png"

# Detect OS
OS="$(uname -s)"
case "${OS}" in
  Linux)  OS="linux" ;;
  Darwin) OS="darwin" ;;
  *)
    echo "Unsupported OS: ${OS}"
    exit 1
    ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "${ARCH}" in
  x86_64)          ARCH="amd64" ;;
  aarch64 | arm64) ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: ${ARCH}"
    exit 1
    ;;
esac

echo "Detected: ${OS}/${ARCH}"

# Detect clipboard backend
detect_clipboard_pkg() {
  if [ "${XDG_SESSION_TYPE:-}" = "wayland" ]; then
    echo "wayland"
  elif [ "${XDG_SESSION_TYPE:-}" = "x11" ]; then
    echo "x11"
  elif [ -n "${WAYLAND_DISPLAY:-}" ]; then
    echo "wayland"
  elif [ -n "${DISPLAY:-}" ]; then
    echo "x11"
  else
    echo "both"
  fi
}

# Install runtime dependencies (Linux only)
install_deps_linux() {
  local clip_backend
  clip_backend="$(detect_clipboard_pkg)"

  case "${clip_backend}" in
    wayland) CLIP_PKGS="wl-clipboard" ;;
    x11)     CLIP_PKGS="xclip" ;;
    both)    CLIP_PKGS="wl-clipboard xclip" ;;
  esac

  echo "Detected display server: ${clip_backend}"

  if command -v apt-get &>/dev/null; then
    echo "Installing dependencies via apt..."
    sudo apt-get install -y libgtk-3-0 libwebkit2gtk-4.0-37 ${CLIP_PKGS}
  elif command -v pacman &>/dev/null; then
    echo "Installing dependencies via pacman..."
    sudo pacman -S --needed --noconfirm gtk3 webkit2gtk ${CLIP_PKGS}
  elif command -v apk &>/dev/null; then
    echo "Installing dependencies via apk (Alpine)..."
    sudo apk add --no-cache gtk+3.0 webkit2gtk ${CLIP_PKGS}
  elif command -v rpm-ostree &>/dev/null; then
    echo "Installing dependencies via rpm-ostree (Fedora Atomic)..."
    sudo rpm-ostree install --idempotent --apply-live gtk3 webkit2gtk3 ${CLIP_PKGS}
  elif command -v dnf &>/dev/null; then
    echo "Installing dependencies via dnf..."
    sudo dnf install -y gtk3 webkit2gtk3 ${CLIP_PKGS}
  elif command -v xbps-install &>/dev/null; then
    echo "Installing dependencies via xbps (Void Linux)..."
    sudo xbps-install -Sy gtk+3 webkit2gtk ${CLIP_PKGS}
  elif command -v zypper &>/dev/null; then
    echo "Installing dependencies via zypper..."
    sudo zypper install -y libgtk-3-0 libwebkit2gtk-4_0-37 ${CLIP_PKGS}
  else
    echo "No supported package manager found (apt, pacman, apk, rpm-ostree, dnf, xbps, zypper)."
    echo "Please install the following libraries manually:"
    echo "  - GTK 3 runtime"
    echo "  - WebKit2GTK 4.0 runtime"
    echo "  - A clipboard tool (wl-clipboard for Wayland, xclip for X11)"
    exit 1
  fi
}

if [ "${OS}" = "linux" ]; then
  install_deps_linux
fi

# Download binary
ASSET_NAME="${BINARY}-${OS}-${ARCH}"
DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${ASSET_NAME}"
TMP_FILE="$(mktemp)"

echo "Downloading ${ASSET_NAME}..."
if command -v curl &>/dev/null; then
  curl -fsSL "${DOWNLOAD_URL}" -o "${TMP_FILE}"
elif command -v wget &>/dev/null; then
  wget -qO "${TMP_FILE}" "${DOWNLOAD_URL}"
else
  echo "Neither curl nor wget found. Please install one and retry."
  exit 1
fi

chmod +x "${TMP_FILE}"
echo "Installing to ${INSTALL_DIR}/${BINARY}..."
sudo mv "${TMP_FILE}" "${INSTALL_DIR}/${BINARY}"

# Download icon and create .desktop entry (Linux only)
install_desktop_linux() {
  mkdir -p "${ICON_DIR}"
  echo "Downloading icon..."
  if command -v curl &>/dev/null; then
    curl -fsSL "${ICON_URL}" -o "${ICON_DIR}/clipmaster.png"
  elif command -v wget &>/dev/null; then
    wget -qO "${ICON_DIR}/clipmaster.png" "${ICON_URL}"
  fi

  mkdir -p "${DESKTOP_DIR}"
  cat > "${DESKTOP_DIR}/clipmaster.desktop" <<EOF
[Desktop Entry]
Name=Clipmaster
Comment=Clipboard manager with multi-machine sync
Exec=${INSTALL_DIR}/${BINARY}
Icon=${ICON_DIR}/clipmaster.png
Type=Application
Categories=Utility;
Terminal=false
EOF
  echo "Desktop entry created at ${DESKTOP_DIR}/clipmaster.desktop"
}

if [ "${OS}" = "linux" ]; then
  install_desktop_linux
fi

echo "Clipmaster installed successfully. Run 'clipmaster' to start."
