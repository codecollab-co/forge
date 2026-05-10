#!/usr/bin/env sh
# install-forge.sh — install the forge CLI from a GitHub Release.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/codecollab-co/forge/main/scripts/install-forge.sh | sh
#   FORGE_VERSION=v0.1.0 sh install-forge.sh         # pin a version
#   FORGE_INSTALL_DIR=$HOME/.local/bin sh install-forge.sh
#
# Verifies the binary against the published checksums.txt before installing.
set -eu

REPO="codecollab-co/forge"
VERSION="${FORGE_VERSION:-latest}"
INSTALL_DIR="${FORGE_INSTALL_DIR:-/usr/local/bin}"

err() { printf 'install-forge: %s\n' "$*" >&2; exit 1; }

# --- Detect OS / arch ------------------------------------------------------
os="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$os" in
  darwin|linux) ;;
  *) err "unsupported OS: $os (use the Scoop bucket for Windows)" ;;
esac

arch="$(uname -m)"
case "$arch" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) err "unsupported arch: $arch" ;;
esac

# --- Resolve version -------------------------------------------------------
if [ "$VERSION" = "latest" ]; then
  VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -1)"
  [ -n "$VERSION" ] || err "could not resolve latest version"
fi
trimmed="${VERSION#v}"

archive="forge_${trimmed}_${os}_${arch}.tar.gz"
base="https://github.com/${REPO}/releases/download/${VERSION}"
sums_url="${base}/checksums.txt"
archive_url="${base}/${archive}"

# --- Download and verify ---------------------------------------------------
tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

printf 'Downloading %s\n' "$archive_url"
curl -fsSL "$archive_url" -o "$tmpdir/$archive"
curl -fsSL "$sums_url" -o "$tmpdir/checksums.txt"

if command -v sha256sum >/dev/null 2>&1; then
  ( cd "$tmpdir" && grep "  $archive\$" checksums.txt | sha256sum -c - ) || err "checksum mismatch"
elif command -v shasum >/dev/null 2>&1; then
  ( cd "$tmpdir" && grep "  $archive\$" checksums.txt | shasum -a 256 -c - ) || err "checksum mismatch"
else
  err "neither sha256sum nor shasum found; install one and re-run"
fi

# --- Extract + install -----------------------------------------------------
tar -xzf "$tmpdir/$archive" -C "$tmpdir"

if [ ! -w "$INSTALL_DIR" ]; then
  printf 'install-forge: %s is not writable, retrying with sudo\n' "$INSTALL_DIR" >&2
  sudo install -m 0755 "$tmpdir/forge" "$INSTALL_DIR/forge"
else
  install -m 0755 "$tmpdir/forge" "$INSTALL_DIR/forge"
fi

printf '\nInstalled %s to %s/forge\n' "$VERSION" "$INSTALL_DIR"
"$INSTALL_DIR/forge" --version || err "post-install check failed"
