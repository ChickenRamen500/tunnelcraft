@echo off
chcp 65001 >nul
setlocal EnableDelayedExpansion
title TunnelCraft Launcher
color 0A

echo.
echo ╔══════════════════════════════════════════════╗
echo ║         TunnelCraft - Auto Launcher          ║
echo ╚══════════════════════════════════════════════╝
echo.

:: ===== 1. Переходим в директорию проекта =====
cd /d "%~dp0"
echo [1/6] Рабочая директория: %CD%

:: ===== 2. Проверка Git =====
where git >nul 2>&1
if %ERRORLEVEL% neq 0 (
    color 0C
    echo [ОШИБКА] Git не найден! Установите: https://git-scm.com/download/win
    goto :fail
)

:: ===== 3. Проверка Go =====
where go >nul 2>&1
if %ERRORLEVEL% neq 0 (
    color 0C
    echo [ОШИБКА] Go не найден! Установите: https://go.dev/dl/
    goto :fail
)

:: ===== 4. Проверка Node.js =====
where node >nul 2>&1
if %ERRORLEVEL% neq 0 (
    color 0C
    echo [ОШИБКА] Node.js не найден! Установите: https://nodejs.org/
    goto :fail
)

:: ===== 5. Проверка Rust =====
where cargo >nul 2>&1
if %ERRORLEVEL% neq 0 (
    color 0C
    echo [ОШИБКА] Rust не найден! Установите: https://rustup.rs/
    goto :fail
)
echo [2/6] Все зависимости найдены (Git, Go, Node, Rust)

:: ===== 6. Выбор режима =====
echo.
echo Выберите режим запуска:
echo   [1] Разработка (dev mode с hot-reload)
echo   [2] Обновить из Git (только git pull)
echo   [3] Полная пересборка (git pull + binaries + build)
echo.
set /p choice="Введите номер (1/2/3) [по умолчанию 1]: "
if "!choice!"=="" set choice=1

:: ===== 7. Git Pull =====
echo.
echo [3/6] Обновление кода из GitHub...
git pull --ff-only
if %ERRORLEVEL% neq 0 (
    echo [ПРЕДУПРЕЖДЕНИЕ] git pull завершился с ошибкой, продолжаем с локальным кодом
)

if "!choice!"=="2" (
    echo.
    echo Готово! Код обновлён.
    pause
    exit /b 0
)

:: ===== 8. Скачивание бинарей (xray, hysteria, wintun, amneziawg) =====
if not exist "bin\xray-core.exe" (
    echo.
    echo [4/6] Скачивание бинарных зависимостей (первый запуск)...
    if exist "scripts\download-binaries.ps1" (
        powershell -ExecutionPolicy Bypass -File scripts\download-binaries.ps1
        if %ERRORLEVEL% neq 0 (
            color 0C
            echo [ОШИБКА] Не удалось скачать бинарники! Проверьте интернет.
            goto :fail
        )
    ) else (
        color 0C
        echo [ОШИБКА] Файл scripts\download-binaries.ps1 не найден!
        goto :fail
    )
) else (
    echo [4/6] Бинарники уже на месте (пропущено)
)

if "!choice!"=="3" (
    echo.
    echo [4.5/6] Принудительная пересборка бинарей...
    powershell -ExecutionPolicy Bypass -File scripts\download-binaries.ps1
)

:: ===== 9. Сборка Go демона =====
echo.
echo [5/6] Сборка Go демона (tunnelcraftd.exe)...
pushd core\cmd\tunnelcraftd
go build -o ..\..\..\bin\tunnelcraftd.exe .
if %ERRORLEVEL% neq 0 (
    color 0C
    echo [ОШИБКА] Сборка Go демона не удалась!
    popd
    goto :fail
)
popd
echo [OK] tunnelcraftd.exe собран

:: ===== 10. Запуск Tauri в dev режиме =====
echo.
echo [6/6] Запуск Tauri приложения...
pushd ui
if not exist "node_modules" (
    echo.
    echo Установка npm пакетов (первый раз, может занять 2-5 минут)...
    call npm install
    if %ERRORLEVEL% neq 0 (
        color 0C
        echo [ОШИБКА] npm install не удался!
        popd
        goto :fail
    )
)
echo.
echo ╔══════════════════════════════════════════════╗
echo ║        Запускаю TunnelCraft...               ║
echo ║   Не закрывайте это окно, пока работаете!    ║
echo ╚══════════════════════════════════════════════╝
echo.
call npm run tauri dev
popd
goto :end

:fail
echo.
echo ╔══════════════════════════════════════════════╗
echo ║        Запуск прерван из-за ошибки          ║
echo ╚══════════════════════════════════════════════╝
pause
exit /b 1

:end
echo.
echo ╔══════════════════════════════════════════════╗
echo ║        TunnelCraft остановлен                ║
echo ╚══════════════════════════════════════════════╝
pause