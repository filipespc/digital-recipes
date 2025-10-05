-- Enable pg_trgm extension for trigram similarity search
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Add trigram index to pantry_items.name for fast fuzzy search
CREATE INDEX IF NOT EXISTS idx_pantry_items_name_trgm ON pantry_items USING gin (name gin_trgm_ops);

-- Add comment to explain the index
COMMENT ON INDEX idx_pantry_items_name_trgm IS 'Trigram index for fuzzy text search on pantry item names';
