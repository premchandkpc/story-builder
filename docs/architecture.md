# Architecture

## System Overview

```
┌──────────────────────────┐
│      React Flow UI       │
│  localhost:5173          │
│  Proxies /api/* → :8080  │
└────────────┬─────────────┘
             │ HTTP
             ▼
┌──────────────────────────┐
│     Go Server (chi)      │
│  localhost:8080           │
│                          │
│  ┌────────────────────┐  │
│  │    Middleware       │  │
│  │  - Logger          │  │
│  │  - Recoverer       │  │
│  │  - RequestID       │  │
│  │  - CORS            │  │
│  │  - RateLimit (Redis)│  │
│  │  - contextTimeout  │  │
│  │    (5min generate) │  │
│  └────────────────────┘  │
│                          │
│  ┌────────────────────┐  │
│  │  API Handlers      │  │
│  │  ────────────────  │  │
│  │  Story             │  │
│  │  Scene             │  │
│  │  Edge              │  │
│  │  Character         │  │
│  │  Generation        │  │
│  │  Timeline          │  │
│  │  Memory            │  │
│  │  Summary           │  │
│  └────────┬───────────┘  │
│           │               │
│  ┌────────▼───────────┐  │
│  │  Service Layer     │  │
│  │  (internal/service) │  │
│  └────────┬───────────┘  │
│           │               │
│  ┌────────▼───────────┐  │
│  │  Repository Layer  │  │
│  │  (interfaces)      │  │
│  └────────┬───────────┘  │
└───────────┼──────────────┘
            │
     ┌──────┴──────┐
     ▼             ▼
┌──────────┐ ┌──────────┐
│ MongoDB  │ │  Redis   │
│ SSOT     │ │ cache    │
│          │ │ rate lim │
│          │ │ locks    │
└──────────┘ └──────────┘
```

## Package Dependency Graph

```
cmd/server/main.go
    │
    ├── internal/api           ─── HTTP handlers + middleware
    │   ├── server.go          ─── chi route definitions + middleware
    │   ├── handlers.go        ─── Handler struct, service interfaces, writeError
    │   ├── stories.go         ─── Story CRUD, generate, generate-title, blueprint
    │   ├── nodes.go           ─── Node CRUD (V2 graph nodes)
    │   ├── edges.go           ─── Edge CRUD (V2 graph edges + legacy scene edges)
    │   ├── characters.go     ─── Character CRUD (V2 top-level + story-based)
    │   ├── scenes.go          ─── Legacy scene CRUD
    │   ├── locations.go       ─── Location CRUD
    │   ├── timeline.go        ─── Timeline events
    │   └── generations.go     ─── Generate prose, list/accept generations
    │
    ├── internal/domain        ─── Domain models (no infra deps)
    │   ├── story/
    │   ├── scene/
    │   ├── character/
    │   ├── location/
    │   ├── memory/
    │   └── timeline/
    │
    ├── internal/service       ─── Business logic
    │   ├── location.go        ─── Location CRUD + GetByName
    │   ├── generation/        ─── Generation pipeline orchestration
    │   ├── validation/        ─── Canon/timeline/character validation
    │   └── graph/             ─── DAG traversal + validation
    │
    ├── internal/repository    ─── Data access interfaces
    │   └── mongo/             ─── MongoDB implementations
    │
    ├── internal/worker        ─── In-process async workers
    │   ├── generate.go        ─── GenerateSceneWorker
    │   ├── extract.go         ─── ExtractStateWorker
    │   ├── memory.go          ─── MemoryUpdateWorker
    │   ├── timeline.go        ─── TimelineWorker
    │   ├── summary.go         ─── SummaryWorker
    │   └── validate.go        ─── ValidationWorker
    │
    ├── internal/llm           ─── LLM clients + router
    │   ├── types.go           ─── ModelTier, service interfaces
    │   ├── router.go          ─── Dispatches by model tier, JSON validation, retry with backoff
    │   ├── client.go          ─── AnthropicClient + OllamaClient
    │   ├── circuitbreaker.go  ─── CircuitBreakerClient wrapper
    │   └── services.go        ─── Prose, Extract, Summary, Merge, Validation, Outline, Title
    │
    ├── internal/graph         ─── DAG data model + traversal
    │   ├── models.go          ─── DAG types
    │   └── traversal.go       ─── TopologicalSort, FindBranches, FindDeadEnds
    │
    ├── internal/cache         ─── Redis cache + rate limiter
    │   ├── prompt_cache.go
    │   ├── rate_limiter.go
    │   └── dist_lock.go
    │
    ├── internal/prompt        ─── Prompt compiler (10-layer hierarchy)
    │
    ├── internal/config        ─── Environment-based config (Port, MongoURI, RedisAddr, AnthropicKey, OllamaURL, LogLevel, etc.)
    │
    └── internal/log           ─── Structured logging
```

