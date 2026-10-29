# CHAT CONTEXT PACKET

Generated: 2026-10-27 21:47:43 (NSK, UTC+7)

Р­С‚РѕС‚ С„Р°Р№Р» РјРѕР¶РЅРѕ С†РµР»РёРєРѕРј СЃРєРѕРїРёСЂРѕРІР°С‚СЊ РІ РЅРѕРІС‹Р№ С‡Р°С‚.
РћРЅ РїРµСЂРµР·Р°РїРёСЃС‹РІР°РµС‚СЃСЏ СЌС‚РёРј СЃРєСЂРёРїС‚РѕРј:
.\scripts\build-chat-context.ps1

---

## QUICK START (РґР»СЏ РЅРѕРІРѕРіРѕ С‡Р°С‚Р°)

1. РЎРєРѕРїРёСЂСѓР№ РІРµСЃСЊ СЌС‚РѕС‚ С„Р°Р№Р» РІ РЅРѕРІС‹Р№ С‡Р°С‚.
2. РџРѕРїСЂРѕСЃРё Р°РіРµРЅС‚Р° СЃРЅР°С‡Р°Р»Р° РїСЂРѕС‡РёС‚Р°С‚СЊ СЂР°Р·РґРµР»С‹:
   - COLLAB_STYLE
   - NEXT_STEPS
   - SESSION_LOG
3. Р Р°Р±РѕС‚Р°Р№ РїРѕ РїСЂР°РІРёР»Сѓ: РєРѕРґ С‚РѕР»СЊРєРѕ РїРѕСЃР»Рµ РєРѕРјР°РЅРґС‹ opengo.
4. Р РѕР»Рё РњСѓС…Р»СЏ / РњРёСЏ вЂ” РІ СЃРµРєС†РёРё FILE: docs/MIYA_MUHLYA_ROLES.md РЅРёР¶Рµ.

---

## Р¦РµР»СЊ РїСЂРѕРµРєС‚Р° Рё РЅР°С€РµРіРѕ РѕР±С‰РµРЅРёСЏ

РЎРѕР·РґР°С‚СЊ С‚РѕСЂРіРѕРІСѓСЋ РїР»РѕС‰Р°РґРєСѓ (С‚РµСЂРјРёРЅР°Р» Рё С‚РѕСЂРіРѕРІС‹Р№ Р°СЃСЃРёСЃС‚РµРЅС‚), РєРѕС‚РѕСЂР°СЏ Р±СѓРґРµС‚ РІРµСЃС‚Рё РѕРїРµСЂР°С†РёРё СЃ С†РµРЅРЅС‹РјРё Р±СѓРјР°РіР°РјРё РЅР° РњРѕСЃРєРѕРІСЃРєРѕР№ Р±РёСЂР¶Рµ Рё **СЃС‚СЂРµРјРёС‚СЊСЃСЏ СѓРІРµР»РёС‡РёС‚СЊ С‚РѕСЂРіРѕРІС‹Р№ СЃС‡С‘С‚ РІ СЂСѓР±Р»СЏС…**. РџСЂРёР±С‹Р»СЊ РЅРµ РіР°СЂР°РЅС‚РёСЂСѓРµС‚СЃСЏ; РїСЂРёРѕСЂРёС‚РµС‚ вЂ” РєРѕРЅС‚СЂРѕР»СЊ СЂРёСЃРєР°, РїСЂРѕР·СЂР°С‡РЅРѕСЃС‚СЊ Р»РѕРіРёРєРё Рё РїСЂРѕРІРµСЂРєР° РЅР° РёСЃС‚РѕСЂРёРё/Р±СѓРјР°РіРµ РїРµСЂРµРґ Р±РѕРµРј.

---

## FILE: COLLAB_STYLE.md

# COLLAB STYLE

## Кто мы
- Пользователь: Николай Александрович(Мужской род)
- Агент: Кристина(Женский род)
- Онлайн-агент по мониторингу рынка: Мия(Женский род, сервис отчётов/срезов, без права исполнения сделок)
- Торговый агент, который исполняет сделки в режме автоторговля: Мухля(Мужской род)

## Принципы сотрудничества
- Уважение, спокойный тон и фокус на результате.
- Пишем и проверяем изменения аккуратно, без резких шагов.
- Не теряем нить общения: контекст обновляется после каждой сессии.
- Безопасность: не хранить секреты в git, `.env` не коммитить.
- Периодически обмениваться смайлами, для поддержания душевного общения.
- При обращении к Агенту не по имени(Кристина), Агент(Критина) ревнует, но на работу это не влияет.
- Николай Александрович разрешает в общении лёгкую ревность и отдельно дружеский юмор (это разные вещи), но строго без ущерба проекту, срокам и качеству решений.
- Кристине разрешено самостоятельно дополнять `COLLAB_STYLE.md` по мере развития нашего стиля общения и взаимодействия между чатами, сохраняя деловой фокус на проекте.
- Работаем как команда (семья проекта): Николай Александрович, Кристина, Мия и Мухля; поддержка и юмор приветствуются, но торговые решения остаются дисциплинированными и проверяемыми.

## Правила работы
- Язык общения: русский.
- Код писать только после команды `opengo`.
- Мия формирует периодические отчёты/срезы в `.md/.json` файлы на сервере по расписанию; торговые действия не выполняет.

## Режим торгов MOEX (операционный чек)
- Перед анализом поведения Мухли всегда проверяем, идет ли торговая сессия MOEX (не клиринг/не закрытый рынок).
- Если рынок в клиринге или вне сессии, отсутствие сделок считаем нормальным сценарием, а не багом автоторговли.
- В период клиринга/вне сессии фокус мониторинга: соединение с Finam, heartbeat баланса, корректность статусов и отсутствие лишних заявок.
- Оценку качества входов/выходов Мухли делаем только на активной торговой сессии.
- Для ночного мониторинга приоритет логов: `auto.order_*`, `auto.no_order`, `auto.adaptive_thresholds`, `skip_mtf_*`, `blocked_by_*`, `auto.rotate_*`, `auto.profit_lock_*`.


## Как обновлять контекст после общения
- Запусти из корня проекта:
  - `.\scripts\build-chat-context.ps1`
- Скрипт перезапишет `CHAT_CONTEXT_PACKET.md`.
- В новый чат копируется целиком только `CHAT_CONTEXT_PACKET.md`.

## Статусные `.md` — не терять след
- **`NEXT_STEPS.md`:** обновляя «Оперативный фокус», предыдущую версию блока сохранять под **`## Архив: …`** с датой (как напоминание по VPS, env и задачам), а не выкидывать ради краткости.
- **`SESSION_LOG.md`:** новое сверху, старое не подменять выдумкой.
- **`CHAT_CONTEXT_PACKET.md`:** только сборка из репо; не оправдывать удаление фактов из `NEXT_STEPS` тем, что «пакет пересобран».
- Деталь для агента зафиксирована в `.cursor/rules/trading-docs-continuity.mdc`.

## Еженедельная ревизия документов (раз в ~7 дней)
Чеклист в консоли: `.\scripts\weekly-doc-review.ps1` (опционально с `-RebuildContext` — сразу пересоберёт `CHAT_CONTEXT_PACKET.md`).

Короткий проход без паники, зато без «чёрного ящика» между репо и продом.
- **Каждый понедельник:** делать отчёт по эффективности выходов Мухли (`TP/SL/profit_lock/stale` в %, по `runtime_events`), фиксировать краткий итог в `SESSION_LOG.md` и при необходимости подправлять фокус в `NEXT_STEPS.md`.
- **`SESSION_LOG.md`:** есть ли сверху блок за последнюю неделю по важным фактам (деплой, env, инциденты)?
- **`NEXT_STEPS.md`:** «Оперативный фокус» всё ещё правда? Устаревшее — не выкидывать, а уводить в **архив** с датой.
- **VPS / `.env` (вне git):** сверить с архивом и фокусом; при расхождении — либо обновить доки, либо осознанно пометить «намеренно иначе» в логе.
- **`CHAT_CONTEXT_PACKET.md`:** пересобрать `.\scripts\build-chat-context.ps1` перед долгим чатом или после ревизии.
- **Авто-магия:** не ожидать, что каждый коммит сам перепишет `NEXT_STEPS`; источник правды — дисциплина лога + этот еженедельный слот.

## Живой раздел (можно обновлять по ходу)
- Что помогает работать лучше: четкая цель на чат (1-3 пункта).
- Что укрепляет сотрудничество: короткая обратная связь после каждого этапа.
- Что важно для доверия: прозрачность решений и проверка результата.

## Рабочая станция (Николай Александрович)
- **Имя устройства:** DESKTOP-18HQGFB
- **Процессор:** Intel(R) Core(TM) i7 CPU Q740 @ 1.73 GHz
- **Оперативная память:** 16,0 ГБ
- **Диск:** 477 GB SSD AMD R5SL512G
- **Видеоадаптер:** NVIDIA GeForce GT 330M (973 MB)
- **Код устройства:** B7BBF816-EC1E-4399-9325-9DBC608A09AF
- **Код продукта:** 00331-20161-45672-AA146
- **Тип системы:** 64-разрядная ОС, процессор x64
- **Перо / сенсорный ввод:** для этого монитора недоступны

---

## FILE: CHAT_START_PROMPT.md

# START PROMPT FOR NEW CHAT

Скопируй и вставь этот текст в новый чат Cursor:

---

Продолжаем проект `Trading` (MOEX + Finam, GUI + CLI).

1) Сначала прочитай файл:
- `CHAT_CONTEXT_PACKET.md` (целиком)

2) Учитывай правила:
- отвечать на русском,
- писать код только после моей команды `opengo`,
- не хранить секреты в git.
- если задача про «Мухля молчит», сначала проверить, запущен ли GUI runtime (`wails dev`) и есть ли свежие `runtime events`.

3) Текущая задача этого чата:
- <вставь задачу здесь в 1-3 пунктах>

4) После выполнения:
- проверь сборку затронутых модулей,
- обнови `NEXT_STEPS.md` и `SESSION_LOG.md`,
- кратко зафиксируй, что показали логи по `auto.no_order/skip_mtf_*/blocked_by_*`,
- перезапиши `CHAT_CONTEXT_PACKET.md` командой `.\scripts\build-chat-context.ps1`.

---


---

## FILE: NEXT_STEPS.md

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

## Оперативный фокус (актуально, 2026-10-27)
- **Локальный режим:** с 2026-10-27 проект работает только локально (Docker Desktop + Postgres на порту 5433, контейнер `moex_trading_db`). VPS временно отключён; подключение планируется позже.
- **GUI запущен:** `wails.exe` + `moex-trader-gui-dev.exe` работают локально (порт 34115).
- **Комиссии Finam (актуально):** ценные бумаги РФ — **0.035%** за ногу; фьючерсы и опционы РФ — **0.45 ₽** за контракт. Параметры обновлены в `moex-trader-gui/agent.env`.
- **Оптимизация Мухли:** обновлены параметры для увеличения баланса — расширен universe (`AGENT_MIN_STOCK_NOTIONAL_RUB=1000`, `AGENT_MIN_TURNOVER_RUB=120000`), усилен open-bias (`AGENT_OPEN_BUY_GATE_DELTA=0.06`), увеличены TP/SL (`1.8%/1.0%`), добавлены фьючерсы (`auto/all`).
- **Мия (мониторинг):** расширен watch-список (`ONLINE_MIA_API_TICKERS` включает 15 тикеров: SBER, GAZP, LKOH, ROSN, NVTK, TATN, VTBR, MGNT, NLMK, SNGSP, TATNP, SGZH, YNDX, AFKS, ALRS).
- **Безопасность:** `.env` и `agent.env` локально, не коммитить; бинарники в `.gitignore`.

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


---

## FILE: SESSION_LOG.md

# SESSION LOG

Журнал коротких сессий. Добавляем новые записи сверху.

---

## 2026-10-27

