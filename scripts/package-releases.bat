@echo off
REM Builds QualiGuard release downloads into .\releases\
setlocal
cd /d "%~dp0.."

if not exist releases mkdir releases
if not exist pack\staging-win mkdir pack\staging-win
if not exist pack\staging-mac mkdir pack\staging-mac
if not exist cmd\qg-install\assets mkdir cmd\qg-install\assets

echo [1/6] Windows CLI...
set GOOS=windows
set GOARCH=amd64
go build -o releases\qg-windows-amd64.exe .\cmd\qg || exit /b 1

echo [2/6] Windows desktop app (QualiGuard.exe)...
go build -ldflags="-H windowsgui" -o pack\staging-win\QualiGuard.exe .\cmd\qg-desktop || exit /b 1
copy /Y qualiguard.yaml pack\staging-win\qualiguard.yaml >nul

echo [3/6] Windows one-click installer...
copy /Y pack\staging-win\QualiGuard.exe cmd\qg-install\assets\QualiGuard.exe >nul
copy /Y qualiguard.yaml cmd\qg-install\assets\qualiguard.yaml >nul
go build -o releases\QualiGuard-Kurulum.exe .\cmd\qg-install || exit /b 1

echo [4/6] Windows panel zip (yedek)...
go build -o pack\staging-win\qg-server.exe .\cmd\qg-server || exit /b 1
copy /Y pack\windows\BASLA.bat pack\staging-win\BASLA.bat >nul
powershell -NoProfile -Command "Compress-Archive -Path 'pack\staging-win\qg-server.exe','pack\staging-win\qualiguard.yaml','pack\staging-win\BASLA.bat' -DestinationPath 'releases\qualiguard-panel-windows.zip' -Force"

echo [5/6] Mac CLI...
set GOOS=darwin
set GOARCH=amd64
go build -o releases\qg-darwin-amd64 .\cmd\qg || exit /b 1

echo [6/6] Mac one-click kurulum...
go build -o pack\staging-mac\qg-server .\cmd\qg-server || exit /b 1
copy /Y qualiguard.yaml pack\staging-mac\qualiguard.yaml >nul
copy /Y pack\macos\QualiGuard-Kur.command pack\staging-mac\QualiGuard-Kur.command >nul
powershell -NoProfile -Command "Compress-Archive -Path 'pack\staging-mac\qg-server','pack\staging-mac\qualiguard.yaml','pack\staging-mac\QualiGuard-Kur.command' -DestinationPath 'releases\qualiguard-mac-kurulum.zip' -Force"

echo.
echo Done. Files in releases\
dir releases
endlocal
