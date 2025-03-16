DROP INDEX idx_stocks_active ON stocks;
ALTER TABLE stocks 
    DROP COLUMN active,
    DROP COLUMN created_at,
    DROP COLUMN updated_at;

DROP INDEX idx_trade_parameters_active ON trade_parameters;
ALTER TABLE trade_parameters 
    DROP COLUMN active,
    DROP COLUMN created_at,
    DROP COLUMN updated_at;

DROP INDEX idx_execution_plans_active ON execution_plans;
ALTER TABLE execution_plans 
    DROP COLUMN active,
    DROP COLUMN updated_at;

DROP INDEX idx_level_entries_active ON level_entries;
ALTER TABLE level_entries 
    DROP COLUMN active,
    DROP COLUMN created_at,
    DROP COLUMN updated_at;

DROP INDEX idx_order_executions_active ON order_executions;
ALTER TABLE order_executions 
    DROP COLUMN active,
    DROP COLUMN created_at,
    DROP COLUMN updated_at;