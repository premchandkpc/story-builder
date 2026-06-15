-- STORY BUILDER V3: Story → Chapter → Scene hierarchy
-- Migrates from flat nodes/edges to chapters/scenes/scene_edges.

-- 1. Chapters
CREATE TABLE IF NOT EXISTS chapters (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    story_id        uuid NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
    title           text NOT NULL DEFAULT '',
    goal            text NOT NULL DEFAULT '',
    order_index     int NOT NULL DEFAULT 0,
    summary         text NOT NULL DEFAULT '',
    status          text NOT NULL DEFAULT 'draft'
                    CHECK (status IN ('draft', 'active', 'completed', 'archived')),
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (story_id, order_index)
);

CREATE INDEX IF NOT EXISTS idx_chapters_story ON chapters(story_id);

-- 2. Create default Chapter 1 for every story
INSERT INTO chapters (story_id, title, order_index)
SELECT DISTINCT n.story_id, 'Chapter 1', 0
FROM nodes n
WHERE NOT EXISTS (SELECT 1 FROM chapters c WHERE c.story_id = n.story_id);

INSERT INTO chapters (story_id, title, order_index)
SELECT s.id, 'Chapter 1', 0
FROM stories s
WHERE NOT EXISTS (SELECT 1 FROM chapters c WHERE c.story_id = s.id);

-- 3. Create scenes table (replaces nodes)
CREATE TABLE IF NOT EXISTS scenes (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    chapter_id          uuid NOT NULL REFERENCES chapters(id) ON DELETE CASCADE,
    story_id            uuid NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
    title               text NOT NULL DEFAULT '',
    beat_intent         text NOT NULL DEFAULT '',
    character_refs      uuid[] NOT NULL DEFAULT '{}',
    location_ref        uuid,
    pov                 text NOT NULL DEFAULT '',
    tone                text NOT NULL DEFAULT '',
    target_words        int NOT NULL DEFAULT 300,
    scene_structure     jsonb NOT NULL DEFAULT '{}',
    parent_scene_id     uuid REFERENCES scenes(id) ON DELETE SET NULL,
    timeline_position   text NOT NULL DEFAULT '',
    flow_type           text NOT NULL DEFAULT 'dialogue'
                        CHECK (flow_type IN ('monologue','dialogue','round_robin','parallel','action','silent','custom')),
    max_turns           int NOT NULL DEFAULT 5,
    status              text NOT NULL DEFAULT 'draft'
                        CHECK (status IN ('draft', 'generated', 'accepted', 'stale')),
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_scenes_chapter ON scenes(chapter_id);
CREATE INDEX IF NOT EXISTS idx_scenes_story ON scenes(story_id);
CREATE INDEX IF NOT EXISTS idx_scenes_parent ON scenes(parent_scene_id);

-- 4. Migrate data from nodes → scenes
INSERT INTO scenes (
    id, chapter_id, story_id, beat_intent,
    character_refs, location_ref, pov, tone, target_words,
    scene_structure, flow_type, max_turns, status,
    created_at, updated_at
)
SELECT
    n.id,
    COALESCE(
        (SELECT c.id FROM chapters c WHERE c.story_id = n.story_id ORDER BY c.order_index LIMIT 1),
        (SELECT id FROM chapters LIMIT 1)
    ),
    n.story_id,
    n.beat_intent,
    n.character_refs,
    n.location_ref,
    n.pov,
    n.tone,
    n.target_words,
    COALESCE(n.scene_structure, '{}'::jsonb),
    COALESCE(ss.flow_type, 'dialogue'),
    GREATEST(COALESCE(ss.max_turns, 5), 1),
    n.status,
    n.created_at,
    n.updated_at
FROM nodes n
LEFT JOIN LATERAL jsonb_to_record(COALESCE(n.scene_structure, '{}'::jsonb)) AS ss(flow_type text, max_turns int) ON true;

-- 5. Drop old FKs pointing to nodes(id)
ALTER TABLE IF EXISTS generations      DROP CONSTRAINT IF EXISTS generations_node_id_fkey;
ALTER TABLE IF EXISTS character_state  DROP CONSTRAINT IF EXISTS character_state_as_of_node_fkey;
ALTER TABLE IF EXISTS story_summaries  DROP CONSTRAINT IF EXISTS story_summaries_node_id_fkey;
ALTER TABLE IF EXISTS scene_turns      DROP CONSTRAINT IF EXISTS scene_turns_node_id_fkey;

-- 6. Rename referencing columns
ALTER TABLE IF EXISTS generations      RENAME COLUMN node_id TO scene_id;
ALTER TABLE IF EXISTS character_state  RENAME COLUMN as_of_node TO as_of_scene;
ALTER TABLE IF EXISTS story_summaries  RENAME COLUMN node_id TO scene_id;
ALTER TABLE IF EXISTS scene_turns      RENAME COLUMN node_id TO scene_id;

-- 7. Add new FKs pointing to scenes(id)
ALTER TABLE IF EXISTS generations      ADD CONSTRAINT fk_generations_scene    FOREIGN KEY (scene_id)    REFERENCES scenes(id) ON DELETE CASCADE;
ALTER TABLE IF EXISTS character_state  ADD CONSTRAINT fk_char_state_scene    FOREIGN KEY (as_of_scene) REFERENCES scenes(id) ON DELETE CASCADE;
ALTER TABLE IF EXISTS story_summaries  ADD CONSTRAINT fk_summaries_scene     FOREIGN KEY (scene_id)    REFERENCES scenes(id) ON DELETE SET NULL;
ALTER TABLE IF EXISTS scene_turns      ADD CONSTRAINT fk_turns_scene         FOREIGN KEY (scene_id)    REFERENCES scenes(id) ON DELETE CASCADE;

-- 8. Create scene_edges (replaces edges)
CREATE TABLE IF NOT EXISTS scene_edges (
    story_id    uuid NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
    from_scene  uuid NOT NULL REFERENCES scenes(id) ON DELETE CASCADE,
    to_scene    uuid NOT NULL REFERENCES scenes(id) ON DELETE CASCADE,
    edge_type   text NOT NULL DEFAULT 'seq'
                CHECK (edge_type IN ('seq', 'fork', 'join', 'choice', 'parallel')),
    condition   text NOT NULL DEFAULT '',
    PRIMARY KEY (story_id, from_scene, to_scene)
);

CREATE INDEX IF NOT EXISTS idx_scene_edges_from ON scene_edges(from_scene);
CREATE INDEX IF NOT EXISTS idx_scene_edges_to   ON scene_edges(to_scene);

-- 9. Migrate edges → scene_edges
INSERT INTO scene_edges (story_id, from_scene, to_scene, edge_type)
SELECT e.story_id, e.from_node, e.to_node, e.edge_type
FROM edges e;

-- 10. Drop old tables
DROP TABLE IF EXISTS edges CASCADE;
DROP TABLE IF EXISTS nodes CASCADE;

-- 11. Enhance stories table
ALTER TABLE IF EXISTS stories ADD COLUMN IF NOT EXISTS genre          text NOT NULL DEFAULT '';
ALTER TABLE IF EXISTS stories ADD COLUMN IF NOT EXISTS theme          text NOT NULL DEFAULT '';
ALTER TABLE IF EXISTS stories ADD COLUMN IF NOT EXISTS main_prompt    text NOT NULL DEFAULT '';
ALTER TABLE IF EXISTS stories ADD COLUMN IF NOT EXISTS general_prompt text NOT NULL DEFAULT '';
