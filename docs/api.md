# API Reference

Base URL: `/api/v1`

All requests and responses are JSON. Standard error format: `{"error": "message"}`

Error responses ≥500 are logged server-side via `slog.Error`.

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
- Request timeout via `AbortController` (default 30s; `stories.generate` uses 300s to accommodate LLM latency)
- HTTP error detection (non-2xx → thrown `Error`)
- 204 No Content → `undefined`

All fetch calls go through a single `request<T>(path, init)` helper — no raw `fetch()` in components.

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

Update story title/metadata.

**Request:**
```json
{"title": "A New Hope"}
```

### `DELETE /api/v1/stories/{id}`

Delete story and all associated data (scenes, edges, characters, memories, etc.).

### `GET /api/v1/stories/{storyID}/topology`

Get full graph topology (nodes + edges) for a story.

**Response 200:**
```json
{
  "nodes": [...],
  "edges": [...]
}
```

### `POST /api/v1/stories/generate`

Generate a full story outline from a synopsis via LLM. Server timeout: 5min.

**Request:**
```json
{"synopsis": "A young hero discovers their destiny..."}
```

**Response 202:**
```json
{"story_id": "6a31a6bab21a1af6e0004db1", "status": "outlined"}
```

### `POST /api/v1/stories/generate-title`

Generate a story title from a synopsis via LLM.

**Request:**
```json
{"synopsis": "A young hero discovers their destiny..."}
```

**Response 200:**
```json
{"title": "The Hero's Awakening"}
```

---

## Graph (V2 Nodes)

### `POST /api/v1/stories/{storyID}/nodes`

Create a scene node in the DAG.

**Request:**
```json
{
  "beat_intent": "Hero discovers the artifact",
  "character_refs": [],
  "location_ref": "ancient ruins",
  "pov": "third-limited",
  "tone": "mysterious",
  "target_words": 500
}
```

**Response 201:** Full node object with generated ID.

### `GET /api/v1/stories/{storyID}/nodes`

List all nodes for a story.

### `GET /api/v1/stories/{storyID}/nodes/{nodeID}`

Get a single node.

### `PUT /api/v1/stories/{storyID}/nodes/{nodeID}`

Update a node. Same body as create.

### `DELETE /api/v1/stories/{storyID}/nodes/{nodeID}`

Delete a node.

### `POST /api/v1/stories/{storyID}/nodes/{nodeID}/generate`

Trigger LLM prose generation for a node. No request body.

**Response 200:** Full generation object.

### `GET /api/v1/stories/{storyID}/nodes/{nodeID}/generations`

List all generations for a node (newest first).

### `POST /api/v1/stories/{storyID}/nodes/{nodeID}/accept`

Accept a generation and update the node's content. Atomically sets `scene.acceptedGenerationId` + `scene.status = "accepted"`. The generation's `accepted` flag is updated as a derived field (backward compat).

**Request:**
```json
{"generation_id": "gen_uuid"}
```

**Response 200:**
```json
{"status": "accepted"}
```

---

## Graph (V2 Edges)

### `POST /api/v1/stories/{storyID}/edges`

Create a directed edge between two nodes.

**Request:**
```json
{
  "from_node": "node_a",
  "to_node": "node_b",
  "edge_type": "seq",
  "condition": ""
}
```

Valid `edge_type` values: `seq`, `fork`, `join`, `choice`

### `GET /api/v1/stories/{storyID}/edges`

List all edges for a story.

### `DELETE /api/v1/stories/{storyID}/edges`

Delete an edge. Uses query params.

**Query params:** `from_scene=node_a&to_scene=node_b`

**Response 204:** No Content.

---

---

## Characters (V2 Top-Level)

### `POST /api/v1/characters`

Create a character. `storyId` is required in body.

**Request:**
```json
{
  "name": "Arya",
  "storyId": "story_1",
  "persona": "rogue",
  "backstory": "Orphaned young, raised by the guild",
  "goals": ["find family"],
  "flaws": ["reckless"],
  "want": "Find her family",
  "need": "Learn to trust again",
  "arcType": "redemption"
}
```

**Response 201:** Full character object with `char_id` and `version: 1`.

### `GET /api/v1/characters`

List characters for a story. Requires `?story_id=...` query param (returns `[]` if omitted).

### `GET /api/v1/characters/{charID}`

Get character by logical ID (latest version).

### `PUT /api/v1/characters/{charID}`

Updates a character by creating a new versioned document (immutable log).

**Request:**
```json
{
  "name": "Arya Stark",
  "storyId": "story_1",
  "persona": "warrior",
  "goals": ["find family", "avenge father"],
  "flaws": ["reckless", "vengeful"]
}
```

**Response 200:** Full character object with incremented version, new `_id`, and same `char_id`.

---

## Characters (Story-Based)

### `POST /api/v1/stories/{storyID}/characters`

Create a character within a story. `storyID` is injected from the URL path.

**Request:** Same body as top-level create, but no `storyId` needed.

**Response 201:** Full character object.

### `GET /api/v1/stories/{storyID}/characters`

List all characters in a story.

---

## Memories

### `GET /api/v1/characters/{charID}/memories`

List semantic memories for a character.

### `POST /api/v1/characters/{charID}/memories/search`

Search memories by semantic similarity. Generates a query embedding and performs vector search.

**Request:**
```json
{
  "story_id": "story_1",
  "query": "betrayal at the castle",
  "limit": 10
}
```

**Required:** `story_id`, `query`. `limit` defaults to 10, max 50.

**Response 200:** Array of memory objects ranked by relevance. Returns 501 if embedding service is not configured.

---

## Timeline

### `POST /api/v1/stories/{storyID}/timeline`

Create a timeline event.

