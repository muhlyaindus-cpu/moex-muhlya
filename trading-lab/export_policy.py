from __future__ import annotations

import sys
from pathlib import Path

_LAB = Path(__file__).resolve().parent
if str(_LAB) not in sys.path:
    sys.path.insert(0, str(_LAB))

import argparse
import json
import os

import numpy as np
from stable_baselines3 import PPO

from policy_defaults import DEFAULT_POLICY


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("--model", required=True)
    ap.add_argument("--policy-out", required=True)
    ap.add_argument("--rsi-buy-below", type=float, default=None)
    ap.add_argument("--rsi-sell-above", type=float, default=None)
    ap.add_argument("--buy-budget-pct", type=float, default=None)
    ap.add_argument("--sell-budget-pct", type=float, default=None)
    ap.add_argument("--min-stock-notional-rub", type=float, default=None)
    args = ap.parse_args()

    model = PPO.load(args.model)

    # Очень простой “экспорт”: мы не тащим нейросеть в Go на первой итерации.
    # Вместо этого берём несколько глобальных параметров, влияющих на поведение
    # нашего Go-агента (агрессия, пороги уверенности и т.п.).
    #
    # Идея: модель учится в лаборатории, а в проде мы применяем результат
    # как “настройку поведения”, оставляя риск-менеджер и контроль за нами.
    rng = np.random.default_rng(42)
    sample = rng.normal(size=(256, model.observation_space.shape[0])).astype(np.float32)
    actions, _ = model.predict(sample, deterministic=True)

    buy_rate = float(np.mean(actions == 1))
    sell_rate = float(np.mean(actions == 2))

    aggression = 0.9 + 0.9 * buy_rate  # 0.9..1.8
    min_confidence = 0.55 + 0.15 * sell_rate  # 0.55..0.70

    policy = {
        **DEFAULT_POLICY,
        "version": 2,
        "aggression": round(aggression, 3),
        "min_confidence": round(min_confidence, 3),
        "notes": "Exported from PPO lab. Safe: parameters only (no NN in Go).",
    }
    if args.rsi_buy_below is not None:
        policy["rsi_buy_below"] = float(args.rsi_buy_below)
    if args.rsi_sell_above is not None:
        policy["rsi_sell_above"] = float(args.rsi_sell_above)
    if args.buy_budget_pct is not None:
        policy["buy_budget_pct"] = float(args.buy_budget_pct)
    if args.sell_budget_pct is not None:
        policy["sell_budget_pct"] = float(args.sell_budget_pct)
    if args.min_stock_notional_rub is not None:
        policy["min_stock_notional_rub"] = float(args.min_stock_notional_rub)

    os.makedirs(os.path.dirname(args.policy_out) or ".", exist_ok=True)
    with open(args.policy_out, "w", encoding="utf-8") as f:
        json.dump(policy, f, ensure_ascii=False, indent=2)
    print(f"Saved policy: {args.policy_out}")


if __name__ == "__main__":
    main()

