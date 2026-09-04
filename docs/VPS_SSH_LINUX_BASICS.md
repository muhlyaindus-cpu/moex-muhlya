# VPS (Timeweb Cloud): PowerShell, SSH и базовые команды Linux

Контекст проекта **Trading (MOEX + Finam)**: на VDS крутится **Мухля** (`muhlya-headless` через **systemd**), **PostgreSQL** — в **Docker**. Рабочий пользователь на сервере: **`traderbot`** (не `root`; вход по SSH-ключу, пароль отключён по hardening).

---

## 1. Где что выполняется

| Где | Что это |
|-----|--------|
| **PowerShell на Windows** (твой ПК) | Клиент SSH/SCP: подключение к VDS и копирование файлов. |
| **Сессия после `ssh …`** | Обычный **bash на Ubuntu** на VDS — дальше все команды ниже **Linux**, не PowerShell. |

Серийная консоль в панели Timeweb и окно PowerShell — разные вещи; удобнее привычный **PowerShell + `ssh`**.

---

## 2. Подключение к нашему серверу из PowerShell

Используем пользователя **`traderbot`** и IP продакшн-VPS проекта:

```powershell
ssh traderbot@YOUR_VPS_HOST
```

Быстрая команда «перехода на сервер» (можно запускать из любой папки PowerShell):

```powershell
ssh traderbot@YOUR_VPS_HOST
```

Дальше все пункты 3–7 — **уже на сервере** (в этой SSH-сессии).

Если в панели Timeweb указано другое имя хоста (`*.twc1.net` и т.д.) — это тот же сервер; подключение по **IP** остаётся актуальным.

Вход под **`root`** в нашем сценарии **не нужен** (и по политике безопасности отключён после настройки).

---

## 3. Навигация к каталогу проекта

Типовые пути:

```bash
cd /opt/trading/app/moex-trader-gui    # корень деплоя GUI/headless, docker-compose, скрипты VPS
pwd
ls
ls -la
cd ..
```

---

## 4. Просмотр файлов

```bash
cat файл                 # показать целиком
less файл                # пролистывать по экрану
head -n 50 файл          # первые 50 строк
tail -n 50 файл          # последние 50 строк
tail -f лог.log          # «онлайн» хвост лога (Ctrl+C — выход)
```

Для **Мухли** логи чаще смотрят через **journald**, не через файл в каталоге:

```bash
sudo journalctl -u muhlya -n 120 --no-pager
sudo journalctl -u muhlya -f
```

---

## 5. Файлы и папки (осторожно на проде)

```bash
cp исходный целевой
cp -r dir1 dir2
mv старое_имя новое_имя
rm файл
rm -r папка              # опасно: рекурсивное удаление
mkdir -p папка
```

Конфиги с секретами (**`.env`**) на сервере не коммитятся в git; правки — через редактор на сервере или выгрузка с ПК с осторожностью.

---

## 6. Поиск

```bash
find . -name "имя"
grep "строка" файл
grep -R "строка" .
```

---

## 7. Наш стек на VDS (не Python/Node «вручную»)

Мухля в бою запускается как **системный сервис**, не как `python main.py`:

```bash
sudo systemctl status muhlya --no-pager
sudo systemctl restart muhlya
```

Бинарник обычно лежит здесь:

```text
/opt/trading/app/moex-trader-gui/muhlya-headless
```

**PostgreSQL** — в **Docker** (контейнер вроде `moex_trading_db`):

```bash
docker compose -f /opt/trading/app/moex-trader-gui/docker-compose.yml ps
docker exec -it moex_trading_db psql -U trader -d moex_trading -c "\dt"
```

Обновление бинарника с Windows: **`scp`** в `/tmp`, затем `install` и перезапуск — см. раздел «PowerShell быстрые команды» в `NEXT_STEPS.md`.

Локальная разработка GUI на Windows: **`wails dev`** в репозитории — это **не** то, что выполняется на VDS в headless-режиме.

---

## 8. Права и владельцы

