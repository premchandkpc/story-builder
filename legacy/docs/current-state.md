# Current State

## Active today

| Component | Status | Notes |
|-----------|--------|-------|
| **Storage** | MongoDB (primary SSOT) | All domain aggregates stored in Mongo |
| **Cache** | Redis (optional) | Rate limiting, in-memory fallback |
| **LLM** | claude-sonnet, claude-haiku, local-7b (OpenCode) | Provider/router abstraction |
| **Graph editor** | Nodes + edges CRUD via REST | V2 graph node model is primary |
| **Generation** | Enqueued as durable Job documents | Worker goroutine runs pipeline |
| **Destkop** | Scene memory, timeline, summaries | Generated pipeline output |
| **Validation** | Post-generation canon validation | 4 validators, claude-haiku |
| **Rate limiting** | Sliding window, method+path scoped keys | Configurable via `cache/rate_limiter.go` |
| **Character versioning** | Append-only version log | CharID + version for immutable history |
| **Character agents** | Per-character autonomous goroutines with persistent state + event loop | `internal/agents/character_agent.go` |
| **Prompt context builder** | Assembles context from characters, bible, memory, timeline, summaries | `internal/service/context.go` |
| **Scene locking** | Optional cross-process generation lock | `internal/repository/mongo/scene_locks.go` |
| **Projections** | Event-replay character view + timeline view cache | `internal/projection/` |
| **Run inspector** | 5-tab UI (Overview/Prompt/Timeline/Events/Cost) with lazy data loading | `web/src/components/RunInspector.tsx` |
| **Blueprint** | Structural plans (acts, arcs, character threads) | `internal/domain/blueprint.go` |
| **Planner / Diff** | Scene planning + generation comparison services | `internal/service/planner.go`, `diff.go` |
| **Planner** | Scene structural analysis (purpose, beats, participant intents) | `internal/service/planner.go` |
| **Diff** | Generation comparison (prose, events, tokens) | `internal/service/diff.go` |
| **Narrative events** | Append-only state mutation log | `internal/service/narrative_event.go` |
| **Run tracking** | Durable orchestration (runs, steps, cost, stats) | `internal/service/run.go` |

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
- `GET /api/v1/stories/{id}/nodes/{id}/plan` (scene plan)
- `GET /api/v1/stories/{id}/nodes/{id}/generations/{genID}/diff?against=` (gen diff)
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
- `GET /api/v1/runs/{id}` / `GET /api/v1/runs/{id}/steps`
- `GET /api/v1/runs/{id}/prompt-sections` / `GET /api/v1/runs/{id}/events` / `GET /api/v1/runs/{id}/cost`
- `POST /api/v1/runs/{id}/cancel`
- `GET /api/v1/stories/{id}/runs` / `GET /api/v1/stories/{id}/runs/stats`
- `GET /api/v1/stories/{id}/narrative-events`
- `GET /api/v1/experimental/stories/{id}/nodes/{id}/narrative-events`

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
