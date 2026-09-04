package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"moex-trader/pkg/moex"

	_ "github.com/lib/pq"
)

type App struct {
	db     *sql.DB
	client *moex.Client
}

func main() {
	fmt.Println("╔════════════════════════════════════════════════════════╗")
	fmt.Println("║     MOEX Trading Terminal v1.0 (PostgreSQL)            ║")
	fmt.Println("║     Московская биржа - Торговый терминал               ║")
	fmt.Println("╚════════════════════════════════════════════════════════╝")
	fmt.Println()

	app := &App{
		client: moex.NewClient(),
	}

	// Подключение к PostgreSQL
	fmt.Print("🔍 Подключение к PostgreSQL... ")
	if err := app.connectPostgres(); err != nil {
		fmt.Printf("❌ Ошибка: %v\n", err)
		fmt.Println("⚠️  Работа без сохранения данных")
		fmt.Println("💡 Проверьте: docker-compose up -d")
	} else {
		fmt.Println("✅ Подключено")
		defer app.db.Close()
	}
	fmt.Println()

	// Проверка MOEX
	fmt.Print("🔍 Проверка подключения к MOEX... ")
	price, err := app.client.GetCurrentPrice("SBER")
	if err != nil {
		fmt.Printf("❌ Ошибка: %v\n", err)
		fmt.Println("⚠️  Работа в офлайн-режиме")
	} else {
		fmt.Printf("✅ Готово! Текущая цена SBER: %.2f RUB\n", price)
	}
	fmt.Println()

	printHelp()

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("\n📊 MOEX> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		parts := strings.Fields(input)
		cmd := strings.ToLower(parts[0])

		switch cmd {
		case "help", "h", "?":
			printHelp()

		case "price", "p":
			if len(parts) < 2 {
				fmt.Println("❌ Укажите тикер: price SBER")
				continue
			}
			ticker := strings.ToUpper(parts[1])

			price, err := app.client.GetCurrentPrice(ticker)
			if err != nil {
				fmt.Printf("❌ Ошибка: %v\n", err)
			} else {
				fmt.Printf("💰 %s: %.2f RUB\n", ticker, price)
				if app.db != nil {
					app.savePrice(ticker, price)
				}
			}

		case "watchlist", "w":
			app.showWatchlist()

		case "positions", "pos":
			app.showPositions()

		case "buy":
			if len(parts) < 4 {
				fmt.Println("❌ Использование: buy <TICKER> <QUANTITY> <PRICE>")
				fmt.Println("   Пример: buy SBER 10 250.5")
				continue
			}
			ticker := strings.ToUpper(parts[1])
			quantity, _ := strconv.Atoi(parts[2])
			price, _ := strconv.ParseFloat(parts[3], 64)
			app.buyPosition(ticker, quantity, price)

		case "sell":
			if len(parts) < 4 {
				fmt.Println("❌ Использование: sell <TICKER> <QUANTITY> <PRICE>")
				fmt.Println("   Пример: sell SBER 5 260.0")
				continue
			}
			ticker := strings.ToUpper(parts[1])
			quantity, _ := strconv.Atoi(parts[2])
			price, _ := strconv.ParseFloat(parts[3], 64)
			app.sellPosition(ticker, quantity, price)

		case "status", "s":
			app.showStatus()

		case "exit", "quit", "q":
			fmt.Println("\n👋 До свидания! Удачной торговли!")
			return

		default:
			fmt.Printf("❌ Неизвестная команда: %s\n", cmd)
			fmt.Println("💡 Введите 'help' для списка команд")
		}
	}
}

func (app *App) connectPostgres() error {
	host := envOrDefault("PGHOST", "localhost")
	port := envOrDefault("PGPORT", "5433")
	user := envOrDefault("PGUSER", "trader")
	password := os.Getenv("PGPASSWORD")
	dbname := envOrDefault("PGDATABASE", "moex_trading")

	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("не удалось открыть соединение: %v", err)
	}

	// Настройка пула соединений
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Проверяем подключение
	if err := db.Ping(); err != nil {
		return fmt.Errorf("не удалось подключиться: %v", err)
	}

	app.db = db
	return nil
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func (app *App) savePrice(ticker string, price float64) {
	query := `INSERT INTO market_data (time, ticker, close, volume) 
              VALUES ($1, $2, $3, $4)`

	_, err := app.db.Exec(query, time.Now(), ticker, price, 0)
	if err != nil {
		fmt.Printf("⚠️  Не удалось сохранить в БД: %v\n", err)
	}
}

