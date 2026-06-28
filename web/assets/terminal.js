(function () {
    'use strict';

    var cfgEl = document.getElementById('terminal-config');
    if (!cfgEl) {
        return;
    }

    var tokenUrl = cfgEl.dataset.tokenUrl;
    var csrf = cfgEl.dataset.csrf;
    var activeRow = null;
    var activeWs = null;
    var activeTerm = null;
    var activeFit = null;

    function loadXterm(cb) {
        if (window.Terminal && window.FitAddon && window.FitAddon.FitAddon) {
            cb();
            return;
        }
        var link = document.createElement('link');
        link.rel = 'stylesheet';
        link.href = 'https://cdn.jsdelivr.net/npm/@xterm/xterm@5.5.0/css/xterm.min.css';
        document.head.appendChild(link);

        var scripts = [
            'https://cdn.jsdelivr.net/npm/@xterm/xterm@5.5.0/lib/xterm.min.js',
            'https://cdn.jsdelivr.net/npm/@xterm/addon-fit@0.10.0/lib/addon-fit.min.js',
        ];
        var loaded = 0;
        scripts.forEach(function (src) {
            var s = document.createElement('script');
            s.src = src;
            s.onload = function () {
                loaded += 1;
                if (loaded === scripts.length) {
                    cb();
                }
            };
            s.onerror = function () {
                alert('Impossibile caricare il terminale web (xterm.js).');
            };
            document.head.appendChild(s);
        });
    }

    function closeTerminal() {
        if (activeWs) {
            try {
                activeWs.close();
            } catch (e) {}
            activeWs = null;
        }
        if (activeTerm) {
            try {
                activeTerm.dispose();
            } catch (e) {}
            activeTerm = null;
        }
        activeFit = null;
        if (activeRow) {
            activeRow.remove();
            activeRow = null;
        }
        document.querySelectorAll('.btn-connect.is-active').forEach(function (btn) {
            btn.classList.remove('is-active');
            btn.textContent = 'CONNETTI';
        });
    }

    function sendResize(ws, term) {
        if (!ws || ws.readyState !== WebSocket.OPEN) {
            return;
        }
        ws.send(JSON.stringify({
            type: 'resize',
            cols: term.cols,
            rows: term.rows,
        }));
    }

    function openTerminal(machine, afterRow, btn) {
        closeTerminal();
        btn.classList.add('is-active');
        btn.textContent = 'CHIUDI';

        var row = document.createElement('tr');
        row.className = 'terminal-row';
        var cell = document.createElement('td');
        cell.colSpan = 8;
        cell.innerHTML =
            '<div class="terminal-panel">' +
            '<div class="terminal-head"><strong>SSH → ' + machine + '</strong>' +
            '<span class="terminal-status">Connessione…</span>' +
            '<button type="button" class="btn btn-sm btn-ghost terminal-close">Chiudi</button></div>' +
            '<div class="terminal-host"></div></div>';
        row.appendChild(cell);
        afterRow.insertAdjacentElement('afterend', row);
        activeRow = row;

        var statusEl = cell.querySelector('.terminal-status');
        var hostEl = cell.querySelector('.terminal-host');
        cell.querySelector('.terminal-close').addEventListener('click', closeTerminal);

        loadXterm(function () {
            var FitAddon = window.FitAddon.FitAddon;
            var term = new Terminal({
                cursorBlink: true,
                fontSize: 14,
                fontFamily: 'Menlo, Monaco, "Courier New", monospace',
                theme: {
                    background: '#0d1117',
                    foreground: '#e6edf3',
                },
            });
            var fit = new FitAddon();
            term.loadAddon(fit);
            term.open(hostEl);
            fit.fit();
            activeTerm = term;
            activeFit = fit;

            var form = new FormData();
            form.append('csrf', csrf);
            form.append('machine', machine);

            fetch(tokenUrl, { method: 'POST', body: form, credentials: 'same-origin' })
                .then(function (r) {
                    return r.json().then(function (data) {
                        if (!r.ok) {
                            throw new Error(data.error || 'Errore token');
                        }
                        return data;
                    });
                })
                .then(function (data) {
                    var url = data.ws_url +
                        '?machine=' + encodeURIComponent(data.machine) +
                        '&token=' + encodeURIComponent(data.token);
                    var ws = new WebSocket(url);
                    ws.binaryType = 'arraybuffer';
                    activeWs = ws;

                    ws.onopen = function () {
                        statusEl.textContent = 'Connesso — inserisci username e password SSH';
                        sendResize(ws, term);
                    };
                    ws.onmessage = function (ev) {
                        if (typeof ev.data === 'string') {
                            term.write(ev.data);
                        } else {
                            term.write(new Uint8Array(ev.data));
                        }
                    };
                    ws.onerror = function () {
                        statusEl.textContent = 'Errore WebSocket (verifica proxy Apache)';
                    };
                    ws.onclose = function () {
                        statusEl.textContent = 'Sessione chiusa';
                        term.write('\r\n\r\n[sessione terminata]\r\n');
                    };
                    term.onData(function (data) {
                        if (ws.readyState === WebSocket.OPEN) {
                            ws.send(data);
                        }
                    });
                    window.addEventListener('resize', function onResize() {
                        if (activeTerm !== term) {
                            window.removeEventListener('resize', onResize);
                            return;
                        }
                        fit.fit();
                        sendResize(ws, term);
                    });
                })
                .catch(function (err) {
                    statusEl.textContent = 'Errore: ' + err.message;
                    term.write('\r\n' + err.message + '\r\n');
                });
        });
    }

    document.addEventListener('click', function (ev) {
        var btn = ev.target.closest('.btn-connect');
        if (!btn) {
            return;
        }
        ev.preventDefault();
        var machine = btn.dataset.machine;
        var row = btn.closest('tr');
        if (!machine || !row) {
            return;
        }
        if (btn.classList.contains('is-active')) {
            closeTerminal();
            return;
        }
        openTerminal(machine, row, btn);
    });
})();
