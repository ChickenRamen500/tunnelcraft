# TunnelCraft

TunnelCraft — это VPN-клиент с современным интерфейсом, построенный на Tauri v2 (React + Rust) и Go демоне.

## Архитектура

- **Frontend**: React + TypeScript (ui/)
- **Backend**: Tauri v2 Rust приложение (ui/src-tauri/)
- **Daemon**: Go демон tunnelcraftd (core/)

## Быстрый старт

### 1. Сборка Go демона

```bash
./build_daemon.sh
```

Это создаст бинарный файл `bin/tunnelcraftd.exe`.

### 2. Запуск в режиме разработки

#### Терминал 1: Запуск Go демона (опционально)

Демон запускается автоматически при старте Tauri приложения, но вы можете запустить его вручную для отладки:

```bash
cd core/cmd/tunnelcraftd
go run . --data ./data --config ./data/config.yaml
```

#### Терминал 2: Запуск Tauri приложения

```bash
cd ui
npm install
npm run tauri dev
```

Приложение автоматически:
1. Запустит tunnelcraftd.exe как sidecar процесс
2. Дождётся готовности HTTP API на порту 50052
3. Откроет окно приложения

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
│   │   ├── hooks/          # React хуки (useGrpc.ts)
│   │   ├── pages/          # Страницы приложения
│   │   └── stores/         # Zustand stores
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

- Go 1.21+
- Node.js 18+
- Rust 1.70+ (для сборки Tauri)
- Windows 10/11 (для работы с WireGuard и TUN адаптерами)

## Лицензия

MIT