func (app *App) buyPosition(ticker string, quantity int, price float64) {
	if app.db == nil {
		fmt.Println("❌ Нет подключения к БД")
		return
	}

	// Начинаем транзакцию
	tx, err := app.db.Begin()
	if err != nil {
		fmt.Printf("❌ Ошибка начала транзакции: %v\n", err)
		return
	}
	defer tx.Rollback()

	// Обновляем или создаем позицию
	query := `
        INSERT INTO positions (ticker, quantity, avg_price) 
        VALUES ($1, $2, $3)
        ON CONFLICT (ticker) 
        DO UPDATE SET 
            quantity = positions.quantity + EXCLUDED.quantity,
            avg_price = CASE 
                WHEN positions.quantity = 0 THEN EXCLUDED.avg_price
                ELSE (positions.avg_price * positions.quantity + EXCLUDED.avg_price * EXCLUDED.quantity) 
                     / (positions.quantity + EXCLUDED.quantity)
            END,
            updated_at = CURRENT_TIMESTAMP
    `

	_, err = tx.Exec(query, ticker, quantity, price)
	if err != nil {
		fmt.Printf("❌ Ошибка обновления позиции: %v\n", err)
		return
	}

	// Сохраняем сделку
	tradeQuery := `
        INSERT INTO trades (ticker, side, quantity, price, status, executed_at) 
        VALUES ($1, 'buy', $2, $3, 'executed', $4)
    `
	_, err = tx.Exec(tradeQuery, ticker, quantity, price, time.Now())
	if err != nil {
		fmt.Printf("❌ Ошибка сохранения сделки: %v\n", err)
		return
	}

	// Фиксируем транзакцию
	if err := tx.Commit(); err != nil {
		fmt.Printf("❌ Ошибка фиксации транзакции: %v\n", err)
		return
	}

	fmt.Printf("✅ Куплено %d %s по цене %.2f RUB\n", quantity, ticker, price)
}

func (app *App) sellPosition(ticker string, quantity int, price float64) {
	if app.db == nil {
		fmt.Println("❌ Нет подключения к БД")
		return
	}

	// Проверяем наличие позиции
	var currentQuantity int
	err := app.db.QueryRow("SELECT quantity FROM positions WHERE ticker = $1", ticker).Scan(&currentQuantity)
	if err != nil {
		fmt.Printf("❌ Ошибка проверки позиции: %v\n", err)
		return
	}

	if currentQuantity < quantity {
		fmt.Printf("❌ Недостаточно акций для продажи. Есть: %d, хотите продать: %d\n", currentQuantity, quantity)
		return
	}

	// Начинаем транзакцию
	tx, err := app.db.Begin()
	if err != nil {
		fmt.Printf("❌ Ошибка начала транзакции: %v\n", err)
		return
	}
	defer tx.Rollback()

	// Обновляем позицию
	newQuantity := currentQuantity - quantity
	var updateQuery string
	if newQuantity == 0 {
		updateQuery = `UPDATE positions SET quantity = 0, updated_at = CURRENT_TIMESTAMP WHERE ticker = $1`
		_, err = tx.Exec(updateQuery, ticker)
	} else {
		updateQuery = `UPDATE positions SET quantity = $1, updated_at = CURRENT_TIMESTAMP WHERE ticker = $2`
		_, err = tx.Exec(updateQuery, newQuantity, ticker)
	}

	if err != nil {
		fmt.Printf("❌ Ошибка обновления позиции: %v\n", err)
		return
	}

	// Сохраняем сделку
	tradeQuery := `
        INSERT INTO trades (ticker, side, quantity, price, status, executed_at) 
        VALUES ($1, 'sell', $2, $3, 'executed', $4)
    `
	_, err = tx.Exec(tradeQuery, ticker, quantity, price, time.Now())
	if err != nil {
		fmt.Printf("❌ Ошибка сохранения сделки: %v\n", err)
		return
	}

	// Фиксируем транзакцию
	if err := tx.Commit(); err != nil {
		fmt.Printf("❌ Ошибка фиксации транзакции: %v\n", err)
		return
	}

	fmt.Printf("✅ Продано %d %s по цене %.2f RUB\n", quantity, ticker, price)
}

