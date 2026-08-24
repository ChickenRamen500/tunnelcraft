@echo off
chcp 65001 >nul
setlocal EnableDelayedExpansion

title TunnelCraft - Build Portable Release
color 0E

echo.
echo ============================================================
echo   TunnelCraft - Build Portable Version for Another PC
echo ============================================================
echo.

:: ===== Set working directory =====
cd /d "%~dp0"
echo [INFO] Working directory: %CD%
echo.

:: ===== Check all required tools =====
set "MISSING="

where git >nul 2>&1
if %ERRORLEVEL% neq 0 set "MISSING=!MISSING! Git"

where go >nul 2>&1
if %ERRORLEVEL% neq 0 set "MISSING=!MISSING! Go"

where node >nul 2>&1
if %ERRORLEVEL% neq 0 set "MISSING=!MISSING! Node.js"

where cargo >nul 2>&1
if %ERRORLEVEL% neq 0 set "MISSING=!MISSING! Rust"

if not "!MISSING!"=="" (
    color 0C
    echo [ERROR] Missing tools:!MISSING!
    echo.
    echo Install:
    echo   Git    - https://git-scm.com/download/win
    echo   Go     - https://go.dev/dl/
    echo   Node   - https://nodejs.org/
    echo   Rust   - https://rustup.rs/
    goto :fail
)

for /f "tokens=3" %%i in ('go version 2^>^&1') do echo [OK] Go: %%i
for /f "tokens=*" %%i in ('node --version 2^>^&1') do echo [OK] Node: %%i
for /f "tokens=*" %%i in ('cargo --version 2^>^&1') do echo [OK] %%i
echo.

:: ==========================================================
:: STEP 1: Git Pull
:: ==========================================================
echo [1/6] Pulling latest changes from GitHub...
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
:: STEP 2: Download external binaries
:: ==========================================================
echo [2/6] Downloading / updating external binaries...
if exist "scripts\download-binaries.ps1" (
    powershell -ExecutionPolicy Bypass -File scripts\download-binaries.ps1
    if %ERRORLEVEL% neq 0 (
        echo [WARN] Some binaries failed to download.
    )
) else (
    echo [WARN] scripts\download-binaries.ps1 not found.
)
echo.

:: ==========================================================
:: STEP 3: Build Go daemon (optimized)
:: ==========================================================
echo [3/6] Building Go daemon (optimized release)...
pushd core\cmd\tunnelcraftd
go build -ldflags "-s -w" -o ..\..\..\bin\tunnelcraftd.exe .
if %ERRORLEVEL% neq 0 (
    color 0C
    echo [ERROR] Go daemon build failed!
    popd
    goto :fail
)
popd
echo [OK] tunnelcraftd.exe built (stripped).
echo.

:: ==========================================================
:: STEP 4: Install npm + Build Tauri release
:: ==========================================================
echo [4/6] Building Tauri release (this will take a while)...
pushd ui

if not exist "node_modules" (
    echo       Installing npm dependencies...
    call npm install
    if %ERRORLEVEL% neq 0 (
        color 0C
        echo [ERROR] npm install failed!
        popd
        goto :fail
    )
)

echo       Running tauri build (compiling Rust release + frontend)...
call npm run tauri build
if %ERRORLEVEL% neq 0 (
    color 0C
    echo [ERROR] Tauri build failed!
    popd
    goto :fail
)
popd
echo [OK] Tauri release built.
echo.

:: ==========================================================
:: STEP 5: Create portable distribution folder
:: ==========================================================
echo [5/6] Creating portable distribution...

set "PORTABLE_DIR=%CD%\TunnelCraft-Portable"
set "TAURI_TARGET=ui\src-tauri\target\release"

:: Clean previous build
if exist "%PORTABLE_DIR%" rmdir /s /q "%PORTABLE_DIR%"
mkdir "%PORTABLE_DIR%"
mkdir "%PORTABLE_DIR%\data"
mkdir "%PORTABLE_DIR%\configs"

:: Copy Tauri app exe
set "TAURI_EXE_COPIED=0"
if exist "%TAURI_TARGET%\tunnelcraft-ui.exe" (
    copy /y "%TAURI_TARGET%\tunnelcraft-ui.exe" "%PORTABLE_DIR%\TunnelCraft.exe" >nul
    echo [OK] TunnelCraft.exe copied.
    set "TAURI_EXE_COPIED=1"
)
if "!TAURI_EXE_COPIED!"=="0" if exist "%TAURI_TARGET%\TunnelCraft.exe" (
    copy /y "%TAURI_TARGET%\TunnelCraft.exe" "%PORTABLE_DIR%\TunnelCraft.exe" >nul
    echo [OK] TunnelCraft.exe copied.
    set "TAURI_EXE_COPIED=1"
)
if "!TAURI_EXE_COPIED!"=="0" (
    echo [WARN] Tauri exe not found in target/release/. Searching...
    for /f "delims=" %%f in ('dir /b /s "%TAURI_TARGET%\*.exe" 2^>nul ^| findstr /i "tunnelcraft"') do (
        copy /y "%%f" "%PORTABLE_DIR%\TunnelCraft.exe" >nul 2>&1
        echo [OK] Found and copied: %%~nxf
        set "TAURI_EXE_COPIED=1"
    )
)

:: Copy Go daemon
copy /y "bin\tunnelcraftd.exe" "%PORTABLE_DIR%\tunnelcraftd.exe" >nul
echo [OK] tunnelcraftd.exe copied.

:: Copy external binaries
for %%f in (xray-core.exe sing-box.exe hysteria.exe wintun.dll wireguard.exe amneziawg-go.exe wireguard-go.exe) do (
    if exist "bin\%%f" (
        copy /y "bin\%%f" "%PORTABLE_DIR%\%%f" >nul
        echo [OK] %%f copied.
    ) else (
        echo [WARN] bin\%%f not found, skipping.
    )
)

:: Copy config templates
if exist "configs" (
    xcopy /y /q "configs\*" "%PORTABLE_DIR%\configs\" >nul 2>&1
    echo [OK] Config templates copied.
)

:: Copy portable start/stop scripts
copy /y "bin\start.bat" "%PORTABLE_DIR%\start.bat" >nul 2>&1
copy /y "bin\stop.bat" "%PORTABLE_DIR%\stop.bat" >nul 2>&1
echo [OK] start.bat / stop.bat copied.

echo.

:: ==========================================================
:: STEP 6: Summary
:: ==========================================================
echo [6/6] Build complete!
echo.
echo ============================================================
echo   Portable folder: %PORTABLE_DIR%
echo.
echo   Copy the entire TunnelCraft-Portable folder to another PC
and run TunnelCraft.exe to start the app.
echo.
echo   NOTE: The target PC needs WebView2 Runtime.
echo   If not installed, the app will auto-download it.
echo ============================================================
echo.
echo Installers (if you prefer):
for /f "delims=" %%f in ('dir /b /s "ui\src-tauri\target\release\bundle\*.msi" 2^>nul') do (
    echo   [MSI]   %%f
)
for /f "delims=" %%f in ('dir /b /s "ui\src-tauri\target\release\bundle\*.exe" 2^>nul') do (
    echo   [NSIS]  %%f
)
echo.
goto :end

:fail
echo.
echo ============================================================
echo   Build aborted due to error.
echo ============================================================
pause
exit /b 1

:end
pause
