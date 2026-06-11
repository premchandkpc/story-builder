CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "vector";

-- CANON: never UPDATE in place. Every edit = new version row.
CREATE TABLE characters (
    id          uuid NOT NULL DEFAULT gen_random_uuid(),
    version     int NOT NULL DEFAULT 1,
    name        text NOT NULL,
    traits      jsonb NOT NULL DEFAULT '[]',
    voice_samples text[] NOT NULL DEFAULT '{}',
    relationships jsonb NOT NULL DEFAULT '{}',
    created_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (id, version)
);

CREATE TABLE locations (
    id          uuid NOT NULL DEFAULT gen_random_uuid(),
    version     int NOT NULL DEFAULT 1,
    name        text NOT NULL,
    description text NOT NULL DEFAULT '',
    props       jsonb NOT NULL DEFAULT '[]',
    created_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (id, version)
);

CREATE TABLE lore (
    id          uuid NOT NULL DEFAULT gen_random_uuid(),
    tags        text[] NOT NULL DEFAULT '{}',
    content     text NOT NULL,
    embedding   vector(768),
    created_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (id)
);

CREATE INDEX idx_lore_tags ON lore USING GIN (tags);
CREATE INDEX idx_lore_embedding ON lore USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- STORY: pins which canon versions it uses
CREATE TABLE stories (
    id          uuid NOT NULL DEFAULT gen_random_uuid(),
    title       text NOT NULL,
    canon_pins  jsonb NOT NULL DEFAULT '{}',
    created_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (id)
);

-- FLOW: DAG nodes
CREATE TABLE nodes (
    id              uuid NOT NULL DEFAULT gen_random_uuid(),
    story_id        uuid NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
    beat_intent     text NOT NULL DEFAULT '',
    character_refs  uuid[] NOT NULL DEFAULT '{}',
    location_ref    uuid,
    pov             text NOT NULL DEFAULT '',
    tone            text NOT NULL DEFAULT '',
    target_words    int NOT NULL DEFAULT 300,
    status          text NOT NULL DEFAULT 'draft'
                    CHECK (status IN ('draft', 'generated', 'accepted', 'stale')),
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (id)
);

CREATE INDEX idx_nodes_story ON nodes(story_id);

-- FLOW: DAG edges
CREATE TABLE edges (
    story_id    uuid NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
    from_node   uuid NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    to_node     uuid NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    edge_type   text NOT NULL DEFAULT 'seq'
                CHECK (edge_type IN ('seq', 'fork', 'join', 'choice')),
    PRIMARY KEY (story_id, from_node, to_node)
);

-- OUTPUT: append-only, regens are siblings
CREATE TABLE generations (
    id              uuid NOT NULL DEFAULT gen_random_uuid(),
    node_id         uuid NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    context_hash    text NOT NULL DEFAULT '',
    prompt_snapshot text NOT NULL DEFAULT '',
    output          text NOT NULL DEFAULT '',
    model           text NOT NULL DEFAULT '',
    accepted        bool NOT NULL DEFAULT false,
    created_at      timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (id)
);

CREATE INDEX idx_generations_node ON generations(node_id);

-- CONTINUITY LEDGER: per-story mutable state, keyed by node
CREATE TABLE character_state (
    story_id        uuid NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
    character_id    uuid NOT NULL,
    as_of_node      uuid NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    state           jsonb NOT NULL DEFAULT '{}',
    -- state shape: {"location":"godown","knows":[],"mood":"neutral","relationships":{},"items":[]}
    updated_at      timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (story_id, character_id, as_of_node)
);

CREATE INDEX idx_char_state_story_node ON character_state(story_id, as_of_node);

-- VIEW: latest version of each character for easy querying
CREATE VIEW latest_characters AS
SELECT DISTINCT ON (id) id, version, name, traits, voice_samples, relationships, created_at
FROM characters
ORDER BY id, version DESC;

-- VIEW: latest version of each location
CREATE VIEW latest_locations AS
SELECT DISTINCT ON (id) id, version, name, description, props, created_at
FROM locations
ORDER BY id, version DESC;
