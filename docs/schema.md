# MongoDB Collections

## Design Principles

- MongoDB is the single source of truth.
- Documents are evolvable — no schema migrations.
- Embed related data when accessed together; reference when independent.
- Graph structure (scene_edges) is a separate collection — DAGs don't embed well.
- Character state is append-only (event-sourcing lite).
- Memories have embeddings for vector search (MongoDB Atlas Search).
- Indexes are defined in application startup code, not migrations.

---

## Collections

### `stories`

```json
{
  "_id": "story_1",
  "title": "Empire of Ash",
  "genre": "fantasy",
  "theme": "redemption",
  "mainPrompt": "",
  "generalPrompt": "",
  "canonPins": {},
  "rootSceneId": "scene_1",
  "status": "draft",
  "createdAt": "2026-06-16T00:00:00Z",
  "updatedAt": "2026-06-16T00:00:00Z"
}
```

**Status transitions:** `draft → active`, `active → completed`, `completed → archived`. Re-opening a draft from active also allowed.

**Indexes:**
- `{ title: 1 }`
- `{ status: 1 }`

---

### `scenes`

```json
{
  "_id": "scene_100",
  "storyId": "story_1",
  "chapterId": "chapter_1",
  "title": "Arrival",
  "beatIntent": "Hero arrives at the castle",
  "summary": "",
  "generatedContent": "",
  "participants": ["char_1", "char_2"],
  "locationRef": "loc_1",
  "pov": "hero",
  "tone": "mysterious",
  "targetWords": 500,
  "flowType": "dialogue",
  "maxTurns": 5,
  "timelinePosition": 12,
  "status": "draft",
  "sceneStructure": {},
  "metadata": {},
  "createdAt": "",
  "updatedAt": ""
}
```

**Status transitions:** `draft → generated → accepted → stale`. Stale scenes can be regenerated (`stale → generated`).

**Indexes:**
- `{ storyId: 1 }`
- `{ storyId: 1, timelinePosition: 1 }`
- `{ status: 1 }`

---

### `scene_edges`

Separate collection for DAG structure. Never embed children in scenes.

```json
{
  "_id": "edge_1",
  "storyId": "story_1",
  "fromSceneId": "scene_a",
  "toSceneId": "scene_b",
  "type": "seq",
  "condition": "",
  "createdAt": ""
}
```

**Edge types:** `seq`, `fork`, `join`, `choice`, `parallel`

**Indexes:**
- `{ storyId: 1 }`
- `{ fromSceneId: 1 }`
- `{ toSceneId: 1 }`
- `{ storyId: 1, fromSceneId: 1, toSceneId: 1 }` (unique)

---

### `characters`

Immutable versioned log. Each update inserts a new document with an incremented version; old versions are preserved.

```json
{
  "_id": "char_1_v1",
  "charId": "char_1",
  "version": 1,
  "storyId": "story_1",
  "name": "Arya",
  "persona": "rogue",
  "backstory": "Orphaned young, raised by the guild",
  "personality": {
    "courage": 8,
    "kindness": 4,
    "deception": 7
  },
  "moralAlignment": "chaotic neutral",
  "goals": ["find family", "avenge father"],
  "flaws": ["reckless", "vengeful"],
  "traits": ["sneaky", "fast"],
  "voiceSamples": ["I don't need your help.", "I've seen worse."],
  "relationships": {
    "char_2": "trusts",
    "char_3": "hunts"
  },
  "relData": [
    {"targetName": "Villain", "trust": 20, "respect": 0, "fear": 70, "affection": 0}
  ],
  "want": "Find family",
  "need": "Learn to trust again",
  "falseBelief": "Revenge brings peace",
  "fear": "Being alone",
  "arcType": "redemption",
  "createdAt": ""
}
```

On update, a new document is inserted with the same `charId`, incremented `version`, and a new `_id`. This creates an immutable version log.

**Indexes:**
- `{ storyId: 1 }`
- `{ storyId: 1, name: 1 }`
- `{ charId: 1, version: -1 }` — efficient latest-version lookup

---

### `character_state`

Event-sourced state history. Always append, never overwrite. Richer fields for narrative simulation.

