#!/bin/sh
set -e

REPO="${REPO:-Awenforever/weclaw_dev}"
BINARY="${BINARY:-weclaw}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
VERSION="${VERSION:-}"
REF="${REF:-main}"
GO_BOOTSTRAP_VERSION="${GO_BOOTSTRAP_VERSION:-1.25.0}"
ACTION="install"
TMP_ROOT=""

usage() {
  cat <<'USAGE'
WeClaw Dev installer

Usage:
  sh install.sh
  sh install.sh --uninstall

Options:
  --version VERSION     Install a specific GitHub Release tag
  --install-dir DIR     Install directory, default: /usr/local/bin
  --repo OWNER/REPO     GitHub repository, default: Awenforever/weclaw_dev
  --ref REF             Source build ref, default: main
  --uninstall           Remove the installed weclaw binary and keep ~/.weclaw data
  --help                Show this help

Examples:
  curl -fsSL https://cdn.jsdelivr.net/gh/Awenforever/weclaw_dev@main/install.sh | sh
  # Fallback:
  curl -fsSL https://raw.githubusercontent.com/Awenforever/weclaw_dev/main/install.sh | sh

  curl -fsSL https://cdn.jsdelivr.net/gh/Awenforever/weclaw_dev@main/install.sh | sh -s -- --uninstall
  # Fallback:
  curl -fsSL https://raw.githubusercontent.com/Awenforever/weclaw_dev/main/install.sh | sh -s -- --uninstall
USAGE
}

parse_args() {
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --help|-h)
        ACTION="help"
        ;;
      --uninstall|uninstall)
        ACTION="uninstall"
        ;;
      --version)
        shift
        if [ "$#" -eq 0 ]; then
          echo "Error: --version requires a value"
          return 1
        fi
        VERSION="$1"
        ;;
      --version=*)
        VERSION="${1#--version=}"
        ;;
      --install-dir)
        shift
        if [ "$#" -eq 0 ]; then
          echo "Error: --install-dir requires a value"
          return 1
        fi
        INSTALL_DIR="$1"
        ;;
      --install-dir=*)
        INSTALL_DIR="${1#--install-dir=}"
        ;;
      --repo)
        shift
        if [ "$#" -eq 0 ]; then
          echo "Error: --repo requires a value"
          return 1
        fi
        REPO="$1"
        ;;
      --repo=*)
        REPO="${1#--repo=}"
        ;;
      --ref)
        shift
        if [ "$#" -eq 0 ]; then
          echo "Error: --ref requires a value"
          return 1
        fi
        REF="$1"
        ;;
      --ref=*)
        REF="${1#--ref=}"
        ;;
      *)
        echo "Error: unknown option: $1"
        usage
        return 1
        ;;
    esac
    shift
  done
}

cleanup_tmp() {
  if [ -n "$TMP_ROOT" ] && [ -d "$TMP_ROOT" ]; then
    rm -rf "$TMP_ROOT"
  fi
}

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

detect_platform() {
  OS=$(uname -s | tr '[:upper:]' '[:lower:]')
  case "$OS" in
    darwin|linux) ;;
    *)
      echo "Unsupported OS: $OS"
      return 1
      ;;
  esac

  ARCH=$(uname -m)
  case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *)
      echo "Unsupported architecture: $ARCH"
      return 1
      ;;
  esac

  echo "Detected: ${OS}/${ARCH}"
}

ensure_tmp_root() {
  if [ -z "$TMP_ROOT" ]; then
    TMP_ROOT=$(mktemp -d 2>/dev/null || mktemp -d -t weclaw-install)
  fi
}

install_binary_file() {
  src="$1"
  target="${INSTALL_DIR}/${BINARY}"
  staged="${INSTALL_DIR}/.${BINARY}-install-$$.new"

  chmod +x "$src"

  if [ -d "$INSTALL_DIR" ] && [ -w "$INSTALL_DIR" ]; then
    cp "$src" "$staged"
    chmod +x "$staged"
    mv -f "$staged" "$target"
  else
    echo "Installing to ${INSTALL_DIR} (requires sudo)..."
    sudo mkdir -p "$INSTALL_DIR"
    sudo cp "$src" "$staged"
    sudo chmod +x "$staged"
    sudo mv -f "$staged" "$target"
  fi
}

