<?php
/**
 * Copia in config.php e adatta i valori.
 * L'API admin di nossh-server ascolta solo su 127.0.0.1 (vedi server.yaml).
 */
return [
    // URL base API admin (solo da localhost sul server ponte).
    'api_url' => 'http://127.0.0.1:8081',

    // Token admin (stesso valore di admin_token in /etc/nossh/server.yaml).
    // In alternativa imposta la variabile d'ambiente NOSSH_ADMIN_TOKEN.
    'admin_token' => '',

    // Percorso URL se installato come sotto-cartella (es. https://tuodominio.it/noddns/).
    // Lascia '' per rilevamento automatico dalla posizione di index.php.
    'base_path' => '/noddns',

    'app_name' => 'Quelo Connect — Admin',

    // true = errori a schermo, stack trace, log in logs/debug.log (disattiva in produzione stabile).
    // Oppure: NOSSH_DEBUG=1 nell'ambiente PHP-FPM.
    'debug' => true,
];
