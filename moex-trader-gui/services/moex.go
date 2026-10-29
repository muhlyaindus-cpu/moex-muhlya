package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

type MOEXClient struct {
	httpClient *http.Client
	baseURL    string
	lotMu      sync.Mutex
	lotCache   map[string]int
	// Кэш SHORTNAME для фьючерсов: SECID → SHORTNAME (например, "MTZ6" → "MTSI-12.26")
	shortNameMu   sync.RWMutex
	shortNameCache map[string]string
	// Кэш DECIMALS (количество знаков после запятой в цене) для форматирования цены ордера
	decimalsMu    sync.Mutex
	decimalsCache map[string]int
}

type Security struct {
	Ticker      string     `json:"ticker"`
	Name        string     `json:"name"`
	Price       float64    `json:"price"`
	NumTrades   int        `json:"num_trades,omitempty"`
	TurnoverRub float64    `json:"turnover_rub,omitempty"`
	ExpiryDate  *time.Time `json:"expiry_date,omitempty"` // Дата экспирации для фьючерсов
}

func NewMOEXClient() *MOEXClient {
	return &MOEXClient{
		httpClient:     &http.Client{Timeout: 10 * time.Second},
		baseURL:        "https://iss.moex.com/iss",
		lotCache:       make(map[string]int),
		shortNameCache: make(map[string]string),
		decimalsCache:  make(map[string]int),
	}
}

