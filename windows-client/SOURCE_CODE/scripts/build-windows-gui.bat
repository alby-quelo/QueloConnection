@echo off
REM Compila quelo-connect.exe sulla macchina Windows (richiede Go installato).
setlocal EnableExtensions
cd /d "%~dp0.."

set "GUI_DIR=cmd\quelo-connect-gui-win"
set "OUT=windows\dist\quelo-connect.exe"

where go >nul 2>&1 || (echo Errore: Go non trovato.& pause& exit /b 1)

if not exist "%GUI_DIR%\quelo-connect.ico" (
  echo Genera quelo-connect.ico con ImageMagick oppure copialo in %GUI_DIR%
)

for /f "delims=" %%G in ('go env GOPATH') do set "GOPATH=%%G"
set "RSRC=%GOPATH%\bin\rsrc.exe"
if exist "%RSRC%" (
  if exist "%GUI_DIR%\quelo-connect.ico" (
    "%RSRC%" -ico "%GUI_DIR%\quelo-connect.ico" -manifest "%GUI_DIR%\quelo-connect.exe.manifest" -o "%GUI_DIR%\rsrc.syso" -arch amd64
  ) else (
    "%RSRC%" -manifest "%GUI_DIR%\quelo-connect.exe.manifest" -o "%GUI_DIR%\rsrc.syso" -arch amd64
  )
) else (
  echo Nota: installa rsrc con: go install github.com/akavel/rsrc@latest
)

go build -ldflags="-H windowsgui" -o "%OUT%" .\%GUI_DIR%
copy /y "%GUI_DIR%\quelo-connect.exe.manifest" "windows\dist\quelo-connect.exe.manifest" >nul

echo.
echo Creato %OUT%
pause