## Frontend Architecture

### Component Tree

```
main.tsx                          ← Entry: StrictMode + QueryClientProvider + RouterProvider
  └── routes.tsx                  ← Route definitions (createBrowserRouter)
       └── Layout.tsx             ← App shell: TopBar + Sidebar + <Outlet/>
            ├── TopBar.tsx        ← Search bar + navigation
            ├── StoryListItem.tsx ← Sidebar story entries (×N)
            ├── HomeView.tsx      ← Home page ("/"): create/generate stories
            └── StoryView.tsx     ← Story detail page ("/stories/:storyId")
                 └── StoryGraph.tsx  ← React Flow canvas + side panel
                      └── SceneNode.tsx  ← Custom React Flow node (×N)
```

### Data Flow (Frontend)

```
React Component
    ↓ useQuery / useMutation
api/hooks.ts  (TanStack React Query)
    ↓
api/client.ts  (fetch() wrapper with timeout/error handling)
    ↓  HTTP /api/v1/*
Go API Server (chi)
```

### Key Libraries

| Library | Purpose |
|---|---|
| `react` 19 | UI framework |
| `@xyflow/react` 12 | DAG graph canvas (React Flow) |
| `@tanstack/react-query` 5 | Data fetching, caching, mutations |
| `react-router-dom` 7 | Client-side routing |
| `vite` 8 | Build tool + dev server |

### Frontend File Map

```
web/src/
  main.tsx              React entry point
  routes.tsx            Route tree
  index.css             Global styles
  api/
    types.ts            TypeScript interfaces (mirrors backend domain)
    client.ts           HTTP API client (fetch + timeout)
    hooks.ts            React Query hooks
  components/
    Layout.tsx          App shell: sidebar + content area
    TopBar.tsx          Top navigation bar
    HomeView.tsx        Landing page
    StoryView.tsx       Story detail wrapper
    StoryGraph.tsx      React Flow canvas + side panel
    SceneNode.tsx       Custom React Flow node
    StoryListItem.tsx   Sidebar story entry
```

### Frontend Type Hierarchy

The frontend types in `web/src/api/types.ts` mirror the backend domain models. Key types:

- `Story` → top-level entity, DAG root
- `GraphNode` / `GraphEdge` → DAG elements rendered by React Flow
- `Scene` / `SceneEdge` → legacy chapter-based model
- `Generation` → LLM output record
- `Topology` → full DAG snapshot (nodes + edges + topological order)
- `NodeStatus` → `draft | generated | accepted | stale`
- `EdgeType` → `seq | fork | join | choice`
- `SceneStructure` / `SceneTurn` → interactive turn-based generation

## Data Flow: Scene Generation

