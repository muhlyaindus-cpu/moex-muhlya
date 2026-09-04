#!/usr/bin/env bash
# Мия: периодический отчёт по состоянию Мухли без внешних токенов.
# Источники: Postgres (runtime_events/trades/positions) + systemctl status.
#
# Пишет:
#   - markdown: /var/log/muhlya-reports/intraday_YYYY-MM-DD_HHMM.md
#   - json:     /var/log/muhlya-reports/intraday_YYYY-MM-DD_HHMM.json
#   - daily:    /var/log/muhlya-reports/daily_YYYY-MM-DD.md (перезаписывается)
#
# Частота записи регулируется по биржевым часам (Europe/Moscow):
#   - в рабочее окно: раз в ONLINE_MIA_ACTIVE_EVERY_MIN (по умолчанию 60 мин)
#   - вне окна:       раз в ONLINE_MIA_OFFHOURS_EVERY_MIN (по умолчанию 120 мин)
#
# Запускать через systemd timer (см. scripts/vps/systemd).

set -euo pipefail

PG_CONTAINER_NAME="${PG_CONTAINER_NAME:-moex_trading_db}"
PGUSER="${PGUSER:-trader}"
PGDATABASE="${PGDATABASE:-moex_trading}"

REPORTS_DIR="${ONLINE_MIA_REPORTS_DIR:-/var/log/muhlya-reports}"
BROKER_SNAPSHOT_PATH="${ONLINE_MIA_BROKER_SNAPSHOT_PATH:-${REPORTS_DIR}/broker_positions_snapshot.json}"
BROKER_SNAPSHOT_STALE_SEC="${ONLINE_MIA_BROKER_SNAPSHOT_STALE_SEC:-900}"
STATE_FILE="${ONLINE_MIA_STATE_FILE:-${REPORTS_DIR}/.miya-state}"

TZ_MSK="${ONLINE_MIA_TZ:-Europe/Moscow}"
ACTIVE_EVERY_MIN="${ONLINE_MIA_ACTIVE_EVERY_MIN:-60}"
OFFHOURS_EVERY_MIN="${ONLINE_MIA_OFFHOURS_EVERY_MIN:-120}"
OPENING_EVERY_MIN="${ONLINE_MIA_OPENING_EVERY_MIN:-15}"
OPENING_WINDOW_MIN="${ONLINE_MIA_OPENING_WINDOW_MIN:-45}"
REPORT_RETENTION_DAYS="${ONLINE_MIA_REPORT_RETENTION_DAYS:-14}"

# Условно рабочее окно для рынка акций MOEX (локально для Москвы).
# Это не торговое правило, а только контроль частоты отчётов.
ACTIVE_START_HM="${ONLINE_MIA_ACTIVE_START_HM:-10:00}"
ACTIVE_END_HM="${ONLINE_MIA_ACTIVE_END_HM:-18:40}"

mkdir -p "$REPORTS_DIR"

if ! docker inspect "$PG_CONTAINER_NAME" >/dev/null 2>&1; then
  echo "miya: docker/container unavailable: $PG_CONTAINER_NAME" >&2
  exit 1
fi

sql_scalar() {
  local sql="$1"
  docker exec "$PG_CONTAINER_NAME" psql -U "$PGUSER" -d "$PGDATABASE" -t -A -c "$sql" | tr -d '\r'
}

sql_raw_lines() {
  local sql="$1"
  docker exec "$PG_CONTAINER_NAME" psql -U "$PGUSER" -d "$PGDATABASE" -t -A -c "$sql" | tr -d '\r'
}

