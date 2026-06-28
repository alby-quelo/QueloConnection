#Requires -RunAsAdministrator
param(
    [string]$BridgeHost = '',
    [string]$BridgePort = '',
    [string]$Token = '',
    [string]$Server = ''
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$InstallDir = Join-Path ${env:ProgramFiles} 'nossh'
$ConfigDir = Join-Path $env:ProgramData 'nossh'
$ConfigFile = Join-Path $ConfigDir 'agent.yaml'
$ServiceName = 'nossh-agent'
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$DefaultAgentPort = 4443

# Variabili script-level (evita problemi di scope in PowerShell)
$script:BridgeHost = $BridgeHost
$script:BridgePort = $BridgePort
$script:Token = $Token
$script:ServerURL = ''

function Write-Step($msg) { Write-Host ''; Write-Host "==> $msg" -ForegroundColor Cyan }
function Write-Ok($msg) { Write-Host "  OK  $msg" -ForegroundColor Green }
function Write-Fail($msg) { Write-Host "  ERR $msg" -ForegroundColor Red }

function Find-AgentExe {
    $candidates = @(
        (Join-Path $ScriptDir 'nossh-agent.exe'),
        (Join-Path $ScriptDir '..\ESEGUIBILI\nossh-agent.exe'),
        (Join-Path $ScriptDir '..\nossh-agent.exe')
    )
    foreach ($p in $candidates) {
        $full = [System.IO.Path]::GetFullPath($p)
        if (Test-Path $full) { return $full }
    }
    throw 'nossh-agent.exe non trovato accanto agli script di installazione.'
}

function Read-InstallParams {
    if ($Server -ne '') {
        if ($Server -match '^(.+):(\d+)$') {
            $script:BridgeHost = $Matches[1].Trim()
            $script:BridgePort = $Matches[2]
        } else {
            $script:BridgeHost = $Server.Trim()
            if ([string]::IsNullOrWhiteSpace($script:BridgePort)) {
                $script:BridgePort = [string]$DefaultAgentPort
            }
        }
    }

    Write-Host ''
    Write-Host '=== Installazione nossh agent (Windows) ===' -ForegroundColor Yellow
    Write-Host 'Dati sul server ponte: /etc/nossh/server.yaml' -ForegroundColor DarkGray
    Write-Host ''

    if ([string]::IsNullOrWhiteSpace($script:BridgeHost)) {
        $script:BridgeHost = (Read-Host 'a) IP o hostname del server ponte').Trim()
    }
    if ([string]::IsNullOrWhiteSpace($script:BridgeHost)) {
        throw 'Host del server ponte obbligatorio.'
    }

    if ([string]::IsNullOrWhiteSpace($script:BridgePort)) {
        $portInput = Read-Host "b) Porta agent sul ponte [$DefaultAgentPort]"
        if ([string]::IsNullOrWhiteSpace($portInput)) {
            $script:BridgePort = [string]$DefaultAgentPort
        } else {
            $script:BridgePort = $portInput.Trim()
        }
    }

    if (-not ($script:BridgePort -match '^\d+$') -or [int]$script:BridgePort -lt 1 -or [int]$script:BridgePort -gt 65535) {
        throw "Porta non valida: $($script:BridgePort)"
    }

    if ([string]::IsNullOrWhiteSpace($script:Token)) {
        Write-Host ''
        Write-Host 'c) Install token (incolla dal server ponte, poi Invio):' -ForegroundColor White
        $script:Token = Read-Host
        $script:Token = $script:Token.Trim()
    }
    if ([string]::IsNullOrWhiteSpace($script:Token)) {
        throw 'Install token obbligatorio.'
    }

    $script:ServerURL = "$($script:BridgeHost):$($script:BridgePort)"
    Write-Host ''
    Write-Host "Server ponte: $($script:ServerURL)" -ForegroundColor DarkGray
}

function Test-BridgeTcp {
    param(
        [string]$TargetHost,
        [int]$Port
    )
    if ([string]::IsNullOrWhiteSpace($TargetHost)) {
        return $false
    }
    try {
        $r = Test-NetConnection -ComputerName $TargetHost -Port $Port -WarningAction SilentlyContinue -ErrorAction Stop
        return [bool]$r.TcpTestSucceeded
    } catch {
        return $false
    }
}

function Test-AgentServiceHealthy {
    $svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
    if (-not $svc) { return $false }
    return $svc.Status -eq 'Running'
}

Read-InstallParams

Write-Step "Verifica raggiungibilita server ponte ($($script:BridgeHost):$($script:BridgePort))..."
if (Test-BridgeTcp -TargetHost $script:BridgeHost -Port ([int]$script:BridgePort)) {
    Write-Ok 'Connessione TCP verso il ponte riuscita.'
} else {
    Write-Fail "Impossibile raggiungere $($script:BridgeHost):$($script:BridgePort) (firewall o indirizzo errato)."
    Write-Host '  Controlla IP/porta e che la 4443 sia aperta sul VPS.' -ForegroundColor Yellow
    Write-Host '  Su WinBoat/VM verifica anche che la rete abbia accesso internet in uscita.' -ForegroundColor Yellow
}

Write-Step 'OpenSSH Server (winget / capability)...'
$opensshScript = Join-Path $ScriptDir 'install-openssh.ps1'
if (-not (Test-Path $opensshScript)) {
    $opensshScript = Join-Path (Split-Path $ScriptDir -Parent) 'SOURCE_CODE\scripts\install-openssh.ps1'
}
& $opensshScript

$sshdOk = $false
$sshdSvc = Get-Service -Name sshd -ErrorAction SilentlyContinue
if ($sshdSvc -and $sshdSvc.Status -eq 'Running') {
    $sshdOk = $true
    Write-Ok 'Servizio sshd in esecuzione.'
} else {
    Write-Fail 'Servizio sshd non attivo.'
}

Write-Step 'Installazione nossh-agent...'
New-Item -ItemType Directory -Force -Path $InstallDir, $ConfigDir | Out-Null

$AgentSrc = Find-AgentExe
$AgentBin = Join-Path $InstallDir 'nossh-agent.exe'
Copy-Item -Force $AgentSrc $AgentBin
Write-Ok "Binario: $AgentBin"

$isNewAgent = -not (Test-Path $ConfigFile)
if ($isNewAgent) {
    Write-Step 'Registrazione agent (generazione codice)...'
    $code = & $AgentBin init-config --server $script:ServerURL --token $script:Token --config $ConfigFile
    Write-Ok "Codice agent: $code"
} else {
    Write-Host "Config esistente: $ConfigFile (codice conservato)" -ForegroundColor Yellow
}

Write-Step 'Servizio Windows nossh-agent...'
$binaryPath = "`"$AgentBin`" -config `"$ConfigFile`""
$existing = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
if ($existing) {
    Stop-Service $ServiceName -Force -ErrorAction SilentlyContinue
    sc.exe delete $ServiceName | Out-Null
    Start-Sleep -Seconds 2
}

$serviceCreated = $false
try {
    New-Service -Name $ServiceName `
        -BinaryPathName $binaryPath `
        -DisplayName 'nossh agent' `
        -Description 'Quelo Connect nossh agent' `
        -StartupType Automatic -ErrorAction Stop | Out-Null
    $serviceCreated = $true
    Write-Ok 'Servizio creato (New-Service).'
} catch {
    Write-Host "  New-Service: $_" -ForegroundColor Yellow
    Write-Host '  Provo sc.exe...' -ForegroundColor DarkGray
    $scOut = & sc.exe create $ServiceName binPath= $binaryPath start= auto DisplayName= 'nossh agent' 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Host "  sc.exe: $scOut" -ForegroundColor Red
        throw "Impossibile creare il servizio nossh-agent (codice $LASTEXITCODE)."
    }
    $serviceCreated = $true
    Write-Ok 'Servizio creato (sc.exe).'
}

