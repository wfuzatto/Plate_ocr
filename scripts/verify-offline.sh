#!/bin/sh
set -eu

test -f go.mod
if grep -Eq '^[[:space:]]*require[[:space:]]|^[[:space:]]*require[[:space:]]*\(' go.mod; then
  echo "go.mod contains external requirements; vendor them before release" >&2
  exit 1
fi

GOPROXY=off GOSUMDB=off go test ./...
CGO_ENABLED=0 GOPROXY=off GOSUMDB=off go build -trimpath -o /tmp/plate-ocr-offline-check ./cmd/plate-ocr
rm -f /tmp/plate-ocr-offline-check
echo "offline verification passed"
