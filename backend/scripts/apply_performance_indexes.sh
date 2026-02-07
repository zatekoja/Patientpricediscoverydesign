#!/bin/bash

# Script to apply performance optimization indexes to the database
# This is safe to run in production as it uses CONCURRENTLY to avoid table locks

set -e

echo "============================================"
echo "Performance Index Migration"
echo "============================================"
echo ""

# Check if PostgreSQL container is running
if ! docker ps | grep -q ppd_postgres; then
    echo "❌ Error: PostgreSQL container (ppd_postgres) is not running"
    echo "   Please start it with: docker-compose up -d postgres"
    exit 1
fi

echo "✓ PostgreSQL container is running"
echo ""

# Check if database exists
if ! docker exec ppd_postgres psql -U postgres -lqt | cut -d \| -f 1 | grep -qw patient_price_discovery; then
    echo "❌ Error: Database 'patient_price_discovery' does not exist"
    exit 1
fi

echo "✓ Database 'patient_price_discovery' exists"
echo ""

# Show current indexes
echo "Current indexes on facilities table:"
docker exec ppd_postgres psql -U postgres -d patient_price_discovery \
    -c "SELECT indexname FROM pg_indexes WHERE tablename = 'facilities' AND schemaname = 'public';" 2>/dev/null || true
echo ""

# Confirm execution
read -p "Apply performance indexes? This will create ~15 indexes (y/n): " -n 1 -r
echo ""
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted"
    exit 0
fi

echo ""
echo "Applying performance indexes..."
echo "This may take 5-10 minutes depending on data size"
echo ""

# Apply the migration
docker exec -i ppd_postgres psql -U postgres -d patient_price_discovery < migrations/005_performance_indexes.sql

echo ""
echo "✓ Migration completed successfully!"
echo ""

# Show new indexes
echo "Indexes after migration:"
docker exec ppd_postgres psql -U postgres -d patient_price_discovery \
    -c "SELECT tablename, indexname, indexdef FROM pg_indexes WHERE schemaname = 'public' ORDER BY tablename, indexname;"

echo ""
echo "============================================"
echo "Performance indexes applied successfully!"
echo "============================================"
echo ""
echo "Next steps:"
echo "1. Restart API services: docker-compose restart api graphql"
echo "2. Monitor query performance"
echo "3. Check index usage: SELECT * FROM pg_stat_user_indexes;"
echo ""

