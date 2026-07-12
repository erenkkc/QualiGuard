@echo off
setlocal
set "PATH=%PATH%;C:\Program Files\Go\bin"
cd /d "%~dp0"

echo QualiGuard Test Suite
echo =====================

go test ./... -count=1
if errorlevel 1 (
  echo.
  echo TEST FAILED
  pause
  exit /b 1
)

echo.
echo All tests passed.
echo.
echo Quick manual check:
echo  1. QualiGuard-Dashboard.bat
echo  2. Canli Analiz - Calistir
echo  3. Kirmizi satira tikla
pause
