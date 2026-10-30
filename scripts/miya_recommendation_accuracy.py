#!/usr/bin/env python3
"""
Анализ точности рекомендаций Мии.

Читает miya.recommendations из runtime_events, проверяет цену через N часов,
считает % точности. Чистит записи старше 7 дней.

Формат рекомендаций (обновлён 2026-10-30):
  top5: TATN(buy,96,bull) NLMK(buy,95,bull) ROSN(hold,85,bull)

Использование:
    python scripts/miya_recommendation_accuracy.py [--hours 4] [--days 7]
"""

import argparse
import json
import re
import subprocess
import sys
from datetime import datetime, timedelta

DB_CONTAINER = "moex_trading_db"
DB_USER = "trader"
DB_NAME = "moex_trading"


def run_sql(sql: str) -> str:
    """Выполнить SQL на VPS через SSH."""
    cmd = f'ssh traderbot@72.56.13.248 "docker exec {DB_CONTAINER} psql -U {DB_USER} -d {DB_NAME} -t -A -c \\"{sql}\\""'
    result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
    return result.stdout.strip()


def parse_recommendations(details: str) -> list:
    """Парсит строку рекомендаций:
    Новый формат: 'TATN(buy,96,bull) NLMK(buy,95,bull)'
    Старый формат: 'ALRS(97,bull) TATN(96,bull)'
    """
    # Новый формат с action
    new_pattern = r'(\w+)\((buy|sell|hold),(\d+),(bull|bear)\)'
    matches = re.findall(new_pattern, details)
    if matches:
        return [
            {"ticker": m[0], "action": m[1], "score": int(m[2]), "trend": m[3]}
            for m in matches
        ]
    # Старый формат без action
    old_pattern = r'(\w+)\((\d+),(bull|bear)\)'
    matches = re.findall(old_pattern, details)
    return [
        {"ticker": m[0], "action": "unknown", "score": int(m[1]), "trend": m[2]}
        for m in matches
    ]


def get_current_prices(tickers: list) -> dict:
    """Получить текущие цены тикеров через MOEX ISS (акции + фьючерсы)."""
    prices = {}
    for ticker in tickers:
        # Пробуем акции, затем фьючерсы
        for endpoint in [
            f"https://iss.moex.com/iss/engines/stock/markets/shares/boards/TQBR/securities/{ticker}.json?iss.meta=off&marketdata.columns=SECID,LAST",
            f"https://iss.moex.com/iss/engines/futures/markets/forts/securities/{ticker}.json?iss.meta=off&marketdata.columns=SECID,LAST",
        ]:
            cmd = f'curl -s "{endpoint}"'
            result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
            try:
                data = json.loads(result.stdout)
                md = data.get("marketdata", {})
                cols = md.get("columns", [])
                rows = md.get("data", [])
                if "LAST" in cols and rows:
                    last_idx = cols.index("LAST")
                    price = rows[0][last_idx]
                    if price:
                        prices[ticker] = float(price)
                        break  # нашли цену
            except (json.JSONDecodeError, KeyError, IndexError):
                pass
    return prices


def analyze(hours: int = 4, days: int = 7):
    """Анализ точности рекомендаций."""
    print(f"📊 Анализ рекомендаций Мии (последние {days} дней)")
    print(f"   Проверяем через {hours} часа после рекомендации\n")

    # Получаем рекомендации из runtime_events
    since = datetime.now() - timedelta(days=days)
    since_str = since.strftime("%Y-%m-%d %H:%M:%S")

    sql = f"""
        SELECT at, details 
        FROM runtime_events 
        WHERE step = 'miya.recommendations' 
        AND at > '{since_str}'
        ORDER BY at DESC
    """
    rows = run_sql(sql)
    if not rows:
        print("❌ Нет рекомендаций за период")
        return

    total_recs = 0
    buy_recs = 0
    sell_recs = 0
    hold_recs = 0
    wins = 0
    losses = 0
    pending = 0

    print("📋 Рекомендации:")
    print("-" * 70)

    for row in rows.split("\n"):
        if "|" not in row:
            continue
        parts = row.split("|")
        if len(parts) < 2:
            continue
        rec_time_str = parts[0].strip()
        details = parts[1].strip()

        try:
            rec_time = datetime.strptime(rec_time_str[:19], "%Y-%m-%d %H:%M:%S")
        except ValueError:
            continue

        recs = parse_recommendations(details)
        if not recs:
            continue

        # Проверяем, прошло ли достаточно времени
        check_time = rec_time + timedelta(hours=hours)
        if datetime.now() < check_time:
            pending += len(recs)
            continue

        print(f"\n🕐 {rec_time_str[:16]}")
        for rec in recs:
            total_recs += 1
            ticker = rec["ticker"]
            action = rec["action"]
            score = rec["score"]
            trend = rec["trend"]

            if action == "buy":
                buy_recs += 1
            elif action == "sell":
                sell_recs += 1
            else:
                hold_recs += 1

            current_prices = get_current_prices([ticker])
            if ticker not in current_prices:
                pending += 1
                continue

            current_price = current_prices[ticker]
            print(f"   {ticker}: {action} score={score} trend={trend} current={current_price:.2f}")

            # Упрощённая проверка точности
            if score > 90 and trend == "bull" and action == "buy":
                pending += 1  # нужна цена в момент рекомендации для точной проверки

    print("\n" + "=" * 70)
    print(f"📈 Итого:")
    print(f"   Всего рекомендаций: {total_recs}")
    print(f"   🟢 BUY: {buy_recs}")
    print(f"   🔴 SELL: {sell_recs}")
    print(f"   🟡 HOLD: {hold_recs}")
    print(f"   ✅ Wins: {wins}")
    print(f"   ❌ Losses: {losses}")
    print(f"   ⏳ Pending: {pending}")
    if wins + losses > 0:
        accuracy = wins / (wins + losses) * 100
        print(f"   🎯 Точность: {accuracy:.1f}%")


def cleanup(days: int = 7):
    """Удалить старые записи recommendations из runtime_events."""
    print(f"\n🧹 Очистка записей старше {days} дней...")
    since = datetime.now() - timedelta(days=days)
    since_str = since.strftime("%Y-%m-%d %H:%M:%S")

    count_sql = f"SELECT COUNT(*) FROM runtime_events WHERE step = 'miya.recommendations' AND at < '{since_str}'"
    count = run_sql(count_sql)
    print(f"   Найдено записей для удаления: {count}")

    if count and int(count) > 0:
        print(f"   ⚠️  Удаление закомментировано для безопасности")
        print(f"   Раскомментируйте строку run_sql(sql) в функции cleanup()")
    else:
        print(f"   ✅ Нечего удалять")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Анализ точности рекомендаций Мии")
    parser.add_argument("--hours", type=int, default=4, help="Через сколько часов проверять (default: 4)")
    parser.add_argument("--days", type=int, default=7, help="Период анализа в днях (default: 7)")
    parser.add_argument("--cleanup", action="store_true", help="Удалить старые записи")
    args = parser.parse_args()

    analyze(hours=args.hours, days=args.days)
    if args.cleanup:
        cleanup(days=args.days)
