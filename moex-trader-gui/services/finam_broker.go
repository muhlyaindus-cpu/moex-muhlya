package services

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	accountspb "github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/accounts"
	orderspb "github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/orders"
	tradev1 "github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1"
	"github.com/gorilla/websocket"
	dec "google.golang.org/genproto/googleapis/type/decimal"
	money "google.golang.org/genproto/googleapis/type/money"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
)

const finamGRPCEndpoint = "api.finam.ru:443"

// FinamBroker — клиент для работы с API Финам
type FinamBroker struct {
	secretReal         string
	currentSecret      string
	tokenJWT           string
	jwtExpiresAt       time.Time
	accounts           map[string]string
	currentAccount     string
	connected          bool
	baseURL            string
	httpClient         *http.Client
	wsConn             *websocket.Conn
	wsMutex            sync.Mutex
	quoteCallbacks     map[string]func(ticker string, price float64)
	subscribedSymbols  map[string]struct{}
	reconnectInProcess bool
	reconnectAttempt   int
	tradingCode        string
	balanceMu          sync.Mutex
	balanceCached      float64
	balanceCachedAt    time.Time
	balanceCacheTTL    time.Duration

	grpcMu   sync.Mutex
	grpcConn *grpc.ClientConn

	// MOEX/ISS base ticker -> полный символ Finam (например "AFU6": "AFU6@FUT"), см. LoadFinamSymbolOverridesFromEnv.
	symbolOverrides map[string]string
	// Кэш успешного/неуспешного GetAsset для символа после OrderSymbolForAPI.
	finamAssetMu      sync.Mutex
	assetValidUntil   map[string]time.Time
	assetInvalidUntil map[string]time.Time

	// Индекс фьючерсов Finam (AllAssets, MIC=FUT): нормализованный ключ -> полный Symbol (ticker@FUT).
	futuresIndexMu     sync.RWMutex
	futuresSymbolByKey map[string]string
	futuresIndexAt     time.Time
	// Успешно подтверждённый GetAsset альтернативный символ (MOEX SECID ≠ тикер Finam в индексе).
	futuresLearnedMu     sync.RWMutex
	futuresSymbolLearned map[string]string
}

type Account struct {
	ID          string
	Name        string
	Type        string
	TradingCode string
}

type OrderInfo struct {
	ID        string
	Ticker    string
	Side      string
	Quantity  int
	Price     float64
	Status    string
	CreatedAt time.Time
}

// AccountTradeRow — сделка из GET /v1/accounts/{accountId}/trades (см. Finam Trade-API AccountsService.Trades).
type AccountTradeRow struct {
	TradeID    string
	Symbol     string
	Side       string // buy | sell
	Quantity   int
	Price      float64
	ExecutedAt time.Time
	OrderID    string
}

// WebSocket сообщения
type WsRequest struct {
	Action string      `json:"action"`
	Type   string      `json:"type,omitempty"`
	Data   interface{} `json:"data,omitempty"`
	Token  string      `json:"token"`
}

type WsEnvelope struct {
	Type             string          `json:"type"`
	SubscriptionKey  string          `json:"subscription_key"`
	SubscriptionType string          `json:"subscription_type"`
	Timestamp        float64         `json:"timestamp"`
	Payload          json.RawMessage `json:"payload"`
	ErrorInfo        *struct {
		Code    int    `json:"code"`
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error_info"`
	EventInfo *struct {
		Event  string `json:"event"`
		Code   int    `json:"code"`
		Reason string `json:"reason"`
	} `json:"event_info"`
}

type QuotePayload struct {
	Quote []struct {
		Symbol   string `json:"symbol"`
		Last     string `json:"last"`
		Ask      string `json:"ask"`
		Bid      string `json:"bid"`
		Volume   string `json:"volume"`
		Turnover string `json:"turnover"`
		Open     string `json:"open"`
		High     string `json:"high"`
		Low      string `json:"low"`
		Close    string `json:"close"`
		Change   string `json:"change"`
	} `json:"quote"`
}

// newFinamHTTPClient — Finam в getting-started просит HTTP/2; на HTTP/1 бывают сбои шлюза.
func newFinamHTTPClient() *http.Client {
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2: true,
		MaxIdleConns:      100,
		TLSClientConfig:   &tls.Config{MinVersion: tls.VersionTLS12},
	}
	return &http.Client{
		Timeout:   15 * time.Second,
		Transport: tr,
	}
}

// finamAuthorizationHeader — как в официальном Go-примере Finam (metadata "Authorization"): сырой JWT.
// Префикс "Bearer " при наличии снимаем, чтобы не дублировать и не ломать REST-шлюз (400 szUnion).
func finamAuthorizationHeader(jwt string) string {
	jwt = strings.TrimSpace(jwt)
	if strings.HasPrefix(strings.ToLower(jwt), "bearer ") {
		jwt = strings.TrimSpace(jwt[len("Bearer "):])
	}
	return jwt
}

// finamAccountIDIsAlphanumericUnion — в кабинете Finam API в «Доступ к счетам» часто показывают id вида 133359RLB2K (не только цифры КлФ).
func finamAccountIDIsAlphanumericUnion(s string) bool {
	if s == "" {
		return false
	}
	hasLetter := false
	for _, r := range s {
		switch {
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z':
			hasLetter = true
		case r >= '0' && r <= '9':
		default:
			return false
		}
	}
	return hasLetter
}

// normalizeFinamAccountIDForREST — в path GET /v1/accounts/{id}: либо только цифры (номер КлФ без префикса), либо буквенно-цифровой union-id с портала токенов.
func normalizeFinamAccountIDForREST(id string) string {
	id = strings.TrimSpace(id)
	for _, p := range []string{"КлФ-", "клф-", "КЛФ-", "KlF-", "klf-", "KLF-"} {
		if strings.HasPrefix(id, p) {
			id = strings.TrimPrefix(id, p)
			break
		}
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return id
	}
	if _, err := strconv.Atoi(id); err == nil {
		return id
	}
	if finamAccountIDIsAlphanumericUnion(id) {
		return id
	}
	var num strings.Builder
	for _, r := range id {
		if r >= '0' && r <= '9' {
			num.WriteRune(r)
		}
	}
	if s := num.String(); s != "" {
		return s
	}
	return id
}