if (-not (Get-Service -Name $ServiceName -ErrorAction SilentlyContinue)) {
    throw 'Servizio nossh-agent non trovato dopo la creazione.'
}

Start-Service $ServiceName -ErrorAction Stop
Start-Sleep -Seconds 3
$svcCheck = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
if (-not $svcCheck -or $svcCheck.Status -ne 'Running') {
    $logPath = Join-Path $env:ProgramData 'nossh\agent.log'
    Write-Fail 'Servizio nossh-agent non resta in esecuzione.'
    if (Test-Path $logPath) {
        Write-Host '  Ultime righe di agent.log:' -ForegroundColor Yellow
        Get-Content $logPath -Tail 15 | ForEach-Object { Write-Host "    $_" }
    }
    throw 'Avvio servizio nossh-agent fallito.'
}
Write-Ok 'Servizio nossh-agent avviato.'

Write-Host '  Attendo connessione al ponte...' -ForegroundColor DarkGray
Start-Sleep -Seconds 5

$agentOk = Test-AgentServiceHealthy
$ponteOk = Test-BridgeTcp -TargetHost $script:BridgeHost -Port ([int]$script:BridgePort)
$codeLine = (Get-Content $ConfigFile | Where-Object { $_ -match '^code:' }) -replace '.*:\s*', '' -replace '"', '' -replace '\s', ''
$hostName = $env:COMPUTERNAME

Write-Host ''
Write-Host '========================================' -ForegroundColor Yellow

if ($agentOk -and $sshdOk -and $ponteOk) {
    Write-Host '  INSTALLAZIONE COMPLETATA CON SUCCESSO' -ForegroundColor Green
    Write-Host '========================================' -ForegroundColor Yellow
    Write-Host ''
    Write-Ok "Agent connesso al server ponte ($($script:ServerURL))."
    Write-Ok 'Servizio nossh-agent attivo.'
    Write-Ok 'OpenSSH locale pronto (127.0.0.1:22).'
} elseif ($agentOk -and $sshdOk) {
    Write-Host '  INSTALLAZIONE OK (verifica rete ponte)' -ForegroundColor Yellow
    Write-Host '========================================' -ForegroundColor Yellow
    Write-Host ''
    Write-Ok 'Servizio nossh-agent attivo.'
    Write-Ok 'OpenSSH locale pronto.'
    Write-Fail 'Test TCP verso il ponte non riuscito - controlla firewall/IP.'
} elseif (-not $agentOk) {
    Write-Host '  INSTALLAZIONE PARZIALE - CONTROLLA TOKEN' -ForegroundColor Red
    Write-Host '========================================' -ForegroundColor Yellow
    Write-Host ''
    Write-Fail 'Il servizio nossh-agent non resta attivo.'
    Write-Host '  Verifica install_token in server.yaml sul ponte.' -ForegroundColor Yellow
} else {
    Write-Host '  INSTALLAZIONE CON ERRORI' -ForegroundColor Red
    Write-Host '========================================' -ForegroundColor Yellow
}

Write-Host ''
Write-Host "  Codice macchina:  $codeLine" -ForegroundColor White
Write-Host "  Hostname:         $hostName" -ForegroundColor White
Write-Host "  Server ponte:     $($script:ServerURL)" -ForegroundColor White
Write-Host ''
Write-Host '  Comunica il CODICE all amministratore del ponte' -ForegroundColor White
Write-Host '  per assegnare il nome macchina (nossh name CODICE nome).' -ForegroundColor White
Write-Host ''

if (-not $agentOk) {
    exit 1
}
