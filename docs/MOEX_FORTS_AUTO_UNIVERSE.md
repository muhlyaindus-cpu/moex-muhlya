# MOEX: акции и FORTS в одном проекте

## Миф и реальность
**FORTS** — не «отдельная биржа рядом с MOEX», а **рынок срочных контрактов** внутри Московской биржи. В терминах [MOEX ISS](https://iss.moex.com/iss/reference/) это другие пары **`engine` / `market`**, чем у акций (например, срочные: `futures` / `forts`, акции: `stock` / `shares`).

Для **котировок, лотов, TRADINGSTATUS и расписания** пути в ISS разные — путать нельзя. В коде Мухли это уже разведено (см. `moex-trader-gui/services/moex.go`: лот сначала с рынка акций, затем с `engines/futures/markets/forts/...`).

## Почему в бою часто «как будто только акции»
1. **`AGENT_STOCK_ALLOWLIST_PATH`** (ЛЧИ и т.п.) — режет **только акции**; фьючерсы этим файлом не ограничиваются.
2. **`AGENT_FUTURES_ALLOWLIST_PATH`** — если файл задан и список непустой, в универсуме остаются только перечисленные фьючерсы (плюс логика auto-learn по preflight).
3. **`AGENT_AUTO_INSTRUMENT_MODE`** — `stocks` = только акции; `futures` = только FORTS; `auto` (по умолчанию) = **оба** класса в одном пуле (после фильтров).
4. **`AGENT_OPEN_STOCKS_ONLY=1`** — **только в фазе `open`** и при включённом open-bias: из универсума на этот цикл выкидываются тикеры, которые эвристика считает фьючерсами (`isLikelyFuturesTickerLocal`: цифра в тикере). На остальной сессии фьючерсы снова в пуле (если не режет режим/allowlist).
5. **Счёт Finam** — маржа, квалификация, доступные контракты: технически пул может быть шире, чем то, что брокер реально исполнит.

## Где смотреть в коде
| Что | Файл / место |
|-----|----------------|
| Сбор универсума auto | `getAutoUniverseSnapshot`, `getUniverseSnapshot` в `moex-trader-gui/app.go` |
| Фильтр акций + фьючерсов по allowlist | `applyStockAllowlist` (название историческое: обрабатывает оба класса) |
| Open-only-акции | `autoLoop`: ветка `phase == "open" && openStocksOnly` |
| Каталог FORTS из ISS | `MOEXClient.GetFuturesInstruments` в `services/moex.go` |
| Сессия / TRADINGSTATUS для фьючерсов | `isTradingAllowedForTicker`, кэш `fortsStatus` |

## Пример env (см. также `moex-trader-gui/.env.example`)
- `AGENT_AUTO_INSTRUMENT_MODE=auto` — акции и фьючерсы в одном авто-пуле.
- `AGENT_AUTO_INSTRUMENT_MODE=futures` — только срочный рынок (после allowlist).
- `AGENT_OPEN_STOCKS_ONLY=1` — в фазе open не рассматривать фьючерсы в ранжировании входов.
