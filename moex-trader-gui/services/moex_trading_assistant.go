package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// MOEXTradingAssistant — торговый ассистент MOEX (правила RSI и политика AGENT_POLICY_PATH).
type MOEXTradingAssistant struct {
	mu               sync.Mutex
	isActive         bool
	lastSignal       *TradeSignal
	baselineCash     float64
	targetCash       float64
	lastObservedCash float64
	aggression       float64
	winStreak        int
	lossStreak       int

	policyPath     string
	policyLoadedAt time.Time
	policy         *AgentPolicy
}

type AgentPolicy struct {
	Version             int     `json:"version"`
	Aggression          float64 `json:"aggression"`
	MinConfidence       float64 `json:"min_confidence"`
	BuyBudgetPct        float64 `json:"buy_budget_pct"`
	SellBudgetPct       float64 `json:"sell_budget_pct"`
	RSIBuyBelow         float64 `json:"rsi_buy_below"`
	RSISellAbove        float64 `json:"rsi_sell_above"`
	MinStockNotionalRub float64 `json:"min_stock_notional_rub"`
	Notes               string  `json:"notes"`
}

type TradeSignal struct {
	Action     string    `json:"action"`
	Quantity   int       `json:"quantity"`
	Price      float64   `json:"price"`
	Confidence float64   `json:"confidence"`
	Reasoning  string    `json:"reasoning"`
	Timestamp  time.Time `json:"timestamp"`
}

// Общие определения для reasoning агента (согласовано с Finam SL/TP и бытовой терминологией).
const (
	agentSLTPGlossary = "SL (Stop Loss) = стоп-лосс — авто-выход при неблагоприятной цене, чтобы ограничить убыток. " +
		"TP (Take Profit) = тейк-профит — авто-фиксация прибыли при достижении целевой цены."
	agentSLTPAfterSignal = " Управление риском: после сделки имеет смысл задать уровни " +
		"SL (стоп-лосс) и/или TP (тейк-профит) под вашу стратегию; в терминале — кнопка SL/TP (связанная заявка Finam)."
	agentSLTPWithPosition = " Открытая позиция: проверьте план по SL/TP — стоп ограничивает просадку, тейк фиксирует цель прибыли."
)

type MarketContext struct {
	Ticker       string
	CurrentPrice float64
	Position     *Position
	Balance      float64
	Indicators   map[string]float64
	// IsFutures — для фьючерсов не применяем пороги/комиссии как для акций MOEX.
	IsFutures bool
	// RoundTripFeePct — доля круга (покупка+продажа), для подсказок в reasoning.
	RoundTripFeePct float64
	// MinStockNotional — минимальная сумма сделки по акциям (₽).
	MinStockNotional float64
}

func NewMOEXTradingAssistant() *MOEXTradingAssistant {
	return &MOEXTradingAssistant{
		isActive:   false,
		aggression: 1.0,
		policyPath: os.Getenv("AGENT_POLICY_PATH"),
	}
}

func (a *MOEXTradingAssistant) SetActive(active bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.isActive = active
}

func (a *MOEXTradingAssistant) IsActive() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.isActive
}

