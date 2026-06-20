# Architecture

## Key Documents

- `docs/agents.md` — Runtime agent architecture (10 agents, orchestration flow, model routing)
- `docs/visual/architecture-overview.md` — Mermaid system diagrams (context, deps, sequences)
- `docs/visual/scene-orchestration.md` — Turn ordering, flow type maps, state machines
- `docs/roadmap.md` — Phased implementation plan (P0→P5)

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
    │   ├── generations.go     ─── Generate prose, list/accept generations
    │   ├── bible.go           ─── Story Bible get/generate
    │   ├── chapters.go        ─── Chapter CRUD
    │   ├── progress.go        ─── SSE progress hub
    │   ├── summaries.go       ─── Summary retrieval
    │   ├── memories.go        ─── Memory search + list
    │   └── helpers.go         ─── writeJSON, param helpers
    │
    ├── internal/domain        ─── Domain models (no infra deps)
    │   ├── story.go           ─── Story entity
    │   ├── scene.go           ─── Scene + SceneEdge + Generation
    │   ├── character.go       ─── Character + CharacterState
    │   ├── location.go        ─── Hierarchical Location (dimension→room)
    │   ├── bible.go           ─── StoryBible (world, rules, magic, factions)
    │   ├── chapter.go         ─── Chapter (Act→Chapter→Scene)
    │   ├── blueprint.go       ─── StoryBlueprint (acts, arcs, threads)
    │   ├── memory.go          ─── CharacterMemory
    │   ├── timeline.go        ─── TimelineEvent
    │   ├── summary.go         ─── Summary
    │   ├── relationship.go    ─── Relationship + RelationshipDelta
    │   ├── scene_turn.go      ─── SceneTurn (turn in agent generation)
    │   ├── agent_run.go       ─── AgentRun (agent execution log)
    │   └── canon_delta.go     ─── CanonDelta (append-only canon changes)
    │
    ├── internal/service       ─── Business logic
    │   ├── story.go           ─── Story CRUD, cascade delete, Scene/Edge/Character/Timeline/Summary/Memory services
    │   ├── location.go        ─── Location CRUD + GetByName
    │   ├── generation.go      ─── Durable pipeline orchestration (context.Background, partial success)
    │   ├── bible.go           ─── Bible generation + storage
    │   ├── chapter.go         ─── Chapter CRUD
    │   └── context.go         ─── ContextBuilder — assembles Bible + states + memories + timeline → 20k prompt
    │
    ├── internal/repository    ─── Data access interfaces
    │   └── mongo/             ─── MongoDB implementations
    │
    ├── internal/agents        ─── Runtime narrative agents
    │   ├── types.go           ─── Agent, AgentSpec, AgentContext, OrchestrationPlan
    │   ├── orchestrator.go    ─── AgentRegistry, Orchestrator (Plan, Execute, RunFinish)
    │   ├── director.go        ─── Director agent (scene planning, turn orchestration)
    │   ├── character_agent.go ─── Character agent (in-character dialogue/action)
    │   ├── narrator.go        ─── Narrator agent (prose stitching)
    │   ├── editor.go          ─── Editor agent (polish, trim, pace)
    │   ├── canon_guard.go     ─── CanonGuard agent (continuity validation)
    │   ├── critic.go          ─── Critic agent (scene scoring)
    │   ├── state_extractor.go ─── StateExtract agent (state delta extraction)
    │   ├── world.go           ─── World agent (faction/lore consistency)
    │   ├── arc.go             ─── Arc agent (plot thread/character arc tracking)
    │   ├── memory_agent.go    ─── Memory agent (layered memory management)
    │   └── agent_repository.go─── SceneTurnRepository interface (shared with scene/)
    │
    ├── internal/scene         ─── Scene turn orchestration
    │   └── turn.go            ─── TurnRepository interfaces + TurnOrchestrator
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
    │   ├── services.go        ─── Prose, Extract, Summary, Merge, Validation, Outline, Title
    │   ├── bible.go            ─── BibleGenerationService
    │   └── context.go          ─── CompiledContext helpers (BuildCanonXML, BuildCharStateXML, Hash)
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
    ├── internal/events        ─── Event bus (domain events, AgentTurnCompleted, SceneTurnsComplete)
    │   ├── events.go           ─── Bus interface + in-memory implementation
    │   └── types.go            ─── Event struct + type constants
    │
    ├── internal/validation     ─── Canon validators
    │   └── validate.go         ─── ValidateAgainstCanon
    │
    ├── internal/test           ─── Test helpers
    │   └── integration/
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

