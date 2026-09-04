package services

import (
	"database/sql"
	"fmt"
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

func NewPostgresDB(host, port, user, password, dbname string) (*PostgresDB, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &PostgresDB{db: db}, nil
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
	if _, err := tx.Exec(query, ticker, quantity, price); err != nil {
		return false
	}

	tradeQuery := `
INSERT INTO trades (ticker, side, quantity, price, status, executed_at)
VALUES ($1, 'buy', $2, $3, 'executed', $4)
`
	if _, err := tx.Exec(tradeQuery, ticker, quantity, price, time.Now()); err != nil {
		return false
	}

	return tx.Commit() == nil
}

func (p *PostgresDB) SellPosition(ticker string, quantity int, price float64) bool {
	var currentQuantity int
	if err := p.db.QueryRow("SELECT quantity FROM positions WHERE ticker = $1", ticker).Scan(&currentQuantity); err != nil {
		return false
	}
	if currentQuantity < quantity {
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
	if _, err := tx.Exec(tradeQuery, ticker, quantity, price, time.Now()); err != nil {
		return false
	}

	return tx.Commit() == nil
}

func (p *PostgresDB) Close() error {
	return p.db.Close()
}
