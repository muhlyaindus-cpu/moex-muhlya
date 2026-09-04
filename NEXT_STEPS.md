# Trading Project Status (MOEX + Finam)

## Кто мы
- Агент: Кристина
- Онлайн-агент мониторинга: Мия (дежурные срезы/отчёты на сервере, без исполнения сделок)
- Пользователь: Николай Александрович
- Роли Мухля / Мия (приоритеты, контракт файлов, чего не дублировать): **`docs/MIYA_MUHLYA_ROLES.md`**

## Цель проекта
- Создать **торгового ассистента** (терминал + сигналы + риск-контур) для счёта, открытого в **Finam**, с доступом к инструментам **MOEX**, разрешённым для этого счёта.
- Задача ассистента в продуктовом смысле: **увеличивать кэш** (денежные средства на счёте) за счёт **операций с ценными бумагами** в рамках лимитов и правил риск-менеджмента. Рост капитала не гарантирован; упор на **контроль риска**, воспроизводимость и проверку на истории/бумаге перед боем.

## Ритм обновления статуса (репо ↔ реальность)
- **После смысловой сессии:** короткий факт в `SESSION_LOG.md` сверху; если меняется картина «что важно сейчас» или прод-флаги — обновить **Оперативный фокус** в этом файле, предыдущую версию блока **заархивировать** (см. ниже), не удалять.
- **Раз в неделю:** `.\scripts\weekly-doc-review.ps1` (или чеклист в `COLLAB_STYLE.md` → «Еженедельная ревизия документов»); при необходимости `.\scripts\build-chat-context.ps1` или тот же скрипт с `-RebuildContext`.
- **Почему не «авто из кода»:** один репозиторий не знает целиком боевой `.env` на VPS и намерения человека; автоподстановка из diff дала бы шум. Надёжнее **событие в логе + недельный осознанный срез**.

## Оперативный фокус (актуально, 2026-09-04)
- **GitHub (public):** репозиторий **`moex-muhlya`** — исходники открыты для улучшений сообществом; боевые секреты и IP VPS **не** в git (IP в доках → плейсхолдер `YOUR_VPS_HOST`).
- **Безопасность публикации:** `.env` локально/на VPS; бинарники `muhlya-headless*` в `.gitignore`; не коммитить токены Finam / пароли Postgres.
- **VPS / торговля:** без смены боевого профиля относительно 2026-04-24 (низкий кэш / исполнимость long); контроль по `runtime_events` на активной сессии.
- **Фьючерсы / allowlist / Мия:** как в архиве 2026-04-24 (`validate-futures`, ручной futures allowlist, `MIYA_WATCH_*`).

## Архив: оперативный фокус (2026-04-24)
Снимок до публикации на GitHub (2026-09-04) — сверка с `.env` на VPS и `SESSION_LOG`.

- **VPS / низкий кэш и входы:** в боевом `.env` ослаблен пакет исполнимости long по акциям: `AGENT_MIN_STOCK_NOTIONAL_RUB=1000` (было 1500), `AGENT_MIN_TURNOVER_RUB=120000` (было 200000), `AGENT_BUY_CANDIDATES_LIMIT=30` (было 20, max в коде 30). Цель — реже `skip_buy_no_affordable_after_scan` при ~9k+ ₽; **контроль на активной сессии:** `auto.order_buy_try`, `order.*`, `reject`-профиль, не раздувать шум. Комментарий-профиль в `moex-trader-gui/.env.example` — см. `SESSION_LOG`.
- **Фьючерсы / allowlist:** по-прежнему `moex-trader-gui/cmd/validate-futures`, ручной `lchi_futures_allowlist.txt`, на VPS `AGENT_FUTURES_ALLOWLIST_AUTO_LEARN=0`; тикер без `GetAsset` ok — не в allowlist.
- **Мия online:** `MIYA_WATCH_*` на VPS; systemd override для `ONLINE_MIA_FUTURES_WATCH_TICKERS` при необходимости.
- **Правда по портфелю:** `broker_positions_snapshot.json` + `miya-report`.
- **Репозиторий:** `docs/VPS_SSH_LINUX_BASICS.md`, `docs/MIYA_MUHLYA_ROLES.md`, `moex-trader-gui/.env.example`.

## Архив: оперативный фокус (2026-04-23)
Снимок до смены фокуса на **2026-04-24** — сверка с `SESSION_LOG` и `.env` на VPS.

