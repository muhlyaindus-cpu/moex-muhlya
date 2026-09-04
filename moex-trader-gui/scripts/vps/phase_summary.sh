#!/usr/bin/env bash
# Сводка: фазы сессии (open/mid/close/offhours) + A/B/C + попытки buy за окно времени.
# Источник: runtime_events (auto.session_tick, auto.abtest_cycle, auto.order_buy_try).
#
# Использование:
#   ./phase_summary.sh           # последние 24 часа
#   ./phase_summary.sh 6         # последние 6 часов
#
# Переменные окружения (как у miya-report.sh):
#   PG_CONTAINER_NAME (default: moex_trading_db), PGUSER, PGDATABASE

set -euo pipefail

HOURS="${1:-24}"
PG_CONTAINER_NAME="${PG_CONTAINER_NAME:-moex_trading_db}"
PGUSER="${PGUSER:-trader}"
PGDATABASE="${PGDATABASE:-moex_trading}"

if ! docker inspect "$PG_CONTAINER_NAME" >/dev/null 2>&1; then
  echo "phase_summary: docker/container unavailable: $PG_CONTAINER_NAME" >&2
  exit 1
fi

run_sql() {
  docker exec "$PG_CONTAINER_NAME" psql -U "$PGUSER" -d "$PGDATABASE" -v ON_ERROR_STOP=1 -c "$1"
}

echo "=== phase_summary: последние ${HOURS} ч ==="
echo
echo "--- auto.session_tick (сколько тиков автоцикла в каждой фазе) ---"
run_sql "SELECT coalesce(substring(details FROM '^phase=([^ ]+)'), '?') AS phase, count(*) AS n
FROM runtime_events
WHERE created_at >= now() - interval '${HOURS} hours' AND step = 'auto.session_tick'
GROUP BY 1 ORDER BY 1;"

echo
echo "--- auto.abtest_cycle: суммы A/B/C по фазам (если AGENT_ABTEST_ENABLED=1) ---"
run_sql "SELECT coalesce(substring(details FROM '^phase=([^ ]+)'), '?') AS phase,
  sum(coalesce(substring(details FROM 'A=([0-9]+)')::int, 0)) AS sum_a,
  sum(coalesce(substring(details FROM 'B=([0-9]+)')::int, 0)) AS sum_b,
  sum(coalesce(substring(details FROM 'C=([0-9]+)')::int, 0)) AS sum_c
FROM runtime_events
WHERE created_at >= now() - interval '${HOURS} hours' AND step = 'auto.abtest_cycle'
GROUP BY 1 ORDER BY 1;"

echo
echo "--- auto.order_buy_try по фазам ---"
run_sql "SELECT coalesce(substring(details FROM 'phase=(open|mid|close|offhours)'), '?') AS phase, count(*) AS n
FROM runtime_events
WHERE created_at >= now() - interval '${HOURS} hours' AND step = 'auto.order_buy_try'
GROUP BY 1 ORDER BY 1;"

echo
echo "--- trades: executed / rejected / placed (без разреза по фазе; для сопоставления по времени см. created_at) ---"
run_sql "SELECT status, count(*) FROM trades WHERE created_at >= now() - interval '${HOURS} hours' GROUP BY 1 ORDER BY 2 DESC;"

exit 0
