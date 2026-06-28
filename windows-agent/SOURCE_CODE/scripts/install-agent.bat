@echo off
setlocal
title Installazione nossh agent
echo.
echo  Avvio installazione (richiede Amministratore)...
echo.
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0install-agent.ps1" %*
echo.
pause
exit /b %ERRORLEVEL%
