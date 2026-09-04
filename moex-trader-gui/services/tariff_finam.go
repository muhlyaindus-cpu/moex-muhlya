package services

import (
	"os"
	"strconv"
	"strings"
)

func envFloat64(key string, def float64) float64 {
	s := strings.TrimSpace(os.Getenv(key))
	if s == "" {
		return def
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return def
	}
	return v
}

// FinamN3MOEXStockRoundTripPct — оценка полного круга (покупка+продажа) для акций на МосБирже:
// 2 × (комиссия брокера за ногу + клиринг за ногу). Без НДС-уточнений.
// Дефолт брокерской ноги — под линейку N3 «Инвестор» (публично «от 0,025%» на MOEX; см. broker.finam.ru).
// При другой ставке по договору задайте FINAM_TARIFF_MOEX_STOCK_BROKER_LEG_PCT.
// Дефолт: брокер 0,025% + клиринг 0,03% с каждой стороны → 0,11% на круг.
func FinamN3MOEXStockRoundTripPct() float64 {
	br := envFloat64("FINAM_TARIFF_MOEX_STOCK_BROKER_LEG_PCT", 0.00025)
	cl := envFloat64("FINAM_TARIFF_MOEX_STOCK_CLEARING_LEG_PCT", 0.0003)
	return 2.0 * (br + cl)
}

// FinamN3MOEXStockCashMultiplier — множитель к цене позиции при оценке «хватит ли кэша»:
// 1 + round_trip + запас на проскальзывание.
func FinamN3MOEXStockCashMultiplier() float64 {
	slip := envFloat64("FINAM_TARIFF_SLIPPAGE_RESERVE_PCT", 0.0005)
	return 1.0 + FinamN3MOEXStockRoundTripPct() + slip
}

// AgentMinStockNotionalRub — минимальная сумма сделки по акциям (чтобы не штрафовать себя оборотом).
func AgentMinStockNotionalRub() float64 {
	return envFloat64("AGENT_MIN_STOCK_NOTIONAL_RUB", 5000)
}
