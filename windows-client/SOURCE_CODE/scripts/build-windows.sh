#!/usr/bin/env bash
# Build nossh.exe from windows/SOURCE_CODE (run on Linux with Go installed).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="${ROOT}/../dist/nossh.exe"

mkdir -p "$(dirname "$OUT")"
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o "$OUT" "${ROOT}/cmd/nossh"
echo "Built: $OUT"
