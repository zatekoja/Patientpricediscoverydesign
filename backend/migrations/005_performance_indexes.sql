-- Performance optimization indexes for read-heavy workload
-- Run this migration during low-traffic periods

-- ============================================================================
-- FACILITIES TABLE INDEXES
-- ============================================================================

-- Composite index for filtered queries (type + active status + rating)
-- Covers common queries: "active facilities of type X ordered by rating"
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_facilities_type_active_rating
  ON facilities(facility_type, is_active, rating DESC)
  WHERE is_active = true;

-- Covering index for search results (includes commonly fetched columns)
-- Reduces need to access table heap for these columns
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_facilities_search_covering
  ON facilities(is_active, facility_type, rating DESC)
  INCLUDE (id, name, review_count, latitude, longitude);

-- Partial index for active facilities only (smaller, faster)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_facilities_active_only
  ON facilities(id, created_at DESC)
  WHERE is_active = true;

-- GiST index for geospatial queries (PostGIS-style)
-- Requires PostGIS extension for ST_MakePoint, or use simpler approach
-- If PostGIS not available, keep the existing btree index on (latitude, longitude)
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_facilities_location_gist
--   ON facilities USING GIST(ST_MakePoint(longitude, latitude));

-- B-tree index for rating-based sorting with active filter
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_facilities_rating_desc
  ON facilities(rating DESC, review_count DESC)
  WHERE is_active = true;

-- Index for name-based search (case-insensitive)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_facilities_name_lower
  ON facilities(LOWER(name));

-- Index for city-based filtering (common use case)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_facilities_city_active
  ON facilities(city, is_active, rating DESC)
  WHERE is_active = true;

-- ============================================================================
-- FACILITY_PROCEDURES TABLE INDEXES
-- ============================================================================

-- Composite index for price range queries
-- Covers: "available procedures at facility X within price range"
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_facility_procedures_price_range
  ON facility_procedures(facility_id, is_available, price ASC);

-- Covering index for procedure lookups
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_facility_procedures_lookup
  ON facility_procedures(procedure_id, is_available)
  INCLUDE (facility_id, price, currency, estimated_duration);

-- Index for finding cheapest procedures
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_facility_procedures_cheapest
  ON facility_procedures(procedure_id, price ASC)
  WHERE is_available = true;

-- ============================================================================
-- PROCEDURES TABLE INDEXES
-- ============================================================================

-- Composite index for category filtering
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_procedures_category_active
  ON procedures(category, is_active, name);

-- Index for name-based search (case-insensitive)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_procedures_name_lower
  ON procedures(LOWER(name))
  WHERE is_active = true;

-- ============================================================================
-- FACILITY_INSURANCE TABLE INDEXES
-- ============================================================================

-- Composite index for accepted insurance queries
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_facility_insurance_accepted
  ON facility_insurance(insurance_provider_id, is_accepted, facility_id)
  WHERE is_accepted = true;

-- Reverse index for facility lookup
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_facility_insurance_facility_lookup
  ON facility_insurance(facility_id, is_accepted)
  INCLUDE (insurance_provider_id);

-- ============================================================================
-- INSURANCE_PROVIDERS TABLE INDEXES
-- ============================================================================

-- Index for active insurance providers with name search
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_insurance_active_name
  ON insurance_providers(is_active, name)
  WHERE is_active = true;

-- ============================================================================
-- APPOINTMENTS TABLE (if exists)
-- ============================================================================

-- Check if appointments table exists before creating indexes
DO $$
BEGIN
  IF EXISTS (SELECT FROM information_schema.tables
             WHERE table_name = 'appointments') THEN

    -- Composite index for user's appointments
    CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_appointments_user_status
      ON appointments(user_id, status, appointment_date DESC);

    -- Index for facility's appointments
    CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_appointments_facility_date
      ON appointments(facility_id, appointment_date, status);

    -- Index for upcoming appointments
    CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_appointments_upcoming
      ON appointments(appointment_date ASC)
      WHERE status IN ('scheduled', 'confirmed');
  END IF;
END
$$;

-- ============================================================================
-- FEEDBACK TABLE (if exists)
-- ============================================================================

DO $$
BEGIN
  IF EXISTS (SELECT FROM information_schema.tables
             WHERE table_name = 'feedback') THEN

    -- Index for facility feedback with rating
    CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_feedback_facility_rating
      ON feedback(facility_id, rating DESC, created_at DESC);

    -- Index for recent feedback
    CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_feedback_recent
      ON feedback(created_at DESC)
      INCLUDE (facility_id, rating);
  END IF;
END
$$;

-- ============================================================================
-- STATISTICS UPDATE
-- ============================================================================

-- Update table statistics for better query planning
ANALYZE facilities;
ANALYZE procedures;
ANALYZE facility_procedures;
ANALYZE facility_insurance;
ANALYZE insurance_providers;

-- Optionally analyze appointments and feedback if they exist
DO $$
BEGIN
  IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'appointments') THEN
    EXECUTE 'ANALYZE appointments';
  END IF;

  IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'feedback') THEN
    EXECUTE 'ANALYZE feedback';
  END IF;
END
$$;

-- ============================================================================
-- NOTES
-- ============================================================================

-- The CONCURRENTLY option allows indexes to be built without locking the table
-- This is important for production deployments with minimal downtime
-- However, CONCURRENTLY cannot be run inside a transaction block

-- Monitor index usage with:
-- SELECT schemaname, tablename, indexname, idx_scan, idx_tup_read, idx_tup_fetch
-- FROM pg_stat_user_indexes
-- WHERE schemaname = 'public'
-- ORDER BY idx_scan DESC;

-- Monitor slow queries with:
-- SELECT query, mean_exec_time, calls
-- FROM pg_stat_statements
-- WHERE mean_exec_time > 100
-- ORDER BY mean_exec_time DESC
-- LIMIT 20;

