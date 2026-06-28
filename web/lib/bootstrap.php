<?php

declare(strict_types=1);

require_once __DIR__ . '/debug.php';

function load_config(): array
{
    $path = __DIR__ . '/../config.php';
    if (!is_file($path)) {
        $path = __DIR__ . '/../config.example.php';
    }
    $cfg = require $path;
    if (!is_array($cfg)) {
        throw new RuntimeException('config.php deve restituire un array');
    }

    $token = getenv('NOSSH_ADMIN_TOKEN');
    if ($token !== false && $token !== '') {
        $cfg['admin_token'] = $token;
    }

    return $cfg;
}

/** Percorso URL della sotto-cartella, es. /quelo-admin (senza slash finale). */
function base_path(array $cfg): string
{
    if (isset($cfg['base_path']) && $cfg['base_path'] !== '') {
        $p = (string) $cfg['base_path'];
        if ($p[0] !== '/') {
            $p = '/' . $p;
        }
        return rtrim($p, '/');
    }
    $dir = str_replace('\\', '/', dirname($_SERVER['SCRIPT_NAME'] ?? '/'));
    if ($dir === '/' || $dir === '.' || $dir === '') {
        return '';
    }
    return rtrim($dir, '/');
}

function url(string $path, array $cfg): string
{
    $base = base_path($cfg);
    $path = ltrim($path, '/');
    if ($path === '') {
        return ($base === '' ? '' : $base) . '/';
    }
    return ($base === '' ? '' : $base) . '/' . $path;
}

function start_session(array $cfg): void
{
    if (session_status() === PHP_SESSION_ACTIVE) {
        return;
    }
    $path = base_path($cfg);
    $cookiePath = ($path === '') ? '/' : $path . '/';
    $secure = (!empty($_SERVER['HTTPS']) && $_SERVER['HTTPS'] !== 'off')
        || (($_SERVER['HTTP_X_FORWARDED_PROTO'] ?? '') === 'https');

    session_set_cookie_params([
        'lifetime' => 0,
        'path' => $cookiePath,
        'secure' => $secure,
        'httponly' => true,
        'samesite' => 'Lax',
    ]);
    session_start();
}

function h(?string $value): string
{
    return htmlspecialchars((string) $value, ENT_QUOTES | ENT_SUBSTITUTE, 'UTF-8');
}

function csrf_token(): string
{
    if (empty($_SESSION['csrf'])) {
        $_SESSION['csrf'] = bin2hex(random_bytes(16));
    }
    return $_SESSION['csrf'];
}

function verify_csrf(): void
{
    $sent = $_POST['csrf'] ?? '';
    if ($sent === '' || !hash_equals(csrf_token(), $sent)) {
        http_response_code(400);
        exit('Richiesta non valida (CSRF).');
    }
}

function flash_set(string $type, string $message): void
{
    $_SESSION['flash'] = ['type' => $type, 'message' => $message];
}

function flash_get(): ?array
{
    if (empty($_SESSION['flash'])) {
        return null;
    }
    $flash = $_SESSION['flash'];
    unset($_SESSION['flash']);
    return $flash;
}

function redirect(string $path, array $cfg): void
{
    header('Location: ' . url($path, $cfg));
    exit;
}

function machine_name_valid(string $name): bool
{
    $name = trim($name);
    if ($name === '' || strlen($name) > 64) {
        return false;
    }
    return (bool) preg_match('/^[a-z0-9]([a-z0-9-]*[a-z0-9])?$/', $name);
}

function admin_client(array $cfg): NosshAdmin
{
    require_once __DIR__ . '/NosshAdmin.php';
    $token = $_SESSION['admin_token'] ?? ($cfg['admin_token'] ?? '');
    return new NosshAdmin(
        $cfg['api_url'] ?? 'http://127.0.0.1:8081',
        (string) $token,
        debug_enabled($cfg)
    );
}

function is_logged_in(): bool
{
    return !empty($_SESSION['admin_token']) || !empty($_SESSION['admin_ok']);
}
