@echo off
chcp 65001 >nul
setlocal EnableDelayedExpansion

title TunnelCraft - Dev (Pull + Build + Run)
color 0A

echo.
echo ============================================================
echo   TunnelCraft - Dev Mode (Pull from GitHub + Build + Run)
echo ============================================================
echo.

cd /d "%~dp0"
echo [INFO] Working directory: %CD%
echo.

where git >nul 2>&1
if %ERRORLEVEL% neq 0 (
    color 0C
    echo [ERROR] Git not found! Install: https://git-scm.com/download/win
    goto :fail
)

where go >nul 2>&1
if %ERRORLEVEL% neq 0 (
    color 0C
    echo [ERROR] Go not found! Install: https://go.dev/dl/
    goto :fail
)

where node >nul 2>&1
if %ERRORLEVEL% neq 0 (
    color 0C
    echo [ERROR] Node.js not found! Install: https://nodejs.org/
    goto :fail
)

where cargo >nul 2>&1
if %ERRORLEVEL% neq 0 (
    color 0C
    echo [ERROR] Rust not found! Install: https://rustup.rs/
    goto :fail
)

echo [OK] All tools found.
echo.

echo [1/5] Pulling latest changes from GitHub...
git pull --ff-only
if %ERRORLEVEL% neq 0 (
    echo [WARN] git pull failed. Continuing with local code...
) else (
    echo [OK] Repository updated.
)
echo.

if not exist "bin\xray-core.exe" (
    echo [2/5] Downloading external binaries (first run)...
    if exist "scripts\download-binaries.ps1" (
        powershell -ExecutionPolicy Bypass -File scripts\download-binaries.ps1
        if %ERRORLEVEL% neq 0 (
            echo [WARN] Some binaries failed to download.
        )
    ) else (
        echo [WARN] scripts\download-binaries.ps1 not found. Skipping.
    )
) else (
    echo [2/5] External binaries already present. Skipped.
)
echo.

echo [3/5] Building Go daemon (tunnelcraftd.exe)...
pushd core\cmd\tunnelcraftd
go build -o ..\..\..\bin\tunnelcraftd.exe .
if %ERRORLEVEL% neq 0 (
    color 0C
    echo [ERROR] Go daemon build failed!
    popd
    goto :fail
)
popd
echo [OK] tunnelcraftd.exe built successfully.
echo.

if not exist "ui\node_modules" (
    echo [4/5] Installing npm dependencies (may take 2-5 min on first run)...
    pushd ui
    call npm install
    if %ERRORLEVEL% neq 0 (
        color 0C
        echo [ERROR] npm install failed!
        popd
        goto :fail
    )
    popd
    echo [OK] npm dependencies installed.
) else (
    echo [4/5] npm dependencies already installed. Skipped.
)
echo.

echo [5/5] Launching TunnelCraft (Tauri dev mode)...
echo.
echo ============================================================
echo   TunnelCraft is starting... Do NOT close this window!
echo ============================================================
echo.
pushd ui
call npm run tauri dev
popd
goto :end

:fail
echo.
echo ============================================================
echo   Launch aborted due to error.
echo ============================================================
pause
exit /b 1

:end
echo.
echo ============================================================
echo   TunnelCraft stopped.
echo ============================================================
pause
