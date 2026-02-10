-- TDD-driven Database Indexes for Facility Services Pagination
-- These indexes optimize the search-across-all-data approach

-- Primary composite index for facility services with filtering and pagination
-- Optimizes: facility_id filter + availability filter + price range + sorting
CREATE INDEX IF NOT EXISTS idx_facility_procedures_search_pagination
ON facility_procedures(facility_id, is_available, price, updated_at)
INCLUDE (procedure_id, currency, estimated_duration);

-- Full-text search index for procedure names and descriptions
-- Optimizes: text search across procedure names and descriptions
CREATE INDEX IF NOT EXISTS idx_procedures_fulltext_search
ON procedures USING gin(to_tsvector('english', name || ' ' || coalesce(description, '')))
WHERE is_active = true;

-- Category and activity filtering index
-- Optimizes: category filters and active procedure filtering
CREATE INDEX IF NOT EXISTS idx_procedures_category_active
ON procedures(category, is_active)
WHERE is_active = true;

-- Join optimization index for facility_procedures + procedures
-- Optimizes: JOINs between facility_procedures and procedures tables
CREATE INDEX IF NOT EXISTS idx_facility_procedures_join_optimization
ON facility_procedures(procedure_id, facility_id, is_available);

-- Price range filtering index
-- Optimizes: price-based WHERE clauses in search queries
CREATE INDEX IF NOT EXISTS idx_facility_procedures_price_range
ON facility_procedures(facility_id, price, is_available)
WHERE is_available = true;

-- Comprehensive search index combining multiple filters
-- Optimizes: complex queries with multiple WHERE conditions
CREATE INDEX IF NOT EXISTS idx_facility_procedures_comprehensive_search
ON facility_procedures(facility_id, is_available, price, estimated_duration, updated_at)
WHERE is_available = true;

-- Partial index for available procedures only (most common query pattern)
-- Optimizes: queries that filter for available procedures only
CREATE INDEX IF NOT EXISTS idx_facility_procedures_available_only
ON facility_procedures(facility_id, price, updated_at)
WHERE is_available = true;

-- Analyze tables to update statistics for query planning
ANALYZE facility_procedures;
ANALYZE procedures;
