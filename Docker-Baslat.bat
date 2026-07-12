@echo off
title QualiGuard — Docker
cd /d "%~dp0"

where docker >nul 2>&1
if errorlevel 1 (
  echo Docker bulunamadi. https://www.docker.com/products/docker-desktop/
  pause
  exit /b 1
)

echo Docker ile QualiGuard baslatiliyor (ilk seferde birkaç dakika surer)...
docker compose up -d --build
if errorlevel 1 (
  echo Docker baslatilamadi.
  pause
  exit /b 1
)

echo Sunucu hazirlaniyor...
timeout /t 8 /nobreak >nul

start "" "http://127.0.0.1:9000/app"

echo.
echo  Panel:   http://127.0.0.1:9000/app
echo  Landing: http://127.0.0.1:9000/
echo.
echo  Durdurmak icin: docker compose down
pause
