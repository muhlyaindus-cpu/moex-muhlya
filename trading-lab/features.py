from __future__ import annotations

import numpy as np
import pandas as pd


def rsi(close: pd.Series, period: int = 14) -> pd.Series:
    delta = close.diff()
    up = delta.clip(lower=0.0)
    down = -delta.clip(upper=0.0)
    roll_up = up.ewm(alpha=1 / period, adjust=False).mean()
    roll_down = down.ewm(alpha=1 / period, adjust=False).mean()
    rs = roll_up / (roll_down.replace(0, np.nan))
    out = 100 - (100 / (1 + rs))
    return out.fillna(50.0)


def add_features(df: pd.DataFrame) -> pd.DataFrame:
    df = df.copy()
    df["ret1"] = df["close"].pct_change().fillna(0.0)
    df["sma10"] = df["close"].rolling(10).mean().bfill()
    df["sma20"] = df["close"].rolling(20).mean().bfill()
    df["rsi14"] = rsi(df["close"], 14)

    # нормализации (простые, чтобы не переусложнять первую итерацию)
    df["sma10_rel"] = (df["sma10"] / df["close"]).replace([np.inf, -np.inf], np.nan).fillna(1.0)
    df["sma20_rel"] = (df["sma20"] / df["close"]).replace([np.inf, -np.inf], np.nan).fillna(1.0)
    df["rsi14_n"] = (df["rsi14"] / 100.0).clip(0.0, 1.0)
    return df


FEATURE_COLS = ["ret1", "sma10_rel", "sma20_rel", "rsi14_n"]

