#!/usr/bin/env bash
# Проверка FORTS-тикеров через Finam GetAsset (тот же путь, что Muhlya / BuyFutures preflight).
# Нужен .env с FINAM_SECRET_REAL в каталоге приложения.
#
#   chmod +x scripts/vps/validate_futures.sh
#   ./scripts/vps/validate_futures.sh SiM6 BRM6 GAZR-6.26
#   ./scripts/vps/validate_futures.sh -f /tmp/forts_candidates.txt
#   ./scripts/vps/validate_futures.sh -allowlist-fragment -f ...  # только прошедшие, для вставки в lchi_futures_allowlist
#
# Сборка бинаря (Linux): GOOS=linux GOARCH=amd64 go build -o validate-futures ./cmd/validate-futures

set -euo pipefail
APP_DIR="${VALIDATE_FUTURES_APP_DIR:-/opt/trading/app/moex-trader-gui}"
cd "$APP_DIR"

if [[ -x ./validate-futures ]]; then
  exec ./validate-futures "$@"
fi
if command -v go >/dev/null 2>&1; then
  exec go run ./cmd/validate-futures "$@"
fi
echo "install Go or build: go build -o validate-futures ./cmd/validate-futures" >&2
exit 1