```bash
ls -la
chmod 755 файл
sudo chown user:group файл
```

Для скриптов бэкапа и каталогов дампов смотри `moex-trader-gui/scripts/vps/` и `NEXT_STEPS.md`.

---

## 9. Полный список команд Linux

Закрытого набора нет. Под задачи «логи, конфиг, рестарт сервиса, Docker, БД» — см. **`NEXT_STEPS.md`**, раздел **«Команды дежурного (VPS)»**.

---

## 10. Поддержка Timeweb

Если речь про саму панель Timeweb Cloud, биллинг или инфраструктуру провайдера: [личный кабинет — поддержка](https://timeweb.cloud/my/support), почта **info@timeweb.cloud**.

Развёртывание **нашего** приложения и `systemd`/`docker` — по документации в репозитории и договорённостям в команде; это не «типовое приложение» из мастера Timeweb «Приложения».

---

## 11. Быстрые срезы работы Мухли (20 минут)

Полезно запускать после изменения `.env`/рестарта, чтобы сравнить «до/после»:

```bash
docker exec moex_trading_db psql -U trader -d moex_trading -c "SELECT step, count(*) FROM runtime_events WHERE created_at >= now() - make_interval(mins => 20) GROUP BY step ORDER BY count(*) DESC;"
```

```bash
docker exec moex_trading_db psql -U trader -d moex_trading -c "SELECT status, count(*) FROM trades WHERE created_at >= now() - make_interval(mins => 20) GROUP BY status ORDER BY count(*) DESC LIMIT 20;"
```

Как читать:

- `runtime_events`: показывает, чем реально занят цикл (`auto.cycle`, `auto.no_order`, `auto.skip_*`, `order.*`).
- `trades by status`: показывает качество исполнения (`placed`) и причины отказов (`rejected:*`).
- Если доминирует `order.futures_finam_getasset_reject` или `rejected: finam asset: cached miss` — шумит фьючерсный контур (обычно помогает временно `AGENT_AUTO_INSTRUMENT_MODE=stocks`).
- Если доминирует `auto.skip_low_balance` / `auto.low_balance_manage_only` — входы паузятся из-за порога баланса.
- Для честного сравнения не дёргать сервис часто: после рестарта подождать 15–20 минут и только потом смотреть срез.

---

## 12. Мини-чеклист дежурного (5 пунктов)

Использовать перед сном/после деплоя/после смены `.env`.

1) Сервис жив:

```bash
systemctl status muhlya --no-pager
```

2) Последние логи без аварий:

```bash
sudo journalctl -u muhlya -n 80 -l --no-pager
```

3) БД контейнер в норме:

```bash
docker compose -f /opt/trading/app/moex-trader-gui/docker-compose.yml ps
```

4) Срез `runtime_events` за 20 минут:

```bash
docker exec moex_trading_db psql -U trader -d moex_trading -c "SELECT step, count(*) FROM runtime_events WHERE created_at >= now() - make_interval(mins => 20) GROUP BY step ORDER BY count(*) DESC;"
```

5) Срез `trades` за 20 минут:

```bash
docker exec moex_trading_db psql -U trader -d moex_trading -c "SELECT status, count(*) FROM trades WHERE created_at >= now() - make_interval(mins => 20) GROUP BY status ORDER BY count(*) DESC LIMIT 20;"
```

Мини-оценка:

- `placed` есть, а `rejected:*` не доминирует — рабочий контур в порядке.
- если лидируют `futures_finam_getasset_reject` / `cached miss` — временно `AGENT_AUTO_INSTRUMENT_MODE=stocks`.
- если лидируют `auto.skip_low_balance` / `auto.low_balance_manage_only` — слишком высокий порог баланса для входов.

---

*Источник идеи: справка Timeweb Cloud; текст адаптирован под проект Trading (Мухля, `traderbot`, пути, стек).*

---

## 13. Мия (файловые отчёты без Telegram/LLM)

Назначение: дежурные срезы по Мухле в файлы на сервере, без токенов и без внешних мессенджеров.