// normalizeTradingCodeForAccountID — в ЛК Финам часто «133359RLB2K/133359RLB2K»; для GetAccount нужен один код.
func normalizeTradingCodeForAccountID(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if i := strings.Index(s, "/"); i >= 0 {
		left, right := strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+1:])
		if left != "" && right != "" && left == right {
			return left
		}
		if left != "" {
			return left
		}
		return right
	}
	return s
}

func finamAccountPath(baseURL, accountID, suffix string) string {
	id := normalizeFinamAccountIDForREST(accountID)
	if id == "" {
		id = strings.TrimSpace(accountID)
	}
	if suffix != "" {
		return fmt.Sprintf("%s/v1/accounts/%s%s", strings.TrimSuffix(baseURL, "/"), url.PathEscape(id), suffix)
	}
	return fmt.Sprintf("%s/v1/accounts/%s", strings.TrimSuffix(baseURL, "/"), url.PathEscape(id))
}

func protoDecimalToFloat(d *dec.Decimal) float64 {
	if d == nil {
		return 0
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(d.GetValue()), 64)
	if err != nil {
		return 0
	}
	return f
}

func floatToDecimalProto(f float64) *dec.Decimal {
	if f == 0 {
		return nil
	}
	s := strconv.FormatFloat(f, 'f', -1, 64)
	return &dec.Decimal{Value: s}
}

func intQtyToDecimalProto(q int) *dec.Decimal {
	if q <= 0 {
		return nil
	}
	return &dec.Decimal{Value: strconv.Itoa(q)}
}

func moneyToFloat64(m *money.Money) float64 {
	if m == nil {
		return 0
	}
	return float64(m.GetUnits()) + float64(m.GetNanos())/1e9
}

func positionsFromFinamGRPC(resp *accountspb.GetAccountResponse) []Position {
	if resp == nil {
		return nil
	}
	out := make([]Position, 0, len(resp.GetPositions()))
	for _, p := range resp.GetPositions() {
		qty := protoDecimalToFloat(p.GetQuantity())
		out = append(out, Position{
			Ticker:       p.GetSymbol(),
			Quantity:     int(qty),
			AvgPrice:     protoDecimalToFloat(p.GetAveragePrice()),
			CurrentPrice: protoDecimalToFloat(p.GetCurrentPrice()),
			PnL:          protoDecimalToFloat(p.GetUnrealizedPnl()),
		})
	}
	return out
}

func balanceFromFinamGRPC(resp *accountspb.GetAccountResponse) (float64, bool) {
	if resp == nil {
		return 0, false
	}
	var rub float64
	foundRub := false
	for _, c := range resp.GetCash() {
		code := strings.ToUpper(strings.TrimSpace(c.GetCurrencyCode()))
		if code == "RUB" || code == "RUR" {
			rub += moneyToFloat64(c)
			foundRub = true
		}
	}
	if foundRub {
		return rub, true
	}
	if mc := resp.GetPortfolioMc(); mc != nil && mc.GetAvailableCash() != nil {
		return protoDecimalToFloat(mc.GetAvailableCash()), true
	}
	if f := resp.GetPortfolioForts(); f != nil && f.GetAvailableCash() != nil {
		return protoDecimalToFloat(f.GetAvailableCash()), true
	}
	if resp.GetEquity() != nil {
		return protoDecimalToFloat(resp.GetEquity()), true
	}
	return 0, false
}

func (b *FinamBroker) ensureFinamGRPC(ctx context.Context) (*grpc.ClientConn, error) {
	b.grpcMu.Lock()
	defer b.grpcMu.Unlock()
	if b.grpcConn != nil {
		return b.grpcConn, nil
	}
	creds := credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12})
	conn, err := grpc.NewClient(finamGRPCEndpoint, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, err
	}
	b.grpcConn = conn
	return b.grpcConn, nil
}

func (b *FinamBroker) closeFinamGRPC() {
	b.grpcMu.Lock()
	defer b.grpcMu.Unlock()
	if b.grpcConn != nil {
		_ = b.grpcConn.Close()
		b.grpcConn = nil
	}
}

// getAccountGRPC — GetAccount по gRPC; обходит баг REST JSON (400 Invalid parameter 'szUnion' на oneof portfolio).
func (b *FinamBroker) getAccountGRPC(ctx context.Context) (*accountspb.GetAccountResponse, error) {
	conn, err := b.ensureFinamGRPC(ctx)
	if err != nil {
		return nil, err
	}
	jwt, err := b.getJWT()
	if err != nil {
		return nil, err
	}
	auth := finamAuthorizationHeader(jwt)
	if auth == "" {
		return nil, errors.New("empty jwt")
	}
	mdCtx := metadata.AppendToOutgoingContext(ctx, "Authorization", auth)
	cli := accountspb.NewAccountsServiceClient(conn)
	aid := normalizeFinamAccountIDForREST(b.GetCurrentAccount())
	if aid == "" {
		return nil, errors.New("no current account selected")
	}
	return cli.GetAccount(mdCtx, &accountspb.GetAccountRequest{AccountId: aid})
}

