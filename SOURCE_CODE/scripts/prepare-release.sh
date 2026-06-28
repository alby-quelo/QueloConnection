#!/usr/bin/env bash
# Prepare QueloConnection release and SOURCE_CODE tree.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
QC="${ROOT}/QueloConnection"
SC="${ROOT}/SOURCE_CODE"
GUI="${SC}/Gui"

echo "==> Building binaries..."
make -C "$ROOT" build

echo "==> Building GUI .deb..."
make -C "$ROOT" gui-deb
DEB="$(ls -t "${ROOT}"/dist/quelo-connect-gui_*.deb 2>/dev/null | head -1)"
if [[ -z "$DEB" || ! -f "$DEB" ]]; then
  echo "GUI .deb non trovato in dist/" >&2
  exit 1
fi

echo "==> Populating QueloConnection..."
mkdir -p "${QC}/bin" "${QC}/deploy" "${QC}/scripts" "${QC}/client-package"
cp "${ROOT}/LICENSE" "${QC}/"
cp "${ROOT}/AUTHOR.txt" "${QC}/" 2>/dev/null || true
cp "${ROOT}/bin/nossh-server" "${ROOT}/bin/nossh-agent" "${ROOT}/bin/nossh" "${QC}/bin/"
cp -r "${ROOT}/deploy/"* "${QC}/deploy/"
cp "${ROOT}/scripts/install-server.sh" \
   "${ROOT}/scripts/install-agent.sh" \
   "${ROOT}/scripts/install-client.sh" \
   "${ROOT}/scripts/pack-client.sh" \
   "${QC}/scripts/"
cp "${ROOT}/scripts/client-package/LEGGIMI.txt" "${QC}/client-package/"
cp "$DEB" "${QC}/"
cp "${ROOT}/packaging/quelo-connect-gui/LEGGIMI.txt" "${QC}/LEGGIMI-GUI.txt"
chmod +x "${QC}/scripts/"*.sh "${QC}/bin/"*

echo "==> Populating SOURCE_CODE..."
rm -rf "${SC}"
mkdir -p "${SC}/scripts/client-package"
for item in cmd internal deploy go.mod go.sum Makefile LICENSE packaging; do
  cp -r "${ROOT}/${item}" "${SC}/"
done
cp "${ROOT}/scripts/"*.sh "${SC}/scripts/"
cp "${ROOT}/scripts/client-package/LEGGIMI.txt" "${SC}/scripts/client-package/"
cp "${ROOT}/scripts/SOURCE_CODE-LEGGIMI.txt" "${SC}/LEGGIMI.txt"
cp "${ROOT}/AUTHOR.txt" "${SC}/" 2>/dev/null || true
chmod +x "${SC}/scripts/"*.sh

echo "==> Populating SOURCE_CODE/Gui..."
mkdir -p "${GUI}/cmd" "${GUI}/internal/client" "${GUI}/scripts"
cp -r "${ROOT}/cmd/quelo-connect-gui" "${GUI}/cmd/"
cp -r "${ROOT}/internal/launcher" "${ROOT}/internal/configfile" "${GUI}/internal/"
cp "${ROOT}/internal/client/check.go" "${GUI}/internal/client/"
cp -r "${ROOT}/packaging" "${GUI}/"
cp "${ROOT}/scripts/build-gui-deb.sh" "${GUI}/scripts/"
cp "${ROOT}/go.mod" "${ROOT}/go.sum" "${GUI}/"
cp "${ROOT}/scripts/SOURCE_CODE-Gui-LEGGIMI.txt" "${GUI}/LEGGIMI.txt"
cat > "${GUI}/Makefile" << 'EOF'
.PHONY: gui-deb
gui-deb:
	bash scripts/build-gui-deb.sh
EOF
chmod +x "${GUI}/scripts/build-gui-deb.sh"

echo ""
echo "Release pronta:"
echo "  QueloConnection/       → distribuzione pubblica (+ $(basename "$DEB"))"
echo "  SOURCE_CODE/           → sorgente completo"
echo "  SOURCE_CODE/Gui/       → sorgente quelo-connect-gui"
ls -lh "${QC}/bin/"
ls -lh "${QC}/"*.deb