- **Фьючерсы / allowlist:** утилита `moex-trader-gui/cmd/validate-futures` и `moex-trader-gui/scripts/vps/validate_futures.sh` — те же `GetAsset`/кандидаты символа, что у Мухли; `config/lchi_futures_allowlist.txt` вручную только из **ok**-тикеров; на VPS `AGENT_FUTURES_ALLOWLIST_AUTO_LEARN=0` (см. `SESSION_LOG`). Тикер вроде `GZM6` может дать `NotFound` у Finam — не класть в allowlist без прогона.
- **VPS / Мухля:** тюнинг под низкий баланс (mixed, пороги, denylist `MSTT,PMSB,ASSB` и др.), ранжирование по expected-profit, `registerBrokerValidationFailure` на `futures_finam_getasset_reject` (анти-спам), свежий `muhlya-headless` — детали в `SESSION_LOG`.
- **Мия online:** в `miya-online-watch` — `MIYA_WATCH_STOCK_TICKERS` / `MIYA_WATCH_FUTURES_TICKERS` и `MIYA_WATCH_TICKERS`; фьючерсный watch на VPS согласовывать с валидностью у брокера.
- **Правда по портфелю в отчётах:** без изменений — `broker_positions_snapshot.json` + порог свежести в `miya-report`.
- **Наблюдение в активную сессию:** эффект — по `runtime_events`; при странностях — срез `vps-slice-4h.sh` / Postgres.
- **Репозиторий:** актуальны `docs/VPS_SSH_LINUX_BASICS.md`, `docs/MIYA_MUHLYA_ROLES.md`, `moex-trader-gui/.env.example`.

## Архив: оперативный фокус (2026-04-20)
Снимок до смены фокуса на 2026-04-23 — **сверка с `SESSION_LOG` и `.env` на VPS**.

- **VPS / Мухля:** задеплоен свежий `muhlya-headless` (обучение + демпф intraday Мии вне сессии акций); в боевой `.env` добавлены `AGENT_EXIT_LEARNING_MIN_SAMPLES`, `AGENT_LEARNING_MIN_TRADES_PER_TICKER`, `AGENT_PNL_SCORE_MIN_TRADES`, `AGENT_MIA_ONLINE_STOCK_CLOSED_INTRADAY_SCALE` — см. `SESSION_LOG`.
- **Мия online:** скрипт без искусственного окна 10:00–18:40 (`ACTIVE_SESSION` убран); таймер по-прежнему раз в минуту; fallback свечей — только `ONLINE_MIA_CANDLE_USE_FALLBACK_IN_ACTIVE` + `…_FALLBACK_MIN`.
- **Правда по портфелю в отчётах:** без изменений — `broker_positions_snapshot.json` + порог свежести в `miya-report`.
- **Наблюдение в активную сессию:** эффект правок — по `runtime_events` (гейты, `auto.miya_online_hint`, входы); при странностях — срез `vps-slice-4h.sh` / Postgres.
- **Репозиторий:** актуальны `docs/VPS_SSH_LINUX_BASICS.md`, `docs/MIYA_MUHLYA_ROLES.md`, `moex-trader-gui/.env.example`.

## Архив: оперативный фокус (2026-04-18)
Снимок «что было важно» до деплоя обучения и чистки Мии — **сверка с `.env` на VPS и журналом `SESSION_LOG`**.

- **Правда по портфелю в отчётах:** Мухля пишет `/var/log/muhlya-reports/broker_positions_snapshot.json` (Finam); Мия в `miya-report` использует его, если файл **свежий** (`ONLINE_MIA_BROKER_SNAPSHOT_STALE_SEC`), иначе — Postgres. При расхождении — рекомендация в отчёте.
- **Мия online:** приоритет **MOEX ISS** (`api_first`), fallback на `market_data`; таймеры `miya-online-watch` / `miya-report` на VPS — рабочий контур.
- **Мухля:** risk-exit не блокируется `daily notional`; опционально **advisory/apply** автоподстройки `AGENT_MAX_DAILY_NOTIONAL` (`AGENT_DAILY_NOTIONAL_TUNE_*`). Параметры VPS — по факту в `.env` (см. `SESSION_LOG`).
- **Наблюдение в активную сессию:** эффект open-bias / входов — по `runtime_events` и intraday Мии; не раздувать лимиты без среза метрик.
- **Репозиторий:** корневой `README.md`, актуальный `moex-trader-gui/README.md`, без `tmp_*.sql` в корне (игнор в `.gitignore`).

## Архив: оперативный фокус (после 2026-04-15)
Снимок настроек и задач на тот момент — **не удалять**: при расследовании «что уже включали на VPS» и при сравнении с текущим `.env` удобнее смотреть сюда, чем копаться в истории чата. Актуальный короткий фокус — в блоке **«Оперативный фокус (актуально, 2026-04-23)»** выше.

