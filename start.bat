@echo off
chcp 65001 >nul
title TunnelCraft Launcher
color 0A

echo ========================================
echo   TunnelCraft - Запуск...
echo ========================================
echo.

:: Переход в директорию скрипта
cd /d "%~dp0"

:: ===== ШАГ 1: Git Pull =====
echo [1/4] Обновление из репозитория...
git fetch --all >nul 2>&1
git pull --rebase origin main >nul 2>&1
if errorlevel 1 (
    echo     [!] Git pull пропущен (нет интернета или конфликты^)
) else (
    echo     [OK] Обновлено
)
echo.

:: ===== ШАГ 2: Проверка Go =====
echo [2/4] Проверка Go...
where go >nul 2>nul
if errorlevel 1 (
    echo     [X] Go не найден!
    echo     Скачай с https://go.dev/dl/
    pause
    exit /b 1
)
for /f "tokens=*" %%i in ('go version') do echo     [OK] %%i
echo.

:: ===== ШАГ 3: Компиляция демона =====
echo [3/4] Компиляция tunnelcraftd...
cd core\cmd\tunnelcraftd
go build -o ..\..\..\bin\tunnelcraftd.exe . 2>&1
if errorlevel 1 (
    echo     [X] Ошибка компиляции!
    pause
    exit /b 1
)
cd ..\..\..
echo     [OK] Демон скомпилирован
echo.

:: ===== ШАГ 4: Запуск Tauri =====
echo [4/4] Запуск Tauri приложения...
echo     (Это может занять 30-60 секунд при первом запуске^)
echo.
cd ui

:: Установить зависимости если node_modules нет
if not exist "node_modules\" (
    echo     Установка npm зависимостей...
    call npm install --silent
    echo.
)

:: Запустить Tauri в dev режиме
call npm run tauri dev

:: ===== Завершение =====
echo.
echo ========================================
echo   Приложение закрыто
echo ========================================
timeout /t 3 >nul