-- Migration: Create strategy_state_transitions table
-- Description: Audit table for tracking strategy state transitions

CREATE TABLE strategy_state_transitions (
    id VARCHAR(36) PRIMARY KEY,
    stock_id VARCHAR(36) NOT NULL,
    strategy_name VARCHAR(100) NOT NULL,
    from_state JSON,
    to_state JSON,
    transition_type VARCHAR(50) NOT NULL,
    reason VARCHAR(255),
    metadata JSON,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for optimal performance
CREATE INDEX idx_strategy_state_transitions_stock_strategy_time ON strategy_state_transitions (stock_id, strategy_name, timestamp);
CREATE INDEX idx_strategy_state_transitions_transition_type ON strategy_state_transitions (transition_type);
CREATE INDEX idx_strategy_state_transitions_timestamp ON strategy_state_transitions (timestamp);

-- Add comments for documentation
ALTER TABLE strategy_state_transitions COMMENT = 'Audit table for tracking strategy state transitions'; 