// NewFinamBroker создаёт новый клиент (только реальный счёт)
func NewFinamBroker(secretReal string) (*FinamBroker, error) {
	if secretReal == "" {
		return nil, fmt.Errorf("no secret provided")
	}

	broker := &FinamBroker{
		secretReal:          secretReal,
		currentSecret:       secretReal,
		baseURL:             "https://api.finam.ru",
		httpClient:          newFinamHTTPClient(),
		accounts:            make(map[string]string),
		quoteCallbacks:      make(map[string]func(ticker string, price float64)),
		subscribedSymbols:   make(map[string]struct{}),
		balanceCacheTTL:     20 * time.Second,
		symbolOverrides:     LoadFinamSymbolOverridesFromEnv(),
		assetValidUntil:     make(map[string]time.Time),
		assetInvalidUntil:   make(map[string]time.Time),
	}

	// Получаем JWT и список доступных счетов
	if err := broker.refreshJWT(); err != nil {
		return nil, fmt.Errorf("failed to get JWT: %w", err)
	}
	if err := broker.refreshAccounts(); err != nil {
		return nil, fmt.Errorf("failed to load accounts: %w", err)
	}

	broker.connected = true
	log.Printf("✅ Connected to Finam Trade API (REAL)")
	if broker.currentAccount != "" {
		log.Printf("📊 Account: %s (real)", broker.currentAccount)
	}

	return broker, nil
}

// refreshJWT получает новый JWT токен из API
func (b *FinamBroker) refreshJWT() error {
	if b.tokenJWT != "" && time.Now().Before(b.jwtExpiresAt) {
		return nil
	}

	url := b.baseURL + "/v1/sessions"
	payload := map[string]string{"secret": b.currentSecret}
	jsonData, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("jwt request failed: status=%d body=%s", resp.StatusCode, string(body))
	}

	var result struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}
	if strings.TrimSpace(result.Token) == "" {
		return errors.New("empty token in jwt response")
	}

	b.tokenJWT = result.Token
	b.jwtExpiresAt = time.Now().Add(14 * time.Minute)
	log.Println("✅ JWT token refreshed")
	return nil
}

func (b *FinamBroker) refreshAccounts() error {
	jwt, err := b.getJWT()
	if err != nil {
		return err
	}

	url := b.baseURL + "/v1/sessions/details"
	payload := map[string]string{"token": jwt}
	jsonData, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("token details failed: status=%d body=%s", resp.StatusCode, string(body))
	}

	var details struct {
		AccountIDs []string `json:"account_ids"`
		Readonly   bool     `json:"readonly"`
	}
	if err := json.Unmarshal(body, &details); err != nil {
		return err
	}
	var raw map[string]interface{}
	tradingFromDetails := ""
	if err := json.Unmarshal(body, &raw); err == nil {
		tradingFromDetails = extractTradingCode(raw)
	}
	envAcc := normalizeFinamAccountIDForREST(os.Getenv("FINAM_ACCOUNT_ID"))
	if len(details.AccountIDs) == 0 {
		if envAcc == "" {
			return errors.New("no accounts returned by token details")
		}
		b.wsMutex.Lock()
		defer b.wsMutex.Unlock()
		b.tradingCode = tradingFromDetails
		b.accounts = map[string]string{envAcc: envAcc}
		b.currentAccount = envAcc
		log.Printf("📊 Account from FINAM_ACCOUNT_ID (token details had no account_ids): %s", envAcc)
		return nil
	}

	b.wsMutex.Lock()
	defer b.wsMutex.Unlock()
	b.tradingCode = tradingFromDetails
	b.accounts = make(map[string]string, len(details.AccountIDs)+1)
	for _, id := range details.AccountIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		norm := normalizeFinamAccountIDForREST(id)
		if norm == "" {
			continue
		}
		b.accounts[norm] = norm
	}
	// По умолчанию — первый счёт из token details; иначе см. FINAM_ACCOUNT_ID.
	for _, id := range details.AccountIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if norm := normalizeFinamAccountIDForREST(id); norm != "" {
			b.currentAccount = norm
			break
		}
	}
	if envAcc != "" {
		b.accounts[envAcc] = envAcc
		b.currentAccount = envAcc
		log.Printf("📊 Account override FINAM_ACCOUNT_ID=%s (должен совпадать с «Доступ к счетам» в кабинете токена)", envAcc)
	} else if tc := normalizeTradingCodeForAccountID(tradingFromDetails); tc != "" && finamAccountIDIsAlphanumericUnion(tc) {
		if _, err := strconv.Atoi(b.currentAccount); err == nil && tc != b.currentAccount {
			numericWas := b.currentAccount
			b.accounts[tc] = tc
			b.currentAccount = tc
			log.Printf("📊 account_id для API: торговый код из token details %s (в account_ids был числовой %s)", tc, numericWas)
		}
	}
	return nil
}

// getJWT возвращает актуальный JWT
func (b *FinamBroker) getJWT() (string, error) {
	if err := b.refreshJWT(); err != nil {
		return "", err
	}
	return b.tokenJWT, nil
}

// connectWebSocket подключается к WebSocket Финам
func (b *FinamBroker) connectWebSocket() {
	url := "wss://api.finam.ru/ws"

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Printf("❌ WebSocket connection failed: %v", err)
		return
	}

	b.wsMutex.Lock()
	b.wsConn = conn
	b.wsMutex.Unlock()

	log.Println("✅ WebSocket connected")

	// keepalive пинги в отдельной горутине
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			b.wsMutex.Lock()
			if b.wsConn != nil {
				if err := b.wsConn.WriteMessage(websocket.PingMessage, nil); err != nil {
					log.Printf("❌ Ping failed: %v", err)
				}
			}
			b.wsMutex.Unlock()
		}
	}()

	jwt, err := b.getJWT()
	if err != nil {
		log.Printf("❌ Failed to get JWT: %v", err)
		return
	}

	handshake := WsRequest{
		Action: "HANDSHAKE",
		Token:  jwt,
	}

	b.wsMutex.Lock()
	err = b.wsConn.WriteJSON(handshake)
	b.wsMutex.Unlock()
	if err != nil {
		log.Printf("❌ Handshake failed: %v", err)
		return
	}

	go b.readWebSocketMessages()
}

