@echo off
cd /d "%~dp0"
echo Avvio Quelo Connect (debug)...
if not exist "quelo-connect.exe.manifest" (
  echo ATTENZIONE: manca quelo-connect.exe.manifest accanto all'exe
)
quelo-connect.exe
echo Exit code: %ERRORLEVEL%
pause