## Agent Orchestration

See `docs/agents.md` for full agent architecture. Summary:

- 10 runtime agents: Director, Character, Narrator, Editor, CanonGuard, Critic, StateExtract, World, Arc, Memory
- `internal/agents/orchestrator.go` — Plan, Execute, RunFinish
- Turn order determined by `scene.FlowType` (monologue/dialogue/round_robin/action/silent)
- Agents called in sequence, each producing a `domain.SceneTurn`
- P0 agents (Director, Character, Narrator, CanonGuard, StateExtract) built first
- Integration: scenes with `sceneStructure` → agent orchestrator; simple scenes → existing pipeline

## Data Flow: Scene Generation

```
User clicks "Generate" on a node
    │
    ▼
api.GenerationHandler.Generate()
    │
    │ 1. Load scene from Mongo
    │ 2. Create Generation doc (status=running)
    │ 3. Spawn goroutine with context.Background() — survives request
    │
    ▼
service.GenerationService.runPipeline()
    │  ┌── service.ContextBuilder.Build()
    │  │   → Bible + character states + locations (hierarchical)
    │  │   → Memories (top-K per character) + timeline
    │  │   → Summaries + blueprint/arcs
    │  │   → Produces ~20k token BuiltContext
    │  └────────────────────────────────────────────
    │
    │ 1. generate (critical — retries 3×)
    │    → ProseService.GenerateScene → Anthropic/Ollama
    │    → Stores output in Mongo
    │
    │ 2. extract state (critical — retries 3×)
    │    → ExtractStateWorker → local-7b
    │
    │ 3. create memories (best-effort)
    │    → MemoryUpdateWorker → MongoDB
    │
    │ 4. record timeline (best-effort)
    │    → TimelineWorker → MongoDB
    │
    │ 5. update summary (best-effort)
    │    → SummaryWorker → local-7b
    │
    │ 6. validate canon (best-effort)
    │    → ValidationWorker → claude-haiku
    │
    ▼
Generation Status:
  - success       → all steps complete
  - partial_success → generate + extract passed, non-critical failed
  - failed        → generate failed after retries
```

Non-critical steps never fail the pipeline. Status is queryable via GET /api/v1/generations/{id}/status.
Pipeline uses context.Background() so it survives HTTP request timeout or client disconnect.

## LLM Router

`internal/llm/router.go` — Dispatches completion requests by model tier:

| Model Tier | Provider | Use |
|---|---|---|
| `claude-sonnet` | Anthropic | High-quality prose generation, Bible generation |
| `claude-haiku` | Anthropic | Fast validation |
| `local-7b` | Ollama | Extraction, summarization, outline, title |

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

This project has evolved from DAG-based scene generation to a persistent narrative simulation engine. Current architecture (Phase 2):

**Phase 2 — Narrative Simulation:**
```
MongoDB + Redis → Go API (chi) → React Flow
  Layers:
    1. Story Bible — world rules, dimensions, magic, factions, cultures (generated once, ~50k tokens)
    2. Location Graph — hierarchical dimension→planet→country→city→building→room
    3. Character State Engine — append-only state per scene, knowledge/health/mood/inventory
    4. Scene Planner — Act→Chapter→Scene hierarchy, guided progression
    5. Context Builder — assembles Bible + states + memories + timeline → ~20k prompt
    6. Durable Pipeline — context.Background(), partial success, retries, status tracking
```

**Phase 3 — Agent Orchestration:**
- 10 runtime narrative agents (see `docs/agents.md`)
- Director-led turn orchestration replacing hardcoded pipeline steps
- Agent context assembly (bible + state + memory + canon)
- Turn-level LLM calls with role-specific system prompts
- Critic-scored scene quality feedback loop

**Phase 4 — Narrative Intelligence:**
- Sequential generation: whole acts, not individual scenes
- Semantic memory recall (embedding-based top-K per character)
- Branch-aware summary merging
- Cross-story canon + character migration

**Infrastructure philosophy:**
- MongoDB + Redis only (no Kafka, no Qdrant, no Postgres)
- Workers run in-process as goroutines (no message queue)
- New data layers proven out before any infra addition
- The moat is narrative intelligence, not database count
