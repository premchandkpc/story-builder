# Database Schema

## Extensions

```sql
CREATE EXTENSION IF NOT EXISTS "pgcrypto";  -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS "vector";    -- pgvector
```

---

## Tables

### `stories`

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `id` | `uuid` | PK, `DEFAULT gen_random_uuid()` | |
| `title` | `text` | NOT NULL | Story title |
| `canon_pins` | `jsonb` | NOT NULL DEFAULT `'{}'` | Maps entity_type → `{id, version}` |
| `created_at` | `timestamptz` | NOT NULL DEFAULT `now()` | |

### `nodes`

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `id` | `uuid` | PK, `DEFAULT gen_random_uuid()` | |
| `story_id` | `uuid` | NOT NULL, FK → `stories(id)` ON DELETE CASCADE | |
| `beat_intent` | `text` | NOT NULL DEFAULT `''` | What happens in this scene |
| `character_refs` | `uuid[]` | NOT NULL DEFAULT `'{}'` | References to characters |
| `location_ref` | `uuid` | nullable | FK target (no formal FK) |
| `pov` | `text` | NOT NULL DEFAULT `''` | Point-of-view character |
| `tone` | `text` | NOT NULL DEFAULT `''` | Mood/style of the scene |
| `target_words` | `int` | NOT NULL DEFAULT `300` | Target word count |
| `status` | `text` | NOT NULL DEFAULT `'draft'`, CHECK `IN ('draft','generated','accepted','stale')` | |
| `scene_structure` | `jsonb` | NOT NULL DEFAULT `'{"flow_type":"monologue","situation_flow":""}'` | Multi-agent structure |
| `created_at` | `timestamptz` | NOT NULL DEFAULT `now()` | |
| `updated_at` | `timestamptz` | NOT NULL DEFAULT `now()` | |

**Indexes:** `idx_nodes_story` on `(story_id)`

### `edges`

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `story_id` | `uuid` | NOT NULL, FK → `stories(id)` ON DELETE CASCADE | |
| `from_node` | `uuid` | NOT NULL, FK → `nodes(id)` ON DELETE CASCADE | |
| `to_node` | `uuid` | NOT NULL, FK → `nodes(id)` ON DELETE CASCADE | |
| `edge_type` | `text` | NOT NULL DEFAULT `'seq'`, CHECK `IN ('seq','fork','join','choice')` | |

**PK:** `(story_id, from_node, to_node)`

### `generations`

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `id` | `uuid` | PK, `DEFAULT gen_random_uuid()` | |
| `node_id` | `uuid` | NOT NULL, FK → `nodes(id)` ON DELETE CASCADE | |
| `context_hash` | `text` | NOT NULL DEFAULT `''` | SHA256 of CompiledContext |
| `prompt_snapshot` | `text` | NOT NULL DEFAULT `''` | Brief prompt summary |
| `output` | `text` | NOT NULL DEFAULT `''` | LLM-generated prose |
| `model` | `text` | NOT NULL DEFAULT `''` | Model tier string |
| `accepted` | `bool` | NOT NULL DEFAULT `false` | User-accepted flag |
| `created_at` | `timestamptz` | NOT NULL DEFAULT `now()` | |

**Indexes:** `idx_generations_node` on `(node_id)`

### `characters`

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `id` | `uuid` | NOT NULL, `DEFAULT gen_random_uuid()` | |
| `version` | `int` | NOT NULL DEFAULT `1` | Auto-incrementing version |
| `name` | `text` | NOT NULL | |
| `persona` | `text` | NOT NULL DEFAULT `''` | Archetype/role (added in 004) |
| `backstory` | `text` | NOT NULL DEFAULT `''` | (added in 004) |
| `moral_alignment` | `text` | NOT NULL DEFAULT `''` | (added in 004) |
| `personality` | `jsonb` | NOT NULL DEFAULT `'[]'` | (added in 004) |
| `flaws` | `jsonb` | NOT NULL DEFAULT `'[]'` | (added in 004) |
| `goals` | `jsonb` | NOT NULL DEFAULT `'[]'` | (added in 004) |
| `traits` | `jsonb` | NOT NULL DEFAULT `'[]'` | Labels/tags like "jedi", "smuggler" |
| `voice_samples` | `text[]` | NOT NULL DEFAULT `'{}'` | Example dialogue lines |
| `relationships` | `jsonb` | NOT NULL DEFAULT `'{}'` | Map of relationship → character ID |
| `parent_id` | `uuid` | nullable | (added in 004) |
| `created_at` | `timestamptz` | NOT NULL DEFAULT `now()` | |

