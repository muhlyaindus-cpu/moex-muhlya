# MOEX Trader (Мухля) — Wails + Go

Десктопный терминал и тот же код в режиме **headless** для VPS (автоторговля Finam / MOEX).

## Сборка

### GUI (Windows / macOS / Linux)

Требуется [Wails v2](https://wails.io/), Node.js для фронта.

```bash
cd frontend && npm install && npm run build && cd ..
wails build
```

### Headless (Мухля на сервере)

Сборка **без** UI (отдельная точка входа `headless_main.go`):

```bash
go build -tags headless -o muhlya-headless .
```

Кросс-сборка под Linux с Windows (PowerShell):

```powershell
$env:GOOS='linux'; $env:GOARCH='amd64'; go build -tags headless -o muhlya-headless .
Remove-Item Env:GOOS -ErrorAction SilentlyContinue; Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
```

## Конфигурация

- Акции vs FORTS, универсум авто, allowlist: **[`../docs/MOEX_FORTS_AUTO_UNIVERSE.md`](../docs/MOEX_FORTS_AUTO_UNIVERSE.md)**.
- Пример переменных: **`.env.example`** в этом каталоге.
- Рабочий файл: **`.env`** (не коммитить): Finam, Postgres, лимиты агента (`AGENT_*`), пути к файлам Мии (`AGENT_MIA_*`).
- Снимок портфеля для отчётов Мии (пишет Мухля в auto): `AGENT_BROKER_POSITION_SNAPSHOT_*`.

## Локальный Postgres

```bash
docker compose up -d
```

Порт и креды — из `.env` / `docker-compose.yml`.

## Структура (кратко)

- `app.go` — логика приложения, автолооп, риск, интеграция с Мией (файлы).
- `services/` — Finam, MOEX, Postgres, ассистент.
- `scripts/vps/` — **Мия** (bash), systemd unit-файлы, деплой-скрипты; онлайн-стакан/лучшие цены — см. [`../docs/MIYA_ORDERBOOK.md`](../docs/MIYA_ORDERBOOK.md).

## Разработка фронта

```bash
wails dev
```

Подробнее у Wails в официальной документации; шаблонный текст шаблона заменён этим README.
