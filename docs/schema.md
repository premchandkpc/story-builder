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
| `genre` | `text` | NOT NULL DEFAULT `''` | Added in 009 |
| `theme` | `text` | NOT NULL DEFAULT `''` | Added in 009 |
| `main_prompt` | `text` | NOT NULL DEFAULT `''` | Added in 009 |
| `general_prompt` | `text` | NOT NULL DEFAULT `''` | Added in 009 |
| `created_at` | `timestamptz` | NOT NULL DEFAULT `now()` | |

### `chapters` (added in 009)

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `id` | `uuid` | PK, `DEFAULT gen_random_uuid()` | |
| `story_id` | `uuid` | NOT NULL, FK → `stories(id)` ON DELETE CASCADE | |
| `title` | `text` | NOT NULL DEFAULT `''` | |
| `goal` | `text` | NOT NULL DEFAULT `''` | |
| `order_index` | `int` | NOT NULL DEFAULT `0` | Unique per story |
| `summary` | `text` | NOT NULL DEFAULT `''` | |
| `status` | `text` | NOT NULL DEFAULT `'draft'`, CHECK `IN ('draft','active','completed','archived')` | |
| `created_at` | `timestamptz` | NOT NULL DEFAULT `now()` | |
| `updated_at` | `timestamptz` | NOT NULL DEFAULT `now()` | |

**Indexes:** `idx_chapters_story` on `(story_id)`, unique on `(story_id, order_index)`

### `scenes` (replaces `nodes` in 009)

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `id` | `uuid` | PK, `DEFAULT gen_random_uuid()` | |
| `chapter_id` | `uuid` | NOT NULL, FK → `chapters(id)` ON DELETE CASCADE | |
| `story_id` | `uuid` | NOT NULL, FK → `stories(id)` ON DELETE CASCADE | |
| `title` | `text` | NOT NULL DEFAULT `''` | |
| `beat_intent` | `text` | NOT NULL DEFAULT `''` | What happens in this scene |
| `character_refs` | `uuid[]` | NOT NULL DEFAULT `'{}'` | References to characters |
| `location_ref` | `uuid` | nullable | FK target (no formal FK) |
| `pov` | `text` | NOT NULL DEFAULT `''` | Point-of-view character |
| `tone` | `text` | NOT NULL DEFAULT `''` | Mood/style of the scene |
| `target_words` | `int` | NOT NULL DEFAULT `300` | Target word count |
| `scene_structure` | `jsonb` | NOT NULL DEFAULT `'{}'` | Multi-agent structure |
| `parent_scene_id` | `uuid` | nullable, self-ref FK | Sub-scene reference |
| `timeline_position` | `text` | NOT NULL DEFAULT `''` | |
| `flow_type` | `text` | NOT NULL DEFAULT `'dialogue'`, CHECK `IN ('monologue','dialogue','round_robin','parallel','action','silent','custom')` | |
| `max_turns` | `int` | NOT NULL DEFAULT `5` | |
| `status` | `text` | NOT NULL DEFAULT `'draft'`, CHECK `IN ('draft','generated','accepted','stale')` | |
| `created_at` | `timestamptz` | NOT NULL DEFAULT `now()` | |
| `updated_at` | `timestamptz` | NOT NULL DEFAULT `now()` | |

**Indexes:** `idx_scenes_chapter` on `(chapter_id)`, `idx_scenes_story` on `(story_id)`, `idx_scenes_parent` on `(parent_scene_id)`

### `scene_edges` (replaces `edges` in 009)

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `story_id` | `uuid` | NOT NULL, FK → `stories(id)` ON DELETE CASCADE | |
| `from_scene` | `uuid` | NOT NULL, FK → `scenes(id)` ON DELETE CASCADE | |
| `to_scene` | `uuid` | NOT NULL, FK → `scenes(id)` ON DELETE CASCADE | |
| `edge_type` | `text` | NOT NULL DEFAULT `'seq'`, CHECK `IN ('seq','fork','join','choice','parallel')` | Added `parallel` in 009 |
| `condition` | `text` | NOT NULL DEFAULT `''` | Added in 009 |

**PK:** `(story_id, from_scene, to_scene)`
**Indexes:** `idx_scene_edges_from` on `(from_scene)`, `idx_scene_edges_to` on `(to_scene)`

### `generations`

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `id` | `uuid` | PK, `DEFAULT gen_random_uuid()` | |
| `scene_id` | `uuid` | NOT NULL, FK → `scenes(id)` ON DELETE CASCADE | Renamed from `node_id` in 009 |
| `context_hash` | `text` | NOT NULL DEFAULT `''` | SHA256 of CompiledContext |
| `prompt_snapshot` | `text` | NOT NULL DEFAULT `''` | Brief prompt summary |
| `output` | `text` | NOT NULL DEFAULT `''` | LLM-generated prose |
| `model` | `text` | NOT NULL DEFAULT `''` | Model tier string |
| `accepted` | `bool` | NOT NULL DEFAULT `false` | User-accepted flag |
| `validation_result` | `jsonb` | nullable | Added in 006 — ValidateScene output |
| `created_at` | `timestamptz` | NOT NULL DEFAULT `now()` | |

**Indexes:** `idx_generations_scene` on `(scene_id)`

### `characters`

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `id` | `uuid` | NOT NULL, `DEFAULT gen_random_uuid()` | |
| `version` | `int` | NOT NULL DEFAULT `1` | Auto-incrementing version |
| `name` | `text` | NOT NULL | |
| `persona` | `text` | NOT NULL DEFAULT `''` | Archetype/role |
| `backstory` | `text` | NOT NULL DEFAULT `''` | |
| `moral_alignment` | `text` | NOT NULL DEFAULT `''` | |
| `personality` | `jsonb` | NOT NULL DEFAULT `'[]'` | |
| `flaws` | `jsonb` | NOT NULL DEFAULT `'[]'` | |
| `goals` | `jsonb` | NOT NULL DEFAULT `'[]'` | |
| `traits` | `jsonb` | NOT NULL DEFAULT `'[]'` | Labels/tags like "jedi", "smuggler" |
| `voice_samples` | `text[]` | NOT NULL DEFAULT `'{}'` | Example dialogue lines |
| `relationships` | `jsonb` | NOT NULL DEFAULT `'{}'` | Map of relationship → character ID |
| `parent_id` | `uuid` | nullable | |
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
| `as_of_scene` | `uuid` | NOT NULL, FK → `scenes(id)` ON DELETE CASCADE | Renamed from `as_of_node` in 009 |
| `state` | `jsonb` | NOT NULL DEFAULT `'{}'` | Shape: `{location, knows[], mood, relationships{}, items[]}` |
| `updated_at` | `timestamptz` | NOT NULL DEFAULT `now()` | |

**PK:** `(story_id, character_id, as_of_scene)`
**Indexes:** `idx_char_state_story_scene` on `(story_id, as_of_scene)`

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

### `actor_traits` (added in 007)

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `id` | `uuid` | PK, `DEFAULT gen_random_uuid()` | |
| `actor_id` | `uuid` | NOT NULL, FK → `actors(id)` ON DELETE CASCADE | |
| `trait_key` | `text` | NOT NULL | |
| `trait_value` | `text` | NOT NULL DEFAULT `''` | |
| `created_at` | `timestamptz` | NOT NULL DEFAULT `now()` | |

**Unique:** `(actor_id, trait_key)`
**Index:** `idx_actor_traits_actor_id` on `(actor_id)`

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
| `scene_id` | `uuid` | NOT NULL, FK → `scenes(id)` ON DELETE CASCADE | Renamed from `node_id` in 009 |
| `turn_number` | `int` | NOT NULL | Sequential turn index |
| `actor_ids` | `uuid[]` | NOT NULL DEFAULT `'{}'` | Which characters act |
| `prompt` | `text` | NOT NULL DEFAULT `''` | LLM prompt for this turn |
| `output` | `text` | NOT NULL DEFAULT `''` | LLM response |
| `model` | `text` | NOT NULL DEFAULT `''` | Model used |
| `status` | `text` | NOT NULL DEFAULT `'done'` | pending/done/accepted/rejected |
| `created_at` | `timestamptz` | NOT NULL DEFAULT `now()` | |

**Index:** `idx_scene_turns_scene` on `(scene_id)`

### `story_summaries` (added in 005)

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `id` | `uuid` | PK, `DEFAULT gen_random_uuid()` | |
| `story_id` | `uuid` | NOT NULL, FK → `stories(id)` ON DELETE CASCADE | |
| `scene_id` | `uuid` | nullable, FK → `scenes(id)` ON DELETE SET NULL | Renamed from `node_id` in 009 |
| `level` | `text` | NOT NULL, CHECK `IN ('scene','act','story')` | Hierarchical level |
| `content` | `text` | NOT NULL | Summary text |
| `word_count` | `int` | NOT NULL DEFAULT `0` | |
| `created_at` | `timestamptz` | NOT NULL DEFAULT `now()` | |

**Partial Unique Indexes:**
- `idx_summaries_scene` — unique on `(story_id, scene_id)` WHERE `level='scene'`
- `idx_summaries_act` — unique on `(story_id, level)` WHERE `level='act'` AND `scene_id IS NULL`
- `idx_summaries_story_level` — unique on `(story_id, level)` WHERE `level='story'` AND `scene_id IS NULL`
- `idx_summaries_scene_id` on `(scene_id)`

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
stories 1──* chapters                   (story_id FK)
stories 1──* scenes                     (story_id FK)
stories 1──* scene_edges                (story_id FK)
stories 1──* casting                    (story_id FK)
stories 1──* character_state            (story_id FK)
stories 1──* story_summaries            (story_id FK)

chapters 1──* scenes                    (chapter_id FK)

scenes 1──* scene_edges (from_scene FK) (from_scene → scenes.id)
scenes 1──* scene_edges (to_scene FK)   (to_scene → scenes.id)
scenes 1──* generations                 (scene_id FK)
scenes 1──* scene_turns                 (scene_id FK)
scenes 1──* character_state             (as_of_scene FK)
scenes 1──* story_summaries             (scene_id FK, nullable)
scenes ?──? scenes (parent_scene_id)    (self-referencing)

characters (id, version) ──* character_state  (character_id)
characters (id, version) ──* casting          (character_id, no FK)
characters (id, version) ──* character_trait_assignments (character_id)

characters.id ──? characters.parent_id        (self-referencing, no FK)

actors 1──* casting                   (actor_id FK)
actors 1──* actor_traits              (actor_id FK)
character_traits 1──* character_trait_assignments (trait_id FK)
```
