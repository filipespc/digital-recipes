-- Drop trigram index
DROP INDEX IF EXISTS idx_pantry_items_name_trgm;

-- Drop pg_trgm extension
DROP EXTENSION IF EXISTS pg_trgm;