**Request:**
```json
{
  "title": "Arrival at Castle",
  "description": "Hero arrives at the castle",
  "order": 12
}
```

**Response 201:** Full event object.

### `GET /api/v1/stories/{storyID}/timeline`

List all timeline events, sorted by order.

---

## Locations

### `GET /api/v1/stories/{storyID}/locations`

List all locations for a story.

### `POST /api/v1/stories/{storyID}/locations`

**Request:**
```json
{
  "name": "Castle Gates",
  "description": "The imposing main entrance",
  "props": ["portcullis", "moat"]
}
```

**Response 201:** Full location object.

### `GET /api/v1/locations/{id}`

Get location by ID.

### `PUT /api/v1/locations/{id}`

Update a location (name not currently mutable via this endpoint).

**Request:**
```json
{
  "description": "The heavily fortified main entrance",
  "props": ["portcullis", "moat", "drawbridge"]
}
```

**Response 200:** Updated location object.

---

## Summaries

### `GET /api/v1/stories/{storyID}/summaries/level`

Get latest summary by level.

Query: `?level=act` or `?level=story` (default: `act`)

### `GET /api/v1/stories/{storyID}/summaries/scenes/{sceneID}`

Get scene-level summary.

### `GET /api/v1/stories/{storyID}/summaries/nodes/{nodeID}`

Get node-level summary.

**Note:** Both `/scenes/{sceneID}` and `/nodes/{nodeID}` map to the same handler, which reads whichever param is present.

---

## Generations

### `GET /api/v1/generations/{genID}/status`

Get generation status with token usage and timing.

**Response 200:**
```json
{
  "id": "gen_uuid",
  "sceneId": "scene_uuid",
  "content": "The castle gates groaned open...",
  "model": "claude-sonnet-4-20250514",
  "status": "success",
  "error": "",
  "promptTokens": 4500,
  "completionTokens": 820,
  "totalTokens": 5320,
  "durationMs": 18400,
  "updatedAt": "2026-06-16T00:00:00Z"
}
```

**Status values:** `pending`, `running`, `success`, `partial_success`, `failed`

### `GET /api/v1/generations/{genID}/progress`

Server-Sent Events stream for generation progress.

**Response 200:** `text/event-stream`

```
event: connected
data: {"genId":"gen_uuid"}

event: progress
data: {"genId":"gen_uuid","step":"generating","status":"running"}

event: progress
data: {"genId":"gen_uuid","step":"complete","status":"success"}
```

---

## Blueprint

### `GET /api/v1/stories/{storyID}/blueprint`

Get the story blueprint (premise, theme, acts, character arcs, plot threads).

**Response 200:**
```json
{
  "premise": "A young blacksmith discovers ancient power",
  "theme": "Power and responsibility",
  "acts": [
    {"number": 1, "title": "Discovery", "summary": "The artifact is found"}
  ],
  "characterArcs": [],
  "plotThreads": [],
  "endingState": "ambiguous"
}
```

### `PUT /api/v1/stories/{storyID}/blueprint`

Update the story blueprint.

**Response 200:**
```json
{"status": "updated"}
```

---

## Story Bible

### `GET /api/v1/stories/{storyID}/bible`

Get the story bible.

**Response 200:** Full bible object, or `{"error": "not found"}`.

### `POST /api/v1/stories/{storyID}/bible/generate`

Generate a new bible via LLM (claude-sonnet). No request body.

**Response 201:** Full bible object.

### `PUT /api/v1/stories/{storyID}/bible`

Update the story bible. Replaces the existing bible document with the sent body.

**Request:** Full or partial bible object (fields are replaced at document level).

**Response 200:** Updated bible object.

### `DELETE /api/v1/stories/{storyID}/bible`

Delete the story bible.

**Response 204:** No Content.

---

## Chapters

### `POST /api/v1/stories/{storyID}/chapters`

Create a chapter.

**Request:**
```json
{
  "actNumber": 1,
  "chapterNumber": 1,
  "title": "Chapter One: The Awakening",
  "summary": "The hero discovers their power",
  "goal": "Establish the ordinary world"
}
```

**Response 201:** Full chapter object with generated ID.

**Note:** If `chapterNumber` is omitted, the field is set to its zero value.

### `GET /api/v1/stories/{storyID}/chapters`

List all chapters for a story, sorted by actNumber then chapterNumber.

### `GET /api/v1/stories/{storyID}/chapters/{id}`

Get a single chapter.

### `PUT /api/v1/stories/{storyID}/chapters/{id}`

Update a chapter.

**Request:**
```json
{
  "title": "Updated Title",
  "summary": "Updated summary",
  "goal": "Updated goal",
  "status": "in_progress"
}
```

### `DELETE /api/v1/stories/{storyID}/chapters/{id}`

Delete a chapter.

**Response 204:** No Content.

---

## Experimental Routes (Stubs)

All feature stubs are behind `/api/v1/experimental`:

| Group | Endpoint | Behavior |
|---|---|---|
| Scene turns | `PUT/GET .../scene/structure`, `POST .../scene/{start,next,finish}`, `GET .../scene/turns` | 501 |
| Actors | `GET/POST /experimental/actors`, `GET/PUT /experimental/actors/{id}` | `[]` / 501 |
| Character traits | `GET /experimental/character-traits`, `POST assign`, `DELETE unassign` | `[]` / 501 |
| Lore | `GET /experimental/lore`, `POST /experimental/lore`, `POST .../search` | `[]` / 501 |
| Casting | `POST /experimental/stories/{id}/casting`, `GET .../casting`, `GET .../casting/actor/{id}`, `GET .../casting/character/{id}` | `[]` / 501 |
| `PUT /api/v1/locations/{id}` | Name field not mutable; accepts `description` + `props` only |
