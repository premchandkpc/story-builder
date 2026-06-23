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
              │
              ▼
┌──────────────────────────┐
│  Narrative Analysis      │
│  (Java Spring Boot)      │
│  localhost:8081           │
│  Readability, Sentiment, │
│  Pacing analysis per      │
│  scene                    │
│  HTTP REST → Go client    │
│  in internal/narrative/  │
└──────────────────────────┘
```

## Package Dependency Graph

```
cmd/server/
    main.go                    Orchestrator (config, mongo, server lifecycle)
    init.go                    Dependency wiring (repos, LLM, services)
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
    │   ├── agent.go           ─── AgentService (orchestrator-based scene generation)
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
    │   ├── register.go        ─── RegisterAll — wires all 10 agents
    │   ├── orchestrator.go    ─── AgentRegistry, Orchestrator (Plan, Execute, RunFinish)
    │   ├── director.go        ─── Director agent (scene planning, real LLM call w/ JSON parse)
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
    │   ├── client.go          ─── AnthropicClient + OpenCodeClient
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
 	├── internal/narrative      ─── Narrative Analysis HTTP client
 	│   ├── client.go           ─── Client (AnalyzeScene, GetSceneAnalysis)
 	│   └── models.go           ─── AnalysisRequest, SceneAnalysis, ReadabilityMetrics, etc.
 	│
 	├── internal/trace          ─── OpenTelemetry tracing
 	│   ├── trace.go            ─── Span type alias, StartSpan/End/SetAttribute/SetError wrappers
 	│   └── init.go             ─── InitFromEnv — OTLP exporter + TracerProvider setup
 	│
 	├── internal/test           ─── Test helpers
	│   └── integration/
	│
	├── internal/config        ─── Environment-based config (Port, MongoURI, RedisAddr, AnthropicKey, OpenCodeURL, LogLevel, etc.)
	│
	└── internal/log           ─── Structured logging
```

## Frontend Architecture

### Component Tree

```
main.tsx                              ← Entry: StrictMode + QueryClientProvider + RouterProvider
  └── routes.tsx                      ← Route definitions (createBrowserRouter)
       └── Layout.tsx                 ← App shell: TopBar + Sidebar + <Outlet/>
            ├── TopBar.tsx            ← Search bar + navigation (editorial masthead)
            ├── StoryListItem.tsx     ← Sidebar story entries (×N)
            ├── HomeView.tsx          ← Home page ("/"): create/generate stories
            └── StoryView.tsx         ← Story detail page ("/stories/:storyId")
                 └── StoryGraph.tsx   ← React Flow canvas + right sidebar
                      ├── SceneNode.tsx   ← Custom React Flow node (×N)
                      └── GraphPanel.tsx  ← 300px right sidebar (tabbed)
                           ├── SceneEditorPanel.tsx  ← Edit tab form
                           ├── NodeInfoPanel.tsx     ← Info tab (read-only)
                           ├── EdgeInfoPanel.tsx     ← Info tab (edge details)
                           ├── GenerationList.tsx    ← Gen tab (LLM outputs)
                           │    └── GenerationCompare.tsx  ← side-by-side diff
                           ├── TurnTimeline.tsx      ← Turns tab
                           │    └── TurnItem.tsx     ← individual turn (×N)
                           └── AgentRunPanel.tsx     ← Agents tab
                                └── AgentRunItem.tsx ← individual run (×N)
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
  index.css             Global styles (warm dark theme, animations, utility classes)
  api/
    types.ts            TypeScript interfaces + shared style objects + SceneNodeData
    client.ts           HTTP API client (fetch + timeout)
    hooks.ts            React Query hooks
  components/
    Layout.tsx          App shell: sidebar + content area
    TopBar.tsx          Top navigation bar (editorial masthead)
    HomeView.tsx        Landing page
    StoryView.tsx       Story detail wrapper
    StoryGraph.tsx      React Flow canvas + right sidebar (orchestrator)
    GraphPanel.tsx      300px right sidebar with tab routing
    SceneEditorPanel.tsx Edit form (beat intent, POV, tone, words)
    NodeInfoPanel.tsx   Read-only node info
    EdgeInfoPanel.tsx   Read-only edge details
    GenerationList.tsx  Generations list with preview/accept
    GenerationCompare.tsx Side-by-side generation diff
    SceneNode.tsx       Custom React Flow node (index card + pin)
    TurnItem.tsx        Individual agent turn (expandable I/O)
    TurnTimeline.tsx    Wrapper mapping turns → TurnItem
    AgentRunItem.tsx    Individual agent run (expandable I/O)
    AgentRunPanel.tsx   Wrapper mapping runs → AgentRunItem
    StatCard.tsx        Shared metric card
    LlmMetricsDashboard.tsx  Token/cost metrics
    CriticScoreDashboard.tsx Critic evaluation scores
    AuditDashboard.tsx  Code audit findings
    CompressionStats.tsx Token compression display
    Toast.tsx           Toast notification system
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
- `SceneNodeData` → React Flow node data type (id, label, beatIntent, pov, tone, status, wordCount, targetWords)

