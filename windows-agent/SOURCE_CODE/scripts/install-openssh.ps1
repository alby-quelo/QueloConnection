#Requires -RunAsAdministrator
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$SshDir = Join-Path $env:ProgramData 'ssh'
$CfgPath = Join-Path $SshDir 'sshd_config'
$CfgDefault = Join-Path $SshDir 'sshd_config_default'
$InstallScript = Join-Path $env:WINDIR 'System32\OpenSSH\Install-sshd.ps1'

function Write-Step($msg) {
    Write-Host ''
    Write-Host "==> $msg" -ForegroundColor Cyan
}

function Test-Command($name) {
    return $null -ne (Get-Command $name -ErrorAction SilentlyContinue)
}

function Install-OpenSSHServer {
    $svc = Get-Service -Name sshd -ErrorAction SilentlyContinue
    $capOk = Get-WindowsCapability -Online -ErrorAction SilentlyContinue |
        Where-Object { $_.Name -like 'OpenSSH.Server*' -and $_.State -eq 'Installed' }

    if ($svc -or $capOk) {
        Write-Host 'OpenSSH Server gia presente sul sistema.'
        return
    }

    if (Test-Command winget) {
        Write-Step 'Installazione OpenSSH Server con winget...'
        try {
            $p = Start-Process winget -ArgumentList @(
                'install', '--id', 'Microsoft.OpenSSH.Beta',
                '--accept-package-agreements', '--accept-source-agreements', '-e', '--disable-interactivity'
            ) -Wait -PassThru -NoNewWindow
            if ($p.ExitCode -eq 0) { return }
            Write-Warning "winget exit code: $($p.ExitCode)"
        } catch {
            Write-Warning "winget non riuscito: $_"
        }
    }

    Write-Step 'Installazione OpenSSH Server (Windows Capability)...'
    $cap = Get-WindowsCapability -Online | Where-Object { $_.Name -like 'OpenSSH.Server*' -and $_.State -ne 'Installed' } | Select-Object -First 1
    if (-not $cap) {
        $capOk2 = Get-WindowsCapability -Online | Where-Object { $_.Name -like 'OpenSSH.Server*' -and $_.State -eq 'Installed' }
        if ($capOk2) {
            Write-Host 'Capability OpenSSH.Server gia installata.'
            return
        }
        throw 'OpenSSH Server non disponibile su questo sistema.'
    }
    Add-WindowsCapability -Online -Name $cap.Name | Out-Null
}

function Initialize-OpenSSHServer {
    Write-Step 'Inizializzazione OpenSSH (servizio + file config)...'

    if (Test-Path $InstallScript) {
        Write-Host "Eseguo: $InstallScript"
        & powershell.exe -NoProfile -ExecutionPolicy Bypass -File $InstallScript
    } else {
        Write-Host 'Install-sshd.ps1 non trovato, procedo con setup manuale.'
    }

    if (-not (Test-Path $SshDir)) {
        New-Item -ItemType Directory -Path $SshDir -Force | Out-Null
    }

    if (-not (Test-Path $CfgPath)) {
        if (Test-Path $CfgDefault) {
            Write-Host 'Creo sshd_config da sshd_config_default...'
            Copy-Item $CfgDefault $CfgPath
        } else {
            Write-Host 'Creo sshd_config minimo...'
            @(
                'Port 22',
                'ListenAddress 127.0.0.1',
                'PasswordAuthentication yes',
                'PubkeyAuthentication yes',
                'PermitEmptyPasswords no',
                'Subsystem sftp sftp-server.exe'
            ) | Set-Content -Path $CfgPath -Encoding ascii
        }
    }

    if (-not (Test-Path $CfgPath)) {
        throw "Impossibile creare $CfgPath"
    }
    Write-Host "Config trovata: $CfgPath"
}

function Set-SshdLocalhostOnly {
    if (-not (Test-Path $CfgPath)) {
        throw "File non trovato: $CfgPath"
    }

    Write-Step 'Configurazione sshd (solo localhost + password utenti Windows)...'
    $backup = "$CfgPath.bak-$(Get-Date -Format 'yyyyMMdd-HHmmss')"
    Copy-Item $CfgPath $backup

    $lines = Get-Content $CfgPath -Encoding utf8
    $out = New-Object System.Collections.Generic.List[string]

    foreach ($line in $lines) {
        if ($line -match '^\s*#?\s*ListenAddress\s') { continue }
        if ($line -match '^\s*#?\s*PasswordAuthentication\s') { continue }
        if ($line -match '^\s*#?\s*PubkeyAuthentication\s') { continue }
        if ($line -match '^\s*#?\s*PermitEmptyPasswords\s') { continue }
        [void]$out.Add($line)
    }

    $insertAt = 0
    for ($i = 0; $i -lt $out.Count; $i++) {
        if ($out[$i] -match '^\s*Port\s') { $insertAt = $i + 1; break }
    }

    $block = @(
        'ListenAddress 127.0.0.1',
        'PasswordAuthentication yes',
        'PubkeyAuthentication yes',
        'PermitEmptyPasswords no'
    )
    for ($j = $block.Count - 1; $j -ge 0; $j--) {
        $out.Insert($insertAt, [string]$block[$j])
    }

    Set-Content -Path $CfgPath -Value $out -Encoding ascii
    Write-Host "Backup config: $backup"
    Write-Host 'sshd ascolta solo su 127.0.0.1 (solo uso locale con nossh-agent).'
}

function Start-SshdService {
    Write-Step 'Avvio servizio sshd...'
    $svc = Get-Service -Name sshd -ErrorAction SilentlyContinue
    if (-not $svc) {
        throw 'Servizio sshd non trovato dopo installazione OpenSSH.'
    }
    Set-Service -Name sshd -StartupType Automatic
    Start-Service sshd
    Write-Host "Stato sshd: $((Get-Service sshd).Status)"
}

Install-OpenSSHServer
Initialize-OpenSSHServer
Set-SshdLocalhostOnly
Start-SshdService
Write-Host ''
Write-Host 'OpenSSH Server pronto.' -ForegroundColor Green
