# API Reference

Base URL: `/api/v1`

All requests and responses are JSON. Standard error format: `{"error": "message"}`

---

## Health

### `GET /api/v1/healthz`

```
Response 200: {"status": "ok"}
```

---

## Story Generator

### `POST /api/v1/stories/generate`

Enqueues an LLM story generation job from a synopsis.

**Request:**
```json
{"synopsis": "A young hero discovers their destiny..."}
```

**Response 202:**
```json
{"story_id": "", "status": "pending"}
```

---

## Actors

### `POST /api/v1/actors`

Create an actor (physical entity who plays a character role).

**Request:**
```json
{
  "name": "Mark Hamill",
  "gender": "male",
  "ethnicity": "American",
  "race": "white",
  "skin_tone": "fair",
  "eye_color": "blue",
  "hair_color": "brown",
  "hair_style": "short",
  "build": "slim",
  "height_cm": 175,
  "weight_kg": 77,
  "age": 25,
  "nationality": "American",
  "traits": {"voice_range": "tenor", "specialty": "voice_acting"}
}
```

**Response 201:** Full actor object.

### `GET /api/v1/actors`

List all actors.

### `GET /api/v1/actors/{id}`

Get actor by UUID.

### `PUT /api/v1/actors/{id}`

Update actor. Same body as create. Returns updated actor.

---

## Characters

### `POST /api/v1/characters`

Create a versioned character.

**Request:**
```json
{
  "name": "Darth Vader",
  "persona": "antagonist",
  "backstory": "Once a Jedi Knight, Anakin Skywalker fell to the dark side...",
  "moral_alignment": "chaotic evil",
  "personality": ["brooding", "ruthless"],
  "flaws": ["arrogance", "attachment"],
  "goals": ["crush the rebellion"],
  "traits": ["dark side", "Sith Lord"],
  "voice_samples": ["I am your father.", "The Force is strong with this one."],
  "relationships": {"son": "...", "master": "Palpatine"},
  "parent_id": null
}
```

**Response 201:** Full character object with `id`, `version=1`.

### `GET /api/v1/characters`

List all characters (latest version each).

### `GET /api/v1/characters/{id}`

Get latest version of character. Query param `?version=N` for specific version.

### `PUT /api/v1/characters/{id}`

Create new version of character. Same body as create. Response returns incremented `version`.

---

## Character Traits

### `POST /api/v1/character-traits`

**Request:**
```json
{
  "name": "brave",
  "category": "personality",
  "description": "Willing to face danger without hesitation"
}
```

### `GET /api/v1/character-traits`

List all traits.

### `GET /api/v1/character-traits/{id}`

Get trait by ID.

### `POST /api/v1/characters/{characterID}/traits/assign`

Assign a trait to a character with intensity (1-10).

**Request:**
```json
{"trait_id": "...", "intensity": 8, "note": "Shown during battle scenes"}
```

### `DELETE /api/v1/characters/{characterID}/traits/{traitID}`

Unassign a trait from a character.

### `GET /api/v1/characters/{characterID}/traits`

Get all trait assignments for a character with trait details.

---

## Locations

### `POST /api/v1/locations`

**Request:**
```json
{
  "name": "Death Star",
  "description": "An armored space station...",
  "props": ["throne room", "detention block", "trash compactor"]
}
```

**Response 201:** Full location with `id`, `version=1`.

### `GET /api/v1/locations`

List all locations (latest version).

### `GET /api/v1/locations/{id}`

Get latest version. Query param `?version=N` for specific version.

### `PUT /api/v1/locations/{id}`

Create new version. Returns incremented `version`.

---

## Lore

### `POST /api/v1/lore`

**Request:**
```json
{
  "tags": ["force", "jedi"],
  "content": "The Force is a mystical energy field..."
}
```

**Response 201:** Full lore object.

### `GET /api/v1/lore`

List all lore entries.

### `POST /api/v1/lore/search`

Search lore by tags or vector similarity.

**Request (tag search):**
```json
{"tags": ["jedi", "force"], "limit": 10}
```

**Request (vector similarity search):**
```json
{
  "embedding": [0.1, -0.05, ...],
  "limit": 5
}
```

---

## Casting

### `POST /api/v1/stories/{storyID}/casting`

Cast an actor to play a character in a story.

**Request:**
```json
{
  "actor_id": "...",
  "character_id": "...",
  "role_type": "lead"
}
```

**Response 201:** Casting object.

### `GET /api/v1/stories/{storyID}/casting`

List casting for a story (with actor + character names).

