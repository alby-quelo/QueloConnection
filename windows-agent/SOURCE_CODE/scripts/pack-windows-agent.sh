#!/usr/bin/env bash
# Sync SOURCE_CODE and build nossh-agent.exe into windows-agent/ESEGUIBILI/
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WA="${ROOT}/windows-agent"
SRC="${WA}/SOURCE_CODE"
BIN="${WA}/ESEGUIBILI"

echo "==> Building nossh-agent (Windows amd64)..."
mkdir -p "$BIN" "${SRC}/scripts"
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o "${BIN}/nossh-agent.exe" "${ROOT}/cmd/nossh-agent"

echo "==> Syncing SOURCE_CODE..."
rm -rf "${SRC}/cmd" "${SRC}/internal" "${SRC}/deploy" "${SRC}/go.mod" "${SRC}/go.sum" "${SRC}/LICENSE" "${SRC}/Makefile" "${SRC}/AUTHOR.txt" 2>/dev/null || true
for item in cmd internal deploy go.mod go.sum LICENSE; do
  cp -r "${ROOT}/${item}" "${SRC}/"
done
cp "${ROOT}/AUTHOR.txt" "${SRC}/" 2>/dev/null || true

cp "${ROOT}/scripts/pack-windows-agent.sh" "${SRC}/scripts/"

cat > "${SRC}/Makefile" << 'EOF'
.PHONY: agent
agent:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o ../ESEGUIBILI/nossh-agent.exe ./cmd/nossh-agent
EOF

cp "${ROOT}/windows-agent/SOURCE_CODE/scripts/install-openssh.ps1" "${BIN}/"
cp "${ROOT}/windows-agent/SOURCE_CODE/scripts/install-agent.ps1" "${BIN}/"
cp "${ROOT}/windows-agent/SOURCE_CODE/scripts/install-agent.bat" "${BIN}/"
cp "${ROOT}/windows-agent/SOURCE_CODE/scripts/uninstall-agent.ps1" "${BIN}/"
cp "${ROOT}/windows-agent/SOURCE_CODE/scripts/uninstall-agent.bat" "${BIN}/"

echo ""
echo "windows-agent pronto:"
ls -lh "${BIN}/"
