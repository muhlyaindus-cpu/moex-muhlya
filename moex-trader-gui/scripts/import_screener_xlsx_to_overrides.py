# -*- coding: utf-8 -*-
"""Одноразовый/повторяемый импорт: Excel Finam screener -> JSON для FINAM_SYMBOL_OVERRIDES_PATH."""
import argparse
import json
from pathlib import Path

import pandas as pd


def main() -> None:
    p = argparse.ArgumentParser()
    p.add_argument("xlsx", type=Path, help="Путь к .xlsx (колонки Тикер, Название)")
    p.add_argument("-o", "--out", type=Path, default=None, help="Выходной .json")
    args = p.parse_args()
    out = args.out or args.xlsx.with_name(args.xlsx.stem + "_overrides.json")

    df = pd.read_excel(args.xlsx)
    tcol, ncol = "Тикер", "Название"
    if tcol not in df.columns or ncol not in df.columns:
        raise SystemExit(f"Нужны колонки {tcol!r} и {ncol!r}, есть: {list(df.columns)}")

    out_map: dict[str, str] = {}
    conflicts: list[tuple[str, str, str]] = []
    for _, row in df.iterrows():
        k = str(row[tcol]).strip() if pd.notna(row[tcol]) else ""
        v = str(row[ncol]).strip() if pd.notna(row[ncol]) else ""
        if not k or k.lower() == "nan":
            continue
        if not v or v.lower() == "nan":
            continue
        if k.startswith("_"):
            continue
        if not v.endswith("@FUT"):
            v = v + "@FUT"
        if k in out_map and out_map[k] != v:
            conflicts.append((k, out_map[k], v))
        out_map[k] = v

    ordered = {
        "_comment": f"Сгенерировано из {args.xlsx.name}. Ключ — Тикер (MOEX/Finam), значение — Название@FUT. Перепроверьте склеенные/дубли перед боем.",
        **dict(sorted(out_map.items(), key=lambda x: x[0].upper())),
    }
    out.parent.mkdir(parents=True, exist_ok=True)
    with open(out, "w", encoding="utf-8") as f:
        json.dump(ordered, f, ensure_ascii=False, indent=2)
    print(f"keys: {len(out_map)}, conflicts: {len(conflicts)} -> {out}")
    if conflicts[:10]:
        print("sample conflicts:", conflicts[:10])


if __name__ == "__main__":
    main()