func normalizeTickerForMOEX(ticker string) string {
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

// SetShortNameCache сохраняет SHORTNAME для SECID (используется для маппинга тикеров фьючерсов)
func (c *MOEXClient) SetShortNameCache(secid, shortName string) {
	if secid == "" || shortName == "" {
		return
	}
	c.shortNameMu.Lock()
	c.shortNameCache[secid] = shortName
	c.shortNameMu.Unlock()
}

// GetShortName возвращает SHORTNAME для SECID (например, "MTZ6" → "MTSI-12.26")
func (c *MOEXClient) GetShortName(secid string) string {
	c.shortNameMu.RLock()
	defer c.shortNameMu.RUnlock()
	return c.shortNameCache[secid]
}

// GetLotSize возвращает размер лота в штуках для инструмента.
// Если не удалось определить, возвращает 1.
func (c *MOEXClient) GetLotSize(ticker string) int {
	ticker = normalizeTickerForMOEX(ticker)
	if ticker == "" {
		return 1
	}

	c.lotMu.Lock()
	if lot, ok := c.lotCache[ticker]; ok && lot > 0 {
		c.lotMu.Unlock()
		return lot
	}
	c.lotMu.Unlock()

	stockPath := fmt.Sprintf("%s/engines/stock/markets/shares/securities/%s.json?iss.meta=off&securities.columns=BOARDID,LOTSIZE", c.baseURL, ticker)
	if lot, err := c.getLotSizeFromPath(stockPath, "TQBR", "TQTF", "TQTD", "SMAL"); err == nil && lot > 0 {
		c.lotMu.Lock()
		c.lotCache[ticker] = lot
		c.lotMu.Unlock()
		return lot
	}

	futuresPath := fmt.Sprintf("%s/engines/futures/markets/forts/securities/%s.json?iss.meta=off&securities.columns=BOARDID,LOTSIZE", c.baseURL, ticker)
	if lot, err := c.getLotSizeFromPath(futuresPath, "RFUD", "ROPD"); err == nil && lot > 0 {
		c.lotMu.Lock()
		c.lotCache[ticker] = lot
		c.lotMu.Unlock()
		return lot
	}

	// Не кэшируем fallback=1: временные сетевые/ISS сбои не должны "залипать"
	// и ломать кратность количества на весь аптайм процесса.
	return 1
}

// GetDecimals возвращает количество знаков после запятой в цене инструмента (DECIMALS из MOEX ISS).
// Используется для правильного форматирования цены лимитного ордера.
// Если не удалось определить, возвращает 2 (безопасный дефолт для большинства акций MOEX).
func (c *MOEXClient) GetDecimals(ticker string) int {
	ticker = normalizeTickerForMOEX(ticker)
	if ticker == "" {
		return 2
	}

	c.decimalsMu.Lock()
	if d, ok := c.decimalsCache[ticker]; ok && d >= 0 {
		c.decimalsMu.Unlock()
		return d
	}
	c.decimalsMu.Unlock()

	// Акции
	stockPath := fmt.Sprintf("%s/engines/stock/markets/shares/securities/%s.json?iss.meta=off&securities.columns=BOARDID,DECIMALS", c.baseURL, ticker)
	if d, err := c.getDecimalsFromPath(stockPath, "TQBR", "TQTF", "TQTD", "SMAL"); err == nil && d >= 0 {
		c.decimalsMu.Lock()
		c.decimalsCache[ticker] = d
		c.decimalsMu.Unlock()
		return d
	}

	// Фьючерсы
	futuresPath := fmt.Sprintf("%s/engines/futures/markets/forts/securities/%s.json?iss.meta=off&securities.columns=BOARDID,DECIMALS", c.baseURL, ticker)
	if d, err := c.getDecimalsFromPath(futuresPath, "RFUD", "ROPD"); err == nil && d >= 0 {
		c.decimalsMu.Lock()
		c.decimalsCache[ticker] = d
		c.decimalsMu.Unlock()
		return d
	}

	return 2
}

// getDecimalsFromPath запрашивает DECIMALS из MOEX ISS для указанной доски.
func (c *MOEXClient) getDecimalsFromPath(path string, primaryBoard string, altBoards ...string) (int, error) {
	resp, err := c.httpClient.Get(path)
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return -1, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return -1, err
	}

	secBlock, _ := result["securities"].(map[string]interface{})
	colsAny, ok := secBlock["columns"].([]interface{})
	if !ok {
		return -1, fmt.Errorf("securities.columns not found")
	}
	dataAny, ok := secBlock["data"].([]interface{})
	if !ok || len(dataAny) == 0 {
		return -1, fmt.Errorf("securities.data not found")
	}

	boardIdx := findColumnIndex(colsAny, "BOARDID")
	decimalsIdx := findColumnIndex(colsAny, "DECIMALS")
	if decimalsIdx < 0 {
		return -1, fmt.Errorf("DECIMALS column not found")
	}

	// Сначала ищем primaryBoard, затем altBoards
	boards := append([]string{primaryBoard}, altBoards...)
	for _, targetBoard := range boards {
		for _, rowAny := range dataAny {
			row, _ := rowAny.([]interface{})
			if decimalsIdx >= len(row) {
				continue
			}
			board := ""
			if boardIdx >= 0 && boardIdx < len(row) {
				board = moexToString(row[boardIdx])
			}
			if board != targetBoard {
				continue
			}
			d := int(moexToFloat(row[decimalsIdx]))
			if d >= 0 {
				return d, nil
			}
		}
	}

	// Если ни одна доска не найдена, берём первое валидное значение
	for _, rowAny := range dataAny {
		row, _ := rowAny.([]interface{})
		if decimalsIdx >= len(row) {
			continue
		}
		d := int(moexToFloat(row[decimalsIdx]))
		if d >= 0 {
			return d, nil
		}
	}

	return -1, fmt.Errorf("DECIMALS not found for any board")
}

func (c *MOEXClient) GetCurrentPrice(ticker string) (float64, error) {
	// Сначала пробуем акции, затем фьючерсы.
	price, err := c.getCurrentPriceFromPath(
		fmt.Sprintf("%s/engines/stock/markets/shares/securities/%s.json", c.baseURL, ticker),
	)
	if err == nil && price > 0 {
		return price, nil
	}
	return c.getCurrentPriceFromPath(
		fmt.Sprintf("%s/engines/futures/markets/forts/securities/%s.json", c.baseURL, ticker),
	)
}

