#!/usr/bin/env bash
# Compila nossh-agent.exe (esegui dalla root SOURCE_CODE su Linux con Go).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
mkdir -p "${ROOT}/../ESEGUIBILI"
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o "${ROOT}/../ESEGUIBILI/nossh-agent.exe "${ROOT}/cmd/nossh-agent"
echo "Creato: ${ROOT}/../ESEGUIBILI/nossh-agent.exe"
