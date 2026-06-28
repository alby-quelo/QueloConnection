@echo off
setlocal EnableExtensions
cd /d "%~dp0"

if not exist "nossh.exe" (
  echo Errore: nossh.exe non trovato in questa cartella.
  pause
  exit /b 1
)

if not exist "client.conf" (
  echo Errore: manca client.conf
  echo Configura il server con Quelo Connect ^(Opzioni - Configura server^)
  echo oppure copia client.conf.example in client.conf e compila host, port e token.
  pause
  exit /b 1
)

set "HOST="
set "PORT="
set "TOKEN="
set "MACHINE="
set "SERVER="
for /f "usebackq eol=# tokens=1,* delims==" %%a in ("client.conf") do (
  if /i "%%a"=="host" set "HOST=%%b"
  if /i "%%a"=="port" set "PORT=%%b"
  if /i "%%a"=="token" set "TOKEN=%%b"
  if /i "%%a"=="machine" set "MACHINE=%%b"
  if /i "%%a"=="server" set "SERVER=%%b"
)

if not "%HOST%"=="" (
  if "%PORT%"=="" set "PORT=7000"
  set "SERVER=%HOST%:%PORT%"
)

if "%SERVER%"=="" (
  echo Errore: imposta host= in client.conf oppure usa Quelo Connect per configurare il server.
  pause
  exit /b 1
)
if "%MACHINE%"=="" (
  echo Errore: imposta machine= in client.conf
  pause
  exit /b 1
)

where ssh >nul 2>&1
if errorlevel 1 (
  echo Errore: comando ssh non trovato.
  echo Installa "OpenSSH Client" da Impostazioni - App - Funzionalita opzionali
  pause
  exit /b 1
)

set /p NOSSH_USER=Username Linux sulla macchina remota: 
if "%NOSSH_USER%"=="" (
  echo Username obbligatorio.
  pause
  exit /b 1
)

title nossh %MACHINE%
echo Connessione a %MACHINE% via %SERVER% ...
echo.

"%~dp0nossh.exe" -server "%SERVER%" connect "%MACHINE%" "%NOSSH_USER%"

echo.
echo Sessione terminata.
pause
