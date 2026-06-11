-- Hierarchical summaries: scene → act → story
CREATE TABLE IF NOT EXISTS story_summaries (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    story_id    uuid NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
    node_id     uuid REFERENCES nodes(id) ON DELETE SET NULL,
    level       text NOT NULL CHECK (level IN ('scene', 'act', 'story')),
    content     text NOT NULL,
    word_count  int NOT NULL DEFAULT 0,
    created_at  timestamptz NOT NULL DEFAULT now()
);

-- scene: unique per story+node; act/story: unique per story+level (node_id = null sentinel)
CREATE UNIQUE INDEX IF NOT EXISTS idx_summaries_scene ON story_summaries(story_id, node_id) WHERE level = 'scene' AND node_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_summaries_act ON story_summaries(story_id, level) WHERE level = 'act' AND node_id IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_summaries_story_level ON story_summaries(story_id, level) WHERE level = 'story' AND node_id IS NULL;
CREATE INDEX IF NOT EXISTS idx_summaries_node ON story_summaries(node_id);
