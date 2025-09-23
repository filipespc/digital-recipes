-- Add recipe time fields and notes for AI processing
-- Phase 3: AI Processing Pipeline enhancement

-- Add time fields for better recipe structure
ALTER TABLE recipes
ADD COLUMN prep_time VARCHAR(50),
ADD COLUMN cook_time VARCHAR(50),
ADD COLUMN total_time VARCHAR(50),
ADD COLUMN notes TEXT;

-- Update status constraint to include 'failed' status
ALTER TABLE recipes
DROP CONSTRAINT recipes_status_check;

ALTER TABLE recipes
ADD CONSTRAINT recipes_status_check
CHECK (status IN ('processing', 'review_required', 'published', 'failed'));

-- Add index for better performance on status queries
CREATE INDEX IF NOT EXISTS idx_recipes_status_created ON recipes(status, created_at DESC);