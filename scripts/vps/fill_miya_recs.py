#!/usr/bin/env python3
"""
Заполняет miya_recommendations из runtime_events и проверяет текущие цены.
Запускать на VPS: python3 /tmp/fill_miya_recs.py
"""

import json
import re
import subprocess
import urllib.request

def run_sql(sql):
    cmd = f'docker exec moex_trading_db psql -U trader -d moex_trading -t -A -c "{sql}"'
    result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
    return result.stdout.strip()

def get_price(ticker):
    url = f"https://iss.moex.com/iss/engines/stock/markets/shares/boards/TQBR/securities/{ticker}.json?iss.meta=off&marketdata.columns=SECID,LAST"
    try:
        with urllib.request.urlopen(url, timeout=5) as resp:
            data = json.loads(resp.read())
            cols = data["marketdata"]["columns"]
            rows = data["marketdata"]["data"]
            if "LAST" in cols and rows and rows[0][cols.index("LAST")]:
                return float(rows[0][cols.index("LAST")])
    except:
        pass
    return None

def parse_recs(details):
    pattern = r'(\w+)\((\d+),(bull|bear)\)'
    return [{"ticker": m[0], "score": int(m[1]), "trend": m[2]} for m in re.findall(pattern, details)]

# Получаем рекомендации
rows = run_sql("SELECT at, details FROM runtime_events WHERE step = 'miya.recommendations' ORDER BY at DESC LIMIT 50")
if not rows:
    print("Нет рекомендаций")
    exit()

inserted = 0
for row in rows.split("\n"):
    if "|" not in row:
        continue
    parts = row.split("|", 1)
    rec_time = parts[0].strip()
    details = parts[1].strip()
    recs = parse_recs(details)
    
    for rec in recs:
        # Проверяем, есть ли уже
        exists = run_sql(f"SELECT COUNT(*) FROM miya_recommendations WHERE recommended_at = '{rec_time}' AND ticker = '{rec['ticker']}'")
        if exists and int(exists) > 0:
            continue
        
        price = get_price(rec["ticker"])
        price_str = str(price) if price else "NULL"
        
        sql = f"""INSERT INTO miya_recommendations (recommended_at, ticker, score, trend, price_at_rec) 
                  VALUES ('{rec_time}', '{rec['ticker']}', {rec['score']}, '{rec['trend']}', {price_str})"""
        run_sql(sql)
        inserted += 1

print(f"Вставлено {inserted} записей")

# Показываем статистику
print("\n📊 Статистика:")
total = run_sql("SELECT COUNT(*) FROM miya_recommendations")
pending = run_sql("SELECT COUNT(*) FROM miya_recommendations WHERE result = 'pending'")
print(f"  Всего: {total}")
print(f"  Pending: {pending}")

# Показываем последние рекомендации
print("\n📋 Последние рекомендации:")
rows = run_sql("SELECT recommended_at, ticker, score, trend, price_at_rec FROM miya_recommendations ORDER BY recommended_at DESC LIMIT 10")
for row in rows.split("\n"):
    if "|" in row:
        parts = row.split("|")
        print(f"  {parts[0][:16]} | {parts[1]:6s} | score={parts[2]:4s} | {parts[3]:5s} | price={parts[4]}")