```json
{
  "_id": "state_1",
  "storyId": "story_1",
  "characterId": "char_1",
  "sceneId": "scene_100",
  "health": 80,
  "mood": "fearful",
  "location": "dungeon",
  "inventory": ["dagger"],
  "knowledge": ["The king is dead", "Castle has secret passages"],
  "doesNotKnow": ["The queen is the real conspirator"],
  "activeGoal": "Escape the dungeon",
  "emotionalState": "anxious",
  "physicalState": "injured",
  "relationships": {
    "char_2": "distrusts"
  },
  "relationshipData": [
    {"targetName": "char_2", "trustDelta": -15, "note": "Betrayed me"}
  ],
  "changes": {
    "learned": ["The king is dead"],
    "mood": "fearful"
  },
  "createdAt": ""
}
```

**Indexes:**
- `{ storyId: 1, characterId: 1, sceneId: 1, createdAt: -1 }`
- `{ storyId: 1, characterId: 1, createdAt: -1 }`
- `{ sceneId: 1 }`

---

### `character_memories`

The AI heart. Each memory is a document with an embedding for vector search.

```json
{
  "_id": "mem_1",
  "storyId": "story_1",
  "characterId": "char_1",
  "sceneId": "scene_100",
  "content": "John betrayed me at the castle gates",
  "type": "event",
  "importance": 0.91,
  "embedding": [0.012, -0.034, ...],
  "createdAt": ""
}
```

**Memory types:** `event`, `dialogue`, `observation`, `injury`, `relationship_change`

**Indexes:**
- `{ storyId: 1, characterId: 1, createdAt: -1 }`
- `{ storyId: 1, characterId: 1, importance: -1 }`
- Vector index on `embedding` (Atlas Search)

---

### `generations`

```json
{
  "_id": "gen_1",
  "storyId": "story_1",
  "sceneId": "scene_100",
  "contextHash": "sha256hex...",
  "promptSnapshot": "POV: Arya | Tone: mysterious",
  "output": "The castle gates groaned open...",
  "model": "claude-sonnet-4-20250514",
  "status": "success",
  "accepted": false,
  "stepStatus": {
    "generate": "done",
    "extract": "done",
    "memory": "done",
    "timeline": "done",
    "summary": "done",
    "validate": "done"
  },
  "validationResult": null,
  "error": "",
  "promptTokens": 4500,
  "completionTokens": 820,
  "totalTokens": 5320,
  "durationMs": 18400,
  "createdAt": "2026-06-16T00:00:00Z",
  "updatedAt": "2026-06-16T00:00:05Z"
}
```

**Status values:** `pending`, `running`, `partial_success`, `success`, `failed`

**Indexes:**
- `{ sceneId: 1, createdAt: -1 }`
- `{ storyId: 1, createdAt: -1 }`

---

### `summaries`

```json
{
  "_id": "sum_1",
  "storyId": "story_1",
  "sceneId": "scene_100",
  "level": "scene",
  "content": "Arya arrives at the castle seeking answers...",
  "wordCount": 120,
  "createdAt": ""
}
```

**Levels:** `scene`, `act`, `story`

**Indexes:**
- `{ storyId: 1, level: 1, createdAt: -1 }`
- `{ storyId: 1, sceneId: 1 }` (partial: level='scene')

---

### `locations`

Hierarchical location graph. Parent chain enables drill-up navigation (room→building→city→country→planet→dimension).

```json
{
  "_id": "loc_1",
  "storyId": "story_1",
  "name": "Castle Gates",
  "description": "The imposing main entrance to the castle",
  "props": ["portcullis", "moat"],
  "locType": "building",
  "parentId": "loc_0",
  "features": ["fortified", "guarded"],
  "atmosphere": "foreboding",
  "children": ["loc_2"],
  "createdAt": "2026-06-16T00:00:00Z",
  "updatedAt": "2026-06-16T00:00:00Z"
}
```

**LocationType enum:** `dimension`, `planet`, `country`, `city`, `district`, `building`, `room`

**Indexes:**
- `{ storyId: 1 }`
- `{ storyId: 1, name: 1 }`
- `{ parentId: 1 }`

---

### `bibles`

Generated once per story via LLM (claude-sonnet, temp 0.3, 8192 max tokens). Never regenerated. Structured domain data (not free-text blob).

