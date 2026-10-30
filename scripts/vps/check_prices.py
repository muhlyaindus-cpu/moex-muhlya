import json, urllib.request
tickers = ['ALRS', 'NLMK', 'SNGSP', 'SBER', 'TATN']
for t in tickers:
    try:
        url = f'https://iss.moex.com/iss/engines/stock/markets/shares/boards/TQBR/securities/{t}.json?iss.meta=off&marketdata.columns=SECID,LAST,CHANGE'
        with urllib.request.urlopen(url, timeout=5) as r:
            d = json.loads(r.read())
            cols = d['marketdata']['columns']
            row = d['marketdata']['data'][0]
            print(f'{t}: price={row[cols.index("LAST")]} change={row[cols.index("CHANGE")]}')
    except Exception as e:
        print(f'{t}: error {e}')
