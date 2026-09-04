## Trading Lab (RL / backtest) — локальная “лаборатория” агента

Цель: обучать/тестировать торгового агента **локально** на исторических данных MOEX, а результат обучения экспортировать в простой артефакт (JSON), который можно использовать в `moex-trader-gui` без облаков.

### Принципы
- **Прод-контур остаётся на Go** (наш терминал/риск/брокер).
- **Обучение — оффлайн** (Python), на исторических данных.
- **Артефакт обучения — локальный файл**, который легко версионировать и откатывать.

### Что внутри
- `moex_iss_download.py`: скачивание свечей (candles) через MOEX ISS.
- `features.py`: базовые фичи (returns, SMA, RSI).
- `env.py`: минимальное Gym-окружение (buy/sell/hold) со штрафом за риск.
- `train_ppo.py`: обучение PPO (Stable-Baselines3) и сохранение модели.
- `export_policy.py`: экспорт “политики” в `policy.json` (поля совместимы с Go `AgentPolicy`: агрессия, RSI, бюджеты и т.д.).
- `policy_defaults.py`: дефолты JSON в одной линии с `moex-trader-gui`.
- `rsi_backtest.py`: упрощённый бэктест правил RSI + доли от кэша (офлайн).
- `validate_policy.py`: train/test split, метрики, пороги; при успехе рядом с политикой появляются `validation_report.json` и `validation.passed` (идея коллег).
- `RL_PLAN_D3QN_PER_MOEX.md`: roadmap адаптации D3QN+PER под наш стек (MOEX + Finam).

### Быстрый старт (Windows)
Из корня `Trading`:

```powershell
python -m venv .venv
.\.venv\Scripts\Activate.ps1
pip install -r .\trading-lab\requirements.txt
```

Скачать данные (пример: SBER, дневные свечи):

```powershell
python .\trading-lab\moex_iss_download.py --symbol SBER --engine stock --market shares --interval 24 --out .\trading-lab\data\SBER_1d.csv
```

Обучить PPO:

```powershell
python .\trading-lab\train_ppo.py --data .\trading-lab\data\SBER_1d.csv --model-out .\trading-lab\artifacts\ppo_sber.zip
```

Экспортировать политику для Go-агента:

```powershell
python .\trading-lab\export_policy.py --model .\trading-lab\artifacts\ppo_sber.zip --policy-out .\trading-lab\artifacts\policy.json
```

Опционально задать пороги/бюджеты (иначе берутся дефолты как в Go):

```powershell
python .\trading-lab\export_policy.py --model .\trading-lab\artifacts\ppo_sber.zip --policy-out .\trading-lab\artifacts\policy.json --rsi-buy-below 30 --rsi-sell-above 68
```

Проверить политику на истории (out-of-sample + пороги; артефакты — в папке с `policy.json`):

```powershell
python .\trading-lab\validate_policy.py --data .\trading-lab\data\SBER_1d.csv --policy .\trading-lab\artifacts\policy.json
```

Строгие пороги (Sharpe, просадка, win rate), как обсуждали с коллегами:

```powershell
python .\trading-lab\validate_policy.py --data .\trading-lab\data\SBER_1d.csv --policy .\trading-lab\artifacts\policy.json --strict
```

Скрипты можно запускать из **корня `Trading`** (пути к `trading-lab/...` как выше).

### Как подключить к нашему GUI-агенту
Мы читаем JSON-политику (локально) из пути:
- переменная окружения `AGENT_POLICY_PATH` (например `trading-lab/artifacts/policy.json`)

Политика влияет на поведение агента (агрессия/ограничения/порог уверенности), но **не отключает риск-менеджер**.

### Важно про качество
- Обязательно используем **walk-forward** (train/test по временным окнам).
- Учитываем комиссии/спред/проскальзывание (иначе будет “прибыль” только на бумаге).
- Делаем сначала **paper** и только потом очень ограниченный live.

### Операционный статус лаборатории
- Для понятного режима использования см. `LAB_FILE_MATRIX.md` (что используем всегда, что по задаче).
- Базовый ежедневный режим лаборатории: `moex_iss_download.py` -> `validate_policy.py` -> артефакты (`validation_report.json`, `validation.passed`).
- `train_ppo.py` и `env.py` включаем только под отдельную гипотезу обучения.

