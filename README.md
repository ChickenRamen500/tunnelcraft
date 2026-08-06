# TunnelCraft

**TunnelCraft** — это VPN-клиент для Windows 10/11, который оборачивает существующие open-source реализации протоколов в единый удобный GUI.

## Поддерживаемые протоколы

| Протокол | Бинарник | Описание |
|----------|----------|----------|
| VLESS | xray-core.exe | VLESS, VLESS+KCP, VLESS+XHTTP+REALITY |
| VMESS | xray-core.exe | VMess с поддержкой транспорта |
| WireGuard | wireguard-go.exe | Классический WireGuard |
| AmneziaWG | amnezia-wg.exe | AmneziaWG / AWG2 |
| Hysteria2 | hysteria.exe | Hysteria2 QUIC-протокол |

## Архитектура

```
┌─────────────────────┐
│   Tauri 2.0 UI      │  React + TypeScript + Vite
│   (ui/src/)         │  Тёмная тема, системный трей
└──────────┬──────────┘
           │ gRPC (127.0.0.1:50051)
┌──────────▼──────────┐
│  tunnelcraftd        │  Go 1.22+ демон
│  (core/)            │  Управление протоколами,
│                     │  TUN-интерфейс, маршрутизация,
│                     │  подписки, fallback
└──────────┬──────────┘
           │ subprocess
┌──────────▼──────────┐
│  Protocol Binaries  │  xray-core, wireguard-go,
│  (bin/)             │  amnezia-wg, hysteria
└─────────────────────┘
```

## Ключевые возможности

- **Единый интерфейс** для всех протоколов VPN
- **Автоматический fallback** — при недоступности серверов подписки переключается на локальный WireGuard/AmneziaWG
- **Split tunneling** — маршрутизация по доменам, IP-адресам, приложениям
- **Авто-обновление подписок** — фоновое обновление серверов
- **Системный трей** — индикатор статуса в трее (зелёный/красный/жёлтый)
- **Kill switch** — блокировка трафика при разрыве VPN
- **DNS-over-HTTPS** — безопасное разрешение DNS

## Сборка и запуск

### Предварительные требования

- Go 1.22+
- Node.js 18+
- Rust (rustup)
- Tauri CLI 2.0

### Установка бинарников

```powershell
# Windows PowerShell
powershell -ExecutionPolicy Bypass -File scripts/download-binaries.ps1
```

### Запуск бэкенда

```bash
cd core && go build -o ../bin/tunnelcraftd.exe ./cmd/tunnelcraftd && ../bin/tunnelcraftd.exe
```

### Запуск фронтенда

```bash
cd ui && npm install && npm run tauri dev
```

## Структура проекта

```
tunnelcraft/
├── core/                  # Go демон (tunnelcraftd)
│   ├── cmd/tunnelcraftd/  # Точка входа
│   ├── internal/          # Внутренние пакеты
│   │   ├── engine/        # Управление подключениями
│   │   ├── protocols/     # Обёртки для бинарников
│   │   ├── tunnel/        # TUN + маршрутизация
│   │   ├── subscription/  # Парсер подписок
│   │   ├── dns/           # DNS управление
│   │   ├── ipc/           # gRPC сервер
│   │   └── config/        # Конфигурация
│   ├── go.mod
│   └── go.sum
├── ui/                    # Tauri 2.0 приложение
│   ├── src-tauri/         # Rust бэкенд
│   └── src/               # React фронтенд
├── proto/                 # gRPC определения
├── bin/                   # Бинарники (не в git)
├── configs/               # Шаблоны конфигов
└── scripts/               # Утилиты
```

## Лицензия

MIT — см. файл [LICENSE](LICENSE).
