# cleanup-pr.ps1
# Скрипт для очистки PR от мусорных файлов с сохранением структуры в папке _trash

$RootPath = Get-Location
$TrashFolder = Join-Path $RootPath "_trash"
$LegitBinPath = Join-Path $RootPath "bin" # Путь к легитимной папке bin (если она в корне)

# Если у вас структура tunnelcraft/bin, раскомментируйте строку ниже и закомментируйте верхнюю
# $LegitBinPath = Join-Path $RootPath "tunnelcraft\bin"

Write-Host "?? Начало очистки проекта..." -ForegroundColor Cyan
Write-Host "Легитимная папка bin (не будет тронута): $LegitBinPath" -ForegroundColor Green

# Список паттернов папок, которые считаются мусором
$JunkPatterns = @(
    "node_modules",
    "vendor",
    "dist",
    "build",
    ".next",
    "out",
    "target",     # Rust target
    "ui/src-tauri/target", # Tauri specific
    "ui/src-tauri/target",
    "__pycache__",
    ".pytest_cache",
    ".idea",
    ".vscode",    # Иногда полезно убрать из PR
    "logs",
    "tmp",
    "temp"
)

# Создаем папку для мусора
if (-not (Test-Path $TrashFolder)) {
    New-Item -ItemType Directory -Path $TrashFolder | Out-Null
    Write-Host "Создана папка для мусора: $TrashFolder"
}

$MovedCount = 0

foreach ($Pattern in $JunkPatterns) {
    # Ищем все папки с таким именем в проекте
    # Исключаем поиск внутри самой папки _trash
    $Folders = Get-ChildItem -Path $RootPath -Recurse -Directory -Force -ErrorAction SilentlyContinue | 
               Where-Object { 
                   $_.Name -eq $Pattern -and 
                   $_.FullName -notlike "$TrashFolder*" 
               }

    foreach ($Folder in $Folders) {
        $FullPath = $Folder.FullName
        
        # КРИТИЧЕСКАЯ ПРОВЕРКА: Игнорируем легитимную папку bin
        # Нормализуем пути для сравнения (чтобы слэши не мешали)
        $NormalizedFull = $FullPath.Replace('\', '/')
        $NormalizedLegit = $LegitBinPath.Replace('\', '/')
        
        if ($NormalizedFull -eq $NormalizedLegit) {
            Write-Host "?? Пропущено (легитимное): $FullPath" -ForegroundColor Yellow
            continue
        }
        
        # Дополнительная проверка: если папка называется bin, но НЕ является нашей легитимной
        if ($Pattern -eq "bin" -and $NormalizedFull -ne $NormalizedLegit) {
             # Это сторонний bin, который нужно убрать
        }

        # Вычисляем относительный путь от корня, чтобы сохранить структуру в _trash
        $RelativePath = $FullPath.Substring($RootPath.Path.Length).TrimStart('\')
        $DestPath = Join-Path $TrashFolder $RelativePath

        try {
            # Создаем директорию назначения
            $DestDir = Split-Path $DestPath -Parent
            if (-not (Test-Path $DestDir)) {
                New-Item -ItemType Directory -Path $DestDir -Force | Out-Null
            }

            # Перемещаем папку
            Move-Item -Path $FullPath -Destination $DestPath -Force
            Write-Host "? Перемещено: $RelativePath -> _trash/$RelativePath" -ForegroundColor Green
            $MovedCount++
        }
        catch {
            Write-Host "? Ошибка при перемещении $FullPath : $_" -ForegroundColor Red
        }
    }
}

Write-Host "----------------------------------------"
Write-Host "?? Готово! Перемещено папок: $MovedCount" -ForegroundColor Cyan
Write-Host "Проверьте папку '$TrashFolder'. Если всё верно, сделайте коммит изменений."
Write-Host "Чтобы окончательно удалить мусор позже, просто удалите папку _trash."