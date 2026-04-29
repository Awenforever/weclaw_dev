#!/bin/sh
set -e

REPO="${REPO:-Awenforever/weclaw_dev}"
BINARY="${BINARY:-weclaw}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
VERSION="${VERSION:-}"
REF="${REF:-main}"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  darwin|linux) ;;
  *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

echo "Detected: ${OS}/${ARCH}"

fetch_release_version() {
  curl -fsSL -H "User-Agent: weclaw-installer" "https://api.github.com/repos/${REPO}/releases/latest" | sed -n 's/.*"tag_name" *: *"\([^"]*\)".*/\1/p'
}

install_release() {
  echo "Fetching latest release..."
  if [ -z "$VERSION" ]; then
    VERSION=$(fetch_release_version || true)
  fi
  if [ -z "$VERSION" ]; then
    return 1
  fi

  echo "Latest version: ${VERSION}"

  FILENAME="${BINARY}_${OS}_${ARCH}"
  URL="https://github.com/${REPO}/releases/download/${VERSION}/${FILENAME}"

  echo "Downloading ${URL}..."
  TMP=$(mktemp)
  if ! curl -fsSL -o "$TMP" "$URL"; then
    rm -f "$TMP"
    return 1
  fi

  chmod +x "$TMP"
  if [ -d "$INSTALL_DIR" ] && [ -w "$INSTALL_DIR" ]; then
    mv "$TMP" "${INSTALL_DIR}/${BINARY}"
  else
    echo "Installing to ${INSTALL_DIR} (requires sudo)..."
    sudo mkdir -p "$INSTALL_DIR"
    sudo mv "$TMP" "${INSTALL_DIR}/${BINARY}"
  fi
}

install_from_source() {
  if ! command -v git >/dev/null 2>&1; then
    echo "Error: git is required to build from source."
    exit 1
  fi
  if ! command -v go >/dev/null 2>&1; then
    echo "Error: go is required to build from source when no release is available."
    exit 1
  fi

  echo "No release found; building from source..."
  TMPDIR=$(mktemp -d)
  SRC="$TMPDIR/src"
  trap 'rm -rf "$TMPDIR"' EXIT INT TERM

  git clone --depth 1 "https://github.com/${REPO}.git" "$SRC"
  if [ -n "$REF" ] && [ "$REF" != "main" ]; then
    (cd "$SRC" && git checkout "$REF")
  fi

  (cd "$SRC" && go build -o "$TMPDIR/${BINARY}" .)
  chmod +x "$TMPDIR/${BINARY}"
  if [ -d "$INSTALL_DIR" ] && [ -w "$INSTALL_DIR" ]; then
    mv "$TMPDIR/${BINARY}" "${INSTALL_DIR}/${BINARY}"
  else
    echo "Installing to ${INSTALL_DIR} (requires sudo)..."
    sudo mkdir -p "$INSTALL_DIR"
    sudo mv "$TMPDIR/${BINARY}" "${INSTALL_DIR}/${BINARY}"
  fi
}

if ! install_release; then
  install_from_source
fi

# Clear macOS quarantine attributes
if [ "$OS" = "darwin" ]; then
  xattr -d com.apple.quarantine "${INSTALL_DIR}/${BINARY}" 2>/dev/null || true
  xattr -d com.apple.provenance "${INSTALL_DIR}/${BINARY}" 2>/dev/null || true
fi

echo ""
if [ -n "$VERSION" ]; then
  echo "weclaw ${VERSION} installed to ${INSTALL_DIR}/${BINARY}"
else
  echo "weclaw installed to ${INSTALL_DIR}/${BINARY}"
fi
echo ""
echo "Get started:"
echo "  weclaw start"
echo "  weclaw start --stdout"
