@echo off

setlocal

call "%~dp0scripts\init-env.bat"



cd /d "%QG_DIR%"

echo QualiGuard Server derleniyor...
go build -o bin\qg-server.exe .\cmd\qg-server || exit /b 1



if not exist "%QG_DATA%" mkdir "%QG_DATA%"



echo QualiGuard Server baslatiliyor...

echo Proje klasoru: %QG_DIR%

if defined QUALIGUARD_PYTHON echo Python: %QUALIGUARD_PYTHON%

echo.

bin\qg-server.exe --data-dir "%QG_DATA%" --work-dir "%QG_DIR%" --config qualiguard.yaml

