"""
Упрощённый бэктест стратегии «RSI + доли от кэша», согласованный с идеей MOEXTradingAssistant в Go.
Не копирует построчно весь адаптивный код агента — фиксирует агрессию/пороги из JSON на всём окне.
"""

from __future__ import annotations

from dataclasses import dataclass

import numpy as np
import pandas as pd

from env import Costs
from features import add_features
from policy_defaults import DEFAULT_POLICY


@dataclass
class BacktestResult:
    equity_curve: np.ndarray
    total_return: float
    max_drawdown: float
    sharpe: float
    win_rate: float
    n_trades: int
    final_cash: float
    final_shares: float


def _sharpe_ratio(returns: np.ndarray, bars_per_year: float) -> float:
    r = returns[np.isfinite(returns)]
    if r.size < 3 or float(np.std(r)) < 1e-12:
        return 0.0
    return float(np.sqrt(bars_per_year) * np.mean(r) / np.std(r))


def _max_drawdown(equity: np.ndarray) -> float:
    if equity.size == 0:
        return 0.0
    peak = np.maximum.accumulate(equity)
    dd = (peak - equity) / np.maximum(peak, 1e-12)
    return float(np.max(dd)) if dd.size else 0.0


def run_rsi_policy_backtest(
    df: pd.DataFrame,
    policy: dict | None = None,
    *,
    costs: Costs | None = None,
    initial_cash: float = 100_000.0,
    max_lot_shares: int = 50,
    bars_per_year: float = 252.0,
) -> BacktestResult:
    policy = {**DEFAULT_POLICY, **(policy or {})}
    costs = costs or Costs()

    fee = costs.fee_bps / 10_000.0
    slip = costs.slippage_bps / 10_000.0

    aggression = float(policy["aggression"])
    buy_pct = float(policy["buy_budget_pct"])
    # sell_budget_pct в Go влияет на объём продажи; в этом бэктесте при RSI>sell закрываем long целиком.
    rsi_buy = float(policy["rsi_buy_below"])
    rsi_sell = float(policy["rsi_sell_above"])
    min_notional = float(policy["min_stock_notional_rub"])

    d = add_features(df).reset_index(drop=True)
    if "close" not in d.columns:
        raise ValueError("DataFrame must contain 'close'")
    if len(d) < 5:
        raise ValueError("Not enough rows for backtest")

    cash = float(initial_cash)
    shares = 0
    entry_cash_out = 0.0  # сколько кэша ушло на вход (включая комиссии), для полного выхода
    trade_pnls: list[float] = []

    equity_list: list[float] = []

    for i in range(len(d)):
        price = float(d.loc[i, "close"])
        rsi = float(d.loc[i, "rsi14"])
        equity = cash + shares * price
        equity_list.append(equity)

        if shares == 0:
            if rsi < rsi_buy:
                budget = cash * buy_pct * aggression
                q = int(budget / price) if price > 0 else 0
                if q < 1:
                    q = 1
                if q > max_lot_shares:
                    q = max_lot_shares
                gross = q * price
                if gross < min_notional:
                    continue
                pay = gross * (1.0 + slip) * (1.0 + fee)
                if pay <= cash and price > 0:
                    cash -= pay
                    shares = q
                    entry_cash_out = pay
        else:
            if rsi > rsi_sell:
                gross = shares * price
                recv = gross * (1.0 - slip) * (1.0 - fee)
                pnl = recv - entry_cash_out
                cash += recv
                trade_pnls.append(pnl)
                shares = 0
                entry_cash_out = 0.0

    eq = np.array(equity_list, dtype=np.float64)
    ret = np.diff(eq) / np.maximum(eq[:-1], 1e-9)

    total_return = float(eq[-1] / max(initial_cash, 1e-9) - 1.0)
    mdd = _max_drawdown(eq)
    sharpe = _sharpe_ratio(ret, bars_per_year)
    wins = sum(1 for x in trade_pnls if x > 0)
    n_trades = len(trade_pnls)
    win_rate = (wins / n_trades) if n_trades else 0.0

    return BacktestResult(
        equity_curve=eq,
        total_return=total_return,
        max_drawdown=mdd,
        sharpe=sharpe,
        win_rate=win_rate,
        n_trades=n_trades,
        final_cash=cash,
        final_shares=float(shares),
    )


def result_to_metrics_dict(r: BacktestResult) -> dict:
    return {
        "total_return": round(r.total_return, 6),
        "max_drawdown": round(r.max_drawdown, 6),
        "sharpe": round(r.sharpe, 4),
        "win_rate": round(r.win_rate, 4),
        "n_trades": r.n_trades,
        "final_equity_estimate": round(float(r.equity_curve[-1]) if r.equity_curve.size else 0.0, 2),
    }
