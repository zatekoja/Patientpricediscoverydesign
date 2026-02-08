-- Remove legacy seeded data that does not originate from provider ingestion.
-- Provider-ingested records use deterministic prefixes (file_price_list_ / proc_ / fp_).

BEGIN;

DELETE FROM facility_procedures
WHERE id NOT LIKE 'fp_%';

DELETE FROM procedures
WHERE id NOT LIKE 'proc_%';

DELETE FROM facilities
WHERE id NOT LIKE 'file_price_list_%';

COMMIT;
