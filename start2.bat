@echo off
title TunnelCraft Launcher
color 0A

echo ========================================
echo   TunnelCraft - Starting...
echo ========================================
echo.

cd /d "%~dp0"

echo [1/4] Updating from repository...
git fetch --all >nul 2>&1
git pull --rebase origin main >nul 2>&1
if errorlevel 1 (
    echo     [!] Git pull skipped
) else (
    echo     [OK] Updated
)
echo.

echo [2/4] Checking Go...
where go >nul 2>nul
if errorlevel 1 (
    echo     [X] Go not found!
    echo     Download from https://go.dev/dl/
    pause
    exit /b 1
)
for /f "tokens=*" %%i in ('go version') do echo     [OK] %%i
echo.

echo [3/4] Building tunnelcraftd...
cd core\cmd\tunnelcraftd
go build -o ..\..\..\bin\tunnelcraftd.exe . 2>&1
if errorlevel 1 (
    echo     [X] Build failed!
    pause
    exit /b 1
)
cd ..\..\..
echo     [OK] Daemon built
echo.

echo [4/4] Starting Tauri app...
echo     (This may take 30-60 seconds on first run)
echo.
cd ui

if not exist "node_modules\" (
    echo     Installing npm dependencies...
    call npm install --silent
    echo.
)

call npm run tauri dev

echo.
echo ========================================
echo   App closed
echo ========================================
timeout /t 3 >nul