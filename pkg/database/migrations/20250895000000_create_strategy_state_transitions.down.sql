-- Migration: Drop strategy_state_transitions table
-- Description: Rollback migration for strategy_state_transitions table

-- Drop indexes first
DROP INDEX IF EXISTS idx_strategy_state_transitions_stock_strategy_time;
DROP INDEX IF EXISTS idx_strategy_state_transitions_transition_type;
DROP INDEX IF EXISTS idx_strategy_state_transitions_timestamp;

-- Drop the table
DROP TABLE IF EXISTS strategy_state_transitions; 