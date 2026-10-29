package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"moex-trader-gui/services"

	"trading-core/engine"

	tradev1 "github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1"
	"github.com/joho/godotenv"
)

var dotenvLoadOnce sync.Once

// brokerPortfolioSnap* — редкая запись JSON-снимка портфеля для Мии (без дополнительных запросов к Finam).
var (
	brokerPortfolioSnapMu   sync.Mutex
	brokerPortfolioSnapLast time.Time
)

func loadDotEnvOnce() {
	dotenvLoadOnce.Do(func() {
		// Загружаем .env (секреты: FINAM_SECRET_REAL, PGHOST и т.д.)
		if err := godotenv.Load(); err != nil {
			fmt.Println("⚠️  .env file not found, using environment variables")
		}
		// Загружаем agent.env (настройки агента: фильтры, TP/SL, фьючерсы и т.д.)
		if err := godotenv.Load("agent.env"); err != nil {
			fmt.Println("⚠️  agent.env file not found, using environment variables")
		}
	})
}

type fortsStatusCacheEntry struct {
	status string
	at     time.Time
}

type stockStatusCacheEntry struct {
	status string
	at     time.Time
}

type App struct {
	ctx      context.Context
	moex     *services.MOEXClient
	db       *services.PostgresDB
	agent    *services.MOEXTradingAssistant
	broker   *services.FinamBroker
	realtime map[string]chan float64
	risk     *engine.RiskManager

	mode      engine.Mode
	autoMu    sync.Mutex
	autoStop  context.CancelFunc
	autoAlive bool
	// true => приоритет автоторговли для текущего пользователя.
	userAutoPreferred bool
	autoCfgMu         sync.RWMutex
	// auto | fixed
	autoInstrumentMode string
	// Используется, когда autoInstrumentMode == "fixed".
	autoFixedTicker string
	// 10..100 (шаг 10): доля кэша, доступная Мухле в auto-режиме.
	autoBudgetPct int
	// TP/SL в процентах для auto-режима.
	autoTPPct float64
	autoSLPct float64

	universeMu       sync.Mutex
	universeCachedAt time.Time
	universe         []services.Security

	// Автообновление тикеров фьючерсов из MOEX ISS (каждую пятницу)
	futuresOverridesUpdateInterval time.Duration
	futuresOverridesLastUpdate     time.Time
	futuresOverridesMu             sync.Mutex

	// Daily Top-N кэш (итеративный выбор инструментов)
	dailyTopNCache     []services.Security
	dailyTopNCacheAt   time.Time
	dailyTopNCacheTTL  time.Duration
	dailyTopNMu        sync.RWMutex

	statsMu       sync.Mutex
	statsCachedAt time.Time
	statsCacheTTL time.Duration
	statsCached   map[string]interface{}

	orderThrottleMu  sync.Mutex
	orderThrottle    map[string]time.Time
	minOrderInterval time.Duration
	tradingLocation  *time.Location
	tradingStatusMu  sync.Mutex
	// Кэш "идут ли торги на площадке" для безопасного учета выходных/праздников.
	exchangeTradingOpenCached       bool
	exchangeTradingOpenCachedAt     time.Time
	exchangeTradingOpenCachedReason string
	stockStatusMu                   sync.Mutex
	stockStatusCache                map[string]stockStatusCacheEntry
	fortsStatusMu                   sync.Mutex
	fortsStatusCache                map[string]fortsStatusCacheEntry
	badTickerMu                     sync.Mutex
	// временная блокировка тикеров после систематических ошибок валидации брокера.
	badTickersUntil map[string]time.Time
	badTickerReason map[string]string
	// мягкая ротация в auto-режиме: временно пропускаем недавно использованные тикеры.
	recentAutoTickerUntil map[string]time.Time
	// статический denylist тикеров из env для оперативной блокировки.
	staticTickerDenylist map[string]struct{}
	// whitelist акций (например ЛЧИ); на фьючерсы/опционы не влияет.
	stockAllowlist map[string]struct{}
	// whitelist фьючерсов (опционально); если пустой — фьючерсы не ограничиваются.
	futuresAllowlist map[string]struct{}
	// Путь к файлу futures allowlist (для авто-обновления списка).
	futuresAllowlistPath string
	// true => автоматически обновлять futures allowlist по результатам preflight.
	futuresAllowlistAutoLearn bool
	futuresAllowlistMu        sync.Mutex

	runtimeMu         sync.Mutex
	runtimeEvents     []RuntimeEvent
	runtimeMaxSize    int
	runtimeDBMaxSize  int
	badTickerCooldown time.Duration
	autoStatusMu      sync.Mutex
	autoStatus        AutoDecisionStatus
	blockedTickersMu  sync.Mutex
	blockedTickers    []BlockedTicker
	blockedTickersAt  time.Time
	profitLockMu      sync.Mutex
	// Трейлинг-пик прибыли по позиции для авто-фиксации.
	profitLockPeakPct map[string]float64
	trailingStopMu    sync.Mutex
	// Цена линии трейлинг-стопа (акции); ключ — верхний регистр тикера. Отсутствие ключа = трейлинг ещё не активирован.
	trailingStopPrice map[string]float64
	// Порог активации и ширина трейлинга (в %).
	autoProfitLockTriggerPct float64
	autoProfitLockTrailPct   float64
	// Ротация капитала: переключаемся в более сильный инструмент.
	autoRotateEnabled    bool
	autoRotateMinDelta   float64
	autoRotateCooldown   time.Duration
	autoRotateLastAt     time.Time
	autoRotateMu         sync.Mutex
	positionAgeMu        sync.Mutex
	positionOpenedAt     map[string]time.Time
	lossStreakMu         sync.Mutex
	autoLossStreak       int
	autoLossPauseUntil   time.Time
	cashPauseMu          sync.Mutex
	buyCashPauseUntil    time.Time
	riskDailyPauseMu     sync.Mutex
	riskDailyPauseUntil  time.Time
	riskDailyPauseReason string
	shortPauseMu         sync.Mutex
	shortOpenPauseUntil  time.Time
	shortOpenTickerMu    sync.Mutex
	shortOpenTickerUntil map[string]time.Time
	pendingBuyGateMu     sync.Mutex
	pendingBuyGateSig    string
	pendingBuyGateSince  time.Time
	noOrderMu            sync.Mutex
	autoNoOrderStreak    int
	sessionPhaseMu       sync.Mutex
	lastSessionPhase     string
	exitProfileMu        sync.Mutex
	lastExitProfile      string
	exitLearnMu          sync.Mutex
	exitLearning         map[string]tickerExitLearning
	exitLearningPath     string
	miyaHintsPath        string
	miyaHintsMu          sync.Mutex
	miyaHints            map[string]float64
	miyaHintsUntil       time.Time
	miyaOnlineSignalPath string
	miyaOnlineSignalMu   sync.Mutex
	miyaOnlineSignal     miyaOnlineSignal
	miyaOnlineSignalTill time.Time
	// Inline MIYA: встроенный режим Мии (запрос MOEX ISS напрямую, без внешнего скрипта)
	miyaInlineEnabled bool
	miyaInlineTickers []string
}

type TradeJournalEntry struct {
	ID         int      `json:"id"`
	Ticker     string   `json:"ticker"`
	Side       string   `json:"side"`
	Quantity   int      `json:"quantity"`
	Price      float64  `json:"price"`
	Status     string   `json:"status"`
	ExecutedAt string   `json:"executed_at"`
	Source     string   `json:"source,omitempty"`      // finam | local
	ExternalID string   `json:"external_id,omitempty"` // trade_id Finam при source=finam
	RealizedPL *float64 `json:"realized_pl,omitempty"` // P/L по сделке закрытия позиции
}

type RuntimeEvent struct {
	At      string `json:"at"`
	Level   string `json:"level"`
	Step    string `json:"step"`
	Details string `json:"details"`
}

type SafetyConfig struct {
	BadTickerCooldownMin     int     `json:"bad_ticker_cooldown_min"`
	AutoBudgetPercent        int     `json:"auto_budget_percent"`
	AutoTPPct                float64 `json:"auto_tp_pct"`
	AutoSLPct                float64 `json:"auto_sl_pct"`
	AutoProfitLockTriggerPct float64 `json:"auto_profit_lock_trigger_pct"`
	AutoProfitLockTrailPct   float64 `json:"auto_profit_lock_trail_pct"`
}

type AutoDecisionStatus struct {
	At                     string `json:"at"`
	UniverseSize           int    `json:"universe_size"`
	PassedFilters          int    `json:"passed_filters"`
	BlockedByCooldown      int    `json:"blocked_by_cooldown"`
	BlockedByQualification int    `json:"blocked_by_qualification"`
	BlockedByRotation      int    `json:"blocked_by_rotation"`
	BlockedByCash          int    `json:"blocked_by_cash"`
	RejectedByLiquidity    int    `json:"rejected_by_liquidity"`
	RejectedByPrice        int    `json:"rejected_by_price"`
	BestBuyTicker          string `json:"best_buy_ticker"`
	BestBuyConf            string `json:"best_buy_conf"`
	BestSellTicker         string `json:"best_sell_ticker"`
	BestSellConf           string `json:"best_sell_conf"`
	ActionTaken            string `json:"action_taken"`
	Note                   string `json:"note"`
}

type BlockedTicker struct {
	Ticker string `json:"ticker"`
	Reason string `json:"reason"`
	Type   string `json:"type"` // "stock" или "futures"
}

type tickerExitLearning struct {
	Ticker     string  `json:"ticker"`
	Samples    int     `json:"samples"`
	Wins       int     `json:"wins"`
	Losses     int     `json:"losses"`
	SumWinPct  float64 `json:"sum_win_pct"`
	SumLossPct float64 `json:"sum_loss_pct"`
	LastReason string  `json:"last_reason"`
	LastPhase  string  `json:"last_phase"`
	LastPnlPct float64 `json:"last_pnl_pct"`
	UpdatedAt  string  `json:"updated_at"`
	TpMulLearn float64 `json:"tp_mul_learn"`
	TpMulHint  float64 `json:"tp_mul_hint"`
	TpMulFinal float64 `json:"tp_mul_final"`
}

type exitLearningStore struct {
	UpdatedAt string                        `json:"updated_at"`
	Tickers   map[string]tickerExitLearning `json:"tickers"`
}

// MiyaRecommendation — рекомендация Мии для конкретного тикера.
type MiyaRecommendation struct {
	Ticker string  `json:"ticker"`
	Score  float64 `json:"score"`  // 0-100, выше = лучше
	RSI    float64 `json:"rsi"`    // RSI(14)
	Trend  string  `json:"trend"`  // "bull" или "bear"
	Reason string  `json:"reason"` // краткое объяснение
}

type miyaOnlineSignal struct {
	State            string
	DataStatus       string
	DefaultMul       float64
	UpRatioPct       float64
	TickersLongCount int
	// Дневной срез из miya-online-watch (MOEX ISS candles interval=24), см. DAILY_* в miya_online_signal.env
	DailyState       string
	DailyDataStatus  string
	DailyDefaultMul  float64
	DailyUpRatioPct  float64
	// Ранг ликвидности (Мия): тикеры от более ликвидного к менее; см. MIYA_LIQUIDITY_* в miya_online_signal.env
	LiquidityOrder  []string
	LiquidityStatus string
	// Список «подготовки» Мии (тот же, что ONLINE_MIA_API_TICKERS); MIYA_WATCH_TICKERS в miya_online_signal.env
	WatchTickers []string
	// Рекомендации Мии — топ-N тикеров для покупки (отсортированы по score)
	Recommendations []MiyaRecommendation
	// Расширенная аналитика
	TrendBullCount int     // кол-во тикеров в бычьем тренде
	RSIAverage     float64 // средний RSI по всем тикерам
}

func normalizeMiyaLiquidityStatus(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	switch s {
	case "ok", "partial", "no_data", "disabled":
		return s
	default:
		return "no_data"
	}
}

func miyaLiquidityRankBonus(ticker string, rank map[string]int, n int, status string, weight float64) float64 {
	if weight <= 0 || n <= 0 || len(rank) == 0 {
		return 0
	}
	st := normalizeMiyaLiquidityStatus(status)
	if st == "no_data" || st == "disabled" {
		return 0
	}
	k := canonicalTickerKey(ticker)
	if k == "" {
		return 0
	}
	r, ok := rank[k]
	if !ok || r < 1 {
		return 0
	}
	return weight * float64(n-r+1) / float64(n)
}

// miyaRecommendationBonus возвращает бонус к rankScore для тикеров из рекомендаций Мии.
// Score 0-100 нормализуется в бонус 0-weight.
func miyaRecommendationBonus(ticker string, recScores map[string]float64, weight float64) float64 {
	if weight <= 0 || len(recScores) == 0 {
		return 0
	}
	k := canonicalTickerKey(ticker)
	if k == "" {
		return 0
	}
	score, ok := recScores[k]
	if !ok || score <= 0 {
		return 0
	}
	// Нормализуем score (0-100) в бонус (0-weight)
	return weight * score / 100.0
}

func miyaOnlineGateDeltas(state string, defaultMul float64, buyMaxDelta float64, shortMaxDelta float64) (float64, float64) {
	state = normalizeMiyaOnlineState(state)
	strength := clampFloat(absFloat(defaultMul-1.0)*20.0, 0.25, 1.0)
	switch state {
	case "bullish":
		return -buyMaxDelta * strength, shortMaxDelta * strength
	case "bearish":
		return buyMaxDelta * strength, -shortMaxDelta * strength
	default:
		return 0, 0
	}
}

type TickerRejectStats struct {
	Ticker        string  `json:"ticker"`
	Total24h      int     `json:"total_24h"`
	Rejected24h   int     `json:"rejected_24h"`
	RejectRate24h float64 `json:"reject_rate_24h"`
	Total7d       int     `json:"total_7d"`
	Rejected7d    int     `json:"rejected_7d"`
	RejectRate7d  float64 `json:"reject_rate_7d"`
}

func normalizeTradeTime(ts time.Time, loc *time.Location) time.Time {
	// Нереалистичные/битые метки времени из внешнего API показываем как "сейчас",
	// чтобы в UI не появлялись даты вроде 1970/0001.
	if ts.IsZero() || ts.Before(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)) {
		return time.Now().UTC()
	}
	// Локальные значения из БД хранятся без таймзоны, поэтому трактуем их как время торговой зоны.
	if ts.Location() == time.UTC && loc != nil {
		return time.Date(ts.Year(), ts.Month(), ts.Day(), ts.Hour(), ts.Minute(), ts.Second(), ts.Nanosecond(), loc)
	}
	return ts
}

func rejectedStatusFromError(err error) string {
	if err == nil {
		return "rejected"
	}
	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		return "rejected"
	}
	if len(msg) > 220 {
		msg = msg[:220] + "..."
	}
	return "rejected: " + msg
}

func canonicalTickerKey(ticker string) string {
	t := strings.ToUpper(strings.TrimSpace(ticker))
	if t == "" {
		return ""
	}
	if i := strings.Index(t, "@"); i > 0 {
		t = t[:i]
	}
	// UI/журнал могут хранить тикер в формате TICKER_TQBR / TICKER_SPBFUT.
	if i := strings.Index(t, "_"); i > 0 {
		t = t[:i]
	}
	return strings.TrimSpace(t)
}

// tickerLearningKey нормализует тикер для карт обучения (совпадает с pnlScores / canonical).
func tickerLearningKey(ticker string) string {
	k := canonicalTickerKey(ticker)
	if k != "" {
		return k
	}
	return strings.ToUpper(strings.TrimSpace(ticker))
}

func (a *App) validateOrderInput(ticker string, quantity int, price float64, isLimit bool, enforceDenylist bool) error {
	ticker = strings.ToUpper(strings.TrimSpace(ticker))
	if ticker == "" {
		return fmt.Errorf("empty ticker")
	}
	if enforceDenylist && a.isTickerStaticallyDenied(ticker) {
		return fmt.Errorf("ticker %s is blocked by AGENT_TICKER_DENYLIST", ticker)
	}
	if until, ok := a.getBadTickerBlockUntil(ticker); ok {
		return fmt.Errorf("ticker %s temporarily blocked after broker validation error until %s", ticker, until.In(a.tradingLocation).Format("15:04:05"))
	}
	if quantity <= 0 {
		return fmt.Errorf("invalid quantity: %d", quantity)
	}
	if isLimit && price <= 0 {
		return fmt.Errorf("invalid limit price: %.4f", price)
	}

	lot := 1
	if a.moex != nil {
		lot = a.moex.GetLotSize(ticker)
	}
	if lot <= 0 {
		lot = 1
	}
	if quantity < lot {
		return fmt.Errorf("quantity %d is below lot size %d for %s", quantity, lot, ticker)
	}
	if quantity%lot != 0 {
		return fmt.Errorf("quantity %d is not multiple of lot size %d for %s", quantity, lot, ticker)
	}
	return nil
}

func (a *App) getBadTickerBlockUntil(ticker string) (time.Time, bool) {
	t := strings.ToUpper(strings.TrimSpace(ticker))
	if t == "" {
		return time.Time{}, false
	}
	a.badTickerMu.Lock()
	defer a.badTickerMu.Unlock()
	until, ok := a.badTickersUntil[t]
	if !ok {
		return time.Time{}, false
	}
	if time.Now().After(until) {
		delete(a.badTickersUntil, t)
		delete(a.badTickerReason, t)
		return time.Time{}, false
	}
	return until, true
}

func (a *App) getBadTickerBlockInfo(ticker string) (time.Time, string, bool) {
	t := strings.ToUpper(strings.TrimSpace(ticker))
	if t == "" {
		return time.Time{}, "", false
	}
	a.badTickerMu.Lock()
	defer a.badTickerMu.Unlock()
	until, ok := a.badTickersUntil[t]
	if !ok {
		return time.Time{}, "", false
	}
	if time.Now().After(until) {
		delete(a.badTickersUntil, t)
		delete(a.badTickerReason, t)
		return time.Time{}, "", false
	}
	return until, a.badTickerReason[t], true
}

func (a *App) registerBrokerValidationFailure(ticker string, err error) {
	if err == nil {
		return
	}
	msg := strings.ToLower(err.Error())
	isQty := strings.Contains(msg, "invalid quantity value") || strings.Contains(msg, "неверное значение параметра 'количество'")
	isShortage := strings.Contains(msg, "[157]") || strings.Contains(msg, "equities shortage") || strings.Contains(msg, "не хватает собственных бумаг")
	isMoneyShortage := strings.Contains(msg, "[156]") || strings.Contains(msg, "money shortage") || strings.Contains(msg, "не хватает денежных средств")
	isQualification := strings.Contains(msg, "[257]") || strings.Contains(msg, "требующие тестирования") || strings.Contains(msg, "confirm your qualification level")
	// Защищаемся от повторных попыток по невалидным тикерам (например, FUT 404).
	isUnknownSecurity := strings.Contains(msg, "security id doesn't exist") || strings.Contains(msg, "security id does not exist")
	if !isQty && !isQualification && !isShortage && !isMoneyShortage && !isUnknownSecurity {
		return
	}
	t := strings.ToUpper(strings.TrimSpace(ticker))
	if t == "" {
		return
	}
	a.badTickerMu.Lock()
	if a.badTickersUntil == nil {
		a.badTickersUntil = make(map[string]time.Time)
	}
	if a.badTickerReason == nil {
		a.badTickerReason = make(map[string]string)
	}
	cooldown := a.badTickerCooldown
	if cooldown <= 0 {
		cooldown = 30 * time.Minute
	}
	if isQualification {
		cooldown = maxDuration(cooldown, 6*time.Hour)
	} else if isShortage {
		cooldown = maxDuration(cooldown, 5*time.Minute)
	} else if isMoneyShortage {
		// По нехватке денег охлаждаем коротко: это не "плохой тикер", а состояние кэша.
		cooldown = 90 * time.Second
	} else if isUnknownSecurity {
		cooldown = maxDuration(cooldown, 6*time.Hour)
	}
	a.badTickersUntil[t] = time.Now().Add(cooldown)
	if isQualification {
		a.badTickerReason[t] = "qualification"
	} else if isMoneyShortage {
		a.badTickerReason[t] = "cash_shortage"
	} else if isUnknownSecurity {
		a.badTickerReason[t] = "unknown_security"
	} else {
		a.badTickerReason[t] = "quantity"
	}
	a.badTickerMu.Unlock()
	if isQualification {
		fmt.Printf("⚠️ Ticker %s temporarily blocked for %s due to broker qualification restriction [257]\n", t, cooldown.Round(time.Minute))
		a.addRuntimeEvent("warn", "order.blocked_qualification", fmt.Sprintf("%s blocked for %s after qualification restriction [257]", t, cooldown.Round(time.Minute)))
		return
	}
	if isShortage {
		fmt.Printf("⚠️ Ticker %s temporarily blocked for %s due to equities shortage [157]\n", t, cooldown.Round(time.Minute))
		a.addRuntimeEvent("warn", "order.blocked_shortage", fmt.Sprintf("%s blocked for %s after equities shortage [157]", t, cooldown.Round(time.Minute)))
		return
	}
	if isMoneyShortage {
		fmt.Printf("⚠️ Ticker %s temporarily blocked for %s due to cash shortage [156]\n", t, cooldown.Round(time.Second))
		a.addRuntimeEvent("warn", "order.blocked_cash_shortage", fmt.Sprintf("%s blocked for %s after money shortage [156]", t, cooldown.Round(time.Second)))
		return
	}
	if isUnknownSecurity {
		fmt.Printf("⚠️ Ticker %s temporarily blocked for %s due to unknown security at broker\n", t, cooldown.Round(time.Minute))
		a.addRuntimeEvent("warn", "order.blocked_unknown_security", fmt.Sprintf("%s blocked for %s after broker unknown security error", t, cooldown.Round(time.Minute)))
		return
	}
	fmt.Printf("⚠️ Ticker %s temporarily blocked for %s due to broker quantity validation error\n", t, cooldown.Round(time.Minute))
	a.addRuntimeEvent("warn", "order.blocked_ticker", fmt.Sprintf("%s blocked for %s after broker validation error", t, cooldown.Round(time.Minute)))
}

func (a *App) GetSafetyConfig() SafetyConfig {
	min := int(a.badTickerCooldown / time.Minute)
	if min < 1 {
		min = 1
	}
	pct := a.getAutoBudgetPercent()
	tp, sl := a.getAutoTPSLPercent()
	plTrigger, plTrail := a.getAutoProfitLockPercent()
	return SafetyConfig{
		BadTickerCooldownMin:     min,
		AutoBudgetPercent:        pct,
		AutoTPPct:                tp,
		AutoSLPct:                sl,
		AutoProfitLockTriggerPct: plTrigger,
		AutoProfitLockTrailPct:   plTrail,
	}
}

func (a *App) SetBadTickerCooldownMinutes(minutes int) bool {
	if minutes < 1 {
		return false
	}
	a.badTickerCooldown = time.Duration(minutes) * time.Minute
	a.addRuntimeEvent("info", "safety.cooldown_changed", fmt.Sprintf("bad_ticker_cooldown_min=%d", minutes))
	return true
}

func (a *App) getAutoBudgetPercent() int {
	a.autoCfgMu.RLock()
	defer a.autoCfgMu.RUnlock()
	pct := a.autoBudgetPct
	if pct < 10 {
		pct = 10
	}
	if pct > 100 {
		pct = 100
	}
	return pct
}

func (a *App) SetAutoBudgetPercent(percent int) bool {
	if percent < 10 || percent > 100 || percent%10 != 0 {
		return false
	}
	a.autoCfgMu.Lock()
	a.autoBudgetPct = percent
	a.autoCfgMu.Unlock()
	a.addRuntimeEvent("info", "safety.auto_budget_changed", fmt.Sprintf("auto_budget_percent=%d", percent))
	return true
}

func (a *App) getAutoTPSLPercent() (float64, float64) {
	a.autoCfgMu.RLock()
	defer a.autoCfgMu.RUnlock()
	tp := a.autoTPPct
	sl := a.autoSLPct
	if tp <= 0 {
		tp = 1.2
	}
	if sl <= 0 {
		sl = 0.8
	}
	return tp, sl
}

func (a *App) SetAutoTPSLPercent(tpPct, slPct float64) bool {
	if tpPct <= 0 || slPct <= 0 {
		return false
	}
	if tpPct > 30 || slPct > 30 {
		return false
	}
	a.autoCfgMu.Lock()
	a.autoTPPct = tpPct
	a.autoSLPct = slPct
	a.autoCfgMu.Unlock()
	a.addRuntimeEvent("info", "safety.tpsl_changed", fmt.Sprintf("tp=%.2f sl=%.2f", tpPct, slPct))
	return true
}

func (a *App) getAutoProfitLockPercent() (float64, float64) {
	a.autoCfgMu.RLock()
	defer a.autoCfgMu.RUnlock()
	trigger := a.autoProfitLockTriggerPct
	trail := a.autoProfitLockTrailPct
	if trigger <= 0 {
		trigger = 10.0
	}
	if trail <= 0 {
		trail = 3.0
	}
	return trigger, trail
}

func (a *App) SetAutoProfitLockPercent(triggerPct, trailPct float64) bool {
	if triggerPct <= 0 || trailPct <= 0 {
		return false
	}
	if triggerPct > 100 || trailPct > 50 {
		return false
	}
	a.autoCfgMu.Lock()
	a.autoProfitLockTriggerPct = triggerPct
	a.autoProfitLockTrailPct = trailPct
	a.autoCfgMu.Unlock()
	a.addRuntimeEvent("info", "safety.profit_lock_changed", fmt.Sprintf("trigger=%.2f trail=%.2f", triggerPct, trailPct))
	return true
}

func (a *App) GetAutoDecisionStatus() AutoDecisionStatus {
	a.autoStatusMu.Lock()
	defer a.autoStatusMu.Unlock()
	return a.autoStatus
}

// GetBlockedTickers возвращает список заблокированных бумаг с причинами блокировки.
// Кэшируется на 30 секунд, чтобы не долбить MOEX ISS API.
func (a *App) GetBlockedTickers() []BlockedTicker {
	a.blockedTickersMu.Lock()
	defer a.blockedTickersMu.Unlock()
	
	// Возвращаем кэш, если он свежий (меньше 30 секунд)
	if time.Since(a.blockedTickersAt) < 30*time.Second && len(a.blockedTickers) > 0 {
		return a.blockedTickers
	}
	
	// Получаем свежий список заблокированных
	universe := a.getUniverseSnapshot(500)
	blocked := make([]BlockedTicker, 0)
	
	for _, s := range universe {
		allowed, reason := a.isTradingAllowedForTicker(s.Ticker)
		if !allowed {
			tickerType := "stock"
			if isLikelyFuturesTickerLocal(s.Ticker) {
				tickerType = "futures"
			}
			blocked = append(blocked, BlockedTicker{
				Ticker: s.Ticker,
				Reason: reason,
				Type:   tickerType,
			})
		}
	}
	
	a.blockedTickers = blocked
	a.blockedTickersAt = time.Now()
	return blocked
}

// ResetAutoContextSoft мягко сбрасывает временные блокировки/паузы автоторговли,
// не удаляя историю сделок и обучение по журналу.
func (a *App) ResetAutoContextSoft() bool {
	now := time.Now()
	a.badTickerMu.Lock()
	a.recentAutoTickerUntil = make(map[string]time.Time)
	for t, until := range a.badTickersUntil {
		if until.Before(now) {
			delete(a.badTickersUntil, t)
			delete(a.badTickerReason, t)
		}
	}
	a.badTickerMu.Unlock()

	a.autoRotateMu.Lock()
	a.autoRotateLastAt = time.Time{}
	a.autoRotateMu.Unlock()

	a.lossStreakMu.Lock()
	a.autoLossStreak = 0
	a.autoLossPauseUntil = time.Time{}
	a.lossStreakMu.Unlock()

	a.profitLockMu.Lock()
	a.profitLockPeakPct = make(map[string]float64)
	a.profitLockMu.Unlock()

	a.trailingStopMu.Lock()
	a.trailingStopPrice = make(map[string]float64)
	a.trailingStopMu.Unlock()

	a.addRuntimeEvent("info", "auto.context_soft_reset", "cleared recent/cooldown/pause/rotate state")
	return true
}

func (a *App) addRuntimeEvent(level, step, details string) {
	level = strings.ToLower(strings.TrimSpace(level))
	if level == "" {
		level = "info"
	}
	ev := RuntimeEvent{
		At:      a.nowInTradingTZ().Format(time.RFC3339),
		Level:   level,
		Step:    strings.TrimSpace(step),
		Details: strings.TrimSpace(details),
	}
	a.runtimeMu.Lock()
	maxSize := a.runtimeMaxSize
	if maxSize <= 0 {
		maxSize = 400
	}
	a.runtimeEvents = append(a.runtimeEvents, ev)
	if len(a.runtimeEvents) > maxSize {
		a.runtimeEvents = a.runtimeEvents[len(a.runtimeEvents)-maxSize:]
	}
	a.runtimeMu.Unlock()
	if a.db != nil {
		evAt, err := time.Parse(time.RFC3339, ev.At)
		if err != nil {
			evAt = time.Now().UTC()
		}
		maxRows := a.runtimeDBMaxSize
		if maxRows <= 0 {
			maxRows = 500
		}
		_ = a.db.LogRuntimeEvent(evAt, ev.Level, ev.Step, ev.Details, maxRows)
	}
}

// GetRuntimeEvents возвращает последние runtime-события (новые сверху).
func (a *App) GetRuntimeEvents(limit int) []RuntimeEvent {
	if limit <= 0 {
		limit = 120
	}
	a.runtimeMu.Lock()
	events := make([]RuntimeEvent, len(a.runtimeEvents))
	copy(events, a.runtimeEvents)
	a.runtimeMu.Unlock()
	if len(events) == 0 {
		return []RuntimeEvent{}
	}
	out := make([]RuntimeEvent, 0, limit)
	for i := len(events) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, events[i])
	}
	return out
}

// GetRuntimeEventsPersistent возвращает последние runtime-события из БД (новые сверху).
func (a *App) GetRuntimeEventsPersistent(limit int) []RuntimeEvent {
	if limit <= 0 {
		limit = 120
	}
	if a.db == nil {
		return []RuntimeEvent{}
	}
	rows, err := a.db.GetRecentRuntimeEvents(limit)
	if err != nil || len(rows) == 0 {
		return []RuntimeEvent{}
	}
	out := make([]RuntimeEvent, 0, len(rows))
	loc := a.tradingLocation
	if loc == nil {
		loc = time.Local
	}
	for _, r := range rows {
		out = append(out, RuntimeEvent{
			At:      normalizeTradeTime(r.At, loc).In(loc).Format(time.RFC3339),
			Level:   strings.TrimSpace(r.Level),
			Step:    strings.TrimSpace(r.Step),
			Details: strings.TrimSpace(r.Details),
		})
	}
	return out
}

// affordableQuantity рассчитывает максимально допустимый объем покупки с запасом
// под комиссии/проскальзывание/резерв.
func (a *App) affordableQuantity(balance, price float64, desired int, isFutures bool, ticker string) int {
	if desired <= 0 || balance <= 0 || price <= 0 {
		return 0
	}

	// Для акций считаем почти по полной цене, для фьючерсов — по ГО (гарантийному обеспечению)
	// ГО по умолчанию = 15% от стоимости контракта (настраивается через AGENT_FUTURES_MARGIN_RATIO)
	effectivePrice := price
	if isFutures {
		marginRatio := envFloat("AGENT_FUTURES_MARGIN_RATIO", 0.15)
		if marginRatio <= 0 || marginRatio > 1 {
			marginRatio = 0.15
		}
		effectivePrice = price * marginRatio
		
		// Кросс-маржа: если есть акции того же эмитента, можно использовать их как обеспечение
		// Это увеличивает покупательную способность для фьючерсов
		crossMarginEnabled := envBool("AGENT_FUTURES_CROSS_MARGIN_ENABLED", true)
		if crossMarginEnabled {
			// Проверяем, есть ли акции того же эмитента в портфеле
			stockTicker := a.extractStockTickerFromFutures(ticker)
			if stockTicker != "" {
				stockPosition := a.getPositionQuantity(stockTicker)
				if stockPosition > 0 {
					// Есть акции — можно увеличить количество фьючерсов
					// Упрощённо: считаем что акции покрывают 50% ГО
					effectivePrice = price * marginRatio * 0.5
					a.addRuntimeEvent("info", "futures.cross_margin", fmt.Sprintf("%s: акции %s qty=%d используются как обеспечение", ticker, stockTicker, stockPosition))
				}
			}
		}
	}

	feeAndSlip := 1.003 // 0.3% запас на издержки (фьючерсы / запасной режим)
	if !isFutures {
		// Акции MOEX: комиссия брокера + клиринг (круг) + проскальзывание — см. services/tariff_finam.go
		feeAndSlip = services.FinamN3MOEXStockCashMultiplier()
	}
	// Доля кэша, разрешённая для реального использования в расчёте qty.
	// 1.00 = использовать весь доступный кэш (с учётом feeAndSlip и лотовых ограничений).
	safetyReserve := envFloat("AGENT_CASH_USAGE_RESERVE", 1.00)
	if safetyReserve <= 0 {
		safetyReserve = 1.00
	}
	if safetyReserve > 1.00 {
		safetyReserve = 1.00
	}
	maxByCash := int((balance * safetyReserve) / (effectivePrice * feeAndSlip))
	if maxByCash < 1 {
		return 0
	}
	if desired > maxByCash {
		return maxByCash
	}
	return desired
}

func (a *App) floorToLot(ticker string, quantity int) int {
	if quantity <= 0 {
		return 0
	}
	lot := 1
	if a.moex != nil {
		lot = a.moex.GetLotSize(ticker)
	}
	if lot <= 1 {
		return quantity
	}
	return floorQuantityByLot(quantity, lot)
}

