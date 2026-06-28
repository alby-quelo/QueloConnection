<?php

declare(strict_types=1);

require __DIR__ . '/lib/bootstrap.php';
require_once __DIR__ . '/lib/terminal.php';

header('Content-Type: application/json; charset=utf-8');

$cfg = load_config();
start_session($cfg);

if (!is_logged_in()) {
    http_response_code(401);
    echo json_encode(['error' => 'Non autenticato']);
    exit;
}

if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
    http_response_code(405);
    echo json_encode(['error' => 'Metodo non consentito']);
    exit;
}

verify_csrf();

$machine = strtolower(trim((string) ($_POST['machine'] ?? '')));
if (!machine_name_valid($machine)) {
    http_response_code(400);
    echo json_encode(['error' => 'Nome macchina non valido']);
    exit;
}

try {
    $token = terminal_sign_token($machine, $cfg);
    echo json_encode([
        'machine' => $machine,
        'token' => $token,
        'ws_url' => terminal_ws_url($cfg),
    ], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
} catch (Throwable $e) {
    http_response_code(500);
    echo json_encode(['error' => $e->getMessage()]);
}
