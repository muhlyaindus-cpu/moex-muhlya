"""
Офлайн-проверка JSON-политики на исторических данных (train / test split).
При успешных порогах пишет validation.passed + validation_report.json — идея коллег (Миша/Маша).
"""

from __future__ import annotations

import sys
from pathlib import Path

_LAB = Path(__file__).resolve().parent
if str(_LAB) not in sys.path:
    sys.path.insert(0, str(_LAB))

import argparse
import hashlib
import json
from datetime import datetime, timezone

import pandas as pd

from env import Costs
from policy_defaults import load_policy
from rsi_backtest import result_to_metrics_dict, run_rsi_policy_backtest


def _utc_iso() -> str:
    return datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def _file_sha256(path: Path) -> str:
    h = hashlib.sha256()
    h.update(path.read_bytes())
    return h.hexdigest()


def main() -> None:
    ap = argparse.ArgumentParser(description="Backtest + gates for agent policy JSON")
    ap.add_argument("--data", required=True, help="CSV with close column (MOEX candles)")
    ap.add_argument("--policy", required=True, help="policy.json path")
    ap.add_argument(
        "--out-dir",
        default=None,
        help="куда писать validation_report.json и validation.passed (по умолчанию — папка с policy.json)",
    )
    ap.add_argument("--initial-cash", type=float, default=100_000.0)
    ap.add_argument("--train-ratio", type=float, default=0.7, help="in-sample share by rows (chronological)")
    ap.add_argument("--fee-bps", type=float, default=10.0)
    ap.add_argument("--slippage-bps", type=float, default=5.0)
    ap.add_argument("--bars-per-year", type=float, default=252.0)
    ap.add_argument("--max-test-drawdown", type=float, default=0.35, help="reject if test max DD above this")
    ap.add_argument("--min-test-sharpe", type=float, default=-0.5, help="reject if test Sharpe below")
    ap.add_argument("--min-test-win-rate", type=float, default=0.35, help="reject if no trades or WR below")
    ap.add_argument(
        "--strict",
        action="store_true",
        help="строгие пороги (Sharpe>=1, MDD<=0.25, WR>=0.45) — для зрелого контура",
    )
    ap.add_argument("--dry-run", action="store_true", help="только отчёт, без validation.passed")
    args = ap.parse_args()

    if args.strict:
        args.max_test_drawdown = 0.25
        args.min_test_sharpe = 1.0
        args.min_test_win_rate = 0.45

    policy_path = Path(args.policy)
    out_dir = Path(args.out_dir) if args.out_dir else policy_path.parent
    out_dir.mkdir(parents=True, exist_ok=True)

    df = pd.read_csv(args.data)
    if "close" not in df.columns:
        raise SystemExit("CSV must contain 'close' column")

    n = len(df)
    if n < 30:
        raise SystemExit("Need more history rows for meaningful split (>= 30)")

    split = int(n * args.train_ratio)
    split = max(10, min(split, n - 10))

    train_df = df.iloc[:split].copy()
    test_df = df.iloc[split:].copy()

    policy = load_policy(policy_path)
    costs = Costs(fee_bps=args.fee_bps, slippage_bps=args.slippage_bps)

    tr_train = run_rsi_policy_backtest(
        train_df,
        policy,
        costs=costs,
        initial_cash=args.initial_cash,
        bars_per_year=args.bars_per_year,
    )
    tr_test = run_rsi_policy_backtest(
        test_df,
        policy,
        costs=costs,
        initial_cash=args.initial_cash,
        bars_per_year=args.bars_per_year,
    )

    m_train = result_to_metrics_dict(tr_train)
    m_test = result_to_metrics_dict(tr_test)

    gates_ok = (
        m_test["max_drawdown"] <= args.max_test_drawdown
        and m_test["sharpe"] >= args.min_test_sharpe
        and m_test["n_trades"] >= 1
        and m_test["win_rate"] >= args.min_test_win_rate
    )

    report = {
        "generated_utc": _utc_iso(),
        "policy_path": str(policy_path.resolve()),
        "policy_sha256": _file_sha256(policy_path),
        "data_path": str(Path(args.data).resolve()),
        "train_ratio": args.train_ratio,
        "gates": {
            "max_test_drawdown": args.max_test_drawdown,
            "min_test_sharpe": args.min_test_sharpe,
            "min_test_win_rate": args.min_test_win_rate,
            "passed": gates_ok,
        },
        "train_metrics": m_train,
        "test_metrics": m_test,
    }

    report_path = out_dir / "validation_report.json"
    report_path.write_text(json.dumps(report, ensure_ascii=False, indent=2), encoding="utf-8")
    print(f"Wrote {report_path}")

    passed_path = out_dir / "validation.passed"
    if args.dry_run:
        print("Dry-run: not writing validation.passed")
        raise SystemExit(0 if gates_ok else 1)

    if gates_ok:
        passed_payload = {
            "ok": True,
            "utc": _utc_iso(),
            "policy_sha256": report["policy_sha256"],
            "test_metrics": m_test,
        }
        passed_path.write_text(json.dumps(passed_payload, ensure_ascii=False, indent=2), encoding="utf-8")
        print(f"Wrote {passed_path}")
        raise SystemExit(0)

    if passed_path.exists():
        passed_path.unlink()
        print(f"Removed stale {passed_path} (gates failed)")
    print("Gates FAILED — see validation_report.json")
    if m_test["n_trades"] < 1:
        print("Hint: в test-окне не было полных кругов (buy→sell). Попробуй --train-ratio, другие пороги RSI или длиннее историю.")
    raise SystemExit(1)


if __name__ == "__main__":
    main()
