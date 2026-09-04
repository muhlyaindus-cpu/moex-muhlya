package services

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

type PostgresDB struct {
	db *sql.DB
}

type Position struct {
	Ticker       string  `json:"ticker"`
	Quantity     int     `json:"quantity"`
	AvgPrice     float64 `json:"avg_price"`
	CurrentPrice float64 `json:"current_price"`
	PnL          float64 `json:"pnl"`
}

type TradeRecord struct {
	ID         int       `json:"id"`
	Ticker     string    `json:"ticker"`
	Side       string    `json:"side"`
	Quantity   int       `json:"quantity"`
	Price      float64   `json:"price"`
	Status     string    `json:"status"`
	ExecutedAt time.Time `json:"executed_at"`
}

type RuntimeEventRecord struct {
	ID        int       `json:"id"`
	At        time.Time `json:"at"`
	Level     string    `json:"level"`
	Step      string    `json:"step"`
	Details   string    `json:"details"`
	CreatedAt time.Time `json:"created_at"`
}

func NewPostgresDB(host, port, user, password, dbname string) (*PostgresDB, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	p := &PostgresDB{db: db}
	if err := p.ensureSchema(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *PostgresDB) ensureSchema() error {
	_, err := p.db.Exec(`
ALTER TABLE IF EXISTS trades
  ALTER COLUMN status TYPE TEXT
`)
	if err != nil {
		return err
	}
	_, err = p.db.Exec(`
UPDATE trades
SET executed_at = NULL
WHERE status = 'rejected' OR status LIKE 'rejected:%'
`)
	if err != nil {
		return err
	}
	_, err = p.db.Exec(`
CREATE TABLE IF NOT EXISTS runtime_events (
  id BIGSERIAL PRIMARY KEY,
  at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  level TEXT NOT NULL,
  step TEXT NOT NULL,
  details TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)
`)
	if err != nil {
		return err
	}
	_, err = p.db.Exec(`CREATE INDEX IF NOT EXISTS idx_runtime_events_created_at ON runtime_events (created_at DESC)`)
	return err
}

func (p *PostgresDB) GetPositions() ([]Position, error) {
	rows, err := p.db.Query(`
SELECT ticker, quantity, avg_price 
FROM positions 
WHERE quantity != 0
ORDER BY ticker
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	positions := make([]Position, 0)
	for rows.Next() {
		var pos Position
		if err := rows.Scan(&pos.Ticker, &pos.Quantity, &pos.AvgPrice); err != nil {
			continue
		}
		positions = append(positions, pos)
	}

	return positions, nil
}

func (p *PostgresDB) BuyPosition(ticker string, quantity int, price float64) bool {
	tx, err := p.db.Begin()
	if err != nil {
		return false
	}
	defer tx.Rollback()

	query := `
INSERT INTO positions (ticker, quantity, avg_price) 
VALUES ($1, $2, $3)
ON CONFLICT (ticker) 
DO UPDATE SET 
quantity = positions.quantity + EXCLUDED.quantity,
avg_price = (positions.avg_price * positions.quantity + EXCLUDED.avg_price * EXCLUDED.quantity) 
   / (positions.quantity + EXCLUDED.quantity),
updated_at = CURRENT_TIMESTAMP
`

	_, err = tx.Exec(query, ticker, quantity, price)
	if err != nil {
		return false
	}

	tradeQuery := `
INSERT INTO trades (ticker, side, quantity, price, status, executed_at) 
VALUES ($1, 'buy', $2, $3, 'executed', $4)
`
	_, err = tx.Exec(tradeQuery, ticker, quantity, price, time.Now())
	if err != nil {
		return false
	}

	return tx.Commit() == nil
}

func (p *PostgresDB) SellPosition(ticker string, quantity int, price float64) bool {
	var currentQuantity int
	err := p.db.QueryRow("SELECT quantity FROM positions WHERE ticker = $1", ticker).Scan(&currentQuantity)
	if err != nil || currentQuantity < quantity {
		return false
	}

	tx, err := p.db.Begin()
	if err != nil {
		return false
	}
	defer tx.Rollback()

	newQuantity := currentQuantity - quantity
	if newQuantity == 0 {
		_, err = tx.Exec("UPDATE positions SET quantity = 0, updated_at = CURRENT_TIMESTAMP WHERE ticker = $1", ticker)
	} else {
		_, err = tx.Exec("UPDATE positions SET quantity = $1, updated_at = CURRENT_TIMESTAMP WHERE ticker = $2", newQuantity, ticker)
	}
	if err != nil {
		return false
	}

	tradeQuery := `
INSERT INTO trades (ticker, side, quantity, price, status, executed_at) 
VALUES ($1, 'sell', $2, $3, 'executed', $4)
`
	_, err = tx.Exec(tradeQuery, ticker, quantity, price, time.Now())
	if err != nil {
		return false
	}

	return tx.Commit() == nil
}

func (p *PostgresDB) Close() error {
	return p.db.Close()
}

func (p *PostgresDB) LogTrade(ticker, side string, quantity int, price float64, status string) bool {
	var executedAt interface{}
	s := strings.ToLower(strings.TrimSpace(status))
	// executed_at должно быть заполнено только для реально дошедших до брокера записей.
	if s == "executed" || strings.HasPrefix(s, "executed:") || s == "placed" || strings.HasPrefix(s, "placed:") {
		executedAt = time.Now()
	}
	_, err := p.db.Exec(`
INSERT INTO trades (ticker, side, quantity, price, status, executed_at)
VALUES ($1, $2, $3, $4, $5, $6)
`, ticker, side, quantity, price, status, executedAt)
	return err == nil
}

func (p *PostgresDB) GetRecentTrades(limit int) ([]TradeRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := p.db.Query(`
SELECT id, ticker, side, quantity, price, status, executed_at
FROM trades
ORDER BY COALESCE(executed_at, created_at) DESC
LIMIT $1
`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]TradeRecord, 0, limit)
	for rows.Next() {
		var t TradeRecord
		if err := rows.Scan(&t.ID, &t.Ticker, &t.Side, &t.Quantity, &t.Price, &t.Status, &t.ExecutedAt); err != nil {
			continue
		}
		out = append(out, t)
	}
	return out, nil
}

func (p *PostgresDB) LogRuntimeEvent(at time.Time, level, step, details string, maxRows int) bool {
	if maxRows <= 0 {
		maxRows = 500
	}
	if maxRows < 50 {
		maxRows = 50
	}
	// Изначально стоял жесткий лимит 5000.
	// Подняли, чтобы не упираться в него при длинных трейдинговых сессиях/сборе логов.
	if maxRows > 50_000 {
		maxRows = 50_000
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	_, err := p.db.Exec(`
INSERT INTO runtime_events (at, level, step, details)
VALUES ($1, $2, $3, $4)
`, at, level, step, details)
	if err != nil {
		return false
	}
	_, _ = p.db.Exec(`
DELETE FROM runtime_events
WHERE id IN (
  SELECT id
  FROM runtime_events
  ORDER BY created_at DESC
  OFFSET $1
)
`, maxRows)
	return true
}

func (p *PostgresDB) GetRecentRuntimeEvents(limit int) ([]RuntimeEventRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 2000 {
		limit = 2000
	}
	rows, err := p.db.Query(`
SELECT id, at, level, step, details, created_at
FROM runtime_events
ORDER BY created_at DESC
LIMIT $1
`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]RuntimeEventRecord, 0, limit)
	for rows.Next() {
		var e RuntimeEventRecord
		if err := rows.Scan(&e.ID, &e.At, &e.Level, &e.Step, &e.Details, &e.CreatedAt); err != nil {
			continue
		}
		out = append(out, e)
	}
	return out, nil
}
