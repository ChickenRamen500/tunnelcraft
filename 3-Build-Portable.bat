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

if not "!MISSING!"=="" goto :err_missing

echo [OK] All tools found.
echo.

echo [1/6] Pulling latest changes from GitHub...
git pull --ff-only
if %ERRORLEVEL% neq 0 goto :warn_pull
echo [OK] Repository updated.
goto :step2

:warn_pull
echo [WARN] git pull failed. Continuing with local code.
:step2
echo.

echo [2/6] Downloading external binaries...
if exist "scripts\download-binaries.ps1" goto :run_ps1
echo [WARN] scripts\download-binaries.ps1 not found.
goto :step3

:run_ps1
powershell -ExecutionPolicy Bypass -File scripts\download-binaries.ps1
if %ERRORLEVEL% neq 0 echo [WARN] Some binaries failed to download.

:step3
echo.
echo [3/6] Building Go daemon (optimized release)...
pushd core\cmd\tunnelcraftd
go build -ldflags "-s -w" -o ..\..\..\bin\tunnelcraftd.exe .
if %ERRORLEVEL% neq 0 goto :err_go_build
popd
echo [OK] tunnelcraftd.exe built (stripped).
echo.

echo [4/6] Building Tauri release...
pushd ui
if not exist "node_modules" goto :install_npm
goto :build_tauri

:install_npm
echo Installing npm dependencies...
call npm install
if %ERRORLEVEL% neq 0 goto :err_npm

:build_tauri
echo Running tauri build...
call npm run tauri build
if %ERRORLEVEL% neq 0 goto :err_tauri
popd
echo [OK] Tauri release built.
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
    set "TAURI_EXE_COPIED=1"
)
if "!TAURI_EXE_COPIED!"=="0" if exist "%TAURI_TARGET%\TunnelCraft.exe" (
    copy /y "%TAURI_TARGET%\TunnelCraft.exe" "%PORTABLE_DIR%\TunnelCraft.exe" >nul
    set "TAURI_EXE_COPIED=1"
)

copy /y "bin\tunnelcraftd.exe" "%PORTABLE_DIR%\tunnelcraftd.exe" >nul

if exist "bin\xray-core.exe" copy /y "bin\xray-core.exe" "%PORTABLE_DIR%\" >nul
if exist "bin\sing-box.exe" copy /y "bin\sing-box.exe" "%PORTABLE_DIR%\" >nul
if exist "bin\hysteria.exe" copy /y "bin\hysteria.exe" "%PORTABLE_DIR%\" >nul
if exist "bin\wintun.dll" copy /y "bin\wintun.dll" "%PORTABLE_DIR%\" >nul
if exist "bin\wireguard.exe" copy /y "bin\wireguard.exe" "%PORTABLE_DIR%\" >nul
if exist "bin\amneziawg-go.exe" copy /y "bin\amneziawg-go.exe" "%PORTABLE_DIR%\" >nul
if exist "bin\wireguard-go.exe" copy /y "bin\wireguard-go.exe" "%PORTABLE_DIR%\" >nul

if exist "configs" xcopy /y /q "configs\*" "%PORTABLE_DIR%\configs\" >nul 2>&1
if exist "bin\start.bat" copy /y "bin\start.bat" "%PORTABLE_DIR%\" >nul
if exist "bin\stop.bat" copy /y "bin\stop.bat" "%PORTABLE_DIR%\" >nul

echo [OK] Portable folder created: %PORTABLE_DIR%
goto :end

:err_missing
color 0C
echo [ERROR] Missing tools:!MISSING!
echo Install Git, Go, Node.js, and Rust.
goto :fail

:err_go_build
color 0C
echo [ERROR] Go daemon build failed!
popd
goto :fail

:err_npm
color 0C
echo [ERROR] npm install failed!
popd
goto :fail

:err_tauri
color 0C
echo [ERROR] Tauri build failed!
popd
goto :fail

:fail
echo.
echo ============================================================
echo   Build aborted due to error.
echo ============================================================
pause
exit /b 1

:end
echo.
echo ============================================================
echo   Build complete! Copy TunnelCraft-Portable to another PC.
echo ============================================================
pause