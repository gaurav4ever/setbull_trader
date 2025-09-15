-- Create strategy_results_v2 table for V2 strategy engine
CREATE TABLE IF NOT EXISTS strategy_results_v2 (
    id BIGSERIAL PRIMARY KEY,
    stock_group_id VARCHAR(36) NOT NULL,
    instrument_key VARCHAR(50) NOT NULL,
    strategy_name VARCHAR(100) NOT NULL,
    strategy_version VARCHAR(20) NOT NULL,
    processing_time BIGINT NOT NULL, -- milliseconds
    memory_used BIGINT NOT NULL, -- bytes
    rows_processed INTEGER NOT NULL,
    columns_added TEXT, -- JSON array
    result_data LONGTEXT, -- JSON object
    error TEXT,
    processed_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_strategy_results_v2_stock_group_id ON strategy_results_v2(stock_group_id);
CREATE INDEX IF NOT EXISTS idx_strategy_results_v2_instrument_key ON strategy_results_v2(instrument_key);
CREATE INDEX IF NOT EXISTS idx_strategy_results_v2_strategy_name ON strategy_results_v2(strategy_name);
CREATE INDEX IF NOT EXISTS idx_strategy_results_v2_processed_at ON strategy_results_v2(processed_at);
CREATE INDEX IF NOT EXISTS idx_strategy_results_v2_created_at ON strategy_results_v2(created_at);

-- Create composite indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_strategy_results_v2_group_strategy ON strategy_results_v2(stock_group_id, strategy_name);
CREATE INDEX IF NOT EXISTS idx_strategy_results_v2_instrument_strategy ON strategy_results_v2(instrument_key, strategy_name);
CREATE INDEX IF NOT EXISTS idx_strategy_results_v2_processed_strategy ON strategy_results_v2(processed_at, strategy_name);

-- Add comments for documentation
COMMENT ON TABLE strategy_results_v2 IS 'Stores results from V2 strategy engine processing';
COMMENT ON COLUMN strategy_results_v2.stock_group_id IS 'ID of the stock group that was processed';
COMMENT ON COLUMN strategy_results_v2.instrument_key IS 'Instrument key of the stock that was processed';
COMMENT ON COLUMN strategy_results_v2.strategy_name IS 'Name of the strategy that was executed';
COMMENT ON COLUMN strategy_results_v2.strategy_version IS 'Version of the strategy that was executed';
COMMENT ON COLUMN strategy_results_v2.processing_time IS 'Time taken to process in milliseconds';
COMMENT ON COLUMN strategy_results_v2.memory_used IS 'Memory used during processing in bytes';
COMMENT ON COLUMN strategy_results_v2.rows_processed IS 'Number of data rows processed';
COMMENT ON COLUMN strategy_results_v2.columns_added IS 'JSON array of column names added by the strategy';
COMMENT ON COLUMN strategy_results_v2.result_data IS 'JSON object containing strategy-specific result data';
COMMENT ON COLUMN strategy_results_v2.error IS 'Error message if processing failed';
COMMENT ON COLUMN strategy_results_v2.processed_at IS 'Timestamp when the processing occurred'; 