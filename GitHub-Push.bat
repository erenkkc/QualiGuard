@echo off
title QualiGuard — GitHub Push (Adim 1)
cd /d "%~dp0"

echo ========================================
echo   ADIM 1 — GitHub'a yukle
echo ========================================
echo.

if not exist ".git" (
  git init
  git branch -M main
)

rem Git kimligi (sadece bu repo — bir kez)
git config user.email >nul 2>&1
if errorlevel 1 (
  echo Ilk kez: Git adin ve e-postan sorulacak (GitHub ile ayni olabilir).
  set /p GITNAME=Adin Soyadin: 
  set /p GITEMAIL=E-posta: 
  git config user.name "%GITNAME%"
  git config user.email "%GITEMAIL%"
  echo.
)

for /f %%i in ('git rev-list --count HEAD 2^>nul') do set COUNT=%%i
if "%COUNT%"=="0" (
  echo Ilk commit olusturuluyor...
  git add -A
  git commit -m "QualiGuard v1 — panel, landing, CI ve deploy"
  if errorlevel 1 (
    echo Commit basarisiz.
    pause
    exit /b 1
  )
)

echo.
echo 1) GitHub'da YENI repo olustur (README ekleme):
start "" "https://github.com/new"

echo.
echo 2) Repo olusturduktan sonra GitHub kullanici adini yaz:
set /p GHUSER=GitHub kullanici adi: 
if "%GHUSER%"=="" (
  echo Iptal.
  pause
  exit /b 1
)

git remote remove origin 2>nul
git remote add origin https://github.com/%GHUSER%/QualiGuard.git

echo.
echo 3) Push — GitHub sifren veya token isteyebilir.
git push -u origin main

if errorlevel 1 (
  echo.
  echo Push basarisiz. Repo adi QualiGuard mi? GitHub'da giris yaptin mi?
  pause
  exit /b 1
)

echo.
echo === ADIM 1 TAMAM ===
echo GitHub Actions otomatik calisir (.github/workflows/qualiguard.yml)
echo.
echo Sonraki: Domain-Deploy.bat  (Adim 2)
pause
