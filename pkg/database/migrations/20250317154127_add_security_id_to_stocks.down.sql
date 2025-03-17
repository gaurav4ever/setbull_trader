DROP INDEX idx_stocks_security_id ON stocks;
ALTER TABLE stocks DROP COLUMN security_id;