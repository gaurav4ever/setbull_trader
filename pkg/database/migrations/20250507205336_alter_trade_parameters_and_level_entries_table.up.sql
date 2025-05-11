-- Trade Parameters Table
ALTER TABLE trade_parameters
  ADD COLUMN ps_type VARCHAR(16),
  ADD COLUMN entry_type VARCHAR(32);

-- Level Entries Table
ALTER TABLE level_entries
  ADD COLUMN ps_type VARCHAR(16),
  ADD COLUMN entry_desc VARCHAR(16);