Артефакты в репозитории:

- `moex-trader-gui/scripts/vps/miya-report.sh`
- `moex-trader-gui/scripts/vps/miya-online-watch.sh` — онлайн-снапшот по свечам (без архива, перезапись latest)
- `moex-trader-gui/scripts/vps/deploy-miya-online-watch.ps1` — автодеплой онлайн-контура с Windows (scp + ssh + enable timer)
- `moex-trader-gui/scripts/vps/phase_summary.sh` — разовая сводка по фазам сессии и A/B/C из Postgres
- `moex-trader-gui/scripts/vps/systemd/miya-report.service`
- `moex-trader-gui/scripts/vps/systemd/miya-report.timer`
- `moex-trader-gui/scripts/vps/systemd/miya-online-watch.service`
- `moex-trader-gui/scripts/vps/systemd/miya-online-watch.timer`

Что делает:

- Берёт данные из `runtime_events`, `trades`, `positions` + статус `muhlya.service`.
- Пишет отчёты в `/var/log/muhlya-reports/` (`.md` и `.json`).
- Добавляет «Утренний чек-лист (45m)» и свечной блок по `market_data` (импульс + лидеры роста/снижения), если данные свечей поступают.
- Отдельный онлайн-контур свечей (`miya-online-watch`) обновляет latest-снимок каждую минуту:
  - `/var/log/muhlya-reports/online_candles_latest.md`
  - `/var/log/muhlya-reports/online_candles_latest.json`
  - `/var/log/muhlya-reports/miya_online_signal.env`
  - `STATE` в сигнале: `bullish | neutral | bearish | no_data` (`no_data` — когда валидных свечей в окне недостаточно)
  - при пустом short/long окне в активную сессию включается fallback-окно (`ONLINE_MIA_CANDLE_FALLBACK_MIN`), статус `DATA_STATUS=fallback_active`
- Таймер срабатывает каждые 30 минут, но скрипт сам дросселирует запись:
  - в активное окно (будни 10:00–18:40 MSK): раз в 60 минут;
  - в фазу открытия (первые 45 минут активного окна): раз в 15 минут;
  - вне окна: раз в 120 минут.
- Cadence регулируется env-параметрами:
  - `ONLINE_MIA_ACTIVE_EVERY_MIN`, `ONLINE_MIA_OFFHOURS_EVERY_MIN`
  - `ONLINE_MIA_OPENING_EVERY_MIN`, `ONLINE_MIA_OPENING_WINDOW_MIN`

Включение на VPS (один раз):

```bash
cd /opt/trading/app/moex-trader-gui
chmod +x scripts/vps/miya-report.sh
chmod +x scripts/vps/miya-online-watch.sh
sudo cp scripts/vps/systemd/miya-report.service /etc/systemd/system/
sudo cp scripts/vps/systemd/miya-report.timer /etc/systemd/system/
sudo cp scripts/vps/systemd/miya-online-watch.service /etc/systemd/system/
sudo cp scripts/vps/systemd/miya-online-watch.timer /etc/systemd/system/
sudo mkdir -p /var/log/muhlya-reports
sudo chown traderbot:traderbot /var/log/muhlya-reports
sudo systemctl daemon-reload
sudo systemctl enable --now miya-report.timer
sudo systemctl enable --now miya-online-watch.timer
```

Тот же набор unit’ов одной командой (после копирования `.service` / `.timer` в `/tmp` на сервере, например через `scp`):

```bash
chmod +x /tmp/install-miya-online-watch-systemd-remote.sh
sudo /tmp/install-miya-online-watch-systemd-remote.sh
```

Скрипт лежит в репозитории: `moex-trader-gui/scripts/vps/install-miya-online-watch-systemd-remote.sh` (ожидает `/tmp/miya-online-watch.service` и `/tmp/miya-online-watch.timer`).

Быстрый вариант с Windows (автодеплой):

```powershell
cd "C:\Users\MyProgram\Trading\moex-trader-gui"
powershell -ExecutionPolicy Bypass -File .\scripts\vps\deploy-miya-online-watch.ps1 -HostName YOUR_VPS_HOST -UserName traderbot
```

