# Database Schema

Extensions: `pgcrypto`, `vector` (pgvector).

---

## `characters`

| Column        | Type      | Notes                    |
|---------------|-----------|--------------------------|
| id            | uuid      | PK (with version), default gen_random_uuid() |
| version       | int       | PK, auto-incremented on UPDATE |
| name          | text      |                          |
| traits        | jsonb     | `[]` default             |
| voice_samples | text[]    | `{}` default             |
| relationships | jsonb     | `{}` default             |
| created_at    | timestamptz | `now()` default        |

PK: `(id, version)`. Never UPDATE in place — each edit inserts a new version row.

### View: `latest_characters`

`SELECT DISTINCT ON (id) ... FROM characters ORDER BY id, version DESC`

---

## `locations`

| Column     | Type      | Notes                    |
|------------|-----------|--------------------------|
| id         | uuid      | PK (with version)        |
| version    | int       | PK                       |
| name       | text      |                          |
| description| text      | `''` default             |
| props      | jsonb     | `[]` default             |
| created_at | timestamptz | `now()` default        |

Same versioning pattern as characters.

### View: `latest_locations`

---

## `lore`

| Column    | Type        | Notes                    |
|-----------|-------------|--------------------------|
| id        | uuid        | PK                       |
| tags      | text[]      | GIN-indexed              |
| content   | text        |                          |
| embedding | vector(768) | ivfflat index, cosine ops |
| created_at| timestamptz |                          |

Indexes:
- `idx_lore_tags` GIN on `tags`
- `idx_lore_embedding` ivfflat on `embedding` (lists=100)

---

## `stories`

| Column    | Type      | Notes                    |
|-----------|-----------|--------------------------|
| id        | uuid      | PK                       |
| title     | text      |                          |
| canon_pins| jsonb     | `{}` default, maps entity type -> version pin |
| created_at| timestamptz | `now()` default        |

---

## `nodes`

| Column        | Type      | Notes                    |
|---------------|-----------|--------------------------|
| id            | uuid      | PK                       |
| story_id      | uuid      | FK -> stories, indexed   |
| beat_intent   | text      |                          |
| character_refs| uuid[]    | References characters    |
| location_ref  | uuid      | Nullable, references location |
| pov           | text      |                          |
| tone          | text      |                          |
| target_words  | int       | Default 300              |
| status        | text      | `draft|generated|accepted|stale` |
| created_at    | timestamptz | `now()` default        |
| updated_at    | timestamptz | `now()` default        |

Index: `idx_nodes_story` on `story_id`.

---

## `edges`

| Column   | Type | Notes                              |
|----------|------|------------------------------------|
| story_id | uuid | FK -> stories, PK                  |
| from_node| uuid | FK -> nodes, PK                    |
| to_node  | uuid | FK -> nodes, PK                    |
| edge_type| text | `seq|fork|join|choice`, default `seq` |

PK: `(story_id, from_node, to_node)`.

---

## `generations`

| Column         | Type      | Notes                    |
|----------------|-----------|--------------------------|
| id             | uuid      | PK                       |
| node_id        | uuid      | FK -> nodes, indexed     |
| context_hash   | text      | SHA256 of CompiledContext |
| prompt_snapshot| text      | Full prompt sent to LLM  |
| output         | text      | LLM response text        |
| model          | text      | Model identifier         |
| accepted       | bool      | Default false            |
| created_at     | timestamptz | `now()` default        |

Index: `idx_generations_node` on `node_id`.

---

## `character_state`

| Column      | Type      | Notes                    |
|-------------|-----------|--------------------------|
| story_id    | uuid      | FK -> stories, PK        |
| character_id| uuid      | PK                       |
| as_of_node  | uuid      | FK -> nodes, PK          |
| state       | jsonb     | `{location, knows[], mood, relationships{}, items[]}` |
| updated_at  | timestamptz | `now()` default        |

PK: `(story_id, character_id, as_of_node)`.

Index: `idx_char_state_story_node` on `(story_id, as_of_node)`.

---

## Migration tracking: `_migrations`

| Column         | Type      | Notes                    |
|----------------|-----------|--------------------------|
| version        | int       | PK                       |
| filename       | text      | E.g. `001_init.sql`      |
| description    | text      | Human-readable summary   |
| migration_type | text      | Default `sql`            |
| checksum       | text      | MD5 of file content      |
| installed_by   | text      | Default `current_user`   |
| installed_on   | timestamptz | Default `now()`        |
| execution_time | int       | Seconds, default 0       |
| success        | bool      | Default true             |
