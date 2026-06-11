-- Separate actor trait storage so flexible trait sets can vary per actor without bloating the actors table
CREATE TABLE IF NOT EXISTS actor_traits (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id     uuid NOT NULL REFERENCES actors(id) ON DELETE CASCADE,
    trait_key    text NOT NULL,
    trait_value  text NOT NULL DEFAULT '',
    created_at   timestamptz NOT NULL DEFAULT now(),
    UNIQUE (actor_id, trait_key)
);

CREATE INDEX IF NOT EXISTS idx_actor_traits_actor_id ON actor_traits(actor_id);

INSERT INTO actor_traits (actor_id, trait_key, trait_value)
SELECT id, key, value
FROM actors,
     jsonb_each_text(COALESCE(actors.traits, '{}'::jsonb)) AS traits(key, value)
ON CONFLICT (actor_id, trait_key) DO NOTHING;
