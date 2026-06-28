#!/usr/bin/env bash
# Build quelo-connect-gui .deb package (GTK3, no Python).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PKG_NAME="quelo-connect-gui"
VERSION="0.1.0-beta"
ARCH="$(dpkg --print-architecture 2>/dev/null || echo amd64)"
STAGE="${ROOT}/dist/deb-build/${PKG_NAME}_${VERSION}_${ARCH}"
OUTPUT="${ROOT}/dist/${PKG_NAME}_${VERSION}_${ARCH}.deb"

echo "==> Checking build dependencies..."
for dep in pkg-config gcc go; do
  command -v "$dep" >/dev/null || { echo "Missing: $dep" >&2; exit 1; }
done

if ! pkg-config --exists gtk+-3.0; then
  echo "Install: sudo apt install libgtk-3-dev" >&2
  exit 1
fi

echo "==> Building ${PKG_NAME} (la prima compilazione GTK può richiedere alcuni minuti)..."
mkdir -p "${STAGE}/DEBIAN" "${STAGE}/usr/bin" "${STAGE}/usr/share/applications"
CGO_ENABLED=1 go build -o "${STAGE}/usr/bin/${PKG_NAME}" "${ROOT}/cmd/quelo-connect-gui"

cp "${ROOT}/packaging/quelo-connect-gui/DEBIAN/control" "${STAGE}/DEBIAN/"
cp "${ROOT}/packaging/quelo-connect-gui/usr/share/applications/quelo-connect-gui.desktop" \
   "${STAGE}/usr/share/applications/"

ICON_SRC="${ROOT}/packaging/quelo-connect-gui/icons"
for size in 48 64 128 256; do
  if [[ -f "${ICON_SRC}/quelo-connect-gui-${size}.png" ]]; then
    mkdir -p "${STAGE}/usr/share/icons/hicolor/${size}x${size}/apps"
    cp "${ICON_SRC}/quelo-connect-gui-${size}.png" \
       "${STAGE}/usr/share/icons/hicolor/${size}x${size}/apps/quelo-connect-gui.png"
  fi
done

chmod 755 "${STAGE}/usr/bin/${PKG_NAME}"
chmod 644 "${STAGE}/usr/share/applications/${PKG_NAME}.desktop"

echo "==> Building .deb..."
mkdir -p "${ROOT}/dist"
rm -f "$OUTPUT"
dpkg-deb --build --root-owner-group "$STAGE" "$OUTPUT"

echo ""
echo "Pacchetto creato:"
echo "  ${OUTPUT}"
echo ""
echo "Installazione (senza avviso permessi, copia prima in /tmp):"
echo "  cp \"${OUTPUT}\" /tmp/"
echo "  sudo apt install /tmp/${PKG_NAME}_${VERSION}_${ARCH}.deb"
