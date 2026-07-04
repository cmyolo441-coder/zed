#!/usr/bin/env bash
# ZED terminal AI agent — one-line installer (Linux / macOS)
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/cmyolo441-coder/zed/main/install.sh | bash
#
# Ye script aapke OS ka sahi binary GitHub Release se download karke
# ~/.local/bin/zed me install kar deti hai. Go ki zaroorat nahi.

set -euo pipefail

REPO="cmyolo441-coder/zed"
BIN_NAME="zed"
INSTALL_DIR="${HOME}/.local/bin"

# --- OS aur ARCH detect karo ---
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$OS" in
  linux)  OS="linux" ;;
  darwin) OS="macos" ;;
  *) echo "  Unsupported OS: $OS" >&2; exit 1 ;;
esac

case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) echo "  Unsupported arch: $ARCH" >&2; exit 1 ;;
esac

ASSET="zed-${OS}-${ARCH}"
URL="https://github.com/${REPO}/releases/latest/download/${ASSET}"

echo ""
echo "  Installing ZED..."
echo "  OS/Arch : ${OS}/${ARCH}"
echo "  Source  : ${URL}"
echo ""

# --- Download ---
mkdir -p "$INSTALL_DIR"
TMP="$(mktemp)"
if ! curl -fsSL "$URL" -o "$TMP"; then
  echo "  Download failed. Kya release me '${ASSET}' asset hai?" >&2
  rm -f "$TMP"
  exit 1
fi

chmod +x "$TMP"
mv "$TMP" "${INSTALL_DIR}/${BIN_NAME}"

echo "  Installed: ${INSTALL_DIR}/${BIN_NAME}"

# --- PATH check ---
case ":${PATH}:" in
  *":${INSTALL_DIR}:"*) : ;;
  *)
    echo ""
    echo "  NOTE: ${INSTALL_DIR} aapke PATH me nahi hai."
    echo "  Ye line apni shell config (~/.bashrc ya ~/.zshrc) me daalo:"
    echo ""
    echo "    export PATH=\"\$HOME/.local/bin:\$PATH\""
    echo ""
    echo "  Phir naya terminal kholo."
    ;;
esac

echo ""
echo "  Done! Ab type karo:  zed"
echo ""
