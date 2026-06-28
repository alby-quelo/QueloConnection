#Requires -RunAsAdministrator
<#
  Rimuove nossh-agent (e opzionalmente la configurazione OpenSSH creata dall installer).
  Uso: powershell -ExecutionPolicy Bypass -File uninstall-agent.ps1
       powershell -ExecutionPolicy Bypass -File uninstall-agent.ps1 -Full
#>
param(
    [switch]$Full
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Continue'

$ServiceName = 'nossh-agent'
$InstallDir = Join-Path ${env:ProgramFiles} 'nossh'
$ConfigDir = Join-Path $env:ProgramData 'nossh'
$SshDir = Join-Path $env:ProgramData 'ssh'

function Write-Step($msg) {
    Write-Host ''
    Write-Host "==> $msg" -ForegroundColor Cyan
}

Write-Host ''
Write-Host '=== Disinstallazione nossh agent ===' -ForegroundColor Yellow
if ($Full) {
    Write-Host 'Modalita completa (-Full): rimuove anche config OpenSSH dell installer.' -ForegroundColor DarkGray
}

Write-Step 'Arresto servizio nossh-agent...'
$svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
if ($svc) {
    Stop-Service $ServiceName -Force -ErrorAction SilentlyContinue
    Start-Sleep -Seconds 2
    sc.exe delete $ServiceName | Out-Null
    Write-Host '  Servizio nossh-agent rimosso.' -ForegroundColor Green
} else {
    Write-Host '  Servizio nossh-agent non presente.' -ForegroundColor DarkGray
}

Write-Step 'Rimozione file nossh-agent...'
if (Test-Path $InstallDir) {
    Remove-Item -LiteralPath $InstallDir -Recurse -Force
    Write-Host "  Rimosso: $InstallDir" -ForegroundColor Green
}
if (Test-Path $ConfigDir) {
    Remove-Item -LiteralPath $ConfigDir -Recurse -Force
    Write-Host "  Rimosso: $ConfigDir" -ForegroundColor Green
}

if ($Full) {
    Write-Step 'Reset configurazione OpenSSH (sshd)...'
    Stop-Service sshd -Force -ErrorAction SilentlyContinue
    $cfg = Join-Path $SshDir 'sshd_config'
    $default = Join-Path $SshDir 'sshd_config_default'
    if (Test-Path $cfg) {
        Remove-Item -LiteralPath $cfg -Force
        Write-Host "  Rimosso: $cfg" -ForegroundColor Green
    }
    Get-ChildItem -Path $SshDir -Filter 'sshd_config.bak-*' -ErrorAction SilentlyContinue | Remove-Item -Force
    if (Test-Path $default) {
        Copy-Item -LiteralPath $default -Destination $cfg
        Write-Host '  Ripristinato sshd_config da sshd_config_default.' -ForegroundColor Green
    }
    Write-Host '  Nota: OpenSSH Server resta installato; solo config ripulita.' -ForegroundColor DarkGray
}

Write-Host ''
Write-Host 'Disinstallazione completata.' -ForegroundColor Green
Write-Host 'Ora puoi rilanciare install-agent.bat come amministratore.' -ForegroundColor White
Write-Host ''
