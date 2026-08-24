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

:: ===== Set working directory =====
cd /d "%~dp0"
echo [INFO] Working directory: %CD%
echo.

:: ===== Check Git =====
where git >nul 2>&1
if %ERRORLEVEL% neq 0 (
    color 0C
    echo [ERROR] Git not found! Install: https://git-scm.com/download/win
    goto :fail
)

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
:: STEP 1: Git Pull
:: ==========================================================
echo [1/5] Pulling latest changes from GitHub...
set "REPO_URL=https://github.com/ChickenRamen500/tunnelcraft.git"

:: Try simple pull first (works if git has cached credentials or SSH)
git pull --ff-only 2>&1
if %ERRORLEVEL% neq 0 (
    :: If simple pull fails, try using .github-token file
    if exist ".github-token" (
        echo       Standard pull failed, retrying with .github-token...
        set /p PAT=<.github-token
        if not "!PAT!"=="" (
            set "AUTH_URL=https://ChickenRamen500:!PAT!@github.com/ChickenRamen500/tunnelcraft.git"
            git remote set-url origin "!AUTH_URL!" 2>nul
            git pull --ff-only 2>&1
            git remote set-url origin "%REPO_URL%" 2>nul
        )
    )
)
if %ERRORLEVEL% neq 0 (
    echo [WARN] git pull failed. Continuing with local code...
) else (
    echo [OK] Repository updated.
)
echo.

:: ==========================================================
:: STEP 2: Download external binaries (if missing)
:: ==========================================================
if not exist "bin\xray-core.exe" (
    echo [2/5] Downloading external binaries (first run)...
    if exist "scripts\download-binaries.ps1" (
        powershell -ExecutionPolicy Bypass -File scripts\download-binaries.ps1
        if %ERRORLEVEL% neq 0 (
            echo [WARN] Some binaries failed to download. Core features may be limited.
        )
    ) else (
        echo [WARN] scripts\download-binaries.ps1 not found. Skipping.
    )
) else (
    echo [2/5] External binaries already present. Skipped.
)
echo.

:: ==========================================================
:: STEP 3: Build Go daemon (tunnelcraftd.exe)
:: ==========================================================
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

:: ==========================================================
:: STEP 4: Install npm dependencies (if needed)
:: ==========================================================
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

:: ==========================================================
:: STEP 5: Launch Tauri dev mode
:: ==========================================================
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