-- Migration: Create strategy_parameters_v2 table
-- Description: Table to store strategy parameters for each candle across all strategies

CREATE TABLE strategy_parameters_v2 (
    id BIGSERIAL PRIMARY KEY,
    stock_id VARCHAR(50) NOT NULL,
    strategy_name VARCHAR(50) NOT NULL,
    candle_timestamp TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Common parameters across all strategies
    in_long_trade BOOLEAN DEFAULT FALSE,
    in_short_trade BOOLEAN DEFAULT FALSE,
    can_generate_long BOOLEAN DEFAULT TRUE,
    can_generate_short BOOLEAN DEFAULT TRUE,
    
    -- 1ST_ENTRY strategy parameters
    mr_high DECIMAL(10,2),
    mr_low DECIMAL(10,2),
    mr_high_with_buffer DECIMAL(10,2),
    mr_low_with_buffer DECIMAL(10,2),
    buffer_percentage DECIMAL(5,4),
    
    -- 2_30_ENTRY strategy parameters
    entry_time TIME,
    range_high DECIMAL(10,2),
    range_low DECIMAL(10,2),
    range_high_entry_price DECIMAL(10,2),
    range_low_entry_price DECIMAL(10,2),
    direction VARCHAR(10),
    
    -- BB_WIDTH_ENTRY strategy parameters
    bb_width_threshold DECIMAL(5,4),
    bb_period INTEGER,
    bb_std_dev DECIMAL(5,2),
    current_bb_width DECIMAL(10,2),
    lowest_bb_width DECIMAL(10,2),
    squeeze_detected BOOLEAN DEFAULT FALSE,
    squeeze_start_time TIMESTAMP,
    squeeze_candle_count INTEGER,
    bb_upper DECIMAL(10,2),
    bb_lower DECIMAL(10,2),
    bb_middle DECIMAL(10,2),
    
    -- Strategy state and metadata
    strategy_state JSONB,
    metadata JSONB,
    
    -- Soft delete
    active BOOLEAN DEFAULT TRUE
);

-- Create indexes for efficient querying
CREATE INDEX idx_strategy_parameters_stock_strategy_timestamp 
ON strategy_parameters_v2 (stock_id, strategy_name, candle_timestamp);

CREATE INDEX idx_strategy_parameters_strategy_timestamp 
ON strategy_parameters_v2 (strategy_name, candle_timestamp);

CREATE INDEX idx_strategy_parameters_candle_timestamp 
ON strategy_parameters_v2 (candle_timestamp);

CREATE INDEX idx_strategy_parameters_active 
ON strategy_parameters_v2 (active) WHERE active = TRUE;

-- Create unique constraint to prevent duplicate entries
CREATE UNIQUE INDEX idx_strategy_parameters_unique 
ON strategy_parameters_v2 (stock_id, strategy_name, candle_timestamp) 
WHERE active = TRUE;

-- Add comments for documentation
COMMENT ON TABLE strategy_parameters_v2 IS 'Stores strategy parameters for each candle across all strategies';
COMMENT ON COLUMN strategy_parameters_v2.stock_id IS 'Stock identifier';
COMMENT ON COLUMN strategy_parameters_v2.strategy_name IS 'Strategy name (1ST_ENTRY, 2_30_ENTRY, BB_WIDTH_ENTRY)';
COMMENT ON COLUMN strategy_parameters_v2.candle_timestamp IS '5-minute candle timestamp';
COMMENT ON COLUMN strategy_parameters_v2.strategy_state IS 'JSON object containing strategy state';
COMMENT ON COLUMN strategy_parameters_v2.metadata IS 'JSON object containing additional metadata'; 