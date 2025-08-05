CREATE TABLE IF NOT EXISTS aggregated_prices (
    id SERIAL PRIMARY KEY,
    symbol TEXT NOT NULL,
    exchange TEXT NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    average_price DOUBLE PRECISION,
    min_price DOUBLE PRECISION,
    max_price DOUBLE PRECISION
);

CREATE TABLE IF NOT EXISTS price_raw (
    id SERIAL PRIMARY KEY,
    symbol TEXT NOT NULL,
    exchange TEXT NOT NULL,
    price DOUBLE PRECISION NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_aggregated_prices_symbol_exchange ON aggregated_prices(symbol, exchange);
CREATE INDEX IF NOT EXISTS idx_aggregated_prices_timestamp ON aggregated_prices(timestamp);
CREATE INDEX IF NOT EXISTS idx_price_raw_symbol_exchange ON price_raw(symbol, exchange);
CREATE INDEX IF NOT EXISTS idx_price_raw_timestamp ON price_raw(timestamp);
CREATE INDEX IF NOT EXISTS idx_price_raw_symbol_timestamp ON price_raw(symbol, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_aggregated_prices_symbol_timestamp ON aggregated_prices(symbol, timestamp DESC);