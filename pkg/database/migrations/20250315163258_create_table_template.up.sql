CREATE TABLE IF NOT EXISTS stocks (
    id VARCHAR(36) PRIMARY KEY,
    symbol VARCHAR(10) NOT NULL,
    name VARCHAR(100) NOT NULL,
    current_price DECIMAL(10,2) NOT NULL,
    is_selected BOOLEAN DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS trade_parameters (
    id VARCHAR(36) PRIMARY KEY,
    stock_id VARCHAR(36) NOT NULL REFERENCES stocks(id) ON DELETE CASCADE,
    starting_price DECIMAL(10,2) NOT NULL,
    sl_percentage DECIMAL(5,2) NOT NULL,
    risk_amount DECIMAL(10,2) NOT NULL DEFAULT 30.0,
    trade_side VARCHAR(4) NOT NULL CHECK (trade_side IN ('BUY', 'SELL'))
);

CREATE TABLE IF NOT EXISTS execution_plans (
    id VARCHAR(36) PRIMARY KEY,
    stock_id VARCHAR(36) NOT NULL REFERENCES stocks(id) ON DELETE CASCADE,
    parameters_id VARCHAR(36) NOT NULL REFERENCES trade_parameters(id) ON DELETE CASCADE,
    total_quantity INT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS level_entries (
    id VARCHAR(36) PRIMARY KEY,
    execution_plan_id VARCHAR(36) NOT NULL REFERENCES execution_plans(id) ON DELETE CASCADE,
    fib_level DECIMAL(5,2) NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    quantity INT NOT NULL,
    description VARCHAR(50) NOT NULL
);

CREATE TABLE IF NOT EXISTS order_executions (
    id VARCHAR(36) PRIMARY KEY,
    execution_plan_id VARCHAR(36) NOT NULL REFERENCES execution_plans(id),
    status VARCHAR(20) NOT NULL,
    executed_at TIMESTAMP NOT NULL DEFAULT NOW(),
    error_message TEXT
);

CREATE INDEX idx_stocks_symbol ON stocks(symbol);
CREATE INDEX idx_stocks_is_selected ON stocks(is_selected);
CREATE INDEX idx_trade_parameters_stock_id ON trade_parameters(stock_id);
CREATE INDEX idx_execution_plans_stock_id ON execution_plans(stock_id);
CREATE INDEX idx_level_entries_execution_plan_id ON level_entries(execution_plan_id);
CREATE INDEX idx_order_executions_execution_plan_id ON order_executions(execution_plan_id);