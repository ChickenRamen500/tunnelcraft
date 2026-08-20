# TunnelCraft

**TunnelCraft** — VPN-клиент для Windows 10/11, оборачивающий open-source реализации протоколов в единый GUI.

## Поддерживаемые протоколы

| Протокол | Бинарник | Описание |
|----------|----------|----------|
| VLESS | xray-core.exe | VLESS, VLESS+KCP, VLESS+XHTTP+REALITY |
| VMESS | xray-core.exe | VMess с поддержкой транспорта |
| WireGuard | wireguard.exe | Классический WireGuard (через Windows Service) |
| AmneziaWG | amnezia-wg.exe | **AWG2 + AWG3** (HeaderProtectionKey, timing, content padding) |
| Hysteria2 | hysteria.exe | Hysteria2 QUIC-протокол |

### AWG3 — что нового

AmneziaWG 3.0 добавляет:
- **HeaderProtectionKey** — ChaCha20-шифрование заголовков пакетов (DPI не видит тип сообщения)
- **ContentPaddingAddition** — кастомный паддинг содержимого
- **Тайминги** — RekeyAfterTime, RekeyTimeout, RejectAfterTime, KeepaliveTimeout, MaxHandshakeAttempts

Все AWG3 поля поддерживаются по всей цепочке: парсинг подписок → конфиг → gRPC → генерация .conf → amnezia-wg.exe.

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
│  Protocol Binaries  │  xray-core, wireguard,
│  (bin/)             │  amnezia-wg, hysteria
└─────────────────────┘
```

## Ключевые возможности

- **Единый интерфейс** для всех протоколов VPN
- **AWG2 + AWG3** — полная поддержка AmneziaWG 3.0 (HeaderProtectionKey и тайминги)
- **Автоматический fallback** — при недоступности серверов подписки переключается на локальный WireGuard/AmneziaWG
- **Split tunneling** — маршрутизация по доменам, IP-адресам, приложениям
- **Авто-обновление подписок** — фоновое обновление серверов
- **Системный трей** — индикатор статуса в трее (зелёный/красный/жёлтый)
- **Kill switch** — блокировка трафика при разрыве VPN
- **DNS-over-HTTPS** — безопасное разрешение DNS

---

## 🚀 Установка и запуск (пошагово)

### Шаг 1. Установить prerequisites

#### 1a. Go 1.22+
Скачай с https://go.dev/dl/ и установи. Проверь:
```
go version
```

#### 1b. Node.js 18+
Скачай с https://nodejs.org/ (LTS версию). Проверь:
```
node -v
npm -v
```

#### 1c. Rust (через rustup)
Скачай с https://rustup.rs/ и установи. Проверь:
```
rustc --version
cargo --version
```

#### 1d. **Visual Studio Build Tools** (ОБЯЗАТЕЛЬНО для Tauri!)

Без этого Rust не сможет скомпилировать Tauri — будет ошибка `linker link.exe not found`.

1. Скачай **Visual Studio Build Tools 2022**: https://visualstudio.microsoft.com/downloads/#build-tools-for-visual-studio-2022
2. Запусти установщик
3. Выбери **"Desktop development with C++"** (Настольная разработка на C++)
4. Нажми "Установить" (размер ~6-8 ГБ)

После установки перезагрузи терминал (закрой и открой PowerShell заново).

#### 1e. Tauri CLI 2.0
```
cargo install tauri-cli --version "^2"
```

### Шаг 2. Клонировать репозиторий

```powershell
git clone https://github.com/ChickenRamen500/tunnelcraft.git
cd tunnelcraft
```

### Шаг 3. Скачать бинарники протоколов

```powershell
powershell -ExecutionPolicy Bypass -File scripts\download-binaries.ps1
```

Это скачает: xray-core.exe, hysteria.exe, wintun.dll, wireguard.exe.

**amnezia-wg.exe** скачивается отдельно (см. ниже).

### Шаг 4. Получить amnezia-wg.exe

Репозиторий `amnezia-vpn/amneziawg-windows` **не имеет релизов**. Варианты:

**Вариант А: Из приложения AmneziaVPN**
1. Установи AmneziaVPN с https://amnezia.org/downloads
2. Найди `amnezia-wg.exe` в директории установки (обычно `C:\Program Files\AmneziaVPN\`)
3. Скопируй в `bin\amnezia-wg.exe`

**Вариант Б: Собрать из исходников**
```
set GOOS=windows
set GOARCH=amd64
go install github.com/amnezia-vpn/amnezia-wg/cmd/amnezia-wg@latest
```
Затем скопируй `%GOPATH%\bin\amnezia-wg.exe` в `bin\`.

**Вариант В: У тебя уже есть файл**
Просто положи `amnezia-wg.exe` в папку `bin\` проекта.

### Шаг 5. Собрать Go-демон

```powershell
cd core
go build -o ..\bin\tunnelcraftd.exe .\cmd\tunnelcraftd\main.go
cd ..
```

### Шаг 6. Запустить в dev-режиме

```powershell
cd ui
npm install
npx tauri dev
```

Откроется окно TunnelCraft.

---

## Альтернативный запуск (только бэкенд)

Если хочешь протестировать только Go-демон без UI:

```powershell
# Терминал 1: запустить демон
.\bin\tunnelcraftd.exe

# Терминал 2: отправить gRPC-запрос (например, получить список серверов)
# (требует grpcurl или аналогичный инструмент)
```

---

## Структура проекта

```
tunnelcraft/
├── core/                  # Go демон (tunnelcraftd)
│   ├── cmd/tunnelcraftd/  # Точка входа
│   ├── internal/          # Внутренние пакеты
│   │   ├── engine/        # Управление подключениями
│   │   ├── protocols/     # Обёртки для бинарников
│   │   ├── tunnel/        # TUN + маршрутизация
│   │   ├── subscription/  # Парсер подписок (AWG2/AWG3 URI, JSON, .conf, sing-box, clash)
│   │   ├── dns/           # DNS управление
│   │   ├── ipc/           # gRPC сервер
│   │   └── config/        # Конфигурация (YAML)
│   ├── go.mod
│   └── go.sum
├── ui/                    # Tauri 2.0 приложение
│   ├── src-tauri/         # Rust бэкенд
│   └── src/               # React фронтенд
├── proto/                 # gRPC определения (tunnelcraft.proto)
├── bin/                   # Бинарники протоколов (НЕ в git)
├── configs/               # Шаблоны конфигов
└── scripts/               # Утилиты (download-binaries.ps1)
```

## Лицензия

MIT — см. файл [LICENSE](LICENSE).
