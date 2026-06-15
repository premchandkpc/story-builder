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
│  └──────────────┘  │  Story/Chapter   │  │  MergeBranches    │  │
│                    │  Scene/Edge       │  │  ValidateScene    │  │
│                    │  Generation       │  │  GenerateStory    │  │
│                    │  Scene/Summary    │  └───────────────────┘  │
│                    │  Blueprint        │                         │
│                    │  Timeline/Title   │                         │
│                    └────────┬─────────┘                         │
│                             │                                    │
│                    ┌────────▼─────────┐                         │
│                    │  Service Layer   │                         │
│                    │  (internal/service)│                        │
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
│  │  canon    │ │  story   │ │ river │ │  ledgers        │      │
│  │  tables   │ │  chapter │ │ jobs  │ │  character_state│      │
│  │           │ │  scene   │ │       │ │                 │      │
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
    │   ├── handlers_characters.go ── Character, Actor, Trait, Casting, Location
    │   ├── handlers_stories.go     ── Story, Chapter, Scene, Edge, StoryGenerator, Title
    │   ├── handlers_generation.go  ── Lore, Generation, Scene (turns), Summary
    │   ├── handlers_timeline.go    ── Timeline events
    │   ├── handlers_blueprints.go  ── Story blueprints
    │   ├── router.go               ─── chi route definitions
    │   ├── middleware.go           ─── RateLimit middleware
    │   ├── mwlogger.go             ─── Structured request logging
    │   ├── request_validation.go   ─── UUID/title validation helpers
    │   ├── request_validation_test.go
    │   ├── handlers_test.go
    │   └── smoke_test.go
    │
    ├── internal/service       ─── Service implementations
    │   ├── canon/             ─── Character, Actor, Trait, Casting, Location, Lore
    │   ├── story/             ─── Story CRUD + graph traversal
    │   ├── chapter/           ─── Chapter CRUD (story→chapter→scene hierarchy)
    │   ├── node/              ─── Node CRUD (legacy, scenes preferred)
    │   ├── edge/              ─── Edge CRUD (legacy, scene_edges preferred)
    │   ├── generation/        ─── Generation + StoryGenerator
    │   ├── scene/             ─── Scene CRUD (multi-agent turns)
    │   ├── context/           ─── Context builder for LLM prompts
    │   ├── memory/            ─── Character memory storage/retrieval
    │   ├── planner/           ─── Scene/chapter planning
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
    ├── internal/character     ─── Unified character domain (Phase 4)
    │   ├── models.go          ─── Definition, State, Memory, RetrievalQuery
    │   └── service.go         ─── Service interface + InMemoryService (wraps canon/ledger/memory)
    │
    ├── internal/canon         ─── Versioned domain types (legacy, migrating to character/)
    │   ├── models.go          ─── Character, Location, Lore, Actor, Card, Casting,
    │   │                          CharacterTrait, TraitAssignment, StoryBible,
    │   │                          ValidationResult, Violation, ValidatorService, Validator
    │   └── memory.go          ─── In-memory stores
    │
    ├── internal/narrative     ─── Narrative domain models
    │   ├── models.go          ─── Blueprint, MemoryStore, StoryAggregate
    │   └── types.go           ─── CharacterArc, PlotThread, Act, Relationship
    │
    ├── internal/timeline      ─── Timeline models
    │   └── models.go          ─── Event, MemoryStore
    │
    ├── internal/ledger        ─── CharacterState per (story, char, scene)
    │   ├── models.go          ─── CharacterState, StateDelta, StateDeltas
    │   └── memory.go          ─── In-memory LedgerService
    │
    ├── internal/compiler      ─── CompiledContext + prompts
    │   ├── compiler.go        ─── CompiledContext, Hash(), Generation types
    │   ├── prompts.go         ─── System prompt builders for all 7 prompts
    │   ├── summary.go         ─── Summary prompt builders
    │   └── memory.go          ─── In-memory store
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
│   ├── queries.sql.go     ─── 70 query methods (70 + 3 extras + 3 actor_traits = 76 total)
│   ├── extras.go          ─── 3 extra query methods
│   ├── actor_traits.go    ─── 3 actor trait query methods
    │   └── helpers.go         ─── UUID conversion helpers
    │
    ├── internal/agent         ─── Agent-based processing
    │   ├── models.go
    │   └── service.go
    │
    ├── internal/authoring     ─── Authoring models
    │   └── models.go
    │
    ├── internal/context       ─── Context models
    │   └── models.go
    │
    ├── internal/cost          ─── Cost tracking
    │   ├── errors.go
    │   ├── models.go
    │   └── store.go
    │
    ├── internal/entity        ─── Entity models
    │   └── models.go
    │
    ├── internal/event         ─── Event store + bus (wired into ledger.MemoryStore)
    │   ├── models.go           ─── 20 event types incl. EvStateDeltaApplied
    │   └── store.go            ─── MemoryStore + MemoryBus
    │
    ├── internal/memory        ─── Character memory system
    │   ├── errors.go
    │   ├── models.go
    │   └── store.go
    │
    ├── internal/planner       ─── Scene/chapter planning
    │   ├── errors.go
    │   ├── models.go
    │   └── store.go
    │
    ├── internal/prompt        ─── Prompt templates
    │   ├── errors.go
    │   ├── models.go
    │   └── store.go
    │
    ├── internal/relationship  ─── Relationship tracking
    │   ├── errors.go
    │   ├── models.go
    │   └── store.go
    │
    ├── internal/revision      ─── Revision history
    │   ├── errors.go
    │   ├── models.go
    │   └── store.go
    │
    ├── internal/search        ─── Search functionality
    │   └── models.go
    │
    ├── internal/storage       ─── Storage abstraction
    │   └── factory.go
    │
    ├── internal/telemetry     ─── Telemetry/metrics
    │   └── metrics.go
    │
    ├── internal/validation    ─── 4 validators (Character/Timeline/Lore/Dialogue)
    │   ├── models.go           ─── ValidationCheck, ValidationReport
    │   └── store.go            ─── MemoryStore + ValidatorService (wired into ValidateSceneWorker)
    │
    ├── internal/workflow      ─── Workflow engine
    │   ├── errors.go
    │   ├── models.go
    │   ├── store.go
    │   └── saga.go
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
    ├── internal/adapter       ─── Adapters
    │   ├── cache/             ─── Cache adapters
    │   ├── kafka/             ─── Kafka message adapters
    │   ├── mongo/             ─── MongoDB adapters
    │   ├── qdrant/            ─── Qdrant vector DB adapters
    │   ├── redis/             ─── Redis adapters
    │   └── repository/        ─── Repository adapters
    │
    └── internal/platform      ─── Platform support (in progress)