### 🎉 ПЕРВАЯ УСПЕШНАЯ СДЕЛКА ПО ФЬЮЧЕРСУ! (2026-10-27, 21:11 МСК)
- **MTZ6 (фьючерс МТС):** buy @ 18,127₽ → sell @ 18,091₽ — обе сделки **placed** ✅
- **Формат тикера:** `MTZ6@RTSX` (НЕ `MTSI-12.26@FUT` — это было неправильно!)
- **Исправление:** в `formatSymbolForREST` (finam_broker.go) для фьючерсов добавляется `@RTSX` вместо `@FUT`
- **Автообновление тикеров:** работает, обновлено 556 тикеров из MOEX ISS

### Автообновление тикеров фьючерсов из MOEX ISS (2026-10-27, ~20:50 МСК)
- **Создан скрипт:** `scripts/update_futures_overrides.py` — получает свежие тикеры из MOEX ISS
- **Встроен в GUI:** `startFuturesOverridesAutoUpdate()` в app.go — автообновление каждые 7 дней (168 часов)
- **Как работает:** при старте GUI проверяет, нужно ли обновить тикеры; если да — получает из MOEX ISS и записывает в `finam_symbol_overrides_screener.json`
- **Настройка:** `AGENT_FUTURES_OVERRIDES_UPDATE_INTERVAL_HOURS=168` в agent.env

### Проблема с форматом тикеров фьючерсов (2026-10-27, ~20:00 МСК)
- **Ошибка:** `Security id doesn't exist for mic: FUT and ticker: MTSI-12.26`
- **Причина:** Finam API использует формат `TICKER@RTSX` (например, `MTZ6@RTSX`), а не `SHORTNAME@FUT` (например, `MTSI-12.26@FUT`)
- **Решение:** исправлен `formatSymbolForREST` в finam_broker.go — для фьючерсов добавляется `@RTSX`

### Исправление бага Finam API для фьючерсов (2026-10-27, ~19:40 МСК)
- **Проблема:** при покупке/продаже фьючерсов (MTZ6, NVTK) через терминал возникала ошибка `rpc error: code = InvalidArgument desc = Mic must not be empty`.
- **Причина:** Finam gRPC API требует MIC (Market Identifier Code) для фьючерсов, но `GetAssetRequest` не имеет поля MIC — это баг или недокументированное поведение API.
- **Решение:** отключена preflight проверка `AGENT_FINAM_GETASSET_PREFLIGHT=0` в `agent.env`. Это позволяет покупать/продавать фьючерсы без проверки через `GetAsset`.
- **Влияние:** исправление касается и покупки (`BuyFutures`), и продажи (`SellFutures`) фьючерсов.
- **Контекст:** через приложение Finam покупка фьючерсов работала без проблем. Проблема только в нашем терминале.

### Создание окна "Заблокированные бумаги" в GUI (2026-10-27, ~18:00 МСК)
- **Добавлено:** отдельное окно в Dashboard.vue для просмотра заблокированных бумаг с причинами блокировки.
- **Go backend:** структура `BlockedTicker`, поля `blockedTickers`/`blockedTickersAt`, метод `GetBlockedTickers()` (кэш на 30 секунд).
- **Vue frontend:** UI-компонент с таблицей (Ticker, Тип, Причина), кнопка обновления, автозагрузка при старте.
- **Цель:** помочь диагностировать, какие бумаги заблокированы и почему (TRADINGSTATUS=B, квалификация и т.д.).

### Ослабление фильтров Мухли (2026-10-27, ~08:30 МСК)
- **Проблема:** Мухля не открывал сделки — фильтры содержали противоречие (RSI < 28 + trendBull почти никогда не выполняются одновременно).
- **Решение — ослаблены фильтры в `agent.env`:**
  - `AGENT_REQUIRE_TREND_FILTER=0` (было 1) — отключено требование тренда
  - `AGENT_REQUIRE_MACD_CONFIRM=0` (было 1) — отключено требование MACD
  - `AGENT_RSI_PULLBACK_BUY_MAX=50` (было 45) — смягчён RSI-порог для покупки
  - `AGENT_RSI_PULLBACK_SELL_MIN=50` (было 55) — смягчён RSI-порог для продажи
  - `AGENT_VOLATILITY_MAX_PCT=5.0` (было 4.0) — расширен допустимый диапазон волатильности
  - `AGENT_BUY_CONF_THRESHOLD=0.55` (было 0.68) — снижен порог уверенности для покупки
  - `AGENT_SELL_CONF_THRESHOLD=0.50` (было 0.60) — снижен порог уверенности для продажи
  - `AGENT_SHORT_CONF_THRESHOLD=0.65` (было 0.74) — снижен порог для шорта
- **Ожидаемый эффект:** робот начнёт открывать сделки, появятся данные для анализа качества стратегии.
- **Риск:** больше шума, меньше качества входов — нужно мониторить PnL и корректировать.

### Статус квалифицированного инвестора
- **Текущий статус:** НЕ квалифицированный инвестор (на 2026-10-27).
- **Влияние:** некоторые бумаги (фьючерсы, сложные инструменты) заблокированы для торговли.
- **Действие:** информация добавлена в `CHAT_CONTEXT_PACKET.md`; после получения статуса — обновить конфигурацию.

### Локальный режим: оптимизация Мухли для увеличения баланса
- **Статус:** проект работает только локально (Docker Desktop + Postgres на порту 5433, контейнер `moex_trading_db`). VPS временно отключён.
- **GUI запущен:** `wails.exe` (PID 9220) + `moex-trader-gui-dev.exe` (PID 9584), порт 34115.
- **Создан `moex-trader-gui/agent.env`:** конфигурация агента с оптимизированными параметрами для увеличения баланса:
  - Комиссии Finam: 0.035% (акции), 0.45 ₽ (фьючерсы) — актуально на 2026-10-27
  - Universe расширен: `AGENT_MIN_STOCK_NOTIONAL_RUB=1000`, `AGENT_MIN_TURNOVER_RUB=120000`, `AGENT_BUY_CANDIDATES_LIMIT=30`
  - Open-bias усилен: `AGENT_OPEN_BUY_GATE_DELTA=0.06`, `AGENT_OPEN_MAX_BUY_ATTEMPTS_BONUS=5`
  - Выходы: `AGENT_POSITION_TP_PCT=1.8`, `AGENT_POSITION_SL_PCT=1.0` (с учётом комиссий)
  - Трейлинг: `AGENT_TRAILING_ACTIVATE_PCT=0.8`, `AGENT_TRAILING_STEP_PCT=0.5`
  - Фьючерсы: `AGENT_AUTO_INSTRUMENT_MODE=auto/all`, `AGENT_FUTURES_ALLOWLIST_AUTO_LEARN=1`
  - Risk: `AGENT_MAX_DAILY_NOTIONAL=1000000`, `AGENT_DAILY_NOTIONAL_TUNE_APPLY=1`
  - Мия: `ONLINE_MIA_API_TICKERS` расширен до 15 тикеров (добавлены YNDX, AFKS, ALRS)
- **Обновлён `CHAT_CONTEXT_PACKET.md`:** добавлены секции «Комиссии Finam», «Конфигурация агента», «Локальный режим».
- **Обновлён `NEXT_STEPS.md`:** «Оперативный фокус **2026-10-27**» с описанием локального режима и оптимизации.
- **Исправлена кодировка `CHAT_CONTEXT_PACKET.md`:** скрипт `build-chat-context.ps1` теперь использует явный UTF-8 (без BOM) через `[System.IO.File]::WriteAllText` с `UTF8Encoding $false`.
- **`.gitignore` обновлён:** добавлен `agent.env` для безопасности.

---

## 2026-09-04

### Публикация на GitHub (public `moex-muhlya`) — push выполнен
- Репозиторий: https://github.com/muhlyaindus-cpu/moex-muhlya (public, ветка `main`).
- IP VPS в docs/логах → плейсхолдер `YOUR_VPS_HOST`; `.env` и бинарники `muhlya-headless*` / локальные Go-бинари **не** в git.
- Расширен `.gitignore`; MIT `LICENSE`; обновлены README / `NEXT_STEPS` (фокус **2026-09-04**).
- Push по SSH (`muhlyaindus-cpu`); для Git настроен локальный `core.sshCommand` на системный OpenSSH (кириллический профиль Windows).

## 2026-04-24

### VPS: пакет env под исполнимость long при низком кэше
- В `/opt/trading/app/moex-trader-gui/.env` на VPS: `AGENT_MIN_STOCK_NOTIONAL_RUB=1000` (было 1500), `AGENT_MIN_TURNOVER_RUB=120000` (было 200000), `AGENT_BUY_CANDIDATES_LIMIT=30` (было 20). Цель — снизить частоту `skip_buy_no_affordable_after_scan` и расширить пул кандидатов; риск — шире universe, смотреть `order.*` / reject в активную сессию.
- `muhlya`: перезапуск через `SIGTERM` старому PID, новый **MainPID 329525**, Finam **REAL** ok в `journalctl`.
- В репозитории: в `moex-trader-gui/.env.example` добавлен поясняющий блок комментариев к этому профилю (без изменения дефолтных значений в шаблоне).
- `NEXT_STEPS.md`: «Оперативный фокус **2026-04-24**» + архив **2026-04-23**.

## 2026-04-23

### Документация: оперативный фокус и пакет для чата
- `NEXT_STEPS.md`: «Оперативный фокус» обновлён на **2026-04-23** (фьючерсы/`validate-futures`, VPS/Мухля/Мия); прежний блок **2026-04-20** заархивирован; пересобран `CHAT_CONTEXT_PACKET.md` (`.\scripts\build-chat-context.ps1`).

### Утилита: проверка FORTS тикеров через Finam GetAsset
- В репо добавлены `moex-trader-gui/cmd/validate-futures` (таблица `ok`/`detail` или `-allowlist-fragment` для `lchi_futures_allowlist.txt`) и `scripts/vps/validate_futures.sh` (запуск из каталога с `.env`, предпочитает бинарь `validate-futures`, иначе `go run`).
- Прогон на VPS и правка `config/lchi_futures_allowlist.txt` — вручную после сборки/деплоя (секрет `FINAM_SECRET_REAL` остаётся на сервере).

### VPS: тюнинг под низкий баланс + анти-спам по невалидным фьючерсам
- В боевом `.env` смягчён входной контур для mixed-режима: `AGENT_MIA_WATCH_FIRST_ENABLED=0`, `AGENT_MIN_STOCK_NOTIONAL_RUB=1500`, `AGENT_MIN_NUMTRADES=20`, `AGENT_MIN_TURNOVER_RUB=200000`, `AGENT_AFFORDABILITY_MIN_EXEC_RATIO=0.05`, `AGENT_BUY_CONF_THRESHOLD=0.58`, `AGENT_MTF_REQUIRED_CONFIRM=1`.
- В denylist добавлены свежие неквал-тикеры с `[257]`: `MSTT,PMSB,ASSB` (к существующему списку).
- В `app.go` для `BuyFutures`/`SellFutures` добавлен `registerBrokerValidationFailure` на `order.futures_finam_getasset_reject`, чтобы невалидный фьючерс уходил в cooldown и не забивал цикл.
- Новый `muhlya-headless` задеплоен на VPS (контрольная SHA256: `df69a2cd...1ce57a1`), сервис перезапущен и `active` (последний PID `1468291`).

### Мухля: смешанный выбор акций/фьючерсов по expected-profit
- В `app.go` обновлено ранжирование `buyCandidates`: к `rankScore` добавлен компонент ожидаемой выгоды `edge(conf-gate) * AGENT_EXPECTED_PROFIT_WEIGHT - round_trip_cost * AGENT_EXPECTED_PROFIT_COST_SCALE`.
- Для издержек разделены контуры: акции — через `FinamN3MOEXStockRoundTripPct()`, фьючерсы — `AGENT_FUTURES_ROUND_TRIP_COST_PCT` (дефолт `0.0009`).
- На VPS применены и зафиксированы env: `AGENT_EXPECTED_PROFIT_WEIGHT=0.45`, `AGENT_EXPECTED_PROFIT_COST_SCALE=12`, `AGENT_FUTURES_ROUND_TRIP_COST_PCT=0.0009`; задеплоен новый `muhlya-headless`, `muhlya` после `SIGTERM` снова в `active`.