// readWebSocketMessages читает сообщения из WebSocket
func (b *FinamBroker) readWebSocketMessages() {
	defer func() {
		b.wsMutex.Lock()
		if b.wsConn != nil {
			b.wsConn.Close()
			b.wsConn = nil
		}
		b.wsMutex.Unlock()
	}()

	for {
		b.wsMutex.Lock()
		conn := b.wsConn
		b.wsMutex.Unlock()

		if conn == nil {
			break
		}

		var envelope WsEnvelope
		err := conn.ReadJSON(&envelope)
		if err != nil {
			if closeErr, ok := err.(*websocket.CloseError); ok {
				log.Printf("❌ WS close: code=%d, text=%s", closeErr.Code, closeErr.Text)
			} else {
				log.Printf("❌ WS read error: %v", err)
			}
			go b.reconnectWebSocket()
			break
		}

		// Обрабатываем событие
		if envelope.Type == "EVENT" && envelope.EventInfo != nil {
			log.Printf("📡 Event: %s", envelope.EventInfo.Event)
			if envelope.EventInfo.Event == "HANDSHAKE_SUCCESS" {
				log.Println("✅ WebSocket handshake successful")
				// После реконнекта восстанавливаем активные подписки
				go b.resubscribeAll()
			}
			continue
		}

		// Ошибки WS не всегда требуют переподключения:
		// сервер может вернуть ERROR для конкретного действия, но держать канал живым.
		if envelope.Type == "ERROR" && envelope.ErrorInfo != nil {
			log.Printf("❌ WebSocket error: code=%d, msg=%s", envelope.ErrorInfo.Code, envelope.ErrorInfo.Message)
			continue
		}

		// Обрабатываем данные
		if envelope.Type == "DATA" {
			log.Printf("📡 Received DATA: type=%s, payload length=%d", envelope.SubscriptionType, len(envelope.Payload))

			if envelope.SubscriptionType == "QUOTES" {
				var payload QuotePayload
				if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
					log.Printf("❌ Failed to parse quote: %v", err)
					continue
				}

				for _, quote := range payload.Quote {
					price := 0.0
					fmt.Sscanf(quote.Last, "%f", &price)

					b.wsMutex.Lock()
					cb, ok := b.quoteCallbacks[quote.Symbol]
					b.wsMutex.Unlock()

					if ok {
						cb(quote.Symbol, price)
					}
				}
			}
		}
	}
}

// reconnectWebSocket переподключает WebSocket
func (b *FinamBroker) reconnectWebSocket() {
	b.wsMutex.Lock()
	if b.reconnectInProcess {
		b.wsMutex.Unlock()
		return
	}
	shouldReconnect := len(b.subscribedSymbols) > 0
	if !shouldReconnect {
		if b.wsConn != nil {
			b.wsConn.Close()
			b.wsConn = nil
		}
		b.wsMutex.Unlock()
		log.Printf("ℹ️ WS reconnect skipped: no active subscriptions")
		return
	}
	b.reconnectInProcess = true
	b.reconnectAttempt++
	attempt := b.reconnectAttempt

	if b.wsConn != nil {
		b.wsConn.Close()
		b.wsConn = nil
	}
	b.wsMutex.Unlock()

	delay := time.Duration(1<<minInt(attempt, 6)) * time.Second // 2s..64s
	log.Printf("🔄 Reconnecting WebSocket... (attempt=%d, delay=%s)", attempt, delay)
	time.Sleep(delay)

	b.connectWebSocket()

	b.wsMutex.Lock()
	b.reconnectInProcess = false
	b.wsMutex.Unlock()
}

// SubscribeToQuotes подписывается на котировки
func (b *FinamBroker) SubscribeToQuotes(ticker string, callback func(ticker string, price float64)) error {
	b.wsMutex.Lock()
	needsConnect := b.wsConn == nil
	b.wsMutex.Unlock()
	if needsConnect {
		b.connectWebSocket()
		b.wsMutex.Lock()
		stillDisconnected := b.wsConn == nil
		b.wsMutex.Unlock()
		if stillDisconnected {
			return fmt.Errorf("WebSocket not connected")
		}
	}

	jwt, err := b.getJWT()
	if err != nil {
		return err
	}

	b.wsMutex.Lock()
	defer b.wsMutex.Unlock()

	symbol := b.formatSymbolForWS(ticker)
	b.quoteCallbacks[symbol] = callback
	b.subscribedSymbols[symbol] = struct{}{}

	data := map[string][]string{
		"symbols": {symbol},
	}

	req := WsRequest{
		Action: "SUBSCRIBE",
		Type:   "QUOTES",
		Data:   data,
		Token:  jwt,
	}

	jsonBytes, _ := json.Marshal(req)
	log.Printf("📡 Sending subscribe: %s", jsonBytes)

	if err := b.wsConn.WriteJSON(req); err != nil {
		return fmt.Errorf("failed to subscribe: %v", err)
	}

	log.Printf("📡 Subscribed to quotes: %s -> %s", ticker, symbol)
	return nil
}

// UnsubscribeFromQuotes отписывается от котировок
func (b *FinamBroker) UnsubscribeFromQuotes(ticker string) error {
	b.wsMutex.Lock()
	defer b.wsMutex.Unlock()

	if b.wsConn == nil {
		return fmt.Errorf("WebSocket not connected")
	}

	jwt, err := b.getJWT()
	if err != nil {
		return err
	}

	symbol := b.formatSymbolForWS(ticker)
	delete(b.quoteCallbacks, symbol)
	delete(b.subscribedSymbols, symbol)

	data := map[string][]string{
		"symbols": {symbol},
	}

	req := WsRequest{
		Action: "UNSUBSCRIBE",
		Type:   "QUOTES",
		Data:   data,
		Token:  jwt,
	}

	jsonBytes, _ := json.Marshal(req)
	log.Printf("📡 Sending unsubscribe: %s", jsonBytes)

	if err := b.wsConn.WriteJSON(req); err != nil {
		return fmt.Errorf("failed to unsubscribe: %v", err)
	}

	log.Printf("📡 Unsubscribed from quotes: %s", ticker)
	return nil
}

