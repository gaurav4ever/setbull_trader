-- Migration: Create strategy_parameters_v2 table
-- Description: Primary table for storing V2 strategy parameters and state

CREATE TABLE strategy_parameters_v2 (
    id VARCHAR(36) PRIMARY KEY,
    stock_id VARCHAR(36) NOT NULL,
    strategy_name VARCHAR(100) NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    parameters JSON NOT NULL,
    state_data JSON,
    metadata JSON,
    version INT DEFAULT 1,
    is_valid BOOLEAN DEFAULT true,
    is_synchronized BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- Create indexes for optimal performance
CREATE INDEX idx_strategy_parameters_v2_stock_strategy ON strategy_parameters_v2 (stock_id, strategy_name);
CREATE INDEX idx_strategy_parameters_v2_timestamp ON strategy_parameters_v2 (timestamp);
CREATE INDEX idx_strategy_parameters_v2_strategy_timestamp ON strategy_parameters_v2 (strategy_name, timestamp);
CREATE INDEX idx_strategy_parameters_v2_valid_sync ON strategy_parameters_v2 (is_valid, is_synchronized);

-- Add comments for documentation
ALTER TABLE strategy_parameters_v2 COMMENT = 'Stores V2 strategy parameters and state for each stock and strategy combination'; 