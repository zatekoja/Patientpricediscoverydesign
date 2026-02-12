-- Split 'administrative' category into 'registration' (for service fees) and 'administrative' (for reports/certs)
-- This fixes the issue where service fees were inflated by including optional administrative items.

UPDATE procedures 
SET category = 'registration' 
WHERE category = 'administrative' 
  AND (
    LOWER(name) LIKE '%card%' 
    OR LOWER(name) LIKE '%folder%' 
    OR LOWER(name) LIKE '%registration%'
    OR LOWER(name) LIKE '%chart%'
  );
