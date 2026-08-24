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

cd /d "%~dp0"
echo [INFO] Working directory: %CD%
echo.

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

echo [OK] All tools found.
echo.

echo [1/6] Pulling latest changes from GitHub...
git pull --ff-only
if %ERRORLEVEL% neq 0 (
    echo [WARN] git pull failed. Continuing with local code...
) else (
    echo [OK] Repository updated.
)
echo.

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

echo       Compiling frontend + Rust release...
call npm run tauri build
set "TAURI_BUILD_ERR=%ERRORLEVEL%"
if %TAURI_BUILD_ERR% neq 0 (
    echo [WARN] Tauri bundle step failed (MSI/NSIS installer not created).
    echo [INFO] Checking if the exe was still built...
    if exist "src-tauri\target\release\tunnelcraft-ui.exe" (
        echo [OK] tunnelcraft-ui.exe exists, continuing with portable build.
    ) else (
        color 0C
        echo [ERROR] Tauri exe not found. Build failed completely.
        popd
        goto :fail
    )
) else (
    echo [OK] Tauri release built successfully.
)
popd
echo.

echo [5/6] Creating portable distribution...

set "PORTABLE_DIR=%CD%\TunnelCraft-Portable"
set "TAURI_TARGET=ui\src-tauri\target\release"

if exist "%PORTABLE_DIR%" rmdir /s /q "%PORTABLE_DIR%"
mkdir "%PORTABLE_DIR%"
mkdir "%PORTABLE_DIR%\data"
mkdir "%PORTABLE_DIR%\configs"

set "TAURI_EXE_COPIED=0"
if exist "%TAURI_TARGET%\tunnelcraft-ui.exe" (
    copy /y "%TAURI_TARGET%\tunnelcraft-ui.exe" "%PORTABLE_DIR%\TunnelCraft.exe" >nul
    echo [OK] TunnelCraft.exe copied.
    set "TAURI_EXE_COPIED=1"
)
if "!TAURI_EXE_COPIED!"=="0" (
    if exist "%TAURI_TARGET%\TunnelCraft.exe" (
        copy /y "%TAURI_TARGET%\TunnelCraft.exe" "%PORTABLE_DIR%\TunnelCraft.exe" >nul
        echo [OK] TunnelCraft.exe copied.
        set "TAURI_EXE_COPIED=1"
    )
)
if "!TAURI_EXE_COPIED!"=="0" (
    echo [WARN] Tauri exe not found in target/release/.
)

copy /y "bin\tunnelcraftd.exe" "%PORTABLE_DIR%\tunnelcraftd.exe" >nul
echo [OK] tunnelcraftd.exe copied.

echo [INFO] Copying external binaries...
if exist "bin\xray-core.exe" (
    copy /y "bin\xray-core.exe" "%PORTABLE_DIR%\xray-core.exe" >nul
    echo [OK] xray-core.exe
)
if exist "bin\sing-box.exe" (
    copy /y "bin\sing-box.exe" "%PORTABLE_DIR%\sing-box.exe" >nul
    echo [OK] sing-box.exe
)
if exist "bin\hysteria.exe" (
    copy /y "bin\hysteria.exe" "%PORTABLE_DIR%\hysteria.exe" >nul
    echo [OK] hysteria.exe
)
if exist "bin\wintun.dll" (
    copy /y "bin\wintun.dll" "%PORTABLE_DIR%\wintun.dll" >nul
    echo [OK] wintun.dll
)
if exist "bin\wireguard.exe" (
    copy /y "bin\wireguard.exe" "%PORTABLE_DIR%\wireguard.exe" >nul
    echo [OK] wireguard.exe
)
if exist "bin\amneziawg-go.exe" (
    copy /y "bin\amneziawg-go.exe" "%PORTABLE_DIR%\amneziawg-go.exe" >nul
    echo [OK] amneziawg-go.exe
)
if exist "bin\wireguard-go.exe" (
    copy /y "bin\wireguard-go.exe" "%PORTABLE_DIR%\wireguard-go.exe" >nul
    echo [OK] wireguard-go.exe
)

if exist "configs" (
    xcopy /y /q "configs\*" "%PORTABLE_DIR%\configs\" >nul 2>&1
    echo [OK] Config templates copied.
)

if exist "bin\start.bat" (
    copy /y "bin\start.bat" "%PORTABLE_DIR%\start.bat" >nul
)
if exist "bin\stop.bat" (
    copy /y "bin\stop.bat" "%PORTABLE_DIR%\stop.bat" >nul
)
echo [OK] start.bat / stop.bat copied.
echo.

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
if exist "ui\src-tauri\target\release\bundle" (
    echo Installers (if you prefer) are in:
    echo   ui\src-tauri\target\release\bundle\
    echo.
)
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
