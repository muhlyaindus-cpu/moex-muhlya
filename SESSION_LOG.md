# SESSION LOG

Журнал коротких сессий. Добавляем новые записи сверху.

---

## 2026-09-04

### Публикация на GitHub (public `moex-muhlya`)
- Репозиторий готовится как **public** для совместных улучшений; IP VPS в docs/логах заменён на плейсхолдер `YOUR_VPS_HOST`.
- Расширен корневой `.gitignore` (бинари `muhlya-headless*`, `*.linux`, артефакты lab); удалён мусорный файл `4)`.
- Установлен GitHub CLI (`winget` → `GitHub.cli`); `.env` с секретами **не** попадает в git.
- `NEXT_STEPS.md`: оперативный фокус **2026-09-04** + архив **2026-04-24**.

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

