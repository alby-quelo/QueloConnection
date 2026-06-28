@echo off
cd /d "%~dp0"
if not exist "quelo-connect.exe" (
  echo Errore: quelo-connect.exe non trovato in questa cartella.
  pause
  exit /b 1
)
start "" "%~dp0quelo-connect.exe"
