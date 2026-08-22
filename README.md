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

## VLESS/Trojan/Hysteria2: как это работает

### Архитектура поддержки протоколов

TunnelCraft использует sidecar-бинарники для поддержки современных протоколов:

1. **sing-box.exe** (основное ядро)
   - Поддерживает: VLESS, Trojan, VMess, Hysteria, Hysteria2, Tuic, Shadowsocks
   - Транспорт: TCP, WS, gRPC, H2, HTTPUpgrade
   - Безопасность: TLS, REALITY
   - **TUN inbound**: создаёт виртуальный сетевой адаптер для полного туннелирования
   - Маршрутизация: обход RU/локальных сетей, блокировка IPv6, DNS через туннель

2. **xray-core.exe** (для KCP/XHTTP)
   - Запускается только если у сервера `transport = kcp` или `xhttp`
   - Слушает mixed inbound на `127.0.0.1:10810`
   - sing-box работает в режиме моста: TUN → SOCKS5 → xray

### Импорт ссылок

Поддерживаются форматы:
- `vless://UUID@host:port?type=ws&security=reality&sni=S&fp=chrome&pbk=K&sid=ID&flow=xtls-rprx-vision#Name`
- `trojan://password@host:port?security=tls&sni=S&type=grpc&serviceName=G#Name`
- `hysteria2://` или `hy2://password@host:port?sni=S&obfs=salamander&obfs-password=O#Name`
- `vmess://BASE64(JSON)`
- `ss://BASE64(method:pass)@host:port#Name`

### Режимы работы

- **TUN Mode**: полный захват трафика через виртуальный адаптер
- **Proxy Mode**: локальный SOCKS5/HTTP прокси
- **Bridge Mode**: TUN + перенаправление в xray (для KCP/XHTTP)

### Пинг и сортировка

- **TCP Ping**: измеряет время подключения к серверу (3 попытки)
- **Real Delay Test**: запускает временный sing-box и делает HTTP-запрос через туннель
- Сортировка: по пингу, имени, стране
- Группировка: по странам (флаги извлекаются из названия сервера)

### DNS Chain

Для резолва эндпоинтов используется цепочка DNS:
1. DoH (DNS over HTTPS) — Cloudflare, Google, Quad9
2. DoT (DNS over TLS) — те же провайдеры на порту 853
3. Plain DNS — обычный UDP на порту 53

Первый успешный ответ побеждает. Настраиваемые таймауты на попытку и общий дедлайн.

### Маскировка

Глобальные настройки маскировки:
- Отпечаток браузера (Chrome/Firefox/Curl)
- Домен прикрытия SNI
- Задержки мусорных пакетов (Jc, Jmin, Jmax)
- I-пакеты (I1–I5) для AmneziaWG