// MarketSnapshotItem — снимок рыночных данных для одного тикера (для inline MIYA).
type MarketSnapshotItem struct {
	Ticker        string
	Last          float64
	Change        float64
	TurnoverRub   float64
	TradingActive bool    // true если торги идут (TRADINGSTATUS = "T" или 2)
	RSI14         float64 // RSI(14) — 0-100, <30 перепродан, >70 перекуплен
	TrendBull     bool    // true если SMA20 > SMA50 (восходящий тренд)
	SMA20         float64 // 20-дневная скользящая средняя
	SMA50         float64 // 50-дневная скользящая средняя
}

// GetMarketSnapshot возвращает LAST, CHANGE, VALTODAY, TRADINGSTATUS для списка тикеров (акции TQBR).
func (c *MOEXClient) GetMarketSnapshot(tickers []string) []MarketSnapshotItem {
	result := make([]MarketSnapshotItem, 0, len(tickers))
	for _, ticker := range tickers {
		t := normalizeTickerForMOEX(ticker)
		if t == "" {
			continue
		}
		u := fmt.Sprintf("%s/engines/stock/markets/shares/boards/TQBR/securities/%s.json?iss.meta=off&marketdata.columns=SECID,LAST,CHANGE,VALTODAY,TRADINGSTATUS", c.baseURL, t)
		resp, err := c.httpClient.Get(u)
		if err != nil {
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}
		var parsed map[string]interface{}
		if err := json.Unmarshal(body, &parsed); err != nil {
			continue
		}
		mdBlock, _ := parsed["marketdata"].(map[string]interface{})
		if mdBlock == nil {
			continue
		}
		cols, _ := mdBlock["columns"].([]interface{})
		data, _ := mdBlock["data"].([]interface{})
		if len(cols) == 0 || len(data) == 0 {
			continue
		}
		lastIdx := findColumnIndex(cols, "LAST")
		changeIdx := findColumnIndex(cols, "CHANGE")
		valIdx := findColumnIndex(cols, "VALTODAY")
		statusIdx := findColumnIndex(cols, "TRADINGSTATUS")
		for _, rowAny := range data {
			row, _ := rowAny.([]interface{})
			if lastIdx >= len(row) {
				continue
			}
			last := moexToFloat(row[lastIdx])
			if last <= 0 {
				continue
			}
			item := MarketSnapshotItem{
				Ticker: t,
				Last:   last,
			}
			if changeIdx >= 0 && changeIdx < len(row) {
				item.Change = moexToFloat(row[changeIdx])
			}
			if valIdx >= 0 && valIdx < len(row) {
				item.TurnoverRub = moexToFloat(row[valIdx])
			}
			if statusIdx >= 0 && statusIdx < len(row) {
				// TRADINGSTATUS может быть строкой "T" (торги идут) или числом 2
				statusVal := row[statusIdx]
				switch v := statusVal.(type) {
				case string:
					item.TradingActive = (v == "T" || v == "2")
				case float64:
					item.TradingActive = (v == 2)
				default:
					s := fmt.Sprintf("%v", statusVal)
					item.TradingActive = (s == "T" || s == "2")
				}
			}
			result = append(result, item)
		}
	}
	return result
}

// EnrichWithAnalytics загружает свечи для каждого тикера и вычисляет RSI(14), SMA20, SMA50, тренд.
// Модифицирует items inplace.
func (c *MOEXClient) EnrichWithAnalytics(items []MarketSnapshotItem) {
	for i := range items {
		// Загружаем 55 дневных свечей (нужно 50 для SMA50 + запас)
		closes, err := c.GetClosePrices(items[i].Ticker, 55)
		if err != nil || len(closes) < 20 {
			continue
		}

		// RSI(14)
		if len(closes) >= 15 {
			items[i].RSI14 = RSI(closes, 14)
		}

		// SMA20 и SMA50
		if len(closes) >= 20 {
			items[i].SMA20 = SMA(closes[len(closes)-20:])
		}
		if len(closes) >= 50 {
			items[i].SMA50 = SMA(closes[len(closes)-50:])
		}

		// Trend: SMA20 > SMA50 = бычий тренд
		if items[i].SMA20 > 0 && items[i].SMA50 > 0 {
			items[i].TrendBull = items[i].SMA20 > items[i].SMA50
		}
	}
}