fetch_release_version() {
  curl -fsSL -H "User-Agent: weclaw-installer" "https://api.github.com/repos/${REPO}/releases/latest" | sed -n 's/.*"tag_name" *: *"\([^"]*\)".*/\1/p'
}


curl_fetch() {
  FETCH_URL="$1"
  FETCH_DEST="$2"
  FETCH_LABEL="$3"

  rm -f "$FETCH_DEST"
  curl -fL \
    --retry 8 \
    --retry-delay 2 \
    --connect-timeout 20 \
    --max-time 300 \
    "$FETCH_URL" \
    -o "$FETCH_DEST"
  FETCH_STATUS="$?"

  if [ "$FETCH_STATUS" = "0" ] && [ -s "$FETCH_DEST" ]; then
    return 0
  fi

  echo "${FETCH_LABEL} download failed with curl status ${FETCH_STATUS}: ${FETCH_URL}" >&2
  rm -f "$FETCH_DEST"
  return "$FETCH_STATUS"
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

  ensure_tmp_root
  TMP="${TMP_ROOT}/${FILENAME}"

  echo "Downloading ${URL}..."
  curl_fetch "$URL" "$TMP" "Release asset"
  ASSET_STATUS="$?"
  if [ "$ASSET_STATUS" != "0" ]; then
    rm -f "$TMP"
    if [ "$ASSET_STATUS" = "22" ]; then
      echo "No release asset found at ${URL}; building from source..."
    else
      echo "Release asset download failed due to network or transport error; trying source fallback..."
    fi
    return 1
  fi

  chmod +x "$TMP"
  install_binary_file "$TMP"
}


download_go_toolchain() {
  if ! command -v tar >/dev/null 2>&1; then
    echo "Error: tar is required to bootstrap Go." >&2
    return 1
  fi

  ensure_tmp_root
  GO_OS="$OS"
  GO_ARCH="$ARCH"
  PRIMARY_URL="https://go.dev/dl/go${GO_BOOTSTRAP_VERSION}.${GO_OS}-${GO_ARCH}.tar.gz"
  FALLBACK_URL="https://dl.google.com/go/go${GO_BOOTSTRAP_VERSION}.${GO_OS}-${GO_ARCH}.tar.gz"
  ARCHIVE="${TMP_ROOT}/go-bootstrap.tar.gz"
  DEST="${TMP_ROOT}/go-bootstrap"

  echo "Go not found or system Go is unsuitable; bootstrapping Go ${GO_BOOTSTRAP_VERSION}." >&2
  rm -rf "$DEST" "$ARCHIVE"
  mkdir -p "$DEST"

  echo "Downloading Go toolchain from ${PRIMARY_URL}..." >&2
  if ! curl_fetch "$PRIMARY_URL" "$ARCHIVE" "Go toolchain"; then
    echo "Primary Go toolchain download failed; trying fallback: ${FALLBACK_URL}" >&2
    if ! curl_fetch "$FALLBACK_URL" "$ARCHIVE" "Go toolchain fallback"; then
      echo "Error: failed to download Go toolchain from go.dev or dl.google.com." >&2
      return 1
    fi
  fi

  if ! tar -C "$DEST" -xzf "$ARCHIVE"; then
    echo "Error: failed to extract bootstrapped Go toolchain." >&2
    return 1
  fi

  if [ ! -x "$DEST/go/bin/go" ]; then
    echo "Error: bootstrapped Go toolchain is invalid." >&2
    return 1
  fi

  printf '%s\n' "$DEST/go/bin/go"
}

