<?php

declare(strict_types=1);

/** @var list<array{time:string,level:string,message:string,context:array}> */
$GLOBALS['_nossh_debug_log'] = [];

function debug_enabled(array $cfg): bool
{
    $env = getenv('NOSSH_DEBUG');
    if ($env !== false && $env !== '' && $env !== '0') {
        return true;
    }
    return !empty($cfg['debug']);
}

function debug_log(string $message, array $context = []): void
{
    $entry = [
        'time' => date('Y-m-d H:i:s'),
        'level' => 'info',
        'message' => $message,
        'context' => $context,
    ];
    $GLOBALS['_nossh_debug_log'][] = $entry;

    $logDir = __DIR__ . '/../logs';
    if (!is_dir($logDir)) {
        @mkdir($logDir, 0750, true);
    }
    $line = json_encode($entry, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
    if ($line !== false) {
        @file_put_contents($logDir . '/debug.log', $line . PHP_EOL, FILE_APPEND | LOCK_EX);
    }
}

function debug_log_error(string $message, array $context = []): void
{
    $entry = [
        'time' => date('Y-m-d H:i:s'),
        'level' => 'error',
        'message' => $message,
        'context' => $context,
    ];
    $GLOBALS['_nossh_debug_log'][] = $entry;

    $logDir = __DIR__ . '/../logs';
    if (!is_dir($logDir)) {
        @mkdir($logDir, 0750, true);
    }
    $line = json_encode($entry, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
    if ($line !== false) {
        @file_put_contents($logDir . '/debug.log', $line . PHP_EOL, FILE_APPEND | LOCK_EX);
    }
}

function debug_entries(): array
{
    return $GLOBALS['_nossh_debug_log'] ?? [];
}

function init_debug(array $cfg): void
{
    if (!debug_enabled($cfg)) {
        return;
    }

    ini_set('display_errors', '1');
    ini_set('display_startup_errors', '1');
    ini_set('log_errors', '1');
    error_reporting(E_ALL);

    $logDir = __DIR__ . '/../logs';
    if (!is_dir($logDir)) {
        @mkdir($logDir, 0750, true);
    }
    ini_set('error_log', $logDir . '/php-errors.log');

    set_error_handler(static function (int $severity, string $message, string $file, int $line): bool {
        if (!(error_reporting() & $severity)) {
            return false;
        }
        $text = sprintf('PHP [%s] %s in %s:%d', debug_severity_name($severity), $message, $file, $line);
        debug_log_error($text, [
            'severity' => $severity,
            'file' => $file,
            'line' => $line,
        ]);
        if (PHP_SAPI !== 'cli') {
            throw new ErrorException($message, 0, $severity, $file, $line);
        }
        return true;
    });

    set_exception_handler(static function (Throwable $e): void {
        debug_log_error('Eccezione non gestita', debug_throwable_context($e));
        if (PHP_SAPI === 'cli') {
            fwrite(STDERR, debug_format_throwable($e) . PHP_EOL);
            exit(1);
        }
        debug_render_fatal_page($e);
        exit(1);
    });

    register_shutdown_function(static function (): void {
        $err = error_get_last();
        if ($err === null) {
            return;
        }
        $fatal = [E_ERROR, E_PARSE, E_CORE_ERROR, E_COMPILE_ERROR, E_USER_ERROR];
        if (!in_array($err['type'], $fatal, true)) {
            return;
        }
        $text = sprintf(
            'Fatal [%s] %s in %s:%d',
            debug_severity_name($err['type']),
            $err['message'],
            $err['file'],
            $err['line']
        );
        debug_log_error($text, $err);
        if (PHP_SAPI !== 'cli' && !headers_sent()) {
            http_response_code(500);
            debug_render_fatal_page(new ErrorException($err['message'], 0, $err['type'], $err['file'], $err['line']));
        }
    });

    debug_log('Debug attivo', [
        'php' => PHP_VERSION,
        'sapi' => PHP_SAPI,
        'script' => $_SERVER['SCRIPT_NAME'] ?? '',
        'method' => $_SERVER['REQUEST_METHOD'] ?? '',
        'uri' => $_SERVER['REQUEST_URI'] ?? '',
    ]);
}

function debug_severity_name(int $severity): string
{
    return match ($severity) {
        E_ERROR => 'E_ERROR',
        E_WARNING => 'E_WARNING',
        E_PARSE => 'E_PARSE',
        E_NOTICE => 'E_NOTICE',
        E_CORE_ERROR => 'E_CORE_ERROR',
        E_CORE_WARNING => 'E_CORE_WARNING',
        E_COMPILE_ERROR => 'E_COMPILE_ERROR',
        E_COMPILE_WARNING => 'E_COMPILE_WARNING',
        E_USER_ERROR => 'E_USER_ERROR',
        E_USER_WARNING => 'E_USER_WARNING',
        E_USER_NOTICE => 'E_USER_NOTICE',
        E_STRICT => 'E_STRICT',
        E_RECOVERABLE_ERROR => 'E_RECOVERABLE_ERROR',
        E_DEPRECATED => 'E_DEPRECATED',
        E_USER_DEPRECATED => 'E_USER_DEPRECATED',
        default => 'UNKNOWN(' . $severity . ')',
    };
}

function debug_throwable_context(Throwable $e): array
{
    return [
        'class' => $e::class,
        'message' => $e->getMessage(),
        'code' => $e->getCode(),
        'file' => $e->getFile(),
        'line' => $e->getLine(),
        'trace' => $e->getTraceAsString(),
        'previous' => $e->getPrevious() ? debug_throwable_context($e->getPrevious()) : null,
    ];
}

function debug_format_throwable(Throwable $e): string
{
    $out = sprintf(
        "[%s] %s\n  in %s:%d\n",
        $e::class,
        $e->getMessage(),
        $e->getFile(),
        $e->getLine()
    );
    $out .= $e->getTraceAsString();
    $prev = $e->getPrevious();
    if ($prev !== null) {
        $out .= "\n\nCaused by:\n" . debug_format_throwable($prev);
    }
    return $out;
}

function debug_format_exception_message(Throwable $e, array $cfg): string
{
    $msg = $e->getMessage();
    if (!debug_enabled($cfg)) {
        return $msg;
    }
    return $msg . ' [' . basename($e->getFile()) . ':' . $e->getLine() . ']';
}

function debug_mask_token(?string $token): string
{
    if ($token === null || $token === '') {
        return '(vuoto)';
    }
    $len = strlen($token);
    if ($len <= 8) {
        return str_repeat('*', $len);
    }
    return substr($token, 0, 4) . '…' . substr($token, -4) . " ($len char)";
}

function debug_request_context(): array
{
    return [
        'time' => date('c'),
        'method' => $_SERVER['REQUEST_METHOD'] ?? '',
        'uri' => $_SERVER['REQUEST_URI'] ?? '',
        'script' => $_SERVER['SCRIPT_NAME'] ?? '',
        'query' => $_GET ?? [],
        'post_keys' => array_keys($_POST ?? []),
        'session_id' => session_id() ?: null,
        'remote_addr' => $_SERVER['REMOTE_ADDR'] ?? '',
        'https' => (!empty($_SERVER['HTTPS']) && $_SERVER['HTTPS'] !== 'off')
            || (($_SERVER['HTTP_X_FORWARDED_PROTO'] ?? '') === 'https'),
    ];
}

function debug_render_fatal_page(Throwable $e): void
{
    if (headers_sent()) {
        echo "\n<pre class=\"debug-fatal\">" . h(debug_format_throwable($e)) . "</pre>\n";
        return;
    }
    header('Content-Type: text/html; charset=utf-8');
    http_response_code(500);
    echo '<!DOCTYPE html><html lang="it"><head><meta charset="utf-8"><title>Errore</title>';
    echo '<style>body{font-family:monospace;background:#1a1a2e;color:#eee;padding:1.5rem}pre{white-space:pre-wrap;background:#111;padding:1rem;border:1px solid #c44;border-radius:6px}</style>';
    echo '</head><body><h1>Errore PHP (debug)</h1><pre>';
    echo h(debug_format_throwable($e));
    echo '</pre></body></html>';
}

function debug_render_panel(array $cfg, ?array $apiDebug = null): void
{
    if (!debug_enabled($cfg)) {
        return;
    }

    $configView = [
        'api_url' => $cfg['api_url'] ?? '',
        'base_path' => base_path($cfg),
        'app_name' => $cfg['app_name'] ?? '',
        'admin_token' => debug_mask_token($cfg['admin_token'] ?? ''),
        'session_token' => debug_mask_token($_SESSION['admin_token'] ?? null),
        'debug' => true,
    ];

    echo '<section class="debug-panel">';
    echo '<h3>Debug</h3>';
    echo '<details open><summary>Richiesta</summary><pre>';
    echo h(json_encode(debug_request_context(), JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES) ?: '{}');
    echo '</pre></details>';

    echo '<details><summary>Config (token mascherato)</summary><pre>';
    echo h(json_encode($configView, JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES) ?: '{}');
    echo '</pre></details>';

    if ($apiDebug !== null) {
        echo '<details open><summary>Ultima chiamata API</summary><pre>';
        echo h(json_encode($apiDebug, JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES) ?: '{}');
        echo '</pre></details>';
    }

    $entries = debug_entries();
    if ($entries !== []) {
        echo '<details><summary>Log sessione (' . count($entries) . ')</summary><pre>';
        echo h(json_encode($entries, JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES) ?: '[]');
        echo '</pre></details>';
    }

    echo '<p class="debug-hint">Log file: <code>logs/debug.log</code> e <code>logs/php-errors.log</code> — disattiva con <code>debug =&gt; false</code> in config.php</p>';
    echo '</section>';
}
