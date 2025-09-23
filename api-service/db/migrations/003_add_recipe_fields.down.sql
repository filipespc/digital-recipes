-- Rollback recipe time fields and notes
-- Phase 3: AI Processing Pipeline rollback

-- Drop the new index
DROP INDEX IF EXISTS idx_recipes_status_created;

-- Revert status constraint to original
ALTER TABLE recipes
DROP CONSTRAINT recipes_status_check;

ALTER TABLE recipes
ADD CONSTRAINT recipes_status_check
CHECK (status IN ('processing', 'review_required', 'published'));

-- Remove the new columns
ALTER TABLE recipes
DROP COLUMN IF EXISTS prep_time,
DROP COLUMN IF EXISTS cook_time,
DROP COLUMN IF EXISTS total_time,
DROP COLUMN IF EXISTS notes;