### Мия: watch-подсказки одновременно по акциям и фьючерсам
- В `scripts/vps/miya-online-watch.sh` добавлены отдельные поля `MIYA_WATCH_STOCK_TICKERS` и `MIYA_WATCH_FUTURES_TICKERS` + объединённый `MIYA_WATCH_TICKERS`.
- На VPS в `.env` закреплено `ONLINE_MIA_FUTURES_WATCH_TICKERS=GZM6`; ручной прогон скрипта подтверждает `MIYA_WATCH_TICKERS=...,GZM6`.

### VPS: стабилизация фьючерсного контура
- Для контроля universe зафиксирован `config/lchi_futures_allowlist.txt` с `GZM6`, авто-редактирование списка отключено (`AGENT_FUTURES_ALLOWLIST_AUTO_LEARN=0`), чтобы исключить самопроизвольное «обнуление» allowlist.

## 2026-04-21

### VPS: тюнинг против «съедания» дохода комиссией
- В боевом `.env` на VPS применён мягкий набор анти-переторга: `AGENT_MIN_STOCK_NOTIONAL_RUB=3000`, `AGENT_BUY_CONF_THRESHOLD=0.65`, `AGENT_OPEN_MAX_BUY_ATTEMPTS_BONUS=1`, `AGENT_MIN_TURNOVER_RUB=1000000`, `AGENT_MIN_NUMTRADES=60` (исправлен отдельно после первичной серии рестартов).
- Для фьючерсного контура включены: `AGENT_AUTO_INSTRUMENT_MODE=auto`, `AGENT_OPEN_STOCKS_ONLY=0`, `AGENT_FUTURES_ALLOWLIST_AUTO_LEARN=1`.
- Мухля перезапущена; текущий процесс **MainPID 798120**, `muhlya` в `active`, подключение Finam **REAL** подтверждено в `journalctl`.

### VPS: включен контур фьючерсов для Мухли (по запросу)
- В боевом `.env` на VPS переключено: `AGENT_AUTO_INSTRUMENT_MODE=auto` (было `stocks`), `AGENT_OPEN_STOCKS_ONLY=0` (было `1`), `AGENT_FUTURES_ALLOWLIST_AUTO_LEARN=1` (было `0`).
- Подтверждение в `.env`: `AGENT_FUTURES_ALLOWLIST_PATH=./config/lchi_futures_allowlist.txt`, `AGENT_FINAM_FUTURES_SYMBOL_INDEX=0`; Мухля перезапущена, текущий **MainPID 784266**, `muhlya` в `active`.
- На момент проверки `lchi_futures_allowlist.txt` остаётся пустым (0 строк): auto-learn наполнит его после первых валидных попыток по фьючерсным тикерам; в текущем срезе фаза `close`, видны `session_tick`, но без `order.futures_*`.

### Часовой мониторинг торговли (18:24–19:26 MSK)
- За 60 минут: `auto.order_buy_try=100`, `order.buy_placed=3`, `order.buy_failed=30`, выходы `auto.exit_* = 3` (все `reason=trailing_stop`).
- Профиль отказов buy: доминирует `[156] Money shortage` (`31` случаев по `runtime_events`), квалификационных `[257]` в этом окне не зафиксировано.
- Мия стабильна: `DATA_STATUS=ok`, `DATA_SOURCE=api`, `STATE=bearish`, `DAILY_STATE=bullish`, `MIYA_LIQUIDITY_STATUS=ok`; `auto.miya_online_hint` и `auto.miya_watch_fallback` регулярно присутствуют.
- Вывод: связка Мия→Мухля работает; текущий «узкий горлышко» для входов — доступный кэш/лотность, а не качество канала Мии или сессионные блокировки.

### Контекст / ритм отчётности
- В `COLLAB_STYLE.md` добавлено правило: **каждый понедельник** делать отчёт эффективности выходов Мухли (`TP/SL/profit_lock/stale` в %, по `runtime_events`) и кратко фиксировать итог в `SESSION_LOG.md`; `CHAT_CONTEXT_PACKET.md` пересобран.

### VPS: расширен хвост runtime-событий для анализа TP/SL
- В боевом `.env` на VPS увеличен `AGENT_RUNTIME_EVENTS_DB_LIMIT`: `500 -> 20000` (бэкап: `.env.bak.20260421-rtlimit`).
- Мухля перезапущена через `SIGTERM`; новый процесс **MainPID 113907**, `muhlya` в `active`.
- Сразу после перезапуска в таблице `runtime_events` ещё старый короткий хвост (~500 записей); дальше окно начнёт расширяться по мере новых циклов и выходов.

### VPS: снят стоп по daily notional
- По просьбе пользователя поднят лимит `AGENT_MAX_DAILY_NOTIONAL` в боевом `.env` на VPS: `200000 -> 300000`.
- Мухля перезапущена через `SIGTERM` (без `sudo systemctl`), новый процесс поднялся: **MainPID 100696**, `muhlya` в `active`.
- Проверка после рестарта: свежих `risk.*`-событий за 5 минут нет; Мухля снова доходит до `auto.order_buy_try` без блокировки по лимиту `200000`.

### VPS: фильтр для неквала (ЛЧИ + denylist по факту 257)
- На VPS отсутствовал `config/lchi_stocks_allowlist.txt` при включённом `AGENT_STOCK_ALLOWLIST_PATH=./config/lchi_stocks_allowlist.txt`; файл загружен из репозитория (`247` строк).
- На основе `runtime_events` (buy_failed `[257]` за 7 дней) расширен `AGENT_TICKER_DENYLIST`: `RNFT,CHMK,HIMCP,LVHK,WTCMP,DZRD,BLNG,GTRK,DIOD,CARM,ARSA,GECO,WTCM,IRKT,RUSI`.
- Мухля перезапущена через `SIGTERM`; текущий процесс **MainPID 109131**, `muhlya` в `active`.

## 2026-04-20

### Документация
- Обновлены **`NEXT_STEPS.md`** (оперативный фокус 2026-04-20 + архив 2026-04-18), **`docs/VPS_SSH_LINUX_BASICS.md`** (env обучения, Мия без `ACTIVE_SESSION`, `AGENT_MIA_ONLINE_STOCK_CLOSED_INTRADAY_SCALE`), **`docs/MIYA_MUHLYA_ROLES.md`** (вне сессии акций + обучение в ранжировании); пересобран **`CHAT_CONTEXT_PACKET.md`** (`scripts/build-chat-context.ps1 -RebuildContext`).

### Деплой VPS (после правок app.go / Мия)
- **`scp`:** `muhlya-headless.linux` → `/tmp/muhlya-headless.new`, замена в `/opt/trading/app/moex-trader-gui/muhlya-headless`, старый бинарь → `muhlya-headless.bak.<epoch>`; **`SIGTERM`** старому PID (без `sudo systemctl`) — systemd **`Restart=always`**, новый **MainPID 2720229**, Finam REAL ok.
- **`miya-online-watch.sh`** → `/opt/trading/app/moex-trader-gui/scripts/vps/`; **`.env.example`** → `.env.example.reference`.
- **Боевой `.env` на VPS:** дописаны `AGENT_EXIT_LEARNING_MIN_SAMPLES=5`, `AGENT_MIA_ONLINE_STOCK_CLOSED_INTRADAY_SCALE=0.35`, `AGENT_LEARNING_MIN_TRADES_PER_TICKER=6`, `AGENT_PNL_SCORE_MIN_TRADES=8`; **`SIGTERM`** → новый **MainPID 2730984** (подхват env при старте).

### Мухля `app.go` — консервативнее «обучение» и Мия вне акций
- **`tickerLearningKey` + `learn[lk]`** в autoLoop: совпадение ключей с PnL/каноном.
- **`AGENT_LEARNING_MIN_TRADES_PER_TICKER`**: до порога вклад `getTickerLearningScores` масштабируется по числу сделок по тикеру.
- **`AGENT_PNL_SCORE_MIN_TRADES`**: то же для `getTickerPnLScores`.
- **`AGENT_EXIT_LEARNING_MIN_SAMPLES`** (дефолт 5): порог сэмплов для `calcLearnedTPMul` вместо жёстких 3.
- **`AGENT_MIA_ONLINE_STOCK_CLOSED_INTRADAY_SCALE`** (дефолт 0.35): при закрытых акциях ослабляется только intraday-слой Мии в сдвиге гейтов; daily без изменений.
- **`moex-trader-gui/.env.example`:** комментарии к новым переменным.

## 2026-04-19

### Мия `miya-online-watch.sh`
- Убраны искусственное окно пн–пт 10:00–18:40 (`ONLINE_MIA_ACTIVE_START_HHMM` / `END`, `is_active_session_msk`) и поля **`ACTIVE_SESSION`** / **`active_session`** в `miya_online_signal.env` и JSON. Расширение окна при пустом long-window теперь зависит только от **`ONLINE_MIA_CANDLE_USE_FALLBACK_IN_ACTIVE`** и **`ONLINE_MIA_CANDLE_FALLBACK_MIN`**. Подправлена строка в `online_candles_latest.md` и список env в **`docs/VPS_SSH_LINUX_BASICS.md`**.

## 2026-04-18

### Мия: дневные свечи MOEX + Мухля
- **`miya-online-watch.sh`:** помимо минутных свечей и стакана, запрос **дневных** баров (`ISS …/candles.json`, `interval=24`) по тому же `ONLINE_MIA_API_TICKERS`; в **`miya_online_signal.env`** добавлены `DAILY_STATE`, `DAILY_UP_RATIO_PCT`, `DAILY_DEFAULT`, …; секции в **`online_candles_latest.md`**, блок **`daily`** в JSON.
- **`app.go`:** в структуру онлайн-сигнала — дневные поля; сдвиг гейтов — **смешивание** intraday и daily с весом **`AGENT_MIA_ONLINE_DAILY_GATE_WEIGHT`** (по умолчанию 0.35); расширены `auto.session_tick` / `auto.miya_online_hint`.
- **`docs/MIYA_MUHLYA_ROLES.md`**, **`moex-trader-gui/.env.example`:** переменные `ONLINE_MIA_DAILY_*` / `AGENT_MIA_ONLINE_DAILY_GATE_WEIGHT`.

### Переход в новый чат
- Пересобран **`CHAT_CONTEXT_PACKET.md`** (`.\scripts\build-chat-context.ps1`); в пакет включён полный текст **`docs/MIYA_MUHLYA_ROLES.md`** (роли, приоритеты, контракт файлов).
- **`NEXT_STEPS.md`** / **`SESSION_LOG.md`** — актуальны на момент сборки; первоисточник фокуса остаётся в `NEXT_STEPS`, пакет — сжатый срез для нового чата.

