#!/bin/sh
set -e

# Install beadwork (bw) from GitHub releases.
# Usage: curl -fsSL https://raw.githubusercontent.com/jallum/beadwork/main/install.sh | sh

REPO="jallum/beadwork"
BINARY="bw"

# Allow override, otherwise prefer ~/.local/bin (no sudo), fall back to /usr/local/bin
if [ -n "$INSTALL_DIR" ]; then
    : # user override
elif [ -d "$HOME/.local/bin" ] && echo "$PATH" | grep -q "$HOME/.local/bin"; then
    INSTALL_DIR="$HOME/.local/bin"
else
    INSTALL_DIR="/usr/local/bin"
fi

fail() { echo "error: $1" >&2; exit 1; }

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
    linux|darwin) ;;
    *) fail "unsupported OS: $OS" ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) fail "unsupported architecture: $ARCH" ;;
esac

# Fetch latest version from GitHub API
VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed 's/.*"v\(.*\)".*/\1/')
[ -z "$VERSION" ] && fail "could not determine latest version"

ASSET="beadwork_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/v${VERSION}/${ASSET}"

echo "installing bw ${VERSION} (${OS}/${ARCH})"

# Download and extract
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

curl -fsSL "$URL" -o "${TMPDIR}/${ASSET}" || fail "download failed: ${URL}"
tar -xzf "${TMPDIR}/${ASSET}" -C "$TMPDIR" || fail "extraction failed"

# Install
if [ -w "$INSTALL_DIR" ]; then
    mv "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
else
    echo "installing to ${INSTALL_DIR} (requires sudo)"
    sudo mv "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
fi

chmod +x "${INSTALL_DIR}/${BINARY}"
echo "installed ${INSTALL_DIR}/${BINARY} (v${VERSION})"