- **Открытие рынка (open-фаза):** на VPS включён мягкий long-bias Мухли (`AGENT_OPEN_LONG_BIAS=1`, `AGENT_OPEN_BUY_GATE_DELTA=0.04`, `AGENT_OPEN_SHORT_GATE_PENALTY=0.15`, `AGENT_OPEN_MAX_BUY_ATTEMPTS_BONUS=3`).
- **Open only stocks:** в фазе `open` включён предфильтр по акциям (`AGENT_OPEN_STOCKS_ONLY=1`) как безопасный universe-слой.
- **Short-ветка:** до прохождения брокерной квалификации держим в щадящем режиме (минимум шумных попыток); основной фокус сейчас на надёжном исполнении `risk-exit` для long.
- **Залипшие pending-buy:** включён bypass `AGENT_PENDING_GLOBAL_BLOCK_MAX_MIN=1` (не блокировать весь buy-поток бесконечно).
- **Мия:** `miya-report.service` восстановлен после `203/EXEC`; отчёты снова пишутся по timer.
- **Срез на ближайшую активную сессию:** отдельно собрать метрики `auto.order_buy_try/open`, `order.buy_placed/open`, `auto.no_order/open`, чтобы подтвердить эффект open-bias.
- **Мия / онлайн-свечи:** контур `miya-online-watch` включён на VPS (минутный таймер, latest snapshot без архива), есть fallback (`DATA_STATUS=fallback_active`) для разреженного потока свечей.
- **Связка «Мия -> Мухля»:** online-signal мягко влияет на buy/short gates (`AGENT_MIA_ONLINE_*`), без жёсткого принуждения к сделке.
- **Risk-exit защита:** daily-notional лимит больше не должен блокировать закрывающие сделки (SL/выходы); контроль после фикса — без `order.sell_failed` по `daily notional`.
- **Дневной лимит риска:** внедрена автоподстройка `AGENT_MAX_DAILY_NOTIONAL` на границе суток (по умолчанию advisory, без автоприменения) для подбора рабочего коридора без ручной «магии».

## Вводные от коллег (Миша, Маша) — как применить к проекту
Оба сошлись: **офлайн-обучение и валидация в `trading-lab`**, **детерминированный Go-агент + жёсткий риск в проде**; **LLM/RAG только как proposer** с human-in-the-loop, без автоторговли по сырому выводу модели.

| Направление | Смысл для нас | Черновой шаг |
|-------------|----------------|--------------|
| **Вариант 1 (Миша): JSON-пайплайн** | Уже есть: PPO → `export_policy.py` → `AGENT_POLICY_PATH`. | Расширить JSON (пороги RSI, бюджеты, опционально таблицы/секции); добавить в лаборатории **бэктест-валидатор** и артефакт «политика прошла проверку» перед ручным копированием в прод. |
| **Оптимизация гиперпараметров** | Optuna/GridSearch по метрикам с риском (не только PnL). | Скрипты в `trading-lab`, отчёт метрик рядом с `policy.json`. |
| **Вариант 2 (Миша): RL → кандидаты** | Несколько чекпоинтов → несколько JSON → сравнение на бумаге. | Сначала вручную переключать `AGENT_POLICY_PATH`; позже — UI/флаги «кандидат / активная» и paper-only кандидаты. |
| **Вариант 3 / Маша: LLM** | Только офлайн: PDF → RAG → черновик JSON или отчёт по журналу. | Отдельный процесс (Python), выход — файл для ревью; в Go **не** звонить в LLM в hot path. |
| **Защита (Миша): хеш политики, аудит** | Понимать, какая политика в бою. | Опционально: логировать путь/хеш JSON при старте; связка с `validation.passed` из лаборатории — по согласованию. |

**Практический порядок:** укрепить **бэктест + пороги метрик** для новой политики → расширить экспорт JSON → (позже) LLM как советчик → изолированный **sandbox/paper** для кандидатов уже частично есть (`paper` в GUI).

**Сделано по этой линии (лаборатория):**
- `trading-lab/validate_policy.py` — бэктест RSI-политики, train/test split, `validation_report.json` + `validation.passed` при успехе порогов; флаг `--strict`.
- `trading-lab/rsi_backtest.py`, `policy_defaults.py` — согласование дефолтов с Go.
- `export_policy.py` — экспорт расширен полями под `AgentPolicy` (RSI, бюджеты, min notional).

## Что уже сделано
- Убраны секреты из репозитория:
  - удален `moex-trader-gui/.env` из git-версии
  - добавлен шаблон `moex-trader-gui/.env.example` без токенов и паролей
- Конфиг БД вынесен в env-переменные в GUI/CLI/desktop.
- Исправлен `go.mod` в `moex-trader` (версия Go и tidy).
- Добавлено общее ядро `trading-core` (типы/интерфейсы/риск-менеджер).
- `FinamBroker` переведен с заглушек на реальные REST-вызовы:
  - JWT (`/v1/sessions`)
  - details/accounts (`/v1/sessions/details`)
  - place/get/cancel orders
  - portfolio (`/v1/accounts/{id}`)
- Улучшен WebSocket:
  - формат подписки `symbols: []`
  - reconnect с backoff
  - восстановление подписок после reconnect
  - убрана автоподписка на `SBER`
