#!/bin/sh
set -e

REPO="${REPO:-Awenforever/weclaw_dev}"
BINARY="${BINARY:-weclaw}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
VERSION="${VERSION:-}"
REF="${REF:-main}"
GO_BOOTSTRAP_VERSION="${GO_BOOTSTRAP_VERSION:-1.25.0}"

resolve_tool() {
  tool="$1"

  if command -v "$tool" >/dev/null 2>&1; then
    command -v "$tool"
    return 0
  fi

  for shell_bin in bash zsh; do
    if command -v "$shell_bin" >/dev/null 2>&1; then
      resolved=$("$shell_bin" -lc "command -v $tool" 2>/dev/null || true)
      if [ -n "$resolved" ] && [ -x "$resolved" ]; then
        printf '%s\n' "$resolved"
        return 0
      fi
    fi
  done

  case "$tool" in
    go)
      for candidate in /usr/local/go/bin/go /opt/homebrew/bin/go /usr/local/bin/go; do
        if [ -x "$candidate" ]; then
          printf '%s\n' "$candidate"
          return 0
        fi
      done
      ;;
    git)
      for candidate in /usr/bin/git /usr/local/bin/git /opt/homebrew/bin/git; do
        if [ -x "$candidate" ]; then
          printf '%s\n' "$candidate"
          return 0
        fi
      done
      ;;
  esac

  return 1
}

download_go_toolchain() {
  if ! command -v tar >/dev/null 2>&1; then
    echo "Error: tar is required to bootstrap Go."
    exit 1
  fi

  GO_OS="$OS"
  GO_ARCH="$ARCH"
  URL="https://go.dev/dl/go${GO_BOOTSTRAP_VERSION}.${GO_OS}-${GO_ARCH}.tar.gz"
  ARCHIVE="$TMPDIR/go-bootstrap.tar.gz"
  DEST="$TMPDIR/go-bootstrap"

  echo "Go not found; bootstrapping Go ${GO_BOOTSTRAP_VERSION} from ${URL}..."
  mkdir -p "$DEST"
  curl -fsSL -o "$ARCHIVE" "$URL"
  tar -C "$DEST" -xzf "$ARCHIVE"

  if [ ! -x "$DEST/go/bin/go" ]; then
    echo "Error: bootstrapped Go toolchain is invalid."
    exit 1
  fi

  printf '%s\n' "$DEST/go/bin/go"
}

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
  GIT_BIN=$(resolve_tool git || true)
  GO_BIN=$(resolve_tool go || true)

  if [ -z "$GIT_BIN" ]; then
    echo "Error: git is required to build from source."
    exit 1
  fi
  echo "No release found; building from source..."
  TMPDIR=$(mktemp -d)
  SRC="$TMPDIR/src"
  trap 'rm -rf "$TMPDIR"' EXIT INT TERM

  if [ -z "$GO_BIN" ]; then
    GO_BIN=$(download_go_toolchain)
  fi

  "$GIT_BIN" clone --depth 1 "https://github.com/${REPO}.git" "$SRC"
  if [ -n "$REF" ] && [ "$REF" != "main" ]; then
    (cd "$SRC" && "$GIT_BIN" checkout "$REF")
  fi

  (cd "$SRC" && "$GO_BIN" build -o "$TMPDIR/${BINARY}" .)
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