# Снимок портфеля пишет Мухля (Finam), без дополнительных запросов к брокеру со стороны Мии.
read_broker_snapshot_fields() {
  local snap="$1"
  broker_snapshot_open="0"
  broker_snapshot_age="999999"
  broker_snapshot_tickers=""
  if [[ ! -f "$snap" ]]; then
    return 1
  fi
  local line
  if ! line="$(SNAP="$snap" python3 -c 'import json, os, time
p = os.environ.get("SNAP", "") or ""
try:
    age = int(time.time() - os.path.getmtime(p))
    with open(p, encoding="utf-8") as f:
        d = json.load(f)
    oc = int(d.get("open_count") or 0)
    ts = [x.get("ticker", "") for x in (d.get("positions") or []) if int(x.get("quantity") or 0) != 0]
    print(str(oc) + "\t" + str(age) + "\t" + ", ".join(ts))
except Exception:
    print("0\t999999\t")
' 2>/dev/null)"; then
    return 1
  fi
  IFS=$'\t' read -r broker_snapshot_open broker_snapshot_age broker_snapshot_tickers <<<"$line"
  if [[ ! "${broker_snapshot_open}" =~ ^[0-9]+$ ]]; then
    broker_snapshot_open="0"
  fi
  if [[ ! "${broker_snapshot_age}" =~ ^[0-9]+$ ]]; then
    broker_snapshot_age="999999"
  fi
  return 0
}

msk_now="$(TZ="$TZ_MSK" date '+%Y-%m-%d %H:%M:%S')"
msk_date="$(TZ="$TZ_MSK" date '+%Y-%m-%d')"
msk_time_hm="$(TZ="$TZ_MSK" date '+%H:%M')"
weekday="$(TZ="$TZ_MSK" date '+%u')" # 1..7
epoch_now="$(date +%s)"

