@echo off
title QualiGuard
cd /d "%~dp0"

echo QualiGuard baslatiliyor...
echo.

rem Sunucuyu ayri pencerede ac (kapatinca sunucu durur)
start "QualiGuard Sunucu" cmd /k "%~dp0server.bat"

rem Sunucunun ayaga kalkmasini bekle
timeout /t 5 /nobreak >nul

rem Tarayicida paneli ac
start "" "http://127.0.0.1:9000/app"

echo.
echo  Panel tarayicida acildi.
echo  Sunucu penceresini KAPATMA — kapatinca site durur.
echo.
echo  Landing:  http://127.0.0.1:9000/
echo  Panel:    http://127.0.0.1:9000/app
echo.
pause
