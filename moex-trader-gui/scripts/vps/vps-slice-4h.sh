#!/usr/bin/env bash
# Срез Postgres по Мухле за последние N часов (по умолчанию 4).
# Запуск на VDS из каталога с docker-compose или из любого места с заданными env:
#   PG_CONTAINER_NAME=moex_trading_db PGUSER=trader PGDATABASE=moex_trading \
#     bash /opt/trading/app/moex-trader-gui/scripts/vps/vps-slice-4h.sh
# Часы: второй аргумент, например: bash vps-slice-4h.sh 6

set -euo pipefail

HOURS="${1:-4}"
CONTAINER="${PG_CONTAINER_NAME:-moex_trading_db}"
PGUSER="${PGUSER:-trader}"
PGDATABASE="${PGDATABASE:-moex_trading}"

echo "=== vps-slice: last ${HOURS}h (container=${CONTAINER}, db=${PGDATABASE}) ==="
echo

echo "=== systemd (muhlya / miya timers) ==="
systemctl is-active muhlya.service 2>/dev/null || true
systemctl is-active miya-report.timer 2>/dev/null || true
systemctl is-active miya-online-watch.timer 2>/dev/null || true
echo

echo "=== runtime_events: top steps ==="
docker exec "$CONTAINER" psql -U "$PGUSER" -d "$PGDATABASE" -c "
SELECT step, count(1) AS n
FROM runtime_events
WHERE created_at >= now() - make_interval(hours => ${HOURS})
GROUP BY step
ORDER BY n DESC
LIMIT 18;
"

echo "=== trades: by status (top 12) ==="
docker exec "$CONTAINER" psql -U "$PGUSER" -d "$PGDATABASE" -c "
SELECT status, count(1) AS n
FROM trades
WHERE created_at >= now() - make_interval(hours => ${HOURS})
GROUP BY status
ORDER BY n DESC
LIMIT 12;
"

echo "=== trades: grouped reject buckets ==="
docker exec "$CONTAINER" psql -U "$PGUSER" -d "$PGDATABASE" -c "
SELECT CASE
  WHEN status LIKE 'rejected: buy blocked by lot affordability%' THEN 'rejected: lot_affordability'
  WHEN status LIKE 'rejected: throttle%' THEN 'rejected: throttle'
  WHEN status LIKE 'rejected: place order failed%' THEN 'rejected: place_order_failed'
  ELSE left(status, 100)
END AS bucket,
count(1) AS n
FROM trades
WHERE created_at >= now() - make_interval(hours => ${HOURS})
GROUP BY 1
ORDER BY n DESC
LIMIT 15;
"

echo "=== trades: placed / executed counts ==="
docker exec "$CONTAINER" psql -U "$PGUSER" -d "$PGDATABASE" -t -A -c "
SELECT 'placed', count(1) FROM trades WHERE created_at >= now() - make_interval(hours => ${HOURS}) AND status = 'placed'
UNION ALL
SELECT 'executed', count(1) FROM trades WHERE created_at >= now() - make_interval(hours => ${HOURS}) AND status = 'executed';
"

echo "=== market_data: rows in window ==="
docker exec "$CONTAINER" psql -U "$PGUSER" -d "$PGDATABASE" -t -A -c "
SELECT count(1) FROM market_data WHERE time >= now() - make_interval(hours => ${HOURS});
"

echo "=== miya_online_signal.env (head) ==="
if [[ -r /var/log/muhlya-reports/miya_online_signal.env ]]; then
  head -25 /var/log/muhlya-reports/miya_online_signal.env
else
  echo "(file missing or unreadable)"
fi

echo
echo "=== done ==="
