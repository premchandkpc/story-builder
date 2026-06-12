# Architecture

## System Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        Browser (React Flow)                     │
│  localhost:5173                                                  │
│  Proxies /api/* → localhost:8080                                 │
└───────────────────────────┬─────────────────────────────────────┘
                            │ HTTP
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Go Server (chi)                            │
│  localhost:8080                                                  │
│                                                                  │
│  ┌──────────────┐  ┌──────────────────┐  ┌───────────────────┐  │
│  │  Middleware   │  │  API Handlers    │  │  River Workers    │  │
│  │  - Logger     │  │  (14 handlers)   │  │  (6 job types)   │  │
│  │  - Recoverer  │  │  ─────────────── │  │  ───────────────  │  │
│  │  - RequestID  │  │  Character/Actor │  │  GenerateScene    │  │
│  │  - CORS       │  │  Location/Trait  │  │  ExtractState     │  │
│  │  - RateLimit  │  │  Lore/Casting    │  │  UpdateSummary    │  │
│  └──────────────┘  │  Story/Node/Edge │  │  MergeBranches    │  │
│                    │  Generation      │  │  ValidateScene    │  │
│                    │  Scene/Summary   │  │  GenerateStory    │  │
│                    │  Blueprint       │  └───────────────────┘  │
│                    │  Timeline/Title  │                         │
│                    └────────┬─────────┘                         │
│                             │                                    │
│                    ┌────────▼─────────┐                         │
│                    │  Service Layer   │                         │
│                    │  (internal/svc)  │                         │
│                    └────────┬─────────┘                         │
│                             │ DB or Memory                      │
│                    ┌────────▼─────────┐                         │
│                    │ Redis Cache      │                         │
│                    │ (prompt cache,   │                         │
│                    │  rate limiter,   │                         │
│                    │  dist lock)      │                         │
│                    └──────────────────┘                         │
└─────────────────────────────┼───────────────────────────────────┘
                              │ pgxpool
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    PostgreSQL + pgvector (5432)                  │
│                                                                  │
│  ┌───────────┐ ┌──────────┐ ┌───────┐ ┌─────────────────┐      │
│  │  canon    │ │  graph   │ │ river │ │  ledgers        │      │
│  │  tables   │ │  tables  │ │ jobs  │ │  character_state│      │
│  └───────────┘ └──────────┘ └───────┘ └─────────────────┘      │
│                                                                  │
│  Extensions: pgcrypto, vector                                    │
└─────────────────────────────────────────────────────────────────┘
```

## Package Dependency Graph

```
cmd/server/main.go
    │
    ├── internal/api           ─── HTTP handlers + middleware
    │   ├── handlers_*.go      ─── 14 handler groups
    │   ├── router.go          ─── chi route definitions
    │   ├── middleware.go      ─── RateLimit middleware
    │   └── mwlogger.go        ─── Structured request logging
    │
    ├── internal/service       ─── Service implementations
    │   ├── canon/             ─── Character, Actor, Trait, Casting, Location, Lore
    │   ├── story/             ─── Story CRUD
    │   ├── node/              ─── Node CRUD
    │   ├── edge/              ─── Edge CRUD
    │   ├── generation/        ─── Generation + StoryGenerator
    │   ├── scene/             ─── Scene CRUD (multi-agent)
    │   ├── summary/           ─── Summary CRUD
    │   ├── blueprint/         ─── Story blueprint (memory-only)
    │   ├── timeline/          ─── Timeline events (memory-only)
    │   └── cache/             ─── Redis cache wrapper + rate limiter
    │
    ├── internal/graph         ─── DAG data model + traversal
    │   ├── models.go          ─── Story, Node, Edge, SceneStructure
    │   ├── traversal.go       ─── TopologicalSort, IdentifyBranches
    │   └── memory.go          ─── In-memory GraphService
    │
    ├── internal/canon         ─── Versioned domain types
    │   ├── models.go          ─── Character, Location, Lore, Card
    │   └── memory.go          ─── In-memory stores
    │
    ├── internal/narrative     ─── Narrative domain models
    │   ├── models.go          ─── Blueprint, MemoryStore, StoryAggregate
    │   └── types.go           ─── CharacterArc, PlotThread, Act, Relationship
    │
    ├── internal/timeline      ─── Timeline models
    │   └── models.go          ─── Event, MemoryStore
    │
    ├── internal/ledger        ─── CharacterState per (story, char, node)
    │   ├── models.go          ─── CharacterState, StateDelta, StateDeltas
    │   └── memory.go          ─── In-memory LedgerService
    │
    ├── internal/compiler      ─── CompiledContext + prompts
    │   ├── compiler.go        ─── CompiledContext, Hash(), Generation types
    │   └── prompts.go         ─── System prompt builders for all 7 prompts
    │
    ├── internal/llm           ─── LLM clients + router + services
    │   ├── types.go           ─── ModelTier, 7 service interfaces, PromptRegistry
    │   ├── router.go          ─── Router: dispatches by model tier
    │   ├── client.go          ─── AnthropicClient + OllamaClient
    │   ├── services.go        ─── 7 service implementations (Prose, Extract, etc.)
    │   └── router_test.go     ─── Router tests
    │
    ├── internal/scene         ─── Multi-agent scene system
    │   ├── types.go           ─── SceneTurn, SceneService, AgentPromptInput
    │   ├── turn.go            ─── WhoActsNext — turn scheduling
    │   └── agent.go           ─── BuildAgentPrompt
    │
    ├── internal/cache         ─── Redis cache primitives
    │   ├── cache.go           ─── RedisClient interface + key prefixes
    │   ├── cached_llm.go      ─── CachedLLMClient wrapper
    │   ├── context_cache.go   ─── ContextCache for generation staleness
    │   ├── prompt_cache.go    ─── PromptCache
    │   ├── rate_limiter.go    ─── SlidingWindowRateLimiter
    │   ├── dist_lock.go       ─── Distributed lock
    │   ├── redis_client.go    ─── GoRedis client adapter
    │   └── workflow.go        ─── Workflow cache
    │
    ├── internal/river         ─── River job types + workers
    │   └── jobs.go            ─── 6 job types + workers
    │
    ├── internal/db            ─── sqlc-generated query layer
    │   ├── db.go              ─── DBTX + Queries
    │   ├── models.go          ─── Generated Go structs
    │   ├── queries.sql.go     ─── 38 query methods
    │   └── helpers.go         ─── UUID conversion helpers
    │
    ├── internal/config        ─── Environment-based config
    │   └── config.go          ─── Config struct, FromEnv()
    │
    ├── internal/log           ─── Structured logging
    │   └── log.go             ─── Config, Init, Err, Duration, WithContext
    │
    ├── internal/migrate       ─── SQL migration runner
    │   └── runner.go          ─── _migrations table, apply/pending
    │
    ├── internal/grpc          ─── gRPC server
    │   └── server/services.go ─── Wraps service interfaces with pb
    │
    └── internal/adapter       ─── Adapters
        ├── cache/             ─── Cache adapters
        └── repository/        ─── Repository adapters
```

## Data Flow: Scene Generation

```
User clicks "Generate" on a node
    │
    ▼
api.GenerationHandler.Generate()
    │
    │ 1. Load node from DB
    │ 2. Compile context (characters, location, lore, state)
    │ 3. Compute CompiledContext.Hash() = SHA256
    │ 4. Check ContextCache (Redis) for cached result
    │ 5. Create generation row (accepted=false)
    │ 6. Enqueue GenerateSceneWorker
    │
    ▼
river.GenerateSceneWorker.Work()
    │
    │ 1. Re-compile prompt params
    │ 2. Call ProseService.GenerateScene(params)
    │    → Router routes to Anthropic (claude-sonnet) or Ollama (local-7b)
    │ 3. Update generation output
    │
    ▼
api.GenerationHandler.AcceptGeneration()
    │
    │ 1. Mark generation accepted
    │ 2. Reject other generations for this node
    │ 3. Enqueue: ExtractStateWorker → UpdateSummaryWorker → MergeBranchesWorker
    │
    ▼
river.ExtractStateWorker  →  LLM extracts state deltas (local-7b via Ollama)
    │
    ▼
river.UpdateSummaryWorker →  LLM updates scene summary (local-7b via Ollama)
    │
    ▼
river.MergeBranchesWorker →  LLM merges branch summaries (claude-haiku via Anthropic)
```

## LLM Router

`internal/llm/router.go` — Dispatches completion requests by model tier:

| Model Tier | Provider | Default Model |
|---|---|---|
| `claude-sonnet` | Anthropic | `claude-sonnet-4-20250514` |
| `claude-haiku` | Anthropic | `claude-haiku-3-5-20250224` |
| `local-7b` | Ollama | `llama3.2:3b` |

Both clients are always created. Router selects based on model tier. Retries: 2 attempts with exponential backoff (250ms, 500ms).

## Redis Cache Layer

Optional Redis (enabled via `REDIS_ADDR` env var). Provides:

| Component | Purpose |
|---|---|
| `PromptCache` | Caches LLM responses for identical prompts (TTL: 1h) |
| `ContextCache` | Caches CompiledContext hash/result for generation staleness |
| `SlidingWindowRateLimiter` | Rate limits LLM API calls per provider |
| `DistLock` | Distributed lock for River job coordination |

When Redis is unavailable, all features degrade gracefully (no caching, no rate limiting).

## Canon Versioning

- Characters and Locations are append-only.
- Each update inserts a new row with `version = MAX(version) + 1`.
- `characters` PK = `(id, version)`, `locations` PK = `(id, version)`.
- `Story.CanonPins` maps entity types to specific `{id, version}` tuples.
- `CompiledContext.Hash()` = SHA256 of the full context → staleness detection.
- Views `latest_characters` and `latest_locations` query the max version per ID.

## DAG Traversal

| Algorithm | Location | Purpose |
|---|---|---|
| `TopologicalSort` | `graph/traversal.go:10` | Kahn's algorithm. Orders nodes for linear execution. Returns error on cycle. |
| `Predecessors` | `graph/traversal.go:66` | Finds all immediate parent nodes of a given node. |
| `IdentifyBranches` | `graph/traversal.go:99` | Walks from fork/choice nodes to join nodes, grouping into Branch structs. |
| `BranchCharacterSets` | `graph/traversal.go:169` | For each fork branch, deduplicates character references. |
| `ForkJoinEdges` | `graph/traversal.go:83` | Filters edges to only fork/join types. |
| `walkToJoin` | `graph/traversal.go:136` | BFS from a branch start to the next join node. |

## Scene Turn Scheduling

`scene/turn.go:8` — `WhoActsNext` determines which actor(s) speak next based on `FlowType`:

| FlowType | Behavior |
|---|---|
| `monologue` | First character in order speaks once |
| `dialogue` | Alternating round-robin through character order |
| `round_robin` | Same as dialogue |
| `parallel` | All characters act simultaneously |
| `custom` | Round-robin starting after last speaker |

## River Queue Configuration

| Queue | Max Workers | Job Types |
|---|---|---|
| `generate` | 2 | GenerateSceneWorker |
| `extract` | 4 | ExtractStateWorker |
| `merge` | 2 | MergeBranchesWorker |
| `validate` | 1 | ValidateSceneWorker |
| `default` | 1 | GenerateStoryWorker, UpdateSummaryWorker |

River's own migration system runs on startup via `rivermigrate`.

## Dual Mode: DB vs In-Memory

When Postgres is unavailable:
- All services fall back to in-memory stores (`graph.NewMemoryStore()`)
- River workers are not started
- LLM calls still work (Router always creates Anthropic + Ollama clients)
- Blueprint + Timeline always use memory stores (no DB backing yet)
- Redis cache is optional; when available wraps LLM client with caching + rate limiting

## Known Gaps

| Area | Issue | File |
|---|---|---|
| pgvector embeddings | `CreateLore` inserts empty vector. Embeddings are never computed. | `internal/service/canon/` |
| Multi-agent scene | `StartScene`/`NextTurn`/`FinishScene` return `fmt.Errorf("not implemented")` in DB mode. | `internal/service/scene/` |
| ExtractState persistence | `ExtractStateWorker` stores result in `StateDelta` but does not persist to `character_state`. | `internal/river/jobs.go` |
| ValidateScene persistence | `ValidateSceneWorker` runs validation but result is not stored. | `internal/river/jobs.go` |
| Blueprint + Timeline | Only memory-backed. No DB tables or sqlc queries exist. | `internal/service/blueprint/`, `internal/service/timeline/` |
| Story generation story_id | `GenerateStoryWorker` returns empty `StoryID`. | `internal/river/jobs.go` |

## gRPC

Protobuf service definitions live in `proto/storybuilder/v1/`:
- `common.proto` — Shared types (UUID, Timestamp, Empty)
- `canon.proto` — Character, Actor, Location, Lore, Trait, Casting
- `graph.proto` — Story, Node, Edge (+ enums for NodeStatus, EdgeType, FlowType)
- `generation.proto` — Generation service
- `scene.proto` — Scene service
- `summary.proto` — Summary service
- `storygen.proto` — StoryGenerator service

gRPC server implementation: `internal/grpc/server/services.go`. Listens on `GRPC_PORT` (default `9090`).

Config: `internal/config/config.go` — `FromEnv()` reads environment variables.