**PK:** `(id, version)` — append-only, each update creates new version row

### `locations`

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `id` | `uuid` | NOT NULL, `DEFAULT gen_random_uuid()` | |
| `version` | `int` | NOT NULL DEFAULT `1` | Auto-incrementing version |
| `name` | `text` | NOT NULL | |
| `description` | `text` | NOT NULL DEFAULT `''` | |
| `props` | `jsonb` | NOT NULL DEFAULT `'[]'` | Available props/features |
| `created_at` | `timestamptz` | NOT NULL DEFAULT `now()` | |

**PK:** `(id, version)` — append-only

### `lore`

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `id` | `uuid` | PK, `DEFAULT gen_random_uuid()` | |
| `tags` | `text[]` | NOT NULL DEFAULT `'{}'` | GIN-indexed |
| `content` | `text` | NOT NULL | World-building text |
| `embedding` | `vector(768)` | nullable | pgvector embedding for similarity search |
| `created_at` | `timestamptz` | NOT NULL DEFAULT `now()` | |

**Indexes:**
- `idx_lore_tags` — GIN on `(tags)`
- `idx_lore_embedding` — IVFFLAT on `(embedding vector_cosine_ops)` with `lists=100`

### `character_state`

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `story_id` | `uuid` | NOT NULL, FK → `stories(id)` ON DELETE CASCADE | |
| `character_id` | `uuid` | NOT NULL | |
| `as_of_node` | `uuid` | NOT NULL, FK → `nodes(id)` ON DELETE CASCADE | Snapshot point in DAG |
| `state` | `jsonb` | NOT NULL DEFAULT `'{}'` | Shape: `{location, knows[], mood, relationships{}, items[]}` |
| `updated_at` | `timestamptz` | NOT NULL DEFAULT `now()` | |

**PK:** `(story_id, character_id, as_of_node)`
**Indexes:** `idx_char_state_story_node` on `(story_id, as_of_node)`

### `actors` (added in 004)

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `id` | `uuid` | PK, `DEFAULT gen_random_uuid()` | |
| `name` | `text` | NOT NULL | Real name |
| `gender` | `text` | NOT NULL DEFAULT `'any'` | |
| `ethnicity` | `text` | NOT NULL DEFAULT `''` | |
| `race` | `text` | NOT NULL DEFAULT `''` | |
| `skin_tone` | `text` | NOT NULL DEFAULT `''` | |
| `eye_color` | `text` | NOT NULL DEFAULT `''` | |
| `hair_color` | `text` | NOT NULL DEFAULT `''` | |
| `hair_style` | `text` | NOT NULL DEFAULT `''` | |
| `build` | `text` | NOT NULL DEFAULT `''` | |
| `height_cm` | `int` | NOT NULL DEFAULT `0` | |
| `weight_kg` | `int` | NOT NULL DEFAULT `0` | |
| `age` | `int` | NOT NULL DEFAULT `0` | |
| `nationality` | `text` | NOT NULL DEFAULT `''` | |
| `traits` | `jsonb` | NOT NULL DEFAULT `'{}'` | Flexible actor attributes |
| `created_at` | `timestamptz` | NOT NULL DEFAULT `now()` | |

### `character_traits` (added in 004)

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `id` | `uuid` | PK, `DEFAULT gen_random_uuid()` | |
| `name` | `text` | NOT NULL | Unique trait name |
| `category` | `text` | NOT NULL DEFAULT `''` | e.g., personality, ability, skill |
| `description` | `text` | NOT NULL DEFAULT `''` | |
| `created_at` | `timestamptz` | NOT NULL DEFAULT `now()` | |

**Index:** `idx_character_traits_name` — unique on `(name)`

### `character_trait_assignments` (added in 004)

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `character_id` | `uuid` | NOT NULL | |
| `trait_id` | `uuid` | NOT NULL, FK → `character_traits(id)` ON DELETE CASCADE | |
| `intensity` | `int` | NOT NULL DEFAULT `5`, CHECK `1..10` | How strongly the trait manifests |
| `note` | `text` | NOT NULL DEFAULT `''` | Contextual note |

**PK:** `(character_id, trait_id)`

### `casting` (added in 004)

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `id` | `uuid` | PK, `DEFAULT gen_random_uuid()` | |
| `story_id` | `uuid` | NOT NULL, FK → `stories(id)` ON DELETE CASCADE | |
| `actor_id` | `uuid` | NOT NULL, FK → `actors(id)` ON DELETE CASCADE | |
| `character_id` | `uuid` | NOT NULL | Reference to character (no FK — versioned) |
| `role_type` | `text` | NOT NULL DEFAULT `'lead'` | lead, supporting, antagonist, mentor |
| `created_at` | `timestamptz` | NOT NULL DEFAULT `now()` | |

