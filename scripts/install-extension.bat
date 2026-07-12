@echo off
title QualiGuard — Eklenti Kur
setlocal

cd /d "%~dp0..\extension\qualiguard"

set "VSIX=qualiguard-0.2.0.vsix"
if not exist "%VSIX%" (
  echo  VSIX bulunamadi. Once scripts\build-extension.bat calistirin.
  pause
  exit /b 1
)

set "CURSOR=%LOCALAPPDATA%\Programs\cursor\resources\app\bin\cursor.cmd"
set "CLI="

if exist "%CURSOR%" set "CLI=%CURSOR%"
if not defined CLI where cursor >nul 2>&1 && set "CLI=cursor"
if not defined CLI where code >nul 2>&1 && set "CLI=code"

if not defined CLI (
  echo  cursor/code PATH'te yok. ELLE KUR:
  echo  Ctrl+Shift+P -^> Install from VSIX
  echo  %CD%\%VSIX%
  pause
  exit /b 1
)

echo  %CLI% --install-extension %VSIX%
call "%CLI%" --install-extension "%CD%\%VSIX%"
if errorlevel 1 (
  echo  Kurulum basarisiz.
  pause
  exit /b 1
)

echo.
echo  Kuruldu v0.2. Cursor'u yeniden baslat (Reload Window).
echo  Sunucu: server.bat
pause