- Добавлен risk-manager:
  - лимит на заявку
  - дневной лимит
  - emergency stop
- В GUI добавлены режимы:
  - `manual`, `auto`, `paper`
  - старт/стоп автоторговли
- AI-панель обновлена:
  - новое имя: "Торговый ассистент MOEX"
  - убран ручной тикер-инпут
  - добавлен поиск и выбор инструмента из списка
- Убраны хардкод-котировки в watchlist:
  - акции и фьючерсы теперь загружаются динамически из MOEX ISS
  - добавлены backend-методы поиска инструментов по строке (`GetStockInstruments`, `GetFuturesInstruments`)
  - в GUI добавлен поиск по акциям и фьючерсам
- Проверен локальный PostgreSQL:
  - контейнер `moex_trading_db` работает (healthy)
  - БД `moex_trading` содержит рабочие таблицы (`market_data`, `positions`, `trades`, `watchlist`)

## Сделано в последних сессиях (2026-04-05)
- Локально: **PostgreSQL 18** переустановлен с полным набором (в т.ч. **`pg_ctl`**), служба **`postgresql-x64-18`** работает; разделены роли **Docker Postgres** (`5433`, `moex_trading`) и **нативного** (`5432`); для GUI/pgAdmin — подключаться к нужному порту; договорённость: основной dev-контур на **Docker**, нативный — по желанию.

## Сделано в последних сессиях (2026-04-08)
- **ЛЧИ-2026:** подтверждено участие; согласован подход по инструментам: акции — по whitelist ЛЧИ, фьючерсы/опционы — отдельный управляемый контур.
- **Акции ЛЧИ:** добавлен `config/lchi_stocks_allowlist.txt` (144 тикера из пользовательского файла `instrumenty-lchi-2026.xlsx`) и фильтрация авто-universe по акциям.
- **Фьючерсы:** добавлен `config/lchi_futures_allowlist.txt` + поддержка `AGENT_FUTURES_ALLOWLIST_PATH`; внедрён авто-режим `AGENT_FUTURES_ALLOWLIST_AUTO_LEARN` (добавляет валидные futures и удаляет `NotFound`).
- **Headless/env:** `AGENT_AUTO_INSTRUMENT_MODE` теперь читается при старте headless; добавлены env-параметры для риск-лимитов (`AGENT_MAX_ORDER_NOTIONAL`, `AGENT_MAX_DAILY_NOTIONAL`) и denylist (`AGENT_TICKER_DENYLIST`).
- **Качество данных сделок:** исправлено логирование `executed_at` — для `rejected` теперь `NULL`, что убрало искажения аналитики.
- **VPS деплой:** исправлен инцидент `203/EXEC` (Windows PE -> Linux ELF), бинарник обновлён, `muhlya.service` стабильно в `active/running`.
- **Операционный контур:** `docs/VPS_SSH_LINUX_BASICS.md` дополнен быстрым входом SSH, 20-мин срезами и мини-чеклистом дежурного.
- **Ночной безопасный режим (на конец сессии):** `AGENT_AUTO_INSTRUMENT_MODE=stocks`, `AGENT_FINAM_FUTURES_SYMBOL_INDEX=0`, `AGENT_FUTURES_ALLOWLIST_AUTO_LEARN=0` для снижения reject-шума по фьючерсам вне активной настройки.

## Сделано в последних сессиях (2026-04-04)
- GUI / конфиг авто: сброс **фиксированного тикера** при переключении режима выбора инструмента с `fixed` на `auto` / акции / фьючерсы / опционы — согласованность `Dashboard.vue` и `SetAutoInstrumentConfig` (`app.go`), без «хвоста» тикера в `GetAutoInstrumentConfig`.
- Уточнения по продукту (из обсуждения): Watchlist строится из **MOEX ISS** (TQBR/FORTS) + тикеры из портфеля/недавних сделок; вкладка не драйвит universe Мухли — universe задаётся автоциклом отдельно. На VPS в выходной в `journalctl` типичен heartbeat **баланса/JWT**; полнота картины по Мухле в **торговую сессию** — по **`runtime_events`** в БД.
- Документация: [`docs/VPS_SSH_LINUX_BASICS.md`](docs/VPS_SSH_LINUX_BASICS.md) — PowerShell → SSH → Linux, команды под наш VDS (`traderbot@YOUR_VPS_HOST`), пути деплоя, `systemd`/`docker`, ссылка на поддержку Timeweb.

