ALTER TABLE facilities
    ADD COLUMN IF NOT EXISTS ward_statuses JSONB;