// clampSellQuantityToAvailableLong ограничивает объём продажи доступным long-остатком.
// Это снижает ошибки брокера вида "Equities shortage".
func (a *App) clampSellQuantityToAvailableLong(ticker string, requested int) (int, error) {
	t := strings.ToUpper(strings.TrimSpace(ticker))
	if t == "" || requested <= 0 {
		return 0, fmt.Errorf("invalid sell request ticker=%q qty=%d", ticker, requested)
	}
	positions := a.GetPortfolio()
	if len(positions) == 0 {
		return 0, fmt.Errorf("no positions available for sell %s", t)
	}
	canonical := canonicalTickerKey(t)
	available := 0
	for _, p := range positions {
		pt := strings.ToUpper(strings.TrimSpace(p.Ticker))
		if pt == "" || p.Quantity <= 0 {
			continue
		}
		if pt == t || canonicalTickerKey(pt) == canonical {
			available += p.Quantity
		}
	}
	if available <= 0 {
		return 0, fmt.Errorf("no long quantity available for sell %s", t)
	}
	if requested > available {
		requested = available
	}
	clamped := a.floorToLot(ticker, requested)
	if clamped <= 0 {
		return 0, fmt.Errorf("sell qty after lot clamp is zero for %s (requested=%d available=%d)", t, requested, available)
	}
	return clamped, nil
}

func floorQuantityByLot(quantity, lot int) int {
	if quantity <= 0 || lot <= 0 {
		return 0
	}
	if lot == 1 {
		return quantity
	}
	return (quantity / lot) * lot
}

func (a *App) isRecentAutoTickerBlocked(ticker string) bool {
	t := strings.ToUpper(strings.TrimSpace(ticker))
	if t == "" {
		return false
	}
	a.badTickerMu.Lock()
	defer a.badTickerMu.Unlock()
	if a.recentAutoTickerUntil == nil {
		a.recentAutoTickerUntil = make(map[string]time.Time)
	}
	until, ok := a.recentAutoTickerUntil[t]
	if !ok {
		return false
	}
	if time.Now().After(until) {
		delete(a.recentAutoTickerUntil, t)
		return false
	}
	return true
}

func (a *App) rememberRecentAutoTicker(ticker string, d time.Duration) {
	t := strings.ToUpper(strings.TrimSpace(ticker))
	if t == "" || d <= 0 {
		return
	}
	a.badTickerMu.Lock()
	if a.recentAutoTickerUntil == nil {
		a.recentAutoTickerUntil = make(map[string]time.Time)
	}
	a.recentAutoTickerUntil[t] = time.Now().Add(d)
	a.badTickerMu.Unlock()
}

func parseTickerDenylist(raw string) map[string]struct{} {
	out := make(map[string]struct{})
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		switch r {
		case ',', ';', '\n', '\r', '\t', ' ':
			return true
		default:
			return false
		}
	})
	for _, p := range parts {
		t := canonicalTickerKey(p)
		if t == "" {
			continue
		}
		out[t] = struct{}{}
	}
	return out
}

func loadTickerAllowlist(path string) map[string]struct{} {
	path = strings.TrimSpace(path)
	if path == "" {
		return map[string]struct{}{}
	}
	f, err := os.Open(path)
	if err != nil {
		return map[string]struct{}{}
	}
	defer f.Close()

	out := make(map[string]struct{})
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		t := canonicalTickerKey(line)
		if t == "" {
			continue
		}
		out[t] = struct{}{}
	}
	return out
}

func (a *App) isStockAllowed(ticker string) bool {
	if a == nil || len(a.stockAllowlist) == 0 {
		return true
	}
	key := canonicalTickerKey(ticker)
	if key == "" {
		return false
	}
	_, ok := a.stockAllowlist[key]
	return ok
}

func (a *App) isFuturesAllowed(ticker string) bool {
	if a == nil || len(a.futuresAllowlist) == 0 {
		return true
	}
	key := canonicalTickerKey(ticker)
	if key == "" {
		return false
	}
	_, ok := a.futuresAllowlist[key]
	return ok
}

func (a *App) shouldAutoLearnFuturesAllowlist() bool {
	return a != nil && a.futuresAllowlistAutoLearn && strings.TrimSpace(a.futuresAllowlistPath) != ""
}

func (a *App) persistFuturesAllowlistLocked() bool {
	path := strings.TrimSpace(a.futuresAllowlistPath)
	if path == "" {
		return false
	}
	keys := make([]string, 0, len(a.futuresAllowlist))
	for k := range a.futuresAllowlist {
		if k == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	body := ""
	if len(keys) > 0 {
		body = strings.Join(keys, "\n") + "\n"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(body), 0o644); err != nil {
		return false
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return false
	}
	return true
}

func (a *App) noteFuturesValidationResult(ticker string, ok bool, why string) {
	if !a.shouldAutoLearnFuturesAllowlist() {
		return
	}
	key := canonicalTickerKey(ticker)
	if key == "" {
		return
	}
	lowerWhy := strings.ToLower(strings.TrimSpace(why))
	isNotFound := strings.Contains(lowerWhy, "notfound") ||
		strings.Contains(lowerWhy, "doesn't exist") ||
		strings.Contains(lowerWhy, "does not exist") ||
		strings.Contains(lowerWhy, "нет инструмента") ||
		strings.Contains(lowerWhy, "security id doesn't exist")

	a.futuresAllowlistMu.Lock()
	changed := false
	if ok {
		if _, exists := a.futuresAllowlist[key]; !exists {
			a.futuresAllowlist[key] = struct{}{}
			changed = true
		}
	} else if isNotFound {
		if _, exists := a.futuresAllowlist[key]; exists {
			delete(a.futuresAllowlist, key)
			changed = true
		}
	}
	if changed {
		_ = a.persistFuturesAllowlistLocked()
	}
	size := len(a.futuresAllowlist)
	a.futuresAllowlistMu.Unlock()

	if changed {
		state := "removed"
		if ok {
			state = "added"
		}
		a.addRuntimeEvent("info", "auto.futures_allowlist_updated", fmt.Sprintf("%s %s size=%d", state, key, size))
	}
}

func (a *App) applyStockAllowlist(universe []services.Security) []services.Security {
	if len(universe) == 0 {
		return universe
	}
	out := make([]services.Security, 0, len(universe))
	for _, s := range universe {
		if isLikelyFuturesTickerLocal(s.Ticker) {
			if !a.isFuturesAllowed(s.Ticker) {
				continue
			}
			out = append(out, s)
			continue
		}
		if !a.isStockAllowed(s.Ticker) {
			continue
		}
		out = append(out, s)
	}
	return out
}

func (a *App) isTickerStaticallyDenied(ticker string) bool {
	if a == nil || len(a.staticTickerDenylist) == 0 {
		return false
	}
	key := canonicalTickerKey(ticker)
	if key == "" {
		return false
	}
	_, blocked := a.staticTickerDenylist[key]
	return blocked
}

func parseAutoInstrumentEnv(modeRaw, tickerRaw string) (string, string) {
	mode := strings.ToLower(strings.TrimSpace(modeRaw))
	ticker := strings.ToUpper(strings.TrimSpace(tickerRaw))
	switch mode {
	case "", "auto", "all":
		return "auto", ""
	case "stocks", "futures", "options":
		return mode, ""
	case "fixed":
		if ticker == "" {
			return "auto", ""
		}
		return "fixed", ticker
	default:
		return "auto", ""
	}
}

type abTestProfile struct {
	Mode            string
	ConfDelta       float64
	BuyGateOffset   float64
	SellGateOffset  float64
	ShortGateOffset float64
	RankScoreDelta  float64
}

// sessionPhaseMoscow классифицирует фазу дня для аналитики (границы — Europe/Moscow, не заменяет isTradingAllowedForTicker).
// open: первые openMin минут основной дневной сессии акций; close: последние closeMin минут до 18:40;
// mid: остальное время, когда идут торги акций (включая вечернюю сессию); offhours: нет торгов или только фьючерсы без акций.
func sessionPhaseMoscow(stockOpen, futuresOpen bool, openMin, closeMin int) string {
	if openMin < 5 {
		openMin = 5
	}
	if closeMin < 5 {
		closeMin = 5
	}
	if !stockOpen && !futuresOpen {
		return "offhours"
	}
	if !stockOpen && futuresOpen {
		return "mid"
	}
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		loc = time.UTC
	}
	now := time.Now().In(loc)
	mins := now.Hour()*60 + now.Minute()
	dow := now.Weekday()

	isWeekend := dow == time.Saturday || dow == time.Sunday
	if isWeekend {
		// Выходной режим MOEX (упрощённо): 10:00–19:00
		if mins < 10*60 || mins >= 19*60 {
			return "offhours"
		}
		if mins < 10*60+openMin {
			return "open"
		}
		if mins >= 19*60-closeMin {
			return "close"
		}
		return "mid"
	}
	// Будни: основная сессия 10:00–18:40, вечерняя 19:00–23:50 — считаем mid; клиринг 18:40–19:00 — offhours для метрик.
	if mins >= 10*60 && mins < 10*60+openMin {
		return "open"
	}
	if mins >= 18*60+40-closeMin && mins < 18*60+40 {
		return "close"
	}
	if mins >= 10*60+openMin && mins < 18*60+40-closeMin {
		return "mid"
	}
	if mins >= 19*60 && mins < 23*60+50 {
		return "mid"
	}
	// Пре-маркет (будни до 10:00 MSK): при открытых торгах по ISS — условно mid.
	if mins >= 6*60+50 && mins < 10*60 {
		return "mid"
	}
	return "offhours"
}

func getABTestProfile(now time.Time, ticker string, isFutures bool, enabled bool, slotMin int) abTestProfile {
	if !enabled {
		return abTestProfile{Mode: "BASE"}
	}
	if slotMin < 5 {
		slotMin = 5
	}
	base := canonicalTickerKey(ticker)
	if base == "" {
		base = strings.ToUpper(strings.TrimSpace(ticker))
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(base))
	slotIdx := int(now.Unix()/int64(slotMin*60)) + int(h.Sum32()%3)
	if isFutures {
		slotIdx++
	}
	switch slotIdx % 3 {
	case 0:
		return abTestProfile{
			Mode:            "A",
			ConfDelta:       -0.02,
			BuyGateOffset:   +0.03,
			SellGateOffset:  +0.02,
			ShortGateOffset: +0.03,
			RankScoreDelta:  -0.01,
		}
	case 1:
		return abTestProfile{
			Mode: "B",
		}
	default:
		return abTestProfile{
			Mode:            "C",
			ConfDelta:       +0.02,
			BuyGateOffset:   -0.03,
			SellGateOffset:  -0.02,
			ShortGateOffset: -0.03,
			RankScoreDelta:  +0.02,
		}
	}
}

func (a *App) mtfConfirm(ticker, action string) (bool, string) {
	if a.moex == nil || ticker == "" {
		return true, "mtf_skip"
	}
	intervals := []int{1, 10, 60, 24, 7, 31, 4}
	avail := 0
	bull := 0
	bear := 0
	for _, iv := range intervals {
		closes, err := a.moex.GetClosePricesByInterval(ticker, 80, iv)
		if err != nil || len(closes) < 20 {
			continue
		}
		avail++
		rsi := services.RSI(closes, 14)
		if rsi <= 45 {
			bull++
		}
		if rsi >= 55 {
			bear++
		}
	}
	if avail < 2 {
		return true, "mtf_not_enough_data"
	}
	// Базово требуем большинство таймфреймов.
	need := avail/2 + 1
	// Мягкое ослабление: если задано меньшее число подтверждений, берём его.
	// Это помогает не "душить" входы в спокойном рынке.
	relaxNeed := envInt("AGENT_MTF_REQUIRED_CONFIRM", 3)
	if relaxNeed > 0 && relaxNeed < need {
		need = relaxNeed
	}
	// Не опускаемся ниже 2 подтверждений, чтобы не снять фильтр полностью.
	if need < 2 {
		need = 2
	}
	if need > avail {
		need = avail
	}
	if strings.EqualFold(action, "buy") {
		// Для входов buy держим MTF-фильтр строгим, но чуть мягче sell:
		// уменьшаем требуемые подтверждения на 1 (по умолчанию), не ниже 2.
		// Это снижает "тишину" без отключения риск-контура.
		buyNeed := need
		buyRelaxDelta := envFloat("AGENT_MTF_BUY_RELAX_DELTA", 0.0)
		if buyRelaxDelta > 0 {
			buyNeed = int(float64(need) - buyRelaxDelta)
		}
		if buyNeed < 2 {
			buyNeed = 2
		}
		if buyNeed > avail {
			buyNeed = avail
		}
		ok := bull >= buyNeed
		return ok, fmt.Sprintf("mtf buy bull=%d bear=%d avail=%d need=%d", bull, bear, avail, buyNeed)
	}
	if strings.EqualFold(action, "sell") {
		// Для sell тоже добавляем relax (было только для buy)
		sellNeed := need
		sellRelaxDelta := envFloat("AGENT_MTF_SELL_RELAX_DELTA", 0.0)
		if sellRelaxDelta > 0 {
			sellNeed = int(float64(need) - sellRelaxDelta)
		}
		if sellNeed < 2 {
			sellNeed = 2
		}
		if sellNeed > avail {
			sellNeed = avail
		}
		ok := bear >= sellNeed
		return ok, fmt.Sprintf("mtf sell bull=%d bear=%d avail=%d need=%d", bull, bear, avail, sellNeed)
	}
	return true, "mtf_unknown_action"
}

func (a *App) cleanupProfitLock(open map[string]struct{}) {
	a.profitLockMu.Lock()
	defer a.profitLockMu.Unlock()
	for t := range a.profitLockPeakPct {
		if _, ok := open[t]; !ok {
			delete(a.profitLockPeakPct, t)
		}
	}
}

func (a *App) cleanupTrailingStop(open map[string]struct{}) {
	a.trailingStopMu.Lock()
	defer a.trailingStopMu.Unlock()
	if a.trailingStopPrice == nil {
		return
	}
	for t := range a.trailingStopPrice {
		if _, ok := open[t]; !ok {
			delete(a.trailingStopPrice, t)
		}
	}
}

func (a *App) trackOpenPositions(open map[string]struct{}, now time.Time) {
	a.positionAgeMu.Lock()
	defer a.positionAgeMu.Unlock()
	if a.positionOpenedAt == nil {
		a.positionOpenedAt = make(map[string]time.Time)
	}
	for t := range open {
		if _, ok := a.positionOpenedAt[t]; !ok {
			a.positionOpenedAt[t] = now
		}
	}
	for t := range a.positionOpenedAt {
		if _, ok := open[t]; !ok {
			delete(a.positionOpenedAt, t)
		}
	}
}

func (a *App) positionHeldFor(ticker string, now time.Time) time.Duration {
	t := strings.ToUpper(strings.TrimSpace(ticker))
	if t == "" {
		return 0
	}
	a.positionAgeMu.Lock()
	defer a.positionAgeMu.Unlock()
	openedAt, ok := a.positionOpenedAt[t]
	if !ok || openedAt.IsZero() {
		return 0
	}
	return now.Sub(openedAt)
}

func (a *App) markAutoExitOutcome(reason string, pnlPct float64, limit int, pause time.Duration) {
	if limit < 1 || pause <= 0 {
		return
	}
	r := strings.ToLower(strings.TrimSpace(reason))
	a.lossStreakMu.Lock()
	defer a.lossStreakMu.Unlock()
	if r == "sl" || pnlPct < 0 {
		a.autoLossStreak++
		if a.autoLossStreak >= limit {
			a.autoLossPauseUntil = time.Now().Add(pause)
		}
		return
	}
	// Любой неубыточный выход сбрасывает серию.
	a.autoLossStreak = 0
}

func (a *App) getAutoLossPause(limit int) (bool, int, time.Duration) {
	if limit < 1 {
		return false, 0, 0
	}
	a.lossStreakMu.Lock()
	defer a.lossStreakMu.Unlock()
	if a.autoLossStreak < limit || a.autoLossPauseUntil.IsZero() {
		return false, a.autoLossStreak, 0
	}
	remain := time.Until(a.autoLossPauseUntil)
	if remain <= 0 {
		a.autoLossPauseUntil = time.Time{}
		a.autoLossStreak = 0
		return false, 0, 0
	}
	return true, a.autoLossStreak, remain
}

func (a *App) setBuyCashPause(d time.Duration) {
	if d <= 0 {
		return
	}
	a.cashPauseMu.Lock()
	defer a.cashPauseMu.Unlock()
	until := time.Now().Add(d)
	if until.After(a.buyCashPauseUntil) {
		a.buyCashPauseUntil = until
	}
}

func (a *App) getBuyCashPauseLeft() time.Duration {
	a.cashPauseMu.Lock()
	defer a.cashPauseMu.Unlock()
	if a.buyCashPauseUntil.IsZero() {
		return 0
	}
	left := time.Until(a.buyCashPauseUntil)
	if left <= 0 {
		a.buyCashPauseUntil = time.Time{}
		return 0
	}
	return left
}

func isRiskDailyNotionalExceeded(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	if msg == "" {
		return false
	}
	return strings.Contains(msg, "daily notional") && strings.Contains(msg, "would exceed max")
}

func tuneDailyNotionalLimit(prevUsed, prevLimit, minLimit, maxLimit, targetLow, targetHigh, stepUpPct, stepDownPct float64) (float64, float64, string) {
	if prevLimit <= 0 {
		return prevLimit, 0, "limit_disabled"
	}
	if minLimit <= 0 {
		minLimit = 10_000
	}
	if maxLimit < minLimit {
		maxLimit = minLimit
	}
	if targetLow < 0 {
		targetLow = 0
	}
	if targetHigh <= targetLow {
		targetHigh = targetLow + 0.20
	}
	if stepUpPct <= 0 {
		stepUpPct = 10
	}
	if stepDownPct <= 0 {
		stepDownPct = 10
	}

	util := prevUsed / prevLimit
	next := prevLimit
	reason := "in_band"
	if util >= targetHigh {
		next = prevLimit * (1.0 + stepUpPct/100.0)
		reason = "high_util_raise"
	} else if util <= targetLow {
		next = prevLimit * (1.0 - stepDownPct/100.0)
		reason = "low_util_lower"
	}
	if next < minLimit {
		next = minLimit
		reason += "_clamp_min"
	}
	if next > maxLimit {
		next = maxLimit
		reason += "_clamp_max"
	}
	// Нормализуем шагом 1к, чтобы лимит был читаем и стабилен.
	next = float64(int(next/1000.0+0.5) * 1000)
	if next < 1 {
		next = 1
	}
	return next, util * 100.0, reason
}

func (a *App) setRiskDailyPause(d time.Duration, reason string) {
	if d <= 0 {
		return
	}
	if d < 10*time.Second {
		d = 10 * time.Second
	}
	if d > 10*time.Minute {
		d = 10 * time.Minute
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "daily notional exceed"
	}
	now := time.Now()
	until := now.Add(d)
	armed := false
	a.riskDailyPauseMu.Lock()
	if until.After(a.riskDailyPauseUntil) {
		armed = a.riskDailyPauseUntil.Before(now)
		a.riskDailyPauseUntil = until
		a.riskDailyPauseReason = reason
	}
	a.riskDailyPauseMu.Unlock()
	if armed {
		a.addRuntimeEvent("warn", "auto.pause_risk_daily_armed", fmt.Sprintf("pause=%s reason=%s", d.Round(time.Second), reason))
	}
}

func (a *App) armRiskDailyPauseFromError(err error) bool {
	if !isRiskDailyNotionalExceeded(err) {
		return false
	}
	pauseSec := envInt("AGENT_RISK_DAILY_REJECT_PAUSE_SEC", 60)
	if pauseSec < 10 {
		pauseSec = 10
	}
	if pauseSec > 600 {
		pauseSec = 600
	}
	a.setRiskDailyPause(time.Duration(pauseSec)*time.Second, err.Error())
	return true
}

func (a *App) getRiskDailyPauseLeft() time.Duration {
	a.riskDailyPauseMu.Lock()
	defer a.riskDailyPauseMu.Unlock()
	if a.riskDailyPauseUntil.IsZero() {
		return 0
	}
	left := time.Until(a.riskDailyPauseUntil)
	if left <= 0 {
		a.riskDailyPauseUntil = time.Time{}
		a.riskDailyPauseReason = ""
		return 0
	}
	return left
}

func (a *App) setShortOpenPause(d time.Duration) {
	if d <= 0 {
		return
	}
	a.shortPauseMu.Lock()
	defer a.shortPauseMu.Unlock()
	until := time.Now().Add(d)
	if until.After(a.shortOpenPauseUntil) {
		a.shortOpenPauseUntil = until
	}
}

func (a *App) getShortOpenPauseLeft() time.Duration {
	a.shortPauseMu.Lock()
	defer a.shortPauseMu.Unlock()
	if a.shortOpenPauseUntil.IsZero() {
		return 0
	}
	left := time.Until(a.shortOpenPauseUntil)
	if left <= 0 {
		a.shortOpenPauseUntil = time.Time{}
		return 0
	}
	return left
}

func (a *App) setShortOpenTickerPause(ticker string, d time.Duration) {
	if d <= 0 {
		return
	}
	key := canonicalTickerKey(ticker)
	if key == "" {
		key = strings.ToUpper(strings.TrimSpace(ticker))
	}
	if key == "" {
		return
	}
	a.shortOpenTickerMu.Lock()
	defer a.shortOpenTickerMu.Unlock()
	until := time.Now().Add(d)
	if prev, ok := a.shortOpenTickerUntil[key]; !ok || until.After(prev) {
		a.shortOpenTickerUntil[key] = until
	}
}

func (a *App) getShortOpenTickerPauseLeft(ticker string) time.Duration {
	key := canonicalTickerKey(ticker)
	if key == "" {
		key = strings.ToUpper(strings.TrimSpace(ticker))
	}
	if key == "" {
		return 0
	}
	a.shortOpenTickerMu.Lock()
	defer a.shortOpenTickerMu.Unlock()
	until, ok := a.shortOpenTickerUntil[key]
	if !ok || until.IsZero() {
		return 0
	}
	left := time.Until(until)
	if left <= 0 {
		delete(a.shortOpenTickerUntil, key)
		return 0
	}
	return left
}

func (a *App) updateNoOrderStreak(placed bool) int {
	a.noOrderMu.Lock()
	defer a.noOrderMu.Unlock()
	if placed {
		a.autoNoOrderStreak = 0
		return 0
	}
	a.autoNoOrderStreak++
	return a.autoNoOrderStreak
}

func (a *App) getNoOrderStreak() int {
	a.noOrderMu.Lock()
	defer a.noOrderMu.Unlock()
	return a.autoNoOrderStreak
}

func (a *App) getAdaptiveConfThresholds(baseBuy, baseSell float64, noOrderStreak int) (float64, float64) {
	if noOrderStreak <= 0 {
		return baseBuy, baseSell
	}
	// После серии "no_order" мягко снижаем пороги, но оставляем MTF/risk/lot фильтры.
	// Это оживляет входы без снятия контуров безопасности.
	step := float64(noOrderStreak/3) * 0.01
	if step > 0.06 {
		step = 0.06
	}
	buy := baseBuy - step
	sell := baseSell - (step * 0.7)
	if buy < 0.60 {
		buy = 0.60
	}
	if sell < 0.55 {
		sell = 0.55
	}
	return buy, sell
}

func (a *App) getAutoRotateConfig() (bool, float64, time.Duration) {
	enabled := a.autoRotateEnabled
	delta := a.autoRotateMinDelta
	cooldown := a.autoRotateCooldown
	if delta <= 0 {
		delta = 0.12
	}
	if cooldown <= 0 {
		cooldown = 20 * time.Minute
	}
	return enabled, delta, cooldown
}

func (a *App) canAutoRotate(now time.Time) bool {
	a.autoRotateMu.Lock()
	defer a.autoRotateMu.Unlock()
	_, _, cooldown := a.getAutoRotateConfig()
	if a.autoRotateLastAt.IsZero() {
		return true
	}
	return now.Sub(a.autoRotateLastAt) >= cooldown
}

func (a *App) markAutoRotate(now time.Time) {
	a.autoRotateMu.Lock()
	a.autoRotateLastAt = now
	a.autoRotateMu.Unlock()
}

func NewApp() *App {
	loadDotEnvOnce()

	loc, err := time.LoadLocation("Asia/Novosibirsk")
	if err != nil {
		loc = time.FixedZone("UTC+7", 7*60*60)
	}
	cooldownMin := envInt("AGENT_BAD_TICKER_COOLDOWN_MIN", 30)
	if cooldownMin < 1 {
		cooldownMin = 1
	}
	autoBudgetPct := envInt("AGENT_AUTO_BUDGET_PERCENT", 100)
	if autoBudgetPct < 10 {
		autoBudgetPct = 10
	}
	if autoBudgetPct > 100 {
		autoBudgetPct = 100
	}
	autoBudgetPct = (autoBudgetPct / 10) * 10
	if autoBudgetPct < 10 {
		autoBudgetPct = 10
	}
	autoTP := envFloat("AGENT_POSITION_TP_PCT", 1.2)
	autoSL := envFloat("AGENT_POSITION_SL_PCT", 0.8)
	profitLockTrigger := envFloat("AGENT_PROFIT_LOCK_TRIGGER_PCT", 10.0)
	profitLockTrail := envFloat("AGENT_PROFIT_LOCK_TRAIL_PCT", 3.0)
	rotateEnabled := strings.TrimSpace(strings.ToLower(os.Getenv("AGENT_AUTO_ROTATE_ENABLED")))
	autoRotateEnabled := rotateEnabled == "" || rotateEnabled == "1" || rotateEnabled == "true" || rotateEnabled == "yes" || rotateEnabled == "on"
	autoRotateMinDelta := envFloat("AGENT_AUTO_ROTATE_MIN_DELTA", 0.12)
	autoRotateCooldownMin := envInt("AGENT_AUTO_ROTATE_COOLDOWN_MIN", 20)
	maxOrderNotional := envFloat("AGENT_MAX_ORDER_NOTIONAL", 50_000)
	maxDailyNotional := envFloat("AGENT_MAX_DAILY_NOTIONAL", 500_000)
	if maxOrderNotional < 0 {
		maxOrderNotional = 50_000
	}
	if maxDailyNotional < 0 {
		maxDailyNotional = 500_000
	}
	staticTickerDenylist := parseTickerDenylist(os.Getenv("AGENT_TICKER_DENYLIST"))
	stockAllowlistPath := envOrDefault("AGENT_STOCK_ALLOWLIST_PATH", "config/lchi_stocks_allowlist.txt")
	stockAllowlist := loadTickerAllowlist(stockAllowlistPath)
	futuresAllowlistPath := strings.TrimSpace(os.Getenv("AGENT_FUTURES_ALLOWLIST_PATH"))
	futuresAllowlist := loadTickerAllowlist(futuresAllowlistPath)
	futuresAllowlistAutoLearnRaw := strings.TrimSpace(strings.ToLower(os.Getenv("AGENT_FUTURES_ALLOWLIST_AUTO_LEARN")))
	futuresAllowlistAutoLearn := futuresAllowlistAutoLearnRaw == "" ||
		futuresAllowlistAutoLearnRaw == "1" ||
		futuresAllowlistAutoLearnRaw == "true" ||
		futuresAllowlistAutoLearnRaw == "yes" ||
		futuresAllowlistAutoLearnRaw == "on"
	autoInstrumentMode, autoFixedTicker := parseAutoInstrumentEnv(
		os.Getenv("AGENT_AUTO_INSTRUMENT_MODE"),
		os.Getenv("AGENT_AUTO_FIXED_TICKER"),
	)
	exitLearningPath := strings.TrimSpace(envOrDefault("AGENT_EXIT_LEARNING_PATH", "data/exit_learning.json"))
	miyaHintsPath := strings.TrimSpace(envOrDefault("AGENT_MIA_EXIT_HINTS_PATH", "/var/log/muhlya-reports/agent_exit_hints.env"))
	miyaOnlineSignalPath := strings.TrimSpace(envOrDefault("AGENT_MIA_ONLINE_SIGNAL_PATH", "/var/log/muhlya-reports/miya_online_signal.env"))
	// Inline MIYA: встроенный режим Мии (запрос MOEX ISS напрямую)
	miyaInlineEnabled := envBool("AGENT_MIA_INLINE_ENABLED", true)
	miyaInlineTickersRaw := strings.TrimSpace(envOrDefault("ONLINE_MIA_API_TICKERS", "SBER,GAZP,LKOH,ROSN,NVTK,TATN,VTBR,MGNT,NLMK,SNGSP,TATNP,SGZH"))
	miyaInlineTickers := make([]string, 0)
	for _, t := range strings.Split(miyaInlineTickersRaw, ",") {
		t = strings.ToUpper(strings.TrimSpace(t))
		if t != "" {
			miyaInlineTickers = append(miyaInlineTickers, t)
		}
	}
	runtimeDBMaxSize := envInt("AGENT_RUNTIME_EVENTS_DB_LIMIT", 500)
	if runtimeDBMaxSize < 50 {
		runtimeDBMaxSize = 50
	}
	// Раньше был жёсткий верхний предел 5000 (во избежание роста потребления памяти).
	// Подняли, чтобы дать проекту работать с более длинными сессиями/журналами.
	if runtimeDBMaxSize > 50_000 {
		runtimeDBMaxSize = 50_000
	}
	if autoTP <= 0 {
		autoTP = 1.2
	}
	if autoSL <= 0 {
		autoSL = 0.8
	}
	if profitLockTrigger <= 0 {
		profitLockTrigger = 10.0
	}
	if profitLockTrail <= 0 {
		profitLockTrail = 3.0
	}
	if autoRotateMinDelta <= 0 {
		autoRotateMinDelta = 0.12
	}
	if autoRotateCooldownMin < 1 {
		autoRotateCooldownMin = 1
	}
	return &App{
		moex:     services.NewMOEXClient(),
		realtime: make(map[string]chan float64),
		risk: engine.NewRiskManager(engine.RiskConfig{
			Enabled:          true,
			MaxOrderNotional: maxOrderNotional,
			MaxDailyNotional: maxDailyNotional,
		}),
		mode:                      engine.ModeAuto,
		userAutoPreferred:         true,
		autoInstrumentMode:        autoInstrumentMode,
		autoFixedTicker:           autoFixedTicker,
		autoBudgetPct:             autoBudgetPct,
		autoTPPct:                 autoTP,
		autoSLPct:                 autoSL,
		statsCacheTTL:             12 * time.Second,
		orderThrottle:             make(map[string]time.Time),
		minOrderInterval:          15 * time.Second,
		tradingLocation:           loc,
		stockStatusCache:          make(map[string]stockStatusCacheEntry),
		fortsStatusCache:          make(map[string]fortsStatusCacheEntry),
		badTickersUntil:           make(map[string]time.Time),
		badTickerReason:           make(map[string]string),
		recentAutoTickerUntil:     make(map[string]time.Time),
		staticTickerDenylist:      staticTickerDenylist,
		stockAllowlist:            stockAllowlist,
		futuresAllowlist:          futuresAllowlist,
		futuresAllowlistPath:      futuresAllowlistPath,
		futuresAllowlistAutoLearn: futuresAllowlistAutoLearn,
		profitLockPeakPct:         make(map[string]float64),
		trailingStopPrice:         make(map[string]float64),
		positionOpenedAt:          make(map[string]time.Time),
		autoProfitLockTriggerPct:  profitLockTrigger,
		autoProfitLockTrailPct:    profitLockTrail,
		autoRotateEnabled:         autoRotateEnabled,
		autoRotateMinDelta:        autoRotateMinDelta,
		autoRotateCooldown:        time.Duration(autoRotateCooldownMin) * time.Minute,
		exitLearning:              make(map[string]tickerExitLearning),
		exitLearningPath:          exitLearningPath,
		miyaHintsPath:             miyaHintsPath,
		miyaHints:                 make(map[string]float64),
		miyaOnlineSignalPath:      miyaOnlineSignalPath,
		miyaInlineEnabled:         miyaInlineEnabled,
		miyaInlineTickers:         miyaInlineTickers,
		miyaOnlineSignal: miyaOnlineSignal{
			State:            "no_data",
			DataStatus:       "no_data",
			DefaultMul:       1.00,
			UpRatioPct:       0,
			TickersLongCount: 0,
			DailyState:       "no_data",
			DailyDataStatus:  "no_data",
			DailyDefaultMul:  1.00,
			DailyUpRatioPct:  0,
			LiquidityStatus:  "no_data",
		},
		shortOpenTickerUntil: make(map[string]time.Time),
		runtimeMaxSize:       500,
		runtimeDBMaxSize:     runtimeDBMaxSize,
		badTickerCooldown:    time.Duration(cooldownMin) * time.Minute,
	}
}

