# Current Architecture (source of truth)

> **Last updated:** June 2026
> **This is the single source of truth. If other docs conflict, this wins.**

## Runtime

| Layer          | Technology            | Status     |
| -------------- | --------------------- | ---------- |
| HTTP server    | Go 1.26, chi v5       | Production |
| Primary DB     | MongoDB               | Production |
| Cache/ratelimit| Redis (optional)      | Optional   |
| LLM providers  | Anthropic Claude, Ollama | Production |
| Background jobs| In-process goroutines | **Known gap – see below** |
| Event bus      | In-memory             | Development |

## What is NOT in use today

- **Postgres** — not connected, not migrated. Any README/doc references are roadmap only.
- **pgvector** — not available.
- **River** — not integrated. Job execution is in-process goroutines.
- **Kafka** — not used.
- **Qdrant** — not used.

## Data model

MongoDB is the SSOT (single source of truth). Collections:

| Collection    | Purpose                        | Status     |
| ------------- | ------------------------------ | ---------- |
| `stories`     | Top-level story entity         | Stable     |
| `scenes`      | DAG nodes (V2 graph unit)      | Stable     |
| `scene_edges` | Directed edges (seq/fork/join/choice) | Stable |
| `generations` | LLM generation attempts        | Beta       |
| `characters`  | Immutable character definitions| Stable     |
| `character_states` | Append-only state per (char, scene) | Beta |
| `memories`    | Semantic memory (embedding-backed) | Beta |
| `timeline`    | Ordered story events           | Beta       |
| `summaries`   | Multi-level story summaries    | Beta       |
| `bibles`      | Generated story bible docs     | Beta       |
| `chapters`    | Legacy chapter grouping        | Deprecated |
| `locations`   | Story locations                | Stable     |

## Background execution

Generation pipeline runs **in-process goroutines** inside the API server process:

```
POST /generate
  → create generation doc (Mongo)
  → spawn goroutine
  → run pipeline (generate → extract → memory → timeline → summary → validate)
  → return 202 Accepted
```

**Known risks:**
- Process restart kills in-flight generations
- No retry durability (survives only in `sync.Map` in-memory)
- No lease/ownership for multi-replica deployments
- No stuck-job recovery on startup

## LLM pipeline

| Step       | Model tier     | Critical |
| ---------- | -------------- | -------- |
| Generate   | claude-sonnet  | Yes      |
| Extract    | local-7b       | No       |
| Memory     | local-7b       | No       |
| Timeline   | N/A (structural) | No     |
| Summary    | local-7b       | No       |
| Validate   | claude-haiku   | No       |

## V1 vs V2

- **V2 (graph-first)** is the canonical model. Nodes + edges + topology.
- **V1 (chapters/scenes)** is still present but deprecated. No new features should target it.
- The API router exposes both. V1 routes will be moved under `/experimental`.

## API surface

See `docs/api-status.md` for a complete endpoint status matrix.

## Graph model

- `Scene` (domain) = V2 node. Used interchangeably with "graph node" in code.
- `SceneEdge` = directed edge between two nodes.
- `topology` endpoint returns nodes + edges + topological sort.
- **Node positions are NOT persisted** (UI-only). This is a known gap.
- **Edge IDs are NOT used by the frontend** — edge identity is derived from array index. This is a known gap.

## Missing production features

- Authentication / authorization
- Tenant isolation
- Request-level ownership checks
- LLM action quotas
- Structured per-generation telemetry
- Integration tests
- CI pipeline