```json
{
  "_id": "bible_1",
  "storyId": "story_1",
  "world": "A sprawling fantasy realm of floating islands and ancient ruins...",
  "dimensions": [
    {
      "name": "Material Plane",
      "description": "The physical world where most of the story takes place",
      "rules": ["Magic requires a catalyst object", "The old gods sleep beneath the mountains"]
    }
  ],
  "worldRules": [
    {"category": "physics", "description": "Magic requires a catalyst object", "strictness": "firm"},
    {"category": "magic", "description": "The old gods cannot directly intervene", "strictness": "absolute"}
  ],
  "magicSystems": [
    {
      "name": "Catalysis",
      "source": "Ambient mana through focus objects (catalysts)",
      "cost": "Stamina and concentration",
      "limitations": ["Requires years of training", "Limited by user's stamina"],
      "users": ["Mages", "Artificers"]
    }
  ],
  "factions": [
    {
      "name": "The Iron Guild",
      "goal": "Control all mana sources",
      "resources": "Wealthy, well-organized",
      "members": ["Grandmaster", "Three Wardens"],
      "relations": "Hostile to unlicensed mages"
    }
  ],
  "cultures": [
    {
      "name": "Valdori",
      "values": ["Honor", "Family", "Tradition"],
      "customs": ["Ancestor worship", "Iron-fasting rites of passage"],
      "technology": "Medieval with steam innovations",
      "government": "Feudal monarchy"
    }
  ],
  "tone": "dark fantasy with moments of hope and levity",
  "centralTheme": "Power corrupts, but redemption is possible through sacrifice",
  "narrativeVoice": "Third-person limited, deep POV following the protagonist",
  "createdAt": "2026-06-16T00:00:00Z",
  "updatedAt": "2026-06-16T00:00:00Z"
}
```

**Indexes:**
- `{ storyId: 1 }` (unique)

---

### `chapters`

Bridges Act→Scene hierarchy. Enables sequential scene planning within act/chapter boundaries.

```json
{
  "_id": "chapter_1",
  "storyId": "story_1",
  "actNumber": 1,
  "chapterNumber": 1,
  "title": "Chapter One: The Awakening",
  "summary": "The hero discovers their hidden power during a festival",
  "goal": "Establish the hero's ordinary world and present the call to adventure",
  "scenes": ["scene_100", "scene_101"],
  "status": "planned",
  "createdAt": "2026-06-16T00:00:00Z",
  "updatedAt": "2026-06-16T00:00:00Z"
}
```

**Status values:** `planned`, `outlined`, `in_progress`, `completed`

**Indexes:**
- `{ storyId: 1, actNumber: 1, chapterNumber: 1 }` (unique)
- `{ storyId: 1 }`

---

### `scene_turns`

Records each agent turn during scene generation.

```json
{
  "_id": "turn_1",
  "sceneId": "scene_100",
  "storyId": "story_1",
  "number": 1,
  "agentId": "director",
  "role": "scene_director",
  "input": "Plan the opening turn for scene_100",
  "output": "Arya enters the throne room, guards flanking...",
  "model": "claude-sonnet",
  "status": "done",
  "error": "",
  "promptTokens": 4500,
  "completionTokens": 820,
  "durationMs": 12400,
  "createdAt": "2026-06-20T00:00:00Z",
  "updatedAt": "2026-06-20T00:00:05Z"
}
```

**Status values:** `pending`, `running`, `done`, `failed`, `skipped`

**Role values:** `director`, `character`, `narrator`, `editor`, `canon_guard`, `critic`, `state_extractor`, `world`, `arc`, `memory`

**Indexes:**
- `{ sceneId: 1, number: 1 }` (unique)
- `{ sceneId: 1, role: 1 }`
- `{ storyId: 1, createdAt: -1 }`

---

### `agent_runs`

Execution log for each agent invocation (one agent may produce multiple turns).

```json
{
  "_id": "run_1",
  "storyId": "story_1",
  "sceneId": "scene_100",
  "turnId": "turn_1",
  "agentType": "director",
  "input": {
    "beatIntent": "Hero arrives at the castle",
    "participants": ["char_1", "char_2"]
  },
  "output": {
    "whoActs": ["char_1"],
    "pressure": 0.5,
    "escalation": "Castle guards confront the hero"
  },
  "model": "claude-sonnet",
  "status": "success",
  "error": "",
  "durationMs": 8400,
  "createdAt": "2026-06-20T00:00:00Z"
}
```

**Agent types:** `director`, `character`, `narrator`, `editor`, `canon_guard`, `critic`, `state_extract`, `world`, `arc`, `memory`, `orchestrator`

**Indexes:**
- `{ storyId: 1, sceneId: 1, createdAt: -1 }`
- `{ storyId: 1, agentType: 1 }`
- `{ sceneId: 1 }`

---

### `canon_deltas`

Append-only log of canon changes, projected into `stories.canonPins` on scene accept.

```json
{
  "_id": "delta_1",
  "storyId": "story_1",
  "sceneId": "scene_100",
  "genId": "gen_1",
  "category": "character_state",
  "fact": "Arya.location",
  "oldValue": "castle_gates",
  "newValue": "throne_room",
  "source": "state_extractor",
  "confidence": 0.95,
  "createdAt": "2026-06-20T00:00:00Z"
}
```

