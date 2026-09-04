"""Значения по умолчанию JSON-политики — в одной линии с Go (services.AgentPolicy + tariff defaults)."""

from __future__ import annotations

import json
from pathlib import Path
from typing import Any

# Синхронизировать с moex-trader-gui/services/moex_trading_assistant.go и tariff_finam.go при изменении.
DEFAULT_POLICY: dict[str, Any] = {
    "version": 1,
    "aggression": 1.0,
    "min_confidence": 0.55,
    "buy_budget_pct": 0.06,
    "sell_budget_pct": 0.05,
    "rsi_buy_below": 28.0,
    "rsi_sell_above": 72.0,
    "min_stock_notional_rub": 5000.0,
    "notes": "",
}


def load_policy(path: str | Path) -> dict[str, Any]:
    p = Path(path)
    raw = json.loads(p.read_text(encoding="utf-8"))
    if not isinstance(raw, dict):
        raise ValueError("policy JSON must be an object")
    merged = {**DEFAULT_POLICY, **raw}
    return merged
