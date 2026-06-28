<?php

declare(strict_types=1);

class NosshAdmin
{
    private string $baseUrl;
    private string $token;
    private bool $debug;

    /** @var array<string, mixed>|null */
    private ?array $lastDebug = null;

    public function __construct(string $baseUrl, string $token, bool $debug = false)
    {
        $this->baseUrl = rtrim($baseUrl, '/');
        $this->token = $token;
        $this->debug = $debug;
    }

    /** @return array<string, mixed>|null */
    public function getLastDebug(): ?array
    {
        return $this->lastDebug;
    }

    public function listAgents(): array
    {
        return $this->request('GET', '/api/agents');
    }

    public function assignName(string $code, string $name): array
    {
        $code = $this->normalizeCode($code);
        return $this->request('POST', '/api/agents/' . rawurlencode($code) . '/name', [
            'name' => $name,
        ]);
    }

    public function rename(string $name, string $newName): array
    {
        return $this->request('POST', '/api/rename', [
            'name' => $name,
            'new_name' => $newName,
        ]);
    }

    public function revoke(string $name): void
    {
        $this->request('POST', '/api/revoke', ['name' => $name]);
    }

    public function delete(string $code): void
    {
        $code = $this->normalizeCode($code);
        $this->request('DELETE', '/api/agents/' . rawurlencode($code));
    }

    public function ping(): bool
    {
        try {
            $this->listAgents();
            return true;
        } catch (Throwable $e) {
            if ($this->debug) {
                debug_log_error('ping() fallito', debug_throwable_context($e));
            }
            return false;
        }
    }

    private function normalizeCode(string $code): string
    {
        return strtoupper(trim($code));
    }

    private function request(string $method, string $path, ?array $body = null)
    {
        if (!function_exists('curl_init')) {
            throw new RuntimeException('Estensione PHP curl non disponibile');
        }

        $url = $this->baseUrl . $path;
        $started = microtime(true);

        $ch = curl_init($url);
        if ($ch === false) {
            throw new RuntimeException('Impossibile inizializzare curl');
        }

        $headers = ['Accept: application/json'];
        if ($this->token !== '') {
            $headers[] = 'Authorization: Bearer ' . $this->token;
        }

        curl_setopt_array($ch, [
            CURLOPT_RETURNTRANSFER => true,
            CURLOPT_CUSTOMREQUEST => $method,
            CURLOPT_HTTPHEADER => $headers,
            CURLOPT_TIMEOUT => 15,
            CURLOPT_CONNECTTIMEOUT => 5,
        ]);

        $jsonBody = null;
        if ($body !== null) {
            $jsonBody = json_encode($body, JSON_UNESCAPED_UNICODE);
            $headers[] = 'Content-Type: application/json';
            curl_setopt($ch, CURLOPT_HTTPHEADER, $headers);
            curl_setopt($ch, CURLOPT_POSTFIELDS, $jsonBody);
        }

        if ($this->debug) {
            debug_log('API request', [
                'method' => $method,
                'url' => $url,
                'body' => $body,
            ]);
        }

        $raw = curl_exec($ch);
        $status = (int) curl_getinfo($ch, CURLINFO_HTTP_CODE);
        $curlErrno = curl_errno($ch);
        $curlError = curl_error($ch);
        $curlInfo = curl_getinfo($ch);
        curl_close($ch);

        $elapsedMs = round((microtime(true) - $started) * 1000, 1);

        $this->lastDebug = [
            'method' => $method,
            'url' => $url,
            'request_body' => $body,
            'http_status' => $status,
            'curl_errno' => $curlErrno,
            'curl_error' => $curlError,
            'response_raw' => is_string($raw) ? $raw : null,
            'response_bytes' => is_string($raw) ? strlen($raw) : 0,
            'elapsed_ms' => $elapsedMs,
            'curl_info' => [
                'primary_ip' => $curlInfo['primary_ip'] ?? null,
                'primary_port' => $curlInfo['primary_port'] ?? null,
                'total_time' => $curlInfo['total_time'] ?? null,
                'connect_time' => $curlInfo['connect_time'] ?? null,
            ],
        ];

        if ($raw === false) {
            $msg = $this->formatApiError('Errore di rete verso API admin', $curlError, $status);
            debug_log_error($msg, $this->lastDebug);
            throw new RuntimeException($msg);
        }

        if ($status === 204) {
            if ($this->debug) {
                debug_log('API response 204', $this->lastDebug);
            }
            return null;
        }

        if ($status === 401) {
            $msg = $this->formatApiError('Token admin non valido', $raw, $status);
            debug_log_error($msg, $this->lastDebug);
            throw new RuntimeException($msg);
        }

        if ($status >= 300) {
            $msg = $this->formatApiError(trim($raw) !== '' ? trim($raw) : 'HTTP ' . $status, $raw, $status);
            debug_log_error('API HTTP error', $this->lastDebug);
            throw new RuntimeException($msg);
        }

        if ($raw === '' || $raw === 'null') {
            return null;
        }

        $data = json_decode($raw, true);
        if (!is_array($data) && $data !== null) {
            $msg = $this->formatApiError('Risposta API non valida (JSON atteso)', $raw, $status);
            debug_log_error($msg, $this->lastDebug);
            throw new RuntimeException($msg);
        }

        if ($this->debug) {
            debug_log('API OK', [
                'url' => $url,
                'status' => $status,
                'elapsed_ms' => $elapsedMs,
                'agents_count' => is_array($data) ? count($data) : null,
            ]);
        }

        return $data;
    }

    private function formatApiError(string $base, string|false|null $raw, int $status): string
    {
        if (!$this->debug) {
            return $base;
        }
        $parts = [$base, 'HTTP ' . $status];
        if (is_string($raw) && $raw !== '') {
            $snippet = strlen($raw) > 500 ? substr($raw, 0, 500) . '…' : $raw;
            $parts[] = 'body: ' . $snippet;
        }
        $parts[] = 'url: ' . $this->baseUrl;
        return implode(' | ', $parts);
    }
}
