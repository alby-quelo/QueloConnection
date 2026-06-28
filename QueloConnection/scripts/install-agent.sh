#!/usr/bin/env bash
# Install nossh agent on a remote Linux machine.
# Usage: curl -fsSL https://example.com/install-agent.sh | sudo bash -s -- \
#          --server bridge.example.com:4443 \
#          --token YOUR_INSTALL_TOKEN
set -euo pipefail

SERVER=""
TOKEN=""
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/nossh"
CONFIG_FILE="${CONFIG_DIR}/agent.yaml"

usage() {
  cat <<'EOF'
Usage: install-agent.sh --server HOST:4443 --token TOKEN

Options:
  --server   Bridge server address (host:port, agent port)
  --token    Install token provided by the bridge admin
  --help     Show this help
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --server) SERVER="$2"; shift 2 ;;
    --token) TOKEN="$2"; shift 2 ;;
    --help) usage; exit 0 ;;
    *) echo "Unknown option: $1" >&2; usage; exit 1 ;;
  esac
done

if [[ -z "$SERVER" || -z "$TOKEN" ]]; then
  echo "Error: --server and --token are required" >&2
  usage
  exit 1
fi

if [[ "$(id -u)" -ne 0 ]]; then
  echo "Run as root (sudo)" >&2
  exit 1
fi

ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

# When distributing releases, set NOSSH_DOWNLOAD_BASE to your release URL.
DOWNLOAD_BASE="${NOSSH_DOWNLOAD_BASE:-}"
BIN="${INSTALL_DIR}/nossh-agent"

mkdir -p "$INSTALL_DIR" "$CONFIG_DIR"

if [[ -n "$DOWNLOAD_BASE" ]]; then
  curl -fsSL "${DOWNLOAD_BASE}/nossh-agent-linux-${ARCH}" -o "$BIN"
  chmod 755 "$BIN"
elif command -v nossh-agent >/dev/null 2>&1; then
  cp "$(command -v nossh-agent)" "$BIN"
else
  echo "nossh-agent binary not found. Build locally or set NOSSH_DOWNLOAD_BASE." >&2
  exit 1
fi

if [[ -f "$CONFIG_FILE" ]]; then
  AGENT_CODE="$(grep '^code:' "$CONFIG_FILE" | awk '{print $2}' | tr -d '"')"
  AGENT_UUID="$(grep '^uuid:' "$CONFIG_FILE" | awk '{print $2}' | tr -d '"')"
  echo "Existing agent config found, keeping code $AGENT_CODE"
else
  AGENT_CODE="$("$BIN" init-config --server "$SERVER" --token "$TOKEN" --config "$CONFIG_FILE")"
fi

if command -v systemctl >/dev/null 2>&1; then
  cat >/etc/systemd/system/nossh-agent.service <<EOF
[Unit]
Description=nossh agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=${BIN} -config ${CONFIG_FILE}
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF
  systemctl daemon-reload
  systemctl enable --now nossh-agent.service
else
  echo "systemd not found; start manually: $BIN -config $CONFIG_FILE" >&2
fi

HOSTNAME="$(hostname)"
CODE="$(grep '^code:' "$CONFIG_FILE" | awk '{print $2}' | tr -d '"')"

cat <<EOF

╔══════════════════════════════════════╗
║  nossh agent installato              ║
║                                      ║
║  Codice macchina:  ${CODE}
║  Hostname:         ${HOSTNAME}
║                                      ║
║  Comunica il codice all'amministratore
║  del server ponte per abilitare l'accesso.
╚══════════════════════════════════════╝

EOF
