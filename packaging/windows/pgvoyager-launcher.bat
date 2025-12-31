@echo off
REM PgVoyager Windows Launcher - starts server and opens browser

setlocal enabledelayedexpansion

set "SCRIPT_DIR=%~dp0"
set "PGVOYAGER_BIN=%SCRIPT_DIR%pgvoyager.exe"
set "CONFIG_FILE=%LOCALAPPDATA%\PgVoyager\config.txt"

REM Load port from config file if it exists
set "PGVOYAGER_PORT=5137"
if exist "%CONFIG_FILE%" (
    for /f "tokens=2 delims==" %%a in ('findstr /i "PGVOYAGER_PORT" "%CONFIG_FILE%"') do set "PGVOYAGER_PORT=%%a"
)

REM Environment variable overrides config file
if defined PGVOYAGER_PORT_ENV set "PGVOYAGER_PORT=%PGVOYAGER_PORT_ENV%"
set "PGVOYAGER_URL=http://localhost:%PGVOYAGER_PORT%"

REM Check if already running
tasklist /FI "IMAGENAME eq pgvoyager.exe" 2>NUL | find /I /N "pgvoyager.exe">NUL
if "%ERRORLEVEL%"=="0" (
    echo PgVoyager is already running, opening browser...
    start "" "%PGVOYAGER_URL%"
    exit /b 0
)

REM Start pgvoyager in background
echo Starting PgVoyager...
set "PGVOYAGER_MODE=production"
start "" /B "%PGVOYAGER_BIN%"

REM Wait for server to be ready
echo Waiting for server...
set /a attempts=0
:waitloop
if %attempts% geq 30 goto timeout
timeout /t 1 /nobreak >NUL
curl -s "%PGVOYAGER_URL%" >NUL 2>&1
if "%ERRORLEVEL%"=="0" goto ready
set /a attempts+=1
goto waitloop

:timeout
echo Warning: Server may not have started properly
goto openbrowser

:ready
echo Server is ready!

:openbrowser
start "" "%PGVOYAGER_URL%"
echo PgVoyager is running at %PGVOYAGER_URL%
echo Press Ctrl+C to stop the server.
pause >NUL
