@echo off
title QualiGuard — Domain Deploy (Adim 2)
cd /d "%~dp0"

echo ========================================
echo   ADIM 2 — Domain + HTTPS deploy
echo ========================================
echo.
echo Once GitHub'a push et (GitHub-Hazirla.bat).
echo.

if not exist ".env" (
  echo .env olusturuluyor...
  copy .env.example .env >nul
)

echo .env dosyasini duzenlemen gereken alanlar:
echo.
echo   QG_DOMAIN=senin-alan-adin.com
echo   QG_EMAIL=eposta@alan-adin.com
echo   QG_PANEL_PASSWORD=guclu-sifre
echo.
notepad .env

echo.
echo Docker Desktop acik olmali.
echo.
set /p OK=Deploy baslatilsin mi? (E/H): 
if /i not "%OK%"=="E" exit /b 0

docker compose -f docker-compose.prod.yml up -d --build
if errorlevel 1 (
  echo Deploy basarisiz. Docker calisiyor mu?
  pause
  exit /b 1
)

echo.
echo Tamam!
echo   Landing: https://SENIN-DOMAIN/
echo   Panel:   https://SENIN-DOMAIN/app
echo   Login:   https://SENIN-DOMAIN/login
echo.
pause