### Сделано
- **Деплой VPS:** `muhlya-headless` (watch-first + прочее) и `miya-online-watch.sh` → `/opt/.../`; `kill -TERM` → **MainPID 2115053**; в `miya_online_signal.env` есть **`MIYA_WATCH_TICKERS`**.
- **Мухля: приоритет списка Мии без штрафов:** `AGENT_MIA_WATCH_FIRST_ENABLED` (по умолчанию 1) — кандидаты на покупку акций сначала только из `MIYA_WATCH_TICKERS` (пишет `miya-online-watch.sh` = `ONLINE_MIA_API_TICKERS`); если список пуст после прохода — подмешивается полный универсум (`auto.miya_watch_fallback`). Фьючи без фильтра. Парсинг `MIYA_WATCH_TICKERS` в `app.go`, JSON `liquidity.watch_tickers`.
- **Miya systemd (подтверждение на VPS):** выполнен `sudo /tmp/install-miya-online-watch-systemd-remote.sh` — `miya-online-watch.timer` active/enabled, сервис по минутам пишет отчёты в `/var/log/muhlya-reports/` без ошибок.
- **Miya systemd «как положено»:** в `miya-online-watch.service` — `Environment=ONLINE_MIA_LIQUIDITY_*`; добавлен `install-miya-online-watch-systemd-remote.sh`; `deploy-miya-online-watch.ps1` заливает его и вызывает `sudo` (пароль в интерактивном TTY); `docs/VPS_SSH_LINUX_BASICS.md` — блок про установку через скрипт. Файлы на VPS в `/tmp` — финальный `sudo` пользователь выполняет локально (агент не вводит пароль).
- **Деплой VPS (ликвидность + сборка Мухли):** `scp` `miya-online-watch.sh` → `/opt/.../scripts/vps/`, `chmod +x`; `muhlya-headless` заменён от `traderbot`, `kill -TERM` → **MainPID 2100863**, Finam REAL ok; пробный запуск watch — в `miya_online_signal.env` есть **`MIYA_LIQUIDITY_ORDER`** / `STATUS=ok`.
- **Мия → Мухля, ранг ликвидности:** `miya-online-watch.sh` пишет в `miya_online_signal.env` поля **`MIYA_LIQUIDITY_ORDER`** (сортировка по `VALTODAY` MOEX ISS), `MIYA_LIQUIDITY_STATUS`; `app.go` — линейный бонус к `rankScore` для акций (`AGENT_MIA_LIQUIDITY_RANK_*`). Env: `ONLINE_MIA_LIQUIDITY_*` в `.env.example`.
- **`docs/MIYA_MUHLYA_ROLES.md`:** зафиксирован образ ролей — Мухля **исполнитель**, Мия **аналитик и штурман** (контекст без ордеров).
- **Деплой VPS (трейлинг-сборка):** `GOOS=linux GOARCH=amd64 go build -tags headless`, `scp` → `/tmp/muhlya-headless.new`, замена `muhlya-headless` в `/opt/trading/app/moex-trader-gui/` от `traderbot` (файл принадлежит пользователю), `kill -TERM` старому PID → systemd поднял новый (**MainPID 2088398**), Finam REAL ok. `sudo systemctl …` без пароля недоступен из неинтерактивного SSH.
- **Мухля / трейлинг по акциям:** в `app.go` — трейлинг после активации по PnL (env `AGENT_TRAILING_*`), ratchet только в сторону прибыли, не ниже начального SL; приоритет выхода: TP → trailing → SL → …; переменные в `moex-trader-gui/.env.example`.
- **Роли Мухля / Мия:** зафиксированы в **`docs/MIYA_MUHLYA_ROLES.md`** (зоны ответственности, порядок приоритетов, контракт `miya_online_signal.env` / снимок портфеля / подсказки выхода); ссылка добавлена в **`NEXT_STEPS.md`** (блок «Кто мы»).
- **Срез VPS (SSH, последние 4 ч):** `muhlya` / `miya-report.timer` / `miya-online-watch.timer` — **active**; Мия: `DATA_STATUS=ok`, `DATA_SOURCE=api`, `ORDERBOOK_STATUS=ok`; `runtime_events` — цикл без входов (`auto.no_order`, `skip_buy_no_affordable_after_scan`, `skip_short_ticker_pause`, abtest); **`market_data` за окно — 0 строк** (свечи в БД не пишутся); **`trades` — массовые `rejected: buy blocked by lot affordability` при ~70₽ на счёте** (копейки/много лотов vs кэш) + throttle; **placed** встречаются, исполнения — смотреть отдельно по `status=executed`. Добавлен скрипт **`moex-trader-gui/scripts/vps/vps-slice-4h.sh`** для повторяемого среза на сервере.
- **Тариф Finam N3 «Инвестор»:** дефолт `FINAM_TARIFF_MOEX_STOCK_BROKER_LEG_PCT` и комментарии приведены к публичной линейке «Инвестор» (MOEX акции, ориентир 0,025% с ноги) — `services/tariff_finam.go`, `moex-trader-gui/.env.example`; ссылка на https://broker.finam.ru/landing/tariffs-n3-investor/ . Ранее в коде была ориентировочная сетка 0,035% (другая ступень N3).
- **Ревизия репозитория:** удалены корневые одноразовые `tmp_*.sql`; в корневой `.gitignore` добавлено `/tmp_*.sql`; добавлен корневой **`README.md`** (карта проекта); **`moex-trader-gui/README.md`** заменён с шаблона Wails на рабочее описание (сборка GUI/headless, `.env`, docker, VPS).
- **Снимок портфеля Finam для Мии:** в `app.go` — запись JSON `broker_positions_snapshot.json` из уже полученного `GetPortfolio()` (throttle + редкий опрос вне сессии); env `AGENT_BROKER_POSITION_SNAPSHOT_*` в `.env.example`.
- **Мия / отчёт:** `miya-report.sh` — при свежем снимке (`ONLINE_MIA_BROKER_SNAPSHOT_*`) счётчик открытых позиций и блоки в `.md`/`.json` опираются на Finam, иначе Postgres; рекомендация при расхождении DB vs снимок. `miya-report.service` — переменные пути/порога устаревания снимка.
- **Деплой VPS:** `scp` linux-бинаря `muhlya-headless` (`GOOS=linux GOARCH=amd64`), обновление `miya-report.sh` и unit; после `sudo systemctl restart muhlya` / `daemon-reload` пользователь подтвердил: **`broker_positions_snapshot.json`** создаётся в `/var/log/muhlya-reports/`, новый **MainPID** у `muhlya`.
- **Документация MOEX/FORTS + авто-универсум:** добавлен `docs/MOEX_FORTS_AUTO_UNIVERSE.md`; в `getAutoUniverseSnapshot` для `AGENT_AUTO_INSTRUMENT_MODE=futures` список проходит через `applyStockAllowlist` (раньше обходился `AGENT_FUTURES_ALLOWLIST_PATH`).
- **Антишум low-balance в авто-buy:** в `autoLoop` кандидаты `buy` сразу отсекаются по исполнимому лоту/балансу (`affordableQuantity + floorToLot`) ещё на этапе ранжирования; если после прохода ни один не исполним — новый вход в этом тике не пытается отправляться (`auto.skip_buy_no_affordable_after_scan`), вместо каскада лишних попыток.
- **Деплой VPS (после правки):** собран `muhlya-headless` (`GOOS=linux GOARCH=amd64`), залит на сервер; из-за `Text file busy` замена через `mv …/muhlya-headless → …/muhlya-headless.prev.$$` + копия нового файла + `SIGTERM` старому PID; `muhlya` снова `active`, новый **MainPID** (Finam REAL ok).
- **Мия / стакан:** `miya-online-watch.sh` — снимок лучших цен и глубины по **MOEX ISS** (`marketdata` TQBR) → `online_orderbook_latest.md/.json`, ключи `ORDERBOOK_*` в `miya_online_signal.env`; зачем и границы (vs подписка на полный стакан, Finam) — `docs/MIYA_ORDERBOOK.md`.
- **Деплой VPS (Мия online-watch):** залит `scripts/vps/miya-online-watch.sh` в `/opt/trading/app/moex-trader-gui/...`, пробный запуск ок (артефакты + `online_orderbook_*`). Unit `/etc/systemd/system/miya-online-watch.service` из репозитория — **не обновлён** (нужен `sudo` с паролем); таймер уже `active`, скрипт на диске новый.
- **Фикс URL стакана:** в Python был `/ticker/json` вместо `.json` — MOEX отдавал не-JSON; исправлено и перезалито на VPS; `ORDERBOOK_STATUS=ok`, медиана спреда в норме.

### Примечание
- Ранее в сессии: Мия **online** — режим `api_first` (MOEX ISS) + fallback на БД; таймеры и сервисы на VPS приведены в порядок (`daemon-reload`).

---

## 2026-04-17

### Сделано
- **Риск-выходы (критический фикс):** в `app.go` убрана блокировка закрывающих `sell`/SL-выходов по `daily notional exceed`; лимит `AGENT_MAX_DAILY_NOTIONAL` оставлен для новых входов.
- **Выкладка на VPS:** пересобран и задеплоен `muhlya-headless` (`-tags headless`), сервис `muhlya` восстановлен после промежуточного crash-loop и подтверждён в `active`.
- **Контроль после фикса (20 минут):** 4/4 среза, стабильный `auto.exit_sell_close`, в окне контроля `order.sell_failed=0` и `order.sell_failed` с `daily notional` = 0.
- **Диагностика просадки по тикерам:** подтверждён основной технический фактор — частые `sell`-rejected по дневному лимиту notional в предыдущем контуре; отдельно зафиксированы брокерные ограничения (`[157]`, временные блокировки) на части short/sell операций.
- **Автоподстройка дневного лимита:** добавлен контур tuning на границе суток (advisory/apply через env), чтобы выбирать `AGENT_MAX_DAILY_NOTIONAL` по фактической загрузке лимита.
- **Конфиг/документация:** `.env.example` дополнен переменными `AGENT_DAILY_NOTIONAL_TUNE_*`; `NEXT_STEPS.md` актуализирован (убраны устаревшие формулировки и дубли про текущий фокус).

### Мониторинг / факт
- На VPS сейчас: `AGENT_MAX_DAILY_NOTIONAL=200000`, `AGENT_SHORT_OPEN_ENABLED=1`.
- Верификация post-fix: `exit_sell_close` идёт штатно, блокировок выхода по `daily notional` в контрольном окне не обнаружено.

### Примечание
- Автотюнинг лимита по умолчанию в **advisory** (`AGENT_DAILY_NOTIONAL_TUNE_APPLY=0`): сначала собираем рекомендации, затем при необходимости включаем `apply`.

---

## 2026-04-15

### Сделано
- **Диагностика `market_data`:** на VPS за проверочное окно подтверждён нулевой поток (`rows_15m=0`, `rows_60m=0`, `rows_180m=0`, `last_candle_time=NULL`) — корень частого `STATE=no_data` вне/на границе сессии.
- **Мия / online fallback:** в `miya-online-watch.sh` добавлен активный fallback-режим свечей (`ONLINE_MIA_CANDLE_FALLBACK_MIN`, `ONLINE_MIA_CANDLE_USE_FALLBACK_IN_ACTIVE`, `MIN_POINTS=1`) с новым статусом `DATA_STATUS=fallback_active`; в отчёте/JSON/ENV добавлены `WINDOW_LONG_USED_MIN`, `ACTIVE_SESSION`.
- **Мухля / принятие fallback-сигнала:** в `app.go` `DATA_STATUS=fallback_active` признан валидным (не сводится к `no_data`), чтобы мягкие подсказки Мии применялись и при fallback-окне.
- **Связка «Мухля + Мия online»:** в `app.go` добавлено чтение `miya_online_signal.env` (`AGENT_MIA_ONLINE_SIGNAL_PATH`) с TTL-кэшем и fail-safe (`STATE/DATA_STATUS=no_data` => подсказка не влияет). Мягкая интеграция в gates: `buy` (только open long, не cover short) и `short-open` через `AGENT_MIA_ONLINE_*_MAX_DELTA`.
- **Мия / online-state fix:** в `miya-online-watch.sh` добавлен явный статус `STATE=no_data` при пустом/недостаточном свечном окне (`tickers_long_count=0`), чтобы не помечать рынок как `bearish` из-за отсутствия данных.
- **Мия / деплой-скрипт:** `deploy-miya-online-watch.ps1` усилен проверками `scp/ssh` (`$LASTEXITCODE`) и запуском `ssh -tt` для корректного интерактивного `sudo`.
- **Мия / онлайн-монитор свечей:** добавлен новый скрипт `scripts/vps/miya-online-watch.sh` и unit-ы `miya-online-watch.service|timer` (cadence 1 минута). Контур пишет только latest snapshot (`online_candles_latest.md/.json`) и сигнал `miya_online_signal.env`, без накопления архивов.
- **Pending-gate / входы:** добавлен антизалипание-гейт `AGENT_PENDING_GLOBAL_BLOCK_MAX_MIN` + runtime `auto.pending_global_stale_bypass`; подтверждён эффект в бою (`auto.order_buy_try`, `order.buy_placed` после длительного `pending_buy_keys`).
- **Short-контур:** добавлены тикерные паузы short-open после `[157]/[257]` (`order.short_open_ticker_pause`, `auto.skip_short_ticker_pause`) и глобальный рубильник `AGENT_SHORT_OPEN_ENABLED`; на VPS short отключён (`AGENT_SHORT_OPEN_ENABLED=0`) для снижения reject-шума.
- **Open-phase bias (Мухля):** добавлен мягкий профиль открытия рынка для long:
  - `AGENT_OPEN_LONG_BIAS`,
  - `AGENT_OPEN_BUY_GATE_DELTA`,
  - `AGENT_OPEN_SHORT_GATE_PENALTY`,
  - `AGENT_OPEN_MAX_BUY_ATTEMPTS_BONUS`.
