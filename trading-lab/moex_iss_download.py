import argparse
import os
from datetime import datetime

import pandas as pd
import requests


ISS_BASE = "https://iss.moex.com/iss"


def fetch_candles(symbol: str, engine: str, market: str, interval: int) -> pd.DataFrame:
    url = f"{ISS_BASE}/engines/{engine}/markets/{market}/securities/{symbol}/candles.json"
    params = {
        "iss.meta": "off",
        "from": "2000-01-01",
        "till": datetime.utcnow().strftime("%Y-%m-%d"),
        "interval": str(interval),
    }
    r = requests.get(url, params=params, timeout=30)
    r.raise_for_status()
    payload = r.json()

    candles = payload.get("candles", {})
    columns = candles.get("columns", [])
    data = candles.get("data", [])
    if not columns or not data:
        return pd.DataFrame()

    df = pd.DataFrame(data, columns=columns)
    # стандартизируем имена
    df.columns = [c.lower() for c in df.columns]

    # типы
    if "begin" in df.columns:
        df["begin"] = pd.to_datetime(df["begin"], errors="coerce", utc=True)
    if "end" in df.columns:
        df["end"] = pd.to_datetime(df["end"], errors="coerce", utc=True)

    for col in ["open", "high", "low", "close", "value", "volume"]:
        if col in df.columns:
            df[col] = pd.to_numeric(df[col], errors="coerce")

    df = df.dropna(subset=["close"]).sort_values("begin").reset_index(drop=True)
    return df


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("--symbol", required=True)
    ap.add_argument("--engine", default="stock", choices=["stock", "futures"])
    ap.add_argument("--market", default="shares", help="shares/forts/…")
    ap.add_argument("--interval", type=int, default=24, help="24=1d, 60=1h, etc (MOEX ISS)")
    ap.add_argument("--out", required=True)
    args = ap.parse_args()

    df = fetch_candles(args.symbol, args.engine, args.market, args.interval)
    os.makedirs(os.path.dirname(args.out), exist_ok=True)
    df.to_csv(args.out, index=False)
    print(f"Saved: {args.out} rows={len(df)}")


if __name__ == "__main__":
    main()

