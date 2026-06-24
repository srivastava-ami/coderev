#!/usr/bin/env bash
# install.sh — download and install the coderev binary
# Dependencies (gitleaks, semgrep, madge) are installed by the binary itself.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/srivastava-ami/coderev/main/scripts/install.sh | bash
#   bash scripts/install.sh [--prefix /usr/local] [--no-deps]
#
# Flags:
#   --prefix <dir>   install coderev under <dir>/bin  (default: ~/.local)
#   --no-deps        skip installing scanner dependencies

set -euo pipefail

REPO="srivastava-ami/coderev"
BIN="coderev"
DEFAULT_PREFIX="${HOME}/.local"
SKIP_DEPS=false
PREFIX=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --prefix)  PREFIX="${2:-}"; shift 2 ;;
    --no-deps) SKIP_DEPS=true; shift ;;
    *)         shift ;;
  esac
done
PREFIX="${PREFIX:-${CODEREV_PREFIX:-${DEFAULT_PREFIX}}}"
DEST="${PREFIX}/bin/${BIN}"

# ── detect OS / arch ─────────────────────────────────────────────────────────
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "${ARCH}" in
  x86_64)        ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "unsupported architecture: ${ARCH}" >&2; exit 1 ;;
esac

if [[ "${OS}" != "linux" && "${OS}" != "darwin" ]]; then
  echo "unsupported OS: ${OS}" >&2
  echo "Windows users: download from https://github.com/${REPO}/releases" >&2
  exit 1
fi

if command -v curl &>/dev/null; then FETCH="curl -fsSL"; else FETCH="wget -qO-"; fi

# ── download coderev binary ───────────────────────────────────────────────────
LATEST=$(${FETCH} "https://api.github.com/repos/${REPO}/releases/latest" \
         | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"\(.*\)".*/\1/')
if [[ -z "${LATEST}" ]]; then
  echo "could not determine latest release — check https://github.com/${REPO}/releases" >&2
  exit 1
fi

ASSET="${BIN}-${LATEST}-${OS}-${ARCH}"
URL="https://github.com/${REPO}/releases/download/${LATEST}/${ASSET}"

TMP="$(mktemp)"
trap 'rm -f "${TMP}"' EXIT

echo "installing coderev ${LATEST} (${OS}/${ARCH}) → ${DEST}"
${FETCH} "${URL}" -o "${TMP}"
chmod +x "${TMP}"
mkdir -p "$(dirname "${DEST}")"
mv "${TMP}" "${DEST}"
echo "✓ coderev installed"

# ── delegate dependency installation to the binary ───────────────────────────
if $SKIP_DEPS; then
  echo "(skipping dependencies — run later: coderev install-deps)"
else
  "${DEST}" install-deps
fi

echo ""
echo "done — run: coderev --version"
echo "(add ${PREFIX}/bin to your PATH if not already there)"
