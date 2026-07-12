@echo off
title QualiGuard — VS Code Eklentisi Derle
cd /d "%~dp0..\extension\qualiguard"

if exist "%ProgramFiles%\nodejs\npm.cmd" (
  set "PATH=%ProgramFiles%\nodejs;%PATH%"
)

where npm >nul 2>&1
if errorlevel 1 (
  echo  HATA: Node.js bulunamadi. https://nodejs.org adresinden kurun.
  pause
  exit /b 1
)

echo  npm install...
call npm install
if errorlevel 1 exit /b 1

echo  npm run compile...
call npm run compile
if errorlevel 1 exit /b 1

echo  VSIX paketleniyor...
call npx --yes @vscode/vsce package --no-dependencies
if errorlevel 1 exit /b 1

echo.
echo  Tamam: qualiguard-0.2.0.vsix
echo  Kurulum: scripts\install-extension.bat
echo  Test: QualiGuard repo acikken F5 (Run Extension)
pause
