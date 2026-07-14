@echo off
title QualiGuard Panel
cd /d "%~dp0"
echo QualiGuard baslatiliyor...
start "" http://127.0.0.1:9000/app
qg-server.exe --host 127.0.0.1 --port 9000 --data-dir "%USERPROFILE%\.qualiguard-local" --work-dir "%~dp0" --config qualiguard.yaml
pause