// MarketRegime — состояние рынка для Мии-стратега.
type MarketRegime struct {
	IMOEX        float64 // индекс МосБиржи
	IMOEXChange  float64 // изменение IMOEX за день (%)
	Volatility   float64 // волатильность рынка (прокси)
	MarketBreadth float64 // % растущих тикеров (up_ratio)
}

// GetMarketRegime возвращает состояние рынка: IMOEX, изменение, волатильность.
func (c *MOEXClient) GetMarketRegime() MarketRegime {
	regime := MarketRegime{}

	// Получаем IMOEX
	url := fmt.Sprintf("%s/engines/stock/markets/index/securities/IMOEX.json?iss.meta=off&marketdata.columns=SECID,LAST,CHANGE", c.baseURL)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return regime
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return regime
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return regime
	}

	mdBlock, _ := result["marketdata"].(map[string]interface{})
	if mdBlock == nil {
		return regime
	}
	cols, _ := mdBlock["columns"].([]interface{})
	data, _ := mdBlock["data"].([]interface{})
	if len(cols) == 0 || len(data) == 0 {
		return regime
	}

	lastIdx := findColumnIndex(cols, "LAST")
	changeIdx := findColumnIndex(cols, "CHANGE")

	for _, rowAny := range data {
		row, _ := rowAny.([]interface{})
		if lastIdx >= 0 && lastIdx < len(row) {
			regime.IMOEX = moexToFloat(row[lastIdx])
		}
		if changeIdx >= 0 && changeIdx < len(row) {
			regime.IMOEXChange = moexToFloat(row[changeIdx])
		}
	}

	return regime
}

func tradingStatusFromMarketdataBlock(marketdata map[string]interface{}) (string, error) {
	colsAny, ok := marketdata["columns"].([]interface{})
	if !ok {
		return "", fmt.Errorf("marketdata.columns not found")
	}
	dataAny, ok := marketdata["data"].([]interface{})
	if !ok {
		return "", fmt.Errorf("marketdata.data not found")
	}
	if len(dataAny) == 0 {
		return "", fmt.Errorf("no marketdata rows")
	}

	tradingStatusIdx := findColumnIndex(colsAny, "TRADINGSTATUS")
	if tradingStatusIdx < 0 {
		return "", fmt.Errorf("TRADINGSTATUS column not found")
	}

	boardIdx := findColumnIndex(colsAny, "BOARDID")
	rowOrder := make([]int, 0, len(dataAny))
	if boardIdx >= 0 {
		for i, rowAny := range dataAny {
			row, ok := rowAny.([]interface{})
			if !ok || boardIdx >= len(row) {
				continue
			}
			if strings.EqualFold(strings.TrimSpace(moexToString(row[boardIdx])), "RFUD") {
				rowOrder = append(rowOrder, i)
				break
			}
		}
	}
	for i := range dataAny {
		if len(rowOrder) > 0 && i == rowOrder[0] {
			continue
		}
		rowOrder = append(rowOrder, i)
	}

	for _, idx := range rowOrder {
		rowAny := dataAny[idx]
		row, ok := rowAny.([]interface{})
		if !ok {
			continue
		}
		if tradingStatusIdx >= len(row) {
			continue
		}
		status := strings.TrimSpace(moexToString(row[tradingStatusIdx]))
		if status != "" {
			return status, nil
		}
	}
	return "", fmt.Errorf("empty TRADINGSTATUS")
}

