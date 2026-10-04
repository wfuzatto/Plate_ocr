#!/bin/sh
set -eu

if [ "$(id -u)" -ne 0 ]; then
  echo "ERRO: execute como root: sh offline/install.sh" >&2
  exit 1
fi

if ! id nvr >/dev/null 2>&1; then
  echo "ERRO: usuário nvr não existe. Instale o NVR antes do Plate OCR." >&2
  exit 1
fi

ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) SRC="bin/linux-amd64/plate-ocr" ;;
  aarch64|arm64) SRC="bin/linux-arm64/plate-ocr" ;;
  *) echo "ERRO: arquitetura não suportada: $ARCH" >&2; exit 1 ;;
esac

BASE="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
test -x "$BASE/$SRC" || { echo "ERRO: binário offline ausente: $BASE/$SRC" >&2; exit 1; }

if [ ! -f /var/lib/nvr/plugin.token ]; then
  echo "ERRO: /var/lib/nvr/plugin.token não existe." >&2
  echo "Inicie o NVR primeiro: systemctl restart nvr" >&2
  exit 1
fi

install -d -m 0750 -o nvr -g nvr /opt/nvr/plugins/plate-ocr /var/lib/plate-ocr/spool /etc/nvr/plugins
install -m 0755 "$BASE/$SRC" /opt/nvr/plugins/plate-ocr/plate-ocr

if [ ! -f /etc/nvr/plugins/plate-ocr.json ]; then
  install -m 0640 -o root -g nvr "$BASE/../configs/default.json" /etc/nvr/plugins/plate-ocr.json
fi

install -m 0644 "$BASE/../deploy/plate-ocr.service" /etc/systemd/system/plate-ocr.service
chown -R nvr:nvr /var/lib/plate-ocr
systemctl daemon-reload
systemctl enable plate-ocr.service
systemctl restart plate-ocr.service

sleep 1
if ! systemctl is-active --quiet plate-ocr.service; then
  echo "ERRO: Plate OCR não iniciou." >&2
  systemctl status plate-ocr.service --no-pager >&2 || true
  exit 1
fi

echo "Plate OCR instalado e iniciado sem acesso à Internet."
echo "Status: systemctl status plate-ocr --no-pager"
echo "Health: curl -fsS http://127.0.0.1:8091/healthz"