```

## Data Flow: Scene Generation

```
User clicks "Generate" on a node
    │
    ▼
api.GenerationHandler.Generate()
    │
    │ 1. Load scene from DB
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
    │ 3. Enqueue: ExtractStateWorker → UpdateSummaryWorker → ValidateSceneWorker
    │
    ▼
river.ExtractStateWorker  →  LLM extracts state deltas (local-7b via Ollama)
    │                        Persists to character_state table
    ▼
river.UpdateSummaryWorker →  LLM updates scene summary (local-7b via Ollama)
                              Upserts story_summaries (level='scene')
    ▼
river.ValidateSceneWorker →  LLM validates draft against canon (Haiku via Anthropic)
                              Stores result in generations.validation_result
```

## LLM Router

`internal/llm/router.go` — Dispatches completion requests by model tier:

| Model Tier | Provider | Default Model |
|---|---|---|
| `claude-sonnet` | Anthropic | `claude-sonnet-4-20250514` |
| `claude-haiku` | Anthropic | `claude-haiku-3-5-20250224` |
| `local-7b` | Ollama | `llama3.2:3b` |

Both clients are always created. Router selects based on model tier. Retries: 2 attempts (1 initial + 2 retries = 3 total) with exponential backoff (250ms, 500ms).

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

## Evolution Path: Narrative OS

This project is evolving from a story graph generator into a **Narrative Operating System** — a reusable platform for books, visual novels, RPGs, movies, anime, comics, interactive stories, and 2D/3D scene generation from a single canonical model.

### Current Model (Phase 0)

```
Story → Chapter → Scene
```

### Target Model (Phase 6)

```
Universe → World → Timeline → Story → Scenario → Scene → Frame
```

Six-phase migration defined in `docs/vision.md` and `docs/adr/0002-narrative-os-direction.md`.

### Key Upgrades

| Area | Current | Future |
|---|---|---|
| Data model | Story→Chapter→Scene | Universe→World→Timeline→Story→Scenario→Scene→Frame |
| Character state | JSONB per scene | Event-sourced ledger |
| Prompts | Hardcoded in registry | Layered compiler (global→story→scene→character) |
| Relationships | JSONB maps | Neo4j graph |
| Memory | None | Qdrant vector semantic retrieval |
| Culture | None | Culture engine for region-aware rendering |
| Emotion | None | Inner/displayed/suppressed emotion engine |
| Rendering | Prose only | Multi-format (prose, VN, screenplay, comic, game) |
| Events | River job queue | Kafka event-driven architecture |
| Validation | None | 4 validators (character, timeline, lore, dialogue) |
| Databases | Postgres | Postgres + MongoDB + Neo4j + Qdrant + Redis |

See `docs/vision.md` for the complete architecture plan.

---

## Known Gaps

| Area | Issue | File / Phase |
|---|---|---|
| pgvector embeddings | `CreateLore` inserts empty vector. Embeddings are never computed. | `internal/service/canon/` |
| Multi-agent scene | `StartScene`/`NextTurn`/`FinishScene` return `fmt.Errorf("not implemented")` in DB mode. | `internal/service/scene/` |
| Blueprint + Timeline | Only memory-backed. No DB tables or sqlc queries exist. | `internal/service/blueprint/`, `internal/service/timeline/` |
| Story→Chapter→Scene | `GenerateStoryWorker` creates scenes but chapter mapping is basic (single default chapter). | `internal/river/jobs.go` |
| No Universe/World model | Stories are top-level, no multiverse or world hierarchy | Phase 1 |
| No event-sourced ledger | Character state is mutable JSONB, not event-sourced | Phase 1 |
| No prompt layering | Prompts are hardcoded in registry, no inheritance/override chain | Phase 2 |
| No character memory | Characters have no persistent memory; LLM forgets past events across scenes | Phase 2 |
| No culture engine | All output is culture-neutral, no region-aware rendering | Phase 3 |
| No emotion engine | Characters have mood (string) but no inner/displayed/suppressed emotion model | Phase 3 |
| No relationship graph | Relationships stored as JSONB maps; complex queries require full scan | Phase 4 |
| No validators | No automated continuity, timeline, lore, or dialogue validation | Phase 5 |
| Single output format | Prose only — no screenplay, VN, comic, or game format support | Phase 6 |
| No Kafka events | Services coupled via Go method calls; no event-driven decoupling | Phase 4 |

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
