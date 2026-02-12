-- Add search concepts and enrichment status tracking to procedure_enrichments
ALTER TABLE procedure_enrichments
    ADD COLUMN IF NOT EXISTS search_concepts JSONB DEFAULT '{}';

ALTER TABLE procedure_enrichments
    ADD COLUMN IF NOT EXISTS enrichment_status VARCHAR(20) DEFAULT 'pending';

ALTER TABLE procedure_enrichments
    ADD COLUMN IF NOT EXISTS enrichment_version INT DEFAULT 1;

ALTER TABLE procedure_enrichments
    ADD COLUMN IF NOT EXISTS retry_count INT DEFAULT 0;

ALTER TABLE procedure_enrichments
    ADD COLUMN IF NOT EXISTS last_error TEXT;

-- GIN index on search_concepts for fast JSONB queries
CREATE INDEX IF NOT EXISTS idx_enrichments_search_concepts
    ON procedure_enrichments USING GIN (search_concepts jsonb_path_ops);

-- Index on enrichment_status for backfill queries
CREATE INDEX IF NOT EXISTS idx_enrichments_status
    ON procedure_enrichments (enrichment_status);