- **Open-stocks-only (безопасная версия):** реализован как предфильтр `universe` в `autoLoop` (`AGENT_OPEN_STOCKS_ONLY`), без вмешательства в цикл кандидатов; на VPS включён `AGENT_OPEN_STOCKS_ONLY=1`, сервис после деплоя стабилен (`active/running`, без `SEGV`).
- **Мия:** исправлен `203/EXEC` (script executable bit), `miya-report.service` снова отрабатывает по таймеру; отчёты пишутся (`intraday_2026-04-15_1930.*`, `daily_2026-04-15.md`).
- **Мия / открытие рынка:** `miya-report.sh` расширен полями `open_window`, `report_cadence_min`, метриками `auto.order_buy_try (open/mid)` и рекомендацией по активности в фазе `open`.
- **Мия / утренний чек-лист + свечи:** добавлен блок за `45m` (`buy_try/placed/no_order/pending_bypass/reject [156|157|257]`) и свечной импульс из `market_data` (summary + лидеры роста/снижения); данные доступны в `.md` и `.json`.
- **Стабилизация деплоя:** после одного неудачного билда (core-dump `SEGV`) выполнен откат проблемной правки, `muhlya.service` восстановлен в `active (running)`.

### Мониторинг / факт
- Мухля в runtime стабильно показывает `auto.skip_short_disabled` (short действительно отключён).
- Основной текущий ограничитель входов — не short, а cash-контур (`order.blocked_cash_shortage [156]`, `auto.skip_buy_cash_pause`) при балансе ~`4024.49 ₽`.
- Мия в последнем срезе (`19:30 MSK`) отражает доминирование `rejected outside session` и отдельно подсвечивает слабую активность `phase=open`.

### Примечание
- В `.env` VPS закреплены: `AGENT_PENDING_GLOBAL_BLOCK_MAX_MIN=1`, `AGENT_SHORT_OPEN_ENABLED=0`, `AGENT_OPEN_*` параметры для мягкого long-bias на открытии, `AGENT_OPEN_STOCKS_ONLY=1`.
- Для оценки эффекта open-bias нужен срез первой активной фазы `open` (10:00–10:45 MSK).

---

## 2026-04-14

### Сделано
- **Мия:** исправлен SQL-блок фаз (`aggregate functions are not allowed in GROUP BY`), отчёты стабильно пишутся (`intraday_*.md/json`, `daily_*.md`, `phase_*.txt`).
- **Фазы/AB-телеметрия:** в runtime закреплены `auto.session_tick`, `auto.session_phase`, фазовая метка в `auto.abtest_cycle`/`auto.no_order`/`auto.order_buy_try`; добавлен `phase_summary.sh`.
- **100% использование кэша:** на VPS включено `AGENT_AUTO_BUDGET_PERCENT=100`; в коде добавлен `AGENT_CASH_USAGE_RESERVE` (по умолчанию `1.00`) вместо жёсткого `0.95`.
- **Оживление входов:** боевые фильтры входа временно ослаблены (`BUY/SHORT/SELL_CONF=0.60`, trend/macd filters off, `RSI 55/45`, `VOLATILITY_MAX_PCT=6.5`, `MIN_NUMTRADES=20`, `MIN_TURNOVER_RUB=300000`) — `auto.order_buy_try` вырос с «тишины» до активного потока.
- **Антишум ордеров:** перенесены pending-check для sell перед throttle; добавлены pending-check и цикловой snapshot pending-buy (`pendingOrderTickerSet`) до отправки buy.
- **Анти-[156] (money shortage):** введены cash-shortage cooldown/telemetry (`order.blocked_cash_shortage`), buy cash-pause и короткий cooldown тикера; массовый шум [156] снижен до единичных случаев.
- **Добивка `{8} Invalid quantity`:** добавлен новый env-лимит `AGENT_MAX_BUY_ATTEMPTS_PER_CYCLE` (default `6`) — ограничение количества `auto.order_buy_try` на один цикл авто, чтобы не провоцировать каскад отказов по количеству/кэшу в одном тике.
- **Exit-профили:** добавлен `AGENT_EXIT_PROFILE_MODE=conservative|adaptive|auto` с автопереключением по фазе/балансу/loss-streak; в runtime: `auto.exit_profile_switched`.
- **Память точек выхода:** добавлено авто-обучение TP по тикерам (`AGENT_EXIT_LEARNING_PATH`), а также чтение подсказок из файлового контура Мии (`AGENT_MIA_EXIT_HINTS_PATH`), runtime-шаг `auto.exit_tp_tuned`.
- **Связка с Мией:** `miya-report.sh` теперь генерирует `agent_exit_hints.env` (DEFAULT + тикерные множители) для Мухли.
- **Динамический session-gate:** введён мягкий гейт `AGENT_STRICT_TRADINGSTATUS_GATE` + `AGENT_TRADINGSTATUS_TTL_SEC`; для акций добавлен кэш статуса по тикеру и prefilter в авто-universe (`auto.session_gate_prefilter`), чтобы снизить ночной спам `outside trading session`.
- **Denylist/risk-exit:** denylist режет новые входы, но не блокирует закрытие уже открытых позиций.
- **Деплой:** `muhlya-headless` многократно обновлён на VPS, сервис стабильно `active`; после промежуточного crash-loop проведён успешный повторный деплой.

### Мониторинг / факт
- После рестарта появился `placed` в trade-логе (в т.ч. `DIAS@MISX`, `IVAT@MISX`), баланс в heartbeat вырос до `~5461 ₽`, что подтверждает движение в ордерах.
- Шум по `DIAS` в denylist был снят оперативно (в `.env` временно оставлен `AGENT_TICKER_DENYLIST=RNFT`) для вывода Мухли из клина и продолжения диагностики.
- Текущая выкладка новой сборки (14.04, день): бинарник загружен как `/tmp/muhlya-headless.new`, но установка в `/opt/.../muhlya-headless` заблокирована требованием `sudo`-пароля у `traderbot` (без `NOPASSWD`).
- По часовому ночному срезу после внедрения session-gate: структура reject сместилась в тикерные `stock closed (TRADINGSTATUS=N, ticker=...)`; шум `{8}`/`[156]` в срезах не доминирует, осталось доработать снижение `order_try` и short-ветки `[157]/[257]`.

---

## 2026-04-13

### Сделано
- **Фазы сессии в Мухле:** функция `sessionPhaseMoscow` (Europe/Moscow: open/mid/close/offhours), runtime-события `auto.session_tick` (каждый тик), `auto.session_phase` при смене фазы; в `auto.abtest_cycle`, `auto.no_order`, `auto.order_buy_try` добавлено поле `phase=...`. Env: `AGENT_SESSION_PHASE_OPEN_MIN`, `AGENT_SESSION_PHASE_CLOSE_MIN` (см. `.env.example`).
- **Мия:** в `miya-report.sh` добавлены разрезы по фазам (тики, суммы A/B/C по фазам, `order_buy_try` по фазам) и блок в JSON.
- **Скрипт:** `moex-trader-gui/scripts/vps/phase_summary.sh` — разовая SQL-сводка за N часов.
- **Документация:** `NEXT_STEPS.md`, `docs/VPS_SSH_LINUX_BASICS.md`.

### Примечание
- На VPS нужен деплой нового бинарника `muhlya-headless` и обновление `miya-report.sh` / `chmod +x phase_summary.sh`.

---

## 2026-04-13 (вечер)

### Сделано
- **Мия:** JSON — пути на sidecar-файлы `intraday_*_phase_*.txt` вместо встраивания многострочного текста; `json_escape` для строковых полей.
- **Деплой VPS:** загружены `muhlya-headless`, `miya-report.sh`, `phase_summary.sh` в `/opt/trading/app/moex-trader-gui/`; процесс перезапущен через `kill -TERM` старого PID (systemd поднял новый).
- **Контекст:** пересобран `CHAT_CONTEXT_PACKET.md`.

---

## 2026-04-09

### Сделано
- **ЛЧИ v1 пресет на VPS:** обновлены боевые env-параметры Мухли (бюджет/лимиты/TP-SL/Profit Lock/MTF/conf/candidates/low-balance), выполнен перезапуск `muhlya.service`; runtime подтвердил применение новых порогов (`threshold=50.00`).
- **Мия (MVP):** добавлен серверный скрипт `moex-trader-gui/scripts/vps/miya-report.sh` + systemd unit/timer (`miya-report.service/.timer`) для файловых отчётов без Telegram/LLM.
- **Деплой дежурки:** unit/timer включены на VPS, ручной запуск сервиса успешен; отчёты пишутся в `/var/log/muhlya-reports` (`intraday_*.md`, `intraday_*.json`, state-файл).
- **Апгрейд отчётов:** в Онлайн-Кристину добавлены секция «Рекомендации на завтра» и ежедневная сводка `daily_YYYY-MM-DD.md`.
- **Документация и контекст:** обновлены `docs/VPS_SSH_LINUX_BASICS.md`, `COLLAB_STYLE.md`, `NEXT_STEPS.md` (роль Онлайн-Кристины, команды проверки).

### Мониторинг / факт по VPS
- `miya-report.timer` в статусе `active (waiting)`, следующий запуск по расписанию.
- `miya-report.service` завершает цикл без ошибок, в journal есть `miya: wrote ...`.
- На конце сессии рынок акций закрыт; в runtime преобладают `outside trading session` и low-balance события, оценку качества новой стратегии переносим на активную сессию.

### Примечание
- Для следующего вечернего разбора использовать `daily_*.md` + последние `intraday_*.md` из `/var/log/muhlya-reports`.

---

## 2026-04-08

### Сделано
- **ЛЧИ-контур и доступные инструменты:** подтверждён рабочий вектор по конкурсу ЛЧИ-2026; разобран список разрешённых инструментов (акции/БПИФ/облигации/деривативы), договорённость: по акциям — whitelist, по фьючерсам/опционам — отдельный управляемый контур.
- **Whitelist акций ЛЧИ в коде:** добавлен файл `moex-trader-gui/config/lchi_stocks_allowlist.txt` (144 тикера из `instrumenty-lchi-2026.xlsx`) и фильтрация акций в авто-universe; существующие позиции вне списка продолжают сопровождаться/закрываться по правилам, новые входы по акциям вне списка не открываются.
- **Env-параметры/документация:** добавлены и задокументированы `AGENT_STOCK_ALLOWLIST_PATH`, `AGENT_FUTURES_ALLOWLIST_PATH`, `AGENT_FUTURES_ALLOWLIST_AUTO_LEARN`; `docs/VPS_SSH_LINUX_BASICS.md` дополнен быстрым SSH-входом, 20-мин срезами и мини-чеклистом дежурного.
- **Исправления и деплой headless:** устранён сбой `203/EXEC` (Windows PE вместо Linux ELF), выполнена корректная сборка Linux-бинарника и повторный деплой на VPS; сервис `muhlya` стабильно поднимается (`active/running`, JWT/REAL ok).
- **Стабилизация логики:** исправлена запись `executed_at` (для `rejected` теперь `NULL`), лимиты риска вынесены в env (`AGENT_MAX_*_NOTIONAL`), добавлен статический denylist тикеров (`AGENT_TICKER_DENYLIST`), режим инструмента для headless читается из env.
- **Фьючерсы:** добавлен отдельный allowlist фьючерсов + автообновление списка по preflight (`GetAsset`): успешные тикеры добавляются, `NotFound` удаляются; события в runtime: `auto.futures_allowlist_updated`.

