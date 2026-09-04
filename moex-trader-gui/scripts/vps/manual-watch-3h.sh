#!/usr/bin/env bash
set -euo pipefail

INTERVAL_MIN="${1:-10}"
DURATION_HOURS="${2:-3}"

if ! [[ "$INTERVAL_MIN" =~ ^[0-9]+$ ]] || [ "$INTERVAL_MIN" -lt 1 ]; then
  echo "INTERVAL_MIN must be positive integer"
  exit 1
fi
if ! [[ "$DURATION_HOURS" =~ ^[0-9]+$ ]] || [ "$DURATION_HOURS" -lt 1 ]; then
  echo "DURATION_HOURS must be positive integer"
  exit 1
fi

REPORT_DIR="$HOME/muhlya-watch-reports"
mkdir -p "$REPORT_DIR"
STAMP="$(date +%Y%m%d_%H%M%S)"
OUT_FILE="$REPORT_DIR/watch_${STAMP}_${DURATION_HOURS}h_${INTERVAL_MIN}m.log"

steps_total=$(( (DURATION_HOURS * 60 + INTERVAL_MIN - 1) / INTERVAL_MIN ))
sleep_sec=$(( INTERVAL_MIN * 60 ))

echo "manual-watch start: $(date -Is)" | tee -a "$OUT_FILE"
echo "interval_min=$INTERVAL_MIN duration_hours=$DURATION_HOURS samples=$steps_total" | tee -a "$OUT_FILE"
echo "output=$OUT_FILE" | tee -a "$OUT_FILE"

for ((i=1; i<=steps_total; i++)); do
  now_iso="$(date -Is)"
  echo "" | tee -a "$OUT_FILE"
  echo "===== sample $i/$steps_total at $now_iso =====" | tee -a "$OUT_FILE"

  docker exec moex_trading_db psql -U trader -d moex_trading -P pager=off -c \
    "SELECT status, count(*) AS n
     FROM trades
     WHERE created_at >= now() - interval '${INTERVAL_MIN} minutes'
     GROUP BY status
     ORDER BY n DESC;" | tee -a "$OUT_FILE"

  docker exec moex_trading_db psql -U trader -d moex_trading -P pager=off -c \
    "SELECT
       count(*) FILTER (WHERE status ILIKE '%Invalid quantity value%') AS invalid_qty,
       count(*) FILTER (WHERE status ILIKE '%[156]%') AS money_shortage_156,
       count(*) FILTER (WHERE status ILIKE '%[157]%') AS equities_shortage_157
     FROM trades
     WHERE created_at >= now() - interval '${INTERVAL_MIN} minutes';" | tee -a "$OUT_FILE"

  docker exec moex_trading_db psql -U trader -d moex_trading -P pager=off -c \
    "SELECT step, count(*) AS n
     FROM runtime_events
     WHERE created_at >= now() - interval '${INTERVAL_MIN} minutes'
       AND step IN (
         'auto.order_buy_try',
         'auto.order_sell_try',
         'auto.order_short_open_try',
         'order.buy_placed',
         'order.sell_placed',
         'order.short_open_placed',
         'order.buy_failed',
         'order.sell_failed',
         'order.short_open_failed',
         'auto.exit_stop_loss',
         'auto.exit_take_profit',
         'auto.exit_stale_timeout'
       )
     GROUP BY step
     ORDER BY n DESC;" | tee -a "$OUT_FILE"

  docker exec moex_trading_db psql -U trader -d moex_trading -P pager=off -c \
    "SELECT to_char(created_at, 'HH24:MI:SS') AS ts, ticker, side, quantity, price, status
     FROM trades
     WHERE created_at >= now() - interval '${INTERVAL_MIN} minutes'
     ORDER BY created_at DESC
     LIMIT 12;" | tee -a "$OUT_FILE"

  if [ "$i" -lt "$steps_total" ]; then
    sleep "$sleep_sec"
  fi
done

echo "" | tee -a "$OUT_FILE"
echo "manual-watch done: $(date -Is)" | tee -a "$OUT_FILE"
echo "$OUT_FILE"
