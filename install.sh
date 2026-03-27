#!/bin/sh
set -e

REPO="favo/pintomind-cli"
BIN="pintomind"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

# Detect OS
case "$(uname -s)" in
  Linux)  OS="linux" ;;
  Darwin) OS="darwin" ;;
  *)
    echo "Unsupported OS: $(uname -s)" >&2
    exit 1
    ;;
esac

# Detect architecture
case "$(uname -m)" in
  x86_64)          ARCH="amd64" ;;
  arm64 | aarch64) ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $(uname -m)" >&2
    exit 1
    ;;
esac

ASSET="${BIN}_${OS}_${ARCH}"
URL="https://github.com/${REPO}/releases/latest/download/${ASSET}"

echo "Downloading ${BIN} for ${OS}/${ARCH}..."

# Download
if command -v curl >/dev/null 2>&1; then
  curl -fsSL "$URL" -o "/tmp/${ASSET}"
elif command -v wget >/dev/null 2>&1; then
  wget -qO "/tmp/${ASSET}" "$URL"
else
  echo "Neither curl nor wget found. Please install one and retry." >&2
  exit 1
fi

# Install
mkdir -p "$INSTALL_DIR"
install -m755 "/tmp/${ASSET}" "${INSTALL_DIR}/${BIN}"
rm "/tmp/${ASSET}"

echo "Installed ${BIN} to ${INSTALL_DIR}/${BIN}"

# Warn if INSTALL_DIR is not on PATH
case ":${PATH}:" in
  *":${INSTALL_DIR}:"*) ;;
  *)
    echo ""
    echo "Note: ${INSTALL_DIR} is not in your PATH."
    echo "Add this to your shell profile:"
    echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
    ;;
esac
