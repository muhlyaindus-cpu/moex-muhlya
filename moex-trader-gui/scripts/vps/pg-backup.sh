#!/usr/bin/env bash
# Ежедневный бэкап PostgreSQL из контейнера Docker (VPS / Linux).
# Формат: custom (-Fc), удобен для pg_restore.
#
# Переменные окружения (опционально):
#   PG_CONTAINER_NAME  — имя контейнера (по умолчанию moex_trading_db)
#   PGUSER             — пользователь БД (по умолчанию trader)
#   PGDATABASE         — имя БД (по умолчанию moex_trading)
#   PG_BACKUP_DIR      — каталог для .dump (по умолчанию /var/backups/moex-trading-pg)
#   PG_BACKUP_RETENTION_DAYS — удалять файлы старше N дней (по умолчанию 14)
#
# Требования: docker, контейнер запущен; пользователь в группе docker (или root).
#
# Ручной запуск:
#   chmod +x pg-backup.sh
#   ./pg-backup.sh

set -euo pipefail

CONTAINER="${PG_CONTAINER_NAME:-moex_trading_db}"
PGUSER="${PGUSER:-trader}"
PGDATABASE="${PGDATABASE:-moex_trading}"
BACKUP_DIR="${PG_BACKUP_DIR:-/var/backups/moex-trading-pg}"
RETENTION_DAYS="${PG_BACKUP_RETENTION_DAYS:-14}"

if ! docker inspect "$CONTAINER" >/dev/null 2>&1; then
  echo "pg-backup: контейнер не найден или Docker недоступен: $CONTAINER" >&2
  exit 1
fi

if ! docker inspect -f '{{.State.Running}}' "$CONTAINER" | grep -q true; then
  echo "pg-backup: контейнер не запущен: $CONTAINER" >&2
  exit 1
fi

mkdir -p "$BACKUP_DIR"
TS="$(date +%Y%m%d_%H%M%S)"
TMP="${BACKUP_DIR}/moex_trading_${TS}.dump.tmp"
OUT="${BACKUP_DIR}/moex_trading_${TS}.dump"

echo "pg-backup: pg_dump -> ${OUT}"
docker exec "$CONTAINER" pg_dump -U "$PGUSER" -Fc "$PGDATABASE" >"$TMP"
mv -f "$TMP" "$OUT"
echo "pg-backup: готово, размер $(du -h "$OUT" | cut -f1)"

if [[ "$RETENTION_DAYS" =~ ^[0-9]+$ ]] && [[ "$RETENTION_DAYS" -gt 0 ]]; then
  echo "pg-backup: ротация: удаляю dump-файлы старше ${RETENTION_DAYS} дн. в ${BACKUP_DIR}"
  find "$BACKUP_DIR" -maxdepth 1 -type f -name 'moex_trading_*.dump' -mtime "+${RETENTION_DAYS}" -print -delete || true
fi

exit 0