func (b *FinamBroker) resubscribeAll() {
	b.wsMutex.Lock()
	defer b.wsMutex.Unlock()

	if b.wsConn == nil {
		return
	}
	if len(b.subscribedSymbols) == 0 {
		return
	}

	jwt, err := b.getJWT()
	if err != nil {
		log.Printf("❌ Failed to get JWT for resubscribe: %v", err)
		return
	}

	symbols := make([]string, 0, len(b.subscribedSymbols))
	for sym := range b.subscribedSymbols {
		symbols = append(symbols, sym)
	}

	req := WsRequest{
		Action: "SUBSCRIBE",
		Type:   "QUOTES",
		Data: map[string][]string{
			"symbols": symbols,
		},
		Token: jwt,
	}

	if err := b.wsConn.WriteJSON(req); err != nil {
		log.Printf("❌ Resubscribe failed: %v", err)
		return
	}
	log.Printf("📡 Resubscribed to %d symbols", len(symbols))
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// formatSymbolForWS форматирует тикер для WebSocket (учитывает переопределения Finam).
func (b *FinamBroker) formatSymbolForWS(ticker string) string {
	return b.OrderSymbolForAPI(ticker)
}

// formatSymbolForREST форматирует тикер для REST API
func (b *FinamBroker) formatSymbolForREST(ticker string) string {
	if strings.Contains(ticker, "@") {
		return ticker
	}
	if isLikelyFutureTicker(ticker) {
		return ticker + "@FUT"
	}
	return ticker + "@MISX"
}

func isLikelyFutureTicker(ticker string) bool {
	// Для FORTS тикеров часто характерно наличие цифр (месяц/год контракта).
	hasDigit := false
	for _, r := range ticker {
		if r >= '0' && r <= '9' {
			hasDigit = true
			break
		}
	}
	return hasDigit
}

// GetAccounts возвращает список доступных счетов
func (b *FinamBroker) GetAccounts() []Account {
	b.wsMutex.Lock()
	defer b.wsMutex.Unlock()

	accounts := make([]Account, 0)
	for name, id := range b.accounts {
		accounts = append(accounts, Account{
			ID:          id,
			Name:        formatAccountDisplay(name),
			Type:        "real",
			TradingCode: b.tradingCode,
		})
	}
	return accounts
}

// GetCurrentAccount возвращает текущий счёт
func (b *FinamBroker) GetCurrentAccount() string {
	b.wsMutex.Lock()
	defer b.wsMutex.Unlock()
	return b.currentAccount
}

func (b *FinamBroker) GetTradingCode() string {
	b.wsMutex.Lock()
	defer b.wsMutex.Unlock()
	return b.tradingCode
}

// SwitchAccount переключает счёт (у нас только один)
func (b *FinamBroker) SwitchAccount(accountName string) error {
	b.wsMutex.Lock()
	defer b.wsMutex.Unlock()
	key := normalizeFinamAccountIDForREST(accountName)
	if key == "" {
		return fmt.Errorf("account not found: %s", accountName)
	}
	if _, ok := b.accounts[key]; !ok {
		return fmt.Errorf("account not found: %s", accountName)
	}
	b.currentAccount = key
	return nil
}

// GetPortfolio возвращает портфель
func (b *FinamBroker) GetPortfolio(ctx context.Context) ([]Position, error) {
	accountID := b.GetCurrentAccount()
	if accountID == "" {
		return nil, errors.New("no current account selected")
	}

	if pb, err := b.getAccountGRPC(ctx); err == nil {
		pos := positionsFromFinamGRPC(pb)
		return pos, nil
	} else {
		log.Printf("⚠️ Finam GetAccount gRPC: %v (fallback REST)", err)
	}

	jwt, err := b.getJWT()
	if err != nil {
		return nil, err
	}

	u := finamAccountPath(b.baseURL, accountID, "")
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", finamAuthorizationHeader(jwt))
	req.Header.Set("Accept", "application/json")

	resp, err := b.doWithRetry(req, 4)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("get portfolio failed: status=%d body=%s", resp.StatusCode, string(body))
	}

	// Структура портфеля может отличаться, поэтому разбираем максимально терпимо.
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	positions := make([]Position, 0)
	rawPositions, _ := raw["positions"].([]interface{})
	for _, item := range rawPositions {
		posMap, _ := item.(map[string]interface{})
		if posMap == nil {
			continue
		}

		symbol, _ := posMap["symbol"].(string)
		qty := extractDecimalValue(posMap["quantity"])
		avg := extractDecimalValue(posMap["avg_price"])
		if avg == 0 {
			avg = extractDecimalValue(posMap["average_price"])
		}
		if avg == 0 {
			avg = extractMoneyValue(posMap["average_price"])
		}

		positions = append(positions, Position{
			Ticker:   symbol,
			Quantity: int(qty),
			AvgPrice: avg,
		})
	}

	return positions, nil
}

// GetBalance возвращает баланс
func (b *FinamBroker) GetBalance(ctx context.Context) (float64, error) {
	b.balanceMu.Lock()
	if !b.balanceCachedAt.IsZero() && time.Since(b.balanceCachedAt) < b.balanceCacheTTL {
		cached := b.balanceCached
		b.balanceMu.Unlock()
		return cached, nil
	}
	b.balanceMu.Unlock()

	accountID := b.GetCurrentAccount()
	if accountID == "" {
		return 0, errors.New("no current account selected")
	}

	if pb, err := b.getAccountGRPC(ctx); err == nil {
		if bal, ok := balanceFromFinamGRPC(pb); ok {
			b.balanceMu.Lock()
			b.balanceCached = bal
			b.balanceCachedAt = time.Now()
			b.balanceMu.Unlock()
			log.Printf("💰 Balance (Finam gRPC) account %s: %.2f", normalizeFinamAccountIDForREST(accountID), bal)
			return bal, nil
		}
	} else {
		log.Printf("⚠️ Finam GetAccount gRPC for balance: %v (fallback REST)", err)
	}

	jwt, err := b.getJWT()
	if err != nil {
		return 0, err
	}

	u := finamAccountPath(b.baseURL, accountID, "")
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", finamAuthorizationHeader(jwt))
	req.Header.Set("Accept", "application/json")

	resp, err := b.doWithRetry(req, 4)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		if resp.StatusCode == http.StatusTooManyRequests {
			b.balanceMu.Lock()
			hasCache := !b.balanceCachedAt.IsZero()
			cached := b.balanceCached
			b.balanceMu.Unlock()
			if hasCache {
				log.Printf("⚠️ Finam rate limit on balance; using cached value %.2f", cached)
				return cached, nil
			}
		}
		return 0, fmt.Errorf("get balance failed: status=%d body=%s", resp.StatusCode, string(body))
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return 0, err
	}

	if bal, ok := extractBalanceFromAccountResponse(raw); ok {
		b.balanceMu.Lock()
		b.balanceCached = bal
		b.balanceCachedAt = time.Now()
		b.balanceMu.Unlock()
		log.Printf("💰 Balance parsed for account %s: %.2f", accountID, bal)
		return bal, nil
	}
	log.Printf("⚠️ Unable to parse balance for account %s. Top-level fields: %v", accountID, mapKeys(raw))
	return 0, errors.New("unable to parse balance from account response")
}