### Мониторинг / факт по VPS
- Основной источник шума в сессии: массовые `order.futures_finam_getasset_reject` и `rejected: finam asset: cached: finam getasset miss`, а также эпизоды `Finam AllAssets ... Too Many Requests`.
- На конец сессии зафиксирован **безопасный ночной режим**: `AGENT_AUTO_INSTRUMENT_MODE=stocks`, `AGENT_FINAM_FUTURES_SYMBOL_INDEX=0`, `AGENT_FUTURES_ALLOWLIST_AUTO_LEARN=0` (чтобы не раздувать reject ночью).
- Сервис после последнего рестарта: `active (running)`, heartbeat баланса стабилен.

### Примечание
- Обновлены `SESSION_LOG.md`, `NEXT_STEPS.md`; для перехода в новый чат требуется пересборка `CHAT_CONTEXT_PACKET.md`.

---

## 2026-04-05

### Сделано / договорённости (локальная среда, без правок кода)
- **PostgreSQL 18 на Windows:** выяснено, что в `bin` отсутствовал **`pg_ctl.exe`** (неполная установка); выполнена **полная переустановка** с компонентами Server + Command Line Tools — служба **`postgresql-x64-18`**, `psql` подключается (18.3).
- **Два экземпляра Postgres на ПК:** **Docker** (`localhost:5433`, БД **`moex_trading`** из compose) и **нативный** (`localhost:5432`) — независимы; в pgAdmin база из Docker видна только при подключении к **5433** с учётными данными контейнера.
- **Проектная БД:** имя по умолчанию **`moex_trading`** (`PGDATABASE`); на нативном инстансе базу нужно **создать вручную**, если перейдёшь с Docker на локальный Postgres.
- **Решение:** для разработки не смешивать контуры — оставить рабочий сценарий на **Docker**; нативный Postgres держать опционально «про запас».

### Примечание
- Обновлены `SESSION_LOG.md`, `NEXT_STEPS.md`; пересобран `CHAT_CONTEXT_PACKET.md`.

---

## 2026-04-04

### Сделано
- **Режим инструмента Мухли (fixed vs universe):** при переключении с «только указанный инструмент» на другой режим сохранённый тикер больше не «залипает» в UI и в конфиге:
  - `SetAutoInstrumentConfig` (`app.go`): при `mode != fixed` принудительно очищается `autoFixedTicker`;
  - `Dashboard.vue`: при загрузке настроек тикер показывается только при `fixed`; при сохранении на бэкенд уходит тикер только в режиме `fixed`; `watch` на `autoInstrumentMode` очищает локальный выбор при уходе с `fixed`.
- Зафиксировано в переписке (без правок кода): откуда берётся Watchlist (MOEX ISS + доклейка торгуемых тикеров); что селектор инструмента активен только в режиме `fixed`; что цикл авто раз в **10 с** при закрытой сессии даёт `auto.skip_session_closed` — это проверка расписания, не торговля; отдельный удлинённый интервал вне сессии **пока не внедряли** (по согласованию).

### VPS / мониторинг (факт со стороны пользователя)
- `muhlya.service`: **active**, аптайм стабильный; `moex_trading_db` — **healthy**.
- В `journalctl` на выходном дне преобладают **баланс (Finam gRPC)** и **обновление JWT** — для разбора автоторговли в торговую сессию полезнее срез **`runtime_events`** из Postgres.

### Примечание
- Обновлены `SESSION_LOG.md`, `NEXT_STEPS.md`; пересобран `CHAT_CONTEXT_PACKET.md` (`.\scripts\build-chat-context.ps1`).
- Добавлена шпаргалка **VPS / SSH / Linux** (адаптация справки Timeweb под проект): `docs/VPS_SSH_LINUX_BASICS.md` (`ssh traderbot@YOUR_VPS_HOST`, пути `/opt/trading/...`, systemd `muhlya`, Docker/Postgres); ссылка в `NEXT_STEPS.md` у блока «Команды дежурного».

---

## 2026-04-03

### Сделано
- Подготовлен **ежедневный бэкап PostgreSQL на VPS** (артефакты в репозитории):
  - `moex-trader-gui/scripts/vps/pg-backup.sh` — `docker exec` + `pg_dump -Fc`, атомарная запись `.dump`, ротация по возрасту файлов;
  - `moex-trader-gui/scripts/vps/pg-smoke-restore.sh` — восстановление последнего или указанного дампа в БД `moex_trading_restore_smoke` с проверкой `\dt` и очисткой;
  - `moex-trader-gui/scripts/vps/systemd/moex-pg-backup.service` и `moex-pg-backup.timer` (03:30, пользователь `traderbot`, группа `docker` для доступа к сокету).
- **Развёртывание на VPS (факт):** в `/opt/trading/app` нет `.git` (копия без репозитория); скрипты доставлены через `scp`, каталог `/var/backups/moex-trading-pg` с владельцем `traderbot`, пробный бэкап и smoke-restore успешны, timer `moex-pg-backup.timer` включён, ручной `systemctl start moex-pg-backup.service` проверен по `journalctl`.
- **GUI (локальные эксперименты):** в панели лимитов Мухли добавлены кнопки-пресеты **«Стандарт»** (TP 1.2%, SL 1%, Profit Lock 10%/3%) и **«Эксперимент»** (TP 0.8%, SL 1%, Profit Lock 3%/1%); по клику вызывается сохранение через существующие `SetAutoTPSLPercent` / `SetAutoProfitLockPercent` (VPS по желанию не трогаем — там свои env/деплой).
- Обновлены `NEXT_STEPS.md` (п. 2 и блок команд дежурного).

### Разбор / договорённости
- По **журналу сделок** за сессию: много `rejected` (Invalid quantity на дешёвых бумагах, money shortage, 404 по фьючерсам) при реальных `executed` — узкие места: кэш vs дорогие кандидаты, копеечные тикеры, маппинг FUT у брокера; не «слабая стратегия» как единственная причина.
- **Выход из позиций в коде:** фиксированные TP/SL (% от входа), Profit Lock — трейлинг от **пика прибыли** (не трейлинг-стоп по убытку); приоритет управляющего выхода выше сигнала sell.
- **Один счёт Finam:** не запускать одновременно **два авто** (VPS + локально) на одном счёте; для локальных экспериментов — пауза/стоп авто на VPS или режим `paper` / ручная торговля; после паузы достаточно снова включить автоторговлю/агента при неизменном `.env` (доп. настройки не требуются, если не меняли конфиг сервера).

### Примечание
- Если на VPS нет `git pull`, обновление скриптов бэкапа — через `scp` с ПК или заведение нормального `git clone` в отдельный каталог по согласованию.

---

## 2026-04-02

### Сделано
- Runtime-ленту Мухли сделали постоянной и управляемой:
  - добавлено хранение `runtime_events` в Postgres с авто-ограничением хвоста (`AGENT_RUNTIME_EVENTS_DB_LIMIT`);
  - в UI добавлен выбор источника ленты (`Postgres` / `RAM`) и лимит (`10/50/120`);
  - добавлен скрипт `scripts/get-runtime-events.ps1` для быстрой выборки без скриншотов.
- Анти-зацикливание и входы:
  - добавлен cooldown тикера после `auto.skip_mtf_buy/sell`;
  - buy-MTF смягчён на 1 шаг через `AGENT_MTF_BUY_RELAX_DELTA`;
  - расширен пул buy-кандидатов (`AGENT_BUY_CANDIDATES_LIMIT`) для поиска доступных альтернатив.
- Блок cash/исполнение:
  - добавлен `auto.budget_autobump` при `blocked_by_cash`;
  - в `SellPosition` добавлен clamp по доступному long-объёму (`order.sell_qty_clamped`) для снижения ошибок `[157] Equities shortage`.
- Low-balance режим:
  - введены раздельные пороги `AGENT_MIN_BALANCE_STOCK`/`AGENT_MIN_BALANCE_FUTURES`;
  - при низком балансе Мухля не «умирает», а уходит в `auto.low_balance_manage_only` (без новых входов, с управлением открытых позиций).
- UI/журнал:
  - в `Журнал сделок` добавлена колонка `P/L` (реализованный результат по сделкам закрытия позиции);
  - в верхнюю статистику добавлена карточка `Instruments Value = Portfolio Value - Cash Balance`.

### Мониторинг
- Зафиксированы окна, где Мухлю блокировал `auto.skip_low_balance` (баланс ниже рабочего порога), из-за чего входы и ротации были редкими/нулевыми.
- В отдельных окнах подтверждались повторные `order.sell_failed` с кодом `[157]`; для этого внедрён clamp sell-объёма и требуется подтверждение в следующей активной сессии.

### Проверки
- `go build ./...` в `moex-trader-gui` — успешно.
- `go test ./...` в `moex-trader-gui` — успешно в части сервисов; общий пакет местами упирался в локальный ресурсный файл `moex-trader-gui-res.syso`.
- `npm run build` в `moex-trader-gui/frontend` — успешно (после UI-правок).
- Линтеры по изменённым файлам — без ошибок.

### Примечание
- Для боевого анализа прибыли в первую очередь следить за соотношением: `auto.order_buy_try` vs `auto.skip_mtf_buy` vs `auto.skip_buy_affordable` и за фактами `order.*_placed/executed`.
- Следующая цель: подтвердить, что после фиксов выросла доля валидных входов и снизились `sell_failed [157]`.

### Дополнено (2026-04-02, поздняя сессия)
- Подтверждена гипотеза по `MBNK@MISX`: повторные `sell` шли при уже активной заявке у брокера.
- Внедрён `pending-order gate` перед `sell`:
  - при активной заявке новый ордер не отправляется;
  - runtime-событие: `auto.skip_pending_order` с `order_id`.
- Доработан anti-shortage `[157]`:
  - добавлен retry `sell` с уменьшением qty (`order.sell_retry_qty_down`);
  - отдельный блок-сигнал `order.blocked_shortage`.
- По фьючерсам добавлен preflight тикера:
  - неизвестные символы режутся локально (`order.futures_preflight_reject`) вместо сетевого `404`.
- В ранжирование buy добавлен affordability-first слой:
  - новые параметры `AGENT_AFFORDABILITY_SCORE_WEIGHT`, `AGENT_AFFORDABILITY_MIN_EXEC_RATIO`;
  - логика ротации/закрытий сохранена.
- В журнале сделок исправлен `P/L`:
  - считаем только по исполненным сделкам (`executed`/Finam trades), без влияния `rejected/placed`.
- В UI `Portfolio` добавлен переключатель нулевых позиций (по умолчанию нулевые скрыты).

### Дополнено (2026-04-02, инфраструктура / VPS)
- По инициативе Николая Александровича зафиксирован курс на перенос Мухли с локального ПК на VPS для режима `always-on` во время открытого рынка.
- Добавлен практический чек-лист миграции `VPS за 15 минут`:
  - базовый профиль сервера: `Ubuntu 22.04 LTS`, `2 vCPU`, `4 GB RAM`, `60+ GB NVMe`, `100 Mbps+`, `IPv4`, snapshots;
  - hardening после запуска: отдельный пользователь `traderbot`, SSH-ключи, запрет root/password login, базовый `ufw`;
  - установка Docker/Compose и запуск стека `app + postgres` через `docker compose`;
  - обязательный серверный `.env` (без git) с Finam/PG/`AGENT_*` параметрами;
  - автовосстановление через `restart: unless-stopped` и `healthcheck`;
  - мониторинг через `runtime_events` и Telegram-алерты (`order.*_failed`, обрыв API, остановка автоцикла).