// fortsRelaxedTradingStatus — для части FORTS (в т.ч. USDRUBF) ISS не отдаёт TRADINGSTATUS в marketdata,
// но есть TRADEDATE и метки времени котировки: по ним оцениваем «идёт ли сессия» (грубо, по дате МСК).
func fortsRelaxedTradingStatus(marketdata map[string]interface{}) (string, error) {
	colsAny, ok := marketdata["columns"].([]interface{})
	if !ok {
		return "", fmt.Errorf("marketdata.columns not found")
	}
	dataAny, ok := marketdata["data"].([]interface{})
	if !ok || len(dataAny) == 0 {
		return "", fmt.Errorf("no marketdata rows")
	}

	rowAny := dataAny[0]
	row, ok := rowAny.([]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected marketdata row format")
	}

	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		loc = time.UTC
	}
	today := time.Now().In(loc).Format("2006-01-02")

	d := fortsTradingDateFromRow(colsAny, row)
	if len(d) >= 10 {
		d = d[:10]
	}
	if d == "" {
		return "", fmt.Errorf("no TRADEDATE/TRADE_SESSION_DATE in relaxed mode")
	}
	if d != today {
		return "N", nil
	}

	sysIdx := findColumnIndex(colsAny, "SYSTIME")
	updIdx := findColumnIndex(colsAny, "UPDATETIME")
	timeIdx := findColumnIndex(colsAny, "TIME")
	active := false
	if sysIdx >= 0 && sysIdx < len(row) && strings.TrimSpace(moexToString(row[sysIdx])) != "" {
		active = true
	}
	if !active && updIdx >= 0 && updIdx < len(row) && strings.TrimSpace(moexToString(row[updIdx])) != "" {
		active = true
	}
	if !active && timeIdx >= 0 && timeIdx < len(row) && strings.TrimSpace(moexToString(row[timeIdx])) != "" {
		active = true
	}
	if !active {
		return "", fmt.Errorf("relaxed: no time fields")
	}
	return "T", nil
}

func fortsTradingDateFromRow(colsAny []interface{}, row []interface{}) string {
	for _, name := range []string{"TRADEDATE", "TRADE_SESSION_DATE"} {
		idx := findColumnIndex(colsAny, name)
		if idx < 0 || idx >= len(row) {
			continue
		}
		s := strings.TrimSpace(moexToString(row[idx]))
		if s != "" {
			return s
		}
	}
	return ""
}

func (c *MOEXClient) fetchTradingStatusFromURL(u string) (string, error) {
	resp, err := c.httpClient.Get(u)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	marketdata, ok := result["marketdata"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("marketdata block not found")
	}
	return tradingStatusFromMarketdataBlock(marketdata)
}

func (c *MOEXClient) fetchFortsTradingStatusFromURL(u string) (string, error) {
	resp, err := c.httpClient.Get(u)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	marketdata, ok := result["marketdata"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("marketdata block not found")
	}
	if s, err := tradingStatusFromMarketdataBlock(marketdata); err == nil {
		return s, nil
	}
	return fortsRelaxedTradingStatus(marketdata)
}

// GetTradingStatus получает TRADINGSTATUS для инструмента на акционном рынке.
// Используется как "аварийный стоп" для выходных/праздников и закрытых сессий.
func (c *MOEXClient) GetTradingStatus(ticker string) (string, error) {
	ticker = strings.ToUpper(strings.TrimSpace(ticker))
	if ticker == "" {
		return "", fmt.Errorf("empty ticker")
	}

	// Для акций берем статус с основной доски TQBR.
	// Иначе ISS может вернуть первый попавшийся board (например SPEQ) со статусом N,
	// что ложно закрывает рынок при активной сессии.
	tqbrURL := fmt.Sprintf("%s/engines/stock/markets/shares/boards/TQBR/securities/%s.json?iss.meta=off&iss.only=marketdata",
		c.baseURL, url.PathEscape(ticker))
	if status, err := c.fetchTradingStatusFromURL(tqbrURL); err == nil {
		return status, nil
	}

	// Fallback на старый путь без board, если по TQBR нет данных.
	u := fmt.Sprintf("%s/engines/stock/markets/shares/securities/%s.json?iss.meta=off&iss.only=marketdata",
		c.baseURL, url.PathEscape(ticker))
	return c.fetchTradingStatusFromURL(u)
}

// GetFortsTradingStatus — TRADINGSTATUS по срочному рынку (FORTS) для SECID MOEX (например SI-3.25, USDRUBF).
// Для инструментов без колонки TRADINGSTATUS (часто USDRUBF) используется дополнительная эвристика по TRADEDATE/времени.
func (c *MOEXClient) GetFortsTradingStatus(secid string) (string, error) {
	secid = strings.TrimSpace(secid)
	if secid == "" {
		return "", fmt.Errorf("empty secid")
	}
	u := fmt.Sprintf("%s/engines/futures/markets/forts/securities/%s.json?iss.meta=off&iss.only=marketdata",
		c.baseURL, url.PathEscape(secid))
	return c.fetchFortsTradingStatusFromURL(u)
}