func cloneStatsMap(src map[string]interface{}) map[string]interface{} {
	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func (a *App) invalidateStatsCache() {
	a.statsMu.Lock()
	a.statsCached = nil
	a.statsCachedAt = time.Time{}
	a.statsMu.Unlock()
}

func (a *App) allowOrderByThrottle(ticker, side string) error {
	key := strings.ToUpper(strings.TrimSpace(ticker)) + "|" + strings.ToLower(strings.TrimSpace(side))
	now := time.Now()
	a.orderThrottleMu.Lock()
	defer a.orderThrottleMu.Unlock()

	if last, ok := a.orderThrottle[key]; ok {
		wait := a.minOrderInterval - now.Sub(last)
		if wait > 0 {
			return fmt.Errorf("throttle: too frequent %s on %s, retry after %s", side, ticker, wait.Round(time.Second))
		}
	}
	a.orderThrottle[key] = now
	return nil
}

func isTerminalOrderStatus(status string) bool {
	s := strings.ToLower(strings.TrimSpace(status))
	if s == "" {
		return false
	}
	return strings.Contains(s, "filled") ||
		strings.Contains(s, "matched") ||
		strings.Contains(s, "cancel") ||
		strings.Contains(s, "reject") ||
		strings.Contains(s, "done") ||
		strings.Contains(s, "executed")
}

func sideMatches(orderSide, side string) bool {
	os := strings.ToLower(strings.TrimSpace(orderSide))
	s := strings.ToLower(strings.TrimSpace(side))
	if os == "" || s == "" {
		return false
	}
	if os == s {
		return true
	}
	if s == "sell" && strings.Contains(os, "sell") {
		return true
	}
	if s == "buy" && strings.Contains(os, "buy") {
		return true
	}
	return false
}

func (a *App) hasPendingOrder(ticker, side string) (bool, string) {
	if a.broker == nil {
		return false, ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	orders, err := a.broker.GetOrders(ctx)
	if err != nil {
		return false, ""
	}
	t := strings.ToUpper(strings.TrimSpace(ticker))
	base := canonicalTickerKey(t)
	for _, o := range orders {
		if isTerminalOrderStatus(o.Status) {
			continue
		}
		ot := strings.ToUpper(strings.TrimSpace(o.Ticker))
		if ot == "" {
			continue
		}
		if ot != t && canonicalTickerKey(ot) != base {
			continue
		}
		if !sideMatches(o.Side, side) {
			continue
		}
		return true, strings.TrimSpace(o.ID)
	}
	return false, ""
}

// pendingOrderTickerSet возвращает карту тикеров с активными (не терминальными) заявками по стороне.
// Ключи включают как полный тикер, так и canonicalTickerKey (без суффиксов), чтобы быстрее отсекать повторы в автоцикле.
func (a *App) pendingOrderTickerSet(side string) map[string]string {
	out := make(map[string]string)
	if a.broker == nil {
		return out
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	orders, err := a.broker.GetOrders(ctx)
	if err != nil {
		return out
	}
	for _, o := range orders {
		if isTerminalOrderStatus(o.Status) || !sideMatches(o.Side, side) {
			continue
		}
		ot := strings.ToUpper(strings.TrimSpace(o.Ticker))
		if ot == "" {
			continue
		}
		id := strings.TrimSpace(o.ID)
		out[ot] = id
		if base := canonicalTickerKey(ot); base != "" {
			if _, ok := out[base]; !ok {
				out[base] = id
			}
		}
	}
	return out
}

func pendingOrderSetSignature(pending map[string]string) string {
	if len(pending) == 0 {
		return ""
	}
	parts := make([]string, 0, len(pending))
	for k, v := range pending {
		key := strings.ToUpper(strings.TrimSpace(k))
		if key == "" {
			continue
		}
		val := strings.TrimSpace(v)
		if val == "" {
			val = "-"
		}
		parts = append(parts, key+"="+val)
	}
	if len(parts) == 0 {
		return ""
	}
	sort.Strings(parts)
	return strings.Join(parts, "|")
}

func (a *App) isPendingBuyGateStale(pending map[string]string, staleAfter time.Duration) (bool, time.Duration) {
	if staleAfter <= 0 {
		return false, 0
	}
	sig := pendingOrderSetSignature(pending)
	now := time.Now()
	a.pendingBuyGateMu.Lock()
	defer a.pendingBuyGateMu.Unlock()
	if sig == "" {
		a.pendingBuyGateSig = ""
		a.pendingBuyGateSince = time.Time{}
		return false, 0
	}
	if sig != a.pendingBuyGateSig {
		a.pendingBuyGateSig = sig
		a.pendingBuyGateSince = now
		return false, 0
	}
	if a.pendingBuyGateSince.IsZero() {
		a.pendingBuyGateSince = now
		return false, 0
	}
	age := now.Sub(a.pendingBuyGateSince)
	return age >= staleAfter, age
}

func (a *App) nowInTradingTZ() time.Time {
	if a.tradingLocation == nil {
		return time.Now()
	}
	return time.Now().In(a.tradingLocation)
}

// getExchangeTradingOpenNow возвращает true, если по MOEX ISS сейчас разрешены торги на площадке.
// Для надежности кэшируем результат на короткий TTL, чтобы не дергать ISS при каждой проверке.
//
// Возвращает ok=false, если статус не удалось получить (тогда вызывающая сторона может использовать fallback).
func (a *App) getExchangeTradingOpenNow() (open bool, reason string, ok bool) {
	if a.moex == nil {
		return false, "", false
	}

	const ttl = 10 * time.Second
	a.tradingStatusMu.Lock()
	if !a.exchangeTradingOpenCachedAt.IsZero() && time.Since(a.exchangeTradingOpenCachedAt) < ttl {
		open := a.exchangeTradingOpenCached
		reason := a.exchangeTradingOpenCachedReason
		a.tradingStatusMu.Unlock()
		return open, reason, true
	}
	a.tradingStatusMu.Unlock()

	// Выбираем эталон: SBER в stock-секции (где TRADINGSTATUS доступен).
	status, err := a.moex.GetTradingStatus("SBER")
	if err != nil {
		return false, "cannot fetch TRADINGSTATUS from MOEX ISS: " + err.Error(), false
	}

	normalized := strings.ToUpper(strings.TrimSpace(status))
	isOpen := normalized == "T"

	reason = ""
	if !isOpen {
		reason = fmt.Sprintf("exchange closed (TRADINGSTATUS=%s)", status)
	}

	a.tradingStatusMu.Lock()
	a.exchangeTradingOpenCached = isOpen
	a.exchangeTradingOpenCachedAt = time.Now()
	a.exchangeTradingOpenCachedReason = reason
	a.tradingStatusMu.Unlock()

	return isOpen, reason, true
}

func tradingStatusCacheTTL() time.Duration {
	sec := envInt("AGENT_TRADINGSTATUS_TTL_SEC", 30)
	if sec < 5 {
		sec = 5
	}
	if sec > 120 {
		sec = 120
	}
	return time.Duration(sec) * time.Second
}

func strictTradingStatusGateEnabled() bool {
	return envBool("AGENT_STRICT_TRADINGSTATUS_GATE", true)
}

func (a *App) getStockTradingStatusCached(ticker string) (string, error) {
	if a.moex == nil {
		return "", fmt.Errorf("no moex client")
	}
	t := canonicalTickerKey(ticker)
	if t == "" {
		return "", fmt.Errorf("empty ticker")
	}
	ttl := tradingStatusCacheTTL()
	a.stockStatusMu.Lock()
	if e, ok := a.stockStatusCache[t]; ok && time.Since(e.at) < ttl {
		s := e.status
		a.stockStatusMu.Unlock()
		return s, nil
	}
	a.stockStatusMu.Unlock()

	s, err := a.moex.GetTradingStatus(t)
	if err != nil {
		return "", err
	}
	a.stockStatusMu.Lock()
	a.stockStatusCache[t] = stockStatusCacheEntry{status: s, at: time.Now()}
	a.stockStatusMu.Unlock()
	return s, nil
}

// defaultFortsHealthSecid — «вечный» фьючерс USD/RUB на MOEX (код USDRUBF), без смены SECID из-за экспирации.
// Описание контракта: https://www.moex.com/en/contract.aspx?code=USDRUBF
const defaultFortsHealthSecid = "USDRUBF"

func fortsHealthSecidFromEnv() string {
	v := strings.TrimSpace(os.Getenv("AGENT_FORTS_MOEX_SECID"))
	if v != "" {
		return strings.ToUpper(v)
	}
	return defaultFortsHealthSecid
}

// fortsMoexSecidCandidates — SECID для запроса статуса в ISS FORTS по конкретному тикеру.
// Сначала AGENT_FORTS_MOEX_SECID (если задан), затем тикер как есть — у Finam и MOEX форматы часто различаются.
// «Здоровье» площадки по умолчанию — отдельно через fortsHealthSecidFromEnv() (USDRUBF).
func fortsMoexSecidCandidates(ticker string) []string {
	seen := make(map[string]struct{})
	var out []string
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		u := strings.ToUpper(s)
		if _, ok := seen[u]; ok {
			return
		}
		seen[u] = struct{}{}
		out = append(out, u)
	}
	if v := os.Getenv("AGENT_FORTS_MOEX_SECID"); v != "" {
		add(v)
	}
	add(ticker)
	return out
}

func (a *App) getFortsTradingStatusCached(secid string) (string, error) {
	if a.moex == nil {
		return "", fmt.Errorf("no moex client")
	}
	secid = strings.TrimSpace(secid)
	if secid == "" {
		return "", fmt.Errorf("empty secid")
	}
	ttl := tradingStatusCacheTTL()
	a.fortsStatusMu.Lock()
	if e, ok := a.fortsStatusCache[secid]; ok && time.Since(e.at) < ttl {
		s := e.status
		a.fortsStatusMu.Unlock()
		return s, nil
	}
	a.fortsStatusMu.Unlock()

	s, err := a.moex.GetFortsTradingStatus(secid)
	if err != nil {
		return "", err
	}
	a.fortsStatusMu.Lock()
	a.fortsStatusCache[secid] = fortsStatusCacheEntry{status: s, at: time.Now()}
	a.fortsStatusMu.Unlock()
	return s, nil
}

func (a *App) localCalendarAndFuturesFallback(now time.Time) (bool, string) {
	date := now.Format("2006-01-02")
	weekday := now.Weekday()

	if a.isHolidayClosedDate(date) {
		return false, "exchange holiday (closed)"
	}

	isWeekendDay := weekday == time.Saturday || weekday == time.Sunday
	if isWeekendDay {
		if a.isWeekendNoTradingDate(date) {
			return false, "weekend closed by calendar"
		}
		if inTimeRange(now, 10, 0, 19, 0) {
			return true, ""
		}
		return false, "weekend session is 10:00-19:00 (Novosibirsk)"
	}

	if a.isWeekendTradingOverrideDate(date) {
		if inTimeRange(now, 10, 0, 19, 0) {
			return true, ""
		}
		return false, "holiday weekend mode is 10:00-19:00 (Novosibirsk)"
	}

	if inTimeRange(now, 8, 50, 23, 50) {
		return true, ""
	}
	if inTimeRange(now, 23, 50, 24, 0) || inTimeRange(now, 0, 0, 0, 30) {
		return false, "futures clearing 23:50-00:30 (Novosibirsk)"
	}
	return false, "futures session is 08:50-23:50 (Novosibirsk)"
}

func (a *App) localCalendarAndStockFallback(now time.Time) (bool, string) {
	date := now.Format("2006-01-02")
	weekday := now.Weekday()

	if a.isHolidayClosedDate(date) {
		return false, "exchange holiday (closed)"
	}

	isWeekendDay := weekday == time.Saturday || weekday == time.Sunday
	if isWeekendDay {
		if a.isWeekendNoTradingDate(date) {
			return false, "weekend closed by calendar"
		}
		if inTimeRange(now, 10, 0, 19, 0) {
			return true, ""
		}
		return false, "weekend session is 10:00-19:00 (Novosibirsk)"
	}

	if a.isWeekendTradingOverrideDate(date) {
		if inTimeRange(now, 10, 0, 19, 0) {
			return true, ""
		}
		return false, "holiday weekend mode is 10:00-19:00 (Novosibirsk)"
	}

	if inTimeRange(now, 6, 50, 19, 0) || inTimeRange(now, 19, 1, 23, 50) {
		return true, ""
	}
	if inTimeRange(now, 19, 0, 19, 1) {
		return false, "stock transition clearing around 19:00 (Novosibirsk)"
	}
	return false, "stock session is 06:50-19:00 and 19:01-23:50 (Novosibirsk)"
}

func (a *App) isHolidayClosedDate(d string) bool {
	// Полное закрытие торгов (2026).
	closed := map[string]struct{}{
		"2026-01-01": {},
		"2026-01-02": {},
		"2026-01-07": {},
		"2026-03-08": {},
		"2026-05-09": {},
		"2026-12-31": {},
	}
	_, ok := closed[d]
	return ok
}

func (a *App) isWeekendNoTradingDate(d string) bool {
	// Выходные без торгов (2026), по предоставленному календарю.
	closed := map[string]struct{}{
		"2026-01-03": {}, "2026-01-04": {},
		"2026-01-10": {}, "2026-01-11": {},
		"2026-02-14": {}, "2026-02-15": {},
		"2026-03-07": {}, "2026-03-21": {}, "2026-03-22": {},
		"2026-05-10": {},
		"2026-06-20": {}, "2026-06-21": {},
		"2026-08-01": {}, "2026-08-02": {},
		"2026-08-15": {}, "2026-08-16": {},
		"2026-09-12": {}, "2026-09-13": {},
		"2026-10-24": {}, "2026-10-25": {},
		"2026-12-05": {}, "2026-12-06": {},
	}
	_, ok := closed[d]
	return ok
}

func (a *App) isWeekendTradingOverrideDate(d string) bool {
	// Праздники с режимом выходного дня (10:00-19:00).
	open := map[string]struct{}{
		"2026-02-23": {},
		"2026-05-01": {},
		"2026-06-12": {},
		"2026-11-04": {},
	}
	_, ok := open[d]
	return ok
}

func inTimeRange(now time.Time, fromHour, fromMin, toHour, toMin int) bool {
	mins := now.Hour()*60 + now.Minute()
	from := fromHour*60 + fromMin
	to := toHour*60 + toMin
	return mins >= from && mins < to
}

func isLikelyFuturesTickerLocal(ticker string) bool {
	// Фьючерсы MOEX имеют формат: 2-4 буквы + 2 цифры (месяц и год)
	// Например: MTZ6, SiZ6, BRX6, GZM6
	// Акции с цифрами: OZON, X5, YNDX — НЕ фьючерсы!
	
	ticker = strings.ToUpper(strings.TrimSpace(ticker))
	
	// Если тикер содержит @ — проверяем суффикс
	if strings.Contains(ticker, "@") {
		return strings.HasSuffix(ticker, "@FUT") || strings.HasSuffix(ticker, "@RTSX")
	}
	
	// Проверяем формат: 2-4 буквы + 2 цифры
	if len(ticker) < 4 || len(ticker) > 6 {
		return false
	}
	
	// Последние 2 символа должны быть цифрами
	lastTwo := ticker[len(ticker)-2:]
	if lastTwo[0] < '0' || lastTwo[0] > '9' || lastTwo[1] < '0' || lastTwo[1] > '9' {
		return false
	}
	
	// Первые символы должны быть буквами
	for i := 0; i < len(ticker)-2; i++ {
		c := ticker[i]
		if c < 'A' || c > 'Z' {
			return false
		}
	}
	
	return true
}

func (a *App) isTradingAllowedForTicker(ticker string) (bool, string) {
	now := a.nowInTradingTZ()
	t := canonicalTickerKey(ticker)

	// Срочный рынок: сначала TRADINGSTATUS по FORTS в ISS (как у акций), иначе — локальное расписание.
	if isLikelyFuturesTickerLocal(t) {
		// SECID для "проверки работы биржи" (может быть недоступен/неподходящим для конкретного пользователя),
		// поэтому используем его как индикатор наличия торгов, но НЕ делаем из него автоматический "жёсткий запрет"
		// для всех остальных FORTS при статусе != T.
		healthSecid := fortsHealthSecidFromEnv()

		// Сначала пытаемся получить TRADINGSTATUS для "кандидатов под конкретный тикер",
		// исключая healthSecid (чтобы не блокировать из-за неподходящего контракта).
		for _, secid := range fortsMoexSecidCandidates(t) {
			if healthSecid != "" && strings.EqualFold(strings.TrimSpace(secid), healthSecid) {
				continue
			}
			status, err := a.getFortsTradingStatusCached(secid)
			if err != nil {
				continue
			}
			n := strings.ToUpper(strings.TrimSpace(status))
			if n == "T" {
				return true, ""
			}
			// Если по конкретному SECID получен явный closed, это более-менее надёжно как запрет.
			return false, fmt.Sprintf("futures closed (TRADINGSTATUS=%s, MOEX %s)", status, secid)
		}

		// Затем используем healthSecid только как "индикатор открыт/закрыт рынок FORTS".
		if healthSecid != "" {
			if status, err := a.getFortsTradingStatusCached(healthSecid); err == nil {
				n := strings.ToUpper(strings.TrimSpace(status))
				if n == "T" {
					return true, ""
				}
			}
		}

		// Если не смогли получить TRADINGSTATUS для FORTS, используем SBER как стоп по закрытию биржи,
		// а дальше — локальный fallback расписания FORTS.
		if exchangeOpen, reason, ok := a.getExchangeTradingOpenNow(); ok && !exchangeOpen {
			return false, reason
		}
		return a.localCalendarAndFuturesFallback(now)
	}

	// Акции: TRADINGSTATUS по SBER + fallback на ручной календарь.
	if strictTradingStatusGateEnabled() {
		if status, err := a.getStockTradingStatusCached(t); err == nil {
			n := strings.ToUpper(strings.TrimSpace(status))
			if n == "T" {
				return true, ""
			}
			return false, fmt.Sprintf("stock closed (TRADINGSTATUS=%s, ticker=%s)", status, t)
		}
	}
	if exchangeOpen, reason, ok := a.getExchangeTradingOpenNow(); ok {
		if !exchangeOpen {
			return false, reason
		}
		return true, ""
	}
	return a.localCalendarAndStockFallback(now)
}

func (a *App) getSessionSnapshot() (bool, string, bool, string, string, string) {
	now := a.nowInTradingTZ()
	stockOpen, stockReason := a.isTradingAllowedForTicker("SBER")
	futuresOpen, futuresReason := a.isTradingAllowedForTicker("Si6")
	return stockOpen, stockReason, futuresOpen, futuresReason, now.Format(time.RFC3339), now.Format("15:04:05")
}

type exitProfileParams struct {
	Name               string
	Reason             string
	TPPct              float64
	SLPct              float64
	TriggerPct         float64
	TrailPct           float64
	StaleBand          float64
	MaxHoldMin         int
	FastMoveTriggerPct float64
}

func (a *App) resolveExitProfile(tpPct, slPct, triggerPct, trailPct, staleBand float64, maxHoldMin int, fastMoveTriggerPct float64, phase string, lowBalance bool, lossStreak int) exitProfileParams {
	mode := strings.ToLower(strings.TrimSpace(envOrDefault("AGENT_EXIT_PROFILE_MODE", "auto")))
	if mode == "" {
		mode = "auto"
	}
	profile := "adaptive"
	reason := "manual_adaptive"
	if mode == "conservative" {
		profile = "conservative"
		reason = "manual_conservative"
	} else if mode == "auto" {
		consStreak := envInt("AGENT_EXIT_AUTO_CONSERVATIVE_STREAK", 2)
		if consStreak < 1 {
			consStreak = 1
		}
		switch {
		case lowBalance:
			profile = "conservative"
			reason = "auto_low_balance"
		case lossStreak >= consStreak:
			profile = "conservative"
			reason = "auto_loss_streak"
		case phase == "open" || phase == "close":
			profile = "conservative"
			reason = "auto_phase_" + phase
		default:
			profile = "adaptive"
			reason = "auto_phase_mid"
		}
	}

	p := exitProfileParams{
		Name:               profile,
		Reason:             reason,
		TPPct:              tpPct,
		SLPct:              slPct,
		TriggerPct:         triggerPct,
		TrailPct:           trailPct,
		StaleBand:          staleBand,
		MaxHoldMin:         maxHoldMin,
		FastMoveTriggerPct: fastMoveTriggerPct,
	}

	if profile == "conservative" {
		p.TPPct *= envFloat("AGENT_EXIT_CONSERVATIVE_TP_MUL", 0.90)
		p.SLPct *= envFloat("AGENT_EXIT_CONSERVATIVE_SL_MUL", 0.75)
		p.TriggerPct *= envFloat("AGENT_EXIT_CONSERVATIVE_TRIGGER_MUL", 0.85)
		p.TrailPct *= envFloat("AGENT_EXIT_CONSERVATIVE_TRAIL_MUL", 0.70)
		p.StaleBand *= envFloat("AGENT_EXIT_CONSERVATIVE_STALE_BAND_MUL", 0.85)
		p.FastMoveTriggerPct *= envFloat("AGENT_EXIT_CONSERVATIVE_FASTMOVE_MUL", 0.85)
		holdMul := envFloat("AGENT_EXIT_CONSERVATIVE_MAX_HOLD_MUL", 1.20)
		p.MaxHoldMin = int(float64(p.MaxHoldMin) * holdMul)
	} else {
		p.TPPct *= envFloat("AGENT_EXIT_ADAPTIVE_TP_MUL", 1.10)
		p.SLPct *= envFloat("AGENT_EXIT_ADAPTIVE_SL_MUL", 1.05)
		p.TriggerPct *= envFloat("AGENT_EXIT_ADAPTIVE_TRIGGER_MUL", 1.05)
		p.TrailPct *= envFloat("AGENT_EXIT_ADAPTIVE_TRAIL_MUL", 1.20)
		p.StaleBand *= envFloat("AGENT_EXIT_ADAPTIVE_STALE_BAND_MUL", 1.15)
		p.FastMoveTriggerPct *= envFloat("AGENT_EXIT_ADAPTIVE_FASTMOVE_MUL", 1.10)
		holdMul := envFloat("AGENT_EXIT_ADAPTIVE_MAX_HOLD_MUL", 0.90)
		p.MaxHoldMin = int(float64(p.MaxHoldMin) * holdMul)
	}

	if p.MaxHoldMin < 15 {
		p.MaxHoldMin = 15
	}
	if p.TPPct <= 0 {
		p.TPPct = tpPct
	}
	if p.SLPct <= 0 {
		p.SLPct = slPct
	}
	if p.TriggerPct <= 0 {
		p.TriggerPct = triggerPct
	}
	if p.TrailPct <= 0 {
		p.TrailPct = trailPct
	}
	if p.StaleBand <= 0 {
		p.StaleBand = staleBand
	}
	if p.FastMoveTriggerPct <= 0 {
		p.FastMoveTriggerPct = fastMoveTriggerPct
	}
	return p
}

func clampFloat(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (a *App) loadExitLearning() {
	path := strings.TrimSpace(a.exitLearningPath)
	if path == "" {
		return
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var s exitLearningStore
	if err := json.Unmarshal(raw, &s); err != nil {
		return
	}
	a.exitLearnMu.Lock()
	if a.exitLearning == nil {
		a.exitLearning = make(map[string]tickerExitLearning)
	}
	for k, v := range s.Tickers {
		key := canonicalTickerKey(k)
		if key == "" {
			key = canonicalTickerKey(v.Ticker)
		}
		if key == "" {
			continue
		}
		a.exitLearning[key] = v
	}
	a.exitLearnMu.Unlock()
}

func (a *App) persistExitLearningLocked() {
	path := strings.TrimSpace(a.exitLearningPath)
	if path == "" {
		return
	}
	bodyObj := exitLearningStore{
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		Tickers:   a.exitLearning,
	}
	body, err := json.MarshalIndent(bodyObj, "", "  ")
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		return
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
	}
}

func (a *App) calcLearnedTPMul(v tickerExitLearning) float64 {
	minSamples := envInt("AGENT_EXIT_LEARNING_MIN_SAMPLES", 5)
	if minSamples < 2 {
		minSamples = 2
	}
	if minSamples > 40 {
		minSamples = 40
	}
	if v.Samples < minSamples {
		return 1.00
	}
	winRate := float64(v.Wins) / float64(maxInt(v.Samples, 1))
	avgWin := 0.0
	if v.Wins > 0 {
		avgWin = v.SumWinPct / float64(v.Wins)
	}
	avgLossAbs := 0.0
	if v.Losses > 0 {
		avgLossAbs = absFloat(v.SumLossPct / float64(v.Losses))
	}
	mul := 1.00
	if winRate >= 0.55 {
		mul += clampFloat((winRate-0.5)*0.8+avgWin*4.0, 0, 0.30)
	} else if winRate < 0.45 {
		mul -= clampFloat((0.45-winRate)*0.9, 0, 0.20)
	}
	if avgLossAbs > 0 && avgWin > 0 && avgLossAbs > avgWin*1.4 && v.Losses >= 3 {
		mul -= 0.08
	}
	return clampFloat(mul, 0.75, 1.45)
}

func (a *App) loadMiyaHintsIfNeeded(now time.Time) {
	path := strings.TrimSpace(a.miyaHintsPath)
	if path == "" {
		return
	}
	a.miyaHintsMu.Lock()
	if now.Before(a.miyaHintsUntil) {
		a.miyaHintsMu.Unlock()
		return
	}
	a.miyaHintsUntil = now.Add(2 * time.Minute)
	a.miyaHintsMu.Unlock()

	raw, err := os.ReadFile(path)
	if err != nil {
		return
	}
	next := make(map[string]float64)
	sc := bufio.NewScanner(strings.NewReader(string(raw)))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.ToUpper(strings.TrimSpace(parts[0]))
		val, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err != nil {
			continue
		}
		next[key] = clampFloat(val, 0.70, 1.50)
	}
	a.miyaHintsMu.Lock()
	a.miyaHints = next
	a.miyaHintsMu.Unlock()
}

func normalizeMiyaOnlineState(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "bullish", "bearish", "neutral", "no_data":
		return strings.ToLower(strings.TrimSpace(raw))
	default:
		return "no_data"
	}
}

func normalizeMiyaOnlineDataStatus(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "ok", "no_data", "fallback_active":
		return strings.ToLower(strings.TrimSpace(raw))
	default:
		return "no_data"
	}
}

func normalizeMiyaOnlineDailyDataStatus(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "ok", "no_data", "partial", "disabled", "stale_api":
		return strings.ToLower(strings.TrimSpace(raw))
	default:
		return "no_data"
	}
}

func (a *App) loadMiyaOnlineSignalIfNeeded(now time.Time) {
	a.miyaOnlineSignalMu.Lock()
	if now.Before(a.miyaOnlineSignalTill) {
		a.miyaOnlineSignalMu.Unlock()
		return
	}
	a.miyaOnlineSignalTill = now.Add(30 * time.Second)
	a.miyaOnlineSignalMu.Unlock()

	path := strings.TrimSpace(a.miyaOnlineSignalPath)
	if path != "" {
		raw, err := os.ReadFile(path)
		if err == nil {
			a.parseMiyaOnlineSignalFile(raw)
			return
		}
	}
	// Файл не найден или путь пустой — пробуем inline MOEX
	if a.miyaInlineEnabled && len(a.miyaInlineTickers) > 0 {
		a.fetchMiyaInlineFromMOEX()
	}
}

func (a *App) parseMiyaOnlineSignalFile(raw []byte) {
	next := miyaOnlineSignal{
		State:            "no_data",
		DataStatus:       "no_data",
		DefaultMul:       1.00,
		UpRatioPct:       0,
		TickersLongCount: 0,
		DailyState:       "no_data",
		DailyDataStatus:  "no_data",
		DailyDefaultMul:  1.00,
		DailyUpRatioPct:  0,
		LiquidityOrder:   nil,
		LiquidityStatus:  "no_data",
		WatchTickers:     nil,
	}
	sc := bufio.NewScanner(strings.NewReader(string(raw)))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.ToUpper(strings.TrimSpace(parts[0]))
		val := strings.TrimSpace(parts[1])
		switch key {
		case "STATE":
			next.State = normalizeMiyaOnlineState(val)
		case "DATA_STATUS":
			next.DataStatus = normalizeMiyaOnlineDataStatus(val)
		case "DEFAULT":
			if f, parseErr := strconv.ParseFloat(val, 64); parseErr == nil {
				next.DefaultMul = clampFloat(f, 0.70, 1.50)
			}
		case "UP_RATIO_PCT":
			if f, parseErr := strconv.ParseFloat(val, 64); parseErr == nil {
				next.UpRatioPct = clampFloat(f, 0, 100)
			}
		case "TICKERS_LONG_COUNT":
			if n, parseErr := strconv.Atoi(val); parseErr == nil && n >= 0 {
				next.TickersLongCount = n
			}
		case "DAILY_STATE":
			next.DailyState = normalizeMiyaOnlineState(val)
		case "DAILY_DATA_STATUS":
			next.DailyDataStatus = normalizeMiyaOnlineDailyDataStatus(val)
		case "DAILY_DEFAULT":
			if f, parseErr := strconv.ParseFloat(val, 64); parseErr == nil {
				next.DailyDefaultMul = clampFloat(f, 0.70, 1.50)
			}
		case "DAILY_UP_RATIO_PCT":
			if f, parseErr := strconv.ParseFloat(val, 64); parseErr == nil {
				next.DailyUpRatioPct = clampFloat(f, 0, 100)
			}
		case "MIYA_LIQUIDITY_ORDER":
			parts := strings.Split(val, ",")
			out := make([]string, 0, len(parts))
			for _, p := range parts {
				t := strings.ToUpper(strings.TrimSpace(p))
				if t != "" {
					out = append(out, t)
				}
			}
			next.LiquidityOrder = out
		case "MIYA_LIQUIDITY_STATUS":
			next.LiquidityStatus = normalizeMiyaLiquidityStatus(val)
		case "MIYA_WATCH_TICKERS":
			parts := strings.Split(val, ",")
			out := make([]string, 0, len(parts))
			for _, p := range parts {
				t := strings.ToUpper(strings.TrimSpace(p))
				if t != "" {
					out = append(out, t)
				}
			}
			next.WatchTickers = out
		}
	}
	if next.State == "no_data" {
		next.DataStatus = "no_data"
		next.DefaultMul = 1.00
	}
	if next.DataStatus == "no_data" {
		next.State = "no_data"
		next.DefaultMul = 1.00
	}
	a.miyaOnlineSignalMu.Lock()
	a.miyaOnlineSignal = next
	a.miyaOnlineSignalMu.Unlock()
}

// fetchMiyaInlineFromMOEX — встроенный режим Мии: запрашивает MOEX ISS напрямую
// для списка тикеров и вычисляет состояние рынка (bullish/bearish/neutral).
func (a *App) fetchMiyaInlineFromMOEX() {
	if a.moex == nil || len(a.miyaInlineTickers) == 0 {
		return
	}

	snapshot := a.moex.GetMarketSnapshot(a.miyaInlineTickers)
	if len(snapshot) == 0 {
		return
	}

	// Загружаем аналитику: RSI(14), SMA20, SMA50, тренд
	a.moex.EnrichWithAnalytics(snapshot)

	upCount := 0
	totalCount := 0
	trendBullCount := 0
	rsiOversold := 0  // RSI < 30
	rsiOverbought := 0 // RSI > 70
	rsiSum := 0.0
	rsiCount := 0
	type liqEntry struct {
		ticker   string
		turnover float64
	}
	liquidityData := make([]liqEntry, 0, len(snapshot))
	recommendations := make([]MiyaRecommendation, 0, len(snapshot))
	maxTurnover := 0.0

	// Первый проход: собираем данные и находим макс. ликвидность
	for _, item := range snapshot {
		if !item.TradingActive {
			continue
		}
		totalCount++
		if item.Change > 0 {
			upCount++
		}
		if item.TrendBull {
			trendBullCount++
		}
		if item.RSI14 > 0 {
			rsiSum += item.RSI14
			rsiCount++
			if item.RSI14 < 30 {
				rsiOversold++
			} else if item.RSI14 > 70 {
				rsiOverbought++
			}
		}
		liquidityData = append(liquidityData, liqEntry{item.Ticker, item.TurnoverRub})
		if item.TurnoverRub > maxTurnover {
			maxTurnover = item.TurnoverRub
		}
	}

	// Второй проход: вычисляем score и формируем рекомендации
	for _, item := range snapshot {
		if !item.TradingActive {
			continue
		}
		score := 50.0 // базовый score
		reasons := make([]string, 0, 4)

		// RSI: перепроданный = хорошо для покупки, перекупленный = плохо
		if item.RSI14 > 0 {
			if item.RSI14 < 30 {
				score += 25
				reasons = append(reasons, fmt.Sprintf("RSI=%.0f(перепродан)", item.RSI14))
			} else if item.RSI14 < 50 {
				score += 15
				reasons = append(reasons, fmt.Sprintf("RSI=%.0f(низкий)", item.RSI14))
			} else if item.RSI14 > 70 {
				score -= 20
				reasons = append(reasons, fmt.Sprintf("RSI=%.0f(перекуплен)", item.RSI14))
			}
		}

		// Тренд: бычий = хорошо
		if item.TrendBull {
			score += 20
			reasons = append(reasons, "тренд↑")
		} else {
			score -= 10
			reasons = append(reasons, "тренд↓")
		}

		// Ликвидность: более ликвидные предпочтительнее
		if maxTurnover > 0 {
			liqScore := (item.TurnoverRub / maxTurnover) * 15
			score += liqScore
		}

		// Дневное изменение: небольшое падение = возможность входа
		if item.Change > 0 && item.Change < 2 {
			score += 5
			reasons = append(reasons, fmt.Sprintf("день+%.1f%%", item.Change))
		} else if item.Change < 0 && item.Change > -3 {
			score += 10 // падение в бычьем тренде = возможность
			reasons = append(reasons, fmt.Sprintf("откат%.1f%%", item.Change))
		}

		// Ограничиваем score диапазоном 0-100
		if score < 0 {
			score = 0
		}
		if score > 100 {
			score = 100
		}

		trend := "bear"
		if item.TrendBull {
			trend = "bull"
		}

		reasonStr := ""
		if len(reasons) > 0 {
			reasonStr = strings.Join(reasons, ", ")
		}

		recommendations = append(recommendations, MiyaRecommendation{
			Ticker: item.Ticker,
			Score:  math.Round(score*10) / 10,
			RSI:    math.Round(item.RSI14*10) / 10,
			Trend:  trend,
			Reason: reasonStr,
		})
	}

	// Сортируем рекомендации по score (убывание)
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Score > recommendations[j].Score
	})
	// Оставляем топ-5
	if len(recommendations) > 5 {
		recommendations = recommendations[:5]
	}

	upRatioPct := 0.0
	if totalCount > 0 {
		upRatioPct = float64(upCount) / float64(totalCount) * 100.0
	}

	state := "neutral"
	if upRatioPct >= 55 {
		state = "bullish"
	} else if upRatioPct <= 35 {
		state = "bearish"
	}

	// Улучшенное определение state с учётом тренда
	trendBullPct := 0.0
	if totalCount > 0 {
		trendBullPct = float64(trendBullCount) / float64(totalCount) * 100.0
	}
	// Если большинство тикеров в бычьем тренде, корректируем state
	if trendBullPct >= 60 && state == "bearish" {
		state = "neutral" // рынок разворачивается
	} else if trendBullPct <= 20 && state == "bullish" {
		state = "neutral" // рынок разворачивается
	}

	// Логируем расширенную аналитику Мии
	rsiAvg := 0.0
	if rsiCount > 0 {
		rsiAvg = rsiSum / float64(rsiCount)
	}
	a.addRuntimeEvent("info", "miya.analytics", fmt.Sprintf("tickers=%d up=%.1f%% trend_bull=%d rsi_avg=%.0f rsi_oversold=%d rsi_overbought=%d state=%s",
		totalCount, upRatioPct, trendBullCount, rsiAvg, rsiOversold, rsiOverbought, state))

	// Логируем топ-5 рекомендаций Мии
	if len(recommendations) > 0 {
		recParts := make([]string, 0, len(recommendations))
		for _, r := range recommendations {
			recParts = append(recParts, fmt.Sprintf("%s(%.0f,%s)", r.Ticker, r.Score, r.Trend))
		}
		a.addRuntimeEvent("info", "miya.recommendations", fmt.Sprintf("top5: %s", strings.Join(recParts, " ")))
	}

	sort.Slice(liquidityData, func(i, j int) bool {
		return liquidityData[i].turnover > liquidityData[j].turnover
	})
	liquidityOrder := make([]string, 0, len(liquidityData))
	for _, ld := range liquidityData {
		liquidityOrder = append(liquidityOrder, ld.ticker)
	}

	next := miyaOnlineSignal{
		State:            state,
		DataStatus:       "ok",
		DefaultMul:       1.00,
		UpRatioPct:       math.Round(upRatioPct*100) / 100,
		TickersLongCount: totalCount,
		DailyState:       "no_data",
		DailyDataStatus:  "no_data",
		DailyDefaultMul:  1.00,
		DailyUpRatioPct:  0,
		LiquidityOrder:   liquidityOrder,
		LiquidityStatus:  "ok",
		WatchTickers:     a.miyaInlineTickers,
		Recommendations:  recommendations,
		TrendBullCount:   trendBullCount,
		RSIAverage:       math.Round(rsiAvg*10) / 10,
	}

	a.miyaOnlineSignalMu.Lock()
	a.miyaOnlineSignal = next
	a.miyaOnlineSignalMu.Unlock()
}

