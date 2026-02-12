-- Create fee_waivers table for 3rd-party sponsored service fee waivers
CREATE TABLE IF NOT EXISTS fee_waivers (
    id VARCHAR(255) PRIMARY KEY,
    sponsor_name VARCHAR(255) NOT NULL,
    sponsor_contact VARCHAR(255),
    facility_id VARCHAR(255) REFERENCES facilities(id) ON DELETE SET NULL,  -- NULL = applies to all facilities
    waiver_type VARCHAR(50) NOT NULL DEFAULT 'full',  -- 'full' or 'partial'
    waiver_amount DECIMAL(10, 2),  -- NULL for full waiver, fixed amount for partial
    max_uses INTEGER,  -- NULL = unlimited
    current_uses INTEGER NOT NULL DEFAULT 0,
    valid_from TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    valid_until TIMESTAMPTZ,  -- NULL = no expiry
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_fee_waivers_facility ON fee_waivers(facility_id);
CREATE INDEX IF NOT EXISTS idx_fee_waivers_active ON fee_waivers(is_active);

-- Add fee tracking columns to appointments
ALTER TABLE appointments
    ADD COLUMN IF NOT EXISTS procedure_price DECIMAL(10, 2),
    ADD COLUMN IF NOT EXISTS service_fee_amount DECIMAL(10, 2),
    ADD COLUMN IF NOT EXISTS fee_waiver_id VARCHAR(255) REFERENCES fee_waivers(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS fee_waiver_applied BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS final_amount DECIMAL(10, 2);

CREATE INDEX IF NOT EXISTS idx_appointments_fee_waiver ON appointments(fee_waiver_id);
