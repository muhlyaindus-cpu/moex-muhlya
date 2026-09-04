#!/usr/bin/env bash
# Smoke-проверка: восстановление последнего (или указанного) дампа в отдельную БД и удаление её.
# Не трогает боевую moex_trading. Запускать вручную после деплоя бэкапов или по расписанию реже основного бэкапа.
#
# Использование:
#   ./pg-smoke-restore.sh /var/backups/moex-trading-pg/moex_trading_20260403_033000.dump
#   PG_RESTORE_TEST_DB=my_test_db ./pg-smoke-restore.sh   # взять последний .dump из PG_BACKUP_DIR
#
# Переменные:
#   PG_CONTAINER_NAME, PGUSER — как в pg-backup.sh
#   PG_BACKUP_DIR — каталог поиска последнего дампа (если аргумент не передан)
#   PG_RESTORE_TEST_DB — имя временной БД (по умолчанию moex_trading_restore_smoke)

set -euo pipefail

CONTAINER="${PG_CONTAINER_NAME:-moex_trading_db}"
PGUSER="${PGUSER:-trader}"
BACKUP_DIR="${PG_BACKUP_DIR:-/var/backups/moex-trading-pg}"
RESTORE_DB="${PG_RESTORE_TEST_DB:-moex_trading_restore_smoke}"

if ! docker inspect "$CONTAINER" >/dev/null 2>&1; then
  echo "pg-smoke-restore: контейнер не найден: $CONTAINER" >&2
  exit 1
fi

DUMP=""
if [[ $# -ge 1 && -n "${1:-}" ]]; then
  DUMP="$1"
else
  # shellcheck disable=SC2012
  DUMP="$(ls -t "$BACKUP_DIR"/moex_trading_*.dump 2>/dev/null | head -1)"
  if [[ -z "$DUMP" || ! -f "$DUMP" ]]; then
    echo "pg-smoke-restore: не найден дамп в $BACKUP_DIR (передайте путь к .dump первым аргументом)" >&2
    exit 1
  fi
  echo "pg-smoke-restore: использую последний дамп: $DUMP"
fi

if [[ ! -f "$DUMP" ]]; then
  echo "pg-smoke-restore: файл не найден: $DUMP" >&2
  exit 1
fi

cleanup() {
  docker exec "$CONTAINER" dropdb -U "$PGUSER" --if-exists "$RESTORE_DB" 2>/dev/null || true
}
trap cleanup EXIT

echo "pg-smoke-restore: удаляю старую тестовую БД (если есть): $RESTORE_DB"
docker exec "$CONTAINER" dropdb -U "$PGUSER" --if-exists "$RESTORE_DB" 2>/dev/null || true

echo "pg-smoke-restore: createdb $RESTORE_DB"
docker exec "$CONTAINER" createdb -U "$PGUSER" "$RESTORE_DB"

echo "pg-smoke-restore: pg_restore (может быть много NOTICE — это нормально)"
set +e
docker exec -i "$CONTAINER" pg_restore -U "$PGUSER" -d "$RESTORE_DB" --no-owner --no-acl <"$DUMP"
RC=$?
set -e
# 0 — ок; 1 — предупреждения; 2+ — фатально
if [[ "$RC" -ge 2 ]]; then
  echo "pg-smoke-restore: pg_restore завершился с кодом $RC" >&2
  exit "$RC"
fi

echo "pg-smoke-restore: проверка таблиц"
docker exec "$CONTAINER" psql -U "$PGUSER" -d "$RESTORE_DB" -c "\dt" | head -40

echo "pg-smoke-restore: OK, удаляю тестовую БД $RESTORE_DB"
cleanup
trap - EXIT

echo "pg-smoke-restore: готово"
