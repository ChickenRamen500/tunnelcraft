@echo off
title TunnelCraft
echo ========================================
echo   TunnelCraft - Portable VPN Client
echo ========================================
echo.

REM Check if already running
tasklist /FI "IMAGENAME eq tunnelcraftd.exe" 2>NUL | find /I "tunnelcraftd.exe" >NUL
if %ERRORLEVEL% equ 0 (
    echo TunnelCraft daemon is already running.
    echo Opening web UI...
    start http://127.0.0.1:50052
    goto :end
)

echo Starting TunnelCraft daemon...
start /B tunnelcraftd.exe --data .\data --config .\data\config.yaml --bin .

REM Wait for daemon to be ready
echo Waiting for daemon...
set retries=0
:waitloop
set /a retries+=1
if %retries% gtr 20 goto :timeout
timeout /t 1 /nobreak >nul
curl -s http://127.0.0.1:50052/api/health >nul 2>&1
if %ERRORLEVEL% neq 0 goto :waitloop

echo Daemon is ready!
echo.
echo Opening web UI in browser...
start http://127.0.0.1:50052
echo.
echo ========================================
echo   TunnelCraft is running
echo   Web UI: http://127.0.0.1:50052
echo   API:    http://127.0.0.1:50052/api/health
echo   Close this window to stop TunnelCraft
echo ========================================
goto :end

:timeout
echo ERROR: Daemon failed to start within 20 seconds.
echo Check data/config.yaml for errors.
pause

:end