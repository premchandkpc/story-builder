-- Add composite B-tree index for story summary queries
CREATE INDEX IF NOT EXISTS idx_story_summaries_story_level ON story_summaries (story_id, level);
