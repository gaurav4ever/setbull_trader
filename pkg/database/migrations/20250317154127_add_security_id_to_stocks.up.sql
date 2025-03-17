ALTER TABLE stocks
ADD COLUMN security_id VARCHAR(20) NULL;

CREATE INDEX idx_stocks_security_id ON stocks(security_id);