func (a *App) getMiyaOnlineSignal(now time.Time) miyaOnlineSignal {
	a.loadMiyaOnlineSignalIfNeeded(now)
	a.miyaOnlineSignalMu.Lock()
	defer a.miyaOnlineSignalMu.Unlock()
	return a.miyaOnlineSignal
}

func (a *App) getTickerExitTPMultiplier(ticker string, now time.Time) (float64, float64, float64) {
	key := canonicalTickerKey(ticker)
	if key == "" {
		return 1.00, 1.00, 1.00
	}
	a.loadMiyaHintsIfNeeded(now)
	learnMul := 1.00
	a.exitLearnMu.Lock()
	if v, ok := a.exitLearning[key]; ok {
		learnMul = a.calcLearnedTPMul(v)
	}
	a.exitLearnMu.Unlock()

	hintMul := 1.00
	a.miyaHintsMu.Lock()
	if a.miyaHints != nil {
		if v, ok := a.miyaHints[key]; ok {
			hintMul = v
		} else if v, ok := a.miyaHints["DEFAULT"]; ok {
			hintMul = v
		}
	}
	a.miyaHintsMu.Unlock()
	finalMul := clampFloat(learnMul*hintMul, 0.70, 1.60)
	return learnMul, hintMul, finalMul
}

func (a *App) rememberExitLearning(ticker, reason, phase string, pnlPct, learnMul, hintMul, finalMul float64) {
	key := canonicalTickerKey(ticker)
	if key == "" {
		return
	}
	a.exitLearnMu.Lock()
	if a.exitLearning == nil {
		a.exitLearning = make(map[string]tickerExitLearning)
	}
	v := a.exitLearning[key]
	v.Ticker = key
	v.Samples++
	if pnlPct > 0 {
		v.Wins++
		v.SumWinPct += pnlPct
	} else if pnlPct < 0 {
		v.Losses++
		v.SumLossPct += pnlPct
	}
	v.LastReason = reason
	v.LastPhase = phase
	v.LastPnlPct = pnlPct
	v.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	v.TpMulLearn = learnMul
	v.TpMulHint = hintMul
	v.TpMulFinal = finalMul
	a.exitLearning[key] = v
	a.persistExitLearningLocked()
	a.exitLearnMu.Unlock()
}

func (a *App) normalizeOrderQuantity(ticker string, quantity int, side string) (int, error) {
	if quantity <= 0 {
		return 0, fmt.Errorf("invalid quantity: %d", quantity)
	}
	lot := 1
	if a.moex != nil {
		lot = a.moex.GetLotSize(ticker)
	}
	if lot <= 1 {
		return quantity, nil
	}

	if quantity%lot == 0 {
		return quantity, nil
	}

	if strings.EqualFold(side, "buy") {
		// Для buy округляем вверх до ближайшего лота, чтобы не получать INVALID_ARGUMENT.
		return ((quantity / lot) + 1) * lot, nil
	}

	// Для sell округляем вниз, чтобы не превысить доступный остаток.
	normalized := (quantity / lot) * lot
	if normalized <= 0 {
		return 0, fmt.Errorf("quantity %d is below lot size %d for %s", quantity, lot, ticker)
	}
	return normalized, nil
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	fmt.Println("🚀 MOEX Trading Terminal starting...")
	a.addRuntimeEvent("info", "startup.begin", "terminal starting")

	loadDotEnvOnce()
	a.loadExitLearning()

	// Подключаемся к PostgreSQL (конфиг через env)
	pgHost := envOrDefault("PGHOST", "localhost")
	pgPort := envOrDefault("PGPORT", "5433")
	pgUser := envOrDefault("PGUSER", "trader")
	pgPassword := os.Getenv("PGPASSWORD")
	pgDatabase := envOrDefault("PGDATABASE", "moex_trading")

	db, err := services.NewPostgresDB(pgHost, pgPort, pgUser, pgPassword, pgDatabase)
	if err != nil {
		fmt.Printf("⚠️  DB error: %v\n", err)
		a.addRuntimeEvent("error", "startup.db_error", err.Error())
	} else {
		a.db = db
		fmt.Println("✅ Connected to PostgreSQL")
		a.addRuntimeEvent("info", "startup.db_connected", fmt.Sprintf("%s:%s/%s", pgHost, pgPort, pgDatabase))
	}

	// Подключаемся к Финам API (только реальный счёт)
	secretReal := os.Getenv("FINAM_SECRET_REAL")

	if secretReal != "" {
		broker, err := services.NewFinamBroker(secretReal)
		if err != nil {
			fmt.Printf("⚠️  Finam error: %v\n", err)
			fmt.Println("   Trading will use simulation")
			a.addRuntimeEvent("error", "startup.finam_error", err.Error())
		} else {
			a.broker = broker
			// Передаём MOEXClient для доступа к кэшу SHORTNAME (маппинг тикеров фьючерсов)
			broker.SetMOEXClient(a.moex)
			fmt.Println("✅ Connected to Finam Trade API (REAL account)")
			a.addRuntimeEvent("info", "startup.finam_connected", "connected to real account")
		}
	} else {
		fmt.Println("⚠️  FINAM_SECRET_REAL not set. Using simulation mode.")
		a.addRuntimeEvent("warn", "startup.simulation_mode", "FINAM_SECRET_REAL is not set")
	}

	a.agent = services.NewMOEXTradingAssistant()
	fmt.Println("✅ Ready")
	a.addRuntimeEvent("info", "startup.ready", "app ready")
	if len(a.stockAllowlist) > 0 {
		a.addRuntimeEvent("info", "startup.stock_allowlist_loaded", fmt.Sprintf("stocks=%d", len(a.stockAllowlist)))
	}
	if len(a.futuresAllowlist) > 0 {
		a.addRuntimeEvent("info", "startup.futures_allowlist_loaded", fmt.Sprintf("futures=%d", len(a.futuresAllowlist)))
	}
	if strings.TrimSpace(a.futuresAllowlistPath) != "" {
		a.addRuntimeEvent("info", "startup.futures_allowlist_mode", fmt.Sprintf("auto_learn=%t", a.futuresAllowlistAutoLearn))
	}
	a.addRuntimeEvent("info", "market.session_rules", "stock/futures sessions are tracked independently with clearing windows")

	// Синхронизируем позиции из Finam API при старте
	a.syncPositionsFromBroker()

	// Запускаем автообновление тикеров фьючерсов из MOEX ISS
	a.startFuturesOverridesAutoUpdate()

	// Инициализируем TTL кэша Daily Top-N
	topNTTLHours := envInt("AGENT_DAILY_TOP_CACHE_TTL_HOURS", 8)
	a.dailyTopNCacheTTL = time.Duration(topNTTLHours) * time.Hour

	// По умолчанию запускаем авто-режим после старта, если есть рабочий брокер.
	// В любой момент можно перейти в manual через UI.
	if a.broker != nil && a.userAutoPreferred {
		if a.agent != nil {
			a.agent.SetActive(true)
		}
		a.mode = engine.ModeAuto
		if a.StartAutoTrading() {
			a.addRuntimeEvent("info", "startup.auto_enabled", "auto mode started by default")
		} else {
			a.addRuntimeEvent("warn", "startup.auto_enable_failed", "failed to start auto mode on startup")
		}
	}
}

func (a *App) GetMode() string {
	return string(a.mode)
}

func (a *App) SetMode(mode string) bool {
	switch engine.Mode(mode) {
	case engine.ModeManual, engine.ModeAuto, engine.ModePaper:
		a.mode = engine.Mode(mode)
		if a.mode == engine.ModeAuto {
			a.userAutoPreferred = true
		}
		if a.mode == engine.ModeManual {
			a.userAutoPreferred = false
		}
		a.addRuntimeEvent("info", "mode.changed", fmt.Sprintf("mode=%s", mode))
		return true
	default:
		a.addRuntimeEvent("warn", "mode.rejected", fmt.Sprintf("unknown mode=%s", mode))
		return false
	}
}

func (a *App) StartAutoTrading() bool {
	a.autoMu.Lock()
	defer a.autoMu.Unlock()

	if a.autoAlive {
		a.addRuntimeEvent("info", "auto.start_skip", "already running")
		return true
	}
	a.userAutoPreferred = true

	ctx, cancel := context.WithCancel(context.Background())
	a.autoStop = cancel
	a.autoAlive = true

	// Автоотмена зависших pending buy ордеров при старте
	a.cancelStalePendingBuyOrders(ctx, 5*time.Minute)

	go a.autoLoop(ctx)
	a.addRuntimeEvent("info", "auto.started", "autotrading loop started")
	return true
}

// cancelStalePendingBuyOrders отменяет все pending buy ордера, висящие дольше staleAfter.
func (a *App) cancelStalePendingBuyOrders(ctx context.Context, staleAfter time.Duration) {
	if a.broker == nil {
		return
	}
	orders, err := a.broker.GetOrders(ctx)
	if err != nil {
		return
	}
	cancelled := 0
	for _, o := range orders {
		if isTerminalOrderStatus(o.Status) {
			continue
		}
		if !strings.EqualFold(o.Side, "buy") && o.Side != "SIDE_BUY" {
			continue
		}
		// Отменяем все pending buy ордера при старте (они могут зависнуть)
		cancelCtx, cancelFn := context.WithTimeout(ctx, 5*time.Second)
		if err := a.broker.CancelOrder(cancelCtx, o.ID); err == nil {
			cancelled++
			a.addRuntimeEvent("info", "auto.cancelled_stale_pending", fmt.Sprintf("order_id=%s ticker=%s", o.ID, o.Ticker))
		}
		cancelFn()
	}
	if cancelled > 0 {
		a.addRuntimeEvent("info", "auto.stale_pending_cleaned", fmt.Sprintf("cancelled=%d", cancelled))
	}
}

// SetAutoInstrumentConfig задаёт режим выбора инструмента в автоторговле.
// mode:
// - "auto"    : весь доступный пул (акции + фьючерсы),
// - "stocks"  : только акции,
// - "futures" : только фьючерсы,
// - "options" : только опционы (пока заглушка, вернет пустой пул),
// - "fixed"   : только указанный ticker.
func (a *App) SetAutoInstrumentConfig(mode string, ticker string) bool {
	mode = strings.ToLower(strings.TrimSpace(mode))
	switch mode {
	case "", "auto":
		mode = "auto"
	case "fixed", "stocks", "futures", "options":
		// ok
	default:
		return false
	}
	ticker = strings.ToUpper(strings.TrimSpace(ticker))
	if mode == "fixed" && ticker == "" {
		return false
	}
	if mode != "fixed" {
		ticker = ""
	}

	a.autoCfgMu.Lock()
	a.autoInstrumentMode = mode
	a.autoFixedTicker = ticker
	a.autoCfgMu.Unlock()
	a.addRuntimeEvent("info", "auto.instrument_config", fmt.Sprintf("mode=%s ticker=%s", mode, ticker))
	return true
}

// GetAutoInstrumentConfig возвращает текущую настройку выбора инструмента автоторговли.
func (a *App) GetAutoInstrumentConfig() map[string]string {
	a.autoCfgMu.RLock()
	defer a.autoCfgMu.RUnlock()
	return map[string]string{
		"mode":   a.autoInstrumentMode,
		"ticker": a.autoFixedTicker,
	}
}

func (a *App) StopAutoTrading() bool {
	a.autoMu.Lock()
	defer a.autoMu.Unlock()
	if !a.autoAlive {
		a.addRuntimeEvent("info", "auto.stop_skip", "already stopped")
		return true
	}
	a.userAutoPreferred = false
	if a.autoStop != nil {
		a.autoStop()
	}
	a.autoAlive = false
	a.autoStop = nil
	a.addRuntimeEvent("info", "auto.stopped", "autotrading loop stopped")
	return true
}