Plus shared style objects: `inputStyle`, `btnStyle`, `spinnerStyle`, `labelStyle`, `cardStyle`, `badgeStyle`, `ghostBtnStyle`, `destructiveBtnStyle`.

## Agent Orchestration

See `docs/agents.md` for full agent architecture. Implemented:

- 10 agents registered via `RegisterAll()` in `internal/agents/register.go`
- `internal/agents/orchestrator.go` — Plan, Execute, RunFinish
- `internal/service/agent.go` — AgentService builds context, runs orchestrator, persists results
- Director agent calls `LLMClient.Complete()` with JSON parsing (others stub until P1)
- Turn order determined by `scene.FlowType` (monologue/dialogue/round_robin/action/silent)
- `GenerationService.Generate()` routes via `agentSvc.IsAgentScene(scene)`:
  - `SceneStructure` set or non-custom `FlowType` → AgentService (no job, no worker pool)
  - Otherwise → enqueue job → 6-worker pipeline
- Repos: `SceneTurnRepository`, `ActorRepository`, `CanonDeltaRepository` in `internal/repository/mongo/`
- API: `GET /turns`, `GET /turns/role`, `GET /deltas`, `POST /deltas` implemented in `internal/api/agents.go`

## Data Flow: Scene Generation

```
User clicks "Generate" on a node
    │
    ▼
api.GenerationHandler.Generate()
    │
    │ 1. Load scene from Mongo
    │
    ▼
service.GenerationService.Generate()
    │
    ├── IsAgentScene(scene)?
    │   (scene.SceneStructure != nil || flowType != custom)
    │   │
    │   ├── YES → Agent Path:
    │   │   AgentService.GenerateScene()
    │   │     → AgentService.BuildContext() — loads same context as worker path
    │   │     → orchestrator.Plan(scene) — returns TurnOrder by FlowType
    │   │     → orchestrator.Execute(plan, ctx) — runs each agent, records turns
    │   │       └── Director (LLM) → Character(s) → Narrator → Editor → CanonGuard
    │   │     → orchestrator.RunFinish(scene, ctx) — StateExtract + Critic + Director
    │   │     → persists turns, generation, canon deltas to Mongo
    │   │     → returns immediately
    │   │
    │   └── NO → Worker Path (async):
    │       Creates Generation doc + Job, returns immediately
    │
    ▼
service.GenerationJobWorker (goroutine, polls for jobs)
    │  Polls for pending jobs, marks running, then runs pipeline
    │
    ▼
service.generation_job_worker.runPipeline()
    │  ┌── service.ContextBuilder.Build()
    │  │   → Bible + character states + locations (hierarchical)
    │  │   → Memories (top-K per character) + timeline
    │  │   → Summaries + blueprint/arcs
    │  │   → Produces ~20k token BuiltContext
    │  └────────────────────────────────────────────
    │
    │ 1. generate (critical — retries 3×)
    │    → ProseService.GenerateScene → Anthropic/OpenCode
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
| `local-7b` | OpenCode | Extraction, summarization, outline, title |

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

Turn ordering is determined by `agents.Orchestrator.Plan()` in `internal/agents/orchestrator.go`:

| FlowType | Turn Sequence |
|---|---|
| `monologue` | Director → Character(1) → Narrator → Editor → CanonGuard |
| `dialogue` | Director → Character(1) → Character(2) → Narrator → Editor → CanonGuard |
| `round_robin` | Director → (Character × N → Narrator → CanonGuard) × maxTurns |
| `parallel` | Director → Character(all) → Narrator → CanonGuard |
| `action` | Director → Character → Narrator → Editor |
| `silent` | Director → Narrator |

`internal/scene/turn.go` also retains a `WhoActsNext` helper for use within individual character turns during execution.

## Canon Versioning

- Characters are **immutable definitions** (never change after creation; updates create new versioned documents).
- Character state is **append-only** (event-sourced per scene).
- Character memories are **append-only** (with vector embeddings for retrieval).
- `CompiledContext.Hash()` = SHA256 of the full context → staleness detection, stored as `gen.ContextHash`.
- Generation acceptance uses `scene.acceptedGenerationId` as source of truth (not per-generation `accepted` flags).

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

**Phase 3 — Agent Orchestration (implemented):**
- 10 runtime narrative agents registered in `internal/agents/` (Director, Narrator, CanonGuard, Editor, Critic, StateExtract, World, Arc, Memory) — all fully wired with real LLM calls
- **Per-character autonomous agents** — each character gets its own `AgentSpec` (registered under `charId`), persistent in-memory `CharacterAgentState` (emotion, goals, thoughts, plan), autonomous proposal system (`QueryProposals`), and event-driven goroutine loop (`CharacterManager`)
- Director agent calls `LLMClient.Complete()` with structured JSON parsing
- `AgentService` in `internal/service/agent.go` — orchestrator-based scene generation with context assembly
- Pipeline hybrid: agent path for structured scenes, worker path for simple scenes
- Turn/Lore/Canon repository interfaces + MongoDB implementations
- Turn-level agent runs with role-specific system prompts
- API endpoints for turns, deltas, and agent-run queries

**Phase 4 — Narrative Intelligence:**
- Sequential generation: whole acts, not individual scenes
- Semantic memory recall (embedding-based top-K per character)
- Branch-aware summary merging
- Cross-story canon + character migration

## OpenTelemetry Tracing

`internal/trace/` wraps OpenTelemetry for distributed tracing. All 10 agents + the LLM router create spans:

| Span | Created by | Attributes |
|------|------------|------------|
| `orchestrator.Plan` | `orchestrator.go` | sceneId, flowType |
| `orchestrator.Execute` | `orchestrator.go` | sceneId, maxTurns |
| `turn.<agent>.<phase>` | `orchestrator.go` | agentType, phase, turnNumber, turnId |
| `finish.<agent>.<phase>` | `orchestrator.go` | agentType, phase |
| `llm.Complete` | `router.go` | model, system_len, user_len |
| `agent.director.<phase>` | `director.go` | sceneId, flowType, turnCount |
| `agent.character.<charID>.<phase>` | `character_agent.go` | charId, charName, directive |
| `agent.narrator.<phase>` | `narrator.go` | sceneId |
| `agent.editor.<phase>` | `editor.go` | — |
| `agent.canon_guard.<phase>` | `canon_guard.go` | — |
| `agent.critic.<phase>` | `critic.go` | — |
| `agent.state_extractor.<phase>` | `state_extractor.go` | — |
| `agent.world.<phase>` | `world.go` | — |
| `agent.arc.<phase>` | `arc.go` | — |
| `agent.memory.<phase>` | `memory_agent.go` | — |

Set `OTEL_EXPORTER_OTLP_ENDPOINT` env var to enable OTLP HTTP export (e.g. `http://localhost:4318/v1/traces`). When unset, the no-op tracer is used and no spans are exported. Service name is `story-builder`.

```go
// Startup — cmd/server/main.go
otelShutdown := trace.InitFromEnv(context.Background())
defer otelShutdown(context.Background())
```

Spans are created with `trace.StartSpan(ctx, "operation.name")` and ended with `defer trace.End(span)`. Errors are recorded via `trace.SetError(span, err)`. All agent Runners, the orchestrator, and the LLM router are instrumented.

**Infrastructure philosophy:**
- MongoDB + Redis only (no Kafka, no Qdrant, no Postgres)
- Workers run in-process as goroutines (no message queue)
- New data layers proven out before any infra addition
- The moat is narrative intelligence, not database count