### `GET /api/v1/casting/actor/{actorID}`

List all roles for an actor across stories (with character name + story title).

### `GET /api/v1/casting/character/{characterID}`

List all actors who played a character across stories (with actor name + story title).

---

## Stories

### `POST /api/v1/stories`

**Request:**
```json
{"title": "A New Hope"}
```

**Response 201:** Full story object with empty `canon_pins`.

### `GET /api/v1/stories`

List all stories.

### `GET /api/v1/stories/{id}`

Get story by ID.

### `GET /api/v1/stories/{storyID}/topology`

Get full graph topology (nodes + edges) for a story.

**Response 200:**
```json
{
  "nodes": [...],
  "edges": [...]
}
```

---

## Nodes (Scenes)

### `POST /api/v1/stories/{storyID}/nodes`

**Request:**
```json
{
  "beat_intent": "Luke meets Obi-Wan for the first time",
  "character_refs": ["char-uuid-1", "char-uuid-2"],
  "location_ref": "loc-uuid-1",
  "pov": "Luke",
  "tone": "mysterious",
  "target_words": 500,
  "scene_structure": {
    "flow_type": "dialogue",
    "character_order": ["uuid1", "uuid2"],
    "situation_flow": "A young farm boy meets an old wizard...",
    "max_turns": 10
  }
}
```

**Response 201:** Full node with `status: "draft"`.

### `GET /api/v1/stories/{storyID}/nodes`

List all nodes for a story.

### `GET /api/v1/stories/{storyID}/nodes/{id}`

Get node by ID.

### `PUT /api/v1/stories/{storyID}/nodes/{id}`

Update node. Same body as create.

---

## Edges

### `POST /api/v1/stories/{storyID}/edges`

Create a directed edge between two nodes.

**Request:**
```json
{
  "from_node": "...",
  "to_node": "...",
  "edge_type": "seq"
}
```

Valid `edge_type` values: `seq`, `fork`, `join`, `choice`.

### `GET /api/v1/stories/{storyID}/edges`

List all edges for a story.

---

## Scene Generation

### `POST /api/v1/stories/{storyID}/nodes/{id}/generate`

Enqueue an LLM generation for this node.

**Response 202:**
```json
{
  "id": "gen-uuid",
  "node_id": "node-uuid",
  "context_hash": "sha256hex...",
  "prompt_snapshot": "POV: Luke | Tone: mysterious | Beat: ...",
  "output": "",
  "model": "claude-sonnet"
}
```

### `GET /api/v1/stories/{storyID}/nodes/{id}/generations`

List all generations for a node (newest first).

### `POST /api/v1/stories/{storyID}/nodes/{id}/accept`

Accept a generation and reject others.

**Request:**
```json
{"generation_id": "gen-uuid"}
```

---

## Scene Structure (Multi-Agent)

### `PUT /api/v1/stories/{storyID}/nodes/{id}/scene/structure`

Set the scene structure for a node.

**Request:**
```json
{
  "scene_structure": {
    "flow_type": "dialogue",
    "character_order": ["uuid1", "uuid2"],
    "situation_flow": "scene description",
    "max_turns": 10
  }
}
```

### `GET /api/v1/stories/{storyID}/nodes/{id}/scene/structure`

Get the scene structure.

### `POST /api/v1/stories/{storyID}/nodes/{id}/scene/start`

Start a multi-agent scene. Returns first turn.

### `POST /api/v1/stories/{storyID}/nodes/{id}/scene/next`

Generate next turn in the scene.

### `POST /api/v1/stories/{storyID}/nodes/{id}/scene/finish`

Finish the scene. Assembles turns into a generation.

### `GET /api/v1/stories/{storyID}/nodes/{id}/scene/turns`

List all turns for the scene.

---

## Summaries

### `GET /api/v1/stories/{storyID}/summaries/level`

Get latest summary by level.

Query: `?level=act` or `?level=story` (default: `act`)

### `GET /api/v1/stories/{storyID}/summaries/count`

Count summaries by level.

Query: `?level=scene` or `?level=act` or `?level=story` (default: `scene`)

Response: `{"count": 42}`

### `GET /api/v1/stories/{storyID}/summaries/elevate`

Check if summaries should be elevated (consolidated) to the next level.

Query: `?level=scene&threshold=10` (default threshold: 10)

Response: `{"should_elevate": true, "level": "scene", "threshold": 10}`

### `GET /api/v1/stories/{storyID}/summaries/nodes/{nodeID}`

Get scene-level summary for a specific node.
