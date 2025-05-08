-- Trade Parameters Table
ALTER TABLE trade_parameters
  DROP COLUMN IF EXISTS ps_type,
  DROP COLUMN IF EXISTS entry_type;

-- Level Entries Table
ALTER TABLE level_entries
  DROP COLUMN IF EXISTS ps_type,
  DROP COLUMN IF EXISTS entry_desc;