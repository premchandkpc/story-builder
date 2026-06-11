-- Actors: physical entity that plays a role (character) in a story
CREATE TABLE IF NOT EXISTS actors (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name        text NOT NULL,
    gender      text NOT NULL DEFAULT 'any',
    ethnicity   text NOT NULL DEFAULT '',
    race        text NOT NULL DEFAULT '',
    skin_tone   text NOT NULL DEFAULT '',
    eye_color   text NOT NULL DEFAULT '',
    hair_color  text NOT NULL DEFAULT '',
    hair_style  text NOT NULL DEFAULT '',
    build       text NOT NULL DEFAULT '',
    height_cm   int NOT NULL DEFAULT 0,
    weight_kg   int NOT NULL DEFAULT 0,
    age         int NOT NULL DEFAULT 0,
    nationality text NOT NULL DEFAULT '',
    traits      jsonb NOT NULL DEFAULT '{}',
    created_at  timestamptz NOT NULL DEFAULT now()
);

-- Enhance characters: add persona, hierarchy, backstory, alignment
ALTER TABLE characters ADD COLUMN IF NOT EXISTS persona         text NOT NULL DEFAULT '';
ALTER TABLE characters ADD COLUMN IF NOT EXISTS backstory       text NOT NULL DEFAULT '';
ALTER TABLE characters ADD COLUMN IF NOT EXISTS moral_alignment text NOT NULL DEFAULT '';
ALTER TABLE characters ADD COLUMN IF NOT EXISTS personality     jsonb NOT NULL DEFAULT '[]';
ALTER TABLE characters ADD COLUMN IF NOT EXISTS flaws           jsonb NOT NULL DEFAULT '[]';
ALTER TABLE characters ADD COLUMN IF NOT EXISTS goals           jsonb NOT NULL DEFAULT '[]';
ALTER TABLE characters ADD COLUMN IF NOT EXISTS parent_id       uuid;

-- Recreate latest_characters view to include new columns
DROP VIEW IF EXISTS latest_characters;
CREATE VIEW latest_characters AS
SELECT DISTINCT ON (id) id, version, name, persona, backstory, moral_alignment, personality, flaws, goals, traits, voice_samples, relationships, parent_id, created_at
FROM characters
ORDER BY id, version DESC;