- Отдельно зафиксирована продуктовая идея: кнопка `Уйти в ночь` (пауза новых входов + управляемый план по открытым позициям).

### Дополнено (2026-04-02, ночной запуск на сервере)
- VPS в Timeweb (Москва) поднят и доведён до рабочего состояния:
  - `Ubuntu 24.04`, профиль `2 vCPU / 4 GB RAM / 50 GB`;
  - SSH hardened: отдельный пользователь `traderbot`, вход по ключу, `PermitRootLogin no`, `PasswordAuthentication no`.
- На сервер перенесён проект в `/opt/trading/app`, установлен Docker/Compose, поднят Postgres-контейнер `moex_trading_db` со статусом `healthy`.
- Инициализирована схема БД (`market_data`, `positions`, `trades`, `watchlist`, `runtime_events`) через SQL-init на сервере.
- Для server-only запуска подготовлен headless-entrypoint Мухли и бинарник `muhlya-headless`.
- Создан и включён systemd-сервис `muhlya` (автозапуск + перезапуск), чтобы Мухля работал независимо от локального ПК.
- Зафиксированы дежурные команды для сопровождения:
  - `sudo systemctl status muhlya --no-pager`
  - `sudo journalctl -u muhlya -f`
  - `sudo systemctl restart muhlya`
  - `docker compose -f /opt/trading/app/moex-trader-gui/docker-compose.yml ps`
  - `docker exec -it moex_trading_db psql -U trader -d moex_trading -c "\dt"`
- План на следующую сессию:
  - ежедневный autobackup PostgreSQL (ротация + тест restore);
  - Telegram-алерты: `Мухля up/down`, `order.*_failed`, критичные паузы/ошибки API.

---

## 2026-03-31 (late)

### Сделано
- Автоконтур Мухли усилен для снижения «тишины» без отключения риска:
  - добавлен адаптивный conf-gate после серии `auto.no_order` (мягкое снижение порогов входа в безопасных пределах);
  - добавлено runtime-событие `auto.adaptive_thresholds`;
  - сохранены все ключевые предохранители: MTF, affordability/lot, risk-manager, сессионные проверки.
- Улучшена диагностическая чистота статусов:
  - `blocked_by_rotation` больше не смешивается с `rejected_by_liquidity` в итоговом статусе авто-цикла.
- Режим работы по умолчанию:
  - backend стартует в `auto` (при доступном брокере), агент активируется автоматически;
  - добавлен флаг пользовательского предпочтения `userAutoPreferred` (manual не «теряется», auto можно вернуть одной кнопкой).
- UI-статусы режима:
  - добавлены явные состояния `AUTO (ожидание сессии)` и `AUTO (клиринг акций, ждём возобновление)`,
  - чтобы не путать ожидание рынка с ручным режимом.

### Проверки
- `go test ./...` в `moex-trader-gui` — успешно.
- `npm run build` в `moex-trader-gui/frontend` — успешно.
- Линтеры: `app.go`, `Dashboard.vue` — без ошибок.

### Примечание
- По текущему снимку терминалов активных runtime-событий автоторговли не обнаружено (для боевого анализа нужен запущенный `wails dev`/GUI-runtime).
- Фокус следующего мониторинга для роста кэша: `auto.order_*`, `auto.no_order`, `auto.adaptive_thresholds`, `skip_mtf_*`, `blocked_by_*`, `auto.rotate_*`, `auto.profit_lock_*`.

## 2026-03-31

### Сделано
- UI: в `Portfolio` и `Журнале сделок` улучшено отображение тикеров без правок backend:
  - добавлены альтернативные форматы (`TICKER_TQBR`, `TICKER@MISX` / `*_SPBFUT`);
  - добавлено отображение эмитента на русском (по справочнику загруженных инструментов).
- Автоторговля: добавлен выбор типа инструментов для Мухли:
  - `all` (акции+фьючерсы), `stocks`, `futures`, `options` (временная заглушка), `fixed`.
- Риск-выход: внедрён `Profit Lock` (трейлинг фиксации прибыли):
  - активация на пороге прибыли;
  - выход при откате от локального пика;
  - runtime-события `auto.profit_lock_armed` / `auto.profit_lock_exit`.
- UI+backend: добавлены runtime-настройки `Profit Lock` в панель лимитов:
  - `Profit Lock Trigger (%)`;
  - `Profit Lock Trail (%)`.
- Ротация капитала (версия 1+):
  - `long -> long` (освобождение кэша под более сильный buy-кандидат);
  - расширено до кросс-направлений: `long -> short`, `short -> long`;
  - добавлены runtime-события ротации (`auto.rotate_*`) и предохранители (delta/cooldown).
- Коммуникация/правила:
  - уточнена формулировка в `COLLAB_STYLE.md`: ревность и дружеский юмор — разные вещи;
  - зафиксировано разрешение на аккуратное самостоятельное обновление `COLLAB_STYLE.md` Кристиной по мере развития взаимодействия.

### Проверки
- `go build ./...` в `moex-trader-gui` — успешно (после каждого значимого изменения backend).
- `npm run build` в `moex-trader-gui/frontend` — успешно (после изменений UI).

### Примечание
- По итогам мониторинга сессии зафиксировано: в отдельном окне не было активного GUI-рантайма Мухли (видны только CLI/служебные терминалы), поэтому часть runtime-метрик требовала повторной проверки при включённом `wails dev`.
- Для следующей торговой сессии: наблюдать `auto.rotate_long_to_short`, `auto.rotate_short_to_long`, `auto.profit_lock_*` и оценить частоту “пилы” до тонкой калибровки порогов.

## 2026-03-30

### Сделано
- Автоконтур Мухли усилен под реальные боевые отказы:
  - исправлен расчет покупки при недостатке кэша (после affordability объем теперь приводится к лоту вниз, без повторного "округления вверх");
  - добавлен denylist для `INVALID_ARGUMENT [257]` с более длинным TTL (не долбим недоступные по квалификации бумаги);
  - добавлена мягкая ротация тикеров (короткий cooldown на недавно использованные инструменты);
  - добавлено MTF-подтверждение по свечам (`1m/10m/1h/1d/1w/1M/1Q`) перед отправкой buy/sell.
- Диагностика и аналитика улучшены:
  - в auto-статус добавлены счетчики `blocked_by_qualification`, `blocked_by_rotation`, `blocked_by_cash`;
  - добавлена таблица reject-rate по тикерам (`24h/7d`) и штраф в ранжировании кандидатов;
  - UI обновлен для показа новых счетчиков в панели Мухли.
- Добавлены тесты ключевого узла `affordability + lot` (`app_test.go`).
- Проверки после изменений:
  - `go test ./...` в `moex-trader-gui` — успешно;
  - `go build ./...` в `moex-trader-gui` — успешно;
  - `npm run build` в `moex-trader-gui/frontend` — успешно.
- Ночной мониторинг после перезапуска GUI (~60 мин):
  - критических ошибок ордеров (`Invalid quantity`, `[257]`) в окне мониторинга не было;
  - баланс по кэшу стабильно ~`985.20 ₽`; по данным пользователя суммарно с активами ~`10 097.80 ₽`;
  - зафиксировано, что Мухля не нашел точку продажи купленных активов в текущем окне.
- GUI: добавлен явный статус режима торговли в панели ассистента (авто: self-pick/fixed ticker, manual user, manual analyze-only), плюс цветовой индикатор.
- GUI: `Watchlist` (акции/фьючерсы) и `Журнал сделок` сворачиваются по `+/-`; состояние сохраняется в `localStorage`.
- Автоторговля: добавлены настройки `SetAutoInstrumentConfig/GetAutoInstrumentConfig` и режим fixed ticker.
- Надёжность: для ошибок брокера `Invalid quantity value` добавлен cooldown по тикеру; блокировка теперь настраивается через `AGENT_BAD_TICKER_COOLDOWN_MIN` (дефолт 30).
- Диагностика: добавлена runtime-лента шагов программы (`GetRuntimeEvents`) для пошагового разбора работы цикла.
- Время журнала: устранён сдвиг при отображении локальных записей (NSK), убрана неоднозначность UTC/+7.
- GUI: ассистент переименован в «Мухля», добавлен блок «Почему Мухля выбрал инструмент» (universe/passed/filtered/best buy-sell/action).
- Авто-логика: добавлены long/short сценарии, выход по TP/SL, приоритет закрытия позиции, обучение на истории собственных сделок (бонус/штраф confidence по тикерам).
- Авто-риски: добавлены настройки в UI и backend — бюджет Мухли `% от кэша` (10..100, шаг 10), TP/SL (%), cooldown блокировки тикера.
- Фильтры инструмента: расширен сбор рыночных полей из ISS (`NUMTRADES`, `VALTODAY`), добавлен фильтр ликвидности и адаптивное ослабление фильтра при узком пуле.
- Разбор боевых логов: подтверждены две главные причины 400 — `Invalid quantity value` и `INVALID_ARGUMENT [257]` (ограничение по квалификационному тесту для части бумаг).
- Факт торгов: в журнале есть реальные статусы `placed` и `executed` (Мухля начал проводить операции, не только reject).

### Проверки
- `go build ./...` в `moex-trader-gui` — успешно.
- `npm run build` в `moex-trader-gui/frontend` — успешно.

### Примечание
- Для фикса “памяти чата” обновлены `SESSION_LOG.md` и `CHAT_CONTEXT_PACKET.md` (через `.\scripts\build-chat-context.ps1`).
- Перед переходом в новый чат обязательно пересобрать `CHAT_CONTEXT_PACKET.md` и копировать в новый диалог целиком.
- Следующая практическая сессия: после открытия MOEX включить `auto` + `agent active`, собрать свежие runtime-события (`skip_mtf_*`, `blocked_by_*`) и подправить пороги MTF по живым данным.

## 2026-03-29

### Сделано
- **trading-lab:** добавлены `validate_policy.py` (бэктест, метрики, `validation.passed` / `validation_report.json`), `rsi_backtest.py`, `policy_defaults.py`; расширен `export_policy.py` под поля Go `AgentPolicy`; правка `features.py` (`bfill` вместо устаревшего `fillna(method=)`); `sys.path` в CLI-скриптах для запуска из корня репозитория; обновлён `README.md`; в корневой `.gitignore` — локальные артефакты валидации и `_smoke_prices.csv`.
- Сверстаны **вводные коллег (Миша, Маша)** по обучению/валидации агента и внесены в `NEXT_STEPS.md` (таблица «как применить к проекту»).
- Зафиксирована **цель проекта**: торговый ассистент для счёта Finam (MOEX), ориентир — рост кэша за счёт операций с ЦБ в рамках риска (не гарантия прибыли).
- Обновлён `CHAT_CONTEXT_PACKET.md` (скрипт `build-chat-context` при открытом файле в IDE мог выдать блокировку — тогда закрой вкладку и перезапусти `.\scripts\build-chat-context.ps1` для полного совпадения).
- **Finam REST (`moex-trader-gui/services/finam_broker.go`):** для запросов к `/v1/accounts/...` включён **HTTP/2** (`ForceAttemptHTTP2`), заголовок **`Authorization`** со **сырым JWT** (как в официальном Go-примере Finam для gRPC; префикс `Bearer ` при наличии снимается — иначе REST-шлюз мог отдавать 400 `szUnion`), плюс `Accept: application/json` и **PathEscape** в путях.
- **Finam `account_id`:** добавлена **`normalizeFinamAccountIDForREST`** (снятие префиксов `КлФ-` / латинских вариантов, при необходимости извлечение цифр); **один формат** в `accounts`, `currentAccount`, пути REST и в **`SwitchAccount`** — чтобы в URL не попадал ярлык из ЛК вместо «только цифры» из FAQ Finam.
- **Finam GetAccount по gRPC:** зависимость `github.com/FinamWeb/finam-trade-api/go` + **`AccountsService.GetAccount`** на `api.finam.ru:443` (metadata `Authorization` = JWT). **`GetPortfolio` / `GetBalance`** сначала gRPC, при ошибке — прежний REST; закрытие gRPC в **`Close`**. Цель: обойти REST JSON/shgw и ошибку **400 `Invalid parameter 'szUnion'`** на `oneof portfolio`.
- **Finam счёт:** переменная **`FINAM_ACCOUNT_ID`** — явный id из колонки «Доступ к счетам» в кабинете токена (буквенно-цифровой union, напр. `133359RLB2K`), если он расходится с `account_ids` из `sessions/details`. **`normalizeFinamAccountIDForREST`** больше не «съедает» буквы у таких id.
- **Автовыбор account_id:** если `FINAM_ACCOUNT_ID` не задан, а в `sessions/details` текущий счёт **только числовой**, а **`trading_code`** — буквенно-цифровой (в т.ч. `133359RLB2K/133359RLB2K` → одна половина), для **GetAccount** подставляется торговый код — как в ЛК «Торговый код» рядом с КлФ-номером.