func (a *MOEXTradingAssistant) Analyze(ctx context.Context, market *MarketContext) (*TradeSignal, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.isActive {
		return nil, fmt.Errorf("agent not active")
	}
	a.maybeReloadPolicyLocked()
	a.learnFromBalance(market.Balance)

	signal := &TradeSignal{
		Action:     "hold",
		Quantity:   0,
		Price:      market.CurrentPrice,
		Confidence: 0.5,
		Reasoning:  "Analysis in progress",
		Timestamp:  time.Now(),
	}

	if rsi, ok := market.Indicators["rsi"]; ok {
		requireTrend := envBool("AGENT_REQUIRE_TREND_FILTER", true)
		requireMACD := envBool("AGENT_REQUIRE_MACD_CONFIRM", true)
		buyRSIMax := envFloat("AGENT_RSI_PULLBACK_BUY_MAX", 45.0)
		sellRSIMin := envFloat("AGENT_RSI_PULLBACK_SELL_MIN", 55.0)
		maxVolPct := envFloat("AGENT_VOLATILITY_MAX_PCT", 4.0) / 100.0
		volPct := market.Indicators["volatility_pct"]
		sma20 := market.Indicators["sma20"]
		sma50 := market.Indicators["sma50"]
		macd := market.Indicators["macd"]
		macdSignal := market.Indicators["macd_signal"]
		trendBull := sma20 > 0 && sma50 > 0 && market.CurrentPrice > sma20 && sma20 >= sma50
		trendBear := sma20 > 0 && sma50 > 0 && market.CurrentPrice < sma20 && sma20 <= sma50
		macdBull := macd > macdSignal
		macdBear := macd < macdSignal
		volOK := maxVolPct <= 0 || volPct <= maxVolPct
		buyBelow, sellAbove := a.effectiveRSIThresholds()
		buyPct, sellPct := a.effectiveBudgetPcts()
		buyGate := rsi < buyBelow && rsi <= buyRSIMax
		if requireTrend {
			buyGate = buyGate && trendBull
		}
		if requireMACD {
			buyGate = buyGate && macdBull
		}
		buyGate = buyGate && volOK
		if buyGate {
			signal.Action = "buy"
			signal.Quantity = a.quantityByBalanceAndPrice(market.Balance, market.CurrentPrice, buyPct)
			signal.Confidence = a.adaptiveConfidence(0.74)
			feeNote := ""
			if !market.IsFutures && market.RoundTripFeePct > 0 {
				feeNote = fmt.Sprintf(" Est. round-trip fees ~%.3f%% (Finam MOEX stock tier).", market.RoundTripFeePct*100)
			}
			signal.Reasoning = fmt.Sprintf(
				"Bull setup: trend=%t macd=%t rsi=%.1f<=%.1f vol=%.2f%%. Goal: grow cash from %.2f to %.2f.%s",
				trendBull, macdBull, rsi, buyRSIMax, volPct*100, a.baselineCash, a.targetCash, feeNote,
			)
		} else {
			sellGate := rsi > sellAbove && rsi >= sellRSIMin
			if requireTrend {
				sellGate = sellGate && trendBear
			}
			if requireMACD {
				sellGate = sellGate && macdBear
			}
			sellGate = sellGate && volOK
			if sellGate {
			signal.Action = "sell"
			signal.Quantity = a.quantityByBalanceAndPrice(market.Balance, market.CurrentPrice, sellPct)
			signal.Confidence = a.adaptiveConfidence(0.64)
			signal.Reasoning = fmt.Sprintf(
				"Bear setup: trend=%t macd=%t rsi=%.1f>=%.1f vol=%.2f%%. Goal: protect and grow cash toward %.2f.",
				trendBear, macdBear, rsi, sellRSIMin, volPct*100, a.targetCash,
			)
			} else {
				signal.Reasoning = fmt.Sprintf(
					"No clean setup: trendBull=%t trendBear=%t macdBull=%t macdBear=%t rsi=%.1f vol=%.2f%%.",
					trendBull, trendBear, macdBull, macdBear, rsi, volPct*100,
				)
			}
		}
	}

	if !market.IsFutures && (signal.Action == "buy" || signal.Action == "sell") {
		minN := a.effectiveMinStockNotional(market.MinStockNotional)
		if minN > 0 && signal.Quantity > 0 && market.CurrentPrice > 0 {
			nom := float64(signal.Quantity) * market.CurrentPrice
			if nom < minN {
				signal.Action = "hold"
				signal.Quantity = 0
				signal.Confidence = 0.45
				signal.Reasoning = fmt.Sprintf(
					"Trade size %.0f ₽ below min notional %.0f ₽ (tariff N3: avoid fee drag on tiny tickets).",
					nom, minN,
				)
			}
		}
	}

	signal.Reasoning = enrichReasoningWithSLTP(signal.Reasoning, signal.Action, market)

	a.lastSignal = signal
	return signal, nil
}

// enrichReasoningWithSLTP добавляет в ответ агента базовые определения SL/TP, чтобы reasoning
// оставался согласованным с кнопкой SL/TP и gRPC PlaceSLTPOrder у брокера.
func enrichReasoningWithSLTP(reason, action string, market *MarketContext) string {
	if market == nil {
		return reason
	}
	r := strings.TrimSpace(reason)
	switch action {
	case "buy", "sell":
		if r == "" {
			r = agentSLTPGlossary + agentSLTPAfterSignal
		} else {
			r = r + " " + agentSLTPGlossary + agentSLTPAfterSignal
		}
	case "hold":
		if market.Position != nil && market.Position.Quantity != 0 {
			if r == "" {
				r = agentSLTPGlossary + " " + agentSLTPWithPosition
			} else {
				r = r + " " + agentSLTPGlossary + " " + agentSLTPWithPosition
			}
		}
	}
	return r
}

func (a *MOEXTradingAssistant) effectiveRSIThresholds() (buyBelow, sellAbove float64) {
	buyBelow, sellAbove = 28.0, 72.0
	if a.policy != nil {
		if a.policy.RSIBuyBelow > 0 {
			buyBelow = a.policy.RSIBuyBelow
		}
		if a.policy.RSISellAbove > 0 {
			sellAbove = a.policy.RSISellAbove
		}
	}
	return buyBelow, sellAbove
}

