package services

import (
	"context"
	"fmt"
	"time"
)

// MOEXTradingAssistant — торговый ассистент MOEX (упрощённая логика RSI для desktop).
type MOEXTradingAssistant struct {
	isActive   bool
	lastSignal *TradeSignal
}

type TradeSignal struct {
	Action     string    `json:"action"`
	Quantity   int       `json:"quantity"`
	Price      float64   `json:"price"`
	Confidence float64   `json:"confidence"`
	Reasoning  string    `json:"reasoning"`
	Timestamp  time.Time `json:"timestamp"`
}

type MarketContext struct {
	Ticker       string
	CurrentPrice float64
	Position     *Position
	Balance      float64
	Indicators   map[string]float64
}

func NewMOEXTradingAssistant() *MOEXTradingAssistant {
	return &MOEXTradingAssistant{isActive: false}
}

func (a *MOEXTradingAssistant) SetActive(active bool) {
	a.isActive = active
}

func (a *MOEXTradingAssistant) IsActive() bool {
	return a.isActive
}

func (a *MOEXTradingAssistant) Analyze(_ context.Context, market *MarketContext) (*TradeSignal, error) {
	if !a.isActive {
		return nil, fmt.Errorf("agent not active")
	}

	signal := &TradeSignal{
		Action:     "hold",
		Quantity:   0,
		Price:      market.CurrentPrice,
		Confidence: 0.5,
		Reasoning:  "Analysis in progress",
		Timestamp:  time.Now(),
	}

	if rsi, ok := market.Indicators["rsi"]; ok {
		if rsi < 30 {
			signal.Action = "buy"
			signal.Quantity = 10
			signal.Confidence = 0.7
			signal.Reasoning = fmt.Sprintf("RSI oversold (%.1f), buy signal", rsi)
		} else if rsi > 70 {
			signal.Action = "sell"
			signal.Quantity = 10
			signal.Confidence = 0.6
			signal.Reasoning = fmt.Sprintf("RSI overbought (%.1f), sell signal", rsi)
		}
	}

	a.lastSignal = signal
	return signal, nil
}

func (a *MOEXTradingAssistant) GetLastSignal() *TradeSignal {
	return a.lastSignal
}
