-- Remove BBW Dashboard columns from stock_candle_data_5min table
ALTER TABLE stock_candle_data_5min 
DROP COLUMN distance_from_min_percent,
DROP COLUMN contracting_sequence_count,
DROP COLUMN candles_in_range_count,
DROP COLUMN alert_triggered,
DROP COLUMN alert_triggered_at,
DROP COLUMN comment;

-- Remove indexes
DROP INDEX IF EXISTS idx_bbw_alert_status ON stock_candle_data_5min;
DROP INDEX IF EXISTS idx_bbw_width_range ON stock_candle_data_5min;
DROP INDEX IF EXISTS idx_candles_in_range ON stock_candle_data_5min;
DROP INDEX IF EXISTS idx_distance_from_min ON stock_candle_data_5min; 