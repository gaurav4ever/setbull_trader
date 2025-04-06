-- Drop indexes (not strictly necessary as they'll be dropped with the table)
DROP INDEX IF EXISTS idx_filtered_stocks_symbol ON filtered_stocks;
DROP INDEX IF EXISTS idx_filtered_stocks_filter_date ON filtered_stocks;
DROP INDEX IF EXISTS idx_filtered_stocks_mamba_count ON filtered_stocks;
DROP INDEX IF EXISTS idx_filtered_stocks_symbol_date ON filtered_stocks;

-- Drop the table
DROP TABLE IF EXISTS filtered_stocks;