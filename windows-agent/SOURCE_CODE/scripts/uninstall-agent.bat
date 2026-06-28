@echo off
title Disinstallazione nossh agent
echo.
echo  Rimuove nossh-agent per reinstallazione pulita.
echo  Per resettare anche la config OpenSSH: uninstall-agent.bat full
echo.
if /i "%~1"=="full" (
  powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0uninstall-agent.ps1" -Full
) else (
  powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0uninstall-agent.ps1"
)
echo.
pause
exit /b %ERRORLEVEL%
