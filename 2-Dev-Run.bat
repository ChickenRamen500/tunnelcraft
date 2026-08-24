@echo off
chcp 65001 >nul
setlocal EnableDelayedExpansion

title TunnelCraft - Dev (Build + Run, No Pull)
color 0A

echo.
echo ============================================================
echo   TunnelCraft - Dev Mode (Build + Run, No Git Pull)
echo ============================================================
echo.

:: ===== Set working directory =====
cd /d "%~dp0"
echo [INFO] Working directory: %CD%
echo.

:: ===== Check Go =====
where go >nul 2>&1
if %ERRORLEVEL% neq 0 (
    color 0C
    echo [ERROR] Go not found! Install: https://go.dev/dl/
    goto :fail
)

:: ===== Check Node.js =====
where node >nul 2>&1
if %ERRORLEVEL% neq 0 (
    color 0C
    echo [ERROR] Node.js not found! Install: https://nodejs.org/
    goto :fail
)

:: ===== Check Rust/Cargo =====
where cargo >nul 2>&1
if %ERRORLEVEL% neq 0 (
    color 0C
    echo [ERROR] Rust not found! Install: https://rustup.rs/
    goto :fail
)

for /f "tokens=3" %%i in ('go version 2^>^&1') do echo [OK] Go: %%i
for /f "tokens=*" %%i in ('node --version 2^>^&1') do echo [OK] Node: %%i
for /f "tokens=*" %%i in ('cargo --version 2^>^&1') do echo [OK] %%i
echo.

:: ==========================================================
:: STEP 1: Check external binaries
:: ==========================================================
if not exist "bin\xray-core.exe" (
    echo [1/4] Downloading external binaries (first run)...
    if exist "scripts\download-binaries.ps1" (
        powershell -ExecutionPolicy Bypass -File scripts\download-binaries.ps1
        if %ERRORLEVEL% neq 0 (
            echo [WARN] Some binaries failed to download. Core features may be limited.
        )
    ) else (
        echo [WARN] scripts\download-binaries.ps1 not found. Skipping.
    )
) else (
    echo [1/4] External binaries already present. Skipped.
)
echo.

:: ==========================================================
:: STEP 2: Build Go daemon (tunnelcraftd.exe)
:: ==========================================================
echo [2/4] Building Go daemon (tunnelcraftd.exe)...
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

:: ==========================================================
:: STEP 3: Install npm dependencies (if needed)
:: ==========================================================
if not exist "ui\node_modules" (
    echo [3/4] Installing npm dependencies (may take 2-5 min on first run)...
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
    echo [3/4] npm dependencies already installed. Skipped.
)
echo.

:: ==========================================================
:: STEP 4: Launch Tauri dev mode
:: ==========================================================
echo [4/4] Launching TunnelCraft (Tauri dev mode)...
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