## Сделано в последних сессиях (2026-04-03)
- Бэкап PostgreSQL для VPS: скрипты `moex-trader-gui/scripts/vps/pg-backup.sh` и `pg-smoke-restore.sh`, unit/timer `scripts/vps/systemd/` (ежедневный `pg_dump`, ротация, smoke-restore в отдельную БД). На сервере проверено: доставка без `git` (`scp`), `chown` на каталог бэкапов, `systemctl enable --now moex-pg-backup.timer`. Инструкция включения — в разделе «Что делаем следующим шагом» (п. 2) и `SESSION_LOG.md`.
- GUI: пресеты лимитов Мухли **Стандарт** / **Эксперимент** (`Dashboard.vue`) — быстрое переключение TP/SL/Profit Lock для локальных экспериментов без обязательного деплоя на VPS.
- Finam ↔ MOEX тикеры: `OrderSymbolForAPI` — ручные переопределения (`FINAM_SYMBOL_OVERRIDES_PATH`), авто-индекс фьючерсов Finam (gRPC `AllAssets`, MIC=FUT, `AGENT_FINAM_FUTURES_SYMBOL_INDEX`), фолбэк `TICKER@FUT`; перед заявкой — `GetAsset` (откл. `AGENT_FINAM_GETASSET_PREFLIGHT=0`). Справочник в ЛК: https://ai.finam.ru/futures

## Сделано в последних сессиях (2026-04-02)
- Runtime-лента вынесена из памяти в Postgres:
  - добавлена таблица `runtime_events` с автосхемой;
  - каждое `addRuntimeEvent` теперь сохраняется и в БД;
  - добавлен метод `GetRuntimeEventsPersistent(limit)`.
- Добавлен контроль хвоста runtime-событий:
  - лимит хранения в БД `AGENT_RUNTIME_EVENTS_DB_LIMIT` (дефолт `500`, диапазон `50..50_000`);
  - создан утилитарный скрипт `scripts/get-runtime-events.ps1` для выборки последних событий.
- UI ленты шагов обновлён:
  - переключатель источника `Postgres / RAM`;
  - выбор лимита отображения (`10/50/120`) + ручной refresh.
- Анти-зацикливание и мягкая разгрузка входов:
  - короткий cooldown после `auto.skip_mtf_buy/sell`;
  - мягкое ослабление MTF только для buy через `AGENT_MTF_BUY_RELAX_DELTA` (по умолчанию `1`);
  - авто-поднятие бюджета на шаг при серии `blocked_by_cash` (`auto.budget_autobump`).
- Надёжность исполнения ордеров:
  - перед sell добавлен clamp по доступному long-объёму (`order.sell_qty_clamped`) для снижения ошибок `[157] Equities shortage`;
  - расширен пул buy-кандидатов (`AGENT_BUY_CANDIDATES_LIMIT`, дефолт `12`, диапазон `4..30`) для поиска более доступных альтернатив.
- Low-balance контур переработан:
  - раздельные пороги `AGENT_MIN_BALANCE_STOCK` и `AGENT_MIN_BALANCE_FUTURES`;
  - при низком балансе включается режим `auto.low_balance_manage_only` (новые входы паузятся, управление открытыми позициями сохраняется).
- Журнал сделок и дашборд:
  - в журнал добавлена колонка `P/L` по сделкам, закрывающим позицию;
  - расчёт `P/L` исправлен: учитываются только фактически исполненные сделки (`executed`/Finam trades), `rejected/placed` не ломают state;
  - верхний блок статистики: добавлена карточка `Instruments Value` = `Portfolio Value - Cash Balance`.
- Исполнение и анти-дубли:
  - добавлен pending-order gate перед `sell` (`auto.skip_pending_order`) — дублирующие заявки не отправляются, пока есть активная заявка у брокера;
  - для shortage `[157]` добавлен retry с уменьшением qty и мягкая пауза по тикеру;
  - добавлен отдельный блок-сигнал `order.blocked_shortage`.
- Фьючерсы:
  - добавлен preflight валидности фьючерсного тикера перед отправкой (`order.futures_preflight_reject`) для снижения `404 Security id doesn't exist`.
- Ранжирование:
  - добавлен мягкий PnL-вес тикера (`AGENT_PNL_SCORE_WEIGHT`);
  - добавлен affordability-first score для buy-кандидатов (`AGENT_AFFORDABILITY_SCORE_WEIGHT`, `AGENT_AFFORDABILITY_MIN_EXEC_RATIO`) без поломки ротации/закрытий.
- Параметры и UX:
  - `.env.example` дополнен консервативным/активным профилями;
  - в `Portfolio` добавлен переключатель скрытия нулевых позиций (по умолчанию нули скрыты).

## Текущее рабочее состояние
- Проекты собираются:
  - `moex-trader-gui`
  - `moex-trader`
  - `moex-trader-desktop`
  - `trading-core`
- Для боевого режима нужен локальный `.env` в `moex-trader-gui/`:
  - `FINAM_SECRET_REAL=<REAL_SECRET>`
  - `PGHOST`, `PGPORT`, `PGUSER`, `PGPASSWORD`, `PGDATABASE`

## Критично по безопасности
- Токен, который ранее попадал в файлы, считать скомпрометированным.
- Обязательно перевыпустить secret в кабинете Finam.

