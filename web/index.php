<?php

declare(strict_types=1);

require __DIR__ . '/lib/bootstrap.php';

$cfg = load_config();
init_debug($cfg);
start_session($cfg);
$appName = $cfg['app_name'] ?? 'Quelo Connect — Admin';
$apiDebug = null;

// Logout
if (isset($_GET['logout'])) {
    unset($_SESSION['admin_token'], $_SESSION['admin_ok']);
    redirect('index.php', $cfg);
}

// Login
if ($_SERVER['REQUEST_METHOD'] === 'POST' && ($_POST['action'] ?? '') === 'login') {
    verify_csrf();
    $token = trim((string) ($_POST['admin_token'] ?? ''));
    if ($token === '' && !empty($cfg['admin_token'])) {
        $token = (string) $cfg['admin_token'];
    }
    if ($token === '') {
        flash_set('error', 'Inserisci il token admin.');
        redirect('index.php', $cfg);
    }
    try {
        require_once __DIR__ . '/lib/NosshAdmin.php';
        $api = new NosshAdmin(
            $cfg['api_url'] ?? 'http://127.0.0.1:8081',
            $token,
            debug_enabled($cfg)
        );
        if (!$api->ping()) {
            $apiDebug = $api->getLastDebug();
            throw new RuntimeException('Token rifiutato dal server ponte');
        }
        $_SESSION['admin_token'] = $token;
        $_SESSION['admin_ok'] = true;
        flash_set('ok', 'Accesso effettuato.');
    } catch (Throwable $e) {
        if ($apiDebug === null && isset($api)) {
            $apiDebug = $api->getLastDebug();
        }
        flash_set('error', debug_format_exception_message($e, $cfg));
        debug_log_error('Login fallito', debug_throwable_context($e));
    }
    redirect('index.php', $cfg);
}

// Actions (logged in)
if ($_SERVER['REQUEST_METHOD'] === 'POST' && is_logged_in()) {
    verify_csrf();
    $action = $_POST['action'] ?? '';
    $api = admin_client($cfg);

    try {
        switch ($action) {
            case 'assign':
                $code = (string) ($_POST['code'] ?? '');
                $name = strtolower(trim((string) ($_POST['name'] ?? '')));
                if (!machine_name_valid($name)) {
                    throw new RuntimeException('Nome macchina non valido (usa lettere minuscole, cifre e trattino).');
                }
                $api->assignName($code, $name);
                flash_set('ok', "Nome \"$name\" assegnato all'agent $code.");
                break;

            case 'rename':
                $name = strtolower(trim((string) ($_POST['name'] ?? '')));
                $newName = strtolower(trim((string) ($_POST['new_name'] ?? '')));
                if (!machine_name_valid($name) || !machine_name_valid($newName)) {
                    throw new RuntimeException('Nome macchina non valido.');
                }
                $api->rename($name, $newName);
                flash_set('ok', "Macchina rinominata: $name → $newName.");
                break;

            case 'revoke':
                $name = strtolower(trim((string) ($_POST['name'] ?? '')));
                if (!machine_name_valid($name)) {
                    throw new RuntimeException('Nome macchina non valido.');
                }
                $api->revoke($name);
                flash_set('ok', "Macchina \"$name\" revocata.");
                break;

            case 'delete':
                $code = (string) ($_POST['code'] ?? '');
                $api->delete($code);
                flash_set('ok', "Agent $code eliminato dal registro.");
                break;

            default:
                throw new RuntimeException('Azione sconosciuta.');
        }
    } catch (Throwable $e) {
        $apiDebug = $api->getLastDebug();
        flash_set('error', debug_format_exception_message($e, $cfg));
        debug_log_error('Azione fallita: ' . $action, array_merge(
            debug_throwable_context($e),
            ['api' => $apiDebug]
        ));
    }
    redirect('index.php', $cfg);
}

$flash = flash_get();
$agents = [];
$apiError = null;

