#!/bin/sh
set -eu
BASE="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64) BIN="$BASE/bin/linux-amd64/plate-ocr" ;;
  aarch64|arm64) BIN="$BASE/bin/linux-arm64/plate-ocr" ;;
  *) echo "unsupported architecture: $ARCH" >&2; exit 1 ;;
esac
CONFIG="$BASE/../configs/default.json"
if [ "$#" -ge 1 ]; then CONFIG="$1"; fi
exec "$BIN" serve -config "$CONFIG"
