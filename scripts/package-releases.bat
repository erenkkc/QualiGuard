@echo off
REM Builds QualiGuard release downloads into .\releases\
setlocal
cd /d "%~dp0.."

if not exist releases mkdir releases
if not exist pack\staging-win mkdir pack\staging-win
if not exist pack\staging-mac mkdir pack\staging-mac

echo [1/4] Windows CLI...
set GOOS=windows
set GOARCH=amd64
go build -o releases\qg-windows-amd64.exe .\cmd\qg || exit /b 1

echo [2/4] Windows panel server...
go build -o pack\staging-win\qg-server.exe .\cmd\qg-server || exit /b 1
copy /Y qualiguard.yaml pack\staging-win\qualiguard.yaml >nul
copy /Y pack\windows\BASLA.bat pack\staging-win\BASLA.bat >nul
powershell -NoProfile -Command "Compress-Archive -Path 'pack\staging-win\*' -DestinationPath 'releases\qualiguard-panel-windows.zip' -Force"

echo [3/4] Mac CLI...
set GOOS=darwin
set GOARCH=amd64
go build -o releases\qg-darwin-amd64 .\cmd\qg || exit /b 1

echo [4/4] Mac panel server...
go build -o pack\staging-mac\qg-server .\cmd\qg-server || exit /b 1
copy /Y qualiguard.yaml pack\staging-mac\qualiguard.yaml >nul
copy /Y pack\macos\start.sh pack\staging-mac\start.sh >nul
powershell -NoProfile -Command "Compress-Archive -Path 'pack\staging-mac\*' -DestinationPath 'releases\qualiguard-panel-macos.zip' -Force"

echo.
echo Done. Files in releases\
dir releases
endlocal