**Indexes:**
- `idx_casting_unique` — unique on `(story_id, actor_id, character_id)`
- `idx_casting_story` on `(story_id)`

### `scene_turns` (added in 003)

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `id` | `uuid` | PK, `DEFAULT gen_random_uuid()` | |
| `node_id` | `uuid` | NOT NULL, FK → `nodes(id)` ON DELETE CASCADE | |
| `turn_number` | `int` | NOT NULL | Sequential turn index |
| `actor_ids` | `uuid[]` | NOT NULL DEFAULT `'{}'` | Which characters act |
| `prompt` | `text` | NOT NULL DEFAULT `''` | LLM prompt for this turn |
| `output` | `text` | NOT NULL DEFAULT `''` | LLM response |
| `model` | `text` | NOT NULL DEFAULT `''` | Model used |
| `status` | `text` | NOT NULL DEFAULT `'done'` | pending/done/accepted/rejected |
| `created_at` | `timestamptz` | NOT NULL DEFAULT `now()` | |

**Index:** `idx_scene_turns_node` on `(node_id)`

### `story_summaries` (added in 005)

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `id` | `uuid` | PK, `DEFAULT gen_random_uuid()` | |
| `story_id` | `uuid` | NOT NULL, FK → `stories(id)` ON DELETE CASCADE | |
| `node_id` | `uuid` | nullable, FK → `nodes(id)` ON DELETE SET NULL | Null for act/story level |
| `level` | `text` | NOT NULL, CHECK `IN ('scene','act','story')` | Hierarchical level |
| `content` | `text` | NOT NULL | Summary text |
| `word_count` | `int` | NOT NULL DEFAULT `0` | |
| `created_at` | `timestamptz` | NOT NULL DEFAULT `now()` | |

**Partial Unique Indexes:**
- `idx_summaries_scene` — unique on `(story_id, node_id)` WHERE `level='scene'`
- `idx_summaries_act` — unique on `(story_id, level)` WHERE `level='act'` AND `node_id IS NULL`
- `idx_summaries_story_level` — unique on `(story_id, level)` WHERE `level='story'` AND `node_id IS NULL`
- `idx_summaries_node` on `(node_id)`

---

## Migration Tracking

### `_migrations`

| Column | Type | Notes |
|---|---|---|
| `version` | `int` | PK |
| `filename` | `text` | NOT NULL |
| `description` | `text` | NOT NULL DEFAULT `''` |
| `migration_type` | `text` | NOT NULL DEFAULT `'sql'` |
| `checksum` | `text` | NOT NULL DEFAULT `''` — MD5 of SQL content |
| `installed_by` | `text` | NOT NULL DEFAULT `current_user` |
| `installed_on` | `timestamptz` | NOT NULL DEFAULT `now()` |
| `execution_time` | `int` | NOT NULL DEFAULT `0` |
| `success` | `bool` | NOT NULL DEFAULT `true` |

---

## Views

### `latest_characters`

```sql
CREATE VIEW latest_characters AS
SELECT DISTINCT ON (id) id, version, name, persona, backstory,
    moral_alignment, personality, flaws, goals, traits,
    voice_samples, relationships, parent_id, created_at
FROM characters
ORDER BY id, version DESC;
```

### `latest_locations`

```sql
CREATE VIEW latest_locations AS
SELECT DISTINCT ON (id) id, version, name, description, props, created_at
FROM locations
ORDER BY id, version DESC;
```

---

## Entity Relationships

```
stories 1──* nodes                    (story_id FK)
stories 1──* edges                    (story_id FK)
stories 1──* casting                  (story_id FK)
stories 1──* character_state          (story_id FK)
stories 1──* story_summaries          (story_id FK)

nodes 1──* edges (from_node FK)       (from_node → nodes.id)
nodes 1──* edges (to_node FK)         (to_node → nodes.id)
nodes 1──* generations                (node_id FK)
nodes 1──* scene_turns                (node_id FK)
nodes 1──* character_state            (as_of_node FK)
nodes 1──* story_summaries            (node_id FK, nullable)

characters (id, version) ──* character_state  (character_id)
characters (id, version) ──* casting          (character_id, no FK)
characters (id, version) ──* character_trait_assignments (character_id)

characters.id ──? characters.parent_id        (self-referencing, no FK)

actors 1──* casting                   (actor_id FK)
character_traits 1──* character_trait_assignments (trait_id FK)
```
