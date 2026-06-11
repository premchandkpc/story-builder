# Service Layer

## Overview

Two implementations exist for every service interface — **DB-backed** (Postgres via sqlc) and **In-memory** (for development without Docker). The server selects which to use at startup based on database connectivity (`cmd/server/main.go:31-143`).

---

## Handler → Service Pattern

Each API handler defines its required service methods as an **inline interface** on the handler struct. This enables:
1. Loose coupling — handlers depend only on what they call
2. Easy testing — mock implementations satisfy the interface
3. Dual (DB + memory) implementations

Example pattern (`handlers_stories.go:12-23`):
```go
type StoryHandler struct {
    Service interface {
        Create(title string) (*graph.Story, error)
        Get(id uuid.UUID) (*graph.Story, error)
        List() ([]graph.Story, error)
        // ...
    }
}
```

---

## Service Implementations

### Character Service

| Method | DB Service (`dbservices_characters.go`) | In-Memory (`canon/memory.go`) |
|---|---|---|
| `Create` | `INSERT INTO characters ... RETURNING *` | Appends to slice with `version=1` |
| `Get(id, version)` | `GetCharacterLatest` or `GetCharacterAtVersion` by version param | Linear scan, can select by version |
| `Update` | `INSERT ... SELECT MAX(version)+1` (append-only) | Appends new version row |
| `List` | `SELECT * FROM latest_characters` | Deduplicates by max version |

DB-to-domain conversion helpers:
- `toDomainChar(db.Character)` — unmarshals traits, personality, flaws, goals, relationships from JSONB
- `toDomainCharFromLatest(db.LatestCharacter)` — same from view

### Actor Service

| Method | DB Service (`dbservices_characters.go:174`) | In-Memory |
|---|---|---|
| `Create` | `INSERT INTO actors ... RETURNING *` | |
| `Get` | `SELECT * FROM actors WHERE id = $1` | |
| `Update` | `UPDATE actors SET ... WHERE id = $1` | |
| `List` | `SELECT * FROM actors ORDER BY name` | |

Traits are stored as JSONB and unmarshalled to `map[string]interface{}`.

### CharacterTrait Service

| Method | DB Service (`dbservices_characters.go:281`) |
|---|---|
| `Create` | `INSERT INTO character_traits` |
| `Get` | `SELECT * FROM character_traits WHERE id = $1` |
| `List` | `SELECT * FROM character_traits ORDER BY name` |
| `Assign` | `INSERT INTO character_trait_assignments ... ON CONFLICT DO UPDATE` |
| `Unassign` | `DELETE FROM character_trait_assignments` |
| `GetAssignments` | `SELECT ca.*, ct.* FROM character_trait_assignments ca JOIN character_traits ct` |

### Casting Service

| Method | DB Service (`dbservices_characters.go:372`) |
|---|---|
| `Create` | `INSERT INTO casting ... RETURNING *` |
| `GetForStory` | `SELECT c.*, a.name, ch.name FROM casting c JOIN actors a JOIN characters ch WHERE story_id = $1` |
| `GetForCharacter` | `SELECT c.*, a.name, s.title FROM casting c JOIN actors a JOIN stories s WHERE character_id = $1` |
| `GetForActor` | `SELECT c.*, ch.name, s.title FROM casting c JOIN characters ch JOIN stories s WHERE actor_id = $1` |

### Location Service

| Method | DB Service (`dbservices_lore.go:15`) | In-Memory (`canon/memory.go`) |
|---|---|---|
| `Create` | `INSERT INTO locations ... RETURNING *` | Appends with `version=1` |
| `Get(id, version)` | `GetLocationLatest` or `GetLocationAtVersion` | Linear scan |
| `Update` | `INSERT ... SELECT MAX(version)+1` (append-only) | Appends new version |
| `List` | `SELECT * FROM latest_locations ORDER BY name` | Deduplicates by max version |

### Lore Service

| Method | DB Service (`dbservices_lore.go:104`) | In-Memory (`canon/memory.go`) |
|---|---|---|
| `Create` | `INSERT INTO lore (tags, content, embedding) VALUES ($1, $2, $3)` — stores empty vector | Appends to slice |
| `List` | `SELECT * FROM lore ORDER BY created_at DESC` | Returns slice copy |
| `SearchByTags` | `SELECT * FROM lore WHERE tags && $1::text[]` — GIN overlap operator | Set intersection on tags |
| `SearchSimilar` | `SELECT * FROM lore ORDER BY embedding <=> $1::vector LIMIT $2` — cosine distance | Returns first N entries (stub) |

### Story (Graph) Service

| Method | DB Service (`dbservices_stories.go:22`) | In-Memory (`graph/memory.go`) |
|---|---|---|
| `Create` | `INSERT INTO stories ... RETURNING *` | Appends with `uuid.New()` |
| `Get` | `SELECT * FROM stories WHERE id = $1` | Linear scan |
| `List` | `SELECT * FROM stories ORDER BY created_at DESC` | Slice copy |
| `CreateEdge` | `INSERT INTO edges ... ON CONFLICT DO NOTHING` | Appends to slice |
| `ListEdges` | `SELECT * FROM edges WHERE story_id = $1` | Filter slice by story_id |
| `GetNode` | Delegates to `getNode()` helper → `GetNode(id)` + `toDomainNode()` | Linear scan |
| `ListNodes` | Delegates to `listNodes()` helper → `ListNodes(storyID)` + conversion | Filter slice by story_id |
| `TopologicalSort` | Lists nodes + edges → `graph.TopologicalSort()` | Same algorithm |

### Node Service

