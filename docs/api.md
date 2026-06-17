# API Reference

Base URL: `/api/v1`

All requests and responses are JSON. Standard error format: `{"error": "message"}`

## Frontend API Client

The frontend consumes all endpoints through `web/src/api/client.ts` — a namespaced `api` object with type-safe methods.

```typescript
import { api } from "../api/client"
const stories = await api.stories.list()
const story   = await api.stories.get("story_1")
const created = await api.stories.create({ title: "My Story" })
```

Every response is typed via generics (e.g. `api.stories.list()` returns `Promise<Story[]>`). The client handles:
- JSON serialization/deserialization
- Request timeout (default 30s) via `AbortController`
- HTTP error detection (non-2xx → thrown `Error`)
- 204 No Content → `undefined`

All fetch calls go through a single `request<T>(path, init)` helper — no raw `fetch()` in components.

### API Groups

The client organizes endpoints into namespaced groups matching the backend route structure:

---

## Health

### `GET /api/v1/healthz`

```
Response 200: {"status": "ok"}
```

---

## Stories

### `POST /api/v1/stories`

**Request:**
```json
{"title": "A New Hope"}
```

**Response 201:** Full story object.

### `GET /api/v1/stories`

List all stories.

### `GET /api/v1/stories/{id}`

Get story by ID.

### `PUT /api/v1/stories/{id}`

Update story title.

**Request:**
```json
{"title": "A New Hope"}
```

### `DELETE /api/v1/stories/{id}`

Delete story and all associated data (scenes, edges, characters, memories, etc.).

### `GET /api/v1/stories/{storyID}/topology`

Get full graph topology (scenes + edges) for a story.

**Response 200:**
```json
{
  "nodes": [...],
  "edges": [...]
}
```

---

## Scenes

### `POST /api/v1/stories/{storyID}/scenes`

**Request:**
```json
{
  "title": "Arrival",
  "beat_intent": "Hero arrives at the castle",
  "participants": ["char_1", "char_2"],
  "location_ref": "loc_1",
  "pov": "hero",
  "tone": "mysterious",
  "target_words": 500,
  "flow_type": "dialogue"
}
```

**Response 201:** Full scene object.

### `GET /api/v1/stories/{storyID}/scenes`

List all scenes for a story.

### `GET /api/v1/stories/{storyID}/scenes/{id}`

Get scene by ID.

### `PUT /api/v1/stories/{storyID}/scenes/{id}`

Update scene. Same body as create.

### `DELETE /api/v1/stories/{storyID}/scenes/{id}`

Delete scene and its edges.

---

## Edges

### `POST /api/v1/stories/{storyID}/edges`

Create a directed edge between two scenes.

**Request:**
```json
{
  "from_scene": "scene_a",
  "to_scene": "scene_b",
  "type": "seq",
  "condition": ""
}
```

Valid `type` values: `seq`, `fork`, `join`, `choice`, `parallel`

### `GET /api/v1/stories/{storyID}/edges`

List all edges for a story.

### `DELETE /api/v1/stories/{storyID}/edges`

Delete an edge.

**Request:**
```json
{
  "from_scene": "scene_a",
  "to_scene": "scene_b"
}
```

---

## Characters

### `POST /api/v1/stories/{storyID}/characters`

**Request:**
```json
{
  "name": "Arya",
  "persona": "rogue",
  "backstory": "Orphaned young, raised by the guild",
  "personality": {"courage": 8, "kindness": 4},
  "moral_alignment": "chaotic neutral",
  "goals": ["find family"],
  "flaws": ["reckless"],
  "traits": ["sneaky"],
  "voice_samples": ["I don't need your help."],
  "relationships": {}
}
```

**Response 201:** Full character object.

### `GET /api/v1/stories/{storyID}/characters`

List all characters in a story.

### `GET /api/v1/characters/{id}`

Get character by ID (across stories).

### `PUT /api/v1/characters/{id}`

**Note:** Character definitions are immutable. Update creates a new document. The old ID is preserved — previous scenes reference the original version.

---

## Generation

### `POST /api/v1/stories/{storyID}/scenes/{id}/generate`

Enqueue generation for a scene.

**Response 202:**
```json
{
  "id": "gen_uuid",
  "scene_id": "scene_uuid",
  "context_hash": "sha256hex...",
  "prompt_snapshot": "POV: Arya | Tone: mysterious",
  "output": "",
  "model": "claude-sonnet",
  "status": "pending"
}
```

### `GET /api/v1/stories/{storyID}/scenes/{id}/generations`

List all generations for a scene (newest first).

### `POST /api/v1/stories/{storyID}/scenes/{id}/accept`

Accept a generation and trigger pipeline (state extraction, memories, timeline, summary, validation).

**Request:**
```json
{"generation_id": "gen_uuid"}
```

**Response 200:**
```json
{
  "generation_id": "gen_uuid",
  "status": "processing",
  "pipeline": ["extract", "memory", "timeline", "summary", "validate"]
}
```

### `GET /api/v1/stories/{storyID}/scenes/{id}/generations/{genID}/status`

Get pipeline status for a generation.

**Response 200:**
```json
{
  "generation_id": "gen_uuid",
  "extract": "done",
  "memory": "done",
  "timeline": "done",
  "summary": "done",
  "validate": "done",
  "validation": {"violations": []}
}
```

---

## Story Generator

### `POST /api/v1/stories/generate`

Enqueue an LLM story generation from a synopsis.

**Request:**
```json
{"synopsis": "A young hero discovers their destiny..."}
```

**Response 202:**
```json
{"story_id": "", "status": "pending"}
```

### `POST /api/v1/stories/generate-title`

**Request:**
```json
{"synopsis": "A young hero discovers their destiny..."}
```

**Response 200:**
```json
{"title": "The Hero's Awakening"}
```

---

## Memories

### `GET /api/v1/characters/{charID}/memories`

List memories for a character.

### `POST /api/v1/characters/{charID}/memories/search`

Search memories by semantic similarity.

**Request:**
```json
{
  "query": "betrayal at the castle",
  "limit": 10
}
```

**Response 200:** Array of memory objects ranked by relevance.

---

## Timeline

### `POST /api/v1/stories/{storyID}/timeline`

Create a timeline event.

**Request:**
```json
{
  "title": "Arrival at Castle",
  "description": "Hero arrives at the castle",
  "order": 12,
  "scene_id": "scene_uuid"
}
```

**Response 201:** Full event object.

### `GET /api/v1/stories/{storyID}/timeline`

List all timeline events, sorted by order.

---

## Summaries

### `GET /api/v1/stories/{storyID}/summaries/level`

Get latest summary by level.

Query: `?level=act` or `?level=story` (default: `act`)

### `GET /api/v1/stories/{storyID}/summaries/scenes/{sceneID}`

Get scene-level summary.

### `GET /api/v1/stories/{storyID}/summaries/count`

Count summaries by level.

Query: `?level=scene` (default)

Response: `{"count": 42}`
