#!/usr/bin/env bash
# Install nossh-server on the bridge machine (Debian and other systemd distros).
set -euo pipefail

PREFIX="${PREFIX:-/usr/local}"
ETCDIR="/etc/nossh"

if [[ "$(id -u)" -ne 0 ]]; then
  echo "Run as root (sudo)" >&2
  exit 1
fi

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
make -C "$ROOT" build

install -d "${PREFIX}/bin" "$ETCDIR" /var/lib/nossh
install -m 755 "$ROOT/bin/nossh-server" "$ROOT/bin/nossh-agent" "$ROOT/bin/nossh" "${PREFIX}/bin/"
if [[ ! -f "$ETCDIR/server.yaml" ]]; then
  install -m 600 "$ROOT/deploy/server.yaml.example" "$ETCDIR/server.yaml"
  echo "Created $ETCDIR/server.yaml — set install_token and admin_token before starting."
fi

if command -v systemctl >/dev/null 2>&1; then
  install -m 644 "$ROOT/deploy/systemd/nossh-server.service" /etc/systemd/system/
  systemctl daemon-reload
  echo "Enable with: systemctl enable --now nossh-server"
else
  echo "systemd not found; start manually: nossh-server -config $ETCDIR/server.yaml"
fi

echo "Admin commands (on bridge server):"
echo "  export NOSSH_ADMIN_TOKEN=...  # same as admin_token in server.yaml"
echo "  nossh list"
echo "  nossh name CODE macchina-nome"
