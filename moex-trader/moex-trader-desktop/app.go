package main

import (
	"context"
	"fmt"
	"os"

	"moex-trader-desktop/backend/services"
)

type App struct {
	ctx   context.Context
	moex  *services.MOEXClient
	db    *services.PostgresDB
	agent *services.MOEXTradingAssistant
}

func NewApp() *App {
	return &App{
		moex: services.NewMOEXClient(),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	fmt.Println("🚀 MOEX Trading Terminal starting...")

	// Подключаемся к PostgreSQL
	pgHost := envOrDefault("PGHOST", "localhost")
	pgPort := envOrDefault("PGPORT", "5433")
	pgUser := envOrDefault("PGUSER", "trader")
	pgPassword := os.Getenv("PGPASSWORD")
	pgDatabase := envOrDefault("PGDATABASE", "moex_trading")

	db, err := services.NewPostgresDB(pgHost, pgPort, pgUser, pgPassword, pgDatabase)
	if err != nil {
		fmt.Printf("⚠️  DB error: %v\n", err)
	} else {
		a.db = db
		fmt.Println("✅ Connected to PostgreSQL")
	}

	// Инициализируем агента
	a.agent = services.NewMOEXTradingAssistant()
	fmt.Println("✅ Ready")
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func (a *App) GetCurrentPrice(ticker string) float64 {
	price, _ := a.moex.GetCurrentPrice(ticker)
	return price
}

func (a *App) GetWatchlist() []services.Security {
	return a.moex.GetWatchlist()
}

func (a *App) GetPortfolio() []services.Position {
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

func (a *App) BuyPosition(ticker string, quantity int, price float64) bool {
	if a.db == nil {
		return false
	}
	return a.db.BuyPosition(ticker, quantity, price)
}

func (a *App) SellPosition(ticker string, quantity int, price float64) bool {
	if a.db == nil {
		return false
	}
	return a.db.SellPosition(ticker, quantity, price)
}

func (a *App) GetAgentStatus() map[string]interface{} {
	if a.agent == nil {
		return map[string]interface{}{"active": false}
	}
	return map[string]interface{}{
		"active":      a.agent.IsActive(),
		"last_signal": a.agent.GetLastSignal(),
	}
}

func (a *App) SetAgentActive(active bool) bool {
	if a.agent == nil {
		return false
	}
	a.agent.SetActive(active)
	return true
}

func (a *App) AnalyzeWithAgent(ticker string) map[string]interface{} {
	if a.agent == nil || !a.agent.IsActive() {
		return map[string]interface{}{"error": "Agent not active"}
	}

	price, _ := a.moex.GetCurrentPrice(ticker)

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

	ctx := &services.MarketContext{
		Ticker:       ticker,
		CurrentPrice: price,
		Position:     position,
		Balance:      100000,
		Indicators: map[string]float64{
			"rsi":   65.5,
			"sma20": price * 0.98,
			"sma50": price * 0.95,
		},
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

func (a *App) GetStatistics() map[string]interface{} {
	stats := map[string]interface{}{
		"total_value":     0.0,
		"total_pnl":       0.0,
		"positions_count": 0,
		"return_pct":      0.0,
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

	return stats
}
