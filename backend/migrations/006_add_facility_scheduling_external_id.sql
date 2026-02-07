ALTER TABLE facilities
ADD COLUMN IF NOT EXISTS scheduling_external_id TEXT;

CREATE INDEX IF NOT EXISTS idx_facilities_scheduling_external_id
ON facilities (scheduling_external_id);