if (is_logged_in()) {
    try {
        $client = admin_client($cfg);
        $agents = $client->listAgents();
        $apiDebug = $client->getLastDebug();
        if (!is_array($agents)) {
            $agents = [];
        }
        usort($agents, static function ($a, $b) {
            $oa = !empty($a['online']) ? 0 : 1;
            $ob = !empty($b['online']) ? 0 : 1;
            if ($oa !== $ob) {
                return $oa <=> $ob;
            }
            return strcmp($a['hostname'] ?? '', $b['hostname'] ?? '');
        });
    } catch (Throwable $e) {
        $apiError = debug_format_exception_message($e, $cfg);
        if (isset($client)) {
            $apiDebug = $client->getLastDebug();
        }
        debug_log_error('listAgents fallito', debug_throwable_context($e));
    }
}

?><!DOCTYPE html>
<html lang="it">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title><?= h($appName) ?></title>
    <link rel="stylesheet" href="<?= h(url('assets/style.css', $cfg)) ?>">
</head>
<body>
<div class="wrap">
    <header class="top">
        <h1><?= h($appName) ?></h1>
        <?php if (is_logged_in()): ?>
            <a class="btn btn-ghost" href="<?= h(url('index.php?logout=1', $cfg)) ?>">Esci</a>
        <?php endif; ?>
    </header>

    <?php if ($flash): ?>
        <div class="flash flash-<?= h($flash['type']) ?>"><?= h($flash['message']) ?></div>
    <?php endif; ?>

    <?php if (!is_logged_in()): ?>
        <section class="card login-card">
            <h2>Accesso</h2>
            <p class="hint">Token admin (stesso valore di <code>admin_token</code> in <code>/etc/nossh/server.yaml</code>).</p>
            <form method="post">
                <input type="hidden" name="action" value="login">
                <input type="hidden" name="csrf" value="<?= h(csrf_token()) ?>">
                <label>
                    Token admin
                    <input type="password" name="admin_token" autocomplete="current-password" required>
                </label>
                <button type="submit" class="btn btn-primary">Entra</button>
            </form>
        </section>
    <?php else: ?>
        <?php if ($apiError): ?>
            <div class="flash flash-error"><?= h($apiError) ?></div>
            <p class="hint">Verifica che <code>nossh-server</code> sia attivo e che l'API risponda su <?= h($cfg['api_url'] ?? '') ?>.</p>
        <?php else: ?>
            <section class="card">
                <div class="card-head">
                    <h2>Agent registrati</h2>
                    <span class="badge"><?= count($agents) ?> totali</span>
                </div>

                <?php if (count($agents) === 0): ?>
                    <p class="empty">Nessun agent connesso al ponte. Installa <code>nossh-agent</code> sulla macchina remota.</p>
                <?php else: ?>
                    <div class="table-wrap">
                        <table>
                            <thead>
                            <tr>
                                <th>Codice</th>
                                <th>Hostname</th>
                                <th>Nome macchina</th>
                                <th>Stato</th>
                                <th>Online</th>
                                <th>Ultimo contatto</th>
                                <th>Azioni</th>
                                <th>Connetti</th>
                            </tr>
                            </thead>
                            <tbody>
                            <?php foreach ($agents as $agent): ?>
                                <?php
                                $code = $agent['code'] ?? '';
                                $hostname = $agent['hostname'] ?? '';
                                $name = $agent['name'] ?? '';
                                $status = $agent['status'] ?? '';
                                $online = !empty($agent['online']);
                                $lastSeen = $agent['last_seen'] ?? '';
                                ?>
                                <tr>
                                    <td><code><?= h($code) ?></code></td>
                                    <td><?= h($hostname) ?></td>
                                    <td><?= $name !== '' ? h($name) : '<span class="muted">—</span>' ?></td>
                                    <td><span class="status status-<?= h($status) ?>"><?= h($status) ?></span></td>
                                    <td><?= $online ? '<span class="online yes">sì</span>' : '<span class="online no">no</span>' ?></td>
                                    <td class="mono"><?= h($lastSeen) ?></td>
                                    <td class="actions">
                                        <?php if ($status === 'pending'): ?>
                                            <form method="post" class="inline-form">
                                                <input type="hidden" name="csrf" value="<?= h(csrf_token()) ?>">
                                                <input type="hidden" name="action" value="assign">
                                                <input type="hidden" name="code" value="<?= h($code) ?>">
                                                <input type="text" name="name" placeholder="nome-macchina" pattern="[a-z0-9]([a-z0-9-]*[a-z0-9])?" required>
                                                <button type="submit" class="btn btn-sm">Assegna nome</button>
                                            </form>
                                        <?php elseif ($status === 'active' && $name !== ''): ?>
                                            <details>
                                                <summary>Rinomina</summary>
                                                <form method="post" class="stack-form">
                                                    <input type="hidden" name="csrf" value="<?= h(csrf_token()) ?>">
                                                    <input type="hidden" name="action" value="rename">
                                                    <input type="hidden" name="name" value="<?= h($name) ?>">
                                                    <input type="text" name="new_name" placeholder="nuovo-nome" pattern="[a-z0-9]([a-z0-9-]*[a-z0-9])?" required>
                                                    <button type="submit" class="btn btn-sm">Rinomina</button>
                                                </form>
                                            </details>
                                            <form method="post" class="inline-form" onsubmit="return confirm('Revocare la macchina <?= h($name) ?>?');">
                                                <input type="hidden" name="csrf" value="<?= h(csrf_token()) ?>">
                                                <input type="hidden" name="action" value="revoke">
                                                <input type="hidden" name="name" value="<?= h($name) ?>">
                                                <button type="submit" class="btn btn-sm btn-warn">Revoca</button>
                                            </form>
                                        <?php endif; ?>
                                        <form method="post" class="inline-form" onsubmit="return confirm('Eliminare l\'agent <?= h($code) ?> dal registro?');">
                                            <input type="hidden" name="csrf" value="<?= h(csrf_token()) ?>">
                                            <input type="hidden" name="action" value="delete">
                                            <input type="hidden" name="code" value="<?= h($code) ?>">
                                            <button type="submit" class="btn btn-sm btn-danger">Elimina</button>
                                        </form>
                                    </td>
                                    <td class="connect-cell">
                                        <?php if ($status === 'active' && $online && $name !== ''): ?>
                                            <button type="button"
                                                    class="btn btn-sm btn-primary btn-connect"
                                                    data-machine="<?= h($name) ?>">CONNETTI</button>
                                        <?php else: ?>
                                            <span class="muted">—</span>
                                        <?php endif; ?>
                                    </td>
                                </tr>
                            <?php endforeach; ?>
                            </tbody>
                        </table>
                    </div>
                <?php endif; ?>
            </section>

            <section class="card legend">
                <h3>Legenda</h3>
                <ul>
                    <li><strong>pending</strong> — agent installato, in attesa che assegni il nome macchina.</li>
                    <li><strong>active</strong> — nome assegnato, i client possono connettersi con quel nome.</li>
                    <li><strong>revoked</strong> — accesso revocato.</li>
                </ul>
            </section>
            <div id="terminal-config" hidden
                 data-token-url="<?= h(url('terminal-token.php', $cfg)) ?>"
                 data-csrf="<?= h(csrf_token()) ?>"></div>
            <script src="<?= h(url('assets/terminal.js', $cfg)) ?>"></script>
        <?php endif; ?>
    <?php endif; ?>

    <?php debug_render_panel($cfg, $apiDebug); ?>

    <footer class="foot">
        <p>API: <?= h($cfg['api_url'] ?? '') ?> · nossh bridge admin</p>
    </footer>
</div>
</body>
</html>
