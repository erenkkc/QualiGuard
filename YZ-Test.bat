@echo off
title QualiGuard — YZ Test
cd /d "%~dp0"
call "%~dp0scripts\init-env.bat"

echo.
echo === 1) Ollama ===
ollama list 2>nul
if errorlevel 1 (
  echo Ollama YOK veya calismiyor. Ollama uygulamasini ac.
  pause
  exit /b 1
)

echo.
echo === 2) QualiGuard sunucu ===
curl -s http://127.0.0.1:9000/api/health
if errorlevel 1 (
  echo.
  echo Sunucu yok — once server.bat baslat.
  pause
  exit /b 1
)

echo.
echo.
echo === 3) Hizli sohbet testi ===
set /p TOKEN=<"%USERPROFILE%\.qualiguard\token.txt"
curl -s -X POST http://127.0.0.1:9000/api/v1/chat ^
  -H "Authorization: Bearer %TOKEN%" ^
  -H "Content-Type: application/json" ^
  -d "{\"messages\":[{\"role\":\"user\",\"content\":\"Merhaba, bir cumleyle kendini tanit.\"}]}"

echo.
echo.
echo Tamam. Panelde Sohbet'i dene: http://127.0.0.1:9000/app#/chat
pause
