#!/usr/bin/env bash
# Мия: онлайн-монитор свечного импульса (без архивирования).
# Пишет только "последний снимок" и сигнал:
#   - /var/log/muhlya-reports/online_candles_latest.md
#   - /var/log/muhlya-reports/online_candles_latest.json
#   - /var/log/muhlya-reports/miya_online_signal.env (в т.ч. MIYA_LIQUIDITY_ORDER по VALTODAY MOEX ISS)
#   - /var/log/muhlya-reports/online_orderbook_latest.md   (лучшие цены / глубина MOEX ISS, см. репозиторий docs/MIYA_ORDERBOOK.md)
#   - /var/log/muhlya-reports/online_orderbook_latest.json

set -euo pipefail

PG_CONTAINER_NAME="${PG_CONTAINER_NAME:-moex_trading_db}"
PGUSER="${PGUSER:-trader}"
PGDATABASE="${PGDATABASE:-moex_trading}"

REPORTS_DIR="${ONLINE_MIA_REPORTS_DIR:-/var/log/muhlya-reports}"
TZ_MSK="${ONLINE_MIA_TZ:-Europe/Moscow}"
ONLINE_STATE_FILE="${ONLINE_MIA_ONLINE_STATE_FILE:-${REPORTS_DIR}/.miya-online-state}"
NO_DATA_ALERT_EVERY_MIN="${ONLINE_MIA_ALERT_NO_DATA_EVERY_MIN:-10}"
DATA_SOURCE_MODE_RAW="$(printf '%s' "${ONLINE_MIA_DATA_SOURCE:-api_first}" | tr '[:upper:]' '[:lower:]')"
API_TICKERS_RAW="${ONLINE_MIA_API_TICKERS:-SBER,GAZP,LKOH,ROSN,NVTK,TATN,VTBR,MGNT,NLMK,SNGSP,TATNP,SGZH}"
FUTURES_WATCH_TICKERS_RAW="${ONLINE_MIA_FUTURES_WATCH_TICKERS:-}"
API_STALE_MIN="${ONLINE_MIA_API_STALE_MIN:-20}"
API_TIMEOUT_SEC="${ONLINE_MIA_API_TIMEOUT_SEC:-6}"

ORDERBOOK_ENABLE_RAW="$(printf '%s' "${ONLINE_MIA_ORDERBOOK_ENABLE:-1}" | tr '[:upper:]' '[:lower:]')"
ORDERBOOK_TICKERS_RAW="${ONLINE_MIA_ORDERBOOK_TICKERS:-}"
ORDERBOOK_TIMEOUT_SEC="${ONLINE_MIA_ORDERBOOK_TIMEOUT_SEC:-4}"

WINDOW_SHORT_MIN="${ONLINE_MIA_CANDLE_SHORT_MIN:-5}"
WINDOW_LONG_MIN="${ONLINE_MIA_CANDLE_LONG_MIN:-15}"
FLAT_BAND_PCT="${ONLINE_MIA_CANDLE_FLAT_BAND_PCT:-0.20}"
MIN_CANDLES="${ONLINE_MIA_CANDLE_MIN_POINTS:-1}"
WINDOW_FALLBACK_MIN="${ONLINE_MIA_CANDLE_FALLBACK_MIN:-45}"
USE_FALLBACK_ACTIVE_RAW="$(printf '%s' "${ONLINE_MIA_CANDLE_USE_FALLBACK_IN_ACTIVE:-1}" | tr '[:upper:]' '[:lower:]')"

BULLISH_THRESHOLD_PCT="${ONLINE_MIA_BULLISH_THRESHOLD_PCT:-55}"
BEARISH_THRESHOLD_PCT="${ONLINE_MIA_BEARISH_THRESHOLD_PCT:-35}"

# Дневные свечи (MOEX ISS candles, interval=24) — тот же список тикеров, что и для минуток/API.
DAILY_ENABLE_RAW="$(printf '%s' "${ONLINE_MIA_DAILY_ENABLE:-1}" | tr '[:upper:]' '[:lower:]')"
DAILY_LOOKBACK_DAYS="${ONLINE_MIA_DAILY_LOOKBACK_DAYS:-45}"
DAILY_BULLISH_PCT="${ONLINE_MIA_DAILY_BULLISH_THRESHOLD_PCT:-55}"
DAILY_BEARISH_PCT="${ONLINE_MIA_DAILY_BEARISH_THRESHOLD_PCT:-35}"
DAILY_MIN_BARS="${ONLINE_MIA_DAILY_MIN_BARS:-2}"

# Ранг ликвидности по VALTODAY (MOEX ISS) для того же списка, что ONLINE_MIA_API_TICKERS — подсказка Мухле (MIYA_LIQUIDITY_*).
ONLINE_MIA_LIQUIDITY_ENABLE_RAW="$(printf '%s' "${ONLINE_MIA_LIQUIDITY_ENABLE:-1}" | tr '[:upper:]' '[:lower:]')"
ONLINE_MIA_LIQUIDITY_TIMEOUT_SEC="${ONLINE_MIA_LIQUIDITY_TIMEOUT_SEC:-6}"

mkdir -p "$REPORTS_DIR"

db_available=1
if ! docker inspect "$PG_CONTAINER_NAME" >/dev/null 2>&1; then
  db_available=0
  if [[ "$DATA_SOURCE_MODE_RAW" == "db_only" ]]; then
    echo "miya-online: docker/container unavailable: $PG_CONTAINER_NAME" >&2
    exit 1
  fi
  echo "miya-online: warn: docker/container unavailable, API-only mode" >&2
fi

sql_scalar() {
  local sql="$1"
  docker exec "$PG_CONTAINER_NAME" psql -U "$PGUSER" -d "$PGDATABASE" -t -A -c "$sql" | tr -d '\r'
}

sql_lines() {
  local sql="$1"
  docker exec "$PG_CONTAINER_NAME" psql -U "$PGUSER" -d "$PGDATABASE" -t -A -c "$sql" | tr -d '\r'
}

strip_spaces_csv() {
  printf '%s' "$1" | tr -d '[:space:]'
}

merge_csv_unique_upper() {
  local left="${1:-}"
  local right="${2:-}"
  printf '%s\n%s\n' "$left" "$right" \
    | tr ',;' '\n' \
    | sed 's/[[:space:]]//g' \
    | awk 'NF { t=toupper($0); if (!seen[t]++) out=(out ? out "," : "") t } END { print out }'
}