func (a *App) autoLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	lastAutoBudgetBump := time.Time{}
	tpCfg, slCfg := a.getAutoTPSLPercent()
	tpPct := tpCfg / 100.0
	slPct := slCfg / 100.0
	shortConfThreshold := envFloat("AGENT_SHORT_CONF_THRESHOLD", 0.70)
	buyConfThreshold := envFloat("AGENT_BUY_CONF_THRESHOLD", 0.68)
	sellConfThreshold := envFloat("AGENT_SELL_CONF_THRESHOLD", 0.60)
	abTestEnabled := envBool("AGENT_ABTEST_ENABLED", false)
	abTestSlotMin := envInt("AGENT_ABTEST_SLOT_MIN", 15)
	if abTestSlotMin < 5 {
		abTestSlotMin = 5
	}
	maxBuyAttemptsPerCycle := envInt("AGENT_MAX_BUY_ATTEMPTS_PER_CYCLE", 6)
	if maxBuyAttemptsPerCycle < 1 {
		maxBuyAttemptsPerCycle = 1
	}
	if maxBuyAttemptsPerCycle > 20 {
		maxBuyAttemptsPerCycle = 20
	}
	openLongBiasEnabled := envBool("AGENT_OPEN_LONG_BIAS", true)
	openBuyGateDelta := envFloat("AGENT_OPEN_BUY_GATE_DELTA", 0.03)
	if openBuyGateDelta < 0 {
		openBuyGateDelta = 0
	}
	if openBuyGateDelta > 0.15 {
		openBuyGateDelta = 0.15
	}
	openShortGatePenalty := envFloat("AGENT_OPEN_SHORT_GATE_PENALTY", 0.10)
	if openShortGatePenalty < 0 {
		openShortGatePenalty = 0
	}
	if openShortGatePenalty > 0.30 {
		openShortGatePenalty = 0.30
	}
	openMaxBuyAttemptsBonus := envInt("AGENT_OPEN_MAX_BUY_ATTEMPTS_BONUS", 2)
	if openMaxBuyAttemptsBonus < 0 {
		openMaxBuyAttemptsBonus = 0
	}
	if openMaxBuyAttemptsBonus > 6 {
		openMaxBuyAttemptsBonus = 6
	}
	openStocksOnly := envBool("AGENT_OPEN_STOCKS_ONLY", false)
	miyaOnlineHintsEnabled := envBool("AGENT_MIA_ONLINE_HINTS_ENABLED", true)
	miyaOnlineBuyGateMaxDelta := envFloat("AGENT_MIA_ONLINE_BUY_GATE_MAX_DELTA", 0.04)
	if miyaOnlineBuyGateMaxDelta < 0 {
		miyaOnlineBuyGateMaxDelta = 0
	}
	if miyaOnlineBuyGateMaxDelta > 0.15 {
		miyaOnlineBuyGateMaxDelta = 0.15
	}
	miyaOnlineShortGateMaxDelta := envFloat("AGENT_MIA_ONLINE_SHORT_GATE_MAX_DELTA", 0.15)
	if miyaOnlineShortGateMaxDelta < 0 {
		miyaOnlineShortGateMaxDelta = 0
	}
	if miyaOnlineShortGateMaxDelta > 0.30 {
		miyaOnlineShortGateMaxDelta = 0.30
	}
	miyaWatchFirst := envBool("AGENT_MIA_WATCH_FIRST_ENABLED", true)
	miyaLiqRankEnabled := envBool("AGENT_MIA_LIQUIDITY_RANK_ENABLED", true)
	miyaLiqRankWeight := envFloat("AGENT_MIA_LIQUIDITY_RANK_WEIGHT", 0.02)
	if miyaLiqRankWeight < 0 {
		miyaLiqRankWeight = 0
	}
	if miyaLiqRankWeight > 0.08 {
		miyaLiqRankWeight = 0.08
	}
	// Бонус за рекомендации Мии (score 0-100 → бонус к rankScore)
	miyaRecWeight := envFloat("AGENT_MIA_RECOMMENDATION_WEIGHT", 0.15)
	if miyaRecWeight < 0 {
		miyaRecWeight = 0
	}
	if miyaRecWeight > 0.30 {
		miyaRecWeight = 0.30
	}
	// Бонус к conf за рекомендации Мии (score 0-100 → бонус к confidence)
	miyaRecConfBonus := envFloat("AGENT_MIA_RECOMMENDATION_CONF_BONUS", 0.20)
	if miyaRecConfBonus < 0 {
		miyaRecConfBonus = 0
	}
	if miyaRecConfBonus > 0.30 {
		miyaRecConfBonus = 0.30
	}
	shortOpenEnabled := envBool("AGENT_SHORT_OPEN_ENABLED", true)
	dailyNotionalTuneEnabled := envBool("AGENT_DAILY_NOTIONAL_TUNE_ENABLED", true)
	dailyNotionalTuneApply := envBool("AGENT_DAILY_NOTIONAL_TUNE_APPLY", false)
	dailyNotionalTuneMin := envFloat("AGENT_DAILY_NOTIONAL_TUNE_MIN", 120_000)
	dailyNotionalTuneMax := envFloat("AGENT_DAILY_NOTIONAL_TUNE_MAX", 600_000)
	dailyNotionalTuneTargetLow := envFloat("AGENT_DAILY_NOTIONAL_TUNE_TARGET_LOW_UTIL", 0.40)
	dailyNotionalTuneTargetHigh := envFloat("AGENT_DAILY_NOTIONAL_TUNE_TARGET_HIGH_UTIL", 0.85)
	dailyNotionalTuneUpStepPct := envFloat("AGENT_DAILY_NOTIONAL_TUNE_UP_STEP_PCT", 10)
	dailyNotionalTuneDownStepPct := envFloat("AGENT_DAILY_NOTIONAL_TUNE_DOWN_STEP_PCT", 10)
	if dailyNotionalTuneMin < 10_000 {
		dailyNotionalTuneMin = 10_000
	}
	if dailyNotionalTuneMax < dailyNotionalTuneMin {
		dailyNotionalTuneMax = dailyNotionalTuneMin
	}
	if dailyNotionalTuneTargetLow < 0 {
		dailyNotionalTuneTargetLow = 0
	}
	if dailyNotionalTuneTargetLow > 0.90 {
		dailyNotionalTuneTargetLow = 0.90
	}
	if dailyNotionalTuneTargetHigh < dailyNotionalTuneTargetLow+0.05 {
		dailyNotionalTuneTargetHigh = dailyNotionalTuneTargetLow + 0.05
	}
	if dailyNotionalTuneTargetHigh > 0.99 {
		dailyNotionalTuneTargetHigh = 0.99
	}
	if dailyNotionalTuneUpStepPct < 1 {
		dailyNotionalTuneUpStepPct = 1
	}
	if dailyNotionalTuneUpStepPct > 50 {
		dailyNotionalTuneUpStepPct = 50
	}
	if dailyNotionalTuneDownStepPct < 1 {
		dailyNotionalTuneDownStepPct = 1
	}
	if dailyNotionalTuneDownStepPct > 50 {
		dailyNotionalTuneDownStepPct = 50
	}
	dailyNotionalTuneLastDay := ""
	dailyNotionalTunePeakUsed := 0.0
	dailyNotionalTuneTrackedLimit := 0.0
	blockNewBuysWhenAnyPending := envBool("AGENT_BLOCK_NEW_BUYS_WHEN_PENDING", true)
	pendingGlobalBlockMaxMin := envInt("AGENT_PENDING_GLOBAL_BLOCK_MAX_MIN", 45)
	if pendingGlobalBlockMaxMin < 0 {
		pendingGlobalBlockMaxMin = 0
	}
	if pendingGlobalBlockMaxMin > 12*60 {
		pendingGlobalBlockMaxMin = 12 * 60
	}
	pendingGlobalBlockMax := time.Duration(pendingGlobalBlockMaxMin) * time.Minute
	sessionPhaseOpenMin := envInt("AGENT_SESSION_PHASE_OPEN_MIN", 45)
	sessionPhaseCloseMin := envInt("AGENT_SESSION_PHASE_CLOSE_MIN", 45)
	learningWeight := envFloat("AGENT_LEARNING_CONF_WEIGHT", 0.12)
	affordabilityScoreWeight := envFloat("AGENT_AFFORDABILITY_SCORE_WEIGHT", 0.22)
	if affordabilityScoreWeight < 0 {
		affordabilityScoreWeight = 0
	}
	if affordabilityScoreWeight > 0.60 {
		affordabilityScoreWeight = 0.60
	}
	affordabilityMinExecRatio := envFloat("AGENT_AFFORDABILITY_MIN_EXEC_RATIO", 0.20)
	if affordabilityMinExecRatio < 0 {
		affordabilityMinExecRatio = 0
	}
	if affordabilityMinExecRatio > 1 {
		affordabilityMinExecRatio = 1
	}
	// Вклад "ожидаемой прибыли" в ранжирование buy-кандидатов:
	// edge(conf-gate) * weight - round_trip_cost * scale.
	expectedProfitWeight := envFloat("AGENT_EXPECTED_PROFIT_WEIGHT", 0.45)
	if expectedProfitWeight < 0 {
		expectedProfitWeight = 0
	}
	if expectedProfitWeight > 1.5 {
		expectedProfitWeight = 1.5
	}
	expectedProfitCostScale := envFloat("AGENT_EXPECTED_PROFIT_COST_SCALE", 12.0)
	if expectedProfitCostScale < 0 {
		expectedProfitCostScale = 0
	}
	if expectedProfitCostScale > 80 {
		expectedProfitCostScale = 80
	}
	futuresRoundTripCostPct := envFloat("AGENT_FUTURES_ROUND_TRIP_COST_PCT", 0.0009)
	if futuresRoundTripCostPct < 0 {
		futuresRoundTripCostPct = 0
	}
	stockRoundTripCostPct := services.FinamN3MOEXStockRoundTripPct()
	minTradesPerDay := envInt("AGENT_MIN_NUMTRADES", 60)
	minTurnover := envFloat("AGENT_MIN_TURNOVER_RUB", 1_000_000)
	maxHoldMin := envInt("AGENT_POSITION_MAX_HOLD_MIN", 180)
	if maxHoldMin < 15 {
		maxHoldMin = 15
	}
	staleBand := envFloat("AGENT_POSITION_STALE_BAND_PCT", 0.25) / 100.0
	if staleBand <= 0 {
		staleBand = 0.0025
	}
	lossStreakLimit := envInt("AGENT_LOSS_STREAK_LIMIT", 3)
	if lossStreakLimit < 1 {
		lossStreakLimit = 1
	}
	lossPauseMin := envInt("AGENT_LOSS_STREAK_PAUSE_MIN", 20)
	if lossPauseMin < 1 {
		lossPauseMin = 1
	}
	lossPause := time.Duration(lossPauseMin) * time.Minute
	fastMoveEnabled := strings.TrimSpace(strings.ToLower(os.Getenv("AGENT_FASTMOVE_ENABLED")))
	isFastMoveEnabled := fastMoveEnabled == "1" || fastMoveEnabled == "true" || fastMoveEnabled == "yes" || fastMoveEnabled == "on"
	fastMoveTriggerPct := envFloat("AGENT_FASTMOVE_TRIGGER_PCT", 1.4) / 100.0
	if fastMoveTriggerPct <= 0 {
		fastMoveTriggerPct = 0.014
	}
	fastMoveWindowSec := envInt("AGENT_FASTMOVE_WINDOW_SEC", 180)
	if fastMoveWindowSec < 30 {
		fastMoveWindowSec = 30
	}
	fastMoveWindow := time.Duration(fastMoveWindowSec) * time.Second
	minBalanceStock := envFloat("AGENT_MIN_BALANCE_STOCK", 200.0)
	minBalanceFutures := envFloat("AGENT_MIN_BALANCE_FUTURES", 100.0)
	if minBalanceStock < 0 {
		minBalanceStock = 0
	}
	if minBalanceFutures < 0 {
		minBalanceFutures = 0
	}
	if abTestEnabled {
		a.addRuntimeEvent("info", "auto.abtest_enabled", fmt.Sprintf("slot_min=%d modes=A/B/C", abTestSlotMin))
	}

	for {
		select {
		case <-ctx.Done():
			a.addRuntimeEvent("info", "auto.loop_exit", "context cancelled")
			return
		case <-ticker.C:
			if a.mode != engine.ModeAuto {
				a.addRuntimeEvent("debug", "auto.skip_mode", fmt.Sprintf("mode=%s", a.mode))
				continue
			}
			if a.agent == nil || !a.agent.IsActive() {
				a.addRuntimeEvent("debug", "auto.skip_agent", "agent inactive")
				continue
			}
			if a.broker == nil {
				a.addRuntimeEvent("warn", "auto.skip_broker", "broker is nil")
				continue
			}
			a.refreshAgentFromJournal()

			bal, _ := a.broker.GetBalance(context.Background())
			stockOpen, stockReason := a.isTradingAllowedForTicker("SBER")
			futuresOpen, futuresReason := a.isTradingAllowedForTicker("Si6")
			phase := sessionPhaseMoscow(stockOpen, futuresOpen, sessionPhaseOpenMin, sessionPhaseCloseMin)
			miyaSig := miyaOnlineSignal{
				State:            "no_data",
				DataStatus:       "no_data",
				DefaultMul:       1.00,
				UpRatioPct:       0,
				TickersLongCount: 0,
				DailyState:       "no_data",
				DailyDataStatus:  "no_data",
				DailyDefaultMul:  1.00,
				DailyUpRatioPct:  0,
				LiquidityStatus:  "no_data",
			}
			miyaBuyGateAdjust := 0.0
			miyaShortGateAdjust := 0.0
			if miyaOnlineHintsEnabled || miyaLiqRankEnabled || miyaWatchFirst {
				miyaSig = a.getMiyaOnlineSignal(time.Now())
				if miyaOnlineHintsEnabled {
					buyI, shortI := miyaOnlineGateDeltas(miyaSig.State, miyaSig.DefaultMul, miyaOnlineBuyGateMaxDelta, miyaOnlineShortGateMaxDelta)
					buyD, shortD := miyaOnlineGateDeltas(miyaSig.DailyState, miyaSig.DailyDefaultMul, miyaOnlineBuyGateMaxDelta, miyaOnlineShortGateMaxDelta)
					dailyW := envFloat("AGENT_MIA_ONLINE_DAILY_GATE_WEIGHT", 0.35)
					if dailyW < 0 {
						dailyW = 0
					}
					if dailyW > 1 {
						dailyW = 1
					}
					if miyaSig.DailyDataStatus == "no_data" || miyaSig.DailyDataStatus == "disabled" || normalizeMiyaOnlineState(miyaSig.DailyState) == "no_data" {
						dailyW = 0
					}
					// Вне основной сессии акций (SBER != T) intraday-слой Мии часто шумит; дневной (daily) оставляем.
					if !stockOpen {
						scale := envFloat("AGENT_MIA_ONLINE_STOCK_CLOSED_INTRADAY_SCALE", 0.35)
						if scale < 0 {
							scale = 0
						}
						if scale > 1 {
							scale = 1
						}
						buyI *= scale
						shortI *= scale
					}
					miyaBuyGateAdjust = (1.0-dailyW)*buyI + dailyW*buyD
					miyaShortGateAdjust = (1.0-dailyW)*shortI + dailyW*shortD
				}
			}
			miyaLiqRank := map[string]int{}
			miyaLiqN := len(miyaSig.LiquidityOrder)
			for i, t := range miyaSig.LiquidityOrder {
				k := canonicalTickerKey(t)
				if k != "" {
					miyaLiqRank[k] = i + 1
				}
			}
			// Рекомендации Мии: тикер → score (0-100)
			miyaRecScores := map[string]float64{}
			for _, rec := range miyaSig.Recommendations {
				k := canonicalTickerKey(rec.Ticker)
				if k != "" {
					miyaRecScores[k] = rec.Score
				}
			}
			miyaWatchSet := map[string]struct{}{}
			for _, t := range miyaSig.WatchTickers {
				k := canonicalTickerKey(t)
				if k != "" {
					miyaWatchSet[k] = struct{}{}
				}
			}
			a.sessionPhaseMu.Lock()
			if phase != a.lastSessionPhase {
				a.addRuntimeEvent("info", "auto.session_phase", fmt.Sprintf("phase=%s stock=%v futures=%v", phase, stockOpen, futuresOpen))
				a.lastSessionPhase = phase
			}
			a.sessionPhaseMu.Unlock()
			a.addRuntimeEvent(
				"debug",
				"auto.session_tick",
				fmt.Sprintf(
					"phase=%s stock=%v futures=%v miya_state=%s data_status=%s up_ratio=%.2f tickers=%d daily_state=%s daily_data=%s daily_up=%.2f",
					phase, stockOpen, futuresOpen, miyaSig.State, miyaSig.DataStatus, miyaSig.UpRatioPct, miyaSig.TickersLongCount,
					miyaSig.DailyState, miyaSig.DailyDataStatus, miyaSig.DailyUpRatioPct,
				),
			)
			if dailyNotionalTuneEnabled && a.risk != nil {
				riskNow := a.risk.Status(time.Now())
				if dailyNotionalTuneLastDay == "" {
					dailyNotionalTuneLastDay = riskNow.Day
					dailyNotionalTunePeakUsed = riskNow.DailyNotional
					dailyNotionalTuneTrackedLimit = riskNow.MaxDailyNotional
				} else if riskNow.Day == dailyNotionalTuneLastDay {
					if riskNow.DailyNotional > dailyNotionalTunePeakUsed {
						dailyNotionalTunePeakUsed = riskNow.DailyNotional
					}
					if riskNow.MaxDailyNotional > 0 {
						dailyNotionalTuneTrackedLimit = riskNow.MaxDailyNotional
					}
				} else {
					prevDay := dailyNotionalTuneLastDay
					prevUsed := dailyNotionalTunePeakUsed
					prevLimit := dailyNotionalTuneTrackedLimit
					if prevLimit <= 0 {
						prevLimit = riskNow.MaxDailyNotional
					}
					nextLimit, utilPct, reason := tuneDailyNotionalLimit(
						prevUsed,
						prevLimit,
						dailyNotionalTuneMin,
						dailyNotionalTuneMax,
						dailyNotionalTuneTargetLow,
						dailyNotionalTuneTargetHigh,
						dailyNotionalTuneUpStepPct,
						dailyNotionalTuneDownStepPct,
					)
					mode := "advisory"
					if dailyNotionalTuneApply {
						mode = "apply"
					}
					a.addRuntimeEvent(
						"info",
						"risk.daily_notional_tune_decision",
						fmt.Sprintf(
							"day=%s used=%.2f limit=%.2f util=%.1f%% next=%.2f mode=%s reason=%s",
							prevDay, prevUsed, prevLimit, utilPct, nextLimit, mode, reason,
						),
					)
					if dailyNotionalTuneApply && prevLimit > 0 && nextLimit > 0 && int(nextLimit+0.5) != int(prevLimit+0.5) {
						maxOrderLimit := riskNow.MaxOrderNotional
						if maxOrderLimit <= 0 {
							maxOrderLimit = envFloat("AGENT_MAX_ORDER_NOTIONAL", 50_000)
						}
						if a.SetRiskLimits(maxOrderLimit, nextLimit) {
							dailyNotionalTuneTrackedLimit = nextLimit
							a.addRuntimeEvent(
								"warn",
								"risk.daily_notional_tuned",
								fmt.Sprintf("day=%s old=%.2f new=%.2f util=%.1f%%", prevDay, prevLimit, nextLimit, utilPct),
							)
						}
					}
					dailyNotionalTuneLastDay = riskNow.Day
					dailyNotionalTunePeakUsed = riskNow.DailyNotional
					if riskNow.MaxDailyNotional > 0 {
						dailyNotionalTuneTrackedLimit = riskNow.MaxDailyNotional
					}
				}
			}
			if !stockOpen && !futuresOpen {
				closedSnapMin := envInt("AGENT_BROKER_POSITION_SNAPSHOT_CLOSED_MIN_SEC", 300)
				if closedSnapMin < 30 {
					closedSnapMin = 30
				}
				if closedSnapMin > 3600 {
					closedSnapMin = 3600
				}
				a.maybeWriteBrokerPortfolioSnapshot(nil, time.Now(), closedSnapMin)
				a.addRuntimeEvent("debug", "auto.skip_session_closed", fmt.Sprintf("stock=%s; futures=%s", stockReason, futuresReason))
				continue
			}
			a.autoCfgMu.RLock()
			modeNow := strings.ToLower(strings.TrimSpace(a.autoInstrumentMode))
			fixedTickerNow := strings.ToUpper(strings.TrimSpace(a.autoFixedTicker))
			a.autoCfgMu.RUnlock()
			activeMinBalance := minBalanceStock
			switch modeNow {
			case "futures":
				activeMinBalance = minBalanceFutures
			case "stocks":
				activeMinBalance = minBalanceStock
			case "fixed":
				if isLikelyFuturesTickerLocal(fixedTickerNow) {
					activeMinBalance = minBalanceFutures
				} else {
					activeMinBalance = minBalanceStock
				}
			default:
				// В mixed-режиме ориентируемся на меньший порог из доступных классов,
				// чтобы не отключать весь цикл, если хотя бы один класс торгуем.
				activeMinBalance = 0
				if stockOpen {
					activeMinBalance = minBalanceStock
				}
				if futuresOpen {
					if activeMinBalance <= 0 || minBalanceFutures < activeMinBalance {
						activeMinBalance = minBalanceFutures
					}
				}
				if activeMinBalance <= 0 {
					activeMinBalance = minBalanceStock
				}
			}
			lowBalance := bal > 0 && bal < activeMinBalance
			if lowBalance {
				a.addRuntimeEvent("warn", "auto.skip_low_balance", fmt.Sprintf("balance=%.2f threshold=%.2f mode=%s", bal, activeMinBalance, modeNow))
			}
			autoBudgetBal := bal * (float64(a.getAutoBudgetPercent()) / 100.0)
			if autoBudgetBal <= 0 {
				autoBudgetBal = bal
			}

			// Снимок портфеля, чтобы не покупать бесконечно (один запрос к брокеру на цикл).
			portfolio := a.GetPortfolio()
			now := time.Now()
			a.maybeWriteBrokerPortfolioSnapshot(portfolio, now, 0)
			type posState struct {
				Qty     int
				Avg     float64
				Current float64
			}
			posMap := map[string]posState{}
			for _, p := range portfolio {
				posMap[p.Ticker] = posState{
					Qty:     p.Quantity,
					Avg:     p.AvgPrice,
					Current: p.CurrentPrice,
				}
			}
			openPos := make(map[string]struct{}, len(posMap))
			for t, p := range posMap {
				if p.Qty != 0 {
					openPos[strings.ToUpper(strings.TrimSpace(t))] = struct{}{}
				}
			}
			a.trackOpenPositions(openPos, now)
			a.cleanupProfitLock(openPos)
			a.cleanupTrailingStop(openPos)
			entriesPaused, streak, pauseLeft := a.getAutoLossPause(lossStreakLimit)
			if entriesPaused {
				a.addRuntimeEvent("warn", "auto.pause_loss_streak", fmt.Sprintf("streak=%d pause_left=%s", streak, pauseLeft.Round(time.Second)))
			}
			if lowBalance {
				entriesPaused = true
				a.addRuntimeEvent("warn", "auto.low_balance_manage_only", fmt.Sprintf("balance=%.2f threshold=%.2f", bal, activeMinBalance))
			}
			if a.getRiskDailyPauseLeft() > 0 {
				entriesPaused = true
			}

			universeLimit := envInt("AGENT_UNIVERSE_LIMIT_PER_CLASS", 260)
			if universeLimit < 50 {
				universeLimit = 50
			}
			if universeLimit > 400 {
				universeLimit = 400
			}
			universe := a.getAutoUniverseSnapshot(universeLimit)
			if phase == "open" && openLongBiasEnabled && openStocksOnly && modeNow != "futures" {
				filtered := make([]services.Security, 0, len(universe))
				removed := 0
				for _, sec := range universe {
					if isLikelyFuturesTickerLocal(sec.Ticker) {
						removed++
						continue
					}
					filtered = append(filtered, sec)
				}
				if removed > 0 {
					a.addRuntimeEvent("debug", "auto.open_stocks_only", fmt.Sprintf("removed_futures=%d kept=%d", removed, len(filtered)))
				}
				universe = filtered
			}
			a.addRuntimeEvent("debug", "auto.cycle", fmt.Sprintf("balance=%.2f universe=%d", bal, len(universe)))
			learn := a.getTickerLearningScores(300)
			pnlScores := a.getTickerPnLScores(500)
			pnlScoreWeight := envFloat("AGENT_PNL_SCORE_WEIGHT", 0.08)
			if pnlScoreWeight < 0 {
				pnlScoreWeight = 0
			}
			if pnlScoreWeight > 0.35 {
				pnlScoreWeight = 0.35
			}
			rejectStats := a.getTickerRejectStats(3000)
			priceByTicker := map[string]float64{}
			for _, s := range universe {
				if s.Price > 0 {
					priceByTicker[s.Ticker] = s.Price
					if base := canonicalTickerKey(s.Ticker); base != "" {
						priceByTicker[base] = s.Price
					}
				}
			}

			// Предохранители, чтобы не “разнести” реальный счёт:
			// максимум 1 сделка за цикл, и только если уверенность высокая.
			maxOrdersThisCycle := 1
			placed := 0
			currentNoOrderStreak := a.getNoOrderStreak()
			buyConfGate, sellConfGate := a.getAdaptiveConfThresholds(buyConfThreshold, sellConfThreshold, currentNoOrderStreak)
			mirrorShortGateRaw := strings.TrimSpace(strings.ToLower(os.Getenv("AGENT_MIRROR_SHORT_TO_BUY_GATE")))
			mirrorShortGate := mirrorShortGateRaw == "" ||
				mirrorShortGateRaw == "1" ||
				mirrorShortGateRaw == "true" ||
				mirrorShortGateRaw == "yes" ||
				mirrorShortGateRaw == "on"
			shortOpenConfGate := shortConfThreshold
			if mirrorShortGate {
				shortOpenConfGate = buyConfGate
			}
			triggerCfg, trailCfg := a.getAutoProfitLockPercent()
			exitProfile := a.resolveExitProfile(
				tpPct,
				slPct,
				triggerCfg/100.0,
				trailCfg/100.0,
				staleBand,
				maxHoldMin,
				fastMoveTriggerPct,
				phase,
				lowBalance,
				streak,
			)
			a.exitProfileMu.Lock()
			exitProfileChanged := a.lastExitProfile != exitProfile.Name
			a.lastExitProfile = exitProfile.Name
			a.exitProfileMu.Unlock()
			if exitProfileChanged {
				a.addRuntimeEvent("info", "auto.exit_profile_switched", fmt.Sprintf("profile=%s reason=%s phase=%s tp=%.2f%% sl=%.2f%% hold=%dmin", exitProfile.Name, exitProfile.Reason, phase, exitProfile.TPPct*100, exitProfile.SLPct*100, exitProfile.MaxHoldMin))
			}

			// Этап 3: работаем с уже открытыми позициями (лонг/шорт) по TP/SL.
			bestExitTicker := ""
			bestExitQty := 0
			bestExitIsBuy := false
			bestExitScore := -1.0
			bestExitPrice := 0.0
			bestExitReason := ""
			bestExitPnlPct := 0.0
			bestExitTpMulLearn := 1.0
			bestExitTpMulHint := 1.0
			bestExitTpMulFinal := 1.0
			triggerPct := exitProfile.TriggerPct
			trailPct := exitProfile.TrailPct
			trailingStopEnabled := envBool("AGENT_TRAILING_STOP_ENABLED", true)
			trailActivatePct := envFloat("AGENT_TRAILING_ACTIVATE_PCT", 0.5) / 100.0
			trailStepPct := envFloat("AGENT_TRAILING_STEP_PCT", 0.35) / 100.0
			if trailStepPct < 0.003 {
				trailStepPct = 0.003
			}
			if trailStepPct > 0.005 {
				trailStepPct = 0.005
			}
			if trailActivatePct < 0.0005 {
				trailActivatePct = 0.0005
			}
			for t, p := range posMap {
				if p.Qty == 0 || p.Avg <= 0 {
					continue
				}
				// Если по конкретному тикеру сейчас сессия закрыта, не формируем
				// попытку выхода/ротации в этом цикле (избегаем лишних rejected).
				if ok, _ := a.isTradingAllowedForTicker(t); !ok {
					continue
				}
				cur := p.Current
				if cur <= 0 {
					cur = priceByTicker[t]
				}
				if cur <= 0 {
					cur = priceByTicker[canonicalTickerKey(t)]
				}
				if cur <= 0 {
					continue
				}
				heldFor := a.positionHeldFor(t, now)
				pnlPct := 0.0
				if p.Qty > 0 {
					pnlPct = (cur - p.Avg) / p.Avg
				} else {
					pnlPct = (p.Avg - cur) / p.Avg
				}
				tt := strings.ToUpper(strings.TrimSpace(t))
				tpMulLearn, tpMulHint, tpMulFinal := a.getTickerExitTPMultiplier(t, now)
				tpTarget := exitProfile.TPPct * tpMulFinal
				profitLockArmedNow := false
				profitLockExit := false
				if triggerPct > 0 && trailPct > 0 {
					a.profitLockMu.Lock()
					peak, armed := a.profitLockPeakPct[tt]
					if !armed {
						if pnlPct >= triggerPct {
							a.profitLockPeakPct[tt] = pnlPct
							profitLockArmedNow = true
						}
					} else {
						if pnlPct > peak {
							peak = pnlPct
							a.profitLockPeakPct[tt] = peak
						}
						if pnlPct <= peak-trailPct {
							profitLockExit = true
						}
					}
					a.profitLockMu.Unlock()
					if profitLockArmedNow {
						a.addRuntimeEvent("info", "auto.profit_lock_armed", fmt.Sprintf("%s pnl=%.2f%% trigger=%.2f%% trail=%.2f%%", t, pnlPct*100, triggerPct*100, trailPct*100))
					}
				}
				trailingExit := false
				stockNotFutures := !isLikelyFuturesTickerLocal(t)
				if trailingStopEnabled && stockNotFutures {
					slDec := exitProfile.SLPct
					initLong := p.Avg * (1.0 - slDec)
					initShort := p.Avg * (1.0 + slDec)
					a.trailingStopMu.Lock()
					line, hasLine := a.trailingStopPrice[tt]
					if p.Qty > 0 {
						if !hasLine && pnlPct >= trailActivatePct {
							line = initLong
							if cur*(1.0-trailStepPct) > line {
								line = cur * (1.0 - trailStepPct)
							}
							a.trailingStopPrice[tt] = line
							hasLine = true
							a.addRuntimeEvent("info", "auto.trailing_armed", fmt.Sprintf("%s long init_stop=%.4f trail=%.4f pnl=%.2f%% step=%.3f%%", t, initLong, line, pnlPct*100, trailStepPct*100))
						} else if hasLine {
							if cur-line >= trailStepPct*cur {
								cand := cur * (1.0 - trailStepPct)
								if cand < initLong {
									cand = initLong
								}
								if cand > line+1e-9 {
									a.addRuntimeEvent("info", "auto.trailing_ratchet", fmt.Sprintf("%s long %.4f -> %.4f", t, line, cand))
									line = cand
									a.trailingStopPrice[tt] = line
								}
							}
						}
						if hasLine && cur <= line {
							trailingExit = true
						}
					} else {
						if !hasLine && pnlPct >= trailActivatePct {
							line = initShort
							c := cur * (1.0 + trailStepPct)
							if c < line {
								line = c
							}
							a.trailingStopPrice[tt] = line
							hasLine = true
							a.addRuntimeEvent("info", "auto.trailing_armed", fmt.Sprintf("%s short init_stop=%.4f trail=%.4f pnl=%.2f%% step=%.3f%%", t, initShort, line, pnlPct*100, trailStepPct*100))
						} else if hasLine {
							if line-cur >= trailStepPct*cur {
								cand := cur * (1.0 + trailStepPct)
								if cand > initShort {
									cand = initShort
								}
								if cand < line-1e-9 {
									a.addRuntimeEvent("info", "auto.trailing_ratchet", fmt.Sprintf("%s short %.4f -> %.4f", t, line, cand))
									line = cand
									a.trailingStopPrice[tt] = line
								}
							}
						}
						if hasLine && cur >= line {
							trailingExit = true
						}
					}
					a.trailingStopMu.Unlock()
				}
				staleExit := heldFor >= time.Duration(exitProfile.MaxHoldMin)*time.Minute && pnlPct <= exitProfile.StaleBand
				fastMoveExit := false
				if isFastMoveEnabled && heldFor > 0 && heldFor <= fastMoveWindow && pnlPct >= exitProfile.FastMoveTriggerPct {
					fastMoveExit = true
				}
				needExit := pnlPct >= tpTarget || trailingExit || pnlPct <= -exitProfile.SLPct || profitLockExit || staleExit || fastMoveExit
				if !needExit {
					continue
				}
				score := 1.0 + absFloat(pnlPct)
				if pnlPct <= -exitProfile.SLPct {
					score = 2.0 + absFloat(pnlPct) // стоп-выход приоритетнее профит-фиксации
				}
				reason := "tp"
				switch {
				case pnlPct >= tpTarget:
					reason = "tp"
				case trailingExit:
					reason = "trailing_stop"
					score = maxFloat(score, 2.35+absFloat(pnlPct))
					a.addRuntimeEvent("info", "auto.trailing_exit", fmt.Sprintf("%s pnl=%.2f%%", t, pnlPct*100))
				case pnlPct <= -exitProfile.SLPct:
					reason = "sl"
				case profitLockExit:
					score = maxFloat(score, 2.4+absFloat(pnlPct))
					reason = "profit_lock"
					a.addRuntimeEvent("info", "auto.profit_lock_exit", fmt.Sprintf("%s pnl=%.2f%%", t, pnlPct*100))
				case staleExit:
					score = maxFloat(score, 1.3+absFloat(pnlPct))
					reason = "stale_timeout"
					a.addRuntimeEvent("info", "auto.exit_stale_timeout", fmt.Sprintf("%s held=%s pnl=%.2f%%", t, heldFor.Round(time.Minute), pnlPct*100))
				case fastMoveExit:
					score = maxFloat(score, 2.2+absFloat(pnlPct))
					reason = "fast_move_guard"
					a.addRuntimeEvent("warn", "auto.fast_move_guard", fmt.Sprintf("%s held=%s pnl=%.2f%% trigger=%.2f%%", t, heldFor.Round(time.Second), pnlPct*100, exitProfile.FastMoveTriggerPct*100))
				default:
					reason = "tp"
				}
				if tpMulFinal > 1.04 || tpMulFinal < 0.96 {
					a.addRuntimeEvent("debug", "auto.exit_tp_tuned", fmt.Sprintf("%s base_tp=%.2f%% tuned_tp=%.2f%% mul=%.3f learn=%.3f hint=%.3f", t, exitProfile.TPPct*100, tpTarget*100, tpMulFinal, tpMulLearn, tpMulHint))
				}
				if score > bestExitScore {
					bestExitScore = score
					bestExitTicker = t
					bestExitQty = intAbs(p.Qty)
					bestExitIsBuy = p.Qty < 0 // закрытие шорта = buy
					bestExitPrice = cur
					bestExitReason = reason
					bestExitPnlPct = pnlPct
					bestExitTpMulLearn = tpMulLearn
					bestExitTpMulHint = tpMulHint
					bestExitTpMulFinal = tpMulFinal
				}
			}

			bestBuy := ""
			bestBuyConf := 0.0
			type buyCandidate struct {
				Ticker    string
				Qty       int
				Conf      float64
				RankScore float64
				ExecRatio float64
				Price     float64
				IsFutures bool
				Mode      string
				GateShift float64
			}
			buyCandidatesLimit := envInt("AGENT_BUY_CANDIDATES_LIMIT", 12)
			if buyCandidatesLimit < 4 {
				buyCandidatesLimit = 4
			}
			if buyCandidatesLimit > 30 {
				buyCandidatesLimit = 30
			}
			buyCandidates := make([]buyCandidate, 0, buyCandidatesLimit)
			buyCandidatesFB := make([]buyCandidate, 0, buyCandidatesLimit)
			buyCandidatesRejectedAffordable := 0
			addBuyCandidateTo := func(slice *[]buyCandidate, ticker string, qty int, conf float64, rankScore float64, execRatio float64, price float64, isFutures bool, mode string, gateShift float64) {
				if ticker == "" || qty <= 0 || price <= 0 {
					return
				}
				for i := range *slice {
					if strings.EqualFold((*slice)[i].Ticker, ticker) {
						if rankScore > (*slice)[i].RankScore {
							(*slice)[i] = buyCandidate{
								Ticker:    ticker,
								Qty:       qty,
								Conf:      conf,
								RankScore: rankScore,
								ExecRatio: execRatio,
								Price:     price,
								IsFutures: isFutures,
								Mode:      mode,
								GateShift: gateShift,
							}
						}
						goto SORTM
					}
				}
				*slice = append(*slice, buyCandidate{
					Ticker:    ticker,
					Qty:       qty,
					Conf:      conf,
					RankScore: rankScore,
					ExecRatio: execRatio,
					Price:     price,
					IsFutures: isFutures,
					Mode:      mode,
					GateShift: gateShift,
				})
			SORTM:
				sort.SliceStable(*slice, func(i, j int) bool {
					if (*slice)[i].RankScore == (*slice)[j].RankScore {
						return (*slice)[i].Conf > (*slice)[j].Conf
					}
					return (*slice)[i].RankScore > (*slice)[j].RankScore
				})
				if len(*slice) > buyCandidatesLimit {
					*slice = (*slice)[:buyCandidatesLimit]
				}
			}
			addBuyCandidate := func(ticker string, qty int, conf float64, rankScore float64, execRatio float64, price float64, isFutures bool, mode string, gateShift float64) {
				dest := &buyCandidates
				if miyaWatchFirst && len(miyaWatchSet) > 0 && !isFutures {
					if _, ok := miyaWatchSet[canonicalTickerKey(ticker)]; !ok {
						dest = &buyCandidatesFB
					}
				}
				addBuyCandidateTo(dest, ticker, qty, conf, rankScore, execRatio, price, isFutures, mode, gateShift)
			}

			bestSell := ""
			bestSellQty := 0
			bestSellConf := 0.0
			bestSellPrice := 0.0
			bestSellIsShortOpen := false
			bestSellProfile := abTestProfile{Mode: "BASE"}
			weakLongTicker := ""
			weakLongQty := 0
			weakLongConf := 2.0
			weakLongPrice := 0.0
			weakShortTicker := ""
			weakShortQty := 0
			weakShortConf := 2.0
			weakShortPrice := 0.0

			passedFilters := 0
			blockedByCooldown := 0
			blockedByQualification := 0
			blockedByRotation := 0
			blockedByCash := 0
			rejectedByLiquidity := 0
			rejectedByPrice := 0
			blockedBySession := 0
			blockedBySessionSample := ""
			sessionGateCache := make(map[string]bool)
			sessionGateReason := make(map[string]string)
			abModeSeen := map[string]int{"A": 0, "B": 0, "C": 0}
			liqTrades := minTradesPerDay
			liqTurnover := minTurnover
			for _, s := range universe {
				if isLikelyFuturesTickerLocal(s.Ticker) {
					if !futuresOpen {
						continue
					}
				} else {
					if !stockOpen {
						continue
					}
				}
				// Статический denylist режем как можно раньше,
				// чтобы не тратить цикл на анализ/ордера по заведомо запрещённым тикерам.
				if a.isTickerStaticallyDenied(s.Ticker) {
					continue
				}
				if strictTradingStatusGateEnabled() {
					key := canonicalTickerKey(s.Ticker)
					if key == "" {
						key = strings.ToUpper(strings.TrimSpace(s.Ticker))
					}
					allowed, checked := sessionGateCache[key]
					reason := sessionGateReason[key]
					if !checked {
						allowed, reason = a.isTradingAllowedForTicker(s.Ticker)
						sessionGateCache[key] = allowed
						sessionGateReason[key] = reason
					}
					if !allowed {
						blockedBySession++
						if blockedBySessionSample == "" && reason != "" {
							blockedBySessionSample = fmt.Sprintf("%s: %s", s.Ticker, reason)
						}
						continue
					}
				}
				if s.Price <= 0 {
					// Без валидной цены risk-manager корректно блокирует notional=0,
					// поэтому заранее отбрасываем такие инструменты.
					rejectedByPrice++
					continue
				}
				if _, reason, blocked := a.getBadTickerBlockInfo(s.Ticker); blocked {
					blockedByCooldown++
					if reason == "qualification" {
						blockedByQualification++
					}
					continue
				}
				if a.isRecentAutoTickerBlocked(s.Ticker) {
					blockedByRotation++
					continue
				}
				lot := a.moex.GetLotSize(s.Ticker)
				if lot < 1 {
					lot = 1
				}
				turnover := s.TurnoverRub
				if turnover <= 0 {
					turnover = float64(s.NumTrades) * s.Price * float64(lot)
				}
				if s.NumTrades > 0 && s.NumTrades < liqTrades {
					rejectedByLiquidity++
					continue
				}
				if turnover > 0 && turnover < liqTurnover {
					rejectedByLiquidity++
					continue
				}
				passedFilters++
				isFuturesTicker := isLikelyFuturesTickerLocal(s.Ticker)
				profile := getABTestProfile(now, s.Ticker, isFuturesTicker, abTestEnabled, abTestSlotMin)
				if profile.Mode != "BASE" {
					abModeSeen[profile.Mode]++
				}

				res := a.AnalyzeWithAgent(s.Ticker)
				action, _ := res["action"].(string)
				conf, _ := res["confidence"].(float64)
				// Лёгкая корректировка confidence по истории сделок тикера.
				lk := tickerLearningKey(s.Ticker)
				conf += learn[lk] * learningWeight
				baseTk := canonicalTickerKey(s.Ticker)
				if baseTk != "" && pnlScores[baseTk] != 0 {
					conf += pnlScores[baseTk] * pnlScoreWeight
				} else {
					conf += pnlScores[s.Ticker] * pnlScoreWeight
				}
				conf += a.rejectPenaltyScore(rejectStats[s.Ticker])
				conf += profile.ConfDelta
				// Бонус к conf за рекомендации Мии (топ-N тикеров)
				if miyaRecConfBonus > 0 {
					if recScore, ok := miyaRecScores[canonicalTickerKey(s.Ticker)]; ok && recScore > 0 {
						conf += recScore / 100.0 * miyaRecConfBonus
					}
				}
				if conf < 0 {
					conf = 0
				}
				if conf > 1 {
					conf = 1
				}
				qtyAny := res["quantity"]
				qty := 0
				switch v := qtyAny.(type) {
				case int:
					qty = v
				case float64:
					qty = int(v)
				default:
					qty = 0
				}
				if qty <= 0 {
					continue
				}
				if qty < lot {
					qty = lot
				}
				if have := posMap[s.Ticker].Qty; have > 0 {
					// Умное анти-зацикливание: мягкий штраф для набора позиции,
					// но если уже много куплено — усиливаем штраф
					haveValue := float64(have) * s.Price
					if haveValue > 5000 {
						conf -= 0.20 // большая позиция — снижаем приоритет
					} else {
						conf -= 0.05 // маленькая позиция — лёгкий штраф (набор разрешён)
					}
					if conf < 0 {
						conf = 0
					}
					if conf < weakLongConf {
						weakLongTicker = s.Ticker
						weakLongQty = have
						weakLongConf = conf
						weakLongPrice = s.Price
					}
				}
				if have := posMap[s.Ticker].Qty; have < 0 {
					shortConf := 0.5
					if action == "sell" {
						shortConf = conf
					} else if action == "buy" {
						shortConf = 1.0 - conf
					}
					if shortConf < weakShortConf {
						weakShortTicker = s.Ticker
						weakShortQty = intAbs(have)
						weakShortConf = shortConf
						weakShortPrice = s.Price
					}
				}

				switch action {
				case "buy":
					have := posMap[s.Ticker].Qty
					// Лонг: открываем из flat; при short — покупка на покрытие.
					if have > 0 {
						continue // long уже есть, новую не наращиваем в этой версии
					}
					if have < 0 {
						coverQty := intAbs(have)
						if qty > coverQty {
							qty = coverQty
						}
					}
					execQty := a.affordableQuantity(autoBudgetBal, s.Price, qty, isFuturesTicker, s.Ticker)
					execQty = a.floorToLot(s.Ticker, execQty)
					if execQty <= 0 {
						buyCandidatesRejectedAffordable++
						continue
					}
					execRatio := 0.0
					if qty > 0 {
						execRatio = float64(execQty) / float64(qty)
						if execRatio > 1 {
							execRatio = 1
						}
					}
					liqBonus := 0.0
					if miyaLiqRankEnabled && !isFuturesTicker {
						liqBonus = miyaLiquidityRankBonus(s.Ticker, miyaLiqRank, miyaLiqN, miyaSig.LiquidityStatus, miyaLiqRankWeight)
					}
					// Бонус за рекомендации Мии (топ-N тикеров с высоким score)
					recBonus := miyaRecommendationBonus(s.Ticker, miyaRecScores, miyaRecWeight)
					candGate := buyConfGate + profile.BuyGateOffset
					expectedEdge := clampFloat(conf-candGate, -0.25, 0.45)
					roundTripCostPct := stockRoundTripCostPct
					if isFuturesTicker {
						roundTripCostPct = futuresRoundTripCostPct
					}
					expectedProfitScore := expectedEdge*expectedProfitWeight - roundTripCostPct*expectedProfitCostScale
					rankScore := conf + execRatio*affordabilityScoreWeight + profile.RankScoreDelta + liqBonus + recBonus + expectedProfitScore
					// Бонус для фьючерсов: +0.10 к rankScore, чтобы они чаще попадали в топ-кандидатов
					if isFuturesTicker {
						rankScore += 0.10
					}
					if execRatio < affordabilityMinExecRatio {
						rankScore -= (affordabilityMinExecRatio - execRatio) * affordabilityScoreWeight
					}
					addBuyCandidate(s.Ticker, execQty, conf, rankScore, execRatio, s.Price, isFuturesTicker, profile.Mode, profile.BuyGateOffset)
				case "sell":
					have := posMap[s.Ticker].Qty
					openShort := false
					if have > 0 {
						if qty > have {
							qty = have
						}
					} else if have == 0 {
						// Зеркальный вход: для открытия шорта используем тот же gate,
						// что и для long-entry (если mirror включен).
						if conf < shortOpenConfGate+profile.ShortGateOffset {
							continue
						}
						openShort = true
					} else {
						// уже в шорте, не усиливаем позицию в этой версии
						continue
					}
					if conf > bestSellConf {
						bestSell = s.Ticker
						bestSellQty = qty
						bestSellConf = conf
						bestSellPrice = s.Price
						bestSellIsShortOpen = openShort
						bestSellProfile = profile
					}
				default:
					// hold
				}
			}

			// Если фильтры оказались слишком жёсткими, ослабляем ликвидность и делаем второй проход.
			if passedFilters < 12 && len(universe) > 0 {
				liqTrades = maxInt(20, liqTrades/2)
				liqTurnover = maxFloat(300000, liqTurnover/4)
				a.addRuntimeEvent("warn", "auto.filter_relaxed", fmt.Sprintf("passed=%d trades>=%d turnover>=%.0f", passedFilters, liqTrades, liqTurnover))
				for _, s := range universe {
					if a.isTickerStaticallyDenied(s.Ticker) {
						continue
					}
					if s.Price <= 0 {
						continue
					}
					if _, _, blocked := a.getBadTickerBlockInfo(s.Ticker); blocked {
						continue
					}
					if a.isRecentAutoTickerBlocked(s.Ticker) {
						continue
					}
					lot := a.moex.GetLotSize(s.Ticker)
					if lot < 1 {
						lot = 1
					}
					turnover := s.TurnoverRub
					if turnover <= 0 {
						turnover = float64(s.NumTrades) * s.Price * float64(lot)
					}
					if s.NumTrades > 0 && s.NumTrades < liqTrades {
						continue
					}
					if turnover > 0 && turnover < liqTurnover {
						continue
					}
					// Проверка экспирации для фьючерсов
					if isLikelyFuturesTickerLocal(s.Ticker) && s.ExpiryDate != nil {
						minDaysToExpiry := envInt("AGENT_FUTURES_MIN_DAYS_TO_EXPIRY", 7)
						if s.IsExpiringSoon(minDaysToExpiry) {
							a.addRuntimeEvent("info", "auto.skip_expiry_soon", fmt.Sprintf("%s expires in %d days (min=%d)", s.Ticker, int(time.Until(*s.ExpiryDate).Hours()/24), minDaysToExpiry))
							continue
						}
					}
					res := a.AnalyzeWithAgent(s.Ticker)
					action, _ := res["action"].(string)
					conf, _ := res["confidence"].(float64)
					isFuturesTicker := isLikelyFuturesTickerLocal(s.Ticker)
					profile := getABTestProfile(now, s.Ticker, isFuturesTicker, abTestEnabled, abTestSlotMin)
					if profile.Mode != "BASE" {
						abModeSeen[profile.Mode]++
					}
					lk := tickerLearningKey(s.Ticker)
					conf += learn[lk] * learningWeight
					baseTk := canonicalTickerKey(s.Ticker)
					if baseTk != "" && pnlScores[baseTk] != 0 {
						conf += pnlScores[baseTk] * pnlScoreWeight
					} else {
						conf += pnlScores[s.Ticker] * pnlScoreWeight
					}
					conf += a.rejectPenaltyScore(rejectStats[s.Ticker])
					conf += profile.ConfDelta
					if conf < 0 {
						conf = 0
					}
					if conf > 1 {
						conf = 1
					}
					qtyAny := res["quantity"]
					qty := 0
					switch v := qtyAny.(type) {
					case int:
						qty = v
					case float64:
						qty = int(v)
					}
					if qty <= 0 {
						continue
					}
					if qty < lot {
						qty = lot
					}
					if have := posMap[s.Ticker].Qty; have > 0 {
						if conf < weakLongConf {
							weakLongTicker = s.Ticker
							weakLongQty = have
							weakLongConf = conf
							weakLongPrice = s.Price
						}
					}
					if have := posMap[s.Ticker].Qty; have < 0 {
						shortConf := 0.5
						if action == "sell" {
							shortConf = conf
						} else if action == "buy" {
							shortConf = 1.0 - conf
						}
						if shortConf < weakShortConf {
							weakShortTicker = s.Ticker
							weakShortQty = intAbs(have)
							weakShortConf = shortConf
							weakShortPrice = s.Price
						}
					}
					switch action {
					case "buy":
						have := posMap[s.Ticker].Qty
						if have > 0 {
							continue
						}
						if have < 0 {
							coverQty := intAbs(have)
							if qty > coverQty {
								qty = coverQty
							}
						}
						execQty := a.affordableQuantity(autoBudgetBal, s.Price, qty, isFuturesTicker, s.Ticker)
						execQty = a.floorToLot(s.Ticker, execQty)
						if execQty <= 0 {
							buyCandidatesRejectedAffordable++
							continue
						}
						execRatio := 0.0
						if qty > 0 {
							execRatio = float64(execQty) / float64(qty)
							if execRatio > 1 {
								execRatio = 1
							}
						}
						liqBonus := 0.0
						if miyaLiqRankEnabled && !isFuturesTicker {
							liqBonus = miyaLiquidityRankBonus(s.Ticker, miyaLiqRank, miyaLiqN, miyaSig.LiquidityStatus, miyaLiqRankWeight)
						}
						recBonus := miyaRecommendationBonus(s.Ticker, miyaRecScores, miyaRecWeight)
						candGate := buyConfGate + profile.BuyGateOffset
						expectedEdge := clampFloat(conf-candGate, -0.25, 0.45)
						roundTripCostPct := stockRoundTripCostPct
						if isFuturesTicker {
							roundTripCostPct = futuresRoundTripCostPct
						}
						expectedProfitScore := expectedEdge*expectedProfitWeight - roundTripCostPct*expectedProfitCostScale
						rankScore := conf + execRatio*affordabilityScoreWeight + profile.RankScoreDelta + liqBonus + recBonus + expectedProfitScore
						// Бонус для фьючерсов: +0.10 к rankScore, чтобы они чаще попадали в топ-кандидатов
						if isFuturesTicker {
							rankScore += 0.10
						}
						if execRatio < affordabilityMinExecRatio {
							rankScore -= (affordabilityMinExecRatio - execRatio) * affordabilityScoreWeight
						}
						addBuyCandidate(s.Ticker, execQty, conf, rankScore, execRatio, s.Price, isFuturesTicker, profile.Mode, profile.BuyGateOffset)
					case "sell":
						have := posMap[s.Ticker].Qty
						openShort := false
						if have > 0 {
							if qty > have {
								qty = have
							}
						} else if have == 0 {
							if conf < shortOpenConfGate+profile.ShortGateOffset {
								continue
							}
							openShort = true
						} else {
							continue
						}
						if conf > bestSellConf {
							bestSell = s.Ticker
							bestSellQty = qty
							bestSellConf = conf
							bestSellPrice = s.Price
							bestSellIsShortOpen = openShort
							bestSellProfile = profile
						}
					}
				}
			}

			if miyaWatchFirst && len(miyaWatchSet) > 0 && len(buyCandidates) == 0 && len(buyCandidatesFB) > 0 {
				buyCandidates = buyCandidatesFB
				a.addRuntimeEvent("info", "auto.miya_watch_fallback", fmt.Sprintf("n=%d watch=%d", len(buyCandidates), len(miyaWatchSet)))
			}
			if len(buyCandidates) > 0 {
				bestBuy = buyCandidates[0].Ticker
				bestBuyConf = buyCandidates[0].Conf
			}

			// Приоритет №1: управляющий выход по TP/SL (и для long, и для short).
			if placed < maxOrdersThisCycle && bestExitTicker != "" && bestExitQty > 0 {
				placedNow := false
				if bestExitIsBuy {
					a.addRuntimeEvent("info", "auto.exit_buy_cover", fmt.Sprintf("%s qty=%d price=%.2f reason=%s", bestExitTicker, bestExitQty, bestExitPrice, bestExitReason))
					placedNow = a.BuyPosition(bestExitTicker, bestExitQty, bestExitPrice)
				} else {
					a.addRuntimeEvent("info", "auto.exit_sell_close", fmt.Sprintf("%s qty=%d price=%.2f reason=%s", bestExitTicker, bestExitQty, bestExitPrice, bestExitReason))
					placedNow = a.SellPosition(bestExitTicker, bestExitQty, bestExitPrice)
				}
				if placedNow {
					exitPnl := 0.0
					if bestExitReason == "sl" {
						exitPnl = -exitProfile.SLPct
					}
					a.markAutoExitOutcome(bestExitReason, exitPnl, lossStreakLimit, lossPause)
					a.rememberExitLearning(bestExitTicker, bestExitReason, phase, bestExitPnlPct, bestExitTpMulLearn, bestExitTpMulHint, bestExitTpMulFinal)
					placed++
				}
			}

			// Приоритет №2: сигнал на продажу (закрытие long / открытие short).
			rotateEnabled, rotateDelta, _ := a.getAutoRotateConfig()
			if entriesPaused {
				rotateEnabled = false
			}
			// Ротация long -> long: освобождаем кэш из слабого лонга под более сильный buy-кандидат.
			if placed < maxOrdersThisCycle &&
				rotateEnabled &&
				bestBuy != "" &&
				weakLongTicker != "" &&
				!strings.EqualFold(bestBuy, weakLongTicker) &&
				weakLongQty > 0 &&
				bestBuyConf-weakLongConf >= rotateDelta &&
				a.canAutoRotate(time.Now()) {
				sellPx := weakLongPrice
				if sellPx <= 0 {
					sellPx = priceByTicker[weakLongTicker]
				}
				if sellPx > 0 {
					a.addRuntimeEvent("info", "auto.rotate_long_to_long", fmt.Sprintf("sell=%s qty=%d conf=%.2f -> buy=%s conf=%.2f delta=%.2f", weakLongTicker, weakLongQty, weakLongConf, bestBuy, bestBuyConf, bestBuyConf-weakLongConf))
					if a.SellPosition(weakLongTicker, weakLongQty, sellPx) {
						a.markAutoRotate(time.Now())
						placed++
					}
				}
			}
			// Ротация long -> short: закрываем слабый лонг, если short-кандидат значительно сильнее.
			if placed < maxOrdersThisCycle &&
				rotateEnabled &&
				bestSell != "" &&
				bestSellIsShortOpen &&
				weakLongTicker != "" &&
				!strings.EqualFold(bestSell, weakLongTicker) &&
				weakLongQty > 0 &&
				bestSellConf-weakLongConf >= rotateDelta &&
				a.canAutoRotate(time.Now()) {
				sellPx := weakLongPrice
				if sellPx <= 0 {
					sellPx = priceByTicker[weakLongTicker]
				}
				if sellPx > 0 {
					a.addRuntimeEvent("info", "auto.rotate_long_to_short", fmt.Sprintf("sell=%s qty=%d conf=%.2f -> short=%s conf=%.2f delta=%.2f", weakLongTicker, weakLongQty, weakLongConf, bestSell, bestSellConf, bestSellConf-weakLongConf))
					if a.SellPosition(weakLongTicker, weakLongQty, sellPx) {
						a.markAutoRotate(time.Now())
						placed++
					}
				}
			}
			// Ротация short -> long: покрываем слабый шорт, если buy-кандидат заметно лучше.
			if placed < maxOrdersThisCycle &&
				rotateEnabled &&
				bestBuy != "" &&
				weakShortTicker != "" &&
				!strings.EqualFold(bestBuy, weakShortTicker) &&
				weakShortQty > 0 &&
				bestBuyConf-weakShortConf >= rotateDelta &&
				a.canAutoRotate(time.Now()) {
				buyPx := weakShortPrice
				if buyPx <= 0 {
					buyPx = priceByTicker[weakShortTicker]
				}
				if buyPx > 0 {
					a.addRuntimeEvent("info", "auto.rotate_short_to_long", fmt.Sprintf("cover=%s qty=%d conf=%.2f -> buy=%s conf=%.2f delta=%.2f", weakShortTicker, weakShortQty, weakShortConf, bestBuy, bestBuyConf, bestBuyConf-weakShortConf))
					if a.BuyPosition(weakShortTicker, weakShortQty, buyPx) {
						a.markAutoRotate(time.Now())
						placed++
					}
				}
			}
			sellActionGate := sellConfGate
			if bestSellIsShortOpen && mirrorShortGate {
				sellActionGate = buyConfGate
			}
			sellActionGate += bestSellProfile.SellGateOffset
			if bestSellIsShortOpen && mirrorShortGate {
				sellActionGate = buyConfGate + bestSellProfile.BuyGateOffset
			}
			if phase == "open" && openLongBiasEnabled && bestSellIsShortOpen {
				sellActionGate += openShortGatePenalty
				if sellActionGate > 0.95 {
					sellActionGate = 0.95
				}
			}
			if bestSellIsShortOpen && miyaShortGateAdjust != 0 {
				sellActionGate += miyaShortGateAdjust
				if sellActionGate > 0.95 {
					sellActionGate = 0.95
				}
				if sellActionGate < 0.45 {
					sellActionGate = 0.45
				}
			}
			if placed < maxOrdersThisCycle && bestSell != "" && bestSellConf >= sellActionGate {
				if ok, note := a.mtfConfirm(bestSell, "sell"); !ok {
					a.addRuntimeEvent("warn", "auto.skip_mtf_sell", fmt.Sprintf("%s %s", bestSell, note))
					a.rememberRecentAutoTicker(bestSell, 60*time.Second)
					bestSell = ""
				}
			}
			if placed < maxOrdersThisCycle && bestSell != "" && bestSellConf >= sellActionGate {
				step := "auto.order_sell_try"
				if bestSellIsShortOpen {
					if entriesPaused {
						a.addRuntimeEvent("warn", "auto.skip_short_loss_pause", fmt.Sprintf("%s conf=%.2f", bestSell, bestSellConf))
						bestSell = ""
					}
					step = "auto.order_short_open_try"
				}
				if bestSell != "" {
					if bestSellIsShortOpen {
						if !shortOpenEnabled {
							a.addRuntimeEvent("warn", "auto.skip_short_disabled", fmt.Sprintf("%s conf=%.2f", bestSell, bestSellConf))
							bestSell = ""
						}
					}
				}
				if bestSell != "" {
					if bestSellIsShortOpen {
						if shortTickerPauseLeft := a.getShortOpenTickerPauseLeft(bestSell); shortTickerPauseLeft > 0 {
							a.addRuntimeEvent("warn", "auto.skip_short_ticker_pause", fmt.Sprintf("%s pause_left=%s", bestSell, shortTickerPauseLeft.Round(time.Second)))
							bestSell = ""
						}
					}
				}
				if bestSell != "" {
					if bestSellIsShortOpen {
						if shortPauseLeft := a.getShortOpenPauseLeft(); shortPauseLeft > 0 {
							a.addRuntimeEvent("warn", "auto.skip_short_global_pause", fmt.Sprintf("%s pause_left=%s", bestSell, shortPauseLeft.Round(time.Second)))
							bestSell = ""
						}
					}
				}
				if bestSell != "" {
					a.addRuntimeEvent(
						"info",
						step,
						fmt.Sprintf(
							"%s qty=%d conf=%.2f gate=%.2f mode=%s price=%.2f miya_state=%s miya_short_adj=%.3f",
							bestSell, bestSellQty, bestSellConf, sellActionGate, bestSellProfile.Mode, bestSellPrice, miyaSig.State, miyaShortGateAdjust,
						),
					)
					sellPlaced := false
					if bestSellIsShortOpen {
						sellPlaced = a.OpenShortPosition(bestSell, bestSellQty, bestSellPrice)
					} else {
						sellPlaced = a.SellPosition(bestSell, bestSellQty, bestSellPrice)
					}
					if sellPlaced {
						a.rememberRecentAutoTicker(bestSell, 3*time.Minute)
						placed++
					}
				}
			}

			// Приоритет №3: покупка (открытие long / закрытие short).
			if placed < maxOrdersThisCycle && len(buyCandidates) > 0 {
				// Снимок pending-buy заявок на весь цикл: не дёргаем брокера на каждый кандидат.
				pendingBuy := a.pendingOrderTickerSet("buy")
				pendingLogged := make(map[string]struct{})
				pendingGateStale, pendingGateAge := a.isPendingBuyGateStale(pendingBuy, pendingGlobalBlockMax)
				if entriesPaused {
					top := buyCandidates[0]
					a.addRuntimeEvent("warn", "auto.skip_buy_loss_pause", fmt.Sprintf("%s conf=%.2f", top.Ticker, top.Conf))
				} else if cashPauseLeft := a.getBuyCashPauseLeft(); cashPauseLeft > 0 {
					blockedByCash++
					a.addRuntimeEvent("warn", "auto.skip_buy_cash_pause", fmt.Sprintf("pause_left=%s", cashPauseLeft.Round(time.Second)))
				} else if blockNewBuysWhenAnyPending && len(pendingBuy) > 0 && !pendingGateStale {
					blockedByCash++
					a.addRuntimeEvent("warn", "auto.skip_buy_global_pending", fmt.Sprintf("pending_buy_keys=%d", len(pendingBuy)))
				} else {
					if blockNewBuysWhenAnyPending && len(pendingBuy) > 0 && pendingGateStale {
						a.addRuntimeEvent(
							"warn",
							"auto.pending_global_stale_bypass",
							fmt.Sprintf("pending_buy_keys=%d age=%s limit=%s", len(pendingBuy), pendingGateAge.Round(time.Second), pendingGlobalBlockMax),
						)
					}
					buyAttemptsCap := maxBuyAttemptsPerCycle
					if phase == "open" && openLongBiasEnabled && openMaxBuyAttemptsBonus > 0 {
						buyAttemptsCap += openMaxBuyAttemptsBonus
						a.addRuntimeEvent("debug", "auto.open_long_bias", fmt.Sprintf("buy_gate_delta=%.2f short_gate_penalty=%.2f buy_attempts_cap=%d", openBuyGateDelta, openShortGatePenalty, buyAttemptsCap))
					}
					if miyaOnlineHintsEnabled && (miyaBuyGateAdjust != 0 || miyaShortGateAdjust != 0 || miyaSig.State == "no_data" || miyaSig.DailyState != "no_data") {
						dw := envFloat("AGENT_MIA_ONLINE_DAILY_GATE_WEIGHT", 0.35)
						if dw < 0 {
							dw = 0
						}
						if dw > 1 {
							dw = 1
						}
						a.addRuntimeEvent(
							"debug",
							"auto.miya_online_hint",
							fmt.Sprintf(
								"state=%s data_status=%s up_ratio=%.2f tickers=%d daily_state=%s daily_data=%s daily_up=%.2f daily_w=%.2f buy_adj=%.3f short_adj=%.3f",
								miyaSig.State, miyaSig.DataStatus, miyaSig.UpRatioPct, miyaSig.TickersLongCount,
								miyaSig.DailyState, miyaSig.DailyDataStatus, miyaSig.DailyUpRatioPct, dw, miyaBuyGateAdjust, miyaShortGateAdjust,
							),
						)
					}
					buyAttempts := 0
					for i, cand := range buyCandidates {
						if buyAttempts >= buyAttemptsCap {
							a.addRuntimeEvent("warn", "auto.stop_buy_attempts_cap", fmt.Sprintf("attempts=%d cap=%d", buyAttempts, buyAttemptsCap))
							break
						}
						candGate := buyConfGate + cand.GateShift
						if phase == "open" && openLongBiasEnabled && !cand.IsFutures {
							candGate -= openBuyGateDelta
						}
						haveQty := posMap[cand.Ticker].Qty
						if miyaBuyGateAdjust != 0 && !cand.IsFutures && haveQty >= 0 {
							candGate += miyaBuyGateAdjust
						}
						if candGate < 0.45 {
							candGate = 0.45
						}
						if candGate > 0.95 {
							candGate = 0.95
						}
						if cand.Conf < candGate {
							continue
						}
						if ok, note := a.mtfConfirm(cand.Ticker, "buy"); !ok {
							a.addRuntimeEvent("warn", "auto.skip_mtf_buy", fmt.Sprintf("%s %s", cand.Ticker, note))
							a.rememberRecentAutoTicker(cand.Ticker, 60*time.Second)
							blockedByRotation++
							continue
						}
						qty := cand.Qty
						if qty <= 0 {
							a.addRuntimeEvent("warn", "auto.skip_buy_affordable", fmt.Sprintf("%s requested_qty=%d", cand.Ticker, cand.Qty))
							blockedByCash++
							a.rememberRecentAutoTicker(cand.Ticker, 90*time.Second)
							continue
						}
						keyFull := strings.ToUpper(strings.TrimSpace(cand.Ticker))
						keyBase := canonicalTickerKey(cand.Ticker)
						pendingID, pending := pendingBuy[keyFull]
						if !pending && keyBase != "" {
							pendingID, pending = pendingBuy[keyBase]
						}
						if pending {
							msg := fmt.Sprintf("pending buy order exists for %s", cand.Ticker)
							if pendingID != "" {
								msg = fmt.Sprintf("%s (order_id=%s)", msg, pendingID)
							}
							if _, seen := pendingLogged[keyFull]; !seen {
								a.addRuntimeEvent("warn", "auto.skip_pending_order", msg)
								pendingLogged[keyFull] = struct{}{}
							}
							a.rememberRecentAutoTicker(cand.Ticker, 90*time.Second)
							continue
						}
						if i > 0 {
							a.addRuntimeEvent("info", "auto.buy_fallback_candidate", fmt.Sprintf("using #%d %s conf=%.2f", i+1, cand.Ticker, cand.Conf))
						}
						a.addRuntimeEvent(
							"info",
							"auto.order_buy_try",
							fmt.Sprintf(
								"%s qty=%d conf=%.2f gate=%.2f mode=%s price=%.2f phase=%s miya_state=%s miya_buy_adj=%.3f",
								cand.Ticker, qty, cand.Conf, candGate, cand.Mode, cand.Price, phase, miyaSig.State, miyaBuyGateAdjust,
							),
						)
						buyAttempts++
						buyPlaced := false
						if cand.IsFutures {
							buyPlaced = a.BuyFutures(cand.Ticker, qty, cand.Price)
						} else {
							buyPlaced = a.BuyPosition(cand.Ticker, qty, cand.Price)
						}
						if buyPlaced {
							a.rememberRecentAutoTicker(cand.Ticker, 3*time.Minute)
							placed++
							bestBuy = cand.Ticker
							bestBuyConf = cand.Conf
							break
						}
						a.addRuntimeEvent("warn", "auto.buy_candidate_failed", fmt.Sprintf("%s conf=%.2f qty=%d", cand.Ticker, cand.Conf, qty))
						if cashPauseLeft := a.getBuyCashPauseLeft(); cashPauseLeft > 0 {
							blockedByCash++
							a.addRuntimeEvent("warn", "auto.stop_buy_attempts_cash_pause", fmt.Sprintf("pause_left=%s", cashPauseLeft.Round(time.Second)))
							break
						}
					}
				}
			} else if placed < maxOrdersThisCycle && len(buyCandidates) == 0 && buyCandidatesRejectedAffordable > 0 {
				blockedByCash += buyCandidatesRejectedAffordable
				a.addRuntimeEvent(
					"info",
					"auto.skip_buy_no_affordable_after_scan",
					fmt.Sprintf("rejected=%d budget=%.2f", buyCandidatesRejectedAffordable, autoBudgetBal),
				)
			}
			if placed == 0 && !lowBalance {
				a.addRuntimeEvent("debug", "auto.no_order", fmt.Sprintf("phase=%s bestSell=%s(%.2f) bestBuy=%s(%.2f)", phase, bestSell, bestSellConf, bestBuy, bestBuyConf))
			}
			if !lowBalance {
				currentNoOrderStreak = a.updateNoOrderStreak(placed > 0)
				buyConfGate, sellConfGate = a.getAdaptiveConfThresholds(buyConfThreshold, sellConfThreshold, currentNoOrderStreak)
				if currentNoOrderStreak >= 6 && currentNoOrderStreak%3 == 0 {
					a.addRuntimeEvent("warn", "auto.adaptive_thresholds", fmt.Sprintf("no_order_streak=%d buy=%.2f sell=%.2f", currentNoOrderStreak, buyConfGate, sellConfGate))
				}
			}
			if abTestEnabled {
				a.addRuntimeEvent("debug", "auto.abtest_cycle", fmt.Sprintf("phase=%s A=%d B=%d C=%d", phase, abModeSeen["A"], abModeSeen["B"], abModeSeen["C"]))
			}
			if blockedBySession > 0 {
				msg := fmt.Sprintf("blocked=%d", blockedBySession)
				if blockedBySessionSample != "" {
					msg = fmt.Sprintf("%s sample=%s", msg, blockedBySessionSample)
				}
				a.addRuntimeEvent("debug", "auto.session_gate_prefilter", msg)
			}
			a.autoStatusMu.Lock()
			a.autoStatus = AutoDecisionStatus{
				At:                     a.nowInTradingTZ().Format(time.RFC3339),
				UniverseSize:           len(universe),
				PassedFilters:          passedFilters,
				BlockedByCooldown:      blockedByCooldown,
				BlockedByQualification: blockedByQualification,
				BlockedByRotation:      blockedByRotation,
				BlockedByCash:          blockedByCash,
				RejectedByLiquidity:    rejectedByLiquidity,
				RejectedByPrice:        rejectedByPrice,
				BestBuyTicker:          bestBuy,
				BestBuyConf:            fmt.Sprintf("%.2f", bestBuyConf),
				BestSellTicker:         bestSell,
				BestSellConf:           fmt.Sprintf("%.2f", bestSellConf),
				ActionTaken:            map[bool]string{true: "order_sent", false: "no_order"}[placed > 0],
				Note:                   fmt.Sprintf("Мухля оценивает ликвидность, лотность, историю ошибок, MTF-RSI/confidence; session_prefilter=%d", blockedBySession),
			}
			a.autoStatusMu.Unlock()
			if placed == 0 && blockedByCash > 0 {
				now := time.Now()
				if lastAutoBudgetBump.IsZero() || now.Sub(lastAutoBudgetBump) >= 10*time.Minute {
					currentBudget := a.getAutoBudgetPercent()
					if currentBudget < 100 {
						nextBudget := currentBudget + 10
						if nextBudget > 100 {
							nextBudget = 100
						}
						a.SetAutoBudgetPercent(nextBudget)
						lastAutoBudgetBump = now
						a.addRuntimeEvent("info", "auto.budget_autobump", fmt.Sprintf("blocked_by_cash=%d budget=%d->%d", blockedByCash, currentBudget, nextBudget))
					}
				}
			}
		}
	}
}

