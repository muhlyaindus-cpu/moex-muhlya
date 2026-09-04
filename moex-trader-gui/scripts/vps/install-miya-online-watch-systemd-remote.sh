#!/usr/bin/env bash
# Запускать на VPS от root: sudo bash /tmp/install-miya-online-watch-systemd-remote.sh
# Файлы unit должны быть заранее в /tmp: miya-online-watch.service, miya-online-watch.timer
set -euo pipefail

SVC_SRC=/tmp/miya-online-watch.service
TMR_SRC=/tmp/miya-online-watch.timer
if [[ ! -f "$SVC_SRC" ]] || [[ ! -f "$TMR_SRC" ]]; then
  echo "missing $SVC_SRC or $TMR_SRC" >&2
  exit 1
fi

install -m 0644 "$SVC_SRC" /etc/systemd/system/miya-online-watch.service
install -m 0644 "$TMR_SRC" /etc/systemd/system/miya-online-watch.timer
systemctl daemon-reload
systemctl enable miya-online-watch.timer
systemctl restart miya-online-watch.timer
# Разовый прогон сразу после обновления unit
systemctl start miya-online-watch.service || true

echo "--- miya-online-watch.timer ---"
systemctl status miya-online-watch.timer --no-pager
echo "--- last miya-online-watch.service ---"
journalctl -u miya-online-watch.service -n 25 --no-pager
