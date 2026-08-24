@echo off
title TunnelCraft - Stopping
echo Stopping TunnelCraft...
taskkill /F /IM tunnelcraftd.exe >nul 2>&1
taskkill /F /IM sing-box.exe >nul 2>&1
taskkill /F /IM xray-core.exe >nul 2>&1
echo TunnelCraft stopped.