// startFuturesOverridesAutoUpdate запускает фоновое автообновление тикеров фьючерсов из MOEX ISS.
// Обновление происходит при старте и далее каждые 7 дней (или как настроено в env).
func (a *App) startFuturesOverridesAutoUpdate() {
	// Интервал обновления: по умолчанию 7 дней, можно настроить через env
	intervalHours := envInt("AGENT_FUTURES_OVERRIDES_UPDATE_INTERVAL_HOURS", 168) // 168h = 7 дней
	a.futuresOverridesUpdateInterval = time.Duration(intervalHours) * time.Hour
	
	// Проверяем, нужно ли обновление при старте
	if a.shouldUpdateFuturesOverrides() {
		go a.updateFuturesOverridesFromMOEX()
	}
	
	// Запускаем фоновый цикл проверки
	go func() {
		ticker := time.NewTicker(1 * time.Hour) // Проверяем раз в час
		defer ticker.Stop()
		for {
			select {
			case <-a.ctx.Done():
				return
			case <-ticker.C:
				if a.shouldUpdateFuturesOverrides() {
					a.updateFuturesOverridesFromMOEX()
				}
			}
		}
	}()
	
	a.addRuntimeEvent("info", "startup.futures_overrides_autoupdate", fmt.Sprintf("interval=%dh", intervalHours))
}

// shouldUpdateFuturesOverrides проверяет, нужно ли обновить тикеры фьючерсов
func (a *App) shouldUpdateFuturesOverrides() bool {
	a.futuresOverridesMu.Lock()
	defer a.futuresOverridesMu.Unlock()
	
	// Если никогда не обновляли — нужно обновить
	if a.futuresOverridesLastUpdate.IsZero() {
		return true
	}
	
	// Если прошло больше интервала — нужно обновить
	return time.Since(a.futuresOverridesLastUpdate) > a.futuresOverridesUpdateInterval
}

// updateFuturesOverridesFromMOEX получает свежие тикеры фьючерсов из MOEX ISS и обновляет overrides файл
func (a *App) updateFuturesOverridesFromMOEX() {
	a.futuresOverridesMu.Lock()
	if !a.futuresOverridesLastUpdate.IsZero() && time.Since(a.futuresOverridesLastUpdate) < a.futuresOverridesUpdateInterval {
		a.futuresOverridesMu.Unlock()
		return
	}
	a.futuresOverridesMu.Unlock()
	
	a.addRuntimeEvent("info", "futures_overrides.update_started", "получение свежих тикеров из MOEX ISS")
	
	// Получаем список фьючерсов из MOEX ISS
	futures := a.moex.GetFuturesInstruments("", 500)
	if len(futures) == 0 {
		a.addRuntimeEvent("warn", "futures_overrides.update_failed", "MOEX ISS вернул пустой список фьючерсов")
		return
	}
	
	// Читаем существующий overrides файл
	overridesPath := "finam_symbol_overrides_screener.json"
	existingData, err := os.ReadFile(overridesPath)
	if err != nil {
		a.addRuntimeEvent("warn", "futures_overrides.read_failed", fmt.Sprintf("не удалось прочитать %s: %v", overridesPath, err))
		return
	}
	
	var overrides map[string]string
	if err := json.Unmarshal(existingData, &overrides); err != nil {
		a.addRuntimeEvent("warn", "futures_overrides.parse_failed", fmt.Sprintf("не удалось распарсить %s: %v", overridesPath, err))
		return
	}
	
	// Обновляем маппинг тикеров фьючерсов
	updatedCount := 0
	for _, f := range futures {
		if f.Ticker == "" || f.Name == "" || f.Name == f.Ticker {
			continue
		}
		
		// Извлекаем базовый тикер из SHORTNAME (например, "MTSI-12.26" → "MTSI")
		baseTicker := extractBaseTickerFromName(f.Name)
		if baseTicker == "" {
			continue
		}
		
		// Формируем тикер для Finam API
		// Finam API использует формат TICKER@RTSX для фьючерсов (например, MTZ6@RTSX)
		finamTicker := f.Ticker + "@RTSX"
		
		// Обновляем маппинг для SECID (например, "MTZ6" → "MTSI-12.26@FUT")
		if overrides[f.Ticker] != finamTicker {
			overrides[f.Ticker] = finamTicker
			updatedCount++
		}
		
		// Также обновляем маппинг для базового тикера (например, "MTSI" → "MTSI-12.26@FUT")
		if overrides[baseTicker] != finamTicker {
			overrides[baseTicker] = finamTicker
			updatedCount++
		}
	}
	
	// Записываем обновлённый файл
	updatedData, err := json.MarshalIndent(overrides, "", "  ")
	if err != nil {
		a.addRuntimeEvent("warn", "futures_overrides.marshal_failed", fmt.Sprintf("не удалось сериализовать: %v", err))
		return
	}
	
	if err := os.WriteFile(overridesPath, updatedData, 0644); err != nil {
		a.addRuntimeEvent("warn", "futures_overrides.write_failed", fmt.Sprintf("не удалось записать %s: %v", overridesPath, err))
		return
	}
	
	// Обновляем кэш broker
	if a.broker != nil {
		a.broker.ReloadSymbolOverrides()
	}
	
	a.futuresOverridesMu.Lock()
	a.futuresOverridesLastUpdate = time.Now()
	a.futuresOverridesMu.Unlock()
	
	a.addRuntimeEvent("info", "futures_overrides.update_completed", fmt.Sprintf("обновлено %d тикеров из %d фьючерсов", updatedCount, len(futures)))
}

// extractBaseTickerFromName извлекает базовый тикер из SHORTNAME
// Например: "MTSI-12.26" → "MTSI", "Si-6.25" → "Si"
func extractBaseTickerFromName(name string) string {
	for i, ch := range name {
		if ch == '-' || ch == '.' {
			return strings.TrimSpace(name[:i])
		}
	}
	return strings.TrimSpace(name)
}

func (a *App) getUniverseSnapshot(limitPerClass int) []services.Security {
	a.universeMu.Lock()
	defer a.universeMu.Unlock()

	// кэш на 60 секунд, чтобы не долбить ISS и не грузить CPU.
	if time.Since(a.universeCachedAt) < 60*time.Second && len(a.universe) > 0 {
		return a.universe
	}

	stocks := a.moex.GetStockInstruments("", limitPerClass)
	futs := a.moex.GetFuturesInstruments("", limitPerClass)
	out := make([]services.Security, 0, len(stocks)+len(futs))
	out = append(out, stocks...)
	out = append(out, futs...)

	a.universe = a.applyStockAllowlist(out)
	a.universeCachedAt = time.Now()
	return a.universe
}

func (a *App) getAutoUniverseSnapshot(limitPerClass int) []services.Security {
	a.autoCfgMu.RLock()
	mode := a.autoInstrumentMode
	ticker := a.autoFixedTicker
	a.autoCfgMu.RUnlock()

	if mode == "fixed" && ticker != "" {
		price, _ := a.moex.GetCurrentPrice(ticker)
		return []services.Security{
			{
				Ticker: ticker,
				Name:   ticker,
				Price:  price,
			},
		}
	}

	// Daily Top-N стратегия: отбираем топ-N ликвидных инструментов
	dailyTopNEnabled := envBool("AGENT_DAILY_TOP_N_ENABLED", false)
	if dailyTopNEnabled {
		return a.getDailyTopN(limitPerClass)
	}

	switch mode {
	case "stocks":
		return a.applyStockAllowlist(a.moex.GetStockInstruments("", limitPerClass))
	case "futures":
		// Как и в смешанном universe: уважаем AGENT_FUTURES_ALLOWLIST_PATH (и auto-learn), иначе режим futures обходил фильтр.
		return a.applyStockAllowlist(a.moex.GetFuturesInstruments("", limitPerClass))
	case "options":
		// Каталог опционов будет добавлен на Phase 2.
		return []services.Security{}
	default:
		return a.getUniverseSnapshot(limitPerClass)
	}
}

// getDailyTopN отбирает топ-N самых ликвидных инструментов и кэширует на весь день
func (a *App) getDailyTopN(limitPerClass int) []services.Security {
	a.dailyTopNMu.RLock()
	// Проверяем, актуален ли кэш
	if len(a.dailyTopNCache) > 0 && time.Since(a.dailyTopNCacheAt) < a.dailyTopNCacheTTL {
		cache := a.dailyTopNCache
		a.dailyTopNMu.RUnlock()
		return cache
	}
	a.dailyTopNMu.RUnlock()

	// Кэш устарел или пуст — обновляем
	a.dailyTopNMu.Lock()
	defer a.dailyTopNMu.Unlock()

	// Двойная проверка после получения блокировки
	if len(a.dailyTopNCache) > 0 && time.Since(a.dailyTopNCacheAt) < a.dailyTopNCacheTTL {
		return a.dailyTopNCache
	}

	a.addRuntimeEvent("info", "daily_top_n.updating", "обновление списка топ-N ликвидных инструментов")

	// Получаем все инструменты
	allInstruments := a.getUniverseSnapshot(limitPerClass)

	// Сортируем по ликвидности (turnover или numtrades)
	criteria := envOrDefault("AGENT_DAILY_TOP_CRITERIA", "turnover")
	sort.SliceStable(allInstruments, func(i, j int) bool {
		if criteria == "numtrades" {
			return allInstruments[i].NumTrades > allInstruments[j].NumTrades
		}
		return allInstruments[i].TurnoverRub > allInstruments[j].TurnoverRub
	})

	// Берём топ-N
	topN := envInt("AGENT_DAILY_TOP_N", 15)
	if topN < 5 {
		topN = 5
	}
	if topN > 50 {
		topN = 50
	}
	if len(allInstruments) > topN {
		allInstruments = allInstruments[:topN]
	}

	// Кэшируем
	a.dailyTopNCache = allInstruments
	a.dailyTopNCacheAt = time.Now()

	a.addRuntimeEvent("info", "daily_top_n.updated", fmt.Sprintf("обновлено: %d инструментов (критерий: %s)", len(allInstruments), criteria))

	return allInstruments
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return n
}

func envFloat(key string, def float64) float64 {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def
	}
	n, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return def
	}
	return n
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

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func intAbs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func maxDuration(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}

// GetCurrentPrice возвращает текущую цену
func (a *App) GetCurrentPrice(ticker string) float64 {
	price, _ := a.moex.GetCurrentPrice(ticker)
	return price
}

func normalizeBaseTicker(ticker string) string {
	t := strings.ToUpper(strings.TrimSpace(ticker))
	if t == "" {
		return ""
	}
	if i := strings.Index(t, "@"); i > 0 {
		t = t[:i]
	}
	if i := strings.Index(t, "_"); i > 0 {
		t = t[:i]
	}
	return strings.TrimSpace(t)
}

