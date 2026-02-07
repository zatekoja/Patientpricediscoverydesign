CREATE TABLE IF NOT EXISTS procedure_enrichments (
    id VARCHAR(255) PRIMARY KEY,
    procedure_id VARCHAR(255) NOT NULL UNIQUE REFERENCES procedures(id) ON DELETE CASCADE,
    description TEXT,
    prep_steps JSONB,
    risks JSONB,
    recovery JSONB,
    provider TEXT,
    model TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_procedure_enrichments_procedure_id
    ON procedure_enrichments (procedure_id);
