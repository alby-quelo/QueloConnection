#!/bin/bash
# Permessi pannello noddns: proprietario alby, gruppo www-data (PHP/Nginx).
# Uso sul server: sudo bash fix-permissions.sh /percorso/noddns

set -euo pipefail

DIR="${1:-}"
if [[ -z "$DIR" || ! -d "$DIR" ]]; then
    echo "Uso: sudo $0 /percorso/alla/cartella/noddns" >&2
    exit 1
fi

DIR="$(realpath "$DIR")"
echo "Imposto permessi su: $DIR"

chown -R alby:www-data "$DIR"

find "$DIR" -type d -exec chmod 2750 {} \;
find "$DIR" -type f -exec chmod 640 {} \;

# Log scrivibili da PHP (www-data)
if [[ -d "$DIR/logs" ]]; then
    chmod 2770 "$DIR/logs"
    chown alby:www-data "$DIR/logs"
fi

if [[ -f "$DIR/config.php" ]]; then
    chmod 640 "$DIR/config.php"
    chown alby:www-data "$DIR/config.php"
fi

echo "Fatto. Verifica:"
ls -la "$DIR"
[[ -d "$DIR/logs" ]] && ls -la "$DIR/logs"
