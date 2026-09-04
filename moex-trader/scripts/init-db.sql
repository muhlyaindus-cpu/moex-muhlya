-- одключаемся к базе moex_trading
\c moex_trading;

-- Таблица для отслеживания инструментов
CREATE TABLE IF NOT EXISTS watchlist (
    id SERIAL PRIMARY KEY,
    ticker VARCHAR(20) UNIQUE NOT NULL,
    name VARCHAR(100),
    added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Таблица для сделок
CREATE TABLE IF NOT EXISTS trades (
    id SERIAL PRIMARY KEY,
    ticker VARCHAR(20) NOT NULL,
    side VARCHAR(10) NOT NULL,
    quantity INTEGER NOT NULL,
    price DECIMAL(20,2) NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    executed_at TIMESTAMP
);

-- Таблица для позиций
CREATE TABLE IF NOT EXISTS positions (
    id SERIAL PRIMARY KEY,
    ticker VARCHAR(20) UNIQUE NOT NULL,
    quantity INTEGER NOT NULL DEFAULT 0,
    avg_price DECIMAL(20,2) DEFAULT 0,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Таблица для рыночных данных
CREATE TABLE IF NOT EXISTS market_data (
    id SERIAL PRIMARY KEY,
    time TIMESTAMP NOT NULL,
    ticker VARCHAR(20) NOT NULL,
    open DECIMAL(20,2),
    high DECIMAL(20,2),
    low DECIMAL(20,2),
    close DECIMAL(20,2),
    volume BIGINT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Создаем индекс для быстрого поиска
CREATE INDEX IF NOT EXISTS idx_market_data_ticker_time ON market_data(ticker, time);

-- обавляем популярные тикеры в watchlist
INSERT INTO watchlist (ticker, name) VALUES
    ('SBER', 'Сбербанк'),
    ('GAZP', 'азпром'),
    ('LKOH', 'укойл'),
    ('YNDX', 'Яндекс'),
    ('TATN', 'Татнефть'),
    ('ROSN', 'оснефть'),
    ('VTBR', 'Т'),
    ('NVTK', 'Т'),
    ('PLZL', 'олюс'),
    ('MGNT', 'агнит')
ON CONFLICT (ticker) DO NOTHING;

-- ыводим сообщение
DO \$\$
BEGIN
    RAISE NOTICE ' аза данных moex_trading успешно инициализирована';
    RAISE NOTICE ' Таблицы созданы: watchlist, trades, positions, market_data';
END \$\$;
