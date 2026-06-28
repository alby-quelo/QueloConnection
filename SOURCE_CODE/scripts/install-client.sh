#!/usr/bin/env bash
# Install nossh client for connecting to a remote machine via the bridge.
#
# Before sharing with someone, put this script and the "nossh" binary in the
# same folder, then zip and send:
#
#   nossh-client/
#     install-client.sh
#     nossh
#
# Usage:
#   bash install-client.sh
#   bash install-client.sh --server IP-PONTE:7000 --machine nome-macchina
#
# After install, connect with:
#   connect-nome-macchina
# or:
#   nossh connect nome-macchina
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SOURCE_BIN="${SCRIPT_DIR}/nossh"

SERVER="${NOSSH_SERVER:-}"
MACHINE="${NOSSH_MACHINE:-}"

# Install for the real user — never into /root when invoked with sudo by mistake.
TARGET_USER="${USER}"
TARGET_HOME="${HOME}"
if [[ $EUID -eq 0 ]]; then
  if [[ -n "${SUDO_USER:-}" ]]; then
    TARGET_USER="${SUDO_USER}"
    TARGET_HOME="$(getent passwd "${SUDO_USER}" | cut -d: -f6)"
    echo "Nota: installazione per l'utente ${TARGET_USER} (non usare sudo la prossima volta)."
  else
    echo "Errore: non eseguire come root." >&2
    echo "Usa:  bash install-client.sh" >&2
    exit 1
  fi
fi

INSTALL_DIR="${TARGET_HOME}/.local/bin"
CONFIG_DIR="${TARGET_HOME}/.config/nossh"
CONFIG_FILE="${CONFIG_DIR}/client.conf"
LOCAL_CONNECT="${SCRIPT_DIR}/connect-${MACHINE}"

usage() {
  cat <<EOF
Usage: $(basename "$0") [options]

Prepares the nossh client on this PC (no root required — do NOT use sudo).

Options:
  --server HOST:7000   Bridge server address (default: ${SERVER})
  --machine NAME       Machine name on the bridge (default: ${MACHINE})
  --help               Show this help

Requires:
  - the "nossh" binary in the same folder as this script
  - openssh-client (ssh command)
EOF
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
  echo "Error: --server and --machine are required (or set NOSSH_SERVER and NOSSH_MACHINE)." >&2
  echo "" >&2
  usage
  exit 1
fi

if ! command -v ssh >/dev/null 2>&1; then
  echo "Error: ssh not found. Install openssh-client first." >&2
  echo "  Debian/Ubuntu: sudo apt install openssh-client" >&2
  exit 1
fi

if [[ ! -f "$SOURCE_BIN" ]]; then
  cat >&2 <<EOF
Error: nossh binary not found at:
  ${SOURCE_BIN}

Put the compiled "nossh" binary in the same folder as this script, then run again.
EOF
  exit 1
fi

mkdir -p "$INSTALL_DIR" "$CONFIG_DIR"
install -m 755 "$SOURCE_BIN" "${INSTALL_DIR}/nossh"
if [[ $EUID -eq 0 ]]; then
  chown "${TARGET_USER}:${TARGET_USER}" "${INSTALL_DIR}/nossh"
fi

cat >"$CONFIG_FILE" <<EOF
# nossh client configuration
server=${SERVER}
machine=${MACHINE}
EOF
chmod 600 "$CONFIG_FILE"
if [[ $EUID -eq 0 ]]; then
  chown "${TARGET_USER}:${TARGET_USER}" "$CONFIG_FILE" "$CONFIG_DIR"
fi

CONNECT_CMD="connect-${MACHINE}"
cat >"${INSTALL_DIR}/${CONNECT_CMD}" <<EOF
#!/usr/bin/env bash
set -euo pipefail
# Quick connect to ${MACHINE} via nossh bridge.
exec "${INSTALL_DIR}/nossh" -server "${SERVER}" connect "${MACHINE}" "\$@"
EOF
chmod 755 "${INSTALL_DIR}/${CONNECT_CMD}"

cat >"${LOCAL_CONNECT}" <<EOF
#!/usr/bin/env bash
set -euo pipefail
exec "${INSTALL_DIR}/nossh" -server "${SERVER}" connect "${MACHINE}" "\$@"
EOF
chmod 755 "${LOCAL_CONNECT}"
if [[ $EUID -eq 0 ]]; then
  chown "${TARGET_USER}:${TARGET_USER}" "${INSTALL_DIR}/${CONNECT_CMD}" "${LOCAL_CONNECT}"
fi

# Optional: wrapper that reads config (for manual "nossh connect")
cat >"${INSTALL_DIR}/nossh-client" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
CONFIG="${HOME}/.config/nossh/client.conf"
SERVER="$(grep '^server=' "$CONFIG" | cut -d= -f2-)"
MACHINE="$(grep '^machine=' "$CONFIG" | cut -d= -f2-)"
BIN="${HOME}/.local/bin/nossh"
exec "$BIN" -server "$SERVER" connect "$MACHINE" "$@"
EOF
chmod 755 "${INSTALL_DIR}/nossh-client"
if [[ $EUID -eq 0 ]]; then
  chown "${TARGET_USER}:${TARGET_USER}" "${INSTALL_DIR}/nossh-client"
fi

SHELL_RC="${TARGET_HOME}/.bashrc"
if [[ ":${PATH}:" != *":${INSTALL_DIR}:"* ]]; then
  MARKER="# nossh client PATH"
  if [[ -f "$SHELL_RC" ]] && ! grep -qF "$MARKER" "$SHELL_RC" 2>/dev/null; then
    cat >>"$SHELL_RC" <<EOF

${MARKER}
export PATH="\${HOME}/.local/bin:\${PATH}"
EOF
    if [[ $EUID -eq 0 ]]; then
      chown "${TARGET_USER}:${TARGET_USER}" "$SHELL_RC"
    fi
    echo "Aggiunto ${INSTALL_DIR} al PATH in ~/.bashrc (nuovo terminale oppure: source ~/.bashrc)"
  else
    echo "Se serve, aggiungi al PATH:  export PATH=\"\${HOME}/.local/bin:\${PATH}\""
  fi
fi

cat <<EOF

╔══════════════════════════════════════╗
║  nossh client installato             ║
║                                      ║
║  Macchina:  ${MACHINE}
║  Ponte:     ${SERVER}
║                                      ║
║  Per connetterti:
║    ${CONNECT_CMD}
║    ./${CONNECT_CMD}
║                                      ║
║  Ti chiederà username e password
║  Linux della macchina remota.
╚══════════════════════════════════════╝

EOF
