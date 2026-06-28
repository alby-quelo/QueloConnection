#!/usr/bin/env bash
# Build nossh-client.zip ready to share.
# Usage:
#   bash scripts/pack-client.sh --server IP:7000 --machine nome-macchina
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="${ROOT}/dist/nossh-client"
ZIP="${ROOT}/dist/nossh-client.zip"
SERVER=""
MACHINE=""
LEGGIMI="${ROOT}/scripts/client-package/LEGGIMI.txt"

usage() {
  echo "Usage: pack-client.sh --server HOST:7000 --machine NAME" >&2
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --server) SERVER="$2"; shift 2 ;;
    --machine) MACHINE="$2"; shift 2 ;;
    --help) usage; exit 0 ;;
    *) echo "Unknown option: $1" >&2; usage; exit 1 ;;
  esac
done

if [[ -z "$SERVER" || -z "$MACHINE" ]]; then
  echo "Error: --server and --machine are required." >&2
  usage
  exit 1
fi

if [[ ! -f "${ROOT}/bin/nossh" ]]; then
  echo "Building nossh client binary..." >&2
  make -C "$ROOT" build
fi

rm -rf "$OUT"
mkdir -p "$OUT"

cp "${ROOT}/bin/nossh" "$OUT/"
cp "${ROOT}/scripts/install-client.sh" "$OUT/"
cp "$LEGGIMI" "$OUT/"
chmod +x "${OUT}/install-client.sh" "${OUT}/nossh"

cat >"${OUT}/INSTALL.txt" <<EOF
Comandi rapidi per questo pacchetto:

  bash install-client.sh --server ${SERVER} --machine ${MACHINE}
  connect-${MACHINE}

EOF

rm -f "$ZIP"
(cd "${ROOT}/dist" && zip -r nossh-client.zip nossh-client)

echo ""
echo "Pacchetto pronto:"
echo "  ${ZIP}"
echo ""
echo "Installazione per l'utente finale:"
echo "  bash install-client.sh --server ${SERVER} --machine ${MACHINE}"
echo ""
ls -lh "$OUT"