> Скрипт попросит пароль SSH/`sudo`, сам загрузит файлы и включит `miya-online-watch.timer` (используется `ssh -tt` для интерактивного `sudo`).

Проверка:

```bash
systemctl status miya-report.timer --no-pager
systemctl status miya-online-watch.timer --no-pager
sudo journalctl -u miya-report.service -n 80 --no-pager
sudo journalctl -u miya-online-watch.service -n 80 --no-pager
ls -la /var/log/muhlya-reports
```

Ручной запуск:

```bash
sudo systemctl start miya-report.service
sudo systemctl start miya-online-watch.service
```

Операционный анти-спам чек (после включения «оживления входов»):

```bash
# 1) Короткий фазовый срез
/opt/trading/app/moex-trader-gui/scripts/vps/phase_summary.sh 1

# 2) Быстрый статус по last 5m
docker exec -i moex_trading_db psql -U trader -d moex_trading -P pager=off -c "
SELECT step, count(*) FROM runtime_events
WHERE created_at >= now() - interval '5 minutes'
  AND step IN ('auto.order_buy_try','auto.no_order','auto.skip_pending_order','auto.skip_buy_affordable')
GROUP BY step ORDER BY count(*) DESC;"
```

Полезные env-рычаги в бою:

- `AGENT_AUTO_BUDGET_PERCENT` и `AGENT_CASH_USAGE_RESERVE` — насколько плотно использовать кэш.
- `AGENT_BUY_CONF_THRESHOLD` / `AGENT_SHORT_CONF_THRESHOLD` — частота входов.
- `AGENT_MIN_NUMTRADES` / `AGENT_MIN_TURNOVER_RUB` — жёсткость ликвидностного фильтра.
- `AGENT_OPEN_LONG_BIAS`, `AGENT_OPEN_BUY_GATE_DELTA`, `AGENT_OPEN_SHORT_GATE_PENALTY`, `AGENT_OPEN_MAX_BUY_ATTEMPTS_BONUS` — мягкий long-акцент Мухли на открытии рынка.
- `ONLINE_MIA_CANDLE_SHORT_MIN`, `ONLINE_MIA_CANDLE_LONG_MIN`, `ONLINE_MIA_BULLISH_THRESHOLD_PCT`, `ONLINE_MIA_BEARISH_THRESHOLD_PCT` — чувствительность онлайн-свечного радара Мии.
- `ONLINE_MIA_CANDLE_MIN_POINTS`, `ONLINE_MIA_CANDLE_FALLBACK_MIN`, `ONLINE_MIA_CANDLE_USE_FALLBACK_IN_ACTIVE` — устойчивость к редкому потоку свечей (расширенное окно fallback при пустом long-window; **без** старого окна пн–пт в скрипте — поля `ONLINE_MIA_ACTIVE_START_HHMM` / `ACTIVE_SESSION` сняты).
- `AGENT_MIA_ONLINE_SIGNAL_PATH`, `AGENT_MIA_ONLINE_HINTS_ENABLED`, `AGENT_MIA_ONLINE_BUY_GATE_MAX_DELTA`, `AGENT_MIA_ONLINE_SHORT_GATE_MAX_DELTA`, `AGENT_MIA_ONLINE_DAILY_GATE_WEIGHT` — связка «Мия -> Мухля» (мягкая подстройка buy/short gate по online + daily).
- `AGENT_MIA_ONLINE_STOCK_CLOSED_INTRADAY_SCALE` (0..1) — когда акции по MOEX **закрыты** (`SBER` не в `T`), ослабить **только intraday**-слой Мии в сдвиге гейтов; daily (`DAILY_*`) не умножается. `1` = отключить ослабление.
- `AGENT_LEARNING_CONF_WEIGHT`, `AGENT_LEARNING_MIN_TRADES_PER_TICKER` — вклад «обучения» из последних `trades` в confidence; до порога по числу сделок на тикер вклад **линейно масштабируется** (анти-шум).
- `AGENT_PNL_SCORE_WEIGHT`, `AGENT_PNL_SCORE_MIN_TRADES` — мягкий PnL-score по тикеру; до порога по числу строк в разборе — масштабирование.
- `AGENT_EXIT_LEARNING_PATH`, `AGENT_EXIT_LEARNING_MIN_SAMPLES` — JSON-память выходов по тикеру; множитель TP из файла не трогает выход, пока мало сэмплов.