// BuyMarket покупает по рыночной цене
func (b *FinamBroker) BuyMarket(ctx context.Context, ticker string, quantity int) (string, error) {
	return b.placeOrder(ctx, ticker, "SIDE_BUY", "ORDER_TYPE_MARKET", quantity, 0)
}

// SellMarket продаёт по рыночной цене
func (b *FinamBroker) SellMarket(ctx context.Context, ticker string, quantity int) (string, error) {
	return b.placeOrder(ctx, ticker, "SIDE_SELL", "ORDER_TYPE_MARKET", quantity, 0)
}

// BuyLimit покупает по лимитной цене
func (b *FinamBroker) BuyLimit(ctx context.Context, ticker string, quantity int, price float64) (string, error) {
	return b.placeOrder(ctx, ticker, "SIDE_BUY", "ORDER_TYPE_LIMIT", quantity, price)
}

// SellLimit продаёт по лимитной цене
func (b *FinamBroker) SellLimit(ctx context.Context, ticker string, quantity int, price float64) (string, error) {
	return b.placeOrder(ctx, ticker, "SIDE_SELL", "ORDER_TYPE_LIMIT", quantity, price)
}

// GetOrders получает активные заявки
func (b *FinamBroker) GetOrders(ctx context.Context) ([]OrderInfo, error) {
	accountID := b.GetCurrentAccount()
	if accountID == "" {
		return nil, errors.New("no current account selected")
	}

	jwt, err := b.getJWT()
	if err != nil {
		return nil, err
	}

	u := finamAccountPath(b.baseURL, accountID, "/orders")
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", finamAuthorizationHeader(jwt))
	req.Header.Set("Accept", "application/json")

	resp, err := b.doWithRetry(req, 3)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("get orders failed: status=%d body=%s", resp.StatusCode, string(body))
	}

	var raw struct {
		Orders []struct {
			OrderID string `json:"order_id"`
			Order   struct {
				Symbol string `json:"symbol"`
				Side   string `json:"side"`
				Type   string `json:"type"`
			} `json:"order"`
			Status string `json:"status"`
		} `json:"orders"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		// иногда API возвращает массив напрямую
		var rawArray []map[string]interface{}
		if err2 := json.Unmarshal(body, &rawArray); err2 != nil {
			return nil, err
		}
		return []OrderInfo{}, nil
	}

	out := make([]OrderInfo, 0, len(raw.Orders))
	for _, o := range raw.Orders {
		out = append(out, OrderInfo{
			ID:     o.OrderID,
			Ticker: o.Order.Symbol,
			Side:   o.Order.Side,
			Status: o.Status,
		})
	}
	return out, nil
}

func accountTradeRowFromProto(t *tradev1.AccountTrade) AccountTradeRow {
	if t == nil {
		return AccountTradeRow{}
	}
	side := "buy"
	if t.GetSide() == tradev1.Side_SIDE_SELL {
		side = "sell"
	}
	qty := int(protoDecimalToFloat(t.GetSize()) + 0.5)
	if qty < 0 {
		qty = 0
	}
	ts := time.Time{}
	if t.GetTimestamp() != nil {
		ts = t.GetTimestamp().AsTime()
	}
	return AccountTradeRow{
		TradeID:    strings.TrimSpace(t.GetTradeId()),
		Symbol:     strings.TrimSpace(t.GetSymbol()),
		Side:       side,
		Quantity:   qty,
		Price:      protoDecimalToFloat(t.GetPrice()),
		ExecutedAt: ts,
		OrderID:    strings.TrimSpace(t.GetOrderId()),
	}
}

// GetAccountTrades загружает историю сделок по текущему счёту (REST GET .../trades).
// limit — макс. число записей; start/end — опциональный интервал (оба nil = без фильтра по времени).
func (b *FinamBroker) GetAccountTrades(ctx context.Context, limit int, start, end *time.Time) ([]AccountTradeRow, error) {
	accountID := b.GetCurrentAccount()
	if accountID == "" {
		return nil, errors.New("no current account selected")
	}
	jwt, err := b.getJWT()
	if err != nil {
		return nil, err
	}

	u := finamAccountPath(b.baseURL, accountID, "/trades")
	q := url.Values{}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	// Finam REST ожидает endTime в interval. Если интервал не передан, задаём безопасный дефолт.
	var from, to time.Time
	if start != nil {
		from = start.UTC()
	}
	if end != nil {
		to = end.UTC()
	}
	if to.IsZero() {
		to = time.Now().UTC()
	}
	if from.IsZero() {
		from = to.AddDate(0, 0, -14)
	}
	if from.After(to) {
		from = to.AddDate(0, 0, -1)
	}
	startVal := from.Format(time.RFC3339Nano)
	endVal := to.Format(time.RFC3339Nano)
	// Поддерживаем оба варианта имён полей: protobuf-json (camelCase) и legacy snake_case.
	q.Set("interval.startTime", startVal)
	q.Set("interval.endTime", endVal)
	q.Set("interval.start_time", startVal)
	q.Set("interval.end_time", endVal)
	if enc := q.Encode(); enc != "" {
		u += "?" + enc
	}

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", finamAuthorizationHeader(jwt))
	req.Header.Set("Accept", "application/json")

	resp, err := b.doWithRetry(req, 3)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("get account trades failed: status=%d body=%s", resp.StatusCode, string(body))
	}

	var tr accountspb.TradesResponse
	opts := protojson.UnmarshalOptions{DiscardUnknown: true}
	if err := opts.Unmarshal(body, &tr); err != nil {
		var wrap struct {
			Trades []json.RawMessage `json:"trades"`
		}
		if err2 := json.Unmarshal(body, &wrap); err2 != nil || len(wrap.Trades) == 0 {
			return nil, fmt.Errorf("decode trades response: %w", err)
		}
		out := make([]AccountTradeRow, 0, len(wrap.Trades))
		for _, raw := range wrap.Trades {
			var t tradev1.AccountTrade
			if err3 := opts.Unmarshal(raw, &t); err3 != nil {
				continue
			}
			out = append(out, accountTradeRowFromProto(&t))
		}
		return out, nil
	}

	raw := tr.GetTrades()
	out := make([]AccountTradeRow, 0, len(raw))
	for _, t := range raw {
		out = append(out, accountTradeRowFromProto(t))
	}
	return out, nil
}

// PlaceSLTPOrder выставляет связанную стоп-заявку SL/TP (gRPC PlaceSLTPOrder, Finam Trade-API v2.13+).
// closeSide — сторона закрытия позиции: SIDE_BUY или SIDE_SELL (для лонга обычно продажа).
// Можно указать только SL, только TP или обе ноги (см. quantity_* и цены > 0).
func (b *FinamBroker) PlaceSLTPOrder(ctx context.Context, ticker string, closeSide tradev1.Side, slQty int, slPrice float64, tpQty int, tpPrice float64) (string, error) {
	hasSL := slQty > 0 && slPrice > 0
	hasTP := tpQty > 0 && tpPrice > 0
	if !hasSL && !hasTP {
		return "", errors.New("нужна хотя бы одна нога: SL или TP (количество и цена > 0)")
	}
	if closeSide != tradev1.Side_SIDE_BUY && closeSide != tradev1.Side_SIDE_SELL {
		return "", errors.New("сторона закрытия: только buy или sell (в терминах Finam SIDE_BUY / SIDE_SELL)")
	}

	conn, err := b.ensureFinamGRPC(ctx)
	if err != nil {
		return "", err
	}
	jwt, err := b.getJWT()
	if err != nil {
		return "", err
	}
	auth := finamAuthorizationHeader(jwt)
	if auth == "" {
		return "", errors.New("empty jwt")
	}
	mdCtx := metadata.AppendToOutgoingContext(ctx, "Authorization", auth)

	aid := normalizeFinamAccountIDForREST(b.GetCurrentAccount())
	if aid == "" {
		return "", errors.New("no current account selected")
	}
	symbol := b.OrderSymbolForAPI(ticker)

	req := &orderspb.SLTPOrder{
		AccountId:   aid,
		Symbol:      symbol,
		Side:        closeSide,
		ValidBefore: orderspb.ValidBefore_VALID_BEFORE_GOOD_TILL_CANCEL,
		Comment:     fmt.Sprintf("moex-sltp|%d", time.Now().Unix()),
	}
	if hasSL {
		req.QuantitySl = intQtyToDecimalProto(slQty)
		req.SlPrice = floatToDecimalProto(slPrice)
	}
	if hasTP {
		req.QuantityTp = intQtyToDecimalProto(tpQty)
		req.TpPrice = floatToDecimalProto(tpPrice)
	}

	cli := orderspb.NewOrdersServiceClient(conn)
	st, err := cli.PlaceSLTPOrder(mdCtx, req)
	if err != nil {
		return "", err
	}
	if st == nil {
		return "", errors.New("пустой ответ PlaceSLTPOrder")
	}
	oid := strings.TrimSpace(st.GetOrderId())
	if oid == "" {
		return "", errors.New("в ответе нет order_id")
	}
	return oid, nil
}

// CancelOrder отменяет заявку
func (b *FinamBroker) CancelOrder(ctx context.Context, orderId string) error {
	accountID := b.GetCurrentAccount()
	if accountID == "" {
		return errors.New("no current account selected")
	}

	jwt, err := b.getJWT()
	if err != nil {
		return err
	}

	u := finamAccountPath(b.baseURL, accountID, "/orders/"+url.PathEscape(strings.TrimSpace(orderId)))
	req, err := http.NewRequestWithContext(ctx, "DELETE", u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", finamAuthorizationHeader(jwt))
	req.Header.Set("Accept", "application/json")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("cancel order failed: status=%d body=%s", resp.StatusCode, string(body))
	}
	return nil
}

// Close закрывает соединение
func (b *FinamBroker) Close() {
	b.wsMutex.Lock()
	if b.wsConn != nil {
		b.wsConn.Close()
		b.wsConn = nil
	}
	b.wsMutex.Unlock()
	b.closeFinamGRPC()
	b.connected = false
	log.Println("✅ Disconnected from Finam Trade API")
}

func (b *FinamBroker) placeOrder(ctx context.Context, ticker, side, orderType string, quantity int, limitPrice float64) (string, error) {
	accountID := b.GetCurrentAccount()
	if accountID == "" {
		return "", errors.New("no current account selected")
	}

	jwt, err := b.getJWT()
	if err != nil {
		return "", err
	}

	symbol := b.OrderSymbolForAPI(ticker)

	u := finamAccountPath(b.baseURL, accountID, "/orders")
	bodyMap := map[string]interface{}{
		"symbol": symbol,
		"quantity": map[string]string{
			"value": strconv.Itoa(quantity),
		},
		"side":          side,
		"type":          orderType,
		"time_in_force": "TIME_IN_FORCE_DAY",
		"comment":       fmt.Sprintf("moex-bot|%s|%d", strings.ToLower(side), time.Now().Unix()),
	}
	if orderType == "ORDER_TYPE_LIMIT" {
		bodyMap["limit_price"] = map[string]string{
			"value": strconv.FormatFloat(limitPrice, 'f', -1, 64),
		}
	}

	jsonData, _ := json.Marshal(bodyMap)
	req, err := http.NewRequestWithContext(ctx, "POST", u, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", finamAuthorizationHeader(jwt))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("place order failed: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var result struct {
		OrderID string `json:"order_id"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", err
	}
	if strings.TrimSpace(result.OrderID) == "" {
		return "", fmt.Errorf("empty order_id in response: %s", string(respBody))
	}
	return result.OrderID, nil
}

func extractDecimalValue(v interface{}) float64 {
	m, _ := v.(map[string]interface{})
	if m == nil {
		return 0
	}
	s, _ := m["value"].(string)
	if s == "" {
		return 0
	}
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func extractMoneyValue(v interface{}) float64 {
	m, _ := v.(map[string]interface{})
	if m == nil {
		return 0
	}
	return extractDecimalValue(m["value"])
}

func extractBalanceFromAccountResponse(raw map[string]interface{}) (float64, bool) {
	priorityFields := []string{"cash", "available", "balance", "equity", "money"}
	for _, key := range priorityFields {
		if v, ok := raw[key]; ok {
			if amount, ok := extractAmountFromUnknown(v); ok {
				return amount, true
			}
		}
	}

	for _, key := range []string{"currencies", "money", "balances", "assets"} {
		if v, ok := raw[key]; ok {
			if amount, ok := extractRUBFromCurrencies(v); ok {
				return amount, true
			}
		}
	}

	if v, ok := raw["currencies"]; ok {
		if amount, ok := extractRUBFromCurrencies(v); ok {
			return amount, true
		}
	}

	if amount, ok := recursiveBalanceSearch(raw); ok {
		return amount, true
	}
	return 0, false
}

func extractRUBFromCurrencies(v interface{}) (float64, bool) {
	items, ok := v.([]interface{})
	if !ok {
		return 0, false
	}
	sum := 0.0
	found := false
	for _, item := range items {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		ccy := strings.ToUpper(strings.TrimSpace(toString(m["currency"])))
		if ccy == "" {
			ccy = strings.ToUpper(strings.TrimSpace(toString(m["code"])))
		}
		if ccy != "RUB" && ccy != "RUR" {
			continue
		}
		for _, k := range []string{"available", "balance", "equity", "amount", "value"} {
			if val, exists := m[k]; exists {
				if n, ok := extractAmountFromUnknown(val); ok {
					sum += n
					found = true
					break
				}
			}
		}
	}
	return sum, found
}

func mapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func recursiveBalanceSearch(v interface{}) (float64, bool) {
	switch t := v.(type) {
	case map[string]interface{}:
		for k, child := range t {
			lk := strings.ToLower(k)
			if strings.Contains(lk, "cash") || strings.Contains(lk, "balance") || strings.Contains(lk, "available") || strings.Contains(lk, "equity") {
				if n, ok := extractAmountFromUnknown(child); ok {
					return n, true
				}
			}
			if n, ok := recursiveBalanceSearch(child); ok {
				return n, true
			}
		}
	case []interface{}:
		for _, child := range t {
			if n, ok := recursiveBalanceSearch(child); ok {
				return n, true
			}
		}
	}
	return 0, false
}

func extractAmountFromUnknown(v interface{}) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case string:
		if t == "" {
			return 0, false
		}
		n, err := strconv.ParseFloat(t, 64)
		if err != nil {
			return 0, false
		}
		return n, true
	case map[string]interface{}:
		for _, key := range []string{"value", "amount", "available", "balance", "equity"} {
			if nested, ok := t[key]; ok {
				if n, ok := extractAmountFromUnknown(nested); ok {
					return n, true
				}
			}
		}
	case []interface{}:
		total := 0.0
		found := false
		for _, item := range t {
			if n, ok := extractAmountFromUnknown(item); ok {
				total += n
				found = true
			}
		}
		return total, found
	}
	return 0, false
}

