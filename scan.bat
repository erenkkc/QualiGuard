@echo off
set "PATH=%PATH%;C:\Program Files\Go\bin"
cd /d "%~dp0"
if not exist "bin\qg.exe" (
  echo QualiGuard derleniyor...
  go build -o bin\qg.exe .\cmd\qg || exit /b 1
)
bin\qg.exe scan --config qualiguard.yaml --verbose %*