json_escape() {
  printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

load_long_snapshot() {
  local window_min="$1"
  summary_long="$(sql_scalar "WITH recent AS (SELECT ticker, time, open, close, row_number() OVER (PARTITION BY ticker ORDER BY time ASC) AS rn_first, row_number() OVER (PARTITION BY ticker ORDER BY time DESC) AS rn_last FROM market_data WHERE time >= now() - make_interval(mins => ${window_min}) AND open IS NOT NULL AND close IS NOT NULL), agg AS (SELECT ticker, max(CASE WHEN rn_first=1 THEN open END) AS first_open, max(CASE WHEN rn_last=1 THEN close END) AS last_close, count(1) AS candles FROM recent GROUP BY ticker), scored AS (SELECT ticker, candles, CASE WHEN first_open > 0 THEN ((last_close - first_open) / first_open) * 100 ELSE NULL END AS ret_pct FROM agg WHERE candles >= ${MIN_CANDLES} AND first_open > 0 AND last_close > 0) SELECT 'tickers=' || count(1)::text || ' up=' || count(1) FILTER (WHERE ret_pct > ${FLAT_BAND_PCT})::text || ' down=' || count(1) FILTER (WHERE ret_pct < -${FLAT_BAND_PCT})::text || ' flat=' || count(1) FILTER (WHERE ret_pct BETWEEN -${FLAT_BAND_PCT} AND ${FLAT_BAND_PCT})::text FROM scored;")"

  tickers_long_count="$(sql_scalar "WITH recent AS (SELECT ticker, time, open, close, row_number() OVER (PARTITION BY ticker ORDER BY time ASC) AS rn_first, row_number() OVER (PARTITION BY ticker ORDER BY time DESC) AS rn_last FROM market_data WHERE time >= now() - make_interval(mins => ${window_min}) AND open IS NOT NULL AND close IS NOT NULL), agg AS (SELECT ticker, max(CASE WHEN rn_first=1 THEN open END) AS first_open, max(CASE WHEN rn_last=1 THEN close END) AS last_close, count(1) AS candles FROM recent GROUP BY ticker), scored AS (SELECT ticker FROM agg WHERE candles >= ${MIN_CANDLES} AND first_open > 0 AND last_close > 0) SELECT count(1) FROM scored;")"

  up_ratio_long="$(sql_scalar "WITH recent AS (SELECT ticker, time, open, close, row_number() OVER (PARTITION BY ticker ORDER BY time ASC) AS rn_first, row_number() OVER (PARTITION BY ticker ORDER BY time DESC) AS rn_last FROM market_data WHERE time >= now() - make_interval(mins => ${window_min}) AND open IS NOT NULL AND close IS NOT NULL), agg AS (SELECT ticker, max(CASE WHEN rn_first=1 THEN open END) AS first_open, max(CASE WHEN rn_last=1 THEN close END) AS last_close, count(1) AS candles FROM recent GROUP BY ticker), scored AS (SELECT CASE WHEN first_open > 0 THEN ((last_close - first_open) / first_open) * 100 ELSE NULL END AS ret_pct FROM agg WHERE candles >= ${MIN_CANDLES} AND first_open > 0 AND last_close > 0) SELECT COALESCE(ROUND(100.0 * SUM(CASE WHEN ret_pct > ${FLAT_BAND_PCT} THEN 1 ELSE 0 END)::numeric / NULLIF(COUNT(*),0), 2), 0) FROM scored;")"

  top_up_long="$(sql_lines "WITH recent AS (SELECT ticker, time, open, close, row_number() OVER (PARTITION BY ticker ORDER BY time ASC) AS rn_first, row_number() OVER (PARTITION BY ticker ORDER BY time DESC) AS rn_last FROM market_data WHERE time >= now() - make_interval(mins => ${window_min}) AND open IS NOT NULL AND close IS NOT NULL), agg AS (SELECT ticker, max(CASE WHEN rn_first=1 THEN open END) AS first_open, max(CASE WHEN rn_last=1 THEN close END) AS last_close, count(1) AS candles FROM recent GROUP BY ticker), scored AS (SELECT ticker, candles, CASE WHEN first_open > 0 THEN ((last_close - first_open) / first_open) * 100 ELSE NULL END AS ret_pct FROM agg WHERE candles >= ${MIN_CANDLES} AND first_open > 0 AND last_close > 0) SELECT ticker || ' | ' || to_char(ret_pct, 'FM990D00') || '% | candles=' || candles::text FROM scored ORDER BY ret_pct DESC NULLS LAST LIMIT 7;")"

  top_down_long="$(sql_lines "WITH recent AS (SELECT ticker, time, open, close, row_number() OVER (PARTITION BY ticker ORDER BY time ASC) AS rn_first, row_number() OVER (PARTITION BY ticker ORDER BY time DESC) AS rn_last FROM market_data WHERE time >= now() - make_interval(mins => ${window_min}) AND open IS NOT NULL AND close IS NOT NULL), agg AS (SELECT ticker, max(CASE WHEN rn_first=1 THEN open END) AS first_open, max(CASE WHEN rn_last=1 THEN close END) AS last_close, count(1) AS candles FROM recent GROUP BY ticker), scored AS (SELECT ticker, candles, CASE WHEN first_open > 0 THEN ((last_close - first_open) / first_open) * 100 ELSE NULL END AS ret_pct FROM agg WHERE candles >= ${MIN_CANDLES} AND first_open > 0 AND last_close > 0) SELECT ticker || ' | ' || to_char(ret_pct, 'FM990D00') || '% | candles=' || candles::text FROM scored ORDER BY ret_pct ASC NULLS LAST LIMIT 7;")"
}

load_db_snapshot() {
  local window_min="$1"
  local prefix="$2"
  local summary_var="${prefix}_summary"
  local tickers_var="${prefix}_tickers_count"
  local up_ratio_var="${prefix}_up_ratio"
  local top_up_var="${prefix}_top_up"
  local top_down_var="${prefix}_top_down"

  local summary tickers up_ratio top_up top_down
  summary="$(sql_scalar "WITH recent AS (SELECT ticker, time, open, close, row_number() OVER (PARTITION BY ticker ORDER BY time ASC) AS rn_first, row_number() OVER (PARTITION BY ticker ORDER BY time DESC) AS rn_last FROM market_data WHERE time >= now() - make_interval(mins => ${window_min}) AND open IS NOT NULL AND close IS NOT NULL), agg AS (SELECT ticker, max(CASE WHEN rn_first=1 THEN open END) AS first_open, max(CASE WHEN rn_last=1 THEN close END) AS last_close, count(1) AS candles FROM recent GROUP BY ticker), scored AS (SELECT ticker, candles, CASE WHEN first_open > 0 THEN ((last_close - first_open) / first_open) * 100 ELSE NULL END AS ret_pct FROM agg WHERE candles >= ${MIN_CANDLES} AND first_open > 0 AND last_close > 0) SELECT 'tickers=' || count(1)::text || ' up=' || count(1) FILTER (WHERE ret_pct > ${FLAT_BAND_PCT})::text || ' down=' || count(1) FILTER (WHERE ret_pct < -${FLAT_BAND_PCT})::text || ' flat=' || count(1) FILTER (WHERE ret_pct BETWEEN -${FLAT_BAND_PCT} AND ${FLAT_BAND_PCT})::text FROM scored;")"
  tickers="$(sql_scalar "WITH recent AS (SELECT ticker, time, open, close, row_number() OVER (PARTITION BY ticker ORDER BY time ASC) AS rn_first, row_number() OVER (PARTITION BY ticker ORDER BY time DESC) AS rn_last FROM market_data WHERE time >= now() - make_interval(mins => ${window_min}) AND open IS NOT NULL AND close IS NOT NULL), agg AS (SELECT ticker, max(CASE WHEN rn_first=1 THEN open END) AS first_open, max(CASE WHEN rn_last=1 THEN close END) AS last_close, count(1) AS candles FROM recent GROUP BY ticker), scored AS (SELECT ticker FROM agg WHERE candles >= ${MIN_CANDLES} AND first_open > 0 AND last_close > 0) SELECT count(1) FROM scored;")"
  up_ratio="$(sql_scalar "WITH recent AS (SELECT ticker, time, open, close, row_number() OVER (PARTITION BY ticker ORDER BY time ASC) AS rn_first, row_number() OVER (PARTITION BY ticker ORDER BY time DESC) AS rn_last FROM market_data WHERE time >= now() - make_interval(mins => ${window_min}) AND open IS NOT NULL AND close IS NOT NULL), agg AS (SELECT ticker, max(CASE WHEN rn_first=1 THEN open END) AS first_open, max(CASE WHEN rn_last=1 THEN close END) AS last_close, count(1) AS candles FROM recent GROUP BY ticker), scored AS (SELECT CASE WHEN first_open > 0 THEN ((last_close - first_open) / first_open) * 100 ELSE NULL END AS ret_pct FROM agg WHERE candles >= ${MIN_CANDLES} AND first_open > 0 AND last_close > 0) SELECT COALESCE(ROUND(100.0 * SUM(CASE WHEN ret_pct > ${FLAT_BAND_PCT} THEN 1 ELSE 0 END)::numeric / NULLIF(COUNT(*),0), 2), 0) FROM scored;")"
  top_up="$(sql_lines "WITH recent AS (SELECT ticker, time, open, close, row_number() OVER (PARTITION BY ticker ORDER BY time ASC) AS rn_first, row_number() OVER (PARTITION BY ticker ORDER BY time DESC) AS rn_last FROM market_data WHERE time >= now() - make_interval(mins => ${window_min}) AND open IS NOT NULL AND close IS NOT NULL), agg AS (SELECT ticker, max(CASE WHEN rn_first=1 THEN open END) AS first_open, max(CASE WHEN rn_last=1 THEN close END) AS last_close, count(1) AS candles FROM recent GROUP BY ticker), scored AS (SELECT ticker, candles, CASE WHEN first_open > 0 THEN ((last_close - first_open) / first_open) * 100 ELSE NULL END AS ret_pct FROM agg WHERE candles >= ${MIN_CANDLES} AND first_open > 0 AND last_close > 0) SELECT ticker || ' | ' || to_char(ret_pct, 'FM990D00') || '% | candles=' || candles::text FROM scored ORDER BY ret_pct DESC NULLS LAST LIMIT 7;")"
  top_down="$(sql_lines "WITH recent AS (SELECT ticker, time, open, close, row_number() OVER (PARTITION BY ticker ORDER BY time ASC) AS rn_first, row_number() OVER (PARTITION BY ticker ORDER BY time DESC) AS rn_last FROM market_data WHERE time >= now() - make_interval(mins => ${window_min}) AND open IS NOT NULL AND close IS NOT NULL), agg AS (SELECT ticker, max(CASE WHEN rn_first=1 THEN open END) AS first_open, max(CASE WHEN rn_last=1 THEN close END) AS last_close, count(1) AS candles FROM recent GROUP BY ticker), scored AS (SELECT ticker, candles, CASE WHEN first_open > 0 THEN ((last_close - first_open) / first_open) * 100 ELSE NULL END AS ret_pct FROM agg WHERE candles >= ${MIN_CANDLES} AND first_open > 0 AND last_close > 0) SELECT ticker || ' | ' || to_char(ret_pct, 'FM990D00') || '% | candles=' || candles::text FROM scored ORDER BY ret_pct ASC NULLS LAST LIMIT 7;")"

  printf -v "$summary_var" '%s' "$summary"
  printf -v "$tickers_var" '%s' "$tickers"
  printf -v "$up_ratio_var" '%s' "$up_ratio"
  printf -v "$top_up_var" '%s' "$top_up"
  printf -v "$top_down_var" '%s' "$top_down"
}

load_api_snapshot() {
  local window_min="$1"
  local prefix="$2"
  local summary_var="${prefix}_summary"
  local tickers_var="${prefix}_tickers_count"
  local up_ratio_var="${prefix}_up_ratio"
  local top_up_var="${prefix}_top_up"
  local top_down_var="${prefix}_top_down"
  local stale_var="${prefix}_stale_count"
  local details_var="${prefix}_api_details"
  local tmp_file
  tmp_file="$(mktemp)"

  API_TICKERS_CSV="$(strip_spaces_csv "$API_TICKERS_RAW")" \
  API_WINDOW_MIN="$window_min" \
  API_MIN_CANDLES="$MIN_CANDLES" \
  API_FLAT_BAND_PCT="$FLAT_BAND_PCT" \
  API_STALE_MIN="$API_STALE_MIN" \
  API_TIMEOUT_SEC="$API_TIMEOUT_SEC" \
  python3 - <<'PY' >"$tmp_file"
import json
import os
import urllib.parse
import urllib.request
from datetime import datetime, timedelta, timezone

def parse_float(v):
    try:
        return float(v)
    except Exception:
        return 0.0

def parse_begin(ts):
    if not ts:
        return None
    ts = str(ts).strip()
    ts = ts.replace(" ", "T")
    if len(ts) == 16:
        ts += ":00"
    try:
        dt = datetime.fromisoformat(ts)
        if dt.tzinfo is None:
            dt = dt.replace(tzinfo=timezone.utc)
        return dt.astimezone(timezone.utc)
    except Exception:
        return None

tickers = [t.strip().upper() for t in os.getenv("API_TICKERS_CSV", "").split(",") if t.strip()]
window_min = max(1, int(float(os.getenv("API_WINDOW_MIN", "15"))))
min_candles = max(1, int(float(os.getenv("API_MIN_CANDLES", "1"))))
flat_band_pct = float(os.getenv("API_FLAT_BAND_PCT", "0.20"))
stale_min = max(1, int(float(os.getenv("API_STALE_MIN", "20"))))
timeout_sec = max(2, int(float(os.getenv("API_TIMEOUT_SEC", "6"))))

now_utc = datetime.now(timezone.utc)
from_dt = now_utc - timedelta(minutes=window_min + 90)
till_dt = now_utc + timedelta(minutes=5)

scored = []
stale_count = 0
errors = []

for ticker in tickers:
    params = {
        "iss.meta": "off",
        "interval": "1",
        "from": from_dt.strftime("%Y-%m-%d %H:%M:%S"),
        "till": till_dt.strftime("%Y-%m-%d %H:%M:%S"),
        "candles.columns": "begin,open,close"
    }
    url = (
        "https://iss.moex.com/iss/engines/stock/markets/shares/boards/TQBR/securities/"
        + urllib.parse.quote(ticker)
        + "/candles.json?"
        + urllib.parse.urlencode(params)
    )
    try:
        with urllib.request.urlopen(url, timeout=timeout_sec) as resp:
            payload = json.loads(resp.read().decode("utf-8", errors="ignore"))
    except Exception as exc:
        errors.append(f"{ticker}: {exc}")
        continue

    candles = payload.get("candles") or {}
    cols = candles.get("columns") or []
    data = candles.get("data") or []
    if not cols or not data:
        continue
    try:
        i_begin = cols.index("begin")
        i_open = cols.index("open")
        i_close = cols.index("close")
    except ValueError:
        continue

    rows = []
    for row in data:
        if not isinstance(row, list):
            continue
        if max(i_begin, i_open, i_close) >= len(row):
            continue
        begin_dt = parse_begin(row[i_begin])
        if begin_dt is None:
            continue
        open_p = parse_float(row[i_open])
        close_p = parse_float(row[i_close])
        if open_p <= 0 or close_p <= 0:
            continue
        rows.append((begin_dt, open_p, close_p))
    if not rows:
        continue
    rows.sort(key=lambda x: x[0])
    filtered = [r for r in rows if r[0] >= now_utc - timedelta(minutes=window_min + 1)]
    if len(filtered) < min_candles:
        continue
    first_open = filtered[0][1]
    last_close = filtered[-1][2]
    if first_open <= 0 or last_close <= 0:
        continue
    ret_pct = ((last_close - first_open) / first_open) * 100.0
    last_begin = filtered[-1][0]
    if now_utc - last_begin > timedelta(minutes=stale_min):
        stale_count += 1
    scored.append(
        {
            "ticker": ticker,
            "candles": len(filtered),
            "ret_pct": ret_pct,
        }
    )

total = len(scored)
up = sum(1 for x in scored if x["ret_pct"] > flat_band_pct)
down = sum(1 for x in scored if x["ret_pct"] < -flat_band_pct)
flat = max(0, total - up - down)
up_ratio = round((100.0 * up / total), 2) if total else 0.0

scored_up = sorted(scored, key=lambda x: x["ret_pct"], reverse=True)[:7]
scored_down = sorted(scored, key=lambda x: x["ret_pct"])[:7]
top_up = "\n".join(f"{x['ticker']} | {x['ret_pct']:.2f}% | candles={x['candles']}" for x in scored_up)
top_down = "\n".join(f"{x['ticker']} | {x['ret_pct']:.2f}% | candles={x['candles']}" for x in scored_down)

summary = f"tickers={total} up={up} down={down} flat={flat}"
err_info = ""
if errors:
    err_info = "; ".join(errors[:3])
    if len(errors) > 3:
        err_info += f"; +{len(errors)-3} more"

print(f"SUMMARY={summary}")
print(f"TICKERS_COUNT={total}")
print(f"UP_RATIO={up_ratio}")
print(f"STALE_COUNT={stale_count}")
print("TOP_UP<<EOF")
print(top_up)
print("EOF")
print("TOP_DOWN<<EOF")
print(top_down)
print("EOF")
print(f"DETAILS={err_info}")
PY

  local summary tickers up_ratio stale details top_up top_down
  summary="$(awk -F= '/^SUMMARY=/{sub(/^SUMMARY=/,""); print; exit}' "$tmp_file")"
  tickers="$(awk -F= '/^TICKERS_COUNT=/{print $2; exit}' "$tmp_file")"
  up_ratio="$(awk -F= '/^UP_RATIO=/{print $2; exit}' "$tmp_file")"
  stale="$(awk -F= '/^STALE_COUNT=/{print $2; exit}' "$tmp_file")"
  details="$(awk -F= '/^DETAILS=/{sub(/^DETAILS=/,""); print; exit}' "$tmp_file")"
  top_up="$(awk '/^TOP_UP<<EOF$/{f=1;next}/^EOF$/{if(f){f=0;exit}}f' "$tmp_file")"
  top_down="$(awk '/^TOP_DOWN<<EOF$/{f=1;next}/^EOF$/{if(f){f=0;exit}}f' "$tmp_file")"
  rm -f "$tmp_file"

  printf -v "$summary_var" '%s' "$summary"
  printf -v "$tickers_var" '%s' "${tickers:-0}"
  printf -v "$up_ratio_var" '%s' "${up_ratio:-0}"
  printf -v "$top_up_var" '%s' "$top_up"
  printf -v "$top_down_var" '%s' "$top_down"
  printf -v "$stale_var" '%s' "${stale:-0}"
  printf -v "$details_var" '%s' "$details"
}

# Дневные свечи TQBR через тот же MOEX ISS /candles.json, что и минутные; interval=24 (1 торговый день).
load_api_daily_snapshot() {
  daily_summary=""
  daily_tickers_count="0"
  daily_up_ratio="0"
  daily_stale_count="0"
  daily_state="no_data"
  daily_default_mul="1.00"
  daily_data_status="no_data"
  daily_top_up=""
  daily_top_down=""
  daily_details=""

  if [[ "$DAILY_ENABLE_RAW" != "1" && "$DAILY_ENABLE_RAW" != "true" && "$DAILY_ENABLE_RAW" != "yes" && "$DAILY_ENABLE_RAW" != "on" ]]; then
    daily_state="no_data"
    daily_default_mul="1.00"
    daily_data_status="disabled"
    daily_details="daily candles disabled by ONLINE_MIA_DAILY_ENABLE"
    return 0
  fi

  local tmp_daily
  tmp_daily="$(mktemp)"
  API_TICKERS_CSV="$(strip_spaces_csv "$API_TICKERS_RAW")" \
  DAILY_LOOKBACK_DAYS="$DAILY_LOOKBACK_DAYS" \
  DAILY_MIN_BARS="$DAILY_MIN_BARS" \
  DAILY_BULLISH_PCT="$DAILY_BULLISH_PCT" \
  DAILY_BEARISH_PCT="$DAILY_BEARISH_PCT" \
  API_STALE_MIN="$API_STALE_MIN" \
  API_TIMEOUT_SEC="$API_TIMEOUT_SEC" \
  python3 - <<'PY' >"$tmp_daily"
import json
import os
import urllib.parse
import urllib.request
from datetime import datetime, timedelta, timezone

def parse_float(v):
    try:
        return float(v)
    except Exception:
        return 0.0

def parse_begin(ts):
    if not ts:
        return None
    ts = str(ts).strip().replace(" ", "T")
    if len(ts) == 16:
        ts += ":00"
    try:
        dt = datetime.fromisoformat(ts)
        if dt.tzinfo is None:
            dt = dt.replace(tzinfo=timezone.utc)
        return dt.astimezone(timezone.utc)
    except Exception:
        return None

tickers = [t.strip().upper() for t in os.getenv("API_TICKERS_CSV", "").split(",") if t.strip()]
lookback = max(7, int(float(os.getenv("DAILY_LOOKBACK_DAYS", "45"))))
min_bars = max(1, int(float(os.getenv("DAILY_MIN_BARS", "2"))))
bull_pct = float(os.getenv("DAILY_BULLISH_PCT", "55"))
bear_pct = float(os.getenv("DAILY_BEARISH_PCT", "35"))
timeout_sec = max(2, int(float(os.getenv("API_TIMEOUT_SEC", "6"))))

now_utc = datetime.now(timezone.utc)
# Дневные бары: запас по календарю + выходные
from_dt = now_utc - timedelta(days=lookback + 14)
till_dt = now_utc + timedelta(days=1)

scored = []
stale_count = 0
errors = []

for ticker in tickers:
    params = {
        "iss.meta": "off",
        "interval": "24",
        "from": from_dt.strftime("%Y-%m-%d %H:%M:%S"),
        "till": till_dt.strftime("%Y-%m-%d %H:%M:%S"),
        "candles.columns": "begin,open,close",
    }
    url = (
        "https://iss.moex.com/iss/engines/stock/markets/shares/boards/TQBR/securities/"
        + urllib.parse.quote(ticker)
        + "/candles.json?"
        + urllib.parse.urlencode(params)
    )
    try:
        with urllib.request.urlopen(url, timeout=timeout_sec) as resp:
            payload = json.loads(resp.read().decode("utf-8", errors="ignore"))
    except Exception as exc:
        errors.append(f"{ticker}: {exc}")
        continue

    candles = payload.get("candles") or {}
    cols = candles.get("columns") or []
    data = candles.get("data") or []
    if not cols or not data:
        continue
    try:
        i_begin = cols.index("begin")
        i_open = cols.index("open")
        i_close = cols.index("close")
    except ValueError:
        continue

    rows = []
    for row in data:
        if not isinstance(row, list) or max(i_begin, i_open, i_close) >= len(row):
            continue
        begin_dt = parse_begin(row[i_begin])
        if begin_dt is None:
            continue
        open_p = parse_float(row[i_open])
        close_p = parse_float(row[i_close])
        if open_p <= 0 or close_p <= 0:
            continue
        rows.append((begin_dt, open_p, close_p))
    if not rows:
        continue
    rows.sort(key=lambda x: x[0])
    if len(rows) < min_bars:
        continue
    # Последняя дневная свеча в выборке — текущая сессионная динамика «день к дню»
    last_begin, last_open, last_close = rows[-1][0], rows[-1][1], rows[-1][2]
    last_ret = ((last_close - last_open) / last_open) * 100.0 if last_open > 0 else 0.0
    # Дневной бар сильно старше календаря — подозрение на «залипшие» данные по тикеру
    if now_utc - last_begin > timedelta(days=14):
        stale_count += 1
    red_streak = 0
    for j in range(len(rows) - 1, -1, -1):
        o, c = rows[j][1], rows[j][2]
        if o > 0 and c < o:
            red_streak += 1
        else:
            break
    sum3 = 0.0
    if len(rows) >= 3:
        for _, o, c in rows[-3:]:
            if o > 0:
                sum3 += ((c - o) / o) * 100.0
    scored.append(
        {
            "ticker": ticker,
            "last_ret": last_ret,
            "sum3_pct": sum3,
            "bars": len(rows),
            "red_streak": red_streak,
        }
    )

total = len(scored)
if total == 0:
    print("DAILY_SUMMARY=tickers=0 up=0 down=0 flat=0")
    print("DAILY_TICKERS_COUNT=0")
    print("DAILY_UP_RATIO_PCT=0")
    print("DAILY_STALE_COUNT=0")
    print("DAILY_STATE=no_data")
    print("DAILY_DEFAULT=1.00")
    print("DAILY_DATA_STATUS=no_data")
    print("TOP_DAILY_UP<<EOF")
    print("EOF")
    print("TOP_DAILY_DOWN<<EOF")
    print("EOF")
    err_info = "; ".join(errors[:5]) if errors else "no bars"
    print(f"DAILY_DETAILS={err_info}")
    raise SystemExit(0)

# "Зелёный день": close > open у последней дневной свечи
up = sum(1 for x in scored if x["last_ret"] > 0)
down = sum(1 for x in scored if x["last_ret"] < 0)
flat = max(0, total - up - down)
up_ratio = round((100.0 * up / total), 2) if total else 0.0

if up_ratio >= bull_pct:
    dstate, dmul = "bullish", 1.05
elif up_ratio <= bear_pct:
    dstate, dmul = "bearish", 0.95
else:
    dstate, dmul = "neutral", 1.00

dst = "ok"
if total < len(tickers) // 2 and len(tickers) > 2:
    dst = "partial"

scored_up = sorted(scored, key=lambda x: x["last_ret"], reverse=True)[:7]
scored_down = sorted(scored, key=lambda x: x["last_ret"])[:7]
top_up = "\n".join(
    f"{x['ticker']} | day={x['last_ret']:.2f}% sum3={x['sum3_pct']:.2f}% streak_red={x['red_streak']} bars={x['bars']}"
    for x in scored_up
)
top_down = "\n".join(
    f"{x['ticker']} | day={x['last_ret']:.2f}% sum3={x['sum3_pct']:.2f}% streak_red={x['red_streak']} bars={x['bars']}"
    for x in scored_down
)

summary = f"tickers={total} up={up} down={down} flat={flat}"
err_info = ""
if errors:
    err_info = "; ".join(errors[:3])
    if len(errors) > 3:
        err_info += f"; +{len(errors)-3} more"

print(f"DAILY_SUMMARY={summary}")
print(f"DAILY_TICKERS_COUNT={total}")
print(f"DAILY_UP_RATIO_PCT={up_ratio}")
print(f"DAILY_STALE_COUNT={stale_count}")
print(f"DAILY_STATE={dstate}")
print(f"DAILY_DEFAULT={dmul:.2f}")
print(f"DAILY_DATA_STATUS={dst}")
print("TOP_DAILY_UP<<EOF")
print(top_up)
print("EOF")
print("TOP_DAILY_DOWN<<EOF")
print(top_down)
print("EOF")
print(f"DAILY_DETAILS={err_info}")
PY

  daily_summary="$(awk -F= '/^DAILY_SUMMARY=/{sub(/^DAILY_SUMMARY=/,""); print; exit}' "$tmp_daily")"
  daily_tickers_count="$(awk -F= '/^DAILY_TICKERS_COUNT=/{print $2; exit}' "$tmp_daily")"
  daily_up_ratio="$(awk -F= '/^DAILY_UP_RATIO_PCT=/{print $2; exit}' "$tmp_daily")"
  daily_stale_count="$(awk -F= '/^DAILY_STALE_COUNT=/{print $2; exit}' "$tmp_daily")"
  daily_state="$(awk -F= '/^DAILY_STATE=/{print $2; exit}' "$tmp_daily")"
  daily_default_mul="$(awk -F= '/^DAILY_DEFAULT=/{print $2; exit}' "$tmp_daily")"
  daily_data_status="$(awk -F= '/^DAILY_DATA_STATUS=/{print $2; exit}' "$tmp_daily")"
  daily_top_up="$(awk '/^TOP_DAILY_UP<<EOF$/{f=1;next}/^EOF$/{if(f){f=0;exit}}f' "$tmp_daily")"
  daily_top_down="$(awk '/^TOP_DAILY_DOWN<<EOF$/{f=1;next}/^EOF$/{if(f){f=0;exit}}f' "$tmp_daily")"
  daily_details="$(awk -F= '/^DAILY_DETAILS=/{sub(/^DAILY_DETAILS=/,""); print; exit}' "$tmp_daily")"
  rm -f "$tmp_daily"
}

load_liquidity_rank_snapshot() {
  liquidity_order=""
  liquidity_status="no_data"
  liquidity_tickers_ok=0
  liquidity_tickers_req=0
  if [[ "$ONLINE_MIA_LIQUIDITY_ENABLE_RAW" != "1" && "$ONLINE_MIA_LIQUIDITY_ENABLE_RAW" != "true" && "$ONLINE_MIA_LIQUIDITY_ENABLE_RAW" != "yes" && "$ONLINE_MIA_LIQUIDITY_ENABLE_RAW" != "on" ]]; then
    liquidity_status="disabled"
    return 0
  fi
  local tmp_liq
  tmp_liq="$(mktemp)"
  API_TICKERS_CSV="$(strip_spaces_csv "$API_TICKERS_RAW")" \
  ONLINE_MIA_LIQUIDITY_TIMEOUT_SEC="$ONLINE_MIA_LIQUIDITY_TIMEOUT_SEC" \
  python3 - <<'PY' >"$tmp_liq"
import json
import os
import urllib.parse
import urllib.request

def parse_float(v):
    try:
        if v is None:
            return 0.0
        return float(v)
    except Exception:
        return 0.0

tickers = [t.strip().upper() for t in os.getenv("API_TICKERS_CSV", "").split(",") if t.strip()]
timeout = max(2, int(float(os.getenv("ONLINE_MIA_LIQUIDITY_TIMEOUT_SEC", "6"))))
rows = []
for ticker in tickers:
    url = (
        "https://iss.moex.com/iss/engines/stock/markets/shares/boards/TQBR/securities/"
        + urllib.parse.quote(ticker)
        + ".json?iss.meta=off&marketdata.columns=SECID,VALTODAY,NUMTRADES"
    )
    try:
        with urllib.request.urlopen(url, timeout=timeout) as resp:
            payload = json.loads(resp.read().decode("utf-8", errors="ignore"))
    except Exception:
        continue
    md = payload.get("marketdata") or {}
    cols = md.get("columns") or []
    data = md.get("data") or []
    if not cols or not data:
        continue
    try:
        i_val = cols.index("VALTODAY")
        i_nt = cols.index("NUMTRADES") if "NUMTRADES" in cols else -1
    except ValueError:
        continue
    row = data[0]
    valtoday = parse_float(row[i_val]) if i_val < len(row) else 0.0
    numtrades = int(parse_float(row[i_nt])) if i_nt >= 0 and i_nt < len(row) else 0
    rows.append((ticker, valtoday, numtrades))

rows.sort(key=lambda x: (-x[1], -x[2], x[0]))
order = ",".join(t[0] for t in rows)
req = len(tickers)
ok = len(rows)
if ok == 0:
    status = "no_data"
elif ok < req:
    status = "partial"
else:
    status = "ok"
print(f"MIYA_LIQUIDITY_ORDER={order}")
print(f"MIYA_LIQUIDITY_STATUS={status}")
print(f"MIYA_LIQUIDITY_TICKERS_OK={ok}")
print(f"MIYA_LIQUIDITY_TICKERS_REQ={req}")
PY
  if [[ -s "$tmp_liq" ]]; then
    liquidity_order="$(grep '^MIYA_LIQUIDITY_ORDER=' "$tmp_liq" | head -1 | cut -d= -f2-)"
    liquidity_status="$(grep '^MIYA_LIQUIDITY_STATUS=' "$tmp_liq" | head -1 | cut -d= -f2-)"
    liquidity_tickers_ok="$(grep '^MIYA_LIQUIDITY_TICKERS_OK=' "$tmp_liq" | head -1 | cut -d= -f2-)"
    liquidity_tickers_req="$(grep '^MIYA_LIQUIDITY_TICKERS_REQ=' "$tmp_liq" | head -1 | cut -d= -f2-)"
  fi
  rm -f "$tmp_liq"
}

load_orderbook_snapshot() {
  orderbook_status="disabled"
  orderbook_source=""
  orderbook_tickers_ok="0"
  orderbook_tickers_req="0"
  orderbook_spread_bps_median=""
  orderbook_imbalance_median=""
  orderbook_note=""
  out_ob_md="${REPORTS_DIR}/online_orderbook_latest.md"
  out_ob_json="${REPORTS_DIR}/online_orderbook_latest.json"

  if [[ "$ORDERBOOK_ENABLE_RAW" != "1" && "$ORDERBOOK_ENABLE_RAW" != "true" && "$ORDERBOOK_ENABLE_RAW" != "yes" && "$ORDERBOOK_ENABLE_RAW" != "on" ]]; then
    orderbook_status="disabled"
    orderbook_source="disabled"
    orderbook_note="orderbook disabled by ONLINE_MIA_ORDERBOOK_ENABLE"
    {
      echo "# Мия: стакан / лучшие цены"
      echo
      echo "- Статус: **disabled** (\`ONLINE_MIA_ORDERBOOK_ENABLE\`)"
    } > "$out_ob_md"
    printf '%s\n' '{"source":"disabled","status":"disabled"}' > "$out_ob_json"
    return 0
  fi

  local tickers_csv
  tickers_csv="$(strip_spaces_csv "${ORDERBOOK_TICKERS_RAW}")"
  if [[ -z "$tickers_csv" ]]; then
    tickers_csv="$(strip_spaces_csv "$API_TICKERS_RAW")"
  fi
  if [[ -z "$tickers_csv" ]]; then
    orderbook_status="no_data"
    orderbook_note="no tickers (set ONLINE_MIA_ORDERBOOK_TICKERS or ONLINE_MIA_API_TICKERS)"
    {
      echo "# Мия: стакан / лучшие цены (MOEX ISS)"
      echo
      echo "- Статус: **no_data** — нет списка тикеров."
    } > "$out_ob_md"
    printf '%s\n' '{"source":"moex_iss","status":"no_data","note":"no tickers"}' > "$out_ob_json"
    return 0
  fi

  local tmp_ob
  tmp_ob="$(mktemp)"
  MSK_NOW="${msk_now}" \
    OUT_ORDERBOOK_MD="${out_ob_md}" \
    OUT_ORDERBOOK_JSON="${out_ob_json}" \
    ORDERBOOK_TICKERS_CSV="${tickers_csv}" \
    ORDERBOOK_TIMEOUT_SEC="${ORDERBOOK_TIMEOUT_SEC}" \
    python3 - <<'PY' >"$tmp_ob"
import json
import os
import statistics
import urllib.parse
import urllib.request

def parse_float(v):
    try:
        if v is None:
            return None
        x = float(v)
        return x if x > 0 else None
    except Exception:
        return None

tickers = [t.strip().upper() for t in os.getenv("ORDERBOOK_TICKERS_CSV", "").split(",") if t.strip()]
timeout = max(2, int(float(os.getenv("ORDERBOOK_TIMEOUT_SEC", "4"))))
out_md = os.getenv("OUT_ORDERBOOK_MD", "")
out_json = os.getenv("OUT_ORDERBOOK_JSON", "")
msk = os.getenv("MSK_NOW", "")

rows = []
errors = []
for ticker in tickers:
    url = (
        "https://iss.moex.com/iss/engines/stock/markets/shares/boards/TQBR/securities/"
        + urllib.parse.quote(ticker)
        + ".json?iss.meta=off&marketdata.columns=SECID,BID,OFFER,LAST,BIDDEPTH,OFFERDEPTH"
    )
    try:
        with urllib.request.urlopen(url, timeout=timeout) as resp:
            payload = json.loads(resp.read().decode("utf-8", errors="ignore"))
    except Exception as exc:
        errors.append(f"{ticker}: {exc}")
        continue
    md = payload.get("marketdata") or {}
    cols = md.get("columns") or []
    data = md.get("data") or []
    if not cols or not data:
        errors.append(f"{ticker}: no marketdata")
        continue
    try:
        i_bid = cols.index("BID")
        i_off = cols.index("OFFER")
        i_last = cols.index("LAST") if "LAST" in cols else -1
        i_bd = cols.index("BIDDEPTH") if "BIDDEPTH" in cols else -1
        i_od = cols.index("OFFERDEPTH") if "OFFERDEPTH" in cols else -1
    except ValueError:
        continue
    row = data[0]
    if max(i_bid, i_off) >= len(row):
        continue
    bid = parse_float(row[i_bid])
    offer = parse_float(row[i_off])
    lastp = parse_float(row[i_last]) if i_last >= 0 and i_last < len(row) else None
    bd = parse_float(row[i_bd]) if i_bd >= 0 and i_bd < len(row) else None
    od = parse_float(row[i_od]) if i_od >= 0 and i_od < len(row) else None
    spread = None
    spread_bps = None
    mid = None
    imb = None
    if bid and offer:
        spread = offer - bid
        mid = (bid + offer) / 2.0
        if mid > 0:
            spread_bps = (spread / mid) * 10000.0
    if bd is not None and od is not None and (bd + od) > 0:
        imb = (bd - od) / (bd + od)
    rows.append(
        {
            "ticker": ticker,
            "bid": bid,
            "offer": offer,
            "last": lastp,
            "bid_depth": bd,
            "offer_depth": od,
            "spread": spread,
            "spread_bps": spread_bps,
            "imbalance": imb,
        }
    )

spreads = [r["spread_bps"] for r in rows if r.get("spread_bps") is not None]
imbs = [r["imbalance"] for r in rows if r.get("imbalance") is not None]
ok = len([r for r in rows if r.get("bid") and r.get("offer")])

def med(xs):
    if not xs:
        return None
    return round(float(statistics.median(xs)), 4)

spread_med = med(spreads)
imb_med = med(imbs)

status = "no_data"
if ok >= 5:
    status = "ok"
elif ok >= 1:
    status = "partial"
else:
    status = "no_data"

obj = {
    "generated_msk": msk,
    "source": "moex_iss",
    "scope": "best_bid_offer_last_depth",
    "note": "Полный многоуровневый стакан в ISS часто только по подписке; здесь — публичные best bid/offer и глубины, если биржа заполняет колонки.",
    "status": status,
    "tickers_requested": len(tickers),
    "tickers_with_quotes": ok,
    "spread_bps_median": spread_med,
    "imbalance_median": imb_med,
    "rows": rows,
    "errors": errors[:20],
}

if out_json:
    with open(out_json, "w", encoding="utf-8") as f:
        json.dump(obj, f, ensure_ascii=False, indent=2)

md = []
md.append("# Мия: стакан / лучшие цены (MOEX ISS, TQBR)")
md.append("")
md.append(f"- Время (MSK): {msk}")
md.append("- Источник: **MOEX ISS** `marketdata` (best bid / best offer / last; глубины — если заполнены)")
md.append(f"- Статус агрегата: **{status}** (тикеров с bid+ask: {ok} / {len(tickers)})")
if spread_med is not None:
    md.append(f"- Медиана спреда: **{spread_med}** bps (по тикерам, где есть bid и ask)")
else:
    md.append("- Медиана спреда: *н/д*")
if imb_med is not None:
    md.append(f"- Медиана дисбаланса глубины (bid-offer)/(bid+offer): **{imb_med}**")
else:
    md.append("- Дисбаланс глубины: *н/д* (колонки глубины часто пустые вне основной сессии)")
md.append("")
md.append("| ticker | bid | offer | last | spread (bps) | bid_depth | offer_depth | imbalance |")
md.append("|---|---:|---:|---:|---:|---:|---:|---:|")
for r in sorted(rows, key=lambda x: x["ticker"]):
    sp = r.get("spread_bps")
    sp_s = f"{sp:.2f}" if sp is not None else ""
    imb = r.get("imbalance")
    imb_s = f"{imb:.4f}" if imb is not None else ""
    bd = r.get("bid_depth")
    od = r.get("offer_depth")
    md.append(
        f"| {r['ticker']} | {r.get('bid') or ''} | {r.get('offer') or ''} | {r.get('last') or ''} | {sp_s} | {bd or ''} | {od or ''} | {imb_s} |"
    )
if errors:
    md.append("")
    md.append("### Ошибки / пропуски")
    for e in errors[:12]:
        md.append(f"- {e}")

if out_md:
    with open(out_md, "w", encoding="utf-8") as f:
        f.write("\n".join(md) + "\n")

print(f"ORDERBOOK_STATUS={status}")
print("ORDERBOOK_SOURCE=moex_iss")
print(f"ORDERBOOK_TICKERS_OK={ok}")
print(f"ORDERBOOK_TICKERS_REQ={len(tickers)}")
if spread_med is not None:
    print(f"ORDERBOOK_SPREAD_BPS_MEDIAN={spread_med}")
else:
    print("ORDERBOOK_SPREAD_BPS_MEDIAN=")
if imb_med is not None:
    print(f"ORDERBOOK_IMBALANCE_MEDIAN={imb_med}")
else:
    print("ORDERBOOK_IMBALANCE_MEDIAN=")
PY

  orderbook_status="$(awk -F= '/^ORDERBOOK_STATUS=/{print $2; exit}' "$tmp_ob")"
  orderbook_source="$(awk -F= '/^ORDERBOOK_SOURCE=/{print $2; exit}' "$tmp_ob")"
  orderbook_tickers_ok="$(awk -F= '/^ORDERBOOK_TICKERS_OK=/{print $2; exit}' "$tmp_ob")"
  orderbook_tickers_req="$(awk -F= '/^ORDERBOOK_TICKERS_REQ=/{print $2; exit}' "$tmp_ob")"
  orderbook_spread_bps_median="$(awk -F= '/^ORDERBOOK_SPREAD_BPS_MEDIAN=/{print $2; exit}' "$tmp_ob")"
  orderbook_imbalance_median="$(awk -F= '/^ORDERBOOK_IMBALANCE_MEDIAN=/{print $2; exit}' "$tmp_ob")"
  rm -f "$tmp_ob"
}

msk_now="$(TZ="$TZ_MSK" date '+%Y-%m-%d %H:%M:%S')"
epoch_now="$(date +%s)"
out_md="${REPORTS_DIR}/online_candles_latest.md"
out_json="${REPORTS_DIR}/online_candles_latest.json"
signal_env="${REPORTS_DIR}/miya_online_signal.env"
out_orderbook_md="${REPORTS_DIR}/online_orderbook_latest.md"
out_orderbook_json="${REPORTS_DIR}/online_orderbook_latest.json"

if [[ "$WINDOW_FALLBACK_MIN" -lt "$WINDOW_LONG_MIN" ]]; then
  WINDOW_FALLBACK_MIN="$WINDOW_LONG_MIN"
fi

window_long_used="$WINDOW_LONG_MIN"
summary_short=""
summary_long=""
tickers_short_count="0"
tickers_long_count="0"
up_ratio_short="0"
up_ratio_long="0"
top_up_long=""
top_down_long=""
stale_short_count="0"
stale_long_count="0"
api_short_details=""
api_long_details=""
data_source="db"

if [[ "$DATA_SOURCE_MODE_RAW" != "db_only" ]]; then
  load_api_snapshot "$WINDOW_SHORT_MIN" "short"
  load_api_snapshot "$window_long_used" "long"
  if awk "BEGIN{exit !(${long_tickers_count:-0} > 0)}"; then
    summary_short="$short_summary"
    summary_long="$long_summary"
    tickers_short_count="${short_tickers_count:-0}"
    tickers_long_count="${long_tickers_count:-0}"
    up_ratio_short="${short_up_ratio:-0}"
    up_ratio_long="${long_up_ratio:-0}"
    top_up_long="$long_top_up"
    top_down_long="$long_top_down"
    stale_short_count="${short_stale_count:-0}"
    stale_long_count="${long_stale_count:-0}"
    api_short_details="$short_api_details"
    api_long_details="$long_api_details"
    data_source="api"
  fi
fi

if [[ "$data_source" == "db" ]] && [[ "$db_available" == "1" ]]; then
  load_db_snapshot "$window_long_used" "long"
  load_db_snapshot "$WINDOW_SHORT_MIN" "short"
  summary_short="$short_summary"
  summary_long="$long_summary"
  tickers_short_count="${short_tickers_count:-0}"
  tickers_long_count="${long_tickers_count:-0}"
  up_ratio_short="${short_up_ratio:-0}"
  up_ratio_long="${long_up_ratio:-0}"
  top_up_long="$long_top_up"
  top_down_long="$long_top_down"
fi
if [[ "$data_source" == "db" ]] && [[ "$db_available" != "1" ]]; then
  summary_short="tickers=0 up=0 down=0 flat=0"
  summary_long="tickers=0 up=0 down=0 flat=0"
  tickers_short_count="0"
  tickers_long_count="0"
  up_ratio_short="0"
  up_ratio_long="0"
  data_status="api_no_data_no_db"
fi

market_state="neutral"
default_mul="1.00"
data_status="${data_status:-ok}"
up_ratio_num="${up_ratio_long:-0}"
tickers_long_num="${tickers_long_count:-0}"
if awk "BEGIN{exit !(${tickers_long_num} <= 0)}"; then
  if [[ "$USE_FALLBACK_ACTIVE_RAW" == "1" || "$USE_FALLBACK_ACTIVE_RAW" == "true" || "$USE_FALLBACK_ACTIVE_RAW" == "yes" || "$USE_FALLBACK_ACTIVE_RAW" == "on" ]]; then
    window_long_used="$WINDOW_FALLBACK_MIN"
    if [[ "$data_source" == "api" ]]; then
      load_api_snapshot "$window_long_used" "long"
      summary_long="$long_summary"
      tickers_long_count="${long_tickers_count:-0}"
      up_ratio_long="${long_up_ratio:-0}"
      top_up_long="$long_top_up"
      top_down_long="$long_top_down"
      stale_long_count="${long_stale_count:-0}"
      api_long_details="$long_api_details"
    else
      if [[ "$db_available" == "1" ]]; then
        load_db_snapshot "$window_long_used" "long"
        summary_long="$long_summary"
        tickers_long_count="${long_tickers_count:-0}"
        up_ratio_long="${long_up_ratio:-0}"
        top_up_long="$long_top_up"
        top_down_long="$long_top_down"
      else
        summary_long="tickers=0 up=0 down=0 flat=0"
        tickers_long_count="0"
        up_ratio_long="0"
        top_up_long=""
        top_down_long=""
      fi
    fi
    up_ratio_num="${up_ratio_long:-0}"
    tickers_long_num="${tickers_long_count:-0}"
    if awk "BEGIN{exit !(${tickers_long_num} > 0)}"; then
      data_status="fallback_active"
    else
      data_status="no_data"
    fi
  else
    data_status="no_data"
  fi
fi
if awk "BEGIN{exit !(${tickers_long_num} <= 0)}"; then
  market_state="no_data"
  default_mul="1.00"
elif [[ "$data_source" == "api" ]] && awk "BEGIN{exit !(${stale_long_count:-0} > 0)}"; then
  data_status="stale_api"
elif awk "BEGIN{exit !(${up_ratio_num} >= ${BULLISH_THRESHOLD_PCT})}"; then
  market_state="bullish"
  default_mul="1.05"
elif awk "BEGIN{exit !(${up_ratio_num} <= ${BEARISH_THRESHOLD_PCT})}"; then
  market_state="bearish"
  default_mul="0.95"
fi

alert_no_data=0
alert_note=""
if awk "BEGIN{exit !(${tickers_long_num} <= 0)}"; then
  alert_no_data=1
  alert_note="no valid candles in ${window_long_used}m window (source=${data_source})"
fi

orderbook_enabled_flag=0
if [[ "$ORDERBOOK_ENABLE_RAW" == "1" || "$ORDERBOOK_ENABLE_RAW" == "true" || "$ORDERBOOK_ENABLE_RAW" == "yes" || "$ORDERBOOK_ENABLE_RAW" == "on" ]]; then
  orderbook_enabled_flag=1
fi
load_orderbook_snapshot
load_api_daily_snapshot
load_liquidity_rank_snapshot

ob_spread_json="null"
if [[ -n "${orderbook_spread_bps_median}" ]]; then
  ob_spread_json="${orderbook_spread_bps_median}"
fi
ob_imb_json="null"
if [[ -n "${orderbook_imbalance_median}" ]]; then
  ob_imb_json="${orderbook_imbalance_median}"
fi
stock_watch_csv="$(strip_spaces_csv "$API_TICKERS_RAW")"
futures_watch_csv="$(strip_spaces_csv "$FUTURES_WATCH_TICKERS_RAW")"
watch_csv="$(merge_csv_unique_upper "$stock_watch_csv" "$futures_watch_csv")"

{
  echo "# Мия: онлайн-монитор свечей"
  echo
  echo "- Время (MSK): ${msk_now}"
  echo "- Источник свечей: ${data_source}"
  echo "- Состояние рынка (long-window ${window_long_used}m): ${market_state}"
  echo "- Доля растущих (>${FLAT_BAND_PCT}%): ${up_ratio_num}%"
  if [[ "${data_source}" == "api" ]]; then
    echo "- API stale (long): ${stale_long_count} тикеров старше ${API_STALE_MIN}m"
  fi
  if [[ "${data_status}" == "no_data" ]]; then
    echo "- Примечание: данных по свечам в окне ${window_long_used}m пока недостаточно."
  elif [[ "${data_status}" == "fallback_active" ]]; then
    echo "- Примечание: применён fallback-режим (${window_long_used}m окно, см. ONLINE_MIA_CANDLE_FALLBACK_MIN)."
  elif [[ "${data_status}" == "api_no_data_no_db" ]]; then
    echo "- Примечание: API не дал валидных свечей, fallback в БД недоступен."
  elif [[ "${data_status}" == "stale_api" ]]; then
    echo "- Примечание: API вернул данные, но часть long-свечей устарела."
  fi
  if [[ "${data_source}" == "api" && -n "${api_long_details}" ]]; then
    echo "- API детали: ${api_long_details}"
  fi
  if [[ "${alert_no_data}" == "1" ]]; then
    echo "- ALERT: ${alert_note}"
  fi
  echo
  echo "## Стакан / лучшие цены (MOEX ISS, см. также \`online_orderbook_latest.md\`)"
  echo "- Статус: **${orderbook_status}** (источник: ${orderbook_source:-n/a})"
  echo "- Тикеров с bid+ask: **${orderbook_tickers_ok:-0}** / ${orderbook_tickers_req:-0}"
  if [[ -n "${orderbook_spread_bps_median}" ]]; then
    echo "- Медиана спреда (bps): **${orderbook_spread_bps_median}**"
  else
    echo "- Медиана спреда (bps): *н/д*"
  fi
  if [[ -n "${orderbook_imbalance_median}" ]]; then
    echo "- Медиана дисбаланса глубины: **${orderbook_imbalance_median}**"
  else
    echo "- Медиана дисбаланса глубины: *н/д*"
  fi
  echo
  echo "## Ликвидность (MOEX ISS \`VALTODAY\`, порядок для Мухли)"
  echo "- Статус: **${liquidity_status:-no_data}**; тикеров с данными: **${liquidity_tickers_ok:-0}** / ${liquidity_tickers_req:-0}"
  if [[ -n "${liquidity_order}" ]]; then
    echo "- Порядок (от более ликвидного): \`${liquidity_order}\`"
  else
    echo "- Порядок: *н/д*"
  fi
  echo
  echo "## Snapshot ${WINDOW_SHORT_MIN}m"
  echo "- ${summary_short:-нет данных}"
  echo
  echo "## Snapshot ${window_long_used}m"
  echo "- ${summary_long:-нет данных}"
  echo
  echo "### Лидеры роста (${window_long_used}m)"
  if [[ -n "$top_up_long" ]]; then
    while IFS= read -r line; do
      [[ -n "$line" ]] && echo "- ${line}"
    done <<< "$top_up_long"
  else
    echo "- нет данных"
  fi
  echo
  echo "### Лидеры снижения (${window_long_used}m)"
  if [[ -n "$top_down_long" ]]; then
    while IFS= read -r line; do
      [[ -n "$line" ]] && echo "- ${line}"
    done <<< "$top_down_long"
  else
    echo "- нет данных"
  fi
  echo
  echo "## Дневной контекст (MOEX ISS \`candles.json\`, \`interval=24\`, тот же список тикеров)"
  echo "- Статус данных: **${daily_data_status:-no_data}**; **STATE=${daily_state:-no_data}**; **DEFAULT=${daily_default_mul:-1.00}**"
  echo "- Доля «зелёных» дней (последняя дневная свеча, close>open): **${daily_up_ratio:-0}%** (тикеров: ${daily_tickers_count:-0})"
  echo "- Пороги (как у intraday, отдельные env): bullish ≥**${DAILY_BULLISH_PCT}%** / bearish ≤**${DAILY_BEARISH_PCT}%**"
  if [[ -n "${daily_details:-}" ]]; then
    echo "- Примечание: ${daily_details}"
  fi
  echo
  echo "### Дневные лидеры роста (последняя дневная свеча)"
  if [[ -n "${daily_top_up:-}" ]]; then
    while IFS= read -r line; do
      [[ -n "$line" ]] && echo "- ${line}"
    done <<< "$daily_top_up"
  else
    echo "- нет данных"
  fi
  echo
  echo "### Дневные лидеры снижения (последняя дневная свеча; полезно для ловли «ножа»)"
  if [[ -n "${daily_top_down:-}" ]]; then
    while IFS= read -r line; do
      [[ -n "$line" ]] && echo "- ${line}"
    done <<< "$daily_top_down"
  else
    echo "- нет данных"
  fi
} > "$out_md"

{
  echo "# miya online signal"
  echo "# updated: ${msk_now}"
  echo "STATE=${market_state}"
  echo "DATA_STATUS=${data_status}"
  echo "DATA_SOURCE=${data_source}"
  echo "DEFAULT=${default_mul}"
  echo "UP_RATIO_PCT=${up_ratio_num}"
  echo "TICKERS_LONG_COUNT=${tickers_long_num}"
  echo "TICKERS_SHORT_COUNT=${tickers_short_count:-0}"
  echo "WINDOW_SHORT_MIN=${WINDOW_SHORT_MIN}"
  echo "WINDOW_LONG_MIN=${WINDOW_LONG_MIN}"
  echo "WINDOW_LONG_USED_MIN=${window_long_used}"
  echo "API_STALE_LONG_COUNT=${stale_long_count:-0}"
  echo "API_STALE_SHORT_COUNT=${stale_short_count:-0}"
  echo "ALERT_NO_DATA=${alert_no_data}"
  echo "ORDERBOOK_ENABLED=${orderbook_enabled_flag}"
  echo "ORDERBOOK_STATUS=${orderbook_status}"
  echo "ORDERBOOK_SOURCE=${orderbook_source}"
  echo "ORDERBOOK_TICKERS_OK=${orderbook_tickers_ok:-0}"
  echo "ORDERBOOK_TICKERS_REQ=${orderbook_tickers_req:-0}"
  echo "ORDERBOOK_SPREAD_BPS_MEDIAN=${orderbook_spread_bps_median}"
  echo "ORDERBOOK_IMBALANCE_MEDIAN=${orderbook_imbalance_median}"
  echo "DAILY_DATA_STATUS=${daily_data_status:-no_data}"
  echo "DAILY_STATE=${daily_state:-no_data}"
  echo "DAILY_DEFAULT=${daily_default_mul:-1.00}"
  echo "DAILY_UP_RATIO_PCT=${daily_up_ratio:-0}"
  echo "DAILY_TICKERS_COUNT=${daily_tickers_count:-0}"
  echo "DAILY_STALE_COUNT=${daily_stale_count:-0}"
  echo "DAILY_SUMMARY=$(printf '%s' "${daily_summary:-}" | tr '\n' ' ')"
  echo "MIYA_LIQUIDITY_ORDER=${liquidity_order}"
  echo "MIYA_LIQUIDITY_STATUS=${liquidity_status}"
  echo "MIYA_LIQUIDITY_TICKERS_OK=${liquidity_tickers_ok:-0}"
  echo "MIYA_LIQUIDITY_TICKERS_REQ=${liquidity_tickers_req:-0}"
  echo "MIYA_WATCH_STOCK_TICKERS=${stock_watch_csv}"
  echo "MIYA_WATCH_FUTURES_TICKERS=${futures_watch_csv}"
  echo "MIYA_WATCH_TICKERS=${watch_csv}"
} > "$signal_env"

cat > "$out_json" <<EOF
{
  "generated_msk": "${msk_now}",
  "state": "$(json_escape "$market_state")",
  "data_status": "$(json_escape "$data_status")",
  "data_source": "$(json_escape "$data_source")",
  "tickers_long_count": ${tickers_long_num:-0},
  "tickers_short_count": ${tickers_short_count:-0},
  "up_ratio_pct": ${up_ratio_num:-0},
  "api_stale_long_count": ${stale_long_count:-0},
  "api_stale_short_count": ${stale_short_count:-0},
  "window_short_min": ${WINDOW_SHORT_MIN},
  "window_long_min": ${WINDOW_LONG_MIN},
  "window_long_used_min": ${window_long_used},
  "alert_no_data": ${alert_no_data},
  "summary_short": "$(json_escape "${summary_short:-}")",
  "summary_long": "$(json_escape "${summary_long:-}")",
  "orderbook": {
    "status": "$(json_escape "${orderbook_status}")",
    "source": "$(json_escape "${orderbook_source}")",
    "tickers_ok": ${orderbook_tickers_ok:-0},
    "tickers_requested": ${orderbook_tickers_req:-0},
    "spread_bps_median": ${ob_spread_json},
    "imbalance_median": ${ob_imb_json},
    "snapshot_md": "$(json_escape "$out_orderbook_md")",
    "snapshot_json": "$(json_escape "$out_orderbook_json")"
  },
  "daily": {
    "data_status": "$(json_escape "${daily_data_status:-}")",
    "state": "$(json_escape "${daily_state:-}")",
    "up_ratio_pct": ${daily_up_ratio:-0},
    "tickers_count": ${daily_tickers_count:-0},
    "stale_count": ${daily_stale_count:-0},
    "default_mul": $(awk -v x="${daily_default_mul:-1}" 'BEGIN{printf "%.4f", x+0}'),
    "lookback_days": ${DAILY_LOOKBACK_DAYS},
    "summary": "$(json_escape "${daily_summary:-}")"
  },
  "liquidity": {
    "order": "$(json_escape "${liquidity_order:-}")",
    "status": "$(json_escape "${liquidity_status:-}")",
    "tickers_ok": ${liquidity_tickers_ok:-0},
    "tickers_req": ${liquidity_tickers_req:-0},
    "watch_tickers": "$(json_escape "${watch_csv}")"
  },
  "signal_env": "$(json_escape "$signal_env")"
}
EOF

last_no_data_alert_epoch=0
if [[ -f "$ONLINE_STATE_FILE" ]]; then
  last_no_data_alert_epoch="$(awk -F= '/^last_no_data_alert_epoch=/{print $2}' "$ONLINE_STATE_FILE" 2>/dev/null || echo 0)"
fi
if [[ -z "$last_no_data_alert_epoch" ]]; then
  last_no_data_alert_epoch=0
fi
if [[ ! "${NO_DATA_ALERT_EVERY_MIN}" =~ ^[0-9]+$ ]] || (( NO_DATA_ALERT_EVERY_MIN < 1 )); then
  NO_DATA_ALERT_EVERY_MIN=10
fi
if (( NO_DATA_ALERT_EVERY_MIN > 240 )); then
  NO_DATA_ALERT_EVERY_MIN=240
fi
no_data_alert_sec=$((NO_DATA_ALERT_EVERY_MIN * 60))
if [[ "${alert_no_data}" == "1" ]] && (( epoch_now - last_no_data_alert_epoch >= no_data_alert_sec )); then
  echo "miya-online: ALERT ${alert_note}"
  last_no_data_alert_epoch="${epoch_now}"
fi
cat > "$ONLINE_STATE_FILE" <<EOF
last_run_epoch=${epoch_now}
last_run_msk=${msk_now}
last_state=${market_state}
last_data_status=${data_status}
last_tickers_long_count=${tickers_long_num}
last_no_data_alert_epoch=${last_no_data_alert_epoch}
EOF

echo "miya-online: wrote ${out_md}"
echo "miya-online: wrote ${out_json}"
echo "miya-online: wrote ${signal_env}"
echo "miya-online: wrote ${out_orderbook_md}"
echo "miya-online: wrote ${out_orderbook_json}"
exit 0
