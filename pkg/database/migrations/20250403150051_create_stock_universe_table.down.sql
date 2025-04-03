-- Drop indexes first
DROP INDEX IF EXISTS idx_stock_universe_symbol;
DROP INDEX IF EXISTS idx_stock_universe_instrument_key;
DROP INDEX IF EXISTS idx_stock_universe_is_selected;

-- Drop the table
DROP TABLE IF EXISTS stock_universe;