install_from_source() {
  ensure_tmp_root
  SRC="${TMP_ROOT}/src"
  SRC_ROOT="${TMP_ROOT}/source-root"
  SRC_ARCHIVE="${TMP_ROOT}/source.tar.gz"

  rm -rf "$SRC" "$SRC_ROOT" "$SRC_ARCHIVE"
  mkdir -p "$SRC_ROOT"

  echo "Building from source fallback..."

  if [ -n "$VERSION" ]; then
    SRC_URL="https://codeload.github.com/${REPO}/tar.gz/refs/tags/${VERSION}"
    echo "Trying source tarball: ${SRC_URL}"
    curl_fetch "$SRC_URL" "$SRC_ARCHIVE" "Source tarball" || true
  fi

  if [ ! -s "$SRC_ARCHIVE" ]; then
    SRC_URL="https://codeload.github.com/${REPO}/tar.gz/refs/heads/${REF}"
    echo "Trying source ref tarball: ${SRC_URL}"
    curl_fetch "$SRC_URL" "$SRC_ARCHIVE" "Source ref tarball" || true
  fi

  if [ -s "$SRC_ARCHIVE" ]; then
    tar -xzf "$SRC_ARCHIVE" -C "$SRC_ROOT"
    SRC_EXTRACTED=$(find "$SRC_ROOT" -mindepth 1 -maxdepth 1 -type d | head -n 1)
    if [ -z "$SRC_EXTRACTED" ]; then
      echo "Error: source tarball extraction produced no source directory." >&2
      return 1
    fi
    mv "$SRC_EXTRACTED" "$SRC"
  else
    GIT_BIN=$(resolve_tool git || true)
    if [ -z "$GIT_BIN" ]; then
      echo "Error: source tarball fallback failed and git is not available." >&2
      return 1
    fi

    echo "Source tarball fallback failed; trying git clone over HTTP/1.1 as last resort..."
    if [ -n "$VERSION" ]; then
      "$GIT_BIN" -c http.version=HTTP/1.1 clone --depth 1 --branch "$VERSION" "https://github.com/${REPO}.git" "$SRC" \
        || "$GIT_BIN" -c http.version=HTTP/1.1 clone --depth 1 --branch "$REF" "https://github.com/${REPO}.git" "$SRC"
    else
      "$GIT_BIN" -c http.version=HTTP/1.1 clone --depth 1 --branch "$REF" "https://github.com/${REPO}.git" "$SRC"
    fi
  fi

  if [ ! -d "$SRC" ]; then
    echo "Error: source fallback failed before build." >&2
    return 1
  fi

  GO_BIN=$(download_go_toolchain)
  if [ -z "$GO_BIN" ] || [ ! -x "$GO_BIN" ]; then
    echo "Error: failed to prepare Go toolchain." >&2
    return 1
  fi

  BUILD_VERSION="${VERSION:-source}"
  (
    cd "$SRC"
    CGO_ENABLED=0 "$GO_BIN" build -trimpath \
      -ldflags="-s -w -X github.com/fastclaw-ai/weclaw/cmd.Version=${BUILD_VERSION}" \
      -o "${TMP_ROOT}/${BINARY}" .
  )
  install_binary_file "${TMP_ROOT}/${BINARY}"
}

uninstall_binary() {
  target="${INSTALL_DIR}/${BINARY}"
  resolved=$(command -v "$BINARY" 2>/dev/null || true)
  if [ -n "$resolved" ]; then
    target="$resolved"
  fi

  if [ ! -e "$target" ]; then
    echo "weclaw binary not found at ${target}"
    echo "User data, if any, remains at ~/.weclaw"
    return 0
  fi

  echo "Removing ${target}..."
  if rm -f "$target" 2>/dev/null; then
    echo "weclaw binary removed."
  else
    echo "Removing ${target} requires sudo..."
    sudo rm -f "$target"
    echo "weclaw binary removed."
  fi

  echo "User data is preserved at ~/.weclaw"
  echo "To reinstall:"
  echo "  curl -fsSL https://cdn.jsdelivr.net/gh/${REPO}@main/install.sh | sh"
  echo "  # Fallback:"
  echo "  curl -fsSL https://raw.githubusercontent.com/${REPO}/main/install.sh | sh"
}

main() {
  parse_args "$@" || return 1

  if [ "$ACTION" = "help" ]; then
    usage
    return 0
  fi

  if [ "$ACTION" = "uninstall" ]; then
    uninstall_binary
    return $?
  fi

  trap cleanup_tmp 0 2 3 15

  detect_platform || return 1

  if ! install_release; then
    install_from_source || return 1
  fi

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
  echo ""
  echo "Update later:"
  echo "  weclaw upgrade"
  echo "  curl -fsSL https://cdn.jsdelivr.net/gh/${REPO}@main/install.sh | sh"
  echo "  # Fallback:"
  echo "  curl -fsSL https://raw.githubusercontent.com/${REPO}/main/install.sh | sh"
  echo ""
  echo "Uninstall:"
  echo "  curl -fsSL https://cdn.jsdelivr.net/gh/${REPO}@main/install.sh | sh -s -- --uninstall"
  echo "  # Fallback:"
  echo "  curl -fsSL https://raw.githubusercontent.com/${REPO}/main/install.sh | sh -s -- --uninstall"
}

main "$@"