func (app *App) showWatchlist() {
	if app.db == nil {
		fmt.Println("❌ Нет подключения к БД")
		return
	}

	rows, err := app.db.Query(`
        SELECT w.ticker, w.name, 
               COALESCE(
                   (SELECT close FROM market_data 
                    WHERE ticker = w.ticker 
                    ORDER BY time DESC LIMIT 1),
                   0
               ) as last_price,
               COALESCE(p.quantity, 0) as quantity
        FROM watchlist w
        LEFT JOIN positions p ON w.ticker = p.ticker
        ORDER BY w.ticker
    `)

	if err != nil {
		fmt.Printf("❌ Ошибка получения данных: %v\n", err)
		return
	}
	defer rows.Close()

	fmt.Println("\n📋 Список отслеживания:")
	fmt.Println("╔════════╦══════════════════════╦════════════╦════════════╦════════════╗")
	fmt.Println("║ Тикер  ║ Название            ║ Цена (RUB) ║ Позиция    ║ Стоимость  ║")
	fmt.Println("╠════════╬══════════════════════╬════════════╬════════════╬════════════╣")

	for rows.Next() {
		var ticker, name string
		var price float64
		var quantity int

		if err := rows.Scan(&ticker, &name, &price, &quantity); err != nil {
			continue
		}

		if price == 0 {
			// Если нет сохраненной цены, пробуем получить текущую
			currentPrice, err := app.client.GetCurrentPrice(ticker)
			if err == nil {
				price = currentPrice
				app.savePrice(ticker, currentPrice)
			} else {
				fmt.Printf("║ %-6s ║ %-20s ║ %-10s ║ %-10d ║ %-10s ║\n",
					ticker, name, "N/A", quantity, "N/A")
				continue
			}
		}

		totalValue := float64(quantity) * price
		fmt.Printf("║ %-6s ║ %-20s ║ %-10.2f ║ %-10d ║ %-10.2f ║\n",
			ticker, name, price, quantity, totalValue)
	}
	fmt.Println("╚════════╩══════════════════════╩════════════╩════════════╩════════════╝")
}

func (app *App) showPositions() {
	if app.db == nil {
		fmt.Println("❌ Нет подключения к БД")
		return
	}

	rows, err := app.db.Query(`
        SELECT ticker, quantity, avg_price, updated_at
        FROM positions 
        WHERE quantity != 0
        ORDER BY ticker
    `)

	if err != nil {
		fmt.Printf("❌ Ошибка получения позиций: %v\n", err)
		return
	}
	defer rows.Close()

	hasPositions := false
	totalValue := 0.0
	totalPnL := 0.0

	fmt.Println("\n💼 Текущие позиции:")
	fmt.Println("╔════════╦══════════════╦══════════════╦══════════════╦══════════════╦══════════════╗")
	fmt.Println("║ Тикер  ║ Количество   ║ Средняя цена║ Текущая цена║ Стоимость    ║ P&L          ║")
	fmt.Println("╠════════╬══════════════╬══════════════╬══════════════╬══════════════╬══════════════╣")

	for rows.Next() {
		var ticker string
		var quantity int
		var avgPrice float64
		var updatedAt time.Time

		if err := rows.Scan(&ticker, &quantity, &avgPrice, &updatedAt); err != nil {
			continue
		}

		currentPrice, err := app.client.GetCurrentPrice(ticker)
		if err != nil {
			fmt.Printf("║ %-6s ║ %-12d ║ %-12.2f ║ %-12s ║ %-12s ║ %-12s ║\n",
				ticker, quantity, avgPrice, "N/A", "N/A", "N/A")
			continue
		}

		positionValue := float64(quantity) * currentPrice
		pnl := float64(quantity) * (currentPrice - avgPrice)
		pnlPercent := (pnl / (float64(quantity) * avgPrice)) * 100

		pnlSymbol := "📈"
		if pnl < 0 {
			pnlSymbol = "📉"
		}

		fmt.Printf("║ %-6s ║ %-12d ║ %-12.2f ║ %-12.2f ║ %-12.2f ║ %-10.2f (%.1f%%) %s ║\n",
			ticker, quantity, avgPrice, currentPrice, positionValue, pnl, pnlPercent, pnlSymbol)

		totalValue += positionValue
		totalPnL += pnl
		hasPositions = true
	}
	fmt.Println("╚════════╩══════════════╩══════════════╩══════════════╩══════════════╩══════════════╝")

	if !hasPositions {
		fmt.Println("  📭 Нет открытых позиций")
		fmt.Println("  💡 Для покупки используйте: buy SBER 10 250.5")
	} else {
		fmt.Printf("\n📊 Итого:\n")
		fmt.Printf("   Общая стоимость портфеля: %.2f RUB\n", totalValue)
		fmt.Printf("   Общий P&L: %.2f RUB\n", totalPnL)
		if totalValue > 0 {
			fmt.Printf("   Доходность: %.2f%%\n", (totalPnL/totalValue)*100)
		}
	}
}