hm_to_min() {
  local hm="$1"
  local hh="${hm%:*}"
  local mm="${hm#*:}"
  if [[ -z "$hh" || -z "$mm" ]]; then
    echo 0
    return
  fi
  echo $((10#$hh * 60 + 10#$mm))
}

active_window=0
if [[ "$weekday" -le 5 ]] && [[ "$msk_time_hm" > "$ACTIVE_START_HM" || "$msk_time_hm" == "$ACTIVE_START_HM" ]] && [[ "$msk_time_hm" < "$ACTIVE_END_HM" || "$msk_time_hm" == "$ACTIVE_END_HM" ]]; then
  active_window=1
fi

every_min="$OFFHOURS_EVERY_MIN"
if [[ "$active_window" -eq 1 ]]; then
  every_min="$ACTIVE_EVERY_MIN"
fi
open_window=0
if [[ "$active_window" -eq 1 ]]; then
  now_min="$(hm_to_min "$msk_time_hm")"
  start_min="$(hm_to_min "$ACTIVE_START_HM")"
  if (( now_min >= start_min && now_min < start_min + OPENING_WINDOW_MIN )); then
    open_window=1
    every_min="$OPENING_EVERY_MIN"
  fi
fi
every_sec=$((every_min * 60))

last_epoch=0
if [[ -f "$STATE_FILE" ]]; then
  last_epoch="$(awk -F= '/^last_written_epoch=/{print $2}' "$STATE_FILE" 2>/dev/null || echo 0)"
fi
if [[ -z "$last_epoch" ]]; then
  last_epoch=0
fi

if (( epoch_now - last_epoch < every_sec )); then
  echo "miya: skip report (throttle ${every_min}m)"
  exit 0
fi

service_active="$(systemctl is-active muhlya.service 2>/dev/null || true)"
service_status_line="$(systemctl status muhlya --no-pager 2>/dev/null | awk 'NR==1{print; exit}' || true)"

new_trades_24h="$(sql_scalar "SELECT count(1) FROM trades WHERE created_at >= now() - interval '24 hours';")"
placed_24h="$(sql_scalar "SELECT count(1) FROM trades WHERE created_at >= now() - interval '24 hours' AND status='placed';")"
executed_24h="$(sql_scalar "SELECT count(1) FROM trades WHERE created_at >= now() - interval '24 hours' AND status='executed';")"
reject_24h="$(sql_scalar "SELECT count(1) FROM trades WHERE created_at >= now() - interval '24 hours' AND status LIKE 'rejected:%';")"
open_positions_db="$(sql_scalar "SELECT count(1) FROM positions WHERE quantity <> 0;")"
open_positions="${open_positions_db}"
positions_source="postgres"
broker_snapshot_open="0"
broker_snapshot_age="999999"
broker_snapshot_tickers=""
if read_broker_snapshot_fields "$BROKER_SNAPSHOT_PATH"; then
  if [[ "${broker_snapshot_age:-999999}" -lt "$BROKER_SNAPSHOT_STALE_SEC" ]]; then
    open_positions="${broker_snapshot_open}"
    positions_source="finam_snapshot"
  fi
fi

no_order_24h="$(sql_scalar "SELECT count(1) FROM runtime_events WHERE created_at >= now() - interval '24 hours' AND step='auto.no_order';")"
skip_low_balance_24h="$(sql_scalar "SELECT count(1) FROM runtime_events WHERE created_at >= now() - interval '24 hours' AND step='auto.skip_low_balance';")"
low_balance_mode_24h="$(sql_scalar "SELECT count(1) FROM runtime_events WHERE created_at >= now() - interval '24 hours' AND step='auto.low_balance_manage_only';")"
stale_timeout_24h="$(sql_scalar "SELECT count(1) FROM runtime_events WHERE created_at >= now() - interval '24 hours' AND step='auto.exit_stale_timeout';")"
session_closed_reject_24h="$(sql_scalar "SELECT count(1) FROM trades WHERE created_at >= now() - interval '24 hours' AND status LIKE 'rejected: outside trading session%';")"

top_rejects_raw="$(sql_scalar "SELECT regexp_replace(status, '[[:space:]]+', ' ', 'g') || ' | ' || count(1) FROM trades WHERE created_at >= now() - interval '24 hours' AND status LIKE 'rejected:%' GROUP BY status ORDER BY count(1) DESC LIMIT 5;")"
top_steps_raw="$(sql_scalar "SELECT step || ' | ' || count(1) FROM runtime_events WHERE created_at >= now() - interval '24 hours' GROUP BY step ORDER BY count(1) DESC LIMIT 8;")"

# Фазы сессии (см. auto.session_tick / sessionPhaseMoscow в app.go), границы времени — Europe/Moscow.
phase_ticks_24h="$(sql_raw_lines "SELECT phase || ' | ' || cnt::text FROM (SELECT coalesce(substring(details from '^phase=([^ ]+)'), '?') AS phase, count(1) AS cnt FROM runtime_events WHERE created_at >= now() - interval '24 hours' AND step = 'auto.session_tick' GROUP BY 1) s ORDER BY phase;")"
abtest_phase_24h="$(sql_raw_lines "SELECT phase || ' | A=' || sum_a::text || ' B=' || sum_b::text || ' C=' || sum_c::text FROM (SELECT coalesce(substring(details from '^phase=([^ ]+)'), '?') AS phase, sum(coalesce(substring(details from 'A=([0-9]+)')::int, 0)) AS sum_a, sum(coalesce(substring(details from 'B=([0-9]+)')::int, 0)) AS sum_b, sum(coalesce(substring(details from 'C=([0-9]+)')::int, 0)) AS sum_c FROM runtime_events WHERE created_at >= now() - interval '24 hours' AND step = 'auto.abtest_cycle' GROUP BY 1) s ORDER BY phase;")"
order_buy_try_phase_24h="$(sql_raw_lines "SELECT phase || ' | ' || cnt::text FROM (SELECT coalesce(substring(details from 'phase=(open|mid|close|offhours)'), '?') AS phase, count(1) AS cnt FROM runtime_events WHERE created_at >= now() - interval '24 hours' AND step = 'auto.order_buy_try' GROUP BY 1) s ORDER BY phase;")"
order_buy_try_open_24h="$(sql_scalar "SELECT count(1) FROM runtime_events WHERE created_at >= now() - interval '24 hours' AND step='auto.order_buy_try' AND details ~ 'phase=open';")"
order_buy_try_mid_24h="$(sql_scalar "SELECT count(1) FROM runtime_events WHERE created_at >= now() - interval '24 hours' AND step='auto.order_buy_try' AND details ~ 'phase=mid';")"

# Утренний чек-лист (последние 45 минут, особенно полезно в 10:00-10:45 MSK).
buy_try_45m="$(sql_scalar "SELECT count(1) FROM runtime_events WHERE created_at >= now() - interval '45 minutes' AND step='auto.order_buy_try';")"
buy_placed_45m="$(sql_scalar "SELECT count(1) FROM runtime_events WHERE created_at >= now() - interval '45 minutes' AND step='order.buy_placed';")"
no_order_45m="$(sql_scalar "SELECT count(1) FROM runtime_events WHERE created_at >= now() - interval '45 minutes' AND step='auto.no_order';")"
pending_bypass_45m="$(sql_scalar "SELECT count(1) FROM runtime_events WHERE created_at >= now() - interval '45 minutes' AND step='auto.pending_global_stale_bypass';")"
reject_156_45m="$(sql_scalar "SELECT count(1) FROM trades WHERE created_at >= now() - interval '45 minutes' AND status LIKE '%[156]%';")"
reject_257_45m="$(sql_scalar "SELECT count(1) FROM trades WHERE created_at >= now() - interval '45 minutes' AND status LIKE '%[257]%';")"
reject_157_45m="$(sql_scalar "SELECT count(1) FROM trades WHERE created_at >= now() - interval '45 minutes' AND status LIKE '%[157]%';")"

# Свечной/импульсный блок по market_data (последние 45 минут).
candle_momentum_summary_45m="$(sql_scalar "WITH recent AS (SELECT ticker, time, open, close, high, low, row_number() OVER (PARTITION BY ticker ORDER BY time ASC) AS rn_first, row_number() OVER (PARTITION BY ticker ORDER BY time DESC) AS rn_last FROM market_data WHERE time >= now() - interval '45 minutes' AND open IS NOT NULL AND close IS NOT NULL), agg AS (SELECT ticker, max(CASE WHEN rn_first=1 THEN open END) AS first_open, max(CASE WHEN rn_last=1 THEN close END) AS last_close, max(high) AS max_high, min(low) AS min_low, count(1) AS candles FROM recent GROUP BY ticker), scored AS (SELECT ticker, first_open, last_close, candles, CASE WHEN first_open > 0 THEN ((last_close - first_open) / first_open) * 100 ELSE NULL END AS ret_pct FROM agg WHERE candles >= 2 AND first_open > 0 AND last_close > 0) SELECT 'tickers=' || count(1)::text || ' up=' || count(1) FILTER (WHERE ret_pct > 0.20)::text || ' down=' || count(1) FILTER (WHERE ret_pct < -0.20)::text || ' flat=' || count(1) FILTER (WHERE ret_pct BETWEEN -0.20 AND 0.20)::text FROM scored;")"
candle_top_up_45m="$(sql_raw_lines "WITH recent AS (SELECT ticker, time, open, close, row_number() OVER (PARTITION BY ticker ORDER BY time ASC) AS rn_first, row_number() OVER (PARTITION BY ticker ORDER BY time DESC) AS rn_last FROM market_data WHERE time >= now() - interval '45 minutes' AND open IS NOT NULL AND close IS NOT NULL), agg AS (SELECT ticker, max(CASE WHEN rn_first=1 THEN open END) AS first_open, max(CASE WHEN rn_last=1 THEN close END) AS last_close, count(1) AS candles FROM recent GROUP BY ticker), scored AS (SELECT ticker, candles, CASE WHEN first_open > 0 THEN ((last_close - first_open) / first_open) * 100 ELSE NULL END AS ret_pct FROM agg WHERE candles >= 2 AND first_open > 0 AND last_close > 0) SELECT ticker || ' | ' || to_char(ret_pct, 'FM990D00') || '% | candles=' || candles::text FROM scored ORDER BY ret_pct DESC NULLS LAST LIMIT 5;")"
candle_top_down_45m="$(sql_raw_lines "WITH recent AS (SELECT ticker, time, open, close, row_number() OVER (PARTITION BY ticker ORDER BY time ASC) AS rn_first, row_number() OVER (PARTITION BY ticker ORDER BY time DESC) AS rn_last FROM market_data WHERE time >= now() - interval '45 minutes' AND open IS NOT NULL AND close IS NOT NULL), agg AS (SELECT ticker, max(CASE WHEN rn_first=1 THEN open END) AS first_open, max(CASE WHEN rn_last=1 THEN close END) AS last_close, count(1) AS candles FROM recent GROUP BY ticker), scored AS (SELECT ticker, candles, CASE WHEN first_open > 0 THEN ((last_close - first_open) / first_open) * 100 ELSE NULL END AS ret_pct FROM agg WHERE candles >= 2 AND first_open > 0 AND last_close > 0) SELECT ticker || ' | ' || to_char(ret_pct, 'FM990D00') || '% | candles=' || candles::text FROM scored ORDER BY ret_pct ASC NULLS LAST LIMIT 5;")"

stamp="$(TZ="$TZ_MSK" date '+%Y-%m-%d_%H%M')"
md_file="${REPORTS_DIR}/intraday_${stamp}.md"
json_file="${REPORTS_DIR}/intraday_${stamp}.json"
daily_file="${REPORTS_DIR}/daily_${msk_date}.md"
hints_file="${REPORTS_DIR}/agent_exit_hints.env"

recommendations=()
if [[ "${service_active}" != "active" ]]; then
  recommendations+=("Проверить сервис muhlya: не active.")
fi
if (( reject_24h > placed_24h + executed_24h )); then
  recommendations+=("Reject доминирует над placed/executed: проверить лимиты, ликвидность и фильтры.")
fi
if (( skip_low_balance_24h > 30 || low_balance_mode_24h > 30 )); then
  recommendations+=("Частые low-balance события: проверить AGENT_MIN_BALANCE_* и рабочий кэш.")
fi
if (( stale_timeout_24h > 20 )); then
  recommendations+=("Много stale-timeout: проверить max hold / stale band и качество точек выхода.")
fi
if (( session_closed_reject_24h > 10 )); then
  recommendations+=("Много reject вне сессии: усилить gate по расписанию перед отправкой заявок.")
fi
if (( order_buy_try_open_24h < 5 )); then
  recommendations+=("Слабая активность на открытии (phase=open): проверить open-bias Мухли и качество prefilter на 10:00–10:45.")
fi
if (( buy_try_45m > 0 && buy_placed_45m == 0 )); then
  recommendations+=("За последние 45 минут были попытки buy без placed: проверить cash-паузы, pending-гейт и shortlist тикеров.")
fi
if (( reject_156_45m >= 3 )); then
  recommendations+=("Частые [156] за 45 минут: снизить таргет дорогих тикеров/лотов, проверить budget и доступный кэш.")
fi
if [[ "${open_positions_db}" != "${broker_snapshot_open}" ]] && [[ "${broker_snapshot_age:-999999}" -lt "$BROKER_SNAPSHOT_STALE_SEC" ]]; then
  recommendations+=("Расхождение позиций: Postgres=${open_positions_db} vs Finam-снимок=${broker_snapshot_open} — локальный учёт не совпадает с брокером.")
fi

default_exit_hint="1.05"
if (( reject_24h > placed_24h + executed_24h )); then
  default_exit_hint="0.95"
fi
if (( stale_timeout_24h > 20 )); then
  default_exit_hint="0.92"
fi
if (( placed_24h + executed_24h > reject_24h )) && (( stale_timeout_24h < 10 )); then
  default_exit_hint="1.10"
fi

ticker_hints_raw="$(sql_raw_lines "WITH stat AS (SELECT upper(split_part(ticker, '@', 1)) AS t, count(1) FILTER (WHERE status='placed') AS placed_n, count(1) FILTER (WHERE status LIKE 'rejected:%') AS reject_n FROM trades WHERE created_at >= now() - interval '24 hours' GROUP BY 1) SELECT t || '|' || placed_n::text || '|' || reject_n::text FROM stat WHERE t <> '' ORDER BY t;")"

{
  echo "# Мия: внутридневной срез"
  echo
  echo "- Время (MSK): ${msk_now}"
  echo "- Окно биржи (эвристика): $([[ "$active_window" -eq 1 ]] && echo "активное" || echo "вне активного окна")"
  echo "- Фаза открытия (эвристика): $([[ "$open_window" -eq 1 ]] && echo "да" || echo "нет")"
  echo "- Cadence отчёта: ${every_min} мин"
  echo "- Сервис muhlya: ${service_active}"
  echo "- Статус: ${service_status_line}"
  echo
  echo "## Ключевые метрики (24ч)"
  echo
  echo "- trades total: ${new_trades_24h}"
  echo "- placed: ${placed_24h}"
  echo "- executed: ${executed_24h}"
  echo "- rejected: ${reject_24h}"
  echo "- open positions (отчётный счётчик, источник=${positions_source}): ${open_positions}"
  echo "- open positions (local DB / Postgres): ${open_positions_db}"
  if [[ -f "$BROKER_SNAPSHOT_PATH" ]]; then
    echo "- Finam snapshot: open=${broker_snapshot_open}, age_sec=${broker_snapshot_age}, file=${BROKER_SNAPSHOT_PATH}"
    if [[ -n "${broker_snapshot_tickers}" ]]; then
      echo "- Finam snapshot tickers: ${broker_snapshot_tickers}"
    fi
  else
    echo "- Finam snapshot: файл не найден (${BROKER_SNAPSHOT_PATH})"
  fi
  echo "- auto.no_order: ${no_order_24h}"
  echo "- auto.skip_low_balance: ${skip_low_balance_24h}"
  echo "- auto.low_balance_manage_only: ${low_balance_mode_24h}"
  echo "- auto.exit_stale_timeout: ${stale_timeout_24h}"
  echo "- rejected outside session: ${session_closed_reject_24h}"
  echo "- auto.order_buy_try (open): ${order_buy_try_open_24h}"
  echo "- auto.order_buy_try (mid): ${order_buy_try_mid_24h}"
  echo
  echo "## Фазы сессии (24ч, Europe/Moscow → auto.session_tick / A/B/C)"
  echo
  echo "Подсчёт тиков автоцикла по фазе (open/mid/close/offhours):"
  if [[ -n "$phase_ticks_24h" ]]; then
    while IFS= read -r line; do
      [[ -n "$line" ]] && echo "- ${line}"
    done <<< "$phase_ticks_24h"
  else
    echo "- нет данных (нужен обновлённый muhlya-headless с auto.session_tick)"
  fi
  echo
  echo "Суммы счётчиков A/B/C по фазам (auto.abtest_cycle, если включён AGENT_ABTEST_ENABLED):"
  if [[ -n "$abtest_phase_24h" ]]; then
    while IFS= read -r line; do
      [[ -n "$line" ]] && echo "- ${line}"
    done <<< "$abtest_phase_24h"
  else
    echo "- нет строк abtest_cycle за 24ч"
  fi
  echo
  echo "Попытки покупки по фазам (auto.order_buy_try):"
  if [[ -n "$order_buy_try_phase_24h" ]]; then
    while IFS= read -r line; do
      [[ -n "$line" ]] && echo "- ${line}"
    done <<< "$order_buy_try_phase_24h"
  else
    echo "- нет order_buy_try за 24ч"
  fi
  echo
  echo "## Топ причин reject (24ч)"
  echo
  if [[ -n "$top_rejects_raw" ]]; then
    while IFS= read -r line; do
      [[ -n "$line" ]] && echo "- ${line}"
    done <<< "$top_rejects_raw"
  else
    echo "- нет reject за 24ч"
  fi
  echo
  echo "## Топ runtime step (24ч)"
  echo
  if [[ -n "$top_steps_raw" ]]; then
    while IFS= read -r line; do
      [[ -n "$line" ]] && echo "- ${line}"
    done <<< "$top_steps_raw"
  else
    echo "- нет runtime событий за 24ч"
  fi
  echo
  echo "## Утренний чек-лист (последние 45 минут)"
  echo
  echo "- auto.order_buy_try (45m): ${buy_try_45m}"
  echo "- order.buy_placed (45m): ${buy_placed_45m}"
  echo "- auto.no_order (45m): ${no_order_45m}"
  echo "- auto.pending_global_stale_bypass (45m): ${pending_bypass_45m}"
  echo "- reject [156] / [157] / [257] (45m): ${reject_156_45m} / ${reject_157_45m} / ${reject_257_45m}"
  echo
  echo "## Свечной импульс (market_data, 45m)"
  echo
  if [[ -n "$candle_momentum_summary_45m" ]]; then
    echo "- summary: ${candle_momentum_summary_45m}"
  else
    echo "- summary: нет данных (проверить поток market_data)"
  fi
  echo
  echo "Лидеры роста (45m):"
  if [[ -n "$candle_top_up_45m" ]]; then
    while IFS= read -r line; do
      [[ -n "$line" ]] && echo "- ${line}"
    done <<< "$candle_top_up_45m"
  else
    echo "- нет данных"
  fi
  echo
  echo "Лидеры снижения (45m):"
  if [[ -n "$candle_top_down_45m" ]]; then
    while IFS= read -r line; do
      [[ -n "$line" ]] && echo "- ${line}"
    done <<< "$candle_top_down_45m"
  else
    echo "- нет данных"
  fi
  echo
  echo "## Рекомендации на завтра"
  echo
  if ((${#recommendations[@]} == 0)); then
    echo "- Критичных флагов не обнаружено, продолжать мониторинг в штатном режиме."
  else
    for rec in "${recommendations[@]}"; do
      echo "- ${rec}"
    done
  fi
} > "$md_file"

{
  echo "# Мия: дневная сводка (${msk_date})"
  echo
  echo "- Последнее обновление (MSK): ${msk_now}"
  echo "- Последний внутридневной отчёт: ${md_file}"
  echo
  echo "## Итоги 24ч"
  echo
  echo "- trades total: ${new_trades_24h}"
  echo "- placed: ${placed_24h}"
  echo "- executed: ${executed_24h}"
  echo "- rejected: ${reject_24h}"
  echo "- open positions (источник=${positions_source}): ${open_positions} (DB=${open_positions_db})"
  echo "- auto.no_order: ${no_order_24h}"
  echo "- auto.skip_low_balance: ${skip_low_balance_24h}"
  echo "- auto.low_balance_manage_only: ${low_balance_mode_24h}"
  echo "- auto.exit_stale_timeout: ${stale_timeout_24h}"
  echo "- rejected outside session: ${session_closed_reject_24h}"
  echo "- auto.order_buy_try (open): ${order_buy_try_open_24h}"
  echo "- auto.order_buy_try (mid): ${order_buy_try_mid_24h}"
  echo "- buy_try_45m / buy_placed_45m: ${buy_try_45m} / ${buy_placed_45m}"
  echo "- candle momentum 45m: ${candle_momentum_summary_45m:-нет данных}"
  echo
  echo "## Фазы сессии (24ч)"
  echo
  if [[ -n "$phase_ticks_24h" ]]; then
    while IFS= read -r line; do
      [[ -n "$line" ]] && echo "- ${line}"
    done <<< "$phase_ticks_24h"
  else
    echo "- нет данных по auto.session_tick"
  fi
  echo
  echo "## Рекомендации на завтра"
  echo
  if ((${#recommendations[@]} == 0)); then
    echo "- Критичных флагов не обнаружено, базовый профиль можно не менять."
  else
    for rec in "${recommendations[@]}"; do
      echo "- ${rec}"
    done
  fi
} > "$daily_file"

{
  echo "# agent exit hints generated by miya"
  echo "# format: TICKER=multiplier, DEFAULT=multiplier"
  echo "DEFAULT=${default_exit_hint}"
  if [[ -n "$ticker_hints_raw" ]]; then
    while IFS='|' read -r t p r; do
      [[ -z "$t" ]] && continue
      mul="1.00"
      if [[ "${r:-0}" -ge 5 && "${p:-0}" -eq 0 ]]; then
        mul="0.90"
      elif [[ "${p:-0}" -ge 3 && "${r:-0}" -le 1 ]]; then
        mul="1.10"
      elif [[ "${r:-0}" -ge 3 && "${p:-0}" -ge 1 ]]; then
        mul="0.96"
      fi
      echo "${t}=${mul}"
    done <<< "$ticker_hints_raw"
  fi
} > "$hints_file"

# Подробности по фазам — в отдельные файлы (валидный JSON без экранирования многострочного текста).
phase_ticks_file="${REPORTS_DIR}/intraday_${stamp}_phase_ticks.txt"
phase_abtest_file="${REPORTS_DIR}/intraday_${stamp}_phase_abtest.txt"
phase_order_buy_file="${REPORTS_DIR}/intraday_${stamp}_phase_order_buy_try.txt"
echo "$phase_ticks_24h" > "$phase_ticks_file"
echo "$abtest_phase_24h" > "$phase_abtest_file"
echo "$order_buy_try_phase_24h" > "$phase_order_buy_file"

json_escape() {
  printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

cat > "$json_file" <<EOF
{
  "generated_msk": "${msk_now}",
  "active_window": ${active_window},
  "open_window": ${open_window},
  "report_cadence_min": ${every_min},
  "service_active": "$(json_escape "$service_active")",
  "metrics_24h": {
    "trades_total": ${new_trades_24h:-0},
    "placed": ${placed_24h:-0},
    "executed": ${executed_24h:-0},
    "rejected": ${reject_24h:-0},
    "open_positions": ${open_positions:-0},
    "open_positions_db": ${open_positions_db:-0},
    "open_positions_broker_snapshot": ${broker_snapshot_open:-0},
    "positions_report_source": "$(json_escape "$positions_source")",
    "broker_snapshot_age_sec": ${broker_snapshot_age:-999999},
    "auto_no_order": ${no_order_24h:-0},
    "auto_skip_low_balance": ${skip_low_balance_24h:-0},
    "auto_low_balance_manage_only": ${low_balance_mode_24h:-0},
    "auto_exit_stale_timeout": ${stale_timeout_24h:-0},
    "rejected_outside_session": ${session_closed_reject_24h:-0},
    "auto_order_buy_try_open": ${order_buy_try_open_24h:-0},
    "auto_order_buy_try_mid": ${order_buy_try_mid_24h:-0}
  },
  "opening_checklist_45m": {
    "auto_order_buy_try": ${buy_try_45m:-0},
    "order_buy_placed": ${buy_placed_45m:-0},
    "auto_no_order": ${no_order_45m:-0},
    "auto_pending_global_stale_bypass": ${pending_bypass_45m:-0},
    "reject_156": ${reject_156_45m:-0},
    "reject_157": ${reject_157_45m:-0},
    "reject_257": ${reject_257_45m:-0}
  },
  "candles_45m": {
    "summary": "$(json_escape "${candle_momentum_summary_45m:-}")"
  },
  "session_phases_24h": {
    "phase_ticks_file": "$(json_escape "$phase_ticks_file")",
    "abtest_ABC_sums_by_phase_file": "$(json_escape "$phase_abtest_file")",
    "order_buy_try_by_phase_file": "$(json_escape "$phase_order_buy_file")"
  },
  "agent_exit_hints_file": "$(json_escape "$hints_file")",
  "report_md": "$(json_escape "$md_file")",
  "report_json": "$(json_escape "$json_file")"
}
EOF

cat > "$STATE_FILE" <<EOF
last_written_epoch=${epoch_now}
last_written_msk=${msk_now}
last_report_md=${md_file}
last_report_json=${json_file}
EOF

echo "miya: wrote ${md_file}"
echo "miya: wrote ${json_file}"
echo "miya: wrote ${daily_file}"

# Ротация файлов отчётов Мии, чтобы каталог не рос бесконечно.
if [[ "${REPORT_RETENTION_DAYS}" =~ ^[0-9]+$ ]] && (( REPORT_RETENTION_DAYS >= 1 )); then
  removed_count="$(
    find "$REPORTS_DIR" -maxdepth 1 -type f \
      \( -name 'intraday_*.md' -o -name 'intraday_*.json' -o -name 'intraday_*_phase_*.txt' -o -name 'daily_*.md' \) \
      -mtime +"${REPORT_RETENTION_DAYS}" -print -delete 2>/dev/null | wc -l | tr -d ' '
  )"
  if [[ "${removed_count}" != "0" ]]; then
    echo "miya: cleanup removed ${removed_count} file(s) older than ${REPORT_RETENTION_DAYS}d"
  fi
fi
exit 0
