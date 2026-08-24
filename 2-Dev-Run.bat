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

cd /d "%~dp0"
echo [INFO] Working directory: %CD%
echo.

where go >nul 2>&1
if %ERRORLEVEL% neq 0 goto :err_go

where node >nul 2>&1
if %ERRORLEVEL% neq 0 goto :err_node

where cargo >nul 2>&1
if %ERRORLEVEL% neq 0 goto :err_cargo

echo [OK] All tools found.
echo.

if exist "bin\xray-core.exe" goto :step2

echo [1/4] Downloading external binaries...
if exist "scripts\download-binaries.ps1" goto :run_ps1
echo [WARN] scripts\download-binaries.ps1 not found.
goto :step2

:run_ps1
powershell -ExecutionPolicy Bypass -File scripts\download-binaries.ps1
if %ERRORLEVEL% neq 0 echo [WARN] Some binaries failed to download.

:step2
echo.
echo [2/4] Building Go daemon...
pushd core\cmd\tunnelcraftd
go build -o ..\..\..\bin\tunnelcraftd.exe .
if %ERRORLEVEL% neq 0 goto :err_go_build
popd
echo [OK] tunnelcraftd.exe built successfully.
echo.

if exist "ui\node_modules" goto :step3

echo [3/4] Installing npm dependencies...
pushd ui
call npm install
if %ERRORLEVEL% neq 0 goto :err_npm
popd
echo [OK] npm dependencies installed.
echo.

:step3
echo [4/4] Launching TunnelCraft...
echo.
echo ============================================================
echo   TunnelCraft is starting. Do NOT close this window!
echo ============================================================
echo.
pushd ui
call npm run tauri dev
popd
goto :end

:err_go
color 0C
echo [ERROR] Go not found! Install: https://go.dev/dl/
goto :fail

:err_node
color 0C
echo [ERROR] Node.js not found! Install: https://nodejs.org/
goto :fail

:err_cargo
color 0C
echo [ERROR] Rust not found! Install: https://rustup.rs/
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