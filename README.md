# TunnelCraft

TunnelCraft — это VPN-клиент с современным интерфейсом, построенный на Tauri v2 (React + Rust) и Go демоне.

## Архитектура

- **Frontend**: React + TypeScript (ui/)
- **Backend**: Tauri v2 Rust приложение (ui/src-tauri/)
- **Daemon**: Go демон tunnelcraftd (core/)

## Быстрый старт

### 1. Установка зависимостей

Перед началом убедитесь, что у вас установлены:

- **Go 1.19+** (для сборки демона и wireguard/amnezia-wg)
- **Node.js 18+** (для frontend)
- **Rust 1.70+** (для сборки Tauri)
- **Git** (для клонирования репозиториев)
- **Windows 10/11** (для работы с WireGuard и TUN адаптерами)

### 2. Скачивание бинарных зависимостей

Скрипт автоматически скачает и скомпилирует все необходимые binaries:

```powershell
powershell -ExecutionPolicy Bypass -File scripts\download-binaries.ps1
```

Что входит в скрипт:
- **xray-core.exe** — скачивается с GitHub releases Xray-core
- **hysteria.exe** — скачивается с GitHub releases Hysteria
- **wintun.dll** — скачивается с wintun.net
- **wireguard.exe** — копируется из установленной WireGuard или строится из исходников
- **wireguard-go.exe** — строится из исходников через `go install`
- **amnezia-wg.exe** — **клонируется репозиторий amnezia-vpn/amneziawg-windows и компилируется**

> **Примечание:** Для компиляции amnezia-wg.exe требуется установленный Go. Скрипт автоматически:
> 1. Клонирует репозиторий https://github.com/amnezia-vpn/amneziawg-windows во временную папку
> 2. Компилирует `amnezia-wg.exe` для Windows amd64
> 3. Копирует результат в `bin/amnezia-wg.exe`
> 4. Очищает временные файлы

Если Go не установлен, вы можете:
- Установить Go и перезапустить скрипт
- Или скопировать `amnezia-wg.exe` из установки AmneziaVPN в папку `bin/`

### 3. Сборка Go демона

```bash
./build_daemon.sh
```

Или вручную:

```bash
cd core/cmd/tunnelcraftd
go build -o ../../bin/tunnelcraftd.exe .
```

Это создаст бинарный файл `bin/tunnelcraftd.exe`.

### 4. Запуск в режиме разработки

#### Терминал 1: Запуск Tauri приложения

```bash
cd ui
npm install
npm run tauri dev
```

Приложение автоматически:
1. Запустит `tunnelcraftd.exe` как sidecar процесс
2. Дождётся готовности HTTP API на порту 50052
3. Откроет окно приложения

> **Примечание:** Демон запускается автоматически при старте Tauri приложения. Ручной запуск нужен только для отладки.

#### Опционально: Ручной запуск демона для отладки

```bash
cd core/cmd/tunnelcraftd
go run . --data ./data --config ./data/config.yaml
```


## API Демона

Go демон предоставляет HTTP REST API на `http://127.0.0.1:50052`:

| Endpoint | Метод | Описание |
|----------|-------|----------|
| `/api/health` | GET | Проверка здоровья демона |
| `/api/status` | GET | Текущий статус подключения |
| `/api/connect` | POST | Подключиться к серверу |
| `/api/disconnect` | POST | Отключиться |
| `/api/servers` | GET | Список серверов |
| `/api/servers/import` | POST | Импортировать сервер из конфига |
| `/api/subscriptions` | GET/POST | Управление подписками |
| `/api/subscriptions/refresh/{id}` | POST | Обновить подписку |
| `/api/settings` | GET/PUT | Настройки приложения |
| `/api/logs` | GET | Логи демона |

### Примеры запросов

```bash
# Проверка здоровья
curl http://127.0.0.1:50052/api/health

# Получить статус
curl http://127.0.0.1:50052/api/status

# Добавить подписку
curl -X POST http://127.0.0.1:50052/api/subscriptions \
  -H "Content-Type: application/json" \
  -d '{"name":"My VPN","url":"https://example.com/sub"}'

# Получить логи
curl http://127.0.0.1:50052/api/logs?limit=50
```

## Структура проекта

```
tunnelcraft/
├── bin/                    # Бинарные файлы (демон и зависимости)
│   └── tunnelcraftd.exe
├── core/                   # Go демон
│   ├── cmd/tunnelcraftd/   # Точка входа демона
│   └── internal/           # Внутренние пакеты Go
├── ui/                     # Tauri приложение
│   ├── src/                # React frontend
│   │   ├── components/     # UI компоненты
│   │   ├── hooks/          # React хуки (useApi.ts - HTTP клиент)
│   │   ├── pages/          # Страницы приложения (Dashboard, Servers, Subscriptions, Logs, Settings)
│   │   └── stores/         # Zustand stores (connection, settings)
│   └── src-tauri/          # Rust backend
│       ├── src/
│       │   ├── main.rs     # Точка входа Tauri
│       │   └── commands.rs # Tauri команды
│       └── tauri.conf.json # Конфигурация Tauri
├── build_daemon.sh         # Скрипт сборки демона
└── README.md
```

## Отладка

### Просмотр логов демона

```bash
curl http://127.0.0.1:50052/api/logs?limit=100
```

### Проверка работы демона

```bash
curl http://127.0.0.1:50052/api/health
```

Если демон не отвечает:
1. Проверьте, что `bin/tunnelcraftd.exe` существует
2. Проверьте логи Tauri приложения в консоли разработчика (F12)
3. Попробуйте запустить демон вручную (см. выше)

## Сборка релиза

```bash
cd ui
npm run tauri build
```

Это создаст установочный пакет в `ui/src-tauri/target/release/bundle/`.

## Требования

- **Go 1.19+** (для сборки демона и wireguard/amnezia-wg)
- **Node.js 18+** (для frontend)
- **Rust 1.70+** (для сборки Tauri)
- **Git** (для клонирования репозиториев)
- **Windows 10/11** (для работы с WireGuard и TUN адаптерами)
- **7-Zip** (опционально, для извлечения wireguard-go.exe из установщика WireGuard)

## Лицензия

MIT
