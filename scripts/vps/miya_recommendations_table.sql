CREATE TABLE IF NOT EXISTS miya_recommendations (
    id SERIAL PRIMARY KEY,
    recommended_at TIMESTAMPTZ NOT NULL,
    ticker VARCHAR(20) NOT NULL,
    score FLOAT NOT NULL,
    trend VARCHAR(10),
    rsi FLOAT,
    price_at_rec FLOAT,
    price_at_check FLOAT,
    checked_at TIMESTAMPTZ,
    result VARCHAR(10) DEFAULT 'pending'
);
CREATE INDEX IF NOT EXISTS idx_miya_rec_ticker ON miya_recommendations(ticker);
CREATE INDEX IF NOT EXISTS idx_miya_rec_result ON miya_recommendations(result);
