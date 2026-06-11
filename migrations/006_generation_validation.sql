ALTER TABLE generations
ADD COLUMN IF NOT EXISTS validation_result jsonb NOT NULL DEFAULT '{}';