// GetClosePrices возвращает последние N цен закрытия (shares candles, дневные).
func (c *MOEXClient) GetClosePrices(ticker string, days int) ([]float64, error) {
	return c.GetClosePricesByInterval(ticker, days, 24)
}

// GetClosePricesByInterval возвращает последние N цен закрытия по interval MOEX ISS.
// Частые интервалы: 1 (1m), 10 (10m), 60 (1h), 24 (1d), 7 (1w), 31 (1M), 4 (1Q).
func (c *MOEXClient) GetClosePricesByInterval(ticker string, count int, interval int) ([]float64, error) {
	if count <= 0 {
		return []float64{}, nil
	}
	if interval <= 0 {
		interval = 24
	}

	closes, err := c.getClosePricesFromPath(
		fmt.Sprintf("%s/engines/stock/markets/shares/securities/%s/candles.json", c.baseURL, ticker),
		count,
		interval,
	)
	if err == nil && len(closes) > 0 {
		return closes, nil
	}
	// fallback для фьючерсов
	return c.getClosePricesFromPath(
		fmt.Sprintf("%s/engines/futures/markets/forts/securities/%s/candles.json", c.baseURL, ticker),
		count,
		interval,
	)
}

func (c *MOEXClient) GetWatchlist() []Security {
	return c.GetStockInstruments("", 50)
}

func (c *MOEXClient) GetStockInstruments(query string, limit int) []Security {
	return c.getMarketInstruments(
		"/engines/stock/markets/shares/boards/TQBR/securities.json",
		query,
		limit,
	)
}

func (c *MOEXClient) GetFuturesInstruments(query string, limit int) []Security {
	return c.getMarketInstruments(
		"/engines/futures/markets/forts/securities.json",
		query,
		limit,
	)
}

func (c *MOEXClient) getCurrentPriceFromPath(path string) (float64, error) {
	resp, err := c.httpClient.Get(path)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, err
	}
	return extractLastPrice(result)
}