func (app *App) showStatus() {
	fmt.Println("\n📊 Статус системы:")
	fmt.Printf("  🕐 Время: %s\n", time.Now().Format("2006-01-02 15:04:05"))

	// Проверка PostgreSQL
	fmt.Printf("  🗄️  PostgreSQL: ")
	if app.db != nil {
		if err := app.db.Ping(); err == nil {
			fmt.Println("✅ Подключено")

			var version string
			app.db.QueryRow("SELECT version()").Scan(&version)
			fmt.Printf("     Версия: %s\n", strings.Split(version, ",")[0])

			// Статистика
			var dbSize string
			app.db.QueryRow("SELECT pg_database_size(current_database())::int / 1024 / 1024").Scan(&dbSize)
			fmt.Printf("     Размер БД: %s MB\n", dbSize)
		} else {
			fmt.Printf("❌ Ошибка: %v\n", err)
		}
	} else {
		fmt.Println("❌ Не подключено")
	}

	// Проверка MOEX
	fmt.Printf("  🌐 MOEX API: ")
	start := time.Now()
	price, err := app.client.GetCurrentPrice("SBER")
	latency := time.Since(start)

	if err != nil {
		fmt.Printf("❌ Недоступен (%v)\n", err)
	} else {
		fmt.Printf("✅ Доступен (латентность: %dms)\n", latency.Milliseconds())
		fmt.Printf("     Текущая цена SBER: %.2f RUB\n", price)
	}

	// Статистика по БД
	if app.db != nil {
		var marketCount int
		app.db.QueryRow("SELECT COUNT(*) FROM market_data").Scan(&marketCount)
		fmt.Printf("  📈 Сохранено цен: %d\n", marketCount)

		var tradeCount int
		app.db.QueryRow("SELECT COUNT(*) FROM trades").Scan(&tradeCount)
		fmt.Printf("  💰 Сделок: %d\n", tradeCount)

		var watchCount int
		app.db.QueryRow("SELECT COUNT(*) FROM watchlist").Scan(&watchCount)
		fmt.Printf("  📋 В watchlist: %d инструментов\n", watchCount)

		var posCount int
		var totalQuantity int
		app.db.QueryRow("SELECT COUNT(*), COALESCE(SUM(quantity), 0) FROM positions WHERE quantity != 0").Scan(&posCount, &totalQuantity)
		fmt.Printf("  💼 Открытых позиций: %d (всего акций: %d)\n", posCount, totalQuantity)
	}
}

func printHelp() {
	fmt.Println(`
╔══════════════════════════════════════════════════════════════════╗
║                        Доступные команды                          ║
╠══════════════════════════════════════════════════════════════════╣
║  help, h, ?           - показать эту справку                     ║
║  price <TICKER>, p    - получить текущую цену                    ║
║  buy <TICKER> <QTY> <PRICE> - купить акции                       ║
║  sell <TICKER> <QTY> <PRICE> - продать акции                     ║
║  watchlist, w         - показать список отслеживания             ║
║  positions, pos       - показать текущие позиции                 ║
║  status, s            - статус подключений                       ║
║  exit, quit, q        - выход из программы                       ║
╠══════════════════════════════════════════════════════════════════╣
║ Примеры:                                                         ║
║  price SBER           - цена Сбербанка                           ║
║  buy SBER 10 250.5    - купить 10 акций Сбера по 250.5          ║
║  sell SBER 5 260.0    - продать 5 акций Сбера по 260.0          ║
║  watchlist            - все инструменты                          ║
║  positions            - текущие позиции и P&L                    ║
║  status               - информация о системе                     ║
╚══════════════════════════════════════════════════════════════════╝
    `)
}
