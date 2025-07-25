-- Add BBW Dashboard columns to stock_candle_data_5min table
ALTER TABLE stock_candle_data_5min 
ADD COLUMN distance_from_min_percent DECIMAL(5,2) DEFAULT NULL COMMENT 'Percentage distance from historical minimum',
ADD COLUMN contracting_sequence_count INT DEFAULT 0 COMMENT 'Consecutive contracting candles',
ADD COLUMN candles_in_range_count INT DEFAULT 0 COMMENT 'Consecutive candles within optimal BBW range (0.1% of min)',
ADD COLUMN alert_triggered BOOLEAN DEFAULT FALSE,
ADD COLUMN alert_triggered_at TIMESTAMP NULL,
ADD COLUMN comment TEXT NULL;

-- Add indexes for BBW dashboard queries
CREATE INDEX idx_bbw_alert_status ON stock_candle_data_5min(instrument_key, alert_triggered);
CREATE INDEX idx_bbw_width_range ON stock_candle_data_5min(bb_width, instrument_key);
CREATE INDEX idx_candles_in_range ON stock_candle_data_5min(instrument_key, candles_in_range_count);
CREATE INDEX idx_distance_from_min ON stock_candle_data_5min(distance_from_min_percent, instrument_key); 