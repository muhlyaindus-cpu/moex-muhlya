package engine

import (
	"context"
	"time"
)

type Mode string

const (
	ModeManual Mode = "manual"
	ModeAuto   Mode = "auto"
	ModePaper  Mode = "paper"
)

type Side string

const (
	SideBuy  Side = "buy"
	SideSell Side = "sell"
)

type OrderType string

const (
	OrderTypeMarket OrderType = "market"
	OrderTypeLimit  OrderType = "limit"
)

type TimeInForce string

const (
	TimeInForceDay TimeInForce = "day"
)

type Decimal struct {
	Value string `json:"value"`
}

type Money struct {
	Currency string  `json:"currency"`
	Value    Decimal `json:"value"`
}

type Position struct {
	Symbol       string  `json:"symbol"`
	Ticker       string  `json:"ticker"`
	Quantity     float64 `json:"quantity"`
	AvgPrice     float64 `json:"avg_price"`
	CurrentPrice float64 `json:"current_price"`
	PnL          float64 `json:"pnl"`
}

type Account struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type OrderRequest struct {
	Symbol      string
	Side        Side
	Type        OrderType
	Quantity    Decimal
	LimitPrice  *Decimal
	TimeInForce TimeInForce
	Comment     string
}

type OrderInfo struct {
	ID        string    `json:"id"`
	Symbol    string    `json:"symbol"`
	Side      Side      `json:"side"`
	Quantity  float64   `json:"quantity"`
	Price     float64   `json:"price"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type Quote struct {
	Symbol string
	Last   float64
	Bid    float64
	Ask    float64
	Time   time.Time
}

type Broker interface {
	Connect(ctx context.Context) error
	Close() error

	GetAccounts(ctx context.Context) ([]Account, error)
	SetCurrentAccount(accountID string) error
	GetCurrentAccount() (string, error)

	GetPortfolio(ctx context.Context) ([]Position, error)
	GetOrders(ctx context.Context) ([]OrderInfo, error)
	PlaceOrder(ctx context.Context, req OrderRequest) (string, error)
	CancelOrder(ctx context.Context, orderID string) error

	SubscribeQuotes(ctx context.Context, symbols []string, onQuote func(q Quote)) (unsubscribe func() error, err error)
}
