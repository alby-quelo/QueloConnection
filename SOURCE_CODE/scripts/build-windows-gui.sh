#!/usr/bin/env bash
# Build quelo-connect.exe (walk GUI) for Windows.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
GUI_DIR="${ROOT}/cmd/quelo-connect-gui-win"
OUT="${1:-${ROOT}/windows/dist/quelo-connect.exe}"

if ! command -v x86_64-w64-mingw32-gcc >/dev/null; then
  echo "Serve il cross-compiler MinGW:" >&2
  echo "  sudo apt install gcc-mingw-w64-x86-64" >&2
  exit 1
fi

mkdir -p "$(dirname "$OUT")"

ICON="${GUI_DIR}/quelo-connect.ico"
if [[ ! -f "$ICON" ]] && command -v magick >/dev/null; then
  magick "${GUI_DIR}/icon.png" -define icon:auto-resize=256,128,64,48,32,16 "$ICON"
elif [[ ! -f "$ICON" ]] && command -v convert >/dev/null; then
  convert "${GUI_DIR}/icon.png" -define icon:auto-resize=256,128,64,48,32,16 "$ICON"
fi

RSRC="$(go env GOPATH)/bin/rsrc"
if [[ -x "$RSRC" ]]; then
  RSRC_ARGS=(-manifest "${GUI_DIR}/quelo-connect.exe.manifest" -o "${GUI_DIR}/rsrc.syso" -arch amd64)
  if [[ -f "$ICON" ]]; then
    RSRC_ARGS=(-ico "$ICON" "${RSRC_ARGS[@]}")
  else
    echo "Nota: manca quelo-connect.ico; l'exe non avrà icona in Explorer." >&2
  fi
  "$RSRC" "${RSRC_ARGS[@]}"
else
  echo "Nota: rsrc non trovato, copia quelo-connect.exe.manifest accanto all'exe." >&2
  rm -f "${GUI_DIR}/rsrc.syso"
fi

CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc \
  go build -ldflags="-H windowsgui -s -w" -o "$OUT" "${GUI_DIR}"

cp "${GUI_DIR}/quelo-connect.exe.manifest" "$(dirname "$OUT")/quelo-connect.exe.manifest"

echo "Built: $OUT"
