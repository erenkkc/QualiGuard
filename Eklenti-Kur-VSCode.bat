@echo off
title QualiGuard — VS Code Eklenti Kur
setlocal

set "QG=%~dp0"
set "VSIX=%QGextension\qualiguard\qualiguard-0.2.0.vsix"
set "VSCODE=%LOCALAPPDATA%\Programs\Microsoft VS Code\bin\code.cmd"

if not exist "%VSIX%" (
  echo VSIX yok. Once scripts\build-extension.bat calistirin.
  pause
  exit /b 1
)

if not exist "%VSCODE%" (
  echo VS Code bulunamadi: %VSCODE%
  echo.
  echo ELLE KUR:
  echo 1. VS Code ac
  echo 2. Ctrl+Shift+P
  echo 3. Extensions: Install from VSIX
  echo 4. Dosya: %VSIX%
  pause
  exit /b 1
)

echo QualiGuard eklentisi VS Code'a kuruluyor...
call "%VSCODE%" --install-extension "%VSIX%"
if errorlevel 1 (
  echo Kurulum basarisiz.
  pause
  exit /b 1
)

echo.
echo TAMAM.
echo 1. VS Code: Ctrl+Shift+P -^> Developer: Reload Window
echo 2. server.bat calistir
echo 3. Ctrl+Shift+P -^> QualiGuard: Workspace'i tara
pause