```
User clicks "Generate" on a node
    │
    ▼
api.GenerationHandler.Generate()
    │
    │ 1. Load scene from Mongo
    │ 2. Compile context (characters, location, memories, state)
    │ 3. Check PromptCache (Redis)
    │ 4. Spawn GenerateSceneWorker
    │
    ▼
worker.GenerateSceneWorker
    │
    │ 1. Call ProseService.GenerateScene(params)
    │    → Router routes to Anthropic (claude-sonnet) or Ollama (local-7b)
    │ 2. Store generation output in Mongo
    │ 3. Pipeline continues:
    │
    ▼
worker.ExtractStateWorker  →  LLM extracts state deltas from scene text
    │                         (local-7b via Ollama)
    ▼
worker.MemoryUpdateWorker  →  Create memories from state changes
    ▼
worker.TimelineWorker      →  Update timeline events
    ▼
worker.SummaryWorker       →  Update scene summary
    ▼
worker.ValidationWorker    →  Validate draft against canon (Haiku via Anthropic)
```

Each worker runs in its own goroutine. Workers communicate through MongoDB (writes are visible to subsequent stages).

## LLM Router

`internal/llm/router.go` — Dispatches completion requests by model tier:

| Model Tier | Provider | Use |
|---|---|---|
| `claude-sonnet` | Anthropic | High-quality prose generation |
| `claude-haiku` | Anthropic | Fast validation |
| `local-7b` | Ollama | Extraction, summarization, outline |

Retries: 1 initial + 2 retries = 3 attempts total, exponential backoff + jitter.
- Anthropic: 1s base, 15s max, 2× (±25% jitter)
- Local: 200ms base, 5s max, 2× (±25% jitter)

Circuit breaker: 5 consecutive failures → open 30s → half-open probe.

JSON output validation: services that expect JSON set `ValidateJSON` on the request; router validates before returning, retries on invalid.

## Redis Cache Layer

Optional Redis (enabled via `REDIS_ADDR`). Never a source of truth.

| Component | Purpose |
|---|---|
| `PromptCache` | Caches LLM responses for identical prompts (TTL: 1h) |
| `SlidingWindowRateLimiter` | Rate limits LLM API calls per provider |
| `DistLock` | Scene generation lock (prevents duplicate generation) |

When Redis is unavailable, all features degrade gracefully (no caching, no rate limiting).

## DAG Traversal

| Algorithm | Purpose |
|---|---|
| `TopologicalSort` | Kahn's algorithm. Orders nodes for linear execution. Returns error on cycle. |
| `FindBranches` | Walks from fork/choice nodes to join nodes. |
| `FindMergePoints` | Identifies nodes where branches converge. |
| `FindDeadEnds` | Finds scenes with no outgoing edges (potential plot holes). |
| `FindUnreachableScenes` | Finds scenes not reachable from root. |
| `ValidateDAG` | Comprehensive graph integrity check (cycles, unreachable, dead ends). |

## Scene Turn Scheduling

`internal/scene/turn.go` — `WhoActsNext` determines which actor speaks next:

| FlowType | Behavior |
|---|---|
| `monologue` | First character speaks once |
| `dialogue` | Alternating round-robin through character order |
| `round_robin` | Same as dialogue |
| `parallel` | All characters act simultaneously |
| `custom` | Round-robin starting after last speaker |

## Canon Versioning

- Characters are **immutable definitions** (never change after creation).
- Character state is **append-only** (event-sourced per scene).
- Character memories are **append-only** (with vector embeddings for retrieval).
- `CompiledContext.Hash()` = SHA256 of the full context → staleness detection.

## Evolution Path

This project evolves from DAG-based story generation toward richer narrative intelligence — better character memory, better validation, better story planning. Infrastructure stays minimal:

**Phase 1 — Current:**
```
MongoDB + Redis → Go API → React Flow
```

**Future additions only when measured bottlenecks prove them:**
- MongoDB replica sets for HA
- Sharding for scale
- (No Kafka, no Qdrant, no Postgres unless forced)

No infrastructure is added before it's needed. The moat is story intelligence, not database count.
