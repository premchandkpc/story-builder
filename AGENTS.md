# Story Builder

## Auto-loaded skills
On every chat start, load ALL available skills: agents-sdk, caveman*, compare-models, customize-opencode, Debug Issue, Explore Codebase, find-skills, frontend-design, Refactor Safely, Review Changes, skill-creator.

Full-stack story graph editor with DAG-based plot structure and LLM-generated prose.

MongoDB + Redis backend. Go (chi) API server. React Flow frontend. No Postgres, Kafka, Qdrant, or River.

## Quick start

```bash
docker compose up -d mongo redis       # minimal infra (Ollama runs locally on host)
docker compose up -d --build             # everything
```

## Architecture

```
React Flow → Go API (chi) → Service Layer → Repository (interfaces)
                                               ├── MongoDB (SSOT)
                                               └── Redis (cache, rate limit, locks)
```

## Key types

| Domain      | Package             | Description |
|-------------|---------------------|-------------|
| Story       | `domain/story`      | DAG root, title, metadata |
| Scene       | `domain/scene`      | DAG node, generated content, participants |
| SceneEdge   | `domain/scene`      | Directed edge, type seq/fork/join/choice |
| Character   | `domain/character`  | Immutable definition, personality, goals |
| CharacterState | `domain/character` | Append-only state per (char, scene) |
| CharacterMemory | `domain/memory`  | Embedding-backed semantic memory |
| TimelineEvent | `domain/timeline` | Ordered story events |
| Generation  | `domain/scene`      | LLM output + validation result |

## Docs discipline

Every code change that affects structure, interfaces, types, or packages MUST update relevant `.md` files in `docs/`:
- `docs/architecture.md` — package tree, deps, flow
- `docs/schema.md` — MongoDB collections, indexes
- `docs/services.md` — service interfaces, workers
- `docs/llm.md` — prompt templates, compiler, model tiers
- `docs/api.md` — HTTP endpoints
- `docs/rules.md` — code conventions

## Repositories

All data access goes through interfaces in `internal/repository/`. MongoDB implementations in `internal/repository/mongo/`. Business logic never touches Mongo directly.

## Workers

Pipeline workers in `internal/worker/` run as goroutines (no River, no message queue):

1. `GenerateSceneWorker` — claude-sonnet prose generation
2. `ExtractStateWorker` — local-7b state delta extraction
3. `MemoryUpdateWorker` — create memories with embeddings
4. `TimelineWorker` — record timeline events
5. `SummaryWorker` — update summaries
6. `ValidationWorker` — claude-haiku canon validation

## DAG engine

`internal/graph/` — TopologicalSort, ValidateDAG, FindBranches, FindDeadEnds, FindUnreachableScenes. Cycle detection blocks generation.

## LLM pipeline

| Step | Model | Temp |
|------|-------|------|
| GenerateScene | claude-sonnet | 0.8 |
| ExtractState | local-7b (Ollama) | 0.0 |
| SummaryUpdate | local-7b | 0.2 |
| ValidateCanon | claude-haiku | 0.0 |
| OutlineStory | local-7b | 0.7 |

## headroom-ai

Context compression for all LLM calls. Go backend wraps LLMClient with `CompressClient` (sends messages to headroom proxy before forwarding to provider). Set `HEADROOM_BASE_URL` to enable. Frontend also uses `headroom-ai` for compression stats in GenerationCompare UI.

```bash
# Start headroom proxy separately, then set:
export HEADROOM_BASE_URL=http://localhost:8787
```

## Building

```bash
go build ./...
```

## Tests

Run the server and curl endpoints to verify. Test priority: graph (cycle detection, topological sort), memory (state changes, retrieval), generation (pipeline), validation (4 validators).