## Что делаем следующим шагом
### План миграции Мухли на VPS (24/7 при открытом рынке)
1. Что купить у провайдера:
   - ОС: `Ubuntu 22.04/24.04 LTS`;
   - тариф: `2 vCPU / 4 GB RAM / 60+ GB NVMe`;
   - сеть: `100 Mbps+`;
   - опции: `IPv4`, `snapshot/backup`;
   - регион: ближе к Москве/СПб.
2. Сразу после создания VPS (безопасность):
   - `apt update && apt -y upgrade`
   - `timedatectl set-timezone Europe/Moscow`
   - `adduser traderbot`
   - `usermod -aG sudo traderbot`
   - дальнейшая работа под `traderbot`, не под `root`.
3. SSH-ключи и запрет пароля:
   - добавить публичный ключ на сервер;
   - в `/etc/ssh/sshd_config`: `PasswordAuthentication no`, `PermitRootLogin no`;
   - перезапуск: `systemctl restart ssh`.
4. Базовый firewall:
   - `ufw allow OpenSSH`
   - `ufw enable`
   - `ufw status`
   - порты приложения открывать только при реальной необходимости.
5. Установка Docker + Compose:
   - `apt -y install ca-certificates curl gnupg`
   - `install -m 0755 -d /etc/apt/keyrings`
   - `curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg`
   - `chmod a+r /etc/apt/keyrings/docker.gpg`
   - `echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo $VERSION_CODENAME) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null`
   - `apt update`
   - `apt -y install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin`
   - `usermod -aG docker traderbot`
6. Деплой проекта:
   - `mkdir -p /opt/trading && cd /opt/trading`
   - `git clone` репозитория;
   - создать серверный `.env` (не в git): Finam secret/JWT логика, `PGUSER`, `PGPASSWORD`, `PGDATABASE`, все `AGENT_*` параметры.
7. Поднять стек:
   - `docker compose up -d`
   - `docker compose ps`
   - `docker logs --tail=100 -f <app_container_name>`
8. Автостарт и самовосстановление:
   - в `docker-compose.yml` для сервисов: `restart: unless-stopped`;
   - `healthcheck` для app и postgres.
9. Мониторинг:
   - `runtime_events` в БД;
   - алерты в Telegram на `order.*_failed`, потерю связи с API, остановку автоцикла.
10. Ночной режим до отдельной кнопки:
   - правило перед офлайном: уменьшить риск/не открывать новые позиции;
   - затем внедрить кнопку `Уйти в ночь` (пауза новых входов + план по открытым позициям).

Мини-вывод:
- Берём VPS `2/4/60`, ставим Docker, переносим `.env`, поднимаем compose, включаем мониторинг.
- Цель: Мухля торгует, даже если локальный ПК выключен.

### Команды дежурного (VPS)
- Базовая шпаргалка: **PowerShell → SSH → Linux** (навигация, просмотр файлов, наш стек): [`docs/VPS_SSH_LINUX_BASICS.md`](docs/VPS_SSH_LINUX_BASICS.md)
- Подключение к серверу: `ssh traderbot@YOUR_VPS_HOST`
- Статус сервиса Мухли: `sudo systemctl status muhlya --no-pager`
- Логи Мухли (хвост): `sudo journalctl -u muhlya -f`
- Перезапуск Мухли: `sudo systemctl restart muhlya`
- Статус БД-контейнера: `docker compose -f /opt/trading/app/moex-trader-gui/docker-compose.yml ps`
- Проверка таблиц БД: `docker exec -it moex_trading_db psql -U trader -d moex_trading -c "\dt"`
- Проверка runtime-событий: `docker exec -it moex_trading_db psql -U trader -d moex_trading -c "SELECT to_char(at AT TIME ZONE 'Asia/Novosibirsk','YYYY-MM-DD HH24:MI:SS') at_nsk, level, step, details FROM runtime_events ORDER BY created_at DESC LIMIT 20;"`
- Бэкап PostgreSQL (после включения timer): `sudo systemctl status moex-pg-backup.timer --no-pager`; последний запуск: `sudo journalctl -u moex-pg-backup.service -n 30 --no-pager`; файлы: `ls -la /var/backups/moex-trading-pg/`; ручной прогон: `sudo systemctl start moex-pg-backup.service`.

1. Подтвердить снижение технических отказов после execution-first правок:
   - падение доли `order.sell_failed [157]` после pending-order gate и shortage-retry;
   - падение `404 unknown security` по фьючерсам после preflight.
