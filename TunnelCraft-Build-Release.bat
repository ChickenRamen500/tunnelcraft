@echo off
chcp 65001 >nul
title TunnelCraft Release Builder
cd /d "%~dp0"

echo Сборка релизного пакета TunnelCraft...
echo.

:: Git pull
git pull

:: Binaries
powershell -ExecutionPolicy Bypass -File scripts\download-binaries.ps1

:: Go daemon
pushd core\cmd\tunnelcraftd
go build -ldflags "-s -w" -o ..\..\..\bin\tunnelcraftd.exe .
popd

:: Tauri build
pushd ui
if not exist "node_modules" call npm install
call npm run tauri build
popd

echo.
echo Готово! Установщик находится в:
echo   ui\src-tauri\target\release\bundle\
pause