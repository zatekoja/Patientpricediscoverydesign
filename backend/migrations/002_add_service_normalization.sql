-- Add normalized service name fields to procedures table
-- This migration adds display_name and normalized_tags for human-readable service names

ALTER TABLE procedures
ADD COLUMN display_name TEXT,
ADD COLUMN normalized_tags TEXT[] DEFAULT '{}';

-- Create GIN index for efficient tag searches
CREATE INDEX IF NOT EXISTS idx_procedures_normalized_tags 
ON procedures USING GIN(normalized_tags);

-- Initialize display_name from name for existing records
UPDATE procedures 
SET display_name = name 
WHERE display_name IS NULL;

-- Add NOT NULL constraint after initialization
ALTER TABLE procedures
ALTER COLUMN display_name SET NOT NULL;