| Method | DB Service (`dbservices_stories.go:122`) |
|---|---|
| `Create` | `INSERT INTO nodes ... RETURNING *` → `toDomainNode()` |
| `Get` | `getNode()` helper |
| `Update` | `UPDATE nodes SET ... WHERE id = $1 RETURNING *` |
| `SetSceneStructure` | `UPDATE nodes SET scene_structure = $2 WHERE id = $1` |
| `List` | `listNodes()` helper |

`toDomainNode()` converts `db.Node` → `graph.Node`:
- Converts `pgtype.UUID` slices to `uuid.UUID` slices
- Unmarshals `SceneStructure` from JSONB
- Handles nullable `LocationRef`

### Generation Service

| Method | DB Service (`dbservices_stories.go:250`) |
|---|---|
| `Generate` | Loads node, compiles context, creates generation row, enqueues `GenerateSceneWorker` via River |
| `AcceptGeneration` | Sets `accepted=true` on the generation, rejects all others with `RejectOtherGenerations` |
| `ListGenerations` | `SELECT * FROM generations WHERE node_id = $1 ORDER BY created_at DESC` |

**`compileContext`** (`dbservices_stories.go:325`):
1. Creates `CompiledContext` with beat_intent, POV, tone, target_words
2. Loads latest character for each `character_ref` → creates `canon.Card`
3. Loads latest location if `location_ref` is set → creates location card
4. Searches lore by character name tags

**River integration:** After creating the generation row, inserts a `GenerateSceneArgs` job to the `generate` queue.

### Scene Service

| Method | DB Service (`dbservices_scene.go:16`) | In-Memory |
|---|---|---|
| `StartScene` | `fmt.Errorf("not implemented")` | Stub |
| `NextTurn` | `fmt.Errorf("not implemented")` | Stub |
| `FinishScene` | `fmt.Errorf("not implemented")` | Stub |
| `GetTurns` | `SELECT * FROM scene_turns WHERE node_id = $1 ORDER BY turn_number` | |
| `SetSceneStructure` | `UPDATE nodes SET scene_structure = $2 WHERE id = $1` | |
| `GetSceneStructure` | `SELECT * FROM nodes WHERE id = $1` → unmarshal scene_structure | |

Multi-agent scene feature is not yet wired to LLM — returns "not implemented" for Start/Next/Finish.

### Summary Service

| Method | DB Service (`dbservices_summaries.go:13`) |
|---|---|
| `UpsertSceneSummary` | `INSERT ... ON CONFLICT (story_id, node_id) WHERE level='scene' DO UPDATE` |
| `UpsertActSummary` | `INSERT ... ON CONFLICT (story_id, level) WHERE level='act' DO UPDATE` |
| `UpsertStorySummary` | `INSERT ... ON CONFLICT (story_id, level) WHERE level='story' DO UPDATE` |
| `GetSceneSummary` | `SELECT * FROM story_summaries WHERE story_id=$1 AND node_id=$2 AND level='scene'` |
| `GetSummaryByLevel` | `SELECT * FROM story_summaries WHERE story_id=$1 AND level=$2 ORDER BY created_at DESC LIMIT 1` |
| `ListSummariesByLevel` | `SELECT * FROM story_summaries WHERE story_id=$1 AND level=$2 ORDER BY created_at DESC` |
| `CountSummariesByLevel` | `SELECT COUNT(*) FROM story_summaries WHERE story_id=$1 AND level=$2` |
| `ShouldElevate` | `count >= threshold` |

### Story Generator Service

| Method | DB Service (`dbservices_stories.go:403`) |
|---|---|
| `GenerateStory` | Enqueues `GenerateStoryArgs` via River → returns `{story_id: "", status: "pending"}` |

---

## River Workers

All workers are in `internal/river/jobs.go`.

### GenerateSceneWorker (queue: `generate`)
1. `compilePromptParams` — loads characters, location, lore, state, summary
2. Calls `ProseService.GenerateScene(params)`
3. Calls `UpdateGenerationOutput` to store result
4. Creates character cards with full trait/voice/relationship detail
5. Loads character state for the as_of_node
6. Loads latest scene summary

### ExtractStateWorker (queue: `extract`)
1. Calls `ExtractionService.ExtractState(sceneText)`
2. Result is not yet stored (stub implementation)

### UpdateSummaryWorker (queue: `default`)
1. Calls `SummaryService.UpdateSummary(previousSummary, acceptedScene)`
2. Upserts scene-level summary

### MergeBranchesWorker (queue: `merge`)
1. Calls `MergeService.MergeBranches(summaryA, summaryB, timelineNote)`
2. Extracts `merged_summary` from result
3. Upserts story-level summary

### ValidateSceneWorker (queue: `validate`)
1. Calls `ValidationService.ValidateAgainstCanon(canonXML, charState, sceneText)`
2. Result is not yet stored (stub implementation)

### GenerateStoryWorker (queue: `default`)
1. Calls `OutlineService.GenerateOutline(synopsis)`
2. Creates story + characters + nodes + edges from outline
3. Maps character names to IDs, beat titles to IDs

---

## Helper Utilities

### UUID Conversion (`dbservices_helpers.go`)
- `toUUID(uuid.UUID) pgtype.UUID` — wraps UUID into pgtype for DB queries
- `fromUUID(pgtype.UUID) uuid.UUID` — extracts UUID from pgtype
- `jsonBytes(v interface{}) []byte` — JSON marshal helper

### Request Validation (`request_validation.go`)
- `parseUUID(s string) (uuid.UUID, error)` — trim+parse UUID
- `parseUUIDList(values []string) ([]uuid.UUID, error)` — batch UUID parse
- `parseOptionalUUID(value *string) (*uuid.UUID, error)` — nullable UUID parse
- `normalizeStoryTitle(title string) (string, error)` — trim+validate title
