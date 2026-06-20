# Current State

## Active today

| Component | Status | Notes |
|-----------|--------|-------|
| **Storage** | MongoDB (primary SSOT) | All domain aggregates stored in Mongo |
| **Cache** | Redis (optional) | Rate limiting, in-memory fallback |
| **LLM** | claude-sonnet, claude-haiku, local-7b (Ollama) | Provider/router abstraction |
| **Graph editor** | Nodes + edges CRUD via REST | V2 graph node model is primary |
| **Generation** | Enqueued as durable Job documents | Worker goroutine runs pipeline |
| **Destkop** | Scene memory, timeline, summaries | Generated pipeline output |
| **Validation** | Post-generation canon validation | 4 validators, claude-haiku |
| **Rate limiting** | Sliding window, method+path scoped keys | Configurable via `cache/rate_limiter.go` |
| **Character versioning** | Append-only version log | CharID + version for immutable history |
| **Prompt context builder** | Assembles context from characters, bible, memory, timeline, summaries | `internal/service/context.go` |

## API surface

### Stable
- `GET /api/v1/healthz`
- `GET/POST /api/v1/stories`
- `GET/PUT/DELETE /api/v1/stories/{id}`
- `POST /api/v1/stories/generate` (outline)
- `GET /api/v1/stories/{id}/topology`
- `GET/POST /api/v1/stories/{id}/nodes`
- `GET/PUT/DELETE /api/v1/stories/{id}/nodes/{id}`
- `POST /api/v1/stories/{id}/nodes/{id}/generate`
- `GET /api/v1/stories/{id}/nodes/{id}/generations`
- `POST /api/v1/stories/{id}/nodes/{id}/accept`
- `GET/POST/DELETE /api/v1/stories/{id}/edges`
- `DELETE /api/v1/stories/{id}/edges/{id}`
- `GET/POST /api/v1/stories/{id}/characters`
- `GET/POST /api/v1/stories/{id}/locations`
- `GET/POST /api/v1/stories/{id}/timeline`
- `GET /api/v1/stories/{id}/summaries/level`
- `GET /api/v1/stories/{id}/summaries/nodes/{id}`
- `GET/PUT /api/v1/stories/{id}/blueprint`
- `GET /api/v1/characters` (list with `?story_id=`)
- `POST /api/v1/characters`
- `GET/PUT /api/v1/characters/{id}`
- `GET /api/v1/characters/{id}/memories`
- `POST /api/v1/characters/{id}/memories/search`
- `GET/PUT/DELETE /api/v1/stories/{id}/bible`
- `POST /api/v1/stories/{id}/bible/generate`
- `GET/PUT /api/v1/locations/{id}`
- `GET /api/v1/generations/{id}/status`
- `GET /api/v1/generations/{id}/progress` (SSE)
- `POST /api/v1/stories/generate-title`

### Experimental (`/api/v1/experimental`)
- Scene turns (interactive generation)
- Actors/character-traits/lore/casting

## Deprecated / removed
- Legacy scene CRUD (V0) — replaced by graph nodes
- `/stories/{id}/scenes` — use `/stories/{id}/nodes`
- Characters traits/assign endpoints — stubs only

## Durable generation flow

```
POST /stories/{id}/nodes/{id}/generate
  → create Generation(status=pending)
  → enqueue Job(type=generate_scene, status=pending)
  → return 202

GenerationJobWorker (goroutine):
  → poll for pending jobs
  → mark running
  → execute: generate → extract → memory → timeline → summary → validate
  → persist step results
  → mark done/failed
  → recover stuck jobs on startup
```

## Planned

| Feature | Priority |
|---------|----------|
| API types/client split by feature | ✅ Done — removed dead stubs |
