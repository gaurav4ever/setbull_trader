-- Create stock_universe table
-- This table stores information about all stocks available for trading
-- It serves as the main reference for stock data in the application

CREATE TABLE IF NOT EXISTS stock_universe (
    id BIGSERIAL PRIMARY KEY,
    symbol VARCHAR(30) NOT NULL,
    name VARCHAR(100) NOT NULL,
    exchange VARCHAR(10) NOT NULL,
    instrument_type VARCHAR(20) NOT NULL,
    isin VARCHAR(20),
    instrument_key VARCHAR(50) NOT NULL,
    trading_symbol VARCHAR(50) NOT NULL,
    exchange_token VARCHAR(20),
    last_price DECIMAL(18, 2),
    tick_size DECIMAL(18, 2),
    lot_size INTEGER,
    is_selected BOOLEAN DEFAULT FALSE,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for faster lookups
CREATE UNIQUE INDEX IF NOT EXISTS idx_stock_universe_symbol ON stock_universe(symbol);
CREATE INDEX IF NOT EXISTS idx_stock_universe_instrument_key ON stock_universe(instrument_key);
CREATE INDEX IF NOT EXISTS idx_stock_universe_is_selected ON stock_universe(is_selected);

-- Add comment to the table for documentation
COMMENT ON TABLE stock_universe IS 'Stores all available stocks from NSE via Upstox with their details';
