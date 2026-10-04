#!/bin/sh
set -eu

if [ "$(id -u)" -ne 0 ]; then echo "run as root" >&2; exit 1; fi
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64) SRC="bin/linux-amd64/plate-ocr" ;;
  aarch64|arm64) SRC="bin/linux-arm64/plate-ocr" ;;
  *) echo "unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

BASE="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
test -x "$BASE/$SRC" || { echo "missing $BASE/$SRC" >&2; exit 1; }

install -d -m 0750 -o nvr -g nvr /opt/nvr/plugins/plate-ocr /var/lib/plate-ocr/spool /etc/nvr/plugins
install -m 0755 "$BASE/$SRC" /opt/nvr/plugins/plate-ocr/plate-ocr

if [ ! -f /etc/nvr/plugins/plate-ocr.json ]; then
  install -m 0640 -o root -g nvr "$BASE/../configs/default.json" /etc/nvr/plugins/plate-ocr.json
fi
install -m 0644 "$BASE/../deploy/plate-ocr.service" /etc/systemd/system/plate-ocr.service
systemctl daemon-reload
systemctl enable plate-ocr.service
echo "installed. Start with: systemctl start plate-ocr"