-- Decoupled, reusable character traits (characteristics)
CREATE TABLE IF NOT EXISTS character_traits (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name        text NOT NULL,
    category    text NOT NULL DEFAULT '',
    description text NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_character_traits_name ON character_traits(name);

-- Which traits a character has, with intensity
CREATE TABLE IF NOT EXISTS character_trait_assignments (
    character_id uuid NOT NULL,
    trait_id     uuid NOT NULL REFERENCES character_traits(id) ON DELETE CASCADE,
    intensity    int NOT NULL DEFAULT 5 CHECK (intensity >= 1 AND intensity <= 10),
    note         text NOT NULL DEFAULT '',
    PRIMARY KEY (character_id, trait_id)
);

-- Casting: actor plays a character role in a story (character_id is versioned, so no FK)
CREATE TABLE IF NOT EXISTS casting (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    story_id     uuid NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
    actor_id     uuid NOT NULL REFERENCES actors(id) ON DELETE CASCADE,
    character_id uuid NOT NULL,
    role_type    text NOT NULL DEFAULT 'lead',
    created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_casting_unique ON casting(story_id, actor_id, character_id);
CREATE INDEX IF NOT EXISTS idx_casting_story ON casting(story_id);

-- Seed some sample inheritance-mapped characters
INSERT INTO characters (id, version, name, persona, backstory, moral_alignment, personality, flaws, goals, parent_id, traits, relationships) VALUES
  ('a0000000-0000-0000-0000-000000000001', 1, 'Darth Vader',
   'antagonist',
   'Once a Jedi Knight, Anakin Skywalker fell to the dark side and became Darth Vader, enforcer of the Galactic Empire.',
   'chaotic evil',
   '["brooding", "ruthless", "patient", "calculating"]'::jsonb,
   '["arrogance", "attachment", "rage"]'::jsonb,
   '["crush the rebellion", "find his son", "destroy the Emperor"]'::jsonb,
   NULL,
   '["dark side", "cyborg", "Sith Lord", "force-user"]'::jsonb,
   '{"son": "a0000000-0000-0000-0000-000000000002", "master": "Palpatine"}'::jsonb),

  ('a0000000-0000-0000-0000-000000000002', 1, 'Luke Skywalker',
   'hero',
   'A farm boy from Tatooine who discovers his Jedi heritage and rises against the Empire.',
   'lawful good',
   '["idealistic", "brave", "impulsive", "compassionate"]'::jsonb,
   '["naivety", "recklessness", "anger"]'::jsonb,
   '["become a Jedi", "save his friends", "redeem his father"]'::jsonb,
   'a0000000-0000-0000-0000-000000000001',
   '["jedi", "force-user", "pilot", "rebel"]'::jsonb,
   '{"father": "a0000000-0000-0000-0000-000000000001", "mentor": "Obi-Wan Kenobi"}'::jsonb),

  ('a0000000-0000-0000-0000-000000000003', 1, 'Leia Organa',
   'princess-diplomat',
   'Princess of Alderaan, leader of the Rebel Alliance, and secret twin of Luke Skywalker.',
   'lawful good',
   '["fierce", "diplomatic", "resilient", "determined"]'::jsonb,
   '["stubbornness", "distrust of imperials", "workaholic"]'::jsonb,
   '["free the galaxy", "restore Alderaan", "lead the Rebellion"]'::jsonb,
   'a0000000-0000-0000-0000-000000000001',
   '["rebel", "leader", "diplomat", "pilot"]'::jsonb,
   '{"father": "a0000000-0000-0000-0000-000000000001", "brother": "a0000000-0000-0000-0000-000000000002"}'::jsonb),

  ('a0000000-0000-0000-0000-000000000004', 1, 'Han Solo',
   'rogue',
   'A smuggler-turned-rebel captain of the Millennium Falcon.',
   'chaotic good',
   '["cocky", "charming", "opportunistic", "loyal"]'::jsonb,
   '["greed", "distrust of authority", "recklessness"]'::jsonb,
   '["pay off Jabba", "keep the Falcon", "win Leia''s heart"]'::jsonb,
   NULL,
   '["smuggler", "pilot", "scoundrel", "rebel"]'::jsonb,
   '{"love_interest": "a0000000-0000-0000-0000-000000000003", "friend": "Chewbacca"}'::jsonb),

  ('a0000000-0000-0000-0000-000000000005', 1, 'Obi-Wan Kenobi',
   'mentor',
   'A wise Jedi Master who watches over Luke from exile.',
   'lawful good',
   '["wise", "serene", "patient", "sacrificial"]'::jsonb,
   '["secrecy", "attachment to the old ways"]'::jsonb,
   '["protect Luke", "defeat the Sith", "restore the Jedi Order"]'::jsonb,
   NULL,
   '["jedi", "master", "force-user", "mentor"]'::jsonb,
   '{"padawan": "a0000000-0000-0000-0000-000000000001", "student": "a0000000-0000-0000-0000-000000000002"}'::jsonb)
ON CONFLICT (id, version) DO NOTHING;

-- Seed reusable character traits
INSERT INTO character_traits (name, category, description) VALUES
  ('brave', 'personality', 'Willing to face danger without hesitation'),
  ('cunning', 'personality', 'Uses cleverness and deceit to achieve goals'),
  ('empathetic', 'personality', 'Deeply understands and shares feelings of others'),
  ('honorable', 'personality', 'Follows a strict moral code'),
  ('impulsive', 'personality', 'Acts without thinking through consequences'),
  ('mysterious', 'personality', 'Keeps secrets and maintains an air of intrigue'),
  ('stoic', 'personality', 'Endures hardship without showing emotion'),
  ('vengeful', 'personality', 'Seeks retribution for past wrongs'),
  ('force-sensitive', 'ability', 'Can sense and manipulate the Force'),
  ('expert-pilot', 'skill', 'Masterful pilot of starships'),
  ('expert-swordsman', 'skill', 'Highly skilled in melee combat'),
  ('diplomatic', 'skill', 'Adept at negotiation and persuasion'),
  ('survivor', 'background', 'Has endured extreme hardship and adapted')
ON CONFLICT (name) DO NOTHING;

-- Seed actors (physical entities who play the character roles)
INSERT INTO actors (id, name, gender, ethnicity, race, skin_tone, eye_color, hair_color, hair_style, build, height_cm, weight_kg, age, nationality) VALUES
  ('b0000000-0000-0000-0000-000000000001', 'David Prowse', 'male', 'British', 'white', 'fair', 'blue', 'brown', 'bald', 'athletic', 201, 120, 45, 'British'),
  ('b0000000-0000-0000-0000-000000000002', 'Mark Hamill', 'male', 'American', 'white', 'fair', 'blue', 'brown', 'short', 'slim', 175, 77, 25, 'American'),
  ('b0000000-0000-0000-0000-000000000003', 'Carrie Fisher', 'female', 'American', 'white', 'fair', 'brown', 'brown', 'long-wavy', 'slim', 155, 54, 23, 'American'),
  ('b0000000-0000-0000-0000-000000000004', 'Harrison Ford', 'male', 'American', 'white', 'fair', 'brown', 'brown', 'medium', 'athletic', 185, 79, 35, 'American'),
  ('b0000000-0000-0000-0000-000000000005', 'Alec Guinness', 'male', 'British', 'white', 'fair', 'blue', 'gray', 'balding', 'slim', 183, 73, 60, 'British')
ON CONFLICT (id) DO NOTHING;

-- Cast the actors to their character roles (for the story — using a known story UUID if it exists)
-- We use a subquery to avoid failing if the test story doesn't exist
INSERT INTO casting (story_id, actor_id, character_id, role_type)
SELECT s.id, a.id, c.id, r.role
FROM (SELECT id FROM stories LIMIT 1) s
CROSS JOIN (VALUES
  ('b0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', 'antagonist'),
  ('b0000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000002', 'lead'),
  ('b0000000-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000003', 'lead'),
  ('b0000000-0000-0000-0000-000000000004', 'a0000000-0000-0000-0000-000000000004', 'supporting'),
  ('b0000000-0000-0000-0000-000000000005', 'a0000000-0000-0000-0000-000000000005', 'mentor')
) AS r(aid, cid, role)
JOIN actors a ON a.id = r.aid::uuid
JOIN characters c ON c.id = r.cid::uuid
ON CONFLICT (story_id, actor_id, character_id) DO NOTHING;