2. ~~Ежедневный бэкап PostgreSQL на VPS~~ — **артефакты в репо** (на сервере включить timer один раз):
   - `moex-trader-gui/scripts/vps/pg-backup.sh` — `pg_dump -Fc`, каталог по умолчанию `/var/backups/moex-trading-pg`, ротация по `PG_BACKUP_RETENTION_DAYS` (дефолт 14);
   - `moex-trader-gui/scripts/vps/pg-smoke-restore.sh` — восстановление в БД `moex_trading_restore_smoke` и удаление (проверка целостности дампа);
   - `moex-trader-gui/scripts/vps/systemd/moex-pg-backup.service` + `moex-pg-backup.timer` (03:30 локального времени сервера, `SupplementaryGroups=docker` для `traderbot`).
   - Установка на VPS: `sudo mkdir -p /var/backups/moex-trading-pg && sudo chown traderbot:traderbot /var/backups/moex-trading-pg`; скопировать unit/timer в `/etc/systemd/system/`, поправить `WorkingDirectory`/`ExecStart` если путь не `/opt/trading/app/moex-trader-gui`; `chmod +x scripts/vps/*.sh`; `systemctl daemon-reload && systemctl enable --now moex-pg-backup.timer`.
3. Подключить Telegram-алерты по критичным событиям:
   - сервис Мухли: `up/down/restart`;
   - runtime-события ошибок ордеров (`order.*_failed`);
   - обрывы API/долгие паузы автоцикла.
4. Проверить результат affordability-first ранжирования:
   - сравнить `skip_buy_affordable` до/после;
   - оценить рост `auto.order_buy_try` и долю `placed/executed`.
5. Проверить результат low-balance контура на живом потоке:
   - при низком кэше фиксировать `auto.low_balance_manage_only` и отсутствие лишних входов;
   - убедиться, что выходы по открытым позициям не блокируются из-за низкого баланса.
6. Подтвердить стабильность нового режима запуска:
   - приложение стартует в `auto` при доступном брокере;
   - в клиринг/вне сессии статус остаётся `AUTO (ожидание сессии)`, а не `manual`;
   - после возобновления торгов Мухля возвращается к попыткам входа без ручного re-enable.
7. Подтвердить в живой сессии MOEX новые контуры управления позицией:
   - `Profit Lock` (`auto.profit_lock_armed` / `auto.profit_lock_exit`);
   - ротацию капитала (`auto.rotate_long_to_long`, `auto.rotate_long_to_short`, `auto.rotate_short_to_long`);
   - проверить, что выходы/ротации происходят без “пилы” и без лишней переторговки.
8. Подтвердить работу обновленного anti-reject контура:
   - проконтролировать `blocked_by_qualification`, `blocked_by_rotation`, `blocked_by_cash`;
   - проверить, что повторов `Invalid quantity` / `[257]` стало заметно меньше.
9. Отладить точки выхода Мухли по уже открытым позициям:
   - собрать runtime-события по TP/SL и `auto.exit_*` в первой активной торговой сессии;
   - проверить, почему для части позиций не формируется sell-решение в нужный момент.
10. Подкрутить MTF-пороги после 1-2 живых сессий:
   - оценить долю `skip_mtf_buy` / `skip_mtf_sell`;
   - учесть новые события `auto.adaptive_thresholds` и сравнить долю `no_order` до/после;
   - скорректировать пороги подтверждения, чтобы не душить сделки излишней консервативностью.
11. Продолжить улучшение ранжирования тикеров:
   - использовать reject-rate (`24h/7d`) + winrate в едином score;
   - добавить явный penalize/blacklist для системно "токсичных" тикеров.
12. Вывести в UI параметры ротации (по аналогии с TP/SL/Profit Lock):
   - `Rotate enabled`, `Min delta`, `Rotate cooldown min`;
   - сохранить через backend методы безопасности/автонастроек.
13. Прогоняем smoke-check боевого контура (микро-объёмы):
   - auth -> accounts -> ws quotes -> tiny limit order -> cancel;
   - сверить, что статус в журнале = `placed/executed`, а не только `rejected`.
14. Phase 2 (после стабилизации акций/фьючерсов):
   - модуль опционов: каталог инструментов, параметры, стратегия и риски.
15. ЛЧИ universe manager (следующий этап):
   - добавить в продуктовый план отдельные контуры для **БПИФ** и **облигаций ЛЧИ** (параметры входа/ликвидности/риска отдельно от акций);
   - для фьючерсов закрепить «чистый» allowlist-контур (автообновление + ручной контроль перед боем), чтобы исключить массовые `getasset miss`.
16. Мия (файловая дежурка на VPS, без Telegram/LLM):
   - включить и поддерживать timer `miya-report.timer`;
   - отчёты хранить в `/var/log/muhlya-reports` (`intraday_*.md/json`, `daily_YYYY-MM-DD.md`);
   - в отчётах — разрез по **фазам сессии** (open/mid/close/offhours) и A/B/C, если в логах есть `auto.session_tick` / `auto.abtest_cycle` (нужен свежий `muhlya-headless`);
   - ручная сводка по фазам: `moex-trader-gui/scripts/vps/phase_summary.sh [часов]`;
   - вечером разбирать `daily_*.md` + последние `intraday_*` вместе с runtime/trades срезами;
   - при необходимости изменить cadence через env в unit (`ONLINE_MIA_ACTIVE_EVERY_MIN`, `ONLINE_MIA_OFFHOURS_EVERY_MIN`, `ONLINE_MIA_OPENING_EVERY_MIN`, `ONLINE_MIA_OPENING_WINDOW_MIN`).