func toString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprintf("%v", t)
	}
}

func formatAccountDisplay(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return id
	}
	if strings.HasPrefix(strings.ToUpper(id), "КЛФ-") {
		return id
	}
	if _, err := strconv.Atoi(id); err == nil {
		return "КлФ-" + id
	}
	return id
}

func extractTradingCode(raw map[string]interface{}) string {
	for _, key := range []string{"trading_code", "trade_code", "client_code", "code"} {
		if v, ok := raw[key]; ok {
			s := strings.TrimSpace(toString(v))
			if s != "" && s != "<nil>" {
				return s
			}
		}
	}
	return ""
}

func (b *FinamBroker) doWithRetry(req *http.Request, maxAttempts int) (*http.Response, error) {
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		clone := req.Clone(req.Context())
		resp, err := b.httpClient.Do(clone)
		if err != nil {
			lastErr = err
		} else {
			// retriable 5xx (включая 504) и 429
			if (resp.StatusCode >= 500 && resp.StatusCode <= 599) || resp.StatusCode == http.StatusTooManyRequests {
				body, _ := io.ReadAll(resp.Body)
				retryAfter := strings.TrimSpace(resp.Header.Get("Retry-After"))
				resp.Body.Close()
				lastErr = fmt.Errorf("upstream %d: %s", resp.StatusCode, string(body))
				if retryAfter != "" {
					log.Printf("⚠️ Finam transient error attempt=%d/%d (retry-after=%s): %v", attempt, maxAttempts, retryAfter, lastErr)
				} else {
					log.Printf("⚠️ Finam transient error attempt=%d/%d: %v", attempt, maxAttempts, lastErr)
				}
			} else {
				return resp, nil
			}
		}

		if attempt < maxAttempts {
			delay := time.Duration(250*(1<<(attempt-1))) * time.Millisecond
			time.Sleep(delay)
		}
	}
	return nil, lastErr
}
