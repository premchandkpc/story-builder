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

**Indexes:**
- `{ title: 1 }`
- `{ status: 1 }`

---

### `scenes`

```json
{
  "_id": "scene_100",
  "storyId": "story_1",
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
  "currentState": {
    "health": 90,
    "location": "castle",
    "mood": "determined",
    "inventory": ["dagger", "cloak"]
  },
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

Event-sourced state history. Always append, never overwrite.

```json
{
  "_id": "state_1",
  "storyId": "story_1",
  "characterId": "char_1",
  "sceneId": "scene_100",
  "changes": {
    "health": -10,
    "trust": 5,
    "location": "dungeon"
  },
  "fullState": {
    "health": 80,
    "location": "dungeon",
    "mood": "fearful",
    "inventory": ["dagger"],
    "relationships": {
      "char_2": "trusts"
    }
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
  "model": "claude-sonnet",
  "accepted": false,
  "validationResult": null,
  "createdAt": ""
}
```

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

### `timeline_events`

```json
{
  "_id": "tl_1",
  "storyId": "story_1",
  "sceneId": "scene_100",
  "title": "Arrival at Castle",
  "description": "Arya arrives at the castle seeking the truth about her father's death",
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

### UI-Only Types (no backend equivalent)

| Type | Purpose |
|---|---|
| `StoryStats` | Aggregated node counts for sidebar (computed client-side) |
| `CreateStoryPayload` / `CreateNodePayload` etc. | Request body shapes |

## Entity Relationships

```
stories 1──* scenes                (storyId)
stories 1──* scene_edges           (storyId)
stories 1──* characters            (storyId)
stories 1──* character_state       (storyId)
stories 1──* character_memories    (storyId)
stories 1──* summaries             (storyId)
stories 1──* timeline_events       (storyId)

scenes 1──* scene_edges            (fromSceneId / toSceneId)
scenes 1──* generations            (sceneId)
scenes 1──* character_state        (sceneId)
scenes 1──* character_memories     (sceneId)
scenes 1──* summaries              (sceneId)

characters 1──* character_state    (characterId)
characters 1──* character_memories (characterId)
```

---

## Index Management

Indexes are defined in `internal/repository/mongo/indexes.go` and created on application startup via `ensureIndexes()`. This avoids the need for a migration system.

```go
func ensureIndexes(ctx context.Context, db *mongo.Database) error {
    // Stories
    // Scenes
    // SceneEdges
    // Characters
    // CharacterState
    // CharacterMemories
    // Generations
    // Summaries
    // TimelineEvents
}
```
