@echo off

title QualiGuard — Sunum Hazirligi

cd /d "%~dp0.."

call scripts\init-env.bat



echo.

echo  ========================================

echo   QualiGuard Staj Sunumu

echo  ========================================

echo.

echo  Adimlar:

echo    1. Genel Bakis — demo kartlari

echo    2. Canli Analiz — python-kritik / javascript-stil

echo    3. Dosya Yukle — zip veya odev dosyasi

echo    4. YZ Sohbet — kod sorusu

echo.

echo  Rehber: docs\09-staj-sunumu-demo.md

echo  ========================================

echo.



if not exist "bin\qg-server.exe" (

  echo  Sunucu derleniyor...

  go build -o bin\qg-server.exe ./cmd/qg-server

  if errorlevel 1 (

    echo  HATA: Derleme basarisiz.

    pause

    exit /b 1

  )

)



echo  Sunucu baslatiliyor...

start "QualiGuard Server" /min bin\qg-server.exe



echo  Sunucu bekleniyor...

set /a tries=0

:wait_loop

set /a tries+=1

powershell -NoProfile -Command "try { (Invoke-WebRequest -UseBasicParsing http://127.0.0.1:9000/api/health -TimeoutSec 2).StatusCode -eq 200 } catch { exit 1 }" >nul 2>&1

if %errorlevel%==0 goto server_ok

if %tries% geq 15 (

  echo  UYARI: Sunucu yanit vermedi. server.bat ile manuel baslatin.

  goto open_browser

)

timeout /t 1 /nobreak >nul

goto wait_loop



:server_ok

echo  Sunucu hazir.



:open_browser

start "" "http://127.0.0.1:9000/#/"

echo  Tarayici acildi — Ctrl+F5 ile onbellegi temizleyin.

echo.

pause

