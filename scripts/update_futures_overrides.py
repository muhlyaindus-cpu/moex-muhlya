#!/usr/bin/env python3
"""
Автоматическое обновление маппинга тикеров фьючерсов MOEX ISS → Finam API.
Запускается периодически (например, раз в неделю) для поддержания актуальности.

Использование:
    python scripts/update_futures_overrides.py

Результат:
    Обновляет файл finam_symbol_overrides_screener.json свежими тикерами из MOEX ISS.
"""

import json
import os
import sys
import urllib.request
from datetime import datetime

MOEX_FUTURES_URL = "https://iss.moex.com/iss/engines/futures/markets/forts/securities.json?iss.meta=off&securities.columns=SECID,SHORTNAME"
# Записываем напрямую в moex-trader-gui (откуда GUI читает overrides)
OVERRIDES_FILE = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "moex-trader-gui", "finam_symbol_overrides_screener.json")


def fetch_moex_futures():
    """Получить список фьючерсов с MOEX ISS"""
    try:
        with urllib.request.urlopen(MOEX_FUTURES_URL, timeout=30) as response:
            data = json.loads(response.read().decode("utf-8"))
        
        securities = data.get("securities", {})
        columns = securities.get("columns", [])
        rows = securities.get("data", [])
        
        secid_idx = columns.index("SECID") if "SECID" in columns else -1
        shortname_idx = columns.index("SHORTNAME") if "SHORTNAME" in columns else -1
        
        if secid_idx < 0 or shortname_idx < 0:
            print("❌ Не удалось найти колонки SECID/SHORTNAME в ответе MOEX ISS")
            return {}
        
        futures_map = {}
        for row in rows:
            secid = row[secid_idx].strip()
            shortname = row[shortname_idx].strip()
            
            if not secid or not shortname:
                continue
            
            # Извлекаем базовый тикер из SHORTNAME
            # Например: "MTSI-12.26" → "MTSI", "Si-6.25" → "Si"
            base_ticker = extract_base_ticker(shortname)
            
            if base_ticker:
                # Формируем тикер для Finam API
                # Формат: "BASETICKER-M.YY@FUT" (например, "MTSI-12.26@FUT")
                finam_ticker = f"{shortname}@FUT"
                futures_map[secid] = finam_ticker
                
                # Также добавляем маппинг для базового тикера
                if base_ticker not in futures_map:
                    futures_map[base_ticker] = finam_ticker
        
        return futures_map
    
    except Exception as e:
        print(f"❌ Ошибка получения данных из MOEX ISS: {e}")
        return {}


def extract_base_ticker(shortname):
    """Извлечь базовый тикер из SHORTNAME (например, 'MTSI-12.26' → 'MTSI')"""
    for i, ch in enumerate(shortname):
        if ch == '-' or ch == '.':
            return shortname[:i].strip()
    return shortname.strip()


def update_overrides_file(new_futures):
    """Обновить файл overrides новыми тикерами фьючерсов"""
    # Читаем существующий файл
    existing = {}
    if os.path.exists(OVERRIDES_FILE):
        try:
            with open(OVERRIDES_FILE, "r", encoding="utf-8") as f:
                existing = json.load(f)
        except Exception as e:
            print(f"⚠️  Не удалось прочитать существующий файл: {e}")
    
    # Добавляем заголовок
    header = {
        "_comment": f"Автообновлено {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}. Ключ — MOEX SECID или базовый тикер, значение — Finam API символ (@FUT).",
        "_source": "MOEX ISS FORTS + Finam screener"
    }
    
    # Объединяем старые и новые записи
    # Новые тикеры перезаписывают старые (актуальнее)
    updated = {**existing, **new_futures}
    
    # Добавляем заголовок в начало
    result = {**header, **updated}
    
    # Записываем файл
    try:
        with open(OVERRIDES_FILE, "w", encoding="utf-8") as f:
            json.dump(result, f, ensure_ascii=False, indent=2)
        print(f"✅ Обновлено {len(new_futures)} тикеров фьючерсов в {OVERRIDES_FILE}")
        return True
    except Exception as e:
        print(f"❌ Ошибка записи файла: {e}")
        return False


def main():
    print(f"🔄 Обновление маппинга тикеров фьючерсов...")
    print(f"📅 {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    print()
    
    # Получаем свежие тикеры из MOEX ISS
    print("📡 Получение списка фьючерсов из MOEX ISS...")
    new_futures = fetch_moex_futures()
    
    if not new_futures:
        print("❌ Не удалось получить тикеры из MOEX ISS")
        sys.exit(1)
    
    print(f"✅ Получено {len(new_futures)} тикеров")
    print()
    
    # Показываем примеры
    print("📋 Примеры маппинга:")
    examples = list(new_futures.items())[:10]
    for secid, finam_ticker in examples:
        print(f"   {secid} → {finam_ticker}")
    if len(new_futures) > 10:
        print(f"   ... и ещё {len(new_futures) - 10}")
    print()
    
    # Обновляем файл
    print("💾 Обновление файла overrides...")
    if update_overrides_file(new_futures):
        print()
        print("✅ Готово! Перезапустите GUI для применения изменений.")
    else:
        print()
        print("❌ Ошибка обновления файла")
        sys.exit(1)


if __name__ == "__main__":
    main()
