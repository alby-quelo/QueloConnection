#!/usr/bin/env bash
# Build portable Windows client and populate windows/dist + windows/SOURCE_CODE.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WIN="${ROOT}/windows"
DIST="${WIN}/dist"
SRC="${WIN}/SOURCE_CODE"
ICON_SRC="${ROOT}/packaging/quelo-connect-gui/icons/quelo-connect-gui-256.png"
if [[ ! -f "$ICON_SRC" ]]; then
  ICON_SRC="${ROOT}/cmd/quelo-connect-gui-win/icon.png"
fi

echo "==> Building nossh.exe (Windows amd64)..."
mkdir -p "$DIST"
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o "${DIST}/nossh.exe" "${ROOT}/cmd/nossh"

echo "==> Building quelo-connect.exe (GUI)..."
if command -v x86_64-w64-mingw32-gcc >/dev/null; then
  CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc \
    go build -ldflags="-H windowsgui" -o "${DIST}/quelo-connect.exe" "${ROOT}/cmd/quelo-connect-gui-win"
else
  echo "   Salto GUI (installa gcc-mingw-w64-x86-64 oppure compila su Windows con scripts/build-windows-gui.bat)"
fi

echo "==> Populating windows/SOURCE_CODE..."
rm -rf "$SRC"
mkdir -p "${SRC}/scripts"
for item in cmd internal go.mod go.sum; do
  cp -r "${ROOT}/${item}" "${SRC}/"
done
cp "${ROOT}/scripts/build-windows.sh" "${ROOT}/scripts/build-windows-gui.sh" "${ROOT}/scripts/build-windows-gui.bat" "${SRC}/scripts/"
cp "${ROOT}/windows/LEGGIMI-SOURCE.txt" "${SRC}/LEGGIMI.txt"
cp "${ROOT}/Makefile.windows" "${SRC}/Makefile"
if [[ -f "$ICON_SRC" ]]; then
  mkdir -p "${SRC}/icons"
  cp "$ICON_SRC" "${SRC}/icons/quelo-connect-gui-256.png"
fi

echo "==> Syncing windows/dist templates..."
cp "${ROOT}/windows/client.conf.example" "${DIST}/client.conf.example"
cp "${ROOT}/windows/connect.bat" "${DIST}/connect.bat"
cp "${ROOT}/windows/connect-gui.bat" "${DIST}/connect-gui.bat"
cp "${ROOT}/windows/connect-gui-debug.bat" "${DIST}/connect-gui-debug.bat"
cp "${ROOT}/cmd/quelo-connect-gui-win/quelo-connect.exe.manifest" "${DIST}/quelo-connect.exe.manifest" 2>/dev/null || true
cp "${ROOT}/windows/LEGGIMI.txt" "${DIST}/LEGGIMI.txt"
if [[ -f "$ICON_SRC" ]]; then
  cp "$ICON_SRC" "${DIST}/quelo-connect.png"
fi

rm -f "${DIST}/client.conf"

echo ""
echo "Windows portable pronto:"
echo "  ${DIST}/"
ls -lh "${DIST}/" 2>/dev/null || true
