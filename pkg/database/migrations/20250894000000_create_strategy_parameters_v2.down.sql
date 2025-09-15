-- Migration: Drop strategy_parameters_v2 table
-- Description: Rollback migration for strategy_parameters_v2 table

-- Drop indexes first
DROP INDEX IF EXISTS idx_strategy_parameters_v2_stock_strategy;
DROP INDEX IF EXISTS idx_strategy_parameters_v2_timestamp;
DROP INDEX IF EXISTS idx_strategy_parameters_v2_strategy_timestamp;
DROP INDEX IF EXISTS idx_strategy_parameters_v2_valid_sync;

-- Drop the table
DROP TABLE IF EXISTS strategy_parameters_v2; 