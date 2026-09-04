# Trading / moex-muhlya — платформа Мухля / Мия

Открытый репозиторий торгового ассистента для **MOEX** через брокера **Finam**: десктоп-терминал, headless-автоторговля (**Мухля**), мониторинг (**Мия**), риск-контур и офлайн-лаборатория политик.

> Прибыль не гарантируется. Это исследовательский/операционный инструмент с упором на контроль риска. Торговля на реальные деньги — только на ваш страх и риск.

## Что здесь главное

| Каталог | Назначение |
|--------|------------|
| **`moex-trader-gui/`** | Основной терминал (Wails + Vue) и **headless**-автоторговля **Мухля** (`-tags headless`), Finam, MOEX ISS, Postgres. |
| **`trading-core/`** | Общий Go-модуль движка (риск, режимы и т.д.). |
| **`moex-trader/`** | Наследие: CLI и старый desktop-контур; **боевой VPS обычно не отсюда**. |
| **`trading-lab/`** | Python: бэктесты, RL, политики — исследования, не runtime VPS. |
| **`docs/`** | Заметки по эксплуатации (в т.ч. VPS/SSH). |
| **Корень** | Контекст чатов (`CHAT_*.md`, `SESSION_LOG.md`, …), скрипты `scripts/`. |

## Быстрый старт (разработка)

1. Postgres: из каталога `moex-trader-gui` — `docker compose up -d` (см. `docker-compose.yml`).
2. Скопировать `moex-trader-gui/.env.example` → `.env`, заполнить секреты Finam и БД.
3. GUI: `wails dev` или `wails build` (см. `moex-trader-gui/README.md`).
4. Headless (локально): `go build -tags headless -o muhlya-headless .` в `moex-trader-gui`.

## VPS (кратко)

- Хост в примерах: плейсхолдер **`YOUR_VPS_HOST`** (подставьте свой IP/DNS).
- Сервис **`muhlya`**: бинарь `muhlya-headless` в `/opt/trading/app/moex-trader-gui/`, конфиг из `.env`, ожидание healthy Postgres в unit.
- **Мия**: `scripts/vps/miya-report.sh`, `miya-online-watch.sh` + **systemd** `.timer` / `.service` в `scripts/vps/systemd/`.
- Отчёты и сигналы: обычно `/var/log/muhlya-reports/`.

Подробнее: `docs/VPS_SSH_LINUX_BASICS.md`.

## Правила безопасности

- Не коммитить `.env` и токены Finam / пароли БД.
- Не публиковать бинарники `muhlya-headless*` и реальные адреса серверов.
- Диагностические SQL и временные файлы — не в корне репозитория (см. `.gitignore`).

## Участие

Issues и Pull Request приветствуются: багфиксы, тесты, документация, улучшения риск-контура и лаборатории. Перед PR — локально `go test` / сборка затронутых модулей; секреты в diff не класть.
