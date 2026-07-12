@echo off
rem QualiGuard — ortak ortam (Go + Python)
set "PATH=%PATH%;C:\Program Files\Go\bin"

if exist "%LOCALAPDATA%\Programs\Python" (
  for /d %%D in ("%LOCALAPDATA%\Programs\Python\Python*") do (
    set "PATH=%%D;%%D\Scripts;%PATH%"
  )
)

if not defined QUALIGUARD_PYTHON (
  if exist "%LOCALAPDATA%\Programs\Python\Python313\python.exe" (
    set "QUALIGUARD_PYTHON=%LOCALAPDATA%\Programs\Python\Python313\python.exe"
  ) else (
    for /d %%D in ("%LOCALAPDATA%\Programs\Python\Python*") do (
      if exist "%%D\python.exe" set "QUALIGUARD_PYTHON=%%D\python.exe"
    )
  )
)

set "QG_DIR=C:\Users\Eren\Desktop\QualiGuard"
set "QG_DATA=%USERPROFILE%\.qualiguard"

rem JavaScript ESLint icin Node.js (varsa PATH'e ekle)
if exist "%ProgramFiles%\nodejs\npx.cmd" (
  set "PATH=%ProgramFiles%\nodejs;%PATH%"
)

rem Python Ruff linter (opsiyonel: pip install ruff)

rem Opsiyonel yapay zeka — birini açın:
rem set "QUALIGUARD_AI_PROVIDER=openai"
rem set "QUALIGUARD_OPENAI_API_KEY=sk-..."
rem set "QUALIGUARD_OPENAI_MODEL=gpt-4o-mini"
rem set "QUALIGUARD_AI_PROVIDER=gemini"
rem set "QUALIGUARD_GEMINI_API_KEY=..."
rem set "QUALIGUARD_GEMINI_MODEL=gemini-2.0-flash"
rem Yerel Ollama — once: ollama pull llama3.2
set "QUALIGUARD_AI_PROVIDER=ollama"
set "QUALIGUARD_OLLAMA_ENABLED=1"
set "QUALIGUARD_OLLAMA_MODEL=llama3.2:latest"
