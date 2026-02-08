-- Migration: Add facility_wards table for ward-level capacity tracking
-- This allows facilities to report capacity status for specific wards/departments
-- (e.g., maternity, pharmacy, inpatient, emergency) rather than just facility-wide

-- Create facility_wards table
CREATE TABLE IF NOT EXISTS facility_wards (
    id VARCHAR(255) PRIMARY KEY,
    facility_id VARCHAR(255) NOT NULL REFERENCES facilities(id) ON DELETE CASCADE,
    ward_name VARCHAR(100) NOT NULL,
    ward_type VARCHAR(50), -- e.g., 'maternity', 'pharmacy', 'inpatient', 'emergency', 'surgery', 'icu', 'pediatrics', 'radiology', 'laboratory'
    capacity_status TEXT,
    avg_wait_minutes INTEGER,
    urgent_care_available BOOLEAN,
    last_updated TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(facility_id, ward_name)
);

-- Indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_facility_wards_facility ON facility_wards(facility_id);
CREATE INDEX IF NOT EXISTS idx_facility_wards_type ON facility_wards(ward_type);
CREATE INDEX IF NOT EXISTS idx_facility_wards_status ON facility_wards(capacity_status) WHERE capacity_status IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_facility_wards_updated ON facility_wards(last_updated);

-- Add comment for documentation
COMMENT ON TABLE facility_wards IS 'Stores ward/department-level capacity information for facilities';
COMMENT ON COLUMN facility_wards.ward_type IS 'Type of ward: maternity, pharmacy, inpatient, emergency, surgery, icu, pediatrics, radiology, laboratory, or custom';
COMMENT ON COLUMN facility_wards.capacity_status IS 'Capacity status for this ward: available, busy, full, or closed';
