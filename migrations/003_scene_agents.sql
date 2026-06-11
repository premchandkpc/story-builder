ALTER TABLE nodes ADD COLUMN IF NOT EXISTS scene_structure jsonb NOT NULL DEFAULT '{"flow_type":"monologue","situation_flow":""}';

CREATE TABLE IF NOT EXISTS scene_turns (
    id          uuid NOT NULL DEFAULT gen_random_uuid(),
    node_id     uuid NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    turn_number int NOT NULL,
    actor_ids   uuid[] NOT NULL DEFAULT '{}',
    prompt      text NOT NULL DEFAULT '',
    output      text NOT NULL DEFAULT '',
    model       text NOT NULL DEFAULT '',
    status      text NOT NULL DEFAULT 'done',
    created_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (id)
);

CREATE INDEX IF NOT EXISTS idx_scene_turns_node ON scene_turns(node_id);