func (c *MOEXClient) getMarketInstruments(endpoint, query string, limit int) []Security {
	if limit <= 0 {
		limit = 50
	}
	if limit > 300 {
		limit = 300
	}

	u, _ := url.Parse(c.baseURL + endpoint)
	q := u.Query()
	q.Set("iss.meta", "off")
	// Добавляем EXPIRATIONDATE для фьючерсов
	if strings.Contains(endpoint, "futures") {
		q.Set("securities.columns", "SECID,SHORTNAME,EXPIRATIONDATE")
	} else {
		q.Set("securities.columns", "SECID,SHORTNAME")
	}
	q.Set("marketdata.columns", "SECID,LAST,NUMTRADES,VALTODAY")
	u.RawQuery = q.Encode()

	resp, err := c.httpClient.Get(u.String())
	if err != nil {
		return []Security{}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []Security{}
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return []Security{}
	}

	secBlock, _ := result["securities"].(map[string]interface{})
	secCols, _ := secBlock["columns"].([]interface{})
	secData, _ := secBlock["data"].([]interface{})
	if len(secCols) == 0 || len(secData) == 0 {
		return []Security{}
	}

	secidIdx := findColumnIndex(secCols, "SECID")
	nameIdx := findColumnIndex(secCols, "SHORTNAME")
	expiryIdx := findColumnIndex(secCols, "EXPIRATIONDATE")
	if secidIdx < 0 {
		return []Security{}
	}

	prices := map[string]float64{}
	marketBlock, _ := result["marketdata"].(map[string]interface{})
	mCols, _ := marketBlock["columns"].([]interface{})
	mData, _ := marketBlock["data"].([]interface{})
	mdSecidIdx := findColumnIndex(mCols, "SECID")
	mdLastIdx := findColumnIndex(mCols, "LAST")
	mdNumTradesIdx := findColumnIndex(mCols, "NUMTRADES")
	mdTurnoverIdx := findColumnIndex(mCols, "VALTODAY")
	numTrades := map[string]int{}
	turnovers := map[string]float64{}
	if mdSecidIdx >= 0 && mdLastIdx >= 0 {
		for _, rowAny := range mData {
			row, _ := rowAny.([]interface{})
			if mdSecidIdx >= len(row) || mdLastIdx >= len(row) {
				continue
			}
			secid := moexToString(row[mdSecidIdx])
			if secid == "" {
				continue
			}
			prices[secid] = moexToFloat(row[mdLastIdx])
			if mdNumTradesIdx >= 0 && mdNumTradesIdx < len(row) {
				numTrades[secid] = int(moexToFloat(row[mdNumTradesIdx]))
			}
			if mdTurnoverIdx >= 0 && mdTurnoverIdx < len(row) {
				turnovers[secid] = moexToFloat(row[mdTurnoverIdx])
			}
		}
	}

	query = strings.ToUpper(strings.TrimSpace(query))
	items := make([]Security, 0, limit)
	for _, rowAny := range secData {
		row, _ := rowAny.([]interface{})
		if secidIdx >= len(row) {
			continue
		}
		ticker := strings.TrimSpace(moexToString(row[secidIdx]))
		if ticker == "" {
			continue
		}

		name := ticker
		if nameIdx >= 0 && nameIdx < len(row) {
			n := strings.TrimSpace(moexToString(row[nameIdx]))
			if n != "" {
				name = n
			}
		}

		if query != "" {
			if !strings.Contains(strings.ToUpper(ticker), query) && !strings.Contains(strings.ToUpper(name), query) {
				continue
			}
		}

		items = append(items, Security{
			Ticker:      ticker,
			Name:        name,
			Price:       prices[ticker],
			NumTrades:   numTrades[ticker],
			TurnoverRub: turnovers[ticker],
			ExpiryDate:  parseExpiryDate(row, expiryIdx),
		})
		
		// Сохраняем SHORTNAME в кэш для маппинга тикеров фьючерсов
		if name != ticker {
			c.SetShortNameCache(ticker, name)
		}
		
		if len(items) >= limit {
			break
		}
	}

	return items
}

// parseExpiryDate парсит дату экспирации из строки MOEX ISS (формат: "2026-12-25" или пустая строка)
func parseExpiryDate(row []interface{}, expiryIdx int) *time.Time {
	if expiryIdx < 0 || expiryIdx >= len(row) {
		return nil
	}
	dateStr := moexToString(row[expiryIdx])
	if dateStr == "" {
		return nil
	}
	// MOEX ISS возвращает дату в формате "YYYY-MM-DD"
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil
	}
	return &t
}

// IsExpiringSoon возвращает true, если до экспирации меньше minDays дней
func (s *Security) IsExpiringSoon(minDays int) bool {
	if s.ExpiryDate == nil {
		return false // Не фьючерс — не проверяем
	}
	daysUntilExpiry := int(time.Until(*s.ExpiryDate).Hours() / 24)
	return daysUntilExpiry < minDays
}

func (c *MOEXClient) getLotSizeFromPath(path string, preferredBoards ...string) (int, error) {
	resp, err := c.httpClient.Get(path)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, err
	}

	secBlock, _ := result["securities"].(map[string]interface{})
	cols, _ := secBlock["columns"].([]interface{})
	data, _ := secBlock["data"].([]interface{})
	if len(cols) == 0 || len(data) == 0 {
		return 0, fmt.Errorf("no securities data")
	}

	lotIdx := findColumnIndex(cols, "LOTSIZE")
	if lotIdx < 0 {
		return 0, fmt.Errorf("lotsize not found")
	}
	boardIdx := findColumnIndex(cols, "BOARDID")

	prefRank := make(map[string]int, len(preferredBoards))
	for i, b := range preferredBoards {
		prefRank[strings.ToUpper(strings.TrimSpace(b))] = i
	}
	bestLot := 0
	bestRank := len(preferredBoards) + 1000
	fallbackLot := 0

	for _, rowAny := range data {
		row, _ := rowAny.([]interface{})
		if lotIdx >= len(row) {
			continue
		}
		lot := int(moexToFloat(row[lotIdx]))
		if lot <= 0 {
			continue
		}
		if fallbackLot == 0 {
			fallbackLot = lot
		}

		if boardIdx >= 0 && boardIdx < len(row) && len(prefRank) > 0 {
			board := strings.ToUpper(strings.TrimSpace(moexToString(row[boardIdx])))
			if rank, ok := prefRank[board]; ok {
				if rank < bestRank {
					bestRank = rank
					bestLot = lot
				}
			}
		}
	}

	if bestLot > 0 {
		return bestLot, nil
	}
	if fallbackLot > 0 {
		return fallbackLot, nil
	}
	return 0, fmt.Errorf("lotsize value not found")
}

