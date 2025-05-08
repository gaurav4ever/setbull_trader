-- Create stock_groups table
CREATE TABLE IF NOT EXISTS stock_groups (
    id VARCHAR(36) PRIMARY KEY,
    entry_type VARCHAR(32) NOT NULL,
    status VARCHAR(16) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- Create stock_group_stocks join table
CREATE TABLE IF NOT EXISTS stock_group_stocks (
    id VARCHAR(36) PRIMARY KEY,
    group_id VARCHAR(36) NOT NULL,
    stock_id VARCHAR(36) NOT NULL,
    CONSTRAINT fk_group FOREIGN KEY (group_id) REFERENCES stock_groups(id) ON DELETE CASCADE,
    CONSTRAINT fk_stock FOREIGN KEY (stock_id) REFERENCES stocks(id) ON DELETE CASCADE
);

-- Enforce max 5 stocks per group (application-level validation recommended)
CREATE INDEX idx_stock_group_stocks_group_id ON stock_group_stocks(group_id);
CREATE INDEX idx_stock_group_stocks_stock_id ON stock_group_stocks(stock_id); 