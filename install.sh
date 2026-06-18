#!/usr/bin/env sh
# kurt installer
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/strk/kurt/main/install.sh | sh
#
# Environment variables:
#   KURT_VERSION  Tag to install (e.g. v1.2.3). Defaults to "latest".
#   KURT_PREFIX   Install dir override. Default: /usr/local/bin, falling back
#                 to $HOME/.local/bin if the former is not writable.

set -eu

REPO="strk/kurt"
BIN_NAME="kurt"
VERSION="${KURT_VERSION:-latest}"

# ---- pretty output ----------------------------------------------------------

info()  { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
warn()  { printf '\033[1;33m!! \033[0m %s\n' "$*" >&2; }
err()   { printf '\033[1;31mxx \033[0m %s\n' "$*" >&2; }
done_() { printf '\033[1;32mok \033[0m %s\n' "$*"; }

# ---- detect downloader ------------------------------------------------------

if command -v curl >/dev/null 2>&1; then
  DL="curl"
elif command -v wget >/dev/null 2>&1; then
  DL="wget"
else
  err "Neither 'curl' nor 'wget' is installed. Please install one and retry."
  exit 1
fi

download() {
  # download <url> <out>
  url="$1"; out="$2"
  if [ "$DL" = "curl" ]; then
    curl -fsSL -o "$out" "$url"
  else
    wget -qO "$out" "$url"
  fi
}

follow_redirect() {
  # follow_redirect <url> -> prints final URL
  url="$1"
  if [ "$DL" = "curl" ]; then
    curl -fsSLI -o /dev/null -w '%{url_effective}' "$url"
  else
    wget --max-redirect=5 --server-response --spider "$url" 2>&1 \
      | awk '/^  Location: /{loc=$2} END{print loc}'
  fi
}

# ---- detect OS / ARCH -------------------------------------------------------

uname_s="$(uname -s)"
uname_m="$(uname -m)"

case "$uname_s" in
  Darwin) GOOS="darwin" ;;
  Linux)  GOOS="linux"  ;;
  *)
    err "Unsupported OS: $uname_s (only Darwin and Linux are supported)"
    exit 1
    ;;
esac

case "$uname_m" in
  x86_64|amd64)         GOARCH="amd64" ;;
  arm64|aarch64)        GOARCH="arm64" ;;
  *)
    err "Unsupported architecture: $uname_m"
    exit 1
    ;;
esac

info "Detected $GOOS/$GOARCH"

# ---- resolve version --------------------------------------------------------

if [ "$VERSION" = "latest" ]; then
  # GitHub redirects /releases/latest to /releases/tag/<tag>.
  latest_url="https://github.com/$REPO/releases/latest"
  final="$(follow_redirect "$latest_url" || true)"
  case "$final" in
    */releases/tag/*)
      VERSION="${final##*/releases/tag/}"
      ;;
    *)
      warn "Could not resolve latest version from GitHub; falling back to 'latest' tag."
      VERSION="latest"
      ;;
  esac
fi

info "Installing kurt $VERSION"

# ---- pick install dir -------------------------------------------------------

PRIMARY_DIR="${KURT_PREFIX:-/usr/local/bin}"
FALLBACK_DIR="$HOME/.local/bin"

writable() {
  d="$1"
  [ -d "$d" ] || mkdir -p "$d" 2>/dev/null || return 1
  # touch a temp file to confirm write perm
  t="$d/.kurt-write-test.$$"
  ( : > "$t" ) 2>/dev/null || return 1
  rm -f "$t"
  return 0
}

if writable "$PRIMARY_DIR"; then
  INSTALL_DIR="$PRIMARY_DIR"
else
  warn "$PRIMARY_DIR is not writable; falling back to $FALLBACK_DIR"
  mkdir -p "$FALLBACK_DIR"
  INSTALL_DIR="$FALLBACK_DIR"
fi

# ---- download ---------------------------------------------------------------

ASSET="kurt_${GOOS}_${GOARCH}"
URL="https://github.com/$REPO/releases/download/$VERSION/$ASSET"

TMPDIR="$(mktemp -d 2>/dev/null || mktemp -d -t kurt)"
trap 'rm -rf "$TMPDIR"' EXIT INT TERM

TMP_BIN="$TMPDIR/$BIN_NAME"
info "Downloading $URL"
if ! download "$URL" "$TMP_BIN"; then
  err "Download failed. Check that release $VERSION exists for $GOOS/$GOARCH."
  err "URL: $URL"
  exit 1
fi

# Sanity check: binary should be > 0 bytes
if [ ! -s "$TMP_BIN" ]; then
  err "Downloaded file is empty: $URL"
  exit 1
fi

chmod +x "$TMP_BIN"

# ---- install ----------------------------------------------------------------

TARGET="$INSTALL_DIR/$BIN_NAME"
if ! mv "$TMP_BIN" "$TARGET" 2>/dev/null; then
  # mv across devices or permission issue: try cp + rm
  if ! cp "$TMP_BIN" "$TARGET"; then
    err "Failed to install to $TARGET"
    exit 1
  fi
fi

done_ "Installed kurt to $TARGET"

# ---- PATH hint --------------------------------------------------------------

case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *)
    warn "$INSTALL_DIR is not on your PATH."
    warn "Add this to your shell rc file:"
    warn "    export PATH=\"$INSTALL_DIR:\$PATH\""
    ;;
esac

# ---- closing message --------------------------------------------------------

cat <<EOF

kurt $VERSION is installed.

Next steps:
  - Initialize your shell:
      kurt init zsh >> ~/.zshrc && source ~/.zshrc
  - Verify your install and dependencies:
      kurt doctor

EOF