func (c *MOEXClient) getClosePricesFromPath(path string, count int, interval int) ([]float64, error) {
	// Берем диапазон с запасом (выходные/праздники), чтобы собрать нужное число свечей.
	till := time.Now().UTC()
	spanDays := count * 3
	switch interval {
	case 1:
		spanDays = 3
	case 10:
		spanDays = 14
	case 60:
		spanDays = 45
	case 24:
		if spanDays < 14 {
			spanDays = 14
		}
	case 7:
		spanDays = 120
		if spanDays < count*21 {
			spanDays = count * 21
		}
	case 31:
		spanDays = 365
		if spanDays < count*93 {
			spanDays = count * 93
		}
	case 4:
		spanDays = 365 * 2
		if spanDays < count*365 {
			spanDays = count * 365
		}
	}
	from := till.AddDate(0, 0, -spanDays)

	u, _ := url.Parse(path)
	q := u.Query()
	q.Set("from", from.Format("2006-01-02"))
	q.Set("till", till.Format("2006-01-02"))
	q.Set("interval", strconv.Itoa(interval))
	u.RawQuery = q.Encode()

	resp, err := c.httpClient.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	candles, _ := result["candles"].(map[string]interface{})
	if candles == nil {
		return nil, fmt.Errorf("candles not found")
	}

	cols, _ := candles["columns"].([]interface{})
	data, _ := candles["data"].([]interface{})
	if len(cols) == 0 || len(data) == 0 {
		return nil, fmt.Errorf("no candles")
	}

	closeIdx := -1
	for i, col := range cols {
		if moexToString(col) == "close" || moexToString(col) == "CLOSE" {
			closeIdx = i
			break
		}
	}
	if closeIdx < 0 {
		return nil, fmt.Errorf("close column not found")
	}

	closes := make([]float64, 0, len(data))
	for _, rowAny := range data {
		row, _ := rowAny.([]interface{})
		if closeIdx >= len(row) {
			continue
		}
		switch v := row[closeIdx].(type) {
		case float64:
			closes = append(closes, v)
		case string:
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				closes = append(closes, f)
			}
		default:
			_ = v
		}
	}

	if len(closes) > count {
		closes = closes[len(closes)-count:]
	}
	return closes, nil
}

func extractLastPrice(result map[string]interface{}) (float64, error) {
	if marketdata, ok := result["marketdata"].(map[string]interface{}); ok {
		if data, ok := marketdata["data"].([]interface{}); ok && len(data) > 0 {
			if row, ok := data[0].([]interface{}); ok {
				if columns, ok := marketdata["columns"].([]interface{}); ok {
					for i, col := range columns {
						if moexToString(col) == "LAST" && i < len(row) {
							price := moexToFloat(row[i])
							if price > 0 {
								return price, nil
							}
						}
					}
				}
			}
		}
	}
	return 0, fmt.Errorf("last price not found")
}

func findColumnIndex(columns []interface{}, name string) int {
	for i, col := range columns {
		if moexToString(col) == name {
			return i
		}
	}
	return -1
}

func moexToString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprintf("%v", t)
	}
}

func moexToFloat(v interface{}) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case string:
		f, err := strconv.ParseFloat(t, 64)
		if err == nil {
			return f
		}
	}
	return 0
}
