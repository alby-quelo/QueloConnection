<?php

declare(strict_types=1);

function terminal_sign_token(string $machine, array $cfg): string
{
    $secret = (string) ($cfg['admin_token'] ?? '');
    if ($secret === '') {
        throw new RuntimeException('admin_token non configurato');
    }
    $exp = time() + 120;
    $data = $machine . '|' . $exp;
    $sig = hash_hmac('sha256', $data, $secret);
    $raw = $data . '|' . $sig;
    return rtrim(strtr(base64_encode($raw), '+/', '-_'), '=');
}

function terminal_ws_url(array $cfg): string
{
    $path = url('ws/terminal', $cfg);
    $secure = (!empty($_SERVER['HTTPS']) && $_SERVER['HTTPS'] !== 'off')
        || (($_SERVER['HTTP_X_FORWARDED_PROTO'] ?? '') === 'https');
    $host = $_SERVER['HTTP_HOST'] ?? 'localhost';
    return ($secure ? 'wss' : 'ws') . '://' . $host . $path;
}
