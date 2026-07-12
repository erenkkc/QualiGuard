@echo off
title QualiGuard — Eklenti Kur (v0.2)
setlocal

set "QG=%~dp0"
set "VSIX=%QGextension\qualiguard\qualiguard-0.2.0.vsix"
set "VSCODE=%LOCALAPPDATA%\Programs\Microsoft VS Code\bin\code.cmd"
set "CURSOR=%LOCALAPPDATA%\Programs\cursor\resources\app\bin\cursor.cmd"
set "CLI="

if not exist "%VSIX%" (
  echo VSIX yok. Once scripts\build-extension.bat calistirin.
  pause
  exit /b 1
)

if exist "%VSCODE%" set "CLI=%VSCODE%"
if not defined CLI if exist "%CURSOR%" set "CLI=%CURSOR%"

if not defined CLI (
  echo VS Code veya Cursor bulunamadi.
  echo Elle: Ctrl+Shift+P -^> Install from VSIX
  echo %VSIX%
  pause
  exit /b 1
)

echo Eklenti kuruluyor...
call "%CLI%" --install-extension "%VSIX%"
if errorlevel 1 (
  echo Kurulum basarisiz.
  pause
  exit /b 1
)

echo.
echo TAMAM. Editor'u yenile (Reload Window).
echo Sunucu: server.bat
pause