**Categories:** `character_state`, `relationship`, `location`, `timeline`, `world`, `plot`, `lore`, `fact`

**Indexes:**
- `{ storyId: 1, createdAt: -1 }`
- `{ sceneId: 1 }`
- `{ storyId: 1, category: 1 }`

---

### `timeline_events`

```json
{
  "_id": "tl_1",
  "storyId": "story_1",
  "sceneId": "scene_100",
  "title": "Arrival at Castle",
  "eventType": "scene",
  "description": "Arya arrives at the castle seeking the truth about her father's death",
  "dependencies": ["tl_0"],
  "consequences": ["tl_2"],
  "order": 12,
  "createdAt": ""
}
```

**Indexes:**
- `{ storyId: 1, order: 1 }`
- `{ storyId: 1, sceneId: 1 }`

---

## Frontend Types

The frontend mirrors backend models as TypeScript interfaces in `web/src/api/types.ts`. Key mappings:

| Frontend Type | Backend Collection | Notes |
|---|---|---|
| `Story` | `stories` | DAG root, title + canon pins |
| `GraphNode` | `scenes` (as graph nodes) | Primary DAG node type used by React Flow |
| `GraphEdge` | `scene_edges` | Directed edges with type |
| `Scene` | `scenes` | Legacy chapter-scoped scene |
| `SceneEdge` | `scene_edges` | Legacy scene-scoped edge |
| `Generation` | `generations` | LLM output records |
| `Topology` | `scenes` + `scene_edges` | Full DAG snapshot |
| `Character` | `characters` | Immutable character definition |
| `Location` | (separate collection) | Story settings |
| `Lore` | (separate collection) | World-building entries |
| `SceneTurn` | (interactive gen) | Single turn in dialogue generation |
| `StorySummary` | `summaries` | Hierarchical summaries |
| `Casting` | (separate) | Actor→Character links |
| `SceneStructure` | embedded in scene | Turn-based flow config |
| `SceneTurn` | `scene_turns` | Agent turn during generation |
| `AgentRun` | `agent_runs` | Agent execution log |
| `CanonDelta` | `canon_deltas` | Append-only canon change log |

### UI-Only Types (no backend equivalent)

| Type | Purpose |
|---|---|
| `StoryStats` | Aggregated node counts for sidebar (computed client-side) |
| `CreateStoryPayload` / `CreateNodePayload` etc. | Request body shapes |

## Entity Relationships

```
stories 1──1 bibles                (storyId)
stories 1──* chapters              (storyId)
stories 1──* locations             (storyId)
stories 1──* scenes                (storyId)
stories 1──* scene_edges           (storyId)
stories 1──* characters            (storyId)
stories 1──* character_state       (storyId)
stories 1──* character_memories    (storyId)
stories 1──* summaries             (storyId)
stories 1──* timeline_events       (storyId)
stories 1──* scene_turns           (storyId)
stories 1──* agent_runs            (storyId)
stories 1──* canon_deltas          (storyId)

scenes 1──* scene_edges            (fromSceneId / toSceneId)
scenes 1──* generations            (sceneId)
scenes 1──* scene_turns            (sceneId)
scenes 1──* agent_runs             (sceneId)
scenes 1──* canon_deltas           (sceneId)
scenes 1──* character_state        (sceneId)
scenes 1──* character_memories     (sceneId)
scenes 1──* summaries              (sceneId)

characters 1──* character_state    (characterId)
characters 1──* character_memories (characterId)

chapters 1──* scenes               (scenes array references)

agent_runs 1──* scene_turns        (turnId)
```

---

## Index Management

Indexes are defined in `internal/repository/mongo/client.go` and created on application startup via `EnsureIndexes()`. This avoids the need for a migration system.

Each index is created individually via `CreateOne`. If an index already exists with the same auto-generated name but different options (e.g. existing non-unique vs requested unique), the conflict is logged as a WARN and skipped — the application continues without crashing. This handles schema evolution across deployments.

```go
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
    coll := db.Collection(collName)
    for _, m := range models {
        _, err := coll.Indexes().CreateOne(ctx, m)
        if err != nil && strings.Contains(err.Error(), "IndexKeySpecsConflict") {
            slog.Warn("index conflict, skipping", "collection", collName)
            continue
        }
        if err != nil {
            return fmt.Errorf("create index for %s: %w", collName, err)
        }
    }
    return nil
}
```