17. Стабилизация исполнения после «оживления входов»:
   - анти-`[156]` уже внедрён (cash-pause + cooldown), но зафиксировать итог по 1-2 активным сессиям;
   - по `{8}` добавлен цикловой limiter `AGENT_MAX_BUY_ATTEMPTS_PER_CYCLE` (снижение спайков `order_buy_try`); проверить эффект на живой сессии;
   - оставить включённым динамический тикерный gate (`AGENT_STRICT_TRADINGSTATUS_GATE=1`, `AGENT_TRADINGSTATUS_TTL_SEC=30`) и сравнить ночной шум до/после;
   - добить остаточные `INVALID_ARGUMENT: {8} Invalid quantity value` и `[157] Equities shortage` (лотность/доступный объём/точка выхода);
   - держать под контролем `auto.skip_pending_order` и `auto.order_buy_try`, чтобы не скатываться в спам-цикл;
   - после стабилизации вернуть часть фильтров (`trend/macd`) к базовому профилю и сравнить PnL по фазам.
18. Пара «Мухля + Мия» (автоподстройка выхода):
   - `miya-report.sh` пишет `/var/log/muhlya-reports/agent_exit_hints.env` (DEFAULT + тикерные множители);
   - Мухля читает hints через `AGENT_MIA_EXIT_HINTS_PATH` и совмещает с локальным `AGENT_EXIT_LEARNING_PATH`;
   - контролировать появление `auto.exit_tp_tuned` и файла `data/exit_learning.json` на живых закрытиях.

### Исследования индикаторов (апрель 2026)
- Полезные внешние материалы:
  - [PipTrend: Top 7 Best Indicators for Trading in 2026](https://piptrend.com/best-indicators-for-trading/)
  - [WunderTrading: Алготрейдинг в 2026 — лучшие индикаторы](https://wundertrading.com/journal/ru/trading-bots/statja/navigating-the-latest-algorithms-for-success)
  - [VibeTrader: Best Technical Indicators for Algorithmic Trading in 2026](https://vibetrader.markets/blog/best-indicators-for-algorithmic-trading.html)
- Консенсус по источникам: не перегружать стратегию; базовая рабочая связка = `trend + momentum + volatility/risk`.
- Практический профиль для Мухли (для следующей живой сессии):
  - тренд: `EMA(20/50)` или фильтр "цена выше/ниже долгой MA";
  - тайминг: `RSI(14)` + `MACD` (подтверждение импульса);
  - волатильность/риск: `ATR(14)` для адаптивного стопа и фильтра "не входить в экстремальную волатильность";
  - подтверждение качества входа: объём/VWAP как дополнительный gate.
- Зеркальность long/short:
  - одинаковый confidence gate для long-entry и short-open;
  - одинаковые TP/SL в процентах (знак PnL зависит от направления позиции);
  - одинаковый MTF-подход для `buy` и `sell`.
- Валидация "торгуем в плюс":
  - считать PnL отдельно по long и short;
  - сравнивать `placed/executed/rejected` по направлениям;
  - менять за итерацию только 1 параметр (иначе нельзя понять причину улучшения/ухудшения).

## Быстрый старт команд
- GUI dev:
  - `cd moex-trader-gui`
  - `wails dev`
- CLI:
  - `cd moex-trader`
  - `go run ./cmd`

---
## PowerShell быстрые команды (Windows -> VPS)
- Перейти в папку проекта:
  - `cd "c:\Users\MyProgram\Trading\moex-trader-gui"`
- Загрузить новый `muhlya-headless` на VPS:
  - `scp "c:\Users\MyProgram\Trading\muhlya-headless" traderbot@YOUR_VPS_HOST:/tmp/muhlya-headless.new`
- Команды на сервере (остановить/заменить/запустить/проверить логи):
  - `ssh traderbot@YOUR_VPS_HOST "sudo systemctl stop muhlya; sudo install -m 0755 /tmp/muhlya-headless.new /opt/trading/app/moex-trader-gui/muhlya-headless; sudo systemctl start muhlya; sudo journalctl -u muhlya -n 80 --no-pager"`
- Мия (проверка таймера и отчётов):
  - `ssh traderbot@YOUR_VPS_HOST "systemctl status miya-report.timer --no-pager; sudo journalctl -u miya-report.service -n 50 --no-pager; ls -la /var/log/muhlya-reports | tail -n 20"`

