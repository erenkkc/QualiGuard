@echo off
title QualiGuard — GitHub'a Yukle
cd /d "%~dp0"

if not exist ".git" (
  echo Git deposu olusturuluyor...
  git init
  git branch -M main
)

where gh >nul 2>&1
if errorlevel 1 (
  echo GitHub CLI (gh) yok.
  echo GitHub-Hazirla.bat dosyasindaki manuel adimlari izleyin.
  start "" "%~dp0GitHub-Hazirla.bat"
  exit /b 1
)

echo GitHub reposu olusturuluyor / baglaniyor...
gh repo create QualiGuard --private --source=. --remote=origin --push
if errorlevel 1 (
  echo.
  echo Otomatik olusturulamadi. Manuel:
  echo   git add .
  echo   git commit -m "QualiGuard ilk surum"
  echo   gh repo create QualiGuard --private --source=. --push
  pause
  exit /b 1
)

echo.
echo Tamam. GitHub Actions otomatik calisacak.
echo Repo: gh repo view --web
pause