---

## 14. Словарь терминов (RU -> что это значит)

Ниже короткая расшифровка англоязычных шагов/статусов, которые часто встречаются в логах и отчётах.

- `auto.order_buy_try` — Мухля пытается отправить заявку на покупку (вход в long или cover short).
- `auto.order_short_open_try` — попытка открыть short-позицию (продажа из flat).
- `auto.order_sell_try` — попытка продажи для закрытия long.
- `auto.no_order` — в этом цикле авто решение было, но ордер не отправлен (фильтры/гейты/паузы).
- `auto.session_tick` — служебный тик автоцикла (heartbeat логики по фазе сессии).
- `auto.session_phase` — смена фазы рынка (`open`, `mid`, `close`, `offhours`).
- `auto.session_gate_prefilter` — предфильтр по сессии/`TRADINGSTATUS`: закрытые тикеры отрезаны до попытки ордера.
- `auto.skip_session_closed` — цикл пропущен, т.к. биржа/секция закрыта.
- `auto.skip_buy_global_pending` — новые покупки временно не отправляются, пока есть pending buy-заявки.
- `auto.skip_buy_cash_pause` — действует короткая пауза после ошибок по кэшу (`[156]`), покупки приторможены.
- `auto.skip_buy_affordable` — кандидат отсеян: доступного кэша не хватает на допустимый объём.
- `auto.skip_short_global_pause` — short-open временно заморожен после проблем `[157]/[257]`.
- `order.short_open_global_pause` — включена глобальная пауза на новые short-open.
- `auto.exit_take_profit` — выход из позиции по цели прибыли (TP).
- `auto.exit_stop_loss` — выход по ограничению убытка (SL).
- `auto.exit_stale_timeout` — выход по «протуханию»: долго в позиции без нормальной динамики.
- `auto.exit_profile_switched` — переключение профиля выхода (`conservative` / `adaptive` / `auto`).
- `auto.exit_tp_tuned` — TP по тикеру автоматически подстроен (память Мухли + hints Мии).
- `order.buy_placed` / `order.sell_placed` — заявка успешно выставлена у брокера.
- `order.buy_failed` / `order.sell_failed` — заявка не прошла (см. текст причины).
- `rejected` — заявка отклонена; причина всегда в хвосте статуса (`rejected: ...`).
- `placed` — заявка принята брокером и выставлена в рынок (ещё не факт исполнения).
- `executed` — сделка исполнена.
- `pending order` — заявка ещё активна/в работе, не завершена терминальным статусом.
- `outside trading session` — заявка отклонена из-за времени торгов/статуса сессии.
- `TRADINGSTATUS=T` — инструмент торгуется сейчас.
- `TRADINGSTATUS=N` — инструмент сейчас не торгуется.
- `DIFF_HOURS` — разница часовых поясов между опорными зонами (в наших проверках обычно `NSK - MSK`); для Новосибирска и Москвы это `+4`.
- `[156] Money shortage` — недостаточно денег под заявку.
- `[157] Equities shortage` — недостаточно бумаг для продажи/закрытия.
- `[257] qualification` — ограничение по квалификации для инструмента.

Мини-подсказка по чтению:

- Если растёт `auto.order_buy_try`, но нет `placed` — смотрим причины `rejected`.
- Если доминирует `outside trading session` — это про время/статус рынка, не про качество сигналов.
- Если доминирует `[156]` — проблема в доступном кэше/размере заявки.
- Если доминирует `[157]` — проблема в доступном объёме позиции на sell/short.
