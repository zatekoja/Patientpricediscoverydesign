-- Ensure all facilities have a non-empty scheduling_external_id slug.
UPDATE facilities
SET scheduling_external_id = lower(trim(both '-' FROM regexp_replace(replace(
  COALESCE(NULLIF(trim(scheduling_external_id), ''), NULLIF(trim(id), ''), NULLIF(trim(name), ''), 'facility-' || substring(md5(random()::text) from 1 for 8)),
  '_',
  '-'
), '[^a-z0-9]+', '-', 'g')))
WHERE scheduling_external_id IS NULL
   OR trim(scheduling_external_id) = '';
