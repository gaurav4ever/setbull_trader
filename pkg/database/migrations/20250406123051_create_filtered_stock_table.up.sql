CREATE TABLE IF NOT EXISTS filtered_stocks (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    symbol VARCHAR(20) NOT NULL,
    instrument_key VARCHAR(50) NOT NULL,
    exchange_token VARCHAR(20) NOT NULL,
    trend VARCHAR(10) NOT NULL COMMENT 'Current trend: BULLISH or BEARISH',
    current_price DECIMAL(10,2) NOT NULL,
    mamba_count INT NOT NULL,
    bullish_mamba_count INT NOT NULL,
    bearish_mamba_count INT NOT NULL,
    avg_mamba_move DECIMAL(10,2) NOT NULL DEFAULT 0 COMMENT 'Average percentage move of Mamba days',
    avg_non_mamba_move DECIMAL(10,2) NOT NULL DEFAULT 0 COMMENT 'Average percentage move of non-Mamba days',
    mamba_series JSON NOT NULL,
    non_mamba_series JSON NOT NULL,
    filter_date DATETIME NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    -- Indexes for common queries
    INDEX idx_filtered_stocks_symbol (symbol),
    INDEX idx_filtered_stocks_filter_date (filter_date),
    INDEX idx_filtered_stocks_mamba_count (mamba_count),
    
    -- Composite unique index for symbol and date queries
    UNIQUE INDEX idx_filtered_stocks_symbol_date (symbol, filter_date),

    -- Constraints
    CONSTRAINT chk_mamba_counts CHECK (
        mamba_count >= 0 AND
        bullish_mamba_count >= 0 AND
        bearish_mamba_count >= 0 AND
        mamba_count = bullish_mamba_count + bearish_mamba_count
    ),
    
    CONSTRAINT chk_current_price CHECK (current_price > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Add table comments
ALTER TABLE filtered_stocks 
COMMENT 'Stores filtered stock data with Mamba movement analysis';

-- Add column comments
ALTER TABLE filtered_stocks
MODIFY COLUMN symbol VARCHAR(20) NOT NULL COMMENT 'Stock symbol',
MODIFY COLUMN instrument_key VARCHAR(50) NOT NULL COMMENT 'Unique identifier for the instrument',
MODIFY COLUMN exchange_token VARCHAR(20) NOT NULL COMMENT 'Exchange-specific token',
MODIFY COLUMN current_price DECIMAL(10,2) NOT NULL COMMENT 'Current price of the stock',
MODIFY COLUMN mamba_count INT NOT NULL COMMENT 'Total number of Mamba moves',
MODIFY COLUMN bullish_mamba_count INT NOT NULL COMMENT 'Number of bullish Mamba moves',
MODIFY COLUMN bearish_mamba_count INT NOT NULL COMMENT 'Number of bearish Mamba moves',
MODIFY COLUMN mamba_series JSON NOT NULL COMMENT 'Array of Mamba moves (1 for bullish, -1 for bearish, 0 for none)',
MODIFY COLUMN non_mamba_series JSON NOT NULL COMMENT 'Array of non-Mamba moves (1 for non-Mamba day, 0 for Mamba day)',
MODIFY COLUMN filter_date DATETIME NOT NULL COMMENT 'Date when the stock was filtered';