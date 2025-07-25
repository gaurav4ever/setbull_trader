CREATE TABLE IF NOT EXISTS stock_candle_data_5min (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    instrument_key VARCHAR(50) NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    open DECIMAL(18,2) NOT NULL,
    high DECIMAL(18,2) NOT NULL,
    low DECIMAL(18,2) NOT NULL,
    close DECIMAL(18,2) NOT NULL,
    volume BIGINT NOT NULL,
    open_interest BIGINT NOT NULL,
    time_interval VARCHAR(20) NOT NULL DEFAULT '5minute',
    bb_upper DECIMAL(18,4) DEFAULT NULL,
    bb_middle DECIMAL(18,4) DEFAULT NULL,
    bb_lower DECIMAL(18,4) DEFAULT NULL,
    bb_width DECIMAL(18,4) DEFAULT NULL,
    bb_width_normalized DECIMAL(18,4) DEFAULT NULL,
    bb_width_normalized_percentage DECIMAL(18,4) DEFAULT NULL,
    ema_5 DECIMAL(18,4) DEFAULT NULL,
    ema_9 DECIMAL(18,4) DEFAULT NULL,
    ema_20 DECIMAL(18,4) DEFAULT NULL,
    ema_50 DECIMAL(18,4) DEFAULT NULL,
    atr DECIMAL(18,4) DEFAULT NULL,
    rsi DECIMAL(18,4) DEFAULT NULL,
    vwap DECIMAL(18,4) DEFAULT NULL,
    ma_9 DECIMAL(18,4) DEFAULT NULL,
    lowest_bb_width DECIMAL(18,4) DEFAULT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW() ON UPDATE NOW(),
    active BOOLEAN DEFAULT TRUE
);

CREATE INDEX idx_stock_candle_5min_instrument_key ON stock_candle_data_5min(instrument_key);
CREATE INDEX idx_stock_candle_5min_timestamp ON stock_candle_data_5min(timestamp);
CREATE INDEX idx_stock_candle_5min_instrument_timestamp ON stock_candle_data_5min(instrument_key, timestamp);
CREATE INDEX idx_stock_candle_5min_interval ON stock_candle_data_5min(time_interval);

ALTER TABLE stock_candle_data_5min 
ADD CONSTRAINT idx_stock_candle_5min_unique 
UNIQUE (instrument_key, timestamp, time_interval); 