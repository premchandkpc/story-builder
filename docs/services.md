# Service Layer

## Overview

All service interfaces live in `internal/service/` with clean package boundaries. Each service has **DB-backed** (Postgres via sqlc) and **In-memory** implementations. The server selects which to use at startup based on database connectivity (`cmd/server/main.go:72-210`).

```
internal/service/
  canon/       Character, Actor, Trait, Casting, Location, Lore
  story/       Story CRUD
  node/        Node CRUD
  edge/        Edge CRUD
  generation/  Generation + StoryGenerator
  scene/       Scene CRUD (multi-agent)
  summary/     Summary CRUD
  blueprint/   Story blueprint (memory-only)
  timeline/    Timeline events (memory-only)
  cache/       Redis cache wrapper + rate limiter
```

Blueprints and timelines are memory-only — no DB tables or sqlc queries exist yet.

---

## Handler → Service Pattern

Each API handler defines its required service methods as a struct field. This enables:
1. Loose coupling — handlers depend only on what they call
2. Easy testing — mock implementations satisfy the interface
3. Dual (DB + memory) implementations

Example pattern:
```go
type CharacterHandler struct {
    Service interface {
        Create(...) (*canon.Character, error)
        Get(id uuid.UUID, version int) (*canon.Character, error)
        Update(...) (*canon.Character, error)
        List() ([]canon.Character, error)
    }
}
```

---

## Canon Services (`internal/service/canon/`)

### Character Service

| Method | DB Service | In-Memory |
|---|---|---|
| `Create(name, persona, backstory, ...)` | `INSERT INTO characters ... RETURNING *` | Appends with `version=1` |
| `Get(id, version)` | `GetCharacterLatest` or `GetCharacterAtVersion` | Linear scan, select by version |
| `Update(id, ...)` | `INSERT ... SELECT MAX(version)+1` (append-only) | Appends new version row |
| `List()` | `SELECT * FROM latest_characters` | Deduplicates by max version |

### Actor Service

| Method | DB Service | In-Memory |
|---|---|---|
| `Create(name, gender, ethnicity, ...)` | `INSERT INTO actors` + `persistActorTraits` | Appends to slice |
| `Get(id)` | `SELECT * FROM actors WHERE id = $1` + `loadActorTraits` | Linear scan |
| `Update(id, ...)` | `UPDATE actors SET ... WHERE id = $1` | Replace by ID |
| `List()` | `SELECT * FROM actors ORDER BY name` | Slice copy |

Actor traits are stored as individual rows in `actor_traits` table (separate from `character_traits`).

### CharacterTrait Service

| Method | DB Service | In-Memory |
|---|---|---|
| `Create(name, category, description)` | `INSERT INTO character_traits` | Appends to slice |
| `Get(id)` | `SELECT * FROM character_traits WHERE id = $1` | Linear scan |
| `List()` | `SELECT * FROM character_traits ORDER BY name` | Slice copy |
| `Assign(charID, traitID, intensity, note)` | `INSERT INTO character_trait_assignments ... ON CONFLICT DO UPDATE` | Map insert |
| `Unassign(charID, traitID)` | `DELETE FROM character_trait_assignments` | Map delete |
| `GetAssignments(charID)` | `SELECT ca.*, ct.* FROM character_trait_assignments ca JOIN character_traits ct` | Filter map |

### Casting Service

| Method | DB Service | In-Memory |
|---|---|---|
| `Create(storyID, actorID, charID, roleType)` | `INSERT INTO casting ... RETURNING *` | Appends to slice |
| `GetForStory(storyID)` | `SELECT c.*, a.name, ch.name FROM casting c JOIN actors a JOIN characters ch` | Filter + join |
| `GetForCharacter(charID)` | `SELECT c.*, a.name, s.title FROM casting c JOIN actors a JOIN stories s` | Filter + join |
| `GetForActor(actorID)` | `SELECT c.*, ch.name, s.title FROM casting c JOIN characters ch JOIN stories s` | Filter + join |

### Location Service

| Method | DB Service | In-Memory |
|---|---|---|
| `Create(name, description, props)` | `INSERT INTO locations ... RETURNING *` | Appends with `version=1` |
| `Get(id, version)` | `GetLocationLatest` or `GetLocationAtVersion` | Linear scan |
| `Update(id, ...)` | `INSERT ... SELECT MAX(version)+1` (append-only) | Appends new version |
| `List()` | `SELECT * FROM latest_locations ORDER BY name` | Deduplicates by max version |

### Lore Service

| Method | DB Service | In-Memory |
|---|---|---|
| `Create(tags, content)` | `INSERT INTO lore (tags, content, embedding) VALUES ($1, $2, $3)` — stores empty vector | Appends to slice |
| `List()` | `SELECT * FROM lore ORDER BY created_at DESC` | Slice copy |
| `SearchByTags(tags)` | `SELECT * FROM lore WHERE tags && $1::text[]` — GIN overlap operator | Set intersection |
| `SearchSimilar(embedding, limit)` | `SELECT * FROM lore ORDER BY embedding <=> $1::vector LIMIT $2` — cosine distance | Returns first N (stub) |

---

## Graph Services

### Story Service (`internal/service/story/`)

| Method | DB Service | In-Memory (`graph/memory.go`) |
|---|---|---|
| `Create(title)` | `INSERT INTO stories ... RETURNING *` | `graph.NewMemoryStore()` |
| `Get(id)` | `SELECT * FROM stories WHERE id = $1` | Linear scan |
| `List()` | `SELECT * FROM stories ORDER BY created_at DESC` | Slice copy |

### Node Service (`internal/service/node/`)

| Method | DB Service | In-Memory |
|---|---|---|
| `Create(storyID, beatIntent, charRefs, ...)` | `INSERT INTO nodes ... RETURNING *` | Appends to store |
| `Get(id)` | `SELECT * FROM nodes WHERE id = $1` | Linear scan |
| `Update(id, ...)` | `UPDATE nodes SET ... WHERE id = $1 RETURNING *` | Replace |
| `List(storyID)` | `SELECT * FROM nodes WHERE story_id = $1` | Filter by storyID |
| `SetSceneStructure(id, structure)` | `UPDATE nodes SET scene_structure = $2 WHERE id = $1` | Inline update |

Domain conversion in `toDomainNode()`:
- Converts `pgtype.UUID` slices to `uuid.UUID` slices
- Unmarshals `SceneStructure` from JSONB
- Handles nullable `LocationRef`

### Edge Service (`internal/service/edge/`)

| Method | DB Service | In-Memory |
|---|---|---|
| `Create(storyID, fromNode, toNode, edgeType)` | `INSERT INTO edges ... ON CONFLICT DO NOTHING` | Appends to slice |
| `List(storyID)` | `SELECT * FROM edges WHERE story_id = $1` | Filter by storyID |

---

## Generation Services (`internal/service/generation/`)

### Generation Service

| Method | DB Service | In-Memory |
|---|---|---|
| `Generate(nodeID)` | Loads node, compiles context, creates generation row, checks ContextCache, enqueues `GenerateSceneWorker` via River | Returns error (stub) |
| `AcceptGeneration(nodeID, genID)` | Sets `accepted=true`, rejects others, enqueues ExtractState → UpdateSummary → MergeBranches → ValidateScene | Stub |
| `ListGenerations(nodeID)` | `SELECT * FROM generations WHERE node_id = $1 ORDER BY created_at DESC` | Returns empty |

**`compileContext`** loads:
1. Latest character for each `character_ref` → creates `canon.Card`
2. Latest location if `location_ref` is set → location card
3. Lore searched by character name tags

**Redis ContextCache:** When available, checks for cached compilation results before enqueuing River jobs.

### Story Generator Service

| Method | DB Service | In-Memory |
|---|---|---|
| `GenerateStory(synopsis)` | Enqueues `GenerateStoryArgs` via River → returns `{story_id: "", status: "pending"}` | Returns error (stub) |

---

## Scene Service (`internal/service/scene/`)

| Method | DB Service | In-Memory |
|---|---|---|
| `StartScene(nodeID)` | `fmt.Errorf("not implemented")` | Stub |
| `NextTurn(nodeID)` | `fmt.Errorf("not implemented")` | Stub |
| `FinishScene(nodeID)` | `fmt.Errorf("not implemented")` | Stub |
| `GetTurns(nodeID)` | `SELECT * FROM scene_turns WHERE node_id = $1 ORDER BY turn_number` | Returns empty |
| `SetSceneStructure(nodeID, structure)` | `UPDATE nodes SET scene_structure = $2 WHERE id = $1` | Inline update |
| `GetSceneStructure(nodeID)` | `SELECT scene_structure FROM nodes WHERE id = $1` | Return from store |

Multi-agent scene feature is not yet wired to LLM.

---

## Summary Service (`internal/service/summary/`)

| Method | DB Service | In-Memory |
|---|---|---|
| `UpsertSceneSummary(storyID, nodeID, content, wordCount)` | `INSERT ... ON CONFLICT (story_id, node_id) WHERE level='scene' DO UPDATE` | Map upsert |
| `UpsertActSummary(storyID, content, wordCount)` | `INSERT ... ON CONFLICT (story_id, level) WHERE level='act' DO UPDATE` | Map upsert |
| `UpsertStorySummary(storyID, content, wordCount)` | `INSERT ... ON CONFLICT (story_id, level) WHERE level='story' DO UPDATE` | Map upsert |
| `GetSceneSummary(storyID, nodeID)` | `SELECT * FROM story_summaries WHERE level='scene' AND story_id=$1 AND node_id=$2` | Map lookup |
| `GetSummaryByLevel(storyID, level)` | `SELECT * FROM story_summaries WHERE story_id=$1 AND level=$2 ORDER BY created_at DESC LIMIT 1` | Filter by level |
| `ListSummariesByLevel(storyID, level)` | `SELECT * FROM story_summaries WHERE story_id=$1 AND level=$2 ORDER BY created_at DESC` | Filter + sort |
| `CountSummariesByLevel(storyID, level)` | `SELECT COUNT(*) FROM story_summaries WHERE story_id=$1 AND level=$2` | Count in map |
| `ShouldElevate(storyID, level, threshold)` | `count >= threshold` | Same logic |

---

## Blueprint Service (`internal/service/blueprint/`)

Memory-only (no DB backing). Wraps `narrative.MemoryStore`.

| Method | Implementation |
|---|---|

|
| `Save(storyID, blueprint)` | Validates + stores in memory map |
| `Get(storyID)` | Returns clone from memory map |

Blueprint model includes: Premise, Theme, Conflict, Stakes, EndState, Acts, PlotThreads, CharacterArcs.

---

## Timeline Service (`internal/service/timeline/`)

Memory-only (no DB backing). Wraps `timeline.MemoryStore`.

| Method | Implementation |
|---|---|

|
| `Save(storyID, event)` | Validates + appends to memory slice |
| `List(storyID)` | Returns sorted by Order → CreatedAt |

Timeline Event model: Title, Description, Order, Timestamp, Location.

---

## Cache Service (`internal/service/cache/`)

Wraps Redis primitives (`internal/cache/`). Optional — degrades gracefully.

| Component | Redis Key Prefix | Purpose |
|---|---|---|
| `ContextCache` | `story:{id}:context*` | Caches CompiledContext hash for staleness |
| `PromptCache` | `prompt:{hash}` | Caches LLM responses for identical prompts |
| `SlidingWindowRateLimiter` | `ratelimit:{prefix}` | Rate limits LLM API calls |
| `DistLock` | `lock:{name}` | Distributed lock for coordination |

`WrapLLMClient()` decorates the LLM client with:
1. `CachedLLMClient` — checks cache before LLM call
2. `RateLimitedLLMClient` — enforces rate limits per provider

---

## River Workers

All workers in `internal/river/jobs.go`.

### GenerateSceneWorker (queue: `generate`)
1. `compilePromptParams` — loads characters, location, lore, state, summary
2. Calls `ProseService.GenerateScene(params)` → Router dispatches to Anthropic/Ollama
3. Calls `UpdateGenerationOutput` to store result

### ExtractStateWorker (queue: `extract`)
1. Calls `ExtractionService.ExtractState(sceneText, roster)`
2. Result is not yet stored (stub — no DB upsert)

### UpdateSummaryWorker (queue: `default`)
1. Calls `SummaryService.UpdateSummary(previousSummary, acceptedScene)`
2. Upserts scene-level summary

### MergeBranchesWorker (queue: `merge`)
1. Calls `MergeService.MergeBranches(summaryA, summaryB, timelineNote)`
2. Extracts `merged_summary` from result
3. Upserts story-level summary

### ValidateSceneWorker (queue: `validate`)
1. Calls `ValidationService.ValidateAgainstCanon(canonXML, charState, sceneText)`
2. Result is not yet stored (stub)

### GenerateStoryWorker (queue: `default`)
1. Calls `OutlineService.GenerateOutline(synopsis)`
2. Creates story + characters + nodes + edges from outline
3. Maps character names to IDs, beat titles to IDs

---

## Helper Utilities

### UUID Conversion (`internal/db/helpers.go`)
- `toUUID(uuid.UUID) pgtype.UUID` — wraps UUID into pgtype for DB queries
- `fromUUID(pgtype.UUID) uuid.UUID` — extracts UUID from pgtype
- `uuidFromBytes([16]byte) uuid.UUID` — converts byte array to UUID

### Request Validation (`internal/api/request_validation.go`)
- `parseUUID(s string) (uuid.UUID, error)` — trim+parse UUID
- `parseUUIDList(values []string) ([]uuid.UUID, error)` — batch UUID parse
- `parseOptionalUUID(value *string) (*uuid.UUID, error)` — nullable UUID parse
- `normalizeStoryTitle(title string) (string, error)` — trim+validate title