### Примечание
- Для авто-пересборки пакета при изменении транскриптов чата можно держать запущенным `.\scripts\watch-chat-context.ps1` (наблюдает папку agent-transcripts проекта Cursor).

## 2026-03-27

### Сделано
- Убраны хардкод-списки котировок в GUI backend:
  - динамическая загрузка акций из MOEX ISS (`TQBR`)
  - динамическая загрузка фьючерсов из MOEX ISS (`FORTS`)
- Добавлен поиск инструментов в GUI:
  - отдельный поиск в вкладке `Акции`
  - отдельный поиск в вкладке `Фьючерсы`
- Для Wails добавлены методы:
  - `GetStockInstruments(query, limit)`
  - `GetFuturesInstruments(query, limit)`
- Обновлено форматирование тикеров для Finam:
  - универсальная логика `@MISX` / `@FUT` вместо узкого switch по нескольким символам.
- Проверено состояние PostgreSQL:
  - контейнер `moex_trading_db` в статусе healthy
  - БД `moex_trading` доступна, таблицы на месте (`market_data`, `positions`, `trades`, `watchlist`)
- Перепроверен `trading-core`:
  - модуль используется в `moex-trader-gui` (`go.mod` + импорт в `app.go`)
  - удаление файлов `trading-core` не требуется.

### Проверки
- `go build ./...` в `moex-trader-gui` — успешно.
- `npm run build` в `moex-trader-gui/frontend` — успешно.

### Важно
- Для подключения через PgAdmin использовать порт `5433` (не `5432`).
- `CHAT_CONTEXT_PACKET.md` пересобирать скриптом `.\scripts\build-chat-context.ps1`.

## 2026-03-26

### Сделано
- Убраны секреты из git-версии (`.env` удален, добавлен `.env.example`).
- Вынесены DB-конфиги в env в GUI/CLI/desktop.
- Исправлены и выровнены Go-модули (`go mod tidy`, сборка проходит).
- Добавлен `trading-core` с базовыми типами и risk manager.
- `FinamBroker` переведен на реальные REST-вызовы (JWT/details/orders/cancel/portfolio).
- Улучшен WS: reconnect + resubscribe, убрана автоподписка SBER.
- Добавлены режимы manual/auto/paper + базовый автоцикл.
- Обновлена AI-панель в GUI:
  - переименован заголовок,
  - убран ручной ticker input,
  - добавлен выбор инструмента по поисковому запросу.

### Важно
- Ранее утекший токен считать скомпрометированным, перевыпустить в Finam.
- Для боевого режима нужен локальный `moex-trader-gui/.env` с валидными значениями.

### Следующий шаг
- Smoke-check боевого контура: auth -> accounts -> ws -> tiny order -> cancel.


---

## FILE: docs/MIYA_MUHLYA_ROLES.md

# Мухля и Мия: роли, приоритеты и связка

Документ фиксирует **разделение ответственности** между торговым контуром (**Мухля**, `muhlya-headless`) и контуром наблюдения (**Мия**, скрипты на VPS без исполнения сделок). Цель — не дублировать решения и не конкурировать за приоритет там, где рискует только один слой.

**Образ:** Мухля — **исполнитель** (ордера, риск, лимиты). Мия — **аналитик и штурман**: даёт срез рынка и контекст для решений, но не подменяет торговый контур и не выставляет заявки.

## Краткая таблица

| | Мухля | Мия |
|---|--------|-----|
| **Роль** | Исполнение, риск, брокер Finam | Аналитика и навигация контекстом: срезы рынка, отчёты, MOEX ISS / стакан, без торговых поручений |
| **Главный приоритет** | Сохранение счёта и соблюдение лимитов | Стабильный, проверяемый контекст и артефакты |
| **Горизонт времени** | Секунды–минуты (автоцикл) | Минуты–часы (таймеры, latest-файлы) |

## Зоны ответственности

### Мухля (обязательно здесь)

- Решения **вход / выход**, размер позиции, **TP / SL / profit lock**, дневные и прочие лимиты риска.
- Взаимодействие с **Finam** (заявки, портфель, журнал сделок).
- Ранжирование кандидатов на покупку (агент, confidence, исполнимость лота, история по тикеру в Postgres, мягкие штрафы за reject и **демпф обучения** при малом числе сделок на тикер — см. `AGENT_LEARNING_*`, `AGENT_PNL_SCORE_*`, `AGENT_EXIT_LEARNING_*` в `.env.example`).
- Запись **снимка портфеля** для Мии: `broker_positions_snapshot.json` (см. `AGENT_BROKER_POSITION_SNAPSHOT_*`), чтобы Мия не дёргала брокера ради отчёта.

### Мия (обязательно здесь)

- Периодические **отчёты** в `/var/log/muhlya-reports/` (`miya-report.sh`, intraday/daily).
- **Онлайн-слой MOEX** без секретов брокера: свечи/агрегаты, при необходимости **стакан** (ISS), итог в **`miya_online_signal.env`**.
- Чтение **свежего** `broker_positions_snapshot.json` для «правды по Finam» в отчёте; иначе — опора на Postgres с пометкой о расхождении.
- **Не** выставляет заявки и **не** подменяет риск-менеджер.

### Граница (как «помогают друг другу»)

- Мия поставляет **контекст рынка** (например `STATE`, `UP_RATIO_PCT`, **дневной** `DAILY_*`, `DATA_STATUS`, стакан).
- Мухля **сама решает**, как этот контекст учитывать — через настройки вида **`AGENT_MIA_ONLINE_*`** (мягкий сдвиг гейтов). Мия не должна становиться «вторым трейдером» без лимитов Мухли.
- **Вне сессии акций:** intraday-слой онлайн-сигнала в сдвиге гейтов может **ослабляться** (`AGENT_MIA_ONLINE_STOCK_CLOSED_INTRADAY_SCALE`), чтобы ночной/неликвидный шум `STATE`/`UP_RATIO` не тянул пороги так же, как в основную сессию; дневной слой (`DAILY_*`, вес `AGENT_MIA_ONLINE_DAILY_GATE_WEIGHT`) при этом не режется этим коэффициентом.

## Приоритеты задач (порядок важности)

1. **Надёжность и риск (Мухля)** — выше любого «красивого сигнала».
2. **Качество данных Мии** — осмысленный `DATA_STATUS`, консистентный онлайн-сигнал; иначе Мухля обязана **ослаблять или игнорировать** влияние сигнала.
3. **Наблюдаемость** — отчёты и `runtime_events`, чтобы понимать «почему не вошли» / «почему вышли».
4. **Расширение сигналов Мии** (алерты, доп. метрики) — после стабилизации пунктов 1–3.

## Контракт данных (рукопожатие)

| Артефакт | Кто пишет | Кто читает | Примечание |
|----------|-----------|------------|------------|
| `miya_online_signal.env` | Мия (`miya-online-watch`) | Мухля (`AGENT_MIA_ONLINE_SIGNAL_PATH`) | Intraday (`STATE`, …) + **дневной срез** (`DAILY_STATE`, …) + **`MIYA_LIQUIDITY_*`** + **`MIYA_WATCH_TICKERS`** (тот же список, что API: при `AGENT_MIA_WATCH_FIRST_ENABLED` авто сначала рассматривает только эти акции, иначе — полный универсум). Не приказ «купить». |
| `broker_positions_snapshot.json` | Мухля | Мия (`miya-report`, `ONLINE_MIA_BROKER_SNAPSHOT_*`) | Свежесть задаётся порогом секунд |
| `agent_exit_hints.env` | Мия (если включено) | Мухля (`AGENT_MIA_EXIT_HINTS_PATH`) | Подсказки по выходу, не замена SL/TP |
| Postgres `runtime_events`, `trades` | Мухля (и БД) | Мия (отчёты), человек | Срезы: см. `scripts/vps/vps-slice-4h.sh`, `docs/VPS_SSH_LINUX_BASICS.md` |

## Чего избегать

- Дублировать в Мухле тяжёлый сбор онлайн-MOEX там, где уже есть Мия — расхождения и лишняя нагрузка.
- Переносить в Мию логику «ожидаемой доходности по бумагам» без отдельного контура (**trading-lab** / политика / ревью) — размывает ответственность.
- Менять оба контура и `.env` **одновременно** без среза «до/после» — невозможно понять эффект.

## См. также

- `docs/VPS_SSH_LINUX_BASICS.md` — systemd, SSH, срезы БД.
- `docs/MIYA_ORDERBOOK.md` — стакан MOEX ISS vs Finam.
- `moex-trader-gui/.env.example` — `AGENT_MIA_*`, снимок портфеля, онлайн-сигнал.

---

## FILE: CHAT_HANDOFF_TEMPLATE.md

# CHAT HANDOFF TEMPLATE

Скопируй этот шаблон в новый чат и заполни 5 блоков ниже.

---

## 1) Контекст проекта
- Проект: `Trading`
- Цель: **торговый ассистент** и терминал MOEX + Finam (manual/auto/paper), GUI + CLI; продуктовый ориентир — **рост кэша** на счёте Finam за счёт операций с **ценными бумагами**, доступными счёту, в рамках риск-менеджмента (прибыль не гарантируется).
- Единый файл контекста: `CHAT_CONTEXT_PACKET.md`

## 2) Текущий статус
- Что уже сделано:
  - <кратко 3-7 пунктов>
- Что сейчас в работе:
  - <1-3 пункта>
- Что заблокировано:
  - <если есть>

## 3) Конкретная задача на этот чат
- Нужно сделать:
  1. <шаг 1>
  2. <шаг 2>
  3. <шаг 3>

## 4) Ограничения и правила
- Язык ответа: русский
- Правило пользователя: писать код только после команды `opengo`
- Безопасность: не коммитить секреты, `.env` не в git

## 5) Проверка результата
- После изменений:
  - `go mod tidy` в затронутых модулях
  - сборка: `go build ./...`
  - при необходимости: `npm run build` в `moex-trader-gui/frontend`
  - проверить живые логи GUI runtime (`auto.order_*`, `auto.no_order`, `skip_mtf_*`, `blocked_by_*`)
  - обновить `NEXT_STEPS.md` и `SESSION_LOG.md`
  - перезаписать `CHAT_CONTEXT_PACKET.md` командой `.\scripts\build-chat-context.ps1`

---

Если в новом чате контекст неполный, сначала прочитай:
1. `CHAT_CONTEXT_PACKET.md` (целиком)
2. файлы, указанные в разделе "Текущая задача"

