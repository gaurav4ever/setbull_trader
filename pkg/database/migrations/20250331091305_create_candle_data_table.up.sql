CREATE TABLE IF NOT EXISTS stock_candle_data (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    instrument_key VARCHAR(50) NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    open DECIMAL(18,2) NOT NULL,
    high DECIMAL(18,2) NOT NULL,
    low DECIMAL(18,2) NOT NULL,
    close DECIMAL(18,2) NOT NULL,
    volume BIGINT NOT NULL,
    open_interest BIGINT NOT NULL,
    time_interval VARCHAR(20) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_stock_candle_instrument_key ON stock_candle_data(instrument_key);
CREATE INDEX idx_stock_candle_timestamp ON stock_candle_data(timestamp);
CREATE INDEX idx_stock_candle_instrument_timestamp ON stock_candle_data(instrument_key, timestamp);
CREATE INDEX idx_stock_candle_interval ON stock_candle_data(time_interval);

ALTER TABLE stock_candle_data 
ADD CONSTRAINT idx_stock_candle_unique 
UNIQUE (instrument_key, timestamp, time_interval);