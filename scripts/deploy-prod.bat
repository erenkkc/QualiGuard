@echo off
title QualiGuard — Production Deploy
cd /d "%~dp0.."

if not exist ".env" (
  echo .env bulunamadi — .env.example kopyalaniyor...
  copy .env.example .env >nul
  echo .env dosyasini duzenleyip QG_DOMAIN ve QG_EMAIL ayarlayin.
  pause
)

echo QualiGuard production stack baslatiliyor...
docker compose -f docker-compose.prod.yml up -d --build
if errorlevel 1 exit /b 1

echo.
echo Tamam. Panel: https://%QG_DOMAIN%/app
echo Landing: https://%QG_DOMAIN%/
pause