func (a *App) tradedTickersByClass(limit int, futures bool) []string {
	if limit <= 0 {
		limit = 200
	}
	seen := make(map[string]struct{}, 64)
	out := make([]string, 0, 32)
	add := func(raw string) {
		t := normalizeBaseTicker(raw)
		if t == "" {
			return
		}
		isFut := isLikelyFuturesTickerLocal(t)
		if futures != isFut {
			return
		}
		if _, ok := seen[t]; ok {
			return
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}

	for _, p := range a.GetPortfolio() {
		add(p.Ticker)
	}
	if a.db != nil {
		if trades, err := a.db.GetRecentTrades(limit); err == nil {
			for _, tr := range trades {
				add(tr.Ticker)
				if len(out) >= 40 {
					break
				}
			}
		}
	}
	return out
}

func mergeWatchlistWithTraded(base []services.Security, tickers []string, resolver func(string, int) []services.Security) []services.Security {
	if len(tickers) == 0 {
		return base
	}
	out := make([]services.Security, 0, len(base)+len(tickers))
	out = append(out, base...)
	seen := make(map[string]struct{}, len(base)+len(tickers))
	for _, s := range base {
		seen[strings.ToUpper(strings.TrimSpace(s.Ticker))] = struct{}{}
	}
	for _, t := range tickers {
		tt := strings.ToUpper(strings.TrimSpace(t))
		if tt == "" {
			continue
		}
		if _, ok := seen[tt]; ok {
			continue
		}
		candidates := resolver(tt, 10)
		added := false
		for _, s := range candidates {
			st := strings.ToUpper(strings.TrimSpace(s.Ticker))
			if st != tt {
				continue
			}
			out = append(out, s)
			seen[st] = struct{}{}
			added = true
			break
		}
		if !added {
			// fallback: чтобы не терять тикер в UI даже если ISS не вернул карточку
			out = append(out, services.Security{Ticker: tt, Name: tt, Price: 0})
			seen[tt] = struct{}{}
		}
	}
	return out
}

func (a *App) isValidFuturesTicker(ticker string) bool {
	if a.moex == nil {
		return true
	}
	t := strings.ToUpper(strings.TrimSpace(ticker))
	if t == "" {
		return false
	}
	candidates := a.moex.GetFuturesInstruments(t, 20)
	for _, s := range candidates {
		if strings.EqualFold(strings.TrimSpace(s.Ticker), t) {
			return true
		}
	}
	return false
}

// GetWatchlist возвращает список акций
func (a *App) GetWatchlist() []services.Security {
	base := a.moex.GetStockInstruments("", 120)
	tradedStocks := a.tradedTickersByClass(240, false)
	return mergeWatchlistWithTraded(base, tradedStocks, a.moex.GetStockInstruments)
}

// GetStockInstruments возвращает акции MOEX по строке поиска.
func (a *App) GetStockInstruments(query string, limit int) []services.Security {
	return a.moex.GetStockInstruments(query, limit)
}

// GetFuturesInstruments возвращает фьючерсы FORTS по строке поиска.
func (a *App) GetFuturesInstruments(query string, limit int) []services.Security {
	return a.moex.GetFuturesInstruments(query, limit)
}

// GetFuturesWatchlist возвращает список фьючерсов FORTS
func (a *App) GetFuturesWatchlist() []services.Security {
	base := a.moex.GetFuturesInstruments("", 120)
	tradedFutures := a.tradedTickersByClass(240, true)
	return mergeWatchlistWithTraded(base, tradedFutures, a.moex.GetFuturesInstruments)
}

// GetPortfolio возвращает портфель
// syncPositionsFromBroker синхронизирует позиции из Finam API с локальной БД
func (a *App) syncPositionsFromBroker() {
	if a.broker == nil || a.db == nil {
		return
	}
	
	ctx := context.Background()
	positions, err := a.broker.GetPortfolio(ctx)
	if err != nil {
		fmt.Printf("⚠️  Failed to get portfolio from broker: %v\n", err)
		return
	}
	
	// Очищаем старые позиции и записываем новые
	a.db.ClearPositions()
	for _, p := range positions {
		if p.Quantity != 0 { // Только открытые позиции
			a.db.UpsertPosition(p.Ticker, p.Quantity, p.AvgPrice)
		}
	}
	
	a.addRuntimeEvent("info", "positions.synced", fmt.Sprintf("синхронизировано %d позиций из Finam", len(positions)))
}

func (a *App) GetPortfolio() []services.Position {
	if a.broker != nil {
		ctx := context.Background()
		positions, err := a.broker.GetPortfolio(ctx)
		if err == nil {
			return positions
		}
		fmt.Printf("⚠️  Failed to get portfolio from broker: %v\n", err)
	}

	if a.db == nil {
		return []services.Position{}
	}

	positions, err := a.db.GetPositions()
	if err != nil {
		return []services.Position{}
	}

	for i := range positions {
		price, _ := a.moex.GetCurrentPrice(positions[i].Ticker)
		positions[i].CurrentPrice = price
		positions[i].PnL = float64(positions[i].Quantity) * (price - positions[i].AvgPrice)
	}

	return positions
}

type brokerPortfolioSnapLine struct {
	Ticker       string  `json:"ticker"`
	Quantity     int     `json:"quantity"`
	AvgPrice     float64 `json:"avg_price"`
	CurrentPrice float64 `json:"current_price"`
	PnL          float64 `json:"pnl,omitempty"`
}

type brokerPortfolioSnapshot struct {
	GeneratedAt string                    `json:"generated_at"`
	Source      string                    `json:"source"`
	Mode        string                    `json:"mode"`
	OpenCount   int                       `json:"open_count"`
	Positions   []brokerPortfolioSnapLine `json:"positions"`
}

// maybeWriteBrokerPortfolioSnapshot пишет снимок портфеля с брокера на диск (для Мии и операторского контроля).
// throttleFloorSec > 0 поднимает минимальный интервал записи (например вне сессии, чтобы не дёргать Finam каждые 10с).
func (a *App) maybeWriteBrokerPortfolioSnapshot(positions []services.Position, now time.Time, throttleFloorSec int) {
	if a == nil {
		return
	}
	if !envBool("AGENT_BROKER_POSITION_SNAPSHOT_ENABLED", true) {
		return
	}
	if a.mode == engine.ModePaper || a.broker == nil {
		return
	}
	minSec := envInt("AGENT_BROKER_POSITION_SNAPSHOT_MIN_SEC", 10)
	if minSec < 0 {
		minSec = 0
	}
	if minSec > 3600 {
		minSec = 3600
	}
	if throttleFloorSec > minSec {
		minSec = throttleFloorSec
	}
	path := strings.TrimSpace(os.Getenv("AGENT_BROKER_POSITION_SNAPSHOT_PATH"))
	if path == "" {
		path = "/var/log/muhlya-reports/broker_positions_snapshot.json"
	}
	brokerPortfolioSnapMu.Lock()
	if minSec > 0 && !brokerPortfolioSnapLast.IsZero() && now.Sub(brokerPortfolioSnapLast) < time.Duration(minSec)*time.Second {
		brokerPortfolioSnapMu.Unlock()
		return
	}
	brokerPortfolioSnapMu.Unlock()

	positionsToWrite := positions
	if positionsToWrite == nil {
		positionsToWrite = a.GetPortfolio()
	}

	lines := make([]brokerPortfolioSnapLine, 0, len(positionsToWrite))
	openN := 0
	for _, p := range positionsToWrite {
		if p.Quantity == 0 {
			continue
		}
		openN++
		t := strings.TrimSpace(p.Ticker)
		lines = append(lines, brokerPortfolioSnapLine{
			Ticker:       t,
			Quantity:     p.Quantity,
			AvgPrice:     p.AvgPrice,
			CurrentPrice: p.CurrentPrice,
			PnL:          p.PnL,
		})
	}
	snap := brokerPortfolioSnapshot{
		GeneratedAt: now.UTC().Format(time.RFC3339),
		Source:      "finam",
		Mode:        string(a.mode),
		OpenCount:   openN,
		Positions:   lines,
	}
	raw, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	tmp, err := os.CreateTemp(dir, ".broker_positions_snapshot_*.tmp")
	if err != nil {
		return
	}
	tmpPath := tmp.Name()
	_, _ = tmp.Write(raw)
	_ = tmp.Sync()
	_ = tmp.Close()
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		_ = os.Remove(tmpPath)
		return
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return
	}
	brokerPortfolioSnapMu.Lock()
	brokerPortfolioSnapLast = now
	brokerPortfolioSnapMu.Unlock()
}

// GetTradeJournal — журнал сделок: при подключённом Finam и не paper-режиме добавляет сделки из API
// GET /v1/accounts/{id}/trades, плюс записи из локальной БД; сортировка по времени (новые сверху).
func (a *App) GetTradeJournal(limit int) []TradeJournalEntry {
	if limit <= 0 {
		limit = 80
	}
	var merged []TradeJournalEntry

	if a.broker != nil && a.mode != engine.ModePaper {
		ctx := context.Background()
		rows, err := a.broker.GetAccountTrades(ctx, limit, nil, nil)
		if err == nil {
			for _, t := range rows {
				at := normalizeTradeTime(t.ExecutedAt, time.UTC)
				merged = append(merged, TradeJournalEntry{
					ID:         0,
					Ticker:     t.Symbol,
					Side:       t.Side,
					Quantity:   t.Quantity,
					Price:      t.Price,
					Status:     "executed",
					ExecutedAt: at.UTC().Format(time.RFC3339),
					Source:     "finam",
					ExternalID: t.TradeID,
				})
			}
		} else {
			fmt.Printf("⚠️ Finam GetAccountTrades: %v\n", err)
		}
	}

	if a.db != nil {
		trades, err := a.db.GetRecentTrades(limit)
		if err == nil {
			for _, t := range trades {
				raw := normalizeTradeTime(t.ExecutedAt, a.tradingLocation)
				// executed_at из локальной БД хранится без timezone.
				// Принудительно считаем его временем Novosibirsk, чтобы исключить +7 сдвиг в UI.
				at := raw
				if a.tradingLocation != nil {
					at = time.Date(raw.Year(), raw.Month(), raw.Day(), raw.Hour(), raw.Minute(), raw.Second(), raw.Nanosecond(), a.tradingLocation)
				}
				merged = append(merged, TradeJournalEntry{
					ID:         t.ID,
					Ticker:     t.Ticker,
					Side:       t.Side,
					Quantity:   t.Quantity,
					Price:      t.Price,
					Status:     t.Status,
					ExecutedAt: at.Format(time.RFC3339),
					Source:     "local",
				})
			}
		}
	}

	sort.Slice(merged, func(i, j int) bool {
		ti, e1 := time.Parse(time.RFC3339, merged[i].ExecutedAt)
		tj, e2 := time.Parse(time.RFC3339, merged[j].ExecutedAt)
		if e1 != nil || e2 != nil {
			return i < j
		}
		return ti.After(tj)
	})
	annotateClosedPositionPL(merged)
	if len(merged) > limit {
		merged = merged[:limit]
	}
	return merged
}

func annotateClosedPositionPL(entries []TradeJournalEntry) {
	type posState struct {
		Qty int
		Avg float64
	}
	if len(entries) == 0 {
		return
	}
	asc := make([]int, len(entries))
	for i := range entries {
		asc[i] = i
	}
	sort.Slice(asc, func(i, j int) bool {
		ei := entries[asc[i]]
		ej := entries[asc[j]]
		ti, e1 := time.Parse(time.RFC3339, ei.ExecutedAt)
		tj, e2 := time.Parse(time.RFC3339, ej.ExecutedAt)
		if e1 != nil && e2 != nil {
			return asc[i] < asc[j]
		}
		if e1 != nil {
			return true
		}
		if e2 != nil {
			return false
		}
		if ti.Equal(tj) {
			return asc[i] < asc[j]
		}
		return ti.Before(tj)
	})

	stateByTicker := make(map[string]posState, 128)
	seenFinamTrade := make(map[string]struct{}, 256)
	for _, idx := range asc {
		e := &entries[idx]
		e.RealizedPL = nil
		if !isFilledLikeTradeForPL(*e) {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(e.Source), "finam") {
			tid := strings.TrimSpace(e.ExternalID)
			if tid != "" {
				if _, ok := seenFinamTrade[tid]; ok {
					continue
				}
				seenFinamTrade[tid] = struct{}{}
			}
		}
		side := strings.ToLower(strings.TrimSpace(e.Side))
		qty := e.Quantity
		price := e.Price
		if qty <= 0 || price <= 0 || (side != "buy" && side != "sell") {
			continue
		}
		key := canonicalTickerKey(e.Ticker)
		if key == "" {
			key = strings.ToUpper(strings.TrimSpace(e.Ticker))
		}
		ps := stateByTicker[key]
		closedQty := 0
		realized := 0.0

		if side == "buy" {
			if ps.Qty < 0 {
				cover := minInt(qty, -ps.Qty)
				realized += (ps.Avg - price) * float64(cover)
				ps.Qty += cover
				closedQty += cover
				qty -= cover
				if ps.Qty == 0 {
					ps.Avg = 0
				}
			}
			if qty > 0 {
				if ps.Qty > 0 {
					total := float64(ps.Qty)*ps.Avg + float64(qty)*price
					ps.Qty += qty
					ps.Avg = total / float64(ps.Qty)
				} else {
					ps.Qty = qty
					ps.Avg = price
				}
			}
		} else { // sell
			if ps.Qty > 0 {
				closeQty := minInt(qty, ps.Qty)
				realized += (price - ps.Avg) * float64(closeQty)
				ps.Qty -= closeQty
				closedQty += closeQty
				qty -= closeQty
				if ps.Qty == 0 {
					ps.Avg = 0
				}
			}
			if qty > 0 {
				if ps.Qty < 0 {
					totalShort := float64(-ps.Qty)*ps.Avg + float64(qty)*price
					ps.Qty -= qty
					ps.Avg = totalShort / float64(-ps.Qty)
				} else {
					ps.Qty = -qty
					ps.Avg = price
				}
			}
		}

		stateByTicker[key] = ps
		if closedQty > 0 && ps.Qty == 0 {
			v := realized
			e.RealizedPL = &v
		}
	}
}

func isFilledLikeTradeForPL(e TradeJournalEntry) bool {
	status := strings.ToLower(strings.TrimSpace(e.Status))
	source := strings.ToLower(strings.TrimSpace(e.Source))
	// Finam trades из API в журнале приходят уже исполненными.
	if source == "finam" {
		return true
	}
	// Для локальных и прочих источников считаем только фактически исполненные.
	return strings.Contains(status, "executed")
}

// PlaceSLTPBrackets выставляет связанную заявку SL/TP через Finam gRPC PlaceSLTPOrder.
// closeSide — сторона закрытия позиции: "buy" или "sell" (для лонга акций обычно "sell").
// Для каждой ноги задайте количество и цену; нулевая цена или количество отключает ногу.
func (a *App) PlaceSLTPBrackets(ticker string, closeSide string, slQty int, slPrice float64, tpQty int, tpPrice float64) (string, error) {
	if a.mode == engine.ModePaper {
		return "", fmt.Errorf("режим paper: SL/TP не отправляется на брокера")
	}
	if a.broker == nil {
		return "", fmt.Errorf("брокер не подключён")
	}
	ticker = strings.TrimSpace(ticker)
	if ticker == "" {
		return "", fmt.Errorf("пустой тикер")
	}
	if ok, reason := a.isTradingAllowedForTicker(ticker); !ok {
		return "", fmt.Errorf("торговая сессия: %s", reason)
	}
	if err := a.allowOrderByThrottle(ticker, "sltp"); err != nil {
		return "", err
	}

	s := strings.ToLower(strings.TrimSpace(closeSide))
	var gside tradev1.Side
	switch s {
	case "buy":
		gside = tradev1.Side_SIDE_BUY
	case "sell":
		gside = tradev1.Side_SIDE_SELL
	default:
		return "", fmt.Errorf("closeSide: укажите buy или sell")
	}

	hasSL := slQty > 0 && slPrice > 0
	hasTP := tpQty > 0 && tpPrice > 0
	if !hasSL && !hasTP {
		return "", fmt.Errorf("нужны SL и/или TP (количество и цена > 0)")
	}

	if a.risk != nil {
		maxN := 0.0
		if hasSL {
			if n := float64(slQty) * slPrice; n > maxN {
				maxN = n
			}
		}
		if hasTP {
			if n := float64(tpQty) * tpPrice; n > maxN {
				maxN = n
			}
		}
		if maxN > 0 {
			eside := engine.SideSell
			if gside == tradev1.Side_SIDE_BUY {
				eside = engine.SideBuy
			}
			refPrice := slPrice
			if refPrice <= 0 {
				refPrice = tpPrice
			}
			qtyEst := slQty
			if tpQty > qtyEst {
				qtyEst = tpQty
			}
			if err := a.risk.Allow(engine.OrderIntent{
				Symbol:   ticker,
				Side:     eside,
				Qty:      float64(qtyEst),
				Price:    refPrice,
				Notional: maxN,
				Time:     time.Now(),
			}); err != nil {
				// Защитные выходы (SL/TP) не должны блокироваться суточным notional-лимитом.
				// Иначе в просадке можно остаться в позиции без возможности закрытия риска.
				if eside == engine.SideSell && isRiskDailyNotionalExceeded(err) {
					a.addRuntimeEvent(
						"warn",
						"risk.daily_notional_bypass_exit_bracket",
						fmt.Sprintf("%s qty=%d ref_price=%.2f notional=%.2f", ticker, qtyEst, refPrice, maxN),
					)
				} else {
					return "", err
				}
			}
		}
	}

	ctx := context.Background()
	id, err := a.broker.PlaceSLTPOrder(ctx, ticker, gside, slQty, slPrice, tpQty, tpPrice)
	if err != nil {
		return "", err
	}
	fmt.Printf("✅ SL/TP заявка выставлена: %s\n", id)
	a.invalidateStatsCache()
	return id, nil
}

// BuyPosition покупает акцию
func (a *App) BuyPosition(ticker string, quantity int, price float64) bool {
	// Округляем цену до 2 знаков после запятой (требование Finam API)
	price = math.Round(price*100) / 100
	if a.mode == engine.ModePaper {
		if a.db == nil {
			return false
		}
		return a.db.BuyPosition(ticker, quantity, price)
	}
	if a.broker != nil {
		if ok, reason := a.isTradingAllowedForTicker(ticker); !ok {
			err := fmt.Errorf("outside trading session: %s", reason)
			fmt.Printf("🛑 Session blocked order: %v\n", err)
			if a.db != nil {
				_ = a.db.LogTrade(ticker, "buy", quantity, price, rejectedStatusFromError(err))
			}
			a.invalidateStatsCache()
			return false
		}
		if pending, pendingID := a.hasPendingOrder(ticker, "buy"); pending {
			msg := fmt.Sprintf("pending buy order exists for %s", ticker)
			if pendingID != "" {
				msg = fmt.Sprintf("%s (order_id=%s)", msg, pendingID)
			}
			a.addRuntimeEvent("warn", "auto.skip_pending_order", msg)
			a.rememberRecentAutoTicker(ticker, 90*time.Second)
			a.invalidateStatsCache()
			return false
		}
		if err := a.allowOrderByThrottle(ticker, "buy"); err != nil {
			fmt.Printf("🛑 Order throttled: %v\n", err)
			if a.db != nil {
				_ = a.db.LogTrade(ticker, "buy", quantity, price, rejectedStatusFromError(err))
			}
			a.invalidateStatsCache()
			return false
		}
		if price <= 0 {
			price, _ = a.moex.GetCurrentPrice(ticker)
		}
		normalizedQty, qErr := a.normalizeOrderQuantity(ticker, quantity, "buy")
		if qErr != nil {
			fmt.Printf("🛑 Quantity rejected: %v\n", qErr)
			if a.db != nil {
				_ = a.db.LogTrade(ticker, "buy", quantity, price, rejectedStatusFromError(qErr))
			}
			a.invalidateStatsCache()
			return false
		}
		if normalizedQty != quantity {
			fmt.Printf("ℹ️ Quantity normalized for buy: %s %d -> %d\n", ticker, quantity, normalizedQty)
		}
		quantity = normalizedQty
		if bal, err := a.broker.GetBalance(context.Background()); err == nil {
			qty := a.affordableQuantity(bal, price, quantity, isLikelyFuturesTickerLocal(ticker), ticker)
			if qty <= 0 {
				fmt.Printf("🛑 Buy blocked by cash limit: ticker=%s qty=%d price=%.2f bal=%.2f\n", ticker, quantity, price, bal)
				return false
			}
			quantity = a.floorToLot(ticker, qty)
			if quantity <= 0 {
				err := fmt.Errorf("buy blocked by lot affordability: ticker=%s qty=%d price=%.2f bal=%.2f", ticker, qty, price, bal)
				fmt.Printf("🛑 %v\n", err)
				if a.db != nil {
					_ = a.db.LogTrade(ticker, "buy", qty, price, rejectedStatusFromError(err))
				}
				a.invalidateStatsCache()
				return false
			}
		}

		if a.risk != nil {
			estPrice := price
			if estPrice <= 0 {
				estPrice, _ = a.moex.GetCurrentPrice(ticker)
			}
			notional := float64(quantity) * estPrice
			if err := a.risk.Allow(engine.OrderIntent{
				Symbol:   ticker,
				Side:     engine.SideBuy,
				Qty:      float64(quantity),
				Price:    estPrice,
				Notional: notional,
				Time:     time.Now(),
			}); err != nil {
				fmt.Printf("🛑 Risk rejected order: %v\n", err)
				a.armRiskDailyPauseFromError(err)
				if a.db != nil {
					_ = a.db.LogTrade(ticker, "buy", quantity, estPrice, rejectedStatusFromError(err))
				}
				a.invalidateStatsCache()
				return false
			}
		}

		if err := a.validateOrderInput(ticker, quantity, price, price > 0, true); err != nil {
			fmt.Printf("🛑 Order precheck failed: %v\n", err)
			if a.db != nil {
				_ = a.db.LogTrade(ticker, "buy", quantity, price, rejectedStatusFromError(err))
			}
			a.invalidateStatsCache()
			return false
		}

		ctx := context.Background()
		lot := 1
		if a.moex != nil {
			lot = a.moex.GetLotSize(ticker)
		}
		if lot < 1 {
			lot = 1
		}
		a.addRuntimeEvent("info", "order.buy_final_qty", fmt.Sprintf("%s qty=%d lot=%d price=%.2f", ticker, quantity, lot, price))

		placeBuy := func(qty int) (string, error) {
			if price > 0 {
				return a.broker.BuyLimit(ctx, ticker, qty, price)
			}
			return a.broker.BuyMarket(ctx, ticker, qty)
		}

		orderId, err := placeBuy(quantity)

		if err != nil {
			msg := strings.ToLower(err.Error())
			isInvalidQty := strings.Contains(msg, "invalid quantity value") || strings.Contains(msg, "неверное значение параметра 'количество'")
			isMoneyShortage := strings.Contains(msg, "[156]") || strings.Contains(msg, "money shortage") || strings.Contains(msg, "не хватает денежных средств")
			isPriceLimits := strings.Contains(msg, "(10032)") || strings.Contains(msg, "price limits") || strings.Contains(msg, "price should be specified")
			if isInvalidQty && quantity > lot {
				retryQty := floorQuantityByLot(quantity-lot, lot)
				if retryQty > 0 && retryQty < quantity {
					a.addRuntimeEvent("warn", "order.buy_retry_qty_down", fmt.Sprintf("%s qty=%d->%d lot=%d", ticker, quantity, retryQty, lot))
					if retryOrderID, retryErr := placeBuy(retryQty); retryErr == nil {
						fmt.Printf("✅ Order placed after qty retry: %s\n", retryOrderID)
						a.addRuntimeEvent("info", "order.buy_placed_retry", fmt.Sprintf("%s qty=%d order_id=%s", ticker, retryQty, retryOrderID))
						if a.db != nil {
							execPrice := price
							if execPrice <= 0 {
								execPrice, _ = a.moex.GetCurrentPrice(ticker)
							}
							_ = a.db.LogTrade(ticker, "buy", retryQty, execPrice, "placed")
						}
						a.invalidateStatsCache()
						return true
					}
				}
			}
			if isMoneyShortage {
				a.addRuntimeEvent("warn", "order.blocked_cash", fmt.Sprintf("%s qty=%d price=%.2f", ticker, quantity, price))
				a.rememberRecentAutoTicker(ticker, 2*time.Minute)
				a.setBuyCashPause(90 * time.Second)
			}
			if isPriceLimits {
				a.addRuntimeEvent("warn", "order.blocked_price_limits", fmt.Sprintf("%s qty=%d price=%.2f — цена вне лимитов Finam", ticker, quantity, price))
				a.rememberRecentAutoTicker(ticker, 10*time.Minute)
				a.registerBrokerValidationFailure(ticker, err)
			}
			fmt.Printf("❌ Order failed: %v\n", err)
			a.addRuntimeEvent("error", "order.buy_failed", fmt.Sprintf("%s qty=%d err=%v", ticker, quantity, err))
			a.registerBrokerValidationFailure(ticker, err)
			if a.db != nil {
				_ = a.db.LogTrade(ticker, "buy", quantity, price, rejectedStatusFromError(err))
			}
			a.invalidateStatsCache()
			return false
		}
		fmt.Printf("✅ Order placed: %s\n", orderId)
		a.addRuntimeEvent("info", "order.buy_placed", fmt.Sprintf("%s qty=%d order_id=%s", ticker, quantity, orderId))
		if a.db != nil {
			execPrice := price
			if execPrice <= 0 {
				execPrice, _ = a.moex.GetCurrentPrice(ticker)
			}
			_ = a.db.LogTrade(ticker, "buy", quantity, execPrice, "placed")
		}
		a.invalidateStatsCache()
		return true
	}

	if a.db == nil {
		return false
	}
	return a.db.BuyPosition(ticker, quantity, price)
}

// SellPosition продаёт акцию
func (a *App) SellPosition(ticker string, quantity int, price float64) bool {
	return a.sellPositionInternal(ticker, quantity, price, false)
}

// OpenShortPosition открывает short по акции (без требования long-остатка).
func (a *App) OpenShortPosition(ticker string, quantity int, price float64) bool {
	// Округляем цену до 2 знаков после запятой (требование Finam API)
	price = math.Round(price*100) / 100
	return a.sellPositionInternal(ticker, quantity, price, true)
}

func (a *App) sellPositionInternal(ticker string, quantity int, price float64, allowShortOpen bool) bool {
	if a.mode == engine.ModePaper {
		if allowShortOpen {
			a.addRuntimeEvent("warn", "order.short_open_blocked_paper", fmt.Sprintf("%s qty=%d", ticker, quantity))
			return false
		}
		if a.db == nil {
			return false
		}
		return a.db.SellPosition(ticker, quantity, price)
	}
	if a.broker != nil {
		if ok, reason := a.isTradingAllowedForTicker(ticker); !ok {
			err := fmt.Errorf("outside trading session: %s", reason)
			fmt.Printf("🛑 Session blocked order: %v\n", err)
			if a.db != nil {
				_ = a.db.LogTrade(ticker, "sell", quantity, price, rejectedStatusFromError(err))
			}
			a.invalidateStatsCache()
			return false
		}
		// Сначала проверяем pending-ордер, чтобы не раздувать шум throttle/rejected
		// при повторных попытках закрытия в соседних циклах авто.
		if pending, pendingID := a.hasPendingOrder(ticker, "sell"); pending {
			msg := fmt.Sprintf("pending sell order exists for %s", ticker)
			if pendingID != "" {
				msg = fmt.Sprintf("%s (order_id=%s)", msg, pendingID)
			}
			a.addRuntimeEvent("warn", "auto.skip_pending_order", msg)
			a.invalidateStatsCache()
			return false
		}
		if err := a.allowOrderByThrottle(ticker, "sell"); err != nil {
			fmt.Printf("🛑 Order throttled: %v\n", err)
			if a.db != nil {
				_ = a.db.LogTrade(ticker, "sell", quantity, price, rejectedStatusFromError(err))
			}
			a.invalidateStatsCache()
			return false
		}
		normalizedQty, qErr := a.normalizeOrderQuantity(ticker, quantity, "sell")
		if qErr != nil {
			fmt.Printf("🛑 Quantity rejected: %v\n", qErr)
			if a.db != nil {
				_ = a.db.LogTrade(ticker, "sell", quantity, price, rejectedStatusFromError(qErr))
			}
			a.invalidateStatsCache()
			return false
		}
		if normalizedQty != quantity {
			fmt.Printf("ℹ️ Quantity normalized for sell: %s %d -> %d\n", ticker, quantity, normalizedQty)
		}
		quantity = normalizedQty
		if !allowShortOpen {
			clampedQty, clampErr := a.clampSellQuantityToAvailableLong(ticker, quantity)
			if clampErr != nil {
				fmt.Printf("🛑 Sell blocked by position availability: %v\n", clampErr)
				if a.db != nil {
					_ = a.db.LogTrade(ticker, "sell", quantity, price, rejectedStatusFromError(clampErr))
				}
				a.invalidateStatsCache()
				return false
			}
			if clampedQty != quantity {
				a.addRuntimeEvent("warn", "order.sell_qty_clamped", fmt.Sprintf("%s qty=%d->%d (available long)", ticker, quantity, clampedQty))
			}
			quantity = clampedQty
		} else {
			a.addRuntimeEvent("info", "order.short_open_intent", fmt.Sprintf("%s qty=%d price=%.2f", ticker, quantity, price))
		}
		if a.risk != nil {
			estPrice := price
			if estPrice <= 0 {
				estPrice, _ = a.moex.GetCurrentPrice(ticker)
			}
			notional := float64(quantity) * estPrice
			if err := a.risk.Allow(engine.OrderIntent{
				Symbol:   ticker,
				Side:     engine.SideSell,
				Qty:      float64(quantity),
				Price:    estPrice,
				Notional: notional,
				Time:     time.Now(),
			}); err != nil {
				// Для закрытия long-позиции пропускаем только daily-notional reject,
				// чтобы стоп/выход не блокировались "дневным потолком".
				if !allowShortOpen && isRiskDailyNotionalExceeded(err) {
					a.addRuntimeEvent(
						"warn",
						"risk.daily_notional_bypass_exit_sell",
						fmt.Sprintf("%s qty=%d est_price=%.2f notional=%.2f", ticker, quantity, estPrice, notional),
					)
				} else {
					fmt.Printf("🛑 Risk rejected order: %v\n", err)
					a.armRiskDailyPauseFromError(err)
					if a.db != nil {
						_ = a.db.LogTrade(ticker, "sell", quantity, estPrice, rejectedStatusFromError(err))
					}
					a.invalidateStatsCache()
					return false
				}
			}
		}

		// Для закрытия существующей long-позиции denylist не должен блокировать risk-exit.
		if err := a.validateOrderInput(ticker, quantity, price, price > 0, allowShortOpen); err != nil {
			fmt.Printf("🛑 Order precheck failed: %v\n", err)
			if a.db != nil {
				_ = a.db.LogTrade(ticker, "sell", quantity, price, rejectedStatusFromError(err))
			}
			a.invalidateStatsCache()
			return false
		}
		ctx := context.Background()
		var orderId string
		var err error

		if price > 0 {
			orderId, err = a.broker.SellLimit(ctx, ticker, quantity, price)
		} else {
			orderId, err = a.broker.SellMarket(ctx, ticker, quantity)
		}

		if err != nil {
			msg := strings.ToLower(err.Error())
			isShortage := strings.Contains(msg, "[157]") || strings.Contains(msg, "equities shortage") || strings.Contains(msg, "не хватает собственных бумаг")
			isQualification := strings.Contains(msg, "[257]") || strings.Contains(msg, "требующие тестирования") || strings.Contains(msg, "confirm your qualification level")
			isPriceError := strings.Contains(msg, "[34]") || strings.Contains(msg, "invalid value for field: price")
			
			// Fallback на фьючерсы при ошибке шорта акции
			if allowShortOpen && (isShortage || isQualification || isPriceError) {
				if a.tryShortFuturesFallback(ticker, quantity, price) {
					return true
				}
			}
			
			if isShortage && !allowShortOpen {
				refreshedQty, clampErr := a.clampSellQuantityToAvailableLong(ticker, quantity)
				if clampErr == nil && refreshedQty > 0 && refreshedQty < quantity {
					a.addRuntimeEvent("warn", "order.sell_retry_qty_down", fmt.Sprintf("%s qty=%d->%d after shortage [157]", ticker, quantity, refreshedQty))
					if price > 0 {
						orderId, err = a.broker.SellLimit(ctx, ticker, refreshedQty, price)
					} else {
						orderId, err = a.broker.SellMarket(ctx, ticker, refreshedQty)
					}
					if err == nil {
						fmt.Printf("✅ Order placed after sell qty retry: %s\n", orderId)
						a.addRuntimeEvent("info", "order.sell_placed_retry", fmt.Sprintf("%s qty=%d order_id=%s", ticker, refreshedQty, orderId))
						if a.db != nil {
							execPrice := price
							if execPrice <= 0 {
								execPrice, _ = a.moex.GetCurrentPrice(ticker)
							}
							_ = a.db.LogTrade(ticker, "sell", refreshedQty, execPrice, "placed")
						}
						a.invalidateStatsCache()
						return true
					}
				}
				// Защита от циклических повторов, если брокер говорит "не хватает бумаг".
				a.rememberRecentAutoTicker(ticker, 5*time.Minute)
			} else if allowShortOpen && (isShortage || isQualification) {
				pauseMin := envInt("AGENT_SHORT_OPEN_GLOBAL_PAUSE_MIN", 20)
				if pauseMin < 1 {
					pauseMin = 1
				}
				pauseDur := time.Duration(pauseMin) * time.Minute
				if isQualification && pauseDur < 60*time.Minute {
					pauseDur = 60 * time.Minute
				}
				a.setShortOpenPause(pauseDur)
				a.addRuntimeEvent("warn", "order.short_open_global_pause", fmt.Sprintf("pause=%s reason=%s", pauseDur.Round(time.Minute), strings.TrimSpace(err.Error())))
				tickerPauseMin := envInt("AGENT_SHORT_OPEN_TICKER_BLOCK_157_MIN", 180)
				if tickerPauseMin < 10 {
					tickerPauseMin = 10
				}
				if tickerPauseMin > 24*60 {
					tickerPauseMin = 24 * 60
				}
				tickerPauseDur := time.Duration(tickerPauseMin) * time.Minute
				if isQualification && tickerPauseDur < 12*time.Hour {
					tickerPauseDur = 12 * time.Hour
				}
				a.setShortOpenTickerPause(ticker, tickerPauseDur)
				a.addRuntimeEvent("warn", "order.short_open_ticker_pause", fmt.Sprintf("%s pause=%s reason=%s", ticker, tickerPauseDur.Round(time.Minute), strings.TrimSpace(err.Error())))
				a.rememberRecentAutoTicker(ticker, 15*time.Minute)
			}
			fmt.Printf("❌ Order failed: %v\n", err)
			failStep := "order.sell_failed"
			if allowShortOpen {
				failStep = "order.short_open_failed"
			}
			a.addRuntimeEvent("error", failStep, fmt.Sprintf("%s qty=%d err=%v", ticker, quantity, err))
			a.registerBrokerValidationFailure(ticker, err)
			if a.db != nil {
				_ = a.db.LogTrade(ticker, "sell", quantity, price, rejectedStatusFromError(err))
			}
			a.invalidateStatsCache()
			return false
		}
		fmt.Printf("✅ Order placed: %s\n", orderId)
		placedStep := "order.sell_placed"
		if allowShortOpen {
			placedStep = "order.short_open_placed"
		}
		a.addRuntimeEvent("info", placedStep, fmt.Sprintf("%s qty=%d order_id=%s", ticker, quantity, orderId))
		if a.db != nil {
			execPrice := price
			if execPrice <= 0 {
				execPrice, _ = a.moex.GetCurrentPrice(ticker)
			}
			_ = a.db.LogTrade(ticker, "sell", quantity, execPrice, "placed")
		}
		a.invalidateStatsCache()
		return true
	}

	if a.db == nil {
		return false
	}
	if allowShortOpen {
		a.addRuntimeEvent("warn", "order.short_open_blocked_no_broker", fmt.Sprintf("%s qty=%d", ticker, quantity))
		return false
	}
	return a.db.SellPosition(ticker, quantity, price)
}

// BuyFutures покупает фьючерс
func (a *App) BuyFutures(ticker string, quantity int, price float64) bool {
	if a.mode == engine.ModePaper {
		// Симуляция фьючерсов пока не реализована
		return false
	}
	if a.broker == nil {
		return false
	}
	if !a.isValidFuturesTicker(ticker) {
		err := fmt.Errorf("futures preflight rejected unknown ticker=%s", ticker)
		a.addRuntimeEvent("warn", "order.futures_preflight_reject", err.Error())
		if a.db != nil {
			_ = a.db.LogTrade(ticker, "buy", quantity, price, rejectedStatusFromError(err))
		}
		a.invalidateStatsCache()
		return false
	}
	if ok, why := a.broker.ValidateFuturesSymbolForAccount(context.Background(), ticker); !ok {
		a.noteFuturesValidationResult(ticker, false, why)
		sym := a.broker.OrderSymbolForAPI(ticker)
		a.registerBrokerValidationFailure(ticker, fmt.Errorf("finam asset: %s", why))
		a.addRuntimeEvent("warn", "order.futures_finam_getasset_reject", fmt.Sprintf("%s finam_sym=%s %s", ticker, sym, why))
		if a.db != nil {
			_ = a.db.LogTrade(ticker, "buy", quantity, price, rejectedStatusFromError(fmt.Errorf("finam asset: %s", why)))
		}
		a.invalidateStatsCache()
		return false
	}
	a.noteFuturesValidationResult(ticker, true, "")
	if ok, reason := a.isTradingAllowedForTicker(ticker); !ok {
		err := fmt.Errorf("outside trading session: %s", reason)
		fmt.Printf("🛑 Session blocked order: %v\n", err)
		if a.db != nil {
			_ = a.db.LogTrade(ticker, "buy", quantity, price, rejectedStatusFromError(err))
		}
		a.invalidateStatsCache()
		return false
	}
	if pending, pendingID := a.hasPendingOrder(ticker, "buy"); pending {
		msg := fmt.Sprintf("pending buy order exists for %s", ticker)
		if pendingID != "" {
			msg = fmt.Sprintf("%s (order_id=%s)", msg, pendingID)
		}
		a.addRuntimeEvent("warn", "auto.skip_pending_order", msg)
		a.rememberRecentAutoTicker(ticker, 90*time.Second)
		a.invalidateStatsCache()
		return false
	}
	if err := a.allowOrderByThrottle(ticker, "buy"); err != nil {
		fmt.Printf("🛑 Order throttled: %v\n", err)
		if a.db != nil {
			_ = a.db.LogTrade(ticker, "buy", quantity, price, rejectedStatusFromError(err))
		}
		a.invalidateStatsCache()
		return false
	}

	if price <= 0 {
		price, _ = a.moex.GetCurrentPrice(ticker)
	}
	normalizedQty, qErr := a.normalizeOrderQuantity(ticker, quantity, "buy")
	if qErr != nil {
		fmt.Printf("🛑 Quantity rejected: %v\n", qErr)
		if a.db != nil {
			_ = a.db.LogTrade(ticker, "buy", quantity, price, rejectedStatusFromError(qErr))
		}
		a.invalidateStatsCache()
		return false
	}
	if normalizedQty != quantity {
		fmt.Printf("ℹ️ Quantity normalized for futures buy: %s %d -> %d\n", ticker, quantity, normalizedQty)
	}
	quantity = normalizedQty
	if bal, err := a.broker.GetBalance(context.Background()); err == nil {
		qty := a.affordableQuantity(bal, price, quantity, true, ticker)
		if qty <= 0 {
			fmt.Printf("🛑 Futures buy blocked by cash limit: ticker=%s qty=%d price=%.2f bal=%.2f\n", ticker, quantity, price, bal)
			return false
		}
		quantity = a.floorToLot(ticker, qty)
		if quantity <= 0 {
			err := fmt.Errorf("futures buy blocked by lot affordability: ticker=%s qty=%d price=%.2f bal=%.2f", ticker, qty, price, bal)
			fmt.Printf("🛑 %v\n", err)
			if a.db != nil {
				_ = a.db.LogTrade(ticker, "buy", qty, price, rejectedStatusFromError(err))
			}
			a.invalidateStatsCache()
			return false
		}
	}

	if a.risk != nil {
		estPrice := price
		if estPrice <= 0 {
			estPrice, _ = a.moex.GetCurrentPrice(ticker)
		}
		notional := float64(quantity) * estPrice
		if err := a.risk.Allow(engine.OrderIntent{
			Symbol:   ticker,
			Side:     engine.SideBuy,
			Qty:      float64(quantity),
			Price:    estPrice,
			Notional: notional,
			Time:     time.Now(),
		}); err != nil {
			fmt.Printf("🛑 Risk rejected futures order: %v\n", err)
			a.armRiskDailyPauseFromError(err)
			if a.db != nil {
				_ = a.db.LogTrade(ticker, "buy", quantity, estPrice, rejectedStatusFromError(err))
			}
			a.invalidateStatsCache()
			return false
		}
	}

	if err := a.validateOrderInput(ticker, quantity, price, price > 0, true); err != nil {
		fmt.Printf("🛑 Futures order precheck failed: %v\n", err)
		if a.db != nil {
			_ = a.db.LogTrade(ticker, "buy", quantity, price, rejectedStatusFromError(err))
		}
		a.invalidateStatsCache()
		return false
	}

	ctx := context.Background()
	var orderId string
	var err error

	if price > 0 {
		orderId, err = a.broker.BuyLimit(ctx, ticker, quantity, price)
	} else {
		orderId, err = a.broker.BuyMarket(ctx, ticker, quantity)
	}

	if err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "[156]") || strings.Contains(msg, "money shortage") || strings.Contains(msg, "не хватает денежных средств") {
			a.addRuntimeEvent("warn", "order.blocked_cash", fmt.Sprintf("%s qty=%d price=%.2f", ticker, quantity, price))
			a.rememberRecentAutoTicker(ticker, 2*time.Minute)
			a.setBuyCashPause(90 * time.Second)
		}
		fmt.Printf("❌ Futures order failed: %v\n", err)
		a.addRuntimeEvent("error", "order.futures_buy_failed", fmt.Sprintf("%s qty=%d err=%v", ticker, quantity, err))
		a.registerBrokerValidationFailure(ticker, err)
		if a.db != nil {
			_ = a.db.LogTrade(ticker, "buy", quantity, price, rejectedStatusFromError(err))
		}
		a.invalidateStatsCache()
		return false
	}
	fmt.Printf("✅ Futures order placed: %s\n", orderId)
	a.addRuntimeEvent("info", "order.futures_buy_placed", fmt.Sprintf("%s qty=%d order_id=%s", ticker, quantity, orderId))
	if a.db != nil {
		execPrice := price
		if execPrice <= 0 {
			execPrice, _ = a.moex.GetCurrentPrice(ticker)
		}
		_ = a.db.LogTrade(ticker, "buy", quantity, execPrice, "placed")
	}
	a.invalidateStatsCache()
	return true
}

// SellFutures продаёт фьючерс
func (a *App) SellFutures(ticker string, quantity int, price float64) bool {
	if a.mode == engine.ModePaper {
		// Симуляция фьючерсов пока не реализована
		return false
	}
	if a.broker == nil {
		return false
	}
	if !a.isValidFuturesTicker(ticker) {
		err := fmt.Errorf("futures preflight rejected unknown ticker=%s", ticker)
		a.addRuntimeEvent("warn", "order.futures_preflight_reject", err.Error())
		if a.db != nil {
			_ = a.db.LogTrade(ticker, "sell", quantity, price, rejectedStatusFromError(err))
		}
		a.invalidateStatsCache()
		return false
	}
	if ok, why := a.broker.ValidateFuturesSymbolForAccount(context.Background(), ticker); !ok {
		a.noteFuturesValidationResult(ticker, false, why)
		sym := a.broker.OrderSymbolForAPI(ticker)
		a.registerBrokerValidationFailure(ticker, fmt.Errorf("finam asset: %s", why))
		a.addRuntimeEvent("warn", "order.futures_finam_getasset_reject", fmt.Sprintf("%s finam_sym=%s %s", ticker, sym, why))
		if a.db != nil {
			_ = a.db.LogTrade(ticker, "sell", quantity, price, rejectedStatusFromError(fmt.Errorf("finam asset: %s", why)))
		}
		a.invalidateStatsCache()
		return false
	}
	a.noteFuturesValidationResult(ticker, true, "")
	if ok, reason := a.isTradingAllowedForTicker(ticker); !ok {
		err := fmt.Errorf("outside trading session: %s", reason)
		fmt.Printf("🛑 Session blocked order: %v\n", err)
		if a.db != nil {
			_ = a.db.LogTrade(ticker, "sell", quantity, price, rejectedStatusFromError(err))
		}
		a.invalidateStatsCache()
		return false
	}
	if pending, pendingID := a.hasPendingOrder(ticker, "sell"); pending {
		msg := fmt.Sprintf("pending sell order exists for %s", ticker)
		if pendingID != "" {
			msg = fmt.Sprintf("%s (order_id=%s)", msg, pendingID)
		}
		a.addRuntimeEvent("warn", "auto.skip_pending_order", msg)
		a.invalidateStatsCache()
		return false
	}
	if err := a.allowOrderByThrottle(ticker, "sell"); err != nil {
		fmt.Printf("🛑 Order throttled: %v\n", err)
		if a.db != nil {
			_ = a.db.LogTrade(ticker, "sell", quantity, price, rejectedStatusFromError(err))
		}
		a.invalidateStatsCache()
		return false
	}

	normalizedQty, qErr := a.normalizeOrderQuantity(ticker, quantity, "sell")
	if qErr != nil {
		fmt.Printf("🛑 Quantity rejected: %v\n", qErr)
		if a.db != nil {
			_ = a.db.LogTrade(ticker, "sell", quantity, price, rejectedStatusFromError(qErr))
		}
		a.invalidateStatsCache()
		return false
	}
	if normalizedQty != quantity {
		fmt.Printf("ℹ️ Quantity normalized for futures sell: %s %d -> %d\n", ticker, quantity, normalizedQty)
	}
	quantity = normalizedQty

	if a.risk != nil {
		estPrice := price
		if estPrice <= 0 {
			estPrice, _ = a.moex.GetCurrentPrice(ticker)
		}
		notional := float64(quantity) * estPrice
		if err := a.risk.Allow(engine.OrderIntent{
			Symbol:   ticker,
			Side:     engine.SideSell,
			Qty:      float64(quantity),
			Price:    estPrice,
			Notional: notional,
			Time:     time.Now(),
		}); err != nil {
			fmt.Printf("🛑 Risk rejected futures order: %v\n", err)
			a.armRiskDailyPauseFromError(err)
			if a.db != nil {
				_ = a.db.LogTrade(ticker, "sell", quantity, estPrice, rejectedStatusFromError(err))
			}
			a.invalidateStatsCache()
			return false
		}
	}

	if err := a.validateOrderInput(ticker, quantity, price, price > 0, true); err != nil {
		fmt.Printf("🛑 Futures order precheck failed: %v\n", err)
		if a.db != nil {
			_ = a.db.LogTrade(ticker, "sell", quantity, price, rejectedStatusFromError(err))
		}
		a.invalidateStatsCache()
		return false
	}

	ctx := context.Background()
	var orderId string
	var err error

	if price > 0 {
		orderId, err = a.broker.SellLimit(ctx, ticker, quantity, price)
	} else {
		orderId, err = a.broker.SellMarket(ctx, ticker, quantity)
	}

	if err != nil {
		fmt.Printf("❌ Futures order failed: %v\n", err)
		a.addRuntimeEvent("error", "order.futures_sell_failed", fmt.Sprintf("%s qty=%d err=%v", ticker, quantity, err))
		a.registerBrokerValidationFailure(ticker, err)
		if a.db != nil {
			_ = a.db.LogTrade(ticker, "sell", quantity, price, rejectedStatusFromError(err))
		}
		a.invalidateStatsCache()
		return false
	}
	fmt.Printf("✅ Futures order placed: %s\n", orderId)
	a.addRuntimeEvent("info", "order.futures_sell_placed", fmt.Sprintf("%s qty=%d order_id=%s", ticker, quantity, orderId))
	if a.db != nil {
		execPrice := price
		if execPrice <= 0 {
			execPrice, _ = a.moex.GetCurrentPrice(ticker)
		}
		_ = a.db.LogTrade(ticker, "sell", quantity, execPrice, "placed")
	}
	a.invalidateStatsCache()
	return true
}

// GetAccounts возвращает список счетов
func (a *App) GetAccounts() []services.Account {
	if a.broker == nil {
		return []services.Account{}
	}
	return a.broker.GetAccounts()
}

// SwitchAccount переключает счёт (у нас только один)
func (a *App) SwitchAccount(accountName string) bool {
	if a.broker == nil {
		return false
	}
	err := a.broker.SwitchAccount(accountName)
	if err != nil {
		fmt.Printf("❌ Failed to switch account: %v\n", err)
		return false
	}
	a.invalidateStatsCache()
	return true
}

// GetCurrentAccount возвращает текущий счёт
func (a *App) GetCurrentAccount() string {
	if a.broker == nil {
		return ""
	}
	return a.broker.GetCurrentAccount()
}

// SubscribeToRealtimeQuotes подписывается на реальные котировки
func (a *App) SubscribeToRealtimeQuotes(ticker string) chan float64 {
	if a.broker == nil {
		return nil
	}

	ch := make(chan float64, 100)

	err := a.broker.SubscribeToQuotes(ticker, func(t string, price float64) {
		select {
		case ch <- price:
		default:
		}
	})

	if err != nil {
		fmt.Printf("❌ Failed to subscribe to %s: %v\n", ticker, err)
		return nil
	}

	return ch
}

// UnsubscribeFromRealtimeQuotes отписывается от котировок
func (a *App) UnsubscribeFromRealtimeQuotes(ticker string) {
	if a.broker == nil {
		return
	}
	a.broker.UnsubscribeFromQuotes(ticker)
}

// GetAgentStatus возвращает статус AI агента
func (a *App) GetAgentStatus() map[string]interface{} {
	if a.agent == nil {
		return map[string]interface{}{"active": false}
	}
	return map[string]interface{}{
		"active":      a.agent.IsActive(),
		"last_signal": a.agent.GetLastSignal(),
	}
}

// GetRiskStatus возвращает текущие лимиты и состояние risk manager
func (a *App) GetRiskStatus() map[string]interface{} {
	if a.risk == nil {
		return map[string]interface{}{"enabled": false}
	}
	return map[string]interface{}{
		"status": a.risk.Status(time.Now()),
	}
}

// SetEmergencyStop включает/выключает аварийную остановку (блокирует отправку заявок)
func (a *App) SetEmergencyStop(stop bool) bool {
	if a.risk == nil {
		return false
	}
	a.risk.SetEmergencyStop(stop)
	return true
}

// SetRiskLimits задаёт лимиты риска (0 => без лимита)
func (a *App) SetRiskLimits(maxOrderNotional float64, maxDailyNotional float64) bool {
	if a.risk == nil {
		return false
	}
	a.risk.SetConfig(engine.RiskConfig{
		Enabled:          true,
		MaxOrderNotional: maxOrderNotional,
		MaxDailyNotional: maxDailyNotional,
	})
	return true
}

// SetAgentActive включает/выключает агента
func (a *App) SetAgentActive(active bool) bool {
	if a.agent == nil {
		return false
	}
	a.agent.SetActive(active)
	return true
}

// AnalyzeWithAgent анализирует инструмент
func (a *App) AnalyzeWithAgent(ticker string) map[string]interface{} {
	if a.agent == nil || !a.agent.IsActive() {
		return map[string]interface{}{"error": "Agent not active"}
	}

	price, _ := a.moex.GetCurrentPrice(ticker)

	indicators := map[string]float64{}
	if closes, err := a.moex.GetClosePrices(ticker, 60); err == nil && len(closes) > 0 {
		indicators["rsi"] = services.RSI(closes, 14)
		if len(closes) >= 20 {
			indicators["sma20"] = services.SMA(closes[len(closes)-20:])
		}
		if len(closes) >= 50 {
			indicators["sma50"] = services.SMA(closes[len(closes)-50:])
		}
		if len(closes) >= 35 {
			macd, signal, hist := services.MACD(closes, 12, 26, 9)
			indicators["macd"] = macd
			indicators["macd_signal"] = signal
			indicators["macd_hist"] = hist
		}
		if len(closes) >= 15 {
			indicators["volatility_pct"] = services.VolatilityPctByCloses(closes, 14)
		}
	}

	var position *services.Position
	if a.db != nil {
		positions, _ := a.db.GetPositions()
		for _, p := range positions {
			if p.Ticker == ticker {
				position = &p
				break
			}
		}
	}

	balance := 100000.0
	if a.broker != nil {
		if b, err := a.broker.GetBalance(context.Background()); err == nil && b > 0 {
			balance = b
		}
	}

	ctx := &services.MarketContext{
		Ticker:           ticker,
		CurrentPrice:     price,
		Position:         position,
		Balance:          balance,
		Indicators:       indicators,
		IsFutures:        isLikelyFuturesTickerLocal(ticker),
		RoundTripFeePct:  services.FinamN3MOEXStockRoundTripPct(),
		MinStockNotional: services.AgentMinStockNotionalRub(),
	}

	signal, err := a.agent.Analyze(context.Background(), ctx)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	return map[string]interface{}{
		"action":     signal.Action,
		"quantity":   signal.Quantity,
		"price":      signal.Price,
		"confidence": signal.Confidence,
		"reasoning":  signal.Reasoning,
	}
}

// GetStatistics возвращает статистику
func (a *App) GetStatistics() map[string]interface{} {
	a.statsMu.Lock()
	if a.statsCached != nil && !a.statsCachedAt.IsZero() && time.Since(a.statsCachedAt) < a.statsCacheTTL {
		cached := cloneStatsMap(a.statsCached)
		cached["stats_cached"] = true
		cached["stats_updated_at"] = a.statsCachedAt.Format(time.RFC3339)
		a.statsMu.Unlock()
		return cached
	}
	a.statsMu.Unlock()

	stats := map[string]interface{}{
		"total_value":          0.0,
		"total_pnl":            0.0,
		"positions_count":      0,
		"return_pct":           0.0,
		"cash_balance":         0.0,
		"cash_balance_error":   "",
		"account_id":           "",
		"stats_cached":         false,
		"stats_updated_at":     "",
		"market_tz":            "Asia/Novosibirsk",
		"stock_session_open":   false,
		"stock_session_note":   "",
		"futures_session_open": false,
		"futures_session_note": "",
		"market_time_local":    "",
	}
	stockOpen, stockReason, futuresOpen, futuresReason, _, localClock := a.getSessionSnapshot()
	stats["stock_session_open"] = stockOpen
	stats["stock_session_note"] = stockReason
	stats["futures_session_open"] = futuresOpen
	stats["futures_session_note"] = futuresReason
	stats["market_time_local"] = localClock

	if a.broker != nil {
		ctx := context.Background()
		stats["account_id"] = a.broker.GetCurrentAccount()
		cashBalance, balErr := a.broker.GetBalance(ctx)
		if balErr == nil {
			stats["cash_balance"] = cashBalance
		} else {
			stats["cash_balance_error"] = balErr.Error()
		}
		positions, err := a.broker.GetPortfolio(ctx)
		if err == nil {
			totalValue := 0.0
			totalPnL := 0.0
			count := 0

			for _, p := range positions {
				if p.Quantity > 0 {
					currentPrice := p.CurrentPrice
					// При fallback REST у Finam может не прийти current_price/pnl.
					// Чтобы не "прыгали" метрики, локально восстанавливаем оценку.
					if currentPrice <= 0 && a.moex != nil {
						if px, pxErr := a.moex.GetCurrentPrice(p.Ticker); pxErr == nil && px > 0 {
							currentPrice = px
						}
					}
					if currentPrice <= 0 {
						currentPrice = p.AvgPrice
					}
					pnl := p.PnL
					if pnl == 0 && p.AvgPrice > 0 {
						pnl = float64(p.Quantity) * (currentPrice - p.AvgPrice)
					}
					totalValue += float64(p.Quantity) * currentPrice
					totalPnL += pnl
					count++
				}
			}

			totalWithCash := totalValue + cashBalance
			stats["total_value"] = totalWithCash
			stats["total_pnl"] = totalPnL
			stats["positions_count"] = count
			if totalWithCash > 0 {
				stats["return_pct"] = (totalPnL / totalWithCash) * 100
			}
			a.statsMu.Lock()
			a.statsCached = cloneStatsMap(stats)
			a.statsCachedAt = time.Now()
			a.statsMu.Unlock()
			stats["stats_cached"] = false
			stats["stats_updated_at"] = time.Now().Format(time.RFC3339)
			return stats
		}
	}

	if a.db != nil {
		positions, _ := a.db.GetPositions()
		totalValue := 0.0
		totalPnL := 0.0
		count := 0

		for _, p := range positions {
			if p.Quantity > 0 {
				totalValue += float64(p.Quantity) * p.CurrentPrice
				totalPnL += p.PnL
				count++
			}
		}

		stats["total_value"] = totalValue
		stats["total_pnl"] = totalPnL
		stats["positions_count"] = count
		if totalValue > 0 {
			stats["return_pct"] = (totalPnL / totalValue) * 100
		}
	}

	a.statsMu.Lock()
	a.statsCached = cloneStatsMap(stats)
	a.statsCachedAt = time.Now()
	a.statsMu.Unlock()
	stats["stats_cached"] = false
	stats["stats_updated_at"] = time.Now().Format(time.RFC3339)

	return stats
}

func (a *App) refreshAgentFromJournal() {
	if a.agent == nil || a.db == nil {
		return
	}
	trades, err := a.db.GetRecentTrades(200)
	if err != nil || len(trades) < 4 {
		return
	}

	type pos struct {
		qty int
		avg float64
	}
	byTicker := map[string]pos{}
	wins := 0
	losses := 0
	realized := 0.0

	// идем от старых к новым, чтобы корректно считать среднюю.
	for i := len(trades) - 1; i >= 0; i-- {
		t := trades[i]
		p := byTicker[t.Ticker]
		switch t.Side {
		case "buy":
			newQty := p.qty + t.Quantity
			if newQty > 0 {
				p.avg = (p.avg*float64(p.qty) + t.Price*float64(t.Quantity)) / float64(newQty)
				p.qty = newQty
			}
		case "sell":
			if p.qty <= 0 {
				continue
			}
			q := t.Quantity
			if q > p.qty {
				q = p.qty
			}
			pnl := (t.Price - p.avg) * float64(q)
			realized += pnl
			if pnl >= 0 {
				wins++
			} else {
				losses++
			}
			p.qty -= q
		}
		byTicker[t.Ticker] = p
	}

	total := wins + losses
	if total == 0 {
		return
	}
	winRate := float64(wins) / float64(total)
	a.agent.ApplyJournalFeedback(winRate, realized)
}

func isRejectedStatus(status string) bool {
	s := strings.ToLower(strings.TrimSpace(status))
	return strings.Contains(s, "rejected") || strings.Contains(s, "invalid_argument") || strings.Contains(s, "invalid quantity value")
}

func (a *App) getTickerRejectStats(limit int) map[string]TickerRejectStats {
	out := map[string]TickerRejectStats{}
	if a.db == nil {
		return out
	}
	if limit <= 0 {
		limit = 3000
	}
	trades, err := a.db.GetRecentTrades(limit)
	if err != nil || len(trades) == 0 {
		return out
	}
	now := time.Now()
	cut24 := now.Add(-24 * time.Hour)
	cut7d := now.Add(-7 * 24 * time.Hour)
	for _, t := range trades {
		tk := strings.ToUpper(strings.TrimSpace(t.Ticker))
		if tk == "" {
			continue
		}
		s := out[tk]
		s.Ticker = tk
		isRej := isRejectedStatus(t.Status)
		if t.ExecutedAt.After(cut7d) {
			s.Total7d++
			if isRej {
				s.Rejected7d++
			}
		}
		if t.ExecutedAt.After(cut24) {
			s.Total24h++
			if isRej {
				s.Rejected24h++
			}
		}
		out[tk] = s
	}
	for tk, s := range out {
		if s.Total24h > 0 {
			s.RejectRate24h = float64(s.Rejected24h) / float64(s.Total24h)
		}
		if s.Total7d > 0 {
			s.RejectRate7d = float64(s.Rejected7d) / float64(s.Total7d)
		}
		out[tk] = s
	}
	return out
}

func (a *App) rejectPenaltyScore(stats TickerRejectStats) float64 {
	pen := 0.0
	if stats.Total24h >= 3 {
		pen -= stats.RejectRate24h * 0.35
	}
	if stats.Total7d >= 6 {
		pen -= stats.RejectRate7d * 0.20
	}
	if pen < -0.45 {
		pen = -0.45
	}
	return pen
}

func (a *App) getTickerLearningScores(limit int) map[string]float64 {
	out := map[string]float64{}
	counts := map[string]int{}
	if a.db == nil {
		return out
	}
	if limit <= 0 {
		limit = 300
	}
	trades, err := a.db.GetRecentTrades(limit)
	if err != nil || len(trades) == 0 {
		return out
	}
	minTrades := envInt("AGENT_LEARNING_MIN_TRADES_PER_TICKER", 6)
	if minTrades < 1 {
		minTrades = 1
	}
	if minTrades > 40 {
		minTrades = 40
	}
	for i, t := range trades {
		w := 1.0 - (float64(i) / float64(len(trades)+1))
		if w < 0.2 {
			w = 0.2
		}
		tk := tickerLearningKey(t.Ticker)
		if tk == "" {
			continue
		}
		counts[tk]++
		s := strings.ToLower(strings.TrimSpace(t.Status))
		switch {
		case strings.Contains(s, "placed") || strings.Contains(s, "executed"):
			out[tk] += 0.10 * w
		case strings.Contains(s, "invalid quantity value"), strings.Contains(s, "неверное значение параметра 'количество'"):
			out[tk] -= 0.55 * w
		case strings.Contains(s, "rejected"):
			out[tk] -= 0.12 * w
		default:
			out[tk] -= 0.03 * w
		}
	}
	for tk, v := range out {
		if c := counts[tk]; c < minTrades {
			v *= float64(c) / float64(minTrades)
		}
		if v > 1 {
			v = 1
		}
		if v < -1 {
			v = -1
		}
		out[tk] = v
	}
	return out
}

// getTickerPnLScores строит мягкий score по реализованному PnL на тикер.
// Положительный PnL слегка повышает приоритет, отрицательный — снижает.
// Диапазон score ограничен [-1..1] и применяется в autoLoop через AGENT_PNL_SCORE_WEIGHT.
func (a *App) getTickerPnLScores(limit int) map[string]float64 {
	out := map[string]float64{}
	if a.db == nil {
		return out
	}
	if limit <= 0 {
		limit = 500
	}
	trades, err := a.db.GetRecentTrades(limit)
	if err != nil || len(trades) == 0 {
		return out
	}
	type pos struct {
		qty int
		avg float64
	}
	posByTicker := map[string]pos{}
	pnlByTicker := map[string]float64{}
	turnoverByTicker := map[string]float64{}
	tradeRowsByTicker := map[string]int{}

	for i := len(trades) - 1; i >= 0; i-- {
		t := trades[i]
		tk := canonicalTickerKey(t.Ticker)
		if tk == "" {
			tk = strings.ToUpper(strings.TrimSpace(t.Ticker))
		}
		if tk == "" || t.Quantity <= 0 || t.Price <= 0 {
			continue
		}
		tradeRowsByTicker[tk]++
		p := posByTicker[tk]
		side := strings.ToLower(strings.TrimSpace(t.Side))
		switch side {
		case "buy":
			if p.qty < 0 {
				cover := minInt(t.Quantity, -p.qty)
				pnlByTicker[tk] += (p.avg - t.Price) * float64(cover)
				turnoverByTicker[tk] += t.Price * float64(cover)
				p.qty += cover
				rest := t.Quantity - cover
				if p.qty == 0 {
					p.avg = 0
				}
				if rest > 0 {
					if p.qty > 0 {
						total := p.avg*float64(p.qty) + t.Price*float64(rest)
						p.qty += rest
						p.avg = total / float64(p.qty)
					} else {
						p.qty = rest
						p.avg = t.Price
					}
				}
			} else {
				total := p.avg*float64(p.qty) + t.Price*float64(t.Quantity)
				p.qty += t.Quantity
				p.avg = total / float64(p.qty)
			}
		case "sell":
			if p.qty > 0 {
				closeQty := minInt(t.Quantity, p.qty)
				pnlByTicker[tk] += (t.Price - p.avg) * float64(closeQty)
				turnoverByTicker[tk] += t.Price * float64(closeQty)
				p.qty -= closeQty
				rest := t.Quantity - closeQty
				if p.qty == 0 {
					p.avg = 0
				}
				if rest > 0 {
					if p.qty < 0 {
						totalShort := p.avg*float64(-p.qty) + t.Price*float64(rest)
						p.qty -= rest
						p.avg = totalShort / float64(-p.qty)
					} else {
						p.qty = -rest
						p.avg = t.Price
					}
				}
			} else {
				totalShort := p.avg*float64(-p.qty) + t.Price*float64(t.Quantity)
				p.qty -= t.Quantity
				p.avg = totalShort / float64(-p.qty)
			}
		}
		posByTicker[tk] = p
	}

	minPnlRows := envInt("AGENT_PNL_SCORE_MIN_TRADES", 8)
	if minPnlRows < 1 {
		minPnlRows = 1
	}
	if minPnlRows > 80 {
		minPnlRows = 80
	}
	for tk, pnl := range pnlByTicker {
		turn := turnoverByTicker[tk]
		if turn <= 0 {
			continue
		}
		edge := pnl / turn // относительная реализованная доходность в истории тикера
		score := edge * 18.0
		if rows := tradeRowsByTicker[tk]; rows < minPnlRows {
			score *= float64(rows) / float64(minPnlRows)
		}
		if score > 1 {
			score = 1
		}
		if score < -1 {
			score = -1
		}
		out[tk] = score
	}
	return out
}

// GetRejectRateTable возвращает сводную таблицу reject-rate по тикерам (24h/7d).
func (a *App) GetRejectRateTable(limit int) []TickerRejectStats {
	stats := a.getTickerRejectStats(5000)
	out := make([]TickerRejectStats, 0, len(stats))
	for _, s := range stats {
		if s.Total24h == 0 && s.Total7d == 0 {
			continue
		}
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool {
		li := out[i].RejectRate24h*0.7 + out[i].RejectRate7d*0.3
		lj := out[j].RejectRate24h*0.7 + out[j].RejectRate7d*0.3
		if li == lj {
			return out[i].Ticker < out[j].Ticker
		}
		return li > lj
	})
	if limit <= 0 {
		limit = 30
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

// extractStockTickerFromFutures извлекает тикер акции из тикера фьючерса
// Например: "AFZ6@RTSX" → "AFLT", "MTZ6@RTSX" → "MTSS"
func (a *App) extractStockTickerFromFutures(futuresTicker string) string {
	// Таблица обратного маппинга фьючерсов → акции
	futuresToStock := map[string]string{
		"AFZ6": "AFLT",  // Аэрофлот
		"GZM6": "GAZP",  // Газпром
		"LKZ6": "LKOH",  // Лукойл
		"NVZ6": "NVTK",  // Новатэк
		"RNZ6": "ROSN",  // Роснефть
		"SBZ6": "SBER",  // Сбербанк
		"VBZ6": "VTBR",  // ВТБ
		"MNZ6": "MGNT",  // Магнит
		"MTZ6": "MTSS",  // МТС
		"TNZ6": "TATN",  // Татнефть
	}
	
	// Убираем @RTSX и т.д.
	baseTicker := strings.Split(futuresTicker, "@")[0]
	baseTicker = strings.ToUpper(baseTicker)
	
	if stockTicker, ok := futuresToStock[baseTicker]; ok {
		return stockTicker
	}
	
	return ""
}

// getPositionQuantity возвращает количество акций в портфеле
func (a *App) getPositionQuantity(ticker string) int {
	if a.db == nil {
		return 0
	}
	positions, err := a.db.GetPositions()
	if err != nil {
		return 0
	}
	for _, p := range positions {
		if strings.EqualFold(p.Ticker, ticker) || strings.EqualFold(p.Ticker, ticker+"@MISX") {
			return p.Quantity
		}
	}
	return 0
}

// tryShortFuturesFallback пытается шортить фьючерс вместо акции при ошибке
func (a *App) tryShortFuturesFallback(stockTicker string, quantity int, price float64) bool {
	// Ищем фьючерс того же эмитента
	futuresTicker := a.findFuturesForStock(stockTicker)
	if futuresTicker == "" {
		a.addRuntimeEvent("warn", "short_fallback.no_futures", fmt.Sprintf("%s — фьючерс не найден", stockTicker))
		return false
	}
	
	a.addRuntimeEvent("info", "short_fallback.trying", fmt.Sprintf("%s → %s (фьючерс вместо акции)", stockTicker, futuresTicker))

	// Пробуем шортить фьючерс
	// Убираем суффикс @RTSX для запроса цены из MOEX ISS
	priceTicker := strings.Split(futuresTicker, "@")[0]
	futuresPrice, err := a.moex.GetCurrentPrice(priceTicker)
	if err != nil || futuresPrice <= 0 {
		a.addRuntimeEvent("warn", "short_fallback.no_price", fmt.Sprintf("%s — не удалось получить цену", futuresTicker))
		return false
	}
	
	// Открываем шорт по фьючерсу
	success := a.OpenShortPosition(futuresTicker, quantity, futuresPrice)
	if success {
		a.addRuntimeEvent("info", "short_fallback.success", fmt.Sprintf("%s → %s qty=%d price=%.2f", stockTicker, futuresTicker, quantity, futuresPrice))
	} else {
		a.addRuntimeEvent("warn", "short_fallback.failed", fmt.Sprintf("%s → %s — не удалось открыть шорт", stockTicker, futuresTicker))
	}
	
	return success
}

// findFuturesForStock ищет фьючерс того же эмитента по таблице маппинга
func (a *App) findFuturesForStock(stockTicker string) string {
	// Таблица маппинга акций → фьючерсов (будет расширяться автоматически)
	stockToFutures := map[string]string{
		"AFLT": "AFZ6@RTSX",  // Аэрофлот
		"GAZP": "GZM6@RTSX",  // Газпром
		"LKOH": "LKZ6@RTSX",  // Лукойл
		"NVTK": "NVZ6@RTSX",  // Новатэк
		"ROSN": "RNZ6@RTSX",  // Роснефть
		"SBER": "SBZ6@RTSX",  // Сбербанк
		"VTBR": "VBZ6@RTSX",  // ВТБ
		"MGNT": "MNZ6@RTSX",  // Магнит
		"MTSS": "MTZ6@RTSX",  // МТС
		"TATN": "TNZ6@RTSX",  // Татнефть
	}
	
	// Нормализуем тикер (убираем @MISX и т.д.)
	normalizedTicker := strings.Split(stockTicker, "@")[0]
	normalizedTicker = strings.ToUpper(normalizedTicker)
	
	if futuresTicker, ok := stockToFutures[normalizedTicker]; ok {
		return futuresTicker
	}
	
	return ""
}
