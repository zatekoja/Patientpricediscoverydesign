ALTER TABLE facilities
    ADD COLUMN IF NOT EXISTS capacity_status TEXT,
    ADD COLUMN IF NOT EXISTS avg_wait_minutes INTEGER,
    ADD COLUMN IF NOT EXISTS urgent_care_available BOOLEAN;
