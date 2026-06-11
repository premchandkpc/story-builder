# API Reference

Base: `/api/v1`

All request/response bodies are JSON.

---

## Characters

### `POST /characters`

Create a character.

```json
{ "name": "Arthur", "traits": ["brave"], "voice_samples": [], "relationships": {} }
```

Response `201`: Character object with `id`, `version`, `created_at`.

### `GET /characters`

List characters (latest version only).

Response `200`: `[{...character}]`

### `GET /characters/{id}`

Get character latest version.

Response `200`: Character object.

### `PUT /characters/{id}`

Update character (creates new version).

Body: same shape as POST.

Response `200`: Character object with incremented `version`.

---

## Locations

### `POST /locations`

```json
{ "name": "Castle", "description": "Stone keep", "props": ["throne"] }
```

Response `201`: Location object.

### `GET /locations`

List locations (latest version only).

### `GET /locations/{id}`

Get location latest version.

### `PUT /locations/{id}`

Update location (creates new version).

---

## Lore

### `POST /lore`

```json
{ "tags": ["magic", "history"], "content": "The red sun rises in the west." }
```

Response `201`: Lore object.

### `GET /lore`

List all lore entries.

### `POST /lore/search`

Search by tags and/or vector similarity.

```json
{ "tags": ["magic"], "embedding": [0.1, 0.2, ...], "limit": 5 }
```

If `embedding` provided, uses pgvector cosine similarity. Otherwise filters by tags (GIN index).

Response `200`: `[{...lore}]`

---

## Stories

### `POST /stories`

```json
{ "title": "The Red Sun" }
```

Response `201`: Story object with `id`, `canon_pins: {}`.

### `GET /stories`

List stories.

### `GET /stories/{id}`

Get story with `canon_pins`.

### `GET /stories/{storyID}/topology`

Get full DAG: nodes + edges.

```json
{ "nodes": [...], "edges": [...] }
```

---

## Nodes (per story)

### `POST /stories/{storyID}/nodes`

```json
{
  "beat_intent": "Arthur meets the wizard",
  "character_refs": ["<uuid>"],
  "location_ref": "<uuid or null>",
  "pov": "Arthur",
  "tone": "mysterious",
  "target_words": 500
}
```

Response `201`: Node object with `status: "draft"`.

### `GET /stories/{storyID}/nodes`

List nodes for story.

### `GET /stories/{storyID}/nodes/{id}`

Get node.

### `PUT /stories/{storyID}/nodes/{id}`

Update node fields. Body same as create.

Response `200`: Updated node.

### `PUT /stories/{storyID}/nodes/{id}/scene/structure`

Set the multi-agent scene structure definition.

```json
{
  "flow_type": "dialogue",
  "character_order": ["<uuid>", "<uuid>"],
  "situation_flow": "Arthur enters the throne room. Morgaine is waiting.",
  "max_turns": 6
}
```

`flow_type`: `monologue`, `dialogue`, `round_robin`, `parallel`, `custom`.

Response `200`.

### `GET /stories/{storyID}/nodes/{id}/scene/structure`

Get scene structure definition.

Response `200`: SceneStructure object.

### `POST /stories/{storyID}/nodes/{id}/scene/start`

Start multi-agent scene (first turn).

Response `200`: SceneTurn object.

### `POST /stories/{storyID}/nodes/{id}/scene/next`

Generate next agent turn.

Response `200`: SceneTurn object.

### `POST /stories/{storyID}/nodes/{id}/scene/finish`

Finish scene → assemble turns into generation, extract state, update summary.

Response `200`: `{"output": "assembled scene text"}`.

### `GET /stories/{storyID}/nodes/{id}/scene/turns`

List all turns for this node/scene.

Response `200`: `[{...SceneTurn}]`

### `POST /stories/{storyID}/nodes/{id}/generate`

Trigger LLM generation for node.

Response `202`: `{id, node_id, context_hash, ...generation}`.

### `POST /stories/{storyID}/nodes/{id}/accept`

Accept a generation.

```json
{ "generation_id": "<uuid>" }
```

Response `200`.

### `GET /stories/{storyID}/nodes/{id}/generations`

List all generations for a node.

Response `200`: `[{...generation}]`

---

## Edges (per story)

### `POST /stories/{storyID}/edges`

Create directed edge between nodes.

```json
{
  "from_node": "<uuid>",
  "to_node": "<uuid>",
  "edge_type": "seq"
}
```

`edge_type`: `seq`, `fork`, `join`, `choice`.

Response `201`.

### `GET /stories/{storyID}/edges`

List all edges for story.

Response `200`: `[{story_id, from_node, to_node, edge_type}]`
