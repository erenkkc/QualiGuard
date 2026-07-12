@echo off
title QualiGuard — GitHub Hazirlik
cd /d "%~dp0"

echo ========================================
echo   QualiGuard - GitHub'a yukleme rehberi
echo ========================================
echo.

if not exist ".git" (
  echo Git deposu olusturuluyor...
  git init
  git branch -M main
  echo.
)

echo ADIM 1 — GitHub'da repo ac
echo   https://github.com/new
echo   Repo adi: QualiGuard  (Public veya Private)
echo   README ekleme — bos birak
echo.

echo ADIM 2 — Asagidaki komutlari sirayla calistir
echo   (KULLANICI yerine kendi GitHub kullanici adin)
echo.
echo   git add .
echo   git commit -m "QualiGuard ilk surum"
echo   git remote add origin https://github.com/KULLANICI/QualiGuard.git
echo   git push -u origin main
echo.

echo ADIM 3 — Otomatik CI
echo   Push sonrasi GitHub Actions calisir:
echo   .github/workflows/qualiguard.yml
echo   - Her PR'da kod taramasi
echo   - Kalite kapisi gecmezse kirmizi X
echo.

echo VS Code GEREKMEZ — sadece tarayici + QualiGuard-Dashboard.bat yeterli.
echo.
echo Otomatik yukleme icin (GitHub CLI gerekir): GitHub-Yukle.bat
echo.
pause