func (a *MOEXTradingAssistant) effectiveBudgetPcts() (buyPct, sellPct float64) {
	buyPct, sellPct = 0.06, 0.05
	if a.policy != nil {
		if a.policy.BuyBudgetPct > 0 {
			buyPct = a.policy.BuyBudgetPct
		}
		if a.policy.SellBudgetPct > 0 {
			sellPct = a.policy.SellBudgetPct
		}
	}
	return buyPct, sellPct
}

func (a *MOEXTradingAssistant) effectiveMinStockNotional(fromContext float64) float64 {
	if a.policy != nil && a.policy.MinStockNotionalRub > 0 {
		return a.policy.MinStockNotionalRub
	}
	if fromContext > 0 {
		return fromContext
	}
	return AgentMinStockNotionalRub()
}

func (a *MOEXTradingAssistant) GetLastSignal() *TradeSignal {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.lastSignal
}

func (a *MOEXTradingAssistant) learnFromBalance(current float64) {
	if current <= 0 {
		return
	}
	if a.baselineCash <= 0 {
		a.baselineCash = current
		// Начальная цель: +5% к стартовому кэшу (динамически от текущего баланса).
		a.targetCash = current * 1.05
		a.lastObservedCash = current
		return
	}

	if a.lastObservedCash > 0 {
		delta := current - a.lastObservedCash
		switch {
		case delta > 0:
			a.winStreak++
			a.lossStreak = 0
			a.aggression += 0.05
		case delta < 0:
			a.lossStreak++
			a.winStreak = 0
			a.aggression -= 0.08
		}
	}

	if a.aggression < 0.4 {
		a.aggression = 0.4
	}
	if a.aggression > 1.8 {
		a.aggression = 1.8
	}

	// Если цель достигнута — ставим следующую.
	if current >= a.targetCash {
		a.targetCash = current * 1.03
	}
	a.lastObservedCash = current
}

func (a *MOEXTradingAssistant) adaptiveQuantity(base int) int {
	q := int(float64(base) * a.aggression)
	if q < 1 {
		return 1
	}
	if q > 50 {
		return 50
	}
	return q
}

func (a *MOEXTradingAssistant) quantityByBalanceAndPrice(balance, price, budgetPct float64) int {
	// fallback на старую схему, если не хватает данных
	if balance <= 0 || price <= 0 {
		return a.adaptiveQuantity(1)
	}
	if budgetPct <= 0 {
		budgetPct = 0.08
	}

	budget := balance * budgetPct * a.aggression
	q := int(budget / price)
	if q < 1 {
		q = 1
	}
	if q > 50 {
		q = 50
	}
	return q
}

func (a *MOEXTradingAssistant) adaptiveConfidence(base float64) float64 {
	c := base + (a.aggression-1.0)*0.1
	if a.policy != nil && a.policy.MinConfidence > 0 {
		if c < a.policy.MinConfidence {
			c = a.policy.MinConfidence
		}
	}
	if c < 0.2 {
		return 0.2
	}
	if c > 0.95 {
		return 0.95
	}
	return c
}

func (a *MOEXTradingAssistant) ApplyJournalFeedback(winRate float64, realizedPnL float64) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if winRate > 0.58 && realizedPnL > 0 {
		a.aggression += 0.05
	} else if winRate < 0.45 || realizedPnL < 0 {
		a.aggression -= 0.07
	}

	if a.aggression < 0.4 {
		a.aggression = 0.4
	}
	if a.aggression > 1.8 {
		a.aggression = 1.8
	}
}

func (a *MOEXTradingAssistant) maybeReloadPolicyLocked() {
	if a.policyPath == "" {
		return
	}
	// Не чаще раза в 10 секунд, чтобы не трогать диск постоянно.
	if !a.policyLoadedAt.IsZero() && time.Since(a.policyLoadedAt) < 10*time.Second {
		return
	}
	a.policyLoadedAt = time.Now()

	b, err := os.ReadFile(a.policyPath)
	if err != nil {
		a.policy = nil
		return
	}

	var p AgentPolicy
	if err := json.Unmarshal(b, &p); err != nil {
		a.policy = nil
		return
	}

	// Мягкое применение: только разумные диапазоны.
	if p.Aggression >= 0.4 && p.Aggression <= 1.8 {
		a.aggression = p.Aggression
	}
	a.policy = &p
}

func envBool(key string, def bool) bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if raw == "" {
		return def
	}
	switch raw {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return def
	}
}

func envFloat(key string, def float64) float64 {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return def
	}
	return v
}
