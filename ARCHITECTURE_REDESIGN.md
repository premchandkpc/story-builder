# Story Builder: Principal Engineer Architecture Redesign

> Prepared for: Production-grade, multi-tenant, AI-native platform targeting 5 years of growth.
> Current state: ~11,500 lines Go, ~60 source files, 21 interfaces, 112 methods, 16 DB tables.

---

## Table of Contents

1. [Current Architecture Analysis](#1-current-architecture-analysis)
2. [Target Modular Monolith](#2-target-modular-monolith)
3. [Bounded Context Design](#3-bounded-context-design)
4. [Event-Driven Architecture](#4-event-driven-architecture)
5. [Domain Event Model](#5-domain-event-model)
6. [Package Structure](#6-package-structure)
7. [Redis Architecture](#7-redis-architecture)
8. [Database Architecture & Scaling](#8-database-architecture--scaling)
9. [Graph Storage Design](#9-graph-storage-design)
10. [MongoDB Evaluation](#10-mongodb-evaluation)
11. [LLM Provider Architecture](#11-llm-provider-architecture)
12. [Prompt Compiler Architecture](#12-prompt-compiler-architecture)
13. [Observability](#13-observability)
14. [Reliability](#14-reliability)
15. [Security](#15-security)
16. [Kubernetes Design](#16-kubernetes-design)
17. [CI/CD Design](#17-cicd-design)
18. [Testing Strategy](#18-testing-strategy)
19. [Refactoring Roadmap](#19-refactoring-roadmap)

---

## 1. Current Architecture Analysis

### 1.1 Current High-Level Architecture

```
┌──────────┐  ┌──────────┐
│  HTTP    │  │  gRPC    │
│  :8080   │  │  :9090   │
│  (chi)   │  │ (proto)  │
└────┬─────┘  └────┬─────┘
     │              │
     ▼              ▼
┌──────────────────────────────┐
│     Handler Layer (api/)     │
│  12 handlers, 25 files       │
│  Parses HTTP/gRPC → calls    │
│  service interfaces          │
└──────────┬───────────────────┘
           │
           ▼
┌──────────────────────────────┐
│  Service Layer (api/*svc*)   │
│  In-Memory + DB-backed impls │
│  11 interfaces, ~112 methods │
│  Mixed business logic +      │
│  persistence orchestration   │
└──────────┬───────────────────┘
           │
     ┌─────┼──────┬────────┐
     ▼     ▼      ▼        ▼
┌────────┐ ┌────────┐ ┌──────────┐ ┌────────┐
│  DB    │ │ River  │ │  LLM    │ │  gRPC  │
│ (sqlc) │ │ (jobs) │ │ (router)│ │ server │
│ pgxpool│ │ 6 jobs │ │ 5 svcs  │ │ 12 svcs│
└────────┘ └────────┘ └──────────┘ └────────┘
```

### 1.2 Critical Issues Identified

#### Tight Coupling (God Services)

**`dbGenerationService.Generate()`** (`dbservices_stories.go:259-326`) —
This single method:
1. Queries the database for the node
2. Compiles context (reads characters, locations, lore from DB)
3. Computes hash via `compiler.CompiledContext`
4. Builds prompt snapshot via `compiler.CompiledContext`
5. Creates a generation record in DB
6. Enqueues a River job
7. Returns HTTP response

**Problem:** One method touches 4 domains (Graph, Canon, Compiler, Generation). Cannot test without all infrastructure.

#### Two-Phase Context Compilation

The CompiledContext is built **twice** during generation:
1. **In API service** (`dbGenerationService.compileContext`) — without `BranchSummary` or `CharState`
2. **In River worker** (`GenerateSceneWorker.compilePromptParams`) — full context

**Problem:** The `context_hash` stored in `generations` is computed from the incomplete version. Context hash doesn't match what was actually sent to the LLM. Makes `IsStale` unreliable.

#### Duplicated Interfaces

21 interfaces across 6 packages with overlapping contracts:
- `api.CharacterService` vs `canon.CharacterService` vs `db.Querier` character methods
- `api.StoryService` vs `graph.GraphService`
- `compiler.GenerationService` vs `api.GenerationService`

**Problem:** Every new feature requires changes in 3+ interface layers. High cognitive load. Violation of DRY.

#### Mixed In-Memory and DB Implementations

Each service has both in-memory and DB implementations, with runtime selection based on DB connectivity (`cmd/server/main.go:39-42`). This adds complexity:
- Two code paths that must stay in sync
- In-memory stores replicate DB logic poorly (no transactions, no constraints)
- Testing requires mocking both or choosing one

**Better approach:** Interface-based testing with testcontainers for DB-backed tests. Drop in-memory stores.

#### Direct DB Access in River Workers

River workers call `w.Queries.*` directly (`river/jobs.go:76, 97, 113, 131, etc.`). Workers should go through service interfaces, not raw sqlc queries.

**Problem:** Workers bypass business logic (validation, authorization, event emission). Schema changes break workers directly.

#### No Domain Events

Every operation is a direct synchronous call. After `AcceptGeneration`, three River jobs are enqueued synchronously with no event-driven coordination.

**Problem:** Adding a new post-accept step (e.g., "notify collaborators", "update search index") requires editing `AcceptGeneration`.

#### LLM Layer Has No Circuit Breaker

The router has retry with backoff (`llm/router.go:39`) but:
- No circuit breaker for provider failures
- No fallback between providers
- No rate limiting
- No cost tracking
- No per-provider timeout (uses client-level 120s timeout)

#### No Caching Layer

Every generation request:
1. Re-reads all characters from DB
2. Re-reads location from DB
3. Re-searchs lore from DB
4. Re-reads character states from DB
5. Re-reads summaries from DB
6. Re-compiles the full prompt

For the same story/node, most of this data is unchanged between generations.

#### Observability Gaps

- No structured logging (stdlib `log.Printf`)
- No metrics (zero Prometheus counters/histograms)
- No tracing (zero OpenTelemetry spans)
- No LLM cost tracking
- No queue depth/monitoring for River

#### Test Coverage

- Only 3 test files exist (`handlers_test.go`, `router_test.go`, `models_test.go`, `smoke_test.go`)
- Smoke test is a manual curl-based test
- No unit tests for services
- No integration tests with testcontainers (despite being in go.mod)
- No contract tests for LLM layer

### 1.3 Dependency Violation Map

```
api/ ───────────► canon/       ✓ (uses domain types)
api/ ───────────► compiler/    ✓ (uses Generation, CompiledContext)
api/ ───────────► graph/       ✓ (uses Story, Node, Edge)
api/ ───────────► db/          ✗ (API service should not depend on DB directly)
river/ ─────────► db/          ✗ (Should use service interfaces, not sqlc)
river/ ─────────► llm/         ✓ (uses service interfaces)
llm/services ───► compiler/   ✓ (uses prompt builders)
llm/services ───► ledger/     ✓ (uses types)
grcp/server ────► api/        ✓ (uses service interfaces)
cmd/main ───────► everything  ✗ (single file wires everything)
```

---

## 2. Target Modular Monolith

### 2.1 High-Level Target Architecture

```
┌──────────────────────────────────────────────────────┐
│                   API Gateway                        │
│            HTTP (chi) :8080 / gRPC :9090              │
│         Auth middleware → rate limiter → router        │
└─────────┬────────────────────────────────────────────┘
          │
          ▼
┌──────────────────────────────────────────────────────┐
│                  Bootstrap Layer                      │
│  Config │ Observability │ DB Pool │ Redis │ Queues   │
└─────────┬────────────────────────────────────────────┘
          │
          ▼
┌──────────────────────────────────────────────────────┐
│              Application Core (Domain)                │
├──────────────┬──────────────┬───────────────────────┤
│  Story BC    │  Canon BC    │   Generation BC        │
│  ─────────   │  ────────    │   ─────────────        │
│  Story       │  Character   │   Prompt               │
│  Node        │  Location    │   Generation            │
│  Edge        │  Lore        │   Draft                 │
│  Chapter     │  Actor       │   Token Usage           │
│              │  Casting     │   Provider Config       │
├──────────────┼──────────────┼───────────────────────┤
│  Validation  │  Compiler BC │   Workflow BC          │
│  BC          │  ─────────── │   ───────────           │
│  ──────────  │  Context     │   Job Orchestration     │
│  Consistency │  Prompt      │   Event Routing         │
│  Quality     │  Cache       │   Pipeline              │
│  Canon       │              │   Dead Letter           │
├──────────────┴──────────────┴───────────────────────┤
│              Shared Kernel                            │
│  UUID │ Time │ Events │ Types │ Errors │ Logger       │
└──────────────────────────────────────────────────────┘
          │
          ▼
┌──────────────────────────────────────────────────────┐
│              Infrastructure Layer                     │
├──────────┬─────────┬──────────┬─────────┬───────────┤
│ Postgres │ Redis   │ LLM      │ Object  │ River     │
│ (sqlc)   │ (cache  │ Provider │ Storage │ (queue)   │
│ pgvector │ +state) │ Adapters │ (S3)    │           │
└──────────┴─────────┴──────────┴─────────┴───────────┘
```

### 2.2 Dependency Rule Enforcement

```
Application (handlers)
    │
    ▼
Domain Interfaces (ports)
    │
    ▼
Infrastructure (adapters)

Domain Layer MUST never import:
    • sqlc, pgx (Postgres)
    • grpc, http (transport)
    • redis/go-redis (cache)
    • river (queue)
    • openai, anthropic (LLM providers)
```

---

## 3. Bounded Context Design

### 3.1 Story Bounded Context

```
Owns: Story, Node, Edge, Chapter, Scene, Branch
Ports:
    StoryRepository interface {
        Create(ctx, title) (*Story, error)
        Get(ctx, id) (*Story, error)
        List(ctx) ([]*Story, error)
        UpdateTitle(ctx, id, title) error
    }
    NodeRepository interface {
        Create(ctx, node *Node) error
        Get(ctx, id) (*Node, error)
        Update(ctx, node *Node) error
        ListByStory(ctx, storyID) ([]*Node, error)
        SetStatus(ctx, id, status) error
        SetSceneStructure(ctx, id, *SceneStructure) error
    }
    EdgeRepository interface {
        Create(ctx, edge *Edge) error
        ListByStory(ctx, storyID) ([]*Edge, error)
        GetOutgoing(ctx, nodeID) ([]*Edge, error)
        GetIncoming(ctx, nodeID) ([]*Edge, error)
    }

Events Emitted:
    StoryCreated{StoryID, Title}
    NodeCreated{StoryID, NodeID, BeatIntent}
    NodeStatusChanged{NodeID, OldStatus, NewStatus}
    EdgeCreated{StoryID, FromNode, ToNode, EdgeType}

Handlers:
    CreateStory(title) → StoryCreated
    CreateNode(storyID, intent) → NodeCreated
    UpdateNodeStatus(nodeID, status) → NodeStatusChanged
```

### 3.2 Canon Bounded Context

```
Owns: Character, Location, Lore, Actor, Casting, CharacterTrait
Ports:
    CharacterRepository interface {
        Create(ctx, *Character) (*Character, error)
        Get(ctx, id, version) (*Character, error)
        GetLatest(ctx, id) (*Character, error)
        Update(ctx, *Character) (*Character, error)
        List(ctx) ([]*Character, error)
    }
    LocationRepository interface { ... }
    LoreRepository interface {
        Create(ctx, *Lore) error
        List(ctx) ([]*Lore, error)
        SearchByTags(ctx, tags) ([]*Lore, error)
        SearchSimilar(ctx, embedding, limit) ([]*Lore, error)
    }
    CharacterTraitRepository interface { ... }
    CastingRepository interface { ... }

Events Emitted:
    CharacterCreated{CharacterID, Name}
    CharacterUpdated{CharacterID, Version}
    LocationCreated{LocationID, Name}
    LoreCreated{LoreID, Tags}
    CastingCreated{StoryID, ActorID, CharacterID}

Handlers:
    (None for now — responds to queries, emits on writes)
```

### 3.3 Compiler Bounded Context

```
Owns: CompiledContext, PromptAssembly, PromptOptimization, ContextCache
Ports:
    CompiledContextRepository interface {
        GetOrCompute(ctx, storyID, nodeID) (*CompiledContext, error)
        Invalidate(ctx, storyID) error
    }

Services:
    ContextCompiler interface {
        Compile(ctx, storyID, nodeID) (*CompiledContext, error)
        CacheKey(storyID, nodeID) string
    }
    PromptAssembler interface {
        BuildProsePrompt(ctx *CompiledContext) Prompt
        BuildStateExtractPrompt(...) Prompt
        BuildSummaryPrompt(...) Prompt
        BuildMergePrompt(...) Prompt
        BuildValidatePrompt(...) Prompt
        BuildOutlinePrompt(synopsis) Prompt
    }

Events Emitted:
    ContextCompiled{StoryID, NodeID, ContextHash}
    ContextCacheInvalidated{StoryID}

Handlers:
    On NodeStatusChanged → Invalidate context cache
    On CharacterUpdated → Invalidate context cache for affected stories
```

### 3.4 Generation Bounded Context

```
Owns: Generation, Draft, TokenUsage, ProviderConfig, LLMProvider
Ports:
    GenerationRepository interface {
        Create(ctx, *Generation) error
        Accept(ctx, genID) error
        RejectOthers(ctx, nodeID, acceptedID) error
        ListByNode(ctx, nodeID) ([]*Generation, error)
        GetAccepted(ctx, nodeID) (*Generation, error)
    }
    LLMProvider interface {
        Complete(ctx, *CompletionRequest) (*CompletionResponse, error)
        Name() string
    }
    TokenTracker interface {
        Track(ctx, provider, model, promptTokens, completionTokens) error
        GetUsage(ctx, storyID) (*TokenUsage, error)
    }

Events Emitted:
    GenerationCreated{NodeID, GenerationID, ContextHash}
    GenerationAccepted{NodeID, GenerationID}
    LLMRequestStarted{Provider, Model, PromptTokens}
    LLMRequestCompleted{Provider, Model, PromptTokens, CompletionTokens, Duration, Cost}

Handlers:
    On GenerationAccepted → Emit ExtractStateRequested
    On GenerationAccepted → Emit SummaryUpdateRequested
    On GenerationAccepted → Emit ValidationRequested
```

### 3.5 Validation Bounded Context

```
Owns: ValidationResult, ConsistencyCheck, QualityScore
Ports:
    ValidationRepository interface {
        Save(ctx, nodeID, genID, *ValidationResult) error
        GetLatest(ctx, nodeID) (*ValidationResult, error)
    }

Services:
    CanonValidator interface {
        Validate(ctx, canonXML, charState, draft) (*ValidationResult, error)
    }
    QualityChecker interface {
        Check(ctx, draft, criteria) (*QualityScore, error)
    }

Events Emitted:
    ValidationCompleted{NodeID, GenerationID, Passed, Violations}
    QualityCheckCompleted{NodeID, Score}

Handlers:
    On ExtractStateRequested → Extract states, persist
    On SummaryUpdateRequested → Update summaries, check elevation
    On ValidationRequested → Validate against canon
```

### 3.6 Workflow Bounded Context

```
Owns: Job, Pipeline, EventRouter, DeadLetterQueue
Ports:
    WorkflowRepository interface {
        CreateJob(ctx, *Job) error
        GetJob(ctx, id) (*Job, error)
        UpdateJobStatus(ctx, id, status) error
        ListPending(ctx) ([]*Job, error)
    }

Services:
    EventRouter interface {
        Publish(ctx, event DomainEvent) error
        Subscribe(eventType, handler EventHandler)
    }
    PipelineOrchestrator interface {
        StartGenerationPipeline(ctx, storyID, nodeID) error
        GetPipelineStatus(ctx, pipelineID) (*PipelineStatus, error)
    }

Events Emitted:
    PipelineStarted{PipelineID, StoryID, NodeID}
    PipelineStepCompleted{PipelineID, Step}
    PipelineFailed{PipelineID, Error}
    PipelineCompleted{PipelineID}

Handlers:
    On StoryCreated → Start outline generation (if synopsis provided)
    On NodeStatusChanged(draft→generating) → Start generation pipeline
    On GenerationAccepted → Start post-accept pipeline
```

---

## 4. Event-Driven Architecture

### 4.1 Current Synchronous Flow (Anti-Pattern)

```
HTTP POST /generate
    │
    ├──► DB Query Node, Characters, Location, Lore
    ├──► CompileContext()
    ├──► CreateGeneration()
    ├──► InsertRiverJob(GenerateScene)
    └──► Return 202

River GenerateScene Worker
    │
    ├──► DB Query (again): Characters, Location, Lore, State, Summary
    ├──► CompilePromptParams()  (Second compilation!)
    ├──► LLM.Complete() → Anthropic
    └──► DB UpdateGenerationOutput

HTTP POST /accept
    │
    ├──► DB AcceptGeneration + RejectOthers
    ├──► InsertRiverJob(ExtractState)
    ├──► InsertRiverJob(UpdateSummary)
    ├──► InsertRiverJob(ValidateScene)
    └──► Return 200
```

### 4.2 Target Event-Driven Flow

```
HTTP POST /generate ─────────► GenerationCreated{NodeID}
                                        │
                                        ▼
                              ┌─────────────────────┐
                              │  EventRouter         │
                              │  (River Queue)       │
                              └────────┬────────────┘
                                       │
                    ┌──────────────────┼──────────────────┐
                    ▼                  ▼                  ▼
            CompileContext      StartGeneration     InvalidateCache
            Requested           Requested            Requested
                    │                  │
                    ▼                  ▼
         ┌─────────────────┐  ┌──────────────────┐
         │ ContextCompiler  │  │ GenerationWorker │
         │ (Redis check)    │  │ (LLM call)       │
         └────────┬────────┘  └────────┬─────────┘
                  │                    │
                  ▼                    ▼
         ContextCompiled{Hash}  GenerationCompleted{Output}
                                       │
                                       ▼
                              ┌─────────────────────┐
                              │  EventRouter         │
                              └────────┬────────────┘
                                       │
                    ┌──────────────────┼──────────────────┐
                    ▼                  ▼                  ▼
            ExtractState        UpdateSummary       ValidateScene
            Requested            Requested            Requested
                    │                  │                  │
                    ▼                  ▼                  ▼
               StateExtracted    SummaryUpdated     ValidationCompleted
                                       │
                                       ▼
                              CheckSummaryElevation
                                       │
                          (if threshold reached)
                                       ▼
                              ElevateSummaryRequested
```

### 4.3 Event Contracts

```go
// ── Domain Events ──────────────────────────────────

type DomainEvent interface {
    EventType() string
    AggregateID() uuid.UUID
    Timestamp() time.Time
}

type StoryCreated struct {
    StoryID uuid.UUID `json:"story_id"`
    Title   string    `json:"title"`
}
func (e StoryCreated) EventType() string    { return "story.created" }
func (e StoryCreated) AggregateID() uuid.UUID { return e.StoryID }
func (e StoryCreated) Timestamp() time.Time   { return time.Now() }

type NodeCreated struct {
    StoryID    uuid.UUID `json:"story_id"`
    NodeID     uuid.UUID `json:"node_id"`
    BeatIntent string    `json:"beat_intent"`
}
func (e NodeCreated) EventType() string    { return "node.created" }
func (e NodeCreated) AggregateID() uuid.UUID { return e.NodeID }

type NodeStatusChanged struct {
    NodeID    uuid.UUID `json:"node_id"`
    OldStatus string    `json:"old_status"`
    NewStatus string    `json:"new_status"`
}
func (e NodeStatusChanged) EventType() string { return "node.status_changed" }

type GenerationRequested struct {
    NodeID       uuid.UUID `json:"node_id"`
    GenerationID uuid.UUID `json:"generation_id"`
    ContextHash  string    `json:"context_hash"`
}
func (e GenerationRequested) EventType() string { return "generation.requested" }

type GenerationCompleted struct {
    NodeID       uuid.UUID `json:"node_id"`
    GenerationID uuid.UUID `json:"generation_id"`
    Output       string    `json:"output"`
    Model        string    `json:"model"`
    TokenUsage   TokenUsage `json:"token_usage"`
}
func (e GenerationCompleted) EventType() string { return "generation.completed" }

type GenerationAccepted struct {
    NodeID       uuid.UUID `json:"node_id"`
    GenerationID uuid.UUID `json:"generation_id"`
}
func (e GenerationAccepted) EventType() string { return "generation.accepted" }

type ContextCompiled struct {
    StoryID     uuid.UUID `json:"story_id"`
    NodeID      uuid.UUID `json:"node_id"`
    ContextHash string    `json:"context_hash"`
}
func (e ContextCompiled) EventType() string { return "context.compiled" }

type ExtractStateRequested struct {
    NodeID       uuid.UUID `json:"node_id"`
    GenerationID uuid.UUID `json:"generation_id"`
    SceneText    string    `json:"scene_text"`
    CharacterRefs []uuid.UUID `json:"character_refs"`
}
func (e ExtractStateRequested) EventType() string { return "extract_state.requested" }

type ValidationRequested struct {
    NodeID       uuid.UUID `json:"node_id"`
    GenerationID uuid.UUID `json:"generation_id"`
    CompiledCanon string   `json:"compiled_canon"`
    CharState    string    `json:"char_state"`
    SceneText    string    `json:"scene_text"`
}
func (e ValidationRequested) EventType() string { return "validation.requested" }

type SummaryUpdateRequested struct {
    StoryID          uuid.UUID `json:"story_id"`
    NodeID           uuid.UUID `json:"node_id"`
    PreviousSummary  string    `json:"previous_summary"`
    AcceptedScene    string    `json:"accepted_scene"`
}
func (e SummaryUpdateRequested) EventType() string { return "summary_update.requested" }

type MergeBranchesRequested struct {
    StoryID      uuid.UUID `json:"story_id"`
    SummaryA     string    `json:"summary_a"`
    SummaryB     string    `json:"summary_b"`
    TimelineNote string    `json:"timeline_note"`
}
func (e MergeBranchesRequested) EventType() string { return "merge_branches.requested" }
```

### 4.4 Event Routing (River-based)

```go
// ── Event Bus ──────────────────────────────────────

type EventBus struct {
    riverClient *river.Client[pgx.Tx]
}

func (b *EventBus) Publish(ctx context.Context, events ...DomainEvent) error {
    for _, e := range events {
        args := RiverEvent{
            EventType:  e.EventType(),
            AggregateID: e.AggregateID(),
            Payload:    mustJSON(e),
            Timestamp:  e.Timestamp(),
        }
        _, err := b.riverClient.Insert(ctx, args, nil)
        if err != nil {
            return fmt.Errorf("publish %s: %w", e.EventType(), err)
        }
    }
    return nil
}

// ── Event Router Worker ────────────────────────────

type EventRouterWorker struct {
    river.WorkerDefaults[RiverEvent]
    Handlers map[string][]EventHandler
}

func (w *EventRouterWorker) Work(ctx context.Context, job *river.Job[RiverEvent]) error {
    handlers, ok := w.Handlers[job.Args.EventType]
    if !ok {
        return nil // no handlers for this event type
    }
    for _, h := range handlers {
        if err := h.Handle(ctx, job.Args); err != nil {
            return err // River will retry
        }
    }
    return nil
}
```

### 4.5 Event Handler Registration

```go
// ── In main.go bootstrap ───────────────────────────

func registerEventHandlers(bus *EventBus, router *EventRouterWorker) {
    // Story events
    router.Register("story.created", bus.Handle(func(ctx, event) {
        // Start story outline generation pipeline
    }))

    // Generation pipeline
    router.Register("generation.requested", bus.Handle(func(ctx, event) {
        // Call LLM, emit generation.completed
    }))
    router.Register("generation.accepted", bus.Handle(func(ctx, event) {
        // Emit extract_state.requested
        // Emit summary_update.requested
        // Emit validation.requested
    }))

    // Post-generation chain
    router.Register("extract_state.requested", bus.Handle(func(ctx, event) {
        // Call LLM for state extraction, persist
    }))
    router.Register("summary_update.requested", bus.Handle(func(ctx, event) {
        // Call LLM for summary update, persist
        // Check elevation threshold
    }))
    router.Register("validation.requested", bus.Handle(func(ctx, event) {
        // Call LLM for validation, persist
    }))
}
```

### 4.6 Queue Topology

| Queue | Workers | Purpose | Priority |
|-------|---------|---------|----------|
| `events` | 3 | Event routing (all domain events) | Critical |
| `generate` | 2 | LLM scene generation (Sonnet — expensive) | High |
| `extract` | 4 | LLM state extraction (Local — cheap) | Normal |
| `validate` | 1 | LLM validation (Haiku) | Low |
| `merge` | 2 | LLM branch merging (Haiku) | Low |
| `summarize` | 2 | LLM summary updates (Local) | Normal |
| `outline` | 1 | LLM story outlining (Sonnet) | Low |
| `dlq` | 1 | Dead letter queue processing | Low |

---

## 5. Target Package Structure

```
internal/
├── domain/
│   ├── story/
│   │   ├── models.go           # Story, Node, Edge, SceneStructure, Branch
│   │   ├── repository.go       # StoryRepository, NodeRepository, EdgeRepository interfaces
│   │   └── events.go           # StoryCreated, NodeCreated, EdgeCreated, NodeStatusChanged
│   ├── canon/
│   │   ├── models.go           # Character, Location, Lore, Actor, Casting, CharacterTrait
│   │   ├── repository.go       # CharacterRepository, LocationRepository, etc.
│   │   └── events.go           # CharacterCreated, LocationCreated, etc.
│   ├── generation/
│   │   ├── models.go           # Generation, Draft, TokenUsage, ProviderConfig
│   │   ├── repository.go       # GenerationRepository
│   │   ├── events.go           # GenerationRequested, GenerationCompleted, etc.
│   │   └── provider.go         # LLMProvider interface
│   ├── compiler/
│   │   ├── models.go           # CompiledContext, CompileInput
│   │   ├── service.go          # ContextCompiler, PromptAssembler interfaces
│   │   └── events.go           # ContextCompiled
│   ├── validation/
│   │   ├── models.go           # ValidationResult, Violation, QualityScore
│   │   └── repository.go       # ValidationRepository interface
│   ├── workflow/
│   │   ├── models.go           # Pipeline, PipelineStep, Job
│   │   └── events.go           # PipelineStarted, PipelineCompleted
│   └── ledger/
│       ├── models.go           # CharacterState, StateDelta, RelationshipChange
│       └── repository.go       # CharacterStateRepository interface
│
├── app/
│   ├── handlers/
│   │   ├── story_handler.go    # HTTP handlers for story CRUD
│   │   ├── canon_handler.go    # HTTP handlers for canon CRUD
│   │   ├── generation_handler.go
│   │   └── timeline_handler.go
│   ├── router.go              # Chi router config
│   └── middleware.go           # Auth, rate limit, logging
│
├── infrastructure/
│   ├── persistence/
│   │   ├── postgres/
│   │   │   ├── story_repository.go      # PostgresStoryRepository
│   │   │   ├── canon_repository.go      # PostgresCharacterRepository, etc.
│   │   │   ├── generation_repository.go
│   │   │   ├── validation_repository.go
│   │   │   └── workflow_repository.go
│   │   └── migrations/
│   │       └── 001_*.sql
│   ├── cache/
│   │   ├── redis_context_cache.go
│   │   ├── redis_prompt_cache.go
│   │   └── redis_workflow_state.go
│   ├── llm/
│   │   ├── provider.go                # LLMProvider interface (moved from domain)
│   │   ├── router.go                  # Provider router with circuit breaker
│   │   ├── anthropic.go               # AnthropicClient adapter
│   │   ├── ollama.go                  # OllamaClient adapter
│   │   ├── openai.go                  # OpenAI adapter (future)
│   │   ├── gemini.go                  # Gemini adapter (future)
│   │   ├── retry.go                   # Retry with exponential backoff
│   │   ├── circuit_breaker.go         # Circuit breaker
│   │   ├── rate_limiter.go            # Per-provider rate limiter
│   │   └── cost_tracker.go            # Token + cost tracking
│   ├── queue/
│   │   ├── river.go                   # River client setup
│   │   ├── event_bus.go               # Event bus (wraps River)
│   │   └── workers.go                 # All River worker implementations
│   └── grpc/
│       └── server/
│           └── services.go            # gRPC handler implementations
│
├── bootstrap/
│   ├── config.go              # Env → Config struct
│   ├── observability.go       # Logger, metrics, tracer init
│   ├── database.go            # pgxpool connection + sqlc Queries
│   ├── redis.go               # go-redis client
│   ├── cache.go               # Redis cache setup
│   ├── queues.go              # River client + workers
│   ├── llm.go                 # LLM provider factory
│   ├── events.go              # Event bus + handlers
│   ├── api.go                 # HTTP server factory
│   └── grpc.go               # gRPC server factory
│
└── shared/
    ├── errors.go              # Domain error types
    ├── types.go               # UUID, Time helpers
    ├── logger.go              # Structured logging interface
    └── events.go              # DomainEvent interface, EventBus interface
```

### 5.1 Key Principles

1. **`domain/` has zero imports outside `shared/`** — No postgres, no redis, no grpc, no http
2. **`infrastructure/` implements `domain/*/repository.go` interfaces** — Swappable implementations
3. **`app/` depends on `domain/` interfaces, not infrastructure** — Handlers use interfaces
4. **`bootstrap/` wires everything together** — The only package that imports infrastructure
5. **`shared/` is the shared kernel** — Types, errors, interfaces used across all domains

### 5.2 Module Dependency Rules

```
app/ → domain/ interfaces
app/ → shared/
domain/ → shared/ ONLY
bootstrap/ → app/ + infrastructure/ + domain/
infrastructure/ → domain/ interfaces + shared/
```

---

## 6. Redis Architecture

### 6.1 Redis as First-Class Component

Redis is **not optional** for production. It serves as:
- Context cache (primary)
- Prompt cache (cost savings)
- Workflow state store
- Rate limiter backend
- Distributed lock manager
- Session store (future multi-tenancy)

### 6.2 Key Namespace Design

```
story:{id}:context           → CompiledContext (JSON, TTL 5min)
story:{id}:context:hash      → context_hash (string, TTL 5min)
story:{id}:prompt:{hash}     → CompletionResponse (JSON, TTL 24h)
story:{id}:characters        → []CharacterCard (JSON, TTL 5min)
story:{id}:lore              → []LoreEntry (JSON, TTL 5min)

node:{id}:state              → CharacterState (JSON, TTL 5min)
node:{id}:summary            → BranchSummary (string, TTL 5min)

pipeline:{id}                → PipelineStatus (JSON, TTL 1h)
job:{id}                     → JobStatus (JSON, TTL 1h)

ratelimit:{provider}:{key}   → counter (TTL sliding window)
lock:{resource}              → distributed lock (TTL 30s)
```

### 6.3 Context Cache Design

```go
// ── Context Cache ──────────────────────────────────

type ContextCache struct {
    client *redis.Client
    ttl    time.Duration // 5 minutes
}

func (c *ContextCache) Get(ctx, storyID, nodeID) (*CompiledContext, error) {
    key := fmt.Sprintf("story:%s:context", storyID)
    data, err := c.client.Get(ctx, key).Bytes()
    if err != nil {
        return nil, fmt.Errorf("cache miss: %w", err)
    }
    var cc CompiledContext
    if err := json.Unmarshal(data, &cc); err != nil {
        return nil, err
    }
    return &cc, nil
}

func (c *ContextCache) Set(ctx, storyID string, cc *CompiledContext) error {
    key := fmt.Sprintf("story:%s:context", storyID)
    data, err := json.Marshal(cc)
    if err != nil {
        return err
    }
    return c.client.Set(ctx, key, data, c.ttl).Err()
}

func (c *ContextCache) Invalidate(ctx, storyID string) error {
    key := fmt.Sprintf("story:%s:context", storyID)
    return c.client.Del(ctx, key).Err()
}
```

**Cache invalidation triggers:**
- Character updated → Invalidate all stories referencing this character
- Lore created/updated → Invalidate all stories referencing matching tags
- Location updated → Invalidate all stories referencing this location
- Node status changed → Invalidate context for this story
- Scene accepted → Invalidate context (state/summary changed)

**TTL strategy:**
- Context cache: 5 minutes (frequently invalidated, small)
- Character cards: 5 minutes (rarely change within a generation session)
- Lore: 5 minutes (rarely changes)
- Prompt cache: 24 hours (exact prompt → exact response is deterministic)

### 6.4 Prompt Cache Design

```go
// ── Prompt Cache ───────────────────────────────────

func promptCacheKey(provider, model, system, user string) string {
    h := sha256.New()
    h.Write([]byte(provider))
    h.Write([]byte(model))
    h.Write([]byte(system))
    h.Write([]byte(user))
    return fmt.Sprintf("prompt:%x", h.Sum(nil))
}

type PromptCache struct {
    client *redis.Client
    ttl    time.Duration // 24 hours
}

func (c *PromptCache) Get(ctx, provider, model, system, user string) (*CompletionResponse, error) {
    key := promptCacheKey(provider, model, system, user)
    data, err := c.client.Get(ctx, key).Bytes()
    if err != nil {
        return nil, err
    }
    var resp CompletionResponse
    if err := json.Unmarshal(data, &resp); err != nil {
        return nil, err
    }
    return &resp, nil
}

func (c *PromptCache) Set(ctx, provider, model, system, user string, resp *CompletionResponse) error {
    key := promptCacheKey(provider, model, system, user)
    data, err := json.Marshal(resp)
    if err != nil {
        return err
    }
    return c.client.Set(ctx, key, data, c.ttl).Err()
}
```

**Cost estimation:**
- Current: Every generation costs $0.015 (Sonnet), 1M gens/month = $15,000
- With prompt cache (identical prompts from retries): 15-25% reduction = ~$2,250-$3,750/month savings
- With context cache: saves ~500ms-2s per generation on DB re-reads
- Single Redis instance (cache-only) costs ~$15/month on a cloud provider

### 6.5 Workflow State Store

```go
// ── Workflow State ─────────────────────────────────

type WorkflowState struct {
    PipelineID  uuid.UUID `json:"pipeline_id"`
    StoryID     uuid.UUID `json:"story_id"`
    NodeID      uuid.UUID `json:"node_id"`
    Status      string    `json:"status"` // running, completed, failed
    CurrentStep string    `json:"current_step"`
    Progress    float64   `json:"progress"`
    StartedAt   time.Time `json:"started_at"`
    UpdatedAt   time.Time `json:"updated_at"`
    Error       string    `json:"error,omitempty"`
}

type WorkflowStore struct {
    client *redis.Client
    ttl    time.Duration // 1 hour
}

func (s *WorkflowStore) Set(ctx, pipelineID string, state *WorkflowState) error {
    key := fmt.Sprintf("pipeline:%s", pipelineID)
    data, err := json.Marshal(state)
    if err != nil {
        return err
    }
    return s.client.Set(ctx, key, data, s.ttl).Err()
}

func (s *WorkflowStore) Get(ctx, pipelineID string) (*WorkflowState, error) {
    key := fmt.Sprintf("pipeline:%s", pipelineID)
    data, err := s.client.Get(ctx, key).Bytes()
    if err != nil {
        return nil, err
    }
    var state WorkflowState
    if err := json.Unmarshal(data, &state); err != nil {
        return nil, err
    }
    return &state, nil
}
```

**Why Redis for workflow state (not Postgres):**
- Status updates every few seconds (high write rate)
- No ACID required (eventual consistency is fine for "is it running?")
- TTL-based cleanup (completed/failed pipelines auto-expire)
- Pub/Sub for real-time status updates (WebSocket push)
- Avoids polling `generations` table for progress

### 6.6 Distributed Locks

```go
// ── Distributed Locks ──────────────────────────────

type DistLock struct {
    client *redis.Client
    ttl    time.Duration // 30 seconds
}

func (l *DistLock) Acquire(ctx, resource string) (bool, error) {
    key := fmt.Sprintf("lock:%s", resource)
    ok, err := l.client.SetNX(ctx, key, "locked", l.ttl).Result()
    return ok, err
}

func (l *DistLock) Release(ctx, resource string) error {
    key := fmt.Sprintf("lock:%s", resource)
    return l.client.Del(ctx, key).Err()
}
```

**Use cases:**
- Prevent concurrent generation for the same node (two clicks)
- Prevent concurrent context compilation for the same story
- Prevent concurrent character state updates from the same scene
- Prevent concurrent summary elevation checks

### 6.7 Rate Limiting

```go
// ── Sliding Window Rate Limiter ────────────────────

type SlidingWindowRateLimiter struct {
    client *redis.Client
    window time.Duration // 1 second
    max    int           // requests per window
}

func (rl *SlidingWindowRateLimiter) Allow(ctx, key string) (bool, error) {
    now := time.Now().UnixMilli()
    windowStart := now - rl.window.Milliseconds()
    redisKey := fmt.Sprintf("ratelimit:%s", key)

    // Remove old entries outside the window
    _, err := rl.client.ZRemRangeByScore(ctx, redisKey, "0", strconv.FormatInt(windowStart, 10)).Result()
    if err != nil {
        return false, err
    }

    // Count remaining entries in window
    count, err := rl.client.ZCard(ctx, redisKey).Result()
    if err != nil {
        return false, err
    }
    if int(count) >= rl.max {
        return false, nil
    }

    // Add current request
    _, err = rl.client.ZAdd(ctx, redisKey, redis.Z{
        Score:  float64(now),
        Member: fmt.Sprintf("%d:%d", now, rand.Int63()),
    }).Result()
    if err != nil {
        return false, err
    }

    // Set TTL on the sorted set
    rl.client.Expire(ctx, redisKey, rl.window).Err()

    return true, nil
}
```

**Rate limits per resource:**
| Resource | Limit | Window |
|----------|-------|--------|
| LLM Anthropic (total) | 50 RPM | 1 min |
| LLM Anthropic Sonnet | 10 RPM | 1 min |
| LLM Ollama (total) | 100 RPM | 1 min |
| HTTP API per tenant | 1000 RPM | 1 min |
| Generation per node | 1 RPS | 1 sec |
| Story generation | 5 RPH | 1 hour |

---

## 7. Database Architecture & Scaling

### 7.1 Current Schema Issues

1. **`characters` and `locations` versioning** uses insert-only with `(id, version)` PK. For 1M characters × 10 versions = 10M rows. The `latest_*` views use `DISTINCT ON` which requires a `Sort` on `(id, version DESC)` — no supporting index.

2. **`nodes.character_refs`** uses `uuid[]` (Postgres array). Array operations don't benefit from indexes. At scale, this becomes slow for "find all nodes character X appears in."

3. **`lore.embedding`** uses `ivfflat` with 100 lists. For 10K+ lore entries, ivfflat recall degrades. Need `hnsw` index for production.

4. **No partitioning** — All 15 tables sit in the default `public` schema. No table partitioning for time-series (generations, character_state, scene_turns).

### 7.2 Schema Recommendations

```sql
-- ── Partitioning ───────────────────────────────────

-- Partition generations by creation month
CREATE TABLE generations (
    id          UUID DEFAULT gen_random_uuid(),
    node_id     UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    context_hash TEXT NOT NULL DEFAULT '',
    prompt_snapshot TEXT NOT NULL DEFAULT '',
    output      TEXT NOT NULL DEFAULT '',
    model       TEXT NOT NULL DEFAULT '',
    accepted    BOOLEAN NOT NULL DEFAULT false,
    validation_result JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
) PARTITION BY RANGE (created_at);

CREATE TABLE generations_2026_01
    PARTITION OF generations
    FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');

CREATE TABLE generations_2026_02
    PARTITION OF generations
    FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');

-- ── Missing Indexes ────────────────────────────────

-- Support DISTINCT ON in latest_characters view
CREATE INDEX idx_characters_id_version
    ON characters(id, version DESC);

-- Support DISTINCT ON in latest_locations view
CREATE INDEX idx_locations_id_version
    ON locations(id, version DESC);

-- Support character lookup in nodes (GIN on uuid[])
CREATE INDEX idx_nodes_character_refs
    ON nodes USING GIN (character_refs);

-- Support character state lookups
CREATE INDEX idx_char_state_character
    ON character_state(character_id, as_of_node DESC);

-- Support story summary lookups
CREATE INDEX idx_summaries_story_level
    ON story_summaries(story_id, level);

-- ── HNSW for production lore search ────────────────
-- (Requires pgvector ≥ 0.5.0)
CREATE INDEX idx_lore_embedding_hnsw
    ON lore USING hnsw (embedding vector_cosine_ops);
```

### 7.3 Scaling Plan

| Tier | Users | Stories | DB Strategy | Redis | Compute |
|------|-------|---------|-------------|-------|---------|
| **Dev** | 1-10 | <100 | Single pg instance | Optional | Single server |
| **Startup** | 100-1K | <10K | Single pg + good indexes | 1 GB | 2-4 servers |
| **Growth** | 1K-10K | <100K | pg + PgBouncer + read replicas | 4 GB | 8-16 servers |
| **Scale** | 10K-100K | <1M | Partitioning + read replicas + connection pooling | 16 GB | 32-64 servers |
| **Enterprise** | 100K-1M | <10M | Sharding + CQRS + materialized views | 64 GB | 100+ servers |

**Read replica strategy:**
```
Primary:
    writes: stories, nodes, edges, characters, locations, generations, character_state
    Wal: streamed to replicas

Replica 1:
    reads: characters, locations (context compilation)
Replica 2:
    reads: lore (vector search — IVFFlat/HNSW)
Replica 3:
    reads: generations, scene_turns (history queries)
```

**Materialized views for read-heavy queries:**
```sql
-- Pre-computed latest character state per story/node
CREATE MATERIALIZED VIEW latest_char_state AS
SELECT DISTINCT ON (cs.story_id, cs.character_id)
    cs.story_id,
    cs.character_id,
    cs.as_of_node,
    cs.state,
    cs.updated_at
FROM character_state cs
ORDER BY cs.story_id, cs.character_id, cs.as_of_node DESC;

-- Pre-computed node + generation join for topology view
CREATE MATERIALIZED VIEW node_with_latest_gen AS
SELECT n.*, g.output, g.model, g.accepted
FROM nodes n
LEFT JOIN LATERAL (
    SELECT output, model, accepted
    FROM generations
    WHERE node_id = n.id AND accepted = true
    LIMIT 1
) g ON true;
```

---

## 8. Graph Storage Design

### 8.1 Current Approach vs Recommended

**Current:** Story graph (nodes + edges) stored in relational tables with `uuid[]` for character references.

**For the story DAG (sequencing, branching):** PostgreSQL is **correct**. A graph database adds operational complexity for no benefit over Postgres with adjacency lists and recursive CTEs.

**For the character relationship graph:** A dedicated graph DB (or Postgres + `ltree`) is useful but not yet critical.

### 8.2 Graph Traversal Examples (Postgres Recursive CTEs)

```sql
-- Topological sort of story DAG
WITH RECURSIVE topo AS (
    SELECT n.id, n.beat_intent, 1 AS depth
    FROM nodes n
    WHERE NOT EXISTS (
        SELECT 1 FROM edges e WHERE e.to_node = n.id AND e.story_id = '...'
    )
    UNION ALL
    SELECT n.id, n.beat_intent, t.depth + 1
    FROM nodes n
    JOIN topo t ON n.id IN (
        SELECT e.to_node FROM edges e WHERE e.from_node = t.id
    )
)
SELECT * FROM topo ORDER BY depth;

-- Find all nodes a character appears in
SELECT n.id, n.beat_intent
FROM nodes n
WHERE n.character_refs @> ARRAY[$1::uuid]  -- $1 is character_id
  AND n.story_id = $2;

-- Find the longest path (critical path) in a story
WITH RECURSIVE paths AS (
    SELECT e.from_node, e.to_node, ARRAY[e.from_node, e.to_node] AS path, 2 AS length
    FROM edges e WHERE e.story_id = '...'
    UNION ALL
    SELECT p.from_node, e.to_node, p.path || e.to_node, p.length + 1
    FROM paths p
    JOIN edges e ON e.from_node = p.to_node AND e.story_id = '...'
)
SELECT path, length
FROM paths
ORDER BY length DESC
LIMIT 1;
```

### 8.3 When to Consider a Graph Database

**Trigger conditions for Neo4j addition:**
1. Character relationship queries exceed 100ms
2. Cross-story character arcs need tracing
3. Canon consistency queries span multiple stories
4. "Find all stories involving character X and location Y" becomes common

**Hybrid approach (recommended for now):**
- Postgres: Story DAG (nodes + edges)
- Postgres + GIN index: Character-to-node mapping (`character_refs` array)
- Redis: For any path/traversal cache (pre-computed topological sort)
- Neo4j (future): Character relationship graph for cross-story analytics

---

## 9. MongoDB Evaluation

### 9.1 Current Usage Pattern

The project currently has **no MongoDB**. The question is whether to add it.

**Use cases where MongoDB is often proposed:**
1. **Prompt snapshots** — Currently stored as `generations.prompt_snapshot TEXT`
2. **LLM responses** — Currently stored as `generations.output TEXT`
3. **Version history** — Character versions (currently append-only rows)
4. **Session state** — Currently not stored

### 9.2 Analysis

| Use Case | Current Solution | MongoDB Advantage | Verdict |
|----------|-----------------|-------------------|---------|
| Prompt snapshots | TEXT column | Schema flexibility | **Stay with Postgres** — TEXT is fine, JSONB is available |
| LLM responses | TEXT column | Large doc storage | **Stay with Postgres** — TEXT supports unlimited length |
| Version history | Append-only rows | Embedded versions | **Stay with Postgres** — Relational joins needed |
| Session state | Not stored | Fast key lookups | **Use Redis** — Not a document store use case |
| Large artifacts | Not stored | GridFS | **Use S3** — GridFS is an anti-pattern |

### 9.3 Recommendation: Do NOT Add MongoDB

**Reasons:**
1. **Operational overhead** — Running two databases doubles operational complexity
2. **No transactional consistency** — Cross-document transactions in MongoDB are slow
3. **pgvector + JSONB covers all use cases** — JSONB for flexible schemas, TEXT for large content
4. **pgvector provides vector search** — No need for MongoDB Atlas vector search (yet)
5. **Cost** — One database is cheaper than two
6. **No migration path** — Current schema is relational; moving to MongoDB means full rewrite

**Use S3 for large artifacts instead:**
- Story exports (PDF, EPUB)
- Prompt archives (beyond last 100 per node)
- Generated media (images, audio in future)

---

## 10. LLM Provider Architecture

### 10.1 Provider Abstraction

```go
// ── Provider Interface (in domain/generation/) ─────

type Provider interface {
    Name() string
    Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)
    SupportsModel(model ModelTier) bool
}

type CompletionRequest struct {
    Model       ModelTier
    System      string
    UserMessage string
    Temperature float64
    MaxTokens   int
    Tools       []ToolDefinition
    ToolChoice  string
}

type CompletionResponse struct {
    Content    string
    ToolUse    map[string]interface{}
    Model      string
    Provider   string
    Usage      TokenUsage
}

type TokenUsage struct {
    PromptTokens     int
    CompletionTokens int
    TotalTokens      int
    CostUSD          float64
}
```

### 10.2 Router with Circuit Breaker

```go
// ── Provider Router (in infrastructure/llm/) ───────

type ProviderRouter struct {
    providers   []Provider          // Ordered by preference
    fallback    Provider
    breaker     *CircuitBreaker
    cache       *PromptCache       // Optional Redis prompt cache
    rateLimiter *SlidingWindowRateLimiter
    costTracker *CostTracker
}

func (r *ProviderRouter) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
    // 1. Check prompt cache (if enabled)
    if r.cache != nil {
        resp, err := r.cache.Get(ctx, req)
        if err == nil {
            return resp, nil
        }
    }

    // 2. Check circuit breaker
    if !r.breaker.IsAvailable() {
        if r.fallback != nil {
            return r.fallback.Complete(ctx, req)
        }
        return nil, ErrCircuitBreakerOpen
    }

    // 3. Try providers in order
    var lastErr error
    for _, p := range r.providers {
        if !p.SupportsModel(req.Model) {
            continue
        }

        // 4. Rate limit check
        allowed, err := r.rateLimiter.Allow(ctx, p.Name())
        if !allowed {
            continue // skip to next provider
        }

        start := time.Now()
        resp, err := r.breaker.Execute(ctx, func() (*CompletionResponse, error) {
            return p.Complete(ctx, req)
        })

        duration := time.Since(start)

        // 5. Track usage and cost
        if resp != nil {
            r.costTracker.Record(ctx, p.Name(), req.Model, resp.Usage, duration)
        }

        // 6. Cache response
        if err == nil && r.cache != nil {
            r.cache.Set(ctx, req, resp)
        }

        if err == nil {
            return resp, nil
        }

        lastErr = err
        r.breaker.RecordFailure()
    }

    return nil, fmt.Errorf("all providers failed: %w", lastErr)
}
```

### 10.3 Circuit Breaker Implementation

```go
type CircuitBreakerState int

const (
    StateClosed   CircuitBreakerState = iota
    StateHalfOpen
    StateOpen
)

type CircuitBreaker struct {
    mu              sync.RWMutex
    state           CircuitBreakerState
    failureCount    int
    failureThreshold int         // 5
    successThreshold int         // 2
    cooldownPeriod  time.Duration // 60 seconds
    lastFailure     time.Time
}

func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() (*CompletionResponse, error)) (*CompletionResponse, error) {
    cb.mu.RLock()
    if cb.state == StateOpen {
        if time.Since(cb.lastFailure) > cb.cooldownPeriod {
            cb.mu.RUnlock()
            cb.transitionTo(StateHalfOpen)
        } else {
            cb.mu.RUnlock()
            return nil, ErrCircuitBreakerOpen
        }
    } else {
        cb.mu.RUnlock()
    }

    resp, err := fn()
    if err != nil {
        cb.RecordFailure()
        return nil, err
    }

    cb.mu.Lock()
    if cb.state == StateHalfOpen {
        cb.successCount++
        if cb.successCount >= cb.successThreshold {
            cb.state = StateClosed
            cb.successCount = 0
            cb.failureCount = 0
        }
    }
    cb.mu.Unlock()

    return resp, nil
}
```

### 10.4 Cost Tracking

```go
type CostTracker struct {
    client *redis.Client
}

type CostEntry struct {
    Provider string    `json:"provider"`
    Model    string    `json:"model"`
    TokensIn int       `json:"tokens_in"`
    TokensOut int      `json:"tokens_out"`
    Cost     float64   `json:"cost_usd"`
    Duration time.Duration `json:"duration"`
    Timestamp time.Time    `json:"timestamp"`
}

func (ct *CostTracker) Record(ctx, provider, model string, usage TokenUsage, duration time.Duration) {
    entry := CostEntry{
        Provider:  provider,
        Model:     model,
        TokensIn:  usage.PromptTokens,
        TokensOut: usage.CompletionTokens,
        Cost:      usage.CostUSD,
        Duration:  duration,
        Timestamp: time.Now(),
    }
    data, _ := json.Marshal(entry)

    // Store in Redis stream
    ct.client.XAdd(ctx, &redis.XAddArgs{
        Stream: "llm:cost:log",
        Values: map[string]interface{}{"entry": string(data)},
    })

    // Increment daily counters
    day := time.Now().Format("2006-01-02")
    ct.client.HIncrByFloat(ctx, fmt.Sprintf("llm:cost:daily:%s", day), provider, usage.CostUSD)
    ct.client.HIncrBy(ctx, fmt.Sprintf("llm:tokens:daily:%s", day), provider, int64(usage.TotalTokens))
}
```

**Estimated cost table (Anthropic):**

| Model | $/M input tokens | $/M output tokens | Cost/generation (500 in, 1000 out) |
|-------|-----------------|------------------|-----------------------------------|
| Sonnet 4 | $3.00 | $15.00 | $0.0165 |
| Haiku 3.5 | $0.80 | $4.00 | $0.0044 |
| Llama 3.2 (local) | $0.00 | $0.00 | $0.00 (compute costs only) |

---

## 11. Prompt Compiler Architecture

### 11.1 Current Problems

1. **Two-phase compilation** — Context built twice, hash from incomplete version
2. **Hard-coded prompt assembly** — `BuildSceneProseSystemPrompt()` is a 60-line function building XML by hand
3. **No prompt validation** — Missing required fields produce empty prompts at runtime
4. **No token budgeting** — No truncation strategy for long context + lore
5. **No prompt optimization** — Empty sections still generate tokens (e.g., empty `<world_rules>`)

### 11.2 Target Architecture

```go
// ── Context Compiler ───────────────────────────────

type ContextCompiler struct {
    storyRepo        story.StoryRepository
    characterRepo    canon.CharacterRepository
    locationRepo     canon.LocationRepository
    loreRepo         canon.LoreRepository
    ledgerRepo       ledger.CharacterStateRepository
    summaryRepo      compiler.SummaryRepository  // NOT the same as LLM SummaryService
    cache            *ContextCache
}

func (cc *ContextCompiler) Compile(ctx, storyID, nodeID uuid.UUID) (*CompiledContext, error) {
    // 1. Check cache
    cached, err := cc.cache.Get(ctx, storyID, nodeID)
    if err == nil {
        return cached, nil
    }

    // 2. Load node to get references
    node, err := cc.storyRepo.GetNode(ctx, nodeID)
    if err != nil {
        return nil, fmt.Errorf("load node: %w", err)
    }

    // 3. Load all referenced entities in parallel
    type charResult struct { idx int; card *canon.Card }
    charCh := make(chan charResult, len(node.CharacterRefs))
    errCh := make(chan error, 1)

    var charCards []*canon.Card
    var locationCard *canon.Card
    var lore []string
    var charState map[string]ledger.CharacterState
    var branchSummary string

    var wg sync.WaitGroup

    // 3a. Load characters
    for i, ref := range node.CharacterRefs {
        wg.Add(1)
        go func(idx int, charID uuid.UUID) {
            defer wg.Done()
            c, err := cc.characterRepo.GetLatest(ctx, charID)
            if err != nil {
                errCh <- fmt.Errorf("load character %s: %w", charID, err)
                return
            }
            charCh <- charResult{idx: idx, card: c.ToCard()}
        }(i, ref)
    }

    // 3b. Load location
    if node.LocationRef != nil {
        wg.Add(1)
        go func() {
            defer wg.Done()
            loc, err := cc.locationRepo.GetLatest(ctx, *node.LocationRef)
            if err != nil {
                errCh <- fmt.Errorf("load location: %w", err)
                return
            }
            locationCard = loc.ToCard()
        }()
    }

    // 3c. Load lore (depends on character names known after load)
    // (Simplified — coordinate with goroutines above)

    // 4. Build compiled context
    ctx := &CompiledContext{
        CharacterCards: charCards,
        LocationCard:   locationCard,
        Lore:           lore,
        BranchSummary:  branchSummary,
        CharState:      charState,
        BeatIntent:     node.BeatIntent,
        POV:            node.POV,
        Tone:           node.Tone,
        TargetWords:    node.TargetWords,
    }

    // 5. Optimize (truncate to token budget)
    ctx.Optimize(tokenBudget)

    // 6. Compute hash
    ctx.Hash()

    // 7. Write-through cache
    cc.cache.Set(ctx, storyID, nodeID, ctx)

    return ctx, nil
}
```

### 11.3 Token Budget Manager

```go
type TokenBudget struct {
    MaxSystemTokens int  // 4000 for Sonnet
    MaxUserTokens   int  // 2000
    CurrentSystem   int
    CurrentUser     int
}

func (cc *CompiledContext) Optimize(budget TokenBudget) {
    // Priority order: BeatIntent > POV > CharacterCards > State > Summary > Lore

    // 1. Characters (most important for scene quality)
    remaining := budget.MaxSystemTokens - estimateTokens(cc.BeatIntent) - estimateTokens(cc.POV)
    charBlocks := make([]string, 0, len(cc.CharacterCards))
    for _, card := range cc.CharacterCards {
        block := card.ToXML()
        if estimateTokens(block) > remaining {
            card.TruncateDescription(50) // Keep name + 50 chars of description
            block = card.ToXML()
        }
        charBlocks = append(charBlocks, block)
    }

    // 2. Lore (least important — truncate aggressively)
    loreBudget := remaining / 4
    cc.Lore = truncateToBudget(cc.Lore, loreBudget)

    // 3. State (important for continuity)
    stateBudget := remaining / 4
    cc.CharState = truncateStateToBudget(cc.CharState, stateBudget)
}
```

### 11.4 Context Cache Invalidation Rules

| Event | Invalidation |
|-------|-------------|
| Character updated | All stories referencing that character |
| Lore created/updated | All stories with characters matching lore tags |
| Location updated | All stories referencing that location |
| Node status accepted | The story context (state changed) |
| Edge created | The story context (topology changed) |
| Story title changed | Story context (minimal — title rarely used in context) |

---

## 12. Observability

### 12.1 Structured Logging

```go
// ── Structured Logger (infrastructure/) ────────────

type Logger struct {
    logger *slog.Logger
}

func NewLogger(env string) *Logger {
    var handler slog.Handler
    if env == "development" {
        handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
    } else {
        handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
    }
    return &Logger{logger: slog.New(handler)}
}

func (l *Logger) Info(msg string, attrs ...slog.Attr) {
    l.logger.LogAttrs(context.Background(), slog.LevelInfo, msg, attrs...)
}
func (l *Logger) Error(msg string, err error, attrs ...slog.Attr) {
    l.logger.LogAttrs(context.Background(), slog.LevelError, msg, append(attrs, slog.Any("error", err))...)
}
```

**Log schema:**
```json
{
    "timestamp": "2026-06-12T10:30:00Z",
    "level": "info",
    "message": "generation.completed",
    "service": "story-builder",
    "provider": "anthropic",
    "model": "claude-sonnet-4-20250514",
    "story_id": "abc-123",
    "node_id": "def-456",
    "generation_id": "ghi-789",
    "tokens_in": 450,
    "tokens_out": 1200,
    "cost_usd": 0.0165,
    "duration_ms": 2350,
    "trace_id": "xyz-999"
}
```

### 12.2 Metrics (Prometheus)

```go
// ── Metrics ────────────────────────────────────────

var (
    RequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "http_request_duration_seconds",
            Help:    "HTTP request duration",
            Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
        },
        []string{"method", "path", "status"},
    )

    GenerationDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "llm_generation_duration_seconds",
            Help:    "LLM generation duration",
            Buckets: []float64{.5, 1, 2.5, 5, 10, 15, 30, 60},
        },
        []string{"provider", "model"},
    )

    TokenUsageTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "llm_tokens_total",
            Help: "Total LLM tokens used",
        },
        []string{"provider", "model", "type"}, // type = prompt/completion
    )

    LLMCost = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "llm_cost_usd_total",
            Help: "Total LLM cost in USD",
        },
        []string{"provider", "model"},
    )

    QueueDepth = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "river_queue_depth",
            Help: "River queue depth",
        },
        []string{"queue"},
    )

    QueueLatency = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "river_queue_latency_seconds",
            Help:    "Time from job insertion to start",
            Buckets: []float64{.1, .5, 1, 2.5, 5, 10, 30, 60},
        },
        []string{"queue", "job_type"},
    )

    GenerationErrors = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "generation_errors_total",
            Help: "Generation errors by type",
        },
        []string{"provider", "error_type"}, // error_type = timeout/rate_limit/content_filter/parse
    )

    CacheHitRate = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "cache_hits_total",
            Help: "Cache hits vs misses",
        },
        []string{"cache_name", "result"}, // result = hit/miss
    )

    ActiveGenerations = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "active_generations",
            Help: "Currently running LLM generations",
        },
    )

    DbQueryDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "db_query_duration_seconds",
            Help:    "Database query duration",
            Buckets: prometheus.DefBuckets,
        },
        []string{"query", "table"},
    )
)
```

**Dashboard panels (Grafana):**
1. **LLM Cost by Provider** — Area chart, daily rollup
2. **Generation Duration P50/P95/P99** — Heatmap
3. **Queue Depth × Queue** — Time series
4. **Cache Hit Rate** — Pie chart, per cache
5. **Error Rate by Type** — Stacked bar
6. **Token Usage by Model** — Stacked area
7. **Active Generations** — Gauge
8. **DB Query Performance** — Table (top 10 slowest)

### 12.3 Tracing (OpenTelemetry)

```go
// ── Trace Setup ────────────────────────────────────

func initTracer(serviceName, otlpEndpoint string) (*sdktrace.TracerProvider, error) {
    exporter, err := otlptracegrpc.New(
        context.Background(),
        otlptracegrpc.WithEndpoint(otlpEndpoint),
        otlptracegrpc.WithInsecure(),
    )
    if err != nil {
        return nil, err
    }

    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceNameKey.String(serviceName),
        )),
    )
    return tp, nil
}
```

**Trace spans:**
```
[HTTP POST /api/v1/stories/{id}/nodes/{id}/generate]
  │
  ├── [COMPILE] context_compiler.Compile (story_id, node_id)
  │     ├── [DB] story_repo.GetNode
  │     ├── [DB] character_repo.GetLatest (x N — parallel)
  │     ├── [DB] location_repo.GetLatest
  │     ├── [DB] lore_repo.SearchByTags
  │     ├── [CACHE] context_cache.Get (miss)
  │     └── [CACHE] context_cache.Set
  │
  ├── [QUEUE] generation_repo.Create
  │
  ├── [QUEUE] river_client.Insert (GenerateScene)
  │
  └── [RIVER] GenerateSceneWorker.Work
        │
        ├── [COMPILE] compilePromptParams
        ├── [LLM] router.Complete
        │     ├── [CACHE] prompt_cache.Get (miss)
        │     ├── [PROVIDER] anthropic.Complete
        │     │     ├── [HTTP] POST api.anthropic.com
        │     │     └── [METRICS] record_token_usage
        │     ├── [CACHE] prompt_cache.Set
        │     └── [CIRCUIT] breaker.RecordSuccess
        │
        └── [DB] update_generation_output
```

### 12.4 Health Check Endpoints

```
GET /healthz          → {"status": "ok"}
GET /readyz           → {"status": "ok", "db": "ok", "redis": "ok", "llm": "ok"}
GET /metrics          → Prometheus metrics
GET /debug/pprof/     → Go pprof
GET /debug/vars       → expvar
```

---

## 13. Reliability

### 13.1 Current Issues

| Issue | Location | Impact | Fix |
|-------|----------|--------|-----|
| No response body close check | `client.go:56,128` | Resource leak | `defer res.Body.Close()` is already done (correct) |
| No context in LLM clients | `client.go:43` | Blocked on dead provider | Fixed in Phase 2 with `NewRequestWithContext` |
| No circuit breaker | `router.go:19` | Retry storm => provider rate limit hit | Implement circuit breaker per-provider |
| No DLQ for failed jobs | `river/jobs.go` | Permanent job failure => scrap | Add River Dead Letter Queue |
| No idempotency | `dbservices_stories.go` | Double-click on generate => 2 generations | Add idempotency key per generation |
| Goroutine leak in context compiler | (current design) | Accumulated goroutines | Add context cancellation + WaitGroup |
| Memory leak in CompiledContext | `compiler.go` | Large prompt snapshots retained | Pool Generation objects |
| No retry for DB transient errors | All db calls | Connection blips kill request | Add retry middleware to pgxpool |

### 13.2 Idempotency Design

```go
// ── Idempotency Key ────────────────────────────────

type IdempotencyMiddleware struct {
    redis *redis.Client
    ttl   time.Duration // 24 hours
}

func (m *IdempotencyMiddleware) Handle(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        key := r.Header.Get("Idempotency-Key")
        if key == "" {
            next.ServeHTTP(w, r)
            return
        }

        // Check if already processed
        existing, err := m.redis.Get(r.Context(), "idempotent:"+key).Result()
        if err == nil {
            // Return cached response
            w.Header().Set("Content-Type", "application/json")
            w.Write([]byte(existing))
            return
        }

        // Wrap response writer to cache response
        mw := &cachedResponseWriter{ResponseWriter: w, key: key, redis: m.redis}
        next.ServeHTTP(mw, r)
    })
}
```

### 13.3 Dead Letter Queue

```go
// ── River DLQ Config ───────────────────────────────

func configureRiverDLQ(riverClient *river.Client[pgx.Tx]) {
    // River v0.39 supports DLQ via Options
    // Jobs that exceed max attempts flow to a separate "dlq" queue
    workers := river.NewWorkers()
    river.AddWorker(workers, &DLQReaperWorker{})
}

type DLQReaperWorker struct {
    river.WorkerDefaults[DLQEntry]
    Queries *db.Queries
}

func (w *DLQReaperWorker) Work(ctx context.Context, job *river.Job[DLQEntry]) error {
    // Log the failed job with full context
    log.Printf("DLJ entry: type=%s attempts=%d error=%v params=%+v",
        job.Args.OriginalType,
        job.Args.Attempts,
        job.Args.Error,
        job.Args.Payload,
    )

    // Store for manual inspection
    data, _ := json.Marshal(job.Args)
    _, err := w.Queries.InsertDLQEntry(ctx, db.InsertDLQEntryParams{
        OriginalType: job.Args.OriginalType,
        Payload:      data,
        Error:        job.Args.Error,
    })
    return err
}
```

### 13.4 Retry Policies

| Job | Max Retries | Backoff | Idempotency |
|-----|-------------|---------|-------------|
| GenerateScene | 3 | Exponential (1s, 2s, 4s) | By generation ID |
| ExtractState | 2 | Exponential (500ms, 1s) | By generation ID |
| UpdateSummary | 2 | Exponential (500ms, 1s) | By (story, node, level) |
| ValidateScene | 2 | Exponential (500ms, 1s) | By generation ID |
| MergeBranches | 3 | Exponential (1s, 2s, 4s) | By branch IDs |
| GenerateOutline | 2 | Exponential (2s, 4s) | By story ID |

---

## 14. Security

### 14.1 Current Gaps

| Issue | Impact |
|-------|--------|
| No authentication on any endpoint | Anyone can create/modify any resource |
| No authorization | No tenant isolation |
| No rate limiting | Abuse via excessive generation calls |
| No input validation at handler layer | Potential prompt injection via `synopsis`, `beat_intent`, etc. |
| API key in environment variable | Good enough for MVP, needs secrets manager for prod |
| No CORS restrictions for production | Currently allows only localhost origins |
| No request size limits | Large payloads can consume memory |
| No prompt injection guards | User-provided content goes directly to LLM prompt |

### 14.2 Authentication & Authorization

```go
// ── JWT Middleware ─────────────────────────────────

type AuthMiddleware struct {
    jwksURL      string
    allowedIssuer string
    audience     string
}

func (m *AuthMiddleware) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := extractBearerToken(r)
        if token == "" {
            writeError(w, http.StatusUnauthorized, "missing authorization header")
            return
        }

        claims, err := m.validateToken(r.Context(), token)
        if err != nil {
            writeError(w, http.StatusUnauthorized, "invalid token")
            return
        }

        // Store tenant info in context
        ctx := context.WithValue(r.Context(), tenantKey, claims.TenantID)
        ctx = context.WithValue(ctx, userKey, claims.Subject)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// ── Authorization ──────────────────────────────────

func requireStoryAccess(storyRepo story.Repository) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            storyID := chi.URLParam(r, "storyID")
            tenantID := r.Context().Value(tenantKey).(string)

            story, err := storyRepo.Get(r.Context(), uuid.MustParse(storyID))
            if err != nil || story.TenantID != tenantID {
                writeError(w, http.StatusNotFound, "not found")
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

### 14.3 Prompt Injection Protection

```go
// ── Input Sanitization ─────────────────────────────

func sanitizeForPrompt(input string) string {
    // Remove XML/HTML tags (LLMs may interpret them)
    input = stripHTML(input)

    // Remove control characters
    input = strings.Map(func(r rune) rune {
        if r < 32 && r != '\n' && r != '\t' {
            return -1
        }
        return r
    }, input)

    // Truncate to max length
    if len(input) > 10000 {
        input = input[:10000]
    }

    return input
}

// ── Prompt Injection Detection ─────────────────────

type PromptInjectionDetector struct {
    patterns []regexp.Regexp
}

func NewPromptInjectionDetector() *PromptInjectionDetector {
    return &PromptInjectionDetector{
        patterns: []regexp.Regexp{
            regexp.MustCompile(`(?i)(?:ignore|disregard|forget)\s+(?:all\s+)?(?:previous|above|prior)`),
            regexp.MustCompile(`(?i)(?:you\s+are\s+(?:now|free)|new\s+instructions|system\s+override)`),
            regexp.MustCompile(`(?i)(?:STOP|HALT|IGNORE)\s*(?::|AND)\s*(?:LISTEN|FOLLOW|DO)`),
        },
    }
}

func (d *PromptInjectionDetector) Detect(input string) bool {
    for _, p := range d.patterns {
        if p.MatchString(input) {
            return true
        }
    }
    return false
}
```

### 14.4 Tenant Isolation

```yaml
# ── Database-level tenant isolation ────────────────
# Every table has tenant_id column
# RLS policies enforce tenant isolation at DB level

CREATE TABLE stories (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  TEXT NOT NULL,
    title      TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Row-Level Security
ALTER TABLE stories ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON stories
    USING (tenant_id = current_setting('app.tenant_id'));
```

---

## 15. Kubernetes Design

### 15.1 Production Deployment Manifest

```yaml
# ── story-builder-api ──────────────────────────────

apiVersion: apps/v1
kind: Deployment
metadata:
  name: story-builder-api
  labels:
    app: story-builder
    tier: api
spec:
  replicas: 3
  strategy:
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  selector:
    matchLabels:
      app: story-builder
      tier: api
  template:
    metadata:
      labels:
        app: story-builder
        tier: api
    spec:
      containers:
      - name: api
        image: story-builder/api:latest
        ports:
        - containerPort: 8080  # HTTP
        - containerPort: 9090  # gRPC
        env:
        - name: PORT
          value: "8080"
        - name: GRPC_PORT
          value: "9090"
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: story-builder-secrets
              key: database-url
        - name: REDIS_URL
          valueFrom:
            secretKeyRef:
              name: story-builder-secrets
              key: redis-url
        - name: ANTHROPIC_API_KEY
          valueFrom:
            secretKeyRef:
              name: story-builder-secrets
              key: anthropic-api-key
        resources:
          requests:
            cpu: 500m
            memory: 512Mi
          limits:
            cpu: 2000m
            memory: 1Gi
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 15
        readinessProbe:
          httpGet:
            path: /readyz
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
---
# ── story-builder-workers ──────────────────────────

apiVersion: apps/v1
kind: Deployment
metadata:
  name: story-builder-workers
  labels:
    app: story-builder
    tier: workers
spec:
  replicas: 2
  selector:
    matchLabels:
      app: story-builder
      tier: workers
  template:
    metadata:
      labels:
        app: story-builder
        tier: workers
    spec:
      containers:
      - name: workers
        image: story-builder/api:latest  # Same binary, worker mode
        command: ["/app/server", "--worker-only"]
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: story-builder-secrets
              key: database-url
        - name: REDIS_URL
          valueFrom:
            secretKeyRef:
              name: story-builder-secrets
              key: redis-url
        - name: ANTHROPIC_API_KEY
          valueFrom:
            secretKeyRef:
              name: story-builder-secrets
              key: anthropic-api-key
        resources:
          requests:
            cpu: 1000m
            memory: 1Gi
          limits:
            cpu: 4000m
            memory: 2Gi
---
# ── Horizontal Pod Autoscaler ──────────────────────

apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: story-builder-api-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: story-builder-api
  minReplicas: 3
  maxReplicas: 20
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
---
# ── Pod Disruption Budget ──────────────────────────

apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: story-builder-api-pdb
spec:
  minAvailable: 2
  selector:
    matchLabels:
      app: story-builder
      tier: api
---
# ── Service ────────────────────────────────────────

apiVersion: v1
kind: Service
metadata:
  name: story-builder-api
spec:
  selector:
    app: story-builder
    tier: api
  ports:
  - name: http
    port: 80
    targetPort: 8080
  - name: grpc
    port: 9090
    targetPort: 9090
  type: ClusterIP
---
# ── ServiceMonitor (Prometheus Operator) ───────────

apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: story-builder-api
spec:
  selector:
    matchLabels:
      app: story-builder
      tier: api
  endpoints:
  - port: http
    path: /metrics
    interval: 15s
---
# ── Ingress ────────────────────────────────────────

apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: story-builder-api
  annotations:
    nginx.ingress.kubernetes.io/rate-limit: "100r/s"
    nginx.ingress.kubernetes.io/proxy-body-size: "10m"
spec:
  rules:
  - host: api.storybuilder.example.com
    http:
      paths:
      - path: /api/v1
        pathType: Prefix
        backend:
          service:
            name: story-builder-api
            port:
              number: 80
```

### 15.2 Scaling Strategy

| Component | Scaling Strategy | Max Replicas |
|-----------|-----------------|--------------|
| API (HTTP+gRPC) | CPU-based HPA (70% target) | 20 |
| Workers (River) | Queue depth-based HPA (custom metrics) | 10 |
| Redis | Memory-based HPA + cluster mode | 3 master + replicas |
| PostgreSQL | Vertical scaling → read replicas → sharding | N/A (StatefulSet) |

**Worker-only mode:**
```go
// main.go
if os.Getenv("WORKER_ONLY") == "true" {
    // Skip API server setup
    // Only connect DB, Redis, River workers
    app.RunWorkers()
    return
}
```

**API-only mode:**
```go
if os.Getenv("API_ONLY") == "true" {
    // Skip worker setup
    app.RunAPI()
    return
}
```

---

## 16. CI/CD

```yaml
# ── .github/workflows/ci.yaml ──────────────────────

name: CI
on:
  push:
    branches: [main]
  pull_request:

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
      with:
        go-version: '1.26'
        cache: true

    - name: golangci-lint
      uses: golangci/golangci-lint-action@v6
      with:
        version: latest
        args: --timeout=5m

    - name: Gosec
      uses: securego/gosec@master
      with:
        args: ./...

    - name: Build
      run: go build ./...

    - name: Unit Tests
      run: go test -race -coverprofile=coverage.out ./...

    - name: Upload coverage
      uses: codecov/codecov-action@v4
      with:
        file: coverage.out

  integration:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: pgvector/pgvector:pg17
        env:
          POSTGRES_DB: storybuilder_test
          POSTGRES_PASSWORD: test
        ports:
          - 5432:5432
      redis:
        image: redis:7
        ports:
          - 6379:6379
    steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
    - name: Integration Tests
      run: go test -tags=integration -count=1 ./internal/...
      env:
        DATABASE_URL: postgres://postgres:test@localhost:5432/storybuilder_test

  security:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    - name: Trivy Scan
      uses: aquasecurity/trivy-action@master
      with:
        scan-type: 'fs'
        scan-ref: '.'
        format: 'sarif'
        output: 'trivy-results.sarif'

  docker:
    needs: [build, integration, security]
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/main'
    steps:
    - uses: actions/checkout@v4
    - name: Build and Push
      uses: docker/build-push-action@v5
      with:
        push: true
        tags: ghcr.io/org/story-builder:latest
```

---

## 17. Testing Strategy

### 17.1 Testing Pyramid

```
            ╱╲
           ╱  ╲
          ╱ E2E╲          2-3 critical user journeys
         ╱──────╲
        ╱  Load  ╲        k6 or vegeta (10 target endpoints)
       ╱──────────╲
      ╱ Integration╲      Repository tests (testcontainers)
     ╱──────────────╲    Worker tests (River test helpers)
    ╱  Contract Tests╲   LLM provider tests (mock server)
   ╱──────────────────╲
  ╱    Unit Tests      ╲  80%+ coverage target
 ╱──────────────────────╲
╱                        ╲
```

### 17.2 Unit Tests

```go
// ── Story Service Tests ───────────────────────────

type mockStoryRepo struct {
    mock.Mock
    story.StoryRepository
}

func (m *mockStoryRepo) Create(ctx, title string) (*story.Story, error) {
    args := m.Called(ctx, title)
    return args.Get(0).(*story.Story), args.Error(1)
}

func TestCreateStory_ValidTitle(t *testing.T) {
    repo := new(mockStoryRepo)
    svc := story.NewService(repo)

    repo.On("Create", mock.Anything, "My Story").
        Return(&story.Story{ID: uuid.New(), Title: "My Story"}, nil)

    s, err := svc.Create(context.Background(), "My Story")
    assert.NoError(t, err)
    assert.Equal(t, "My Story", s.Title)
    repo.AssertExpectations(t)
}
```

### 17.3 Integration Tests (Repository Layer)

```go
// ── Story Repository Test ──────────────────────────

func TestPostgresStoryRepository(t *testing.T) {
    ctx := context.Background()
    pool := setupTestDB(t) // testcontainers postgres + pgvector
    defer pool.Close()

    repo := postgres.NewStoryRepository(pool)

    t.Run("create and get", func(t *testing.T) {
        s, err := repo.Create(ctx, "Test Story")
        require.NoError(t, err)
        assert.NotEqual(t, uuid.Nil, s.ID)
        assert.Equal(t, "Test Story", s.Title)

        got, err := repo.Get(ctx, s.ID)
        require.NoError(t, err)
        assert.Equal(t, s.ID, got.ID)
        assert.Equal(t, s.Title, got.Title)
    })

    t.Run("list returns all", func(t *testing.T) {
        stories, err := repo.List(ctx)
        require.NoError(t, err)
        assert.GreaterOrEqual(t, len(stories), 1)
    })
}

func setupTestDB(t *testing.T) *pgxpool.Pool {
    t.Helper()
    ctx := context.Background()

    req := testcontainers.ContainerRequest{
        Image: "pgvector/pgvector:pg17",
        Env: map[string]string{
            "POSTGRES_DB":       "test",
            "POSTGRES_PASSWORD": "test",
        },
        ExposedPorts: []string{"5432/tcp"},
    }

    pg, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: req,
        Started:          true,
    })
    require.NoError(t, err)

    port, err := pg.MappedPort(ctx, "5432")
    require.NoError(t, err)

    url := fmt.Sprintf("postgres://postgres:test@localhost:%s/test?sslmode=disable", port.Port())
    pool, err := pgxpool.New(ctx, url)
    require.NoError(t, err)

    // Run migrations
    runMigrations(ctx, pool)

    t.Cleanup(func() {
        pool.Close()
        pg.Terminate(ctx)
    })

    return pool
}
```

### 17.4 Worker Tests

```go
// ── GenerateScene Worker Test ─────────────────────

func TestGenerateSceneWorker(t *testing.T) {
    ctx := context.Background()

    // Setup test DB
    pool := setupTestDB(t)
    queries := db.New(pool)

    // Setup mock LLM service
    mockProse := new(mockProseService)
    mockProse.On("GenerateScene", mock.Anything, mock.Anything).
        Return(&llm.CompletionResponse{Content: "Scene text..."}, nil)

    // Create worker
    worker := river.NewGenerateSceneWorker(mockProse, queries)

    // Create a test job
    args := river.GenerateSceneArgs{
        StoryID: testStoryID,
        NodeID:  testNodeID,
        GenID:   testGenID,
        // ...
    }

    job := &river.Job[GenerateSceneArgs]{
        Args: args,
    }

    // Execute worker
    err := worker.Work(ctx, job)
    assert.NoError(t, err)

    // Verify output was persisted
    gen, err := queries.GetGeneration(ctx, testGenID)
    assert.NoError(t, err)
    assert.Equal(t, "Scene text...", gen.Output)
}
```

### 17.5 Performance Benchmarks

```go
// ── Context Compiler Benchmark ─────────────────────

func BenchmarkContextCompilation(b *testing.B) {
    // Setup: pre-populate DB with 10 characters, 5 lore entries, 1 location
    pool := setupTestDB(b)
    repo := postgres.NewStoryRepository(pool)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        cc, err := repo.Compile(ctx, testStoryID, testNodeID)
        if err != nil {
            b.Fatal(err)
        }
        _ = cc.Hash()
    }
}

// ~50-200ms per compilation (dominated by DB reads)
// ~2-5ms with Redis cache
```

### 17.6 Chaos Tests

```go
// ── Circuit Breaker Integration Test ───────────────

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
    cb := NewCircuitBreaker(3, 2, 100*time.Millisecond)

    // Simulate 3 failures
    for i := 0; i < 3; i++ {
        _, err := cb.Execute(ctx, func() (*CompletionResponse, error) {
            return nil, errors.New("provider error")
        })
        assert.Error(t, err)
    }

    // Circuit should be open
    _, err := cb.Execute(ctx, func() (*CompletionResponse, error) {
        return &CompletionResponse{Content: "should not reach"}, nil
    })
    assert.ErrorIs(t, err, ErrCircuitBreakerOpen)
}
```

---

## 18. Refactoring Roadmap

### Phase 1: Critical Fixes (Week 1) — DONE

| Task | Status |
|------|--------|
| Remove panics in compiler (`Hash()`) | ✅ |
| Add context propagation through all service layers | ✅ |
| Replace `uuid.MustParse` with proper error handling (43 sites) | ✅ |
| Replace `_ = json.Unmarshal` with error logging (13 sites) | ✅ |
| Add context cancellation to LLM clients | ✅ |
| Retry sleep respects context in router | ✅ |

### Phase 2: Service Decoupling (Weeks 2-3)

| Task | Effort | Depends On |
|------|--------|------------|
| Define domain boundaries + packages | 2 days | Phase 1 |
| Extract `domain/story/` package | 2 days | Domain boundaries |
| Extract `domain/canon/` package | 2 days | Domain boundaries |
| Extract `domain/generation/` package | 2 days | Domain boundaries |
| Extract `domain/compiler/` package | 1 day | Domain boundaries |
| Extract `domain/ledger/` package | 1 day | Domain boundaries |
| Extract `domain/workflow/` package | 1 day | Domain boundaries |
| Define repositories per domain | 2 days | Domain packages |
| Create `shared/` kernel (types, errors) | 1 day | Domain boundaries |
| Fix two-phase context compilation | 1 day | Domain packages |

**Risk:** Longest phase — ~10 days of pure refactoring with no user-facing features.

### Phase 3: Redis Introduction (Week 4)

| Task | Effort | Depends On |
|------|--------|------------|
| Add Redis client to bootstrap | 1 day | Phase 2 |
| Implement context cache | 2 days | Redis bootstrap |
| Wire cache invalidation events | 2 days | Context cache |
| Implement prompt cache | 2 days | Redis bootstrap |
| Implement workflow state store | 1 day | Redis bootstrap |
| Implement distributed locks | 1 day | Redis bootstrap |
| Implement rate limiter | 1 day | Redis bootstrap |

**ROI:** Context cache saves ~1-2s per generation. Prompt cache saves ~15-25% LLM costs.

### Phase 4: Observability (Week 5)

| Task | Effort | Depends On |
|------|--------|------------|
| Migrate to structured JSON logging (slog) | 1 day | Phase 2 |
| Add Prometheus metrics instrumentation | 2 days | Phase 2 |
| Add OpenTelemetry tracing | 2 days | Phase 2 |
| Set up Grafana dashboard | 1 day | Metrics |
| Set up Loki log aggregation | 0.5 day | Logging |
| Add LLM cost tracking | 1 day | Phase 3 |

### Phase 5: Reliability (Weeks 6-7)

| Task | Effort | Depends On |
|------|--------|------------|
| Implement circuit breaker for LLM providers | 2 days | Phase 3 |
| Implement fallback provider chain | 1 day | Circuit breaker |
| Add River DLQ + reaper | 1 day | Phase 3 |
| Add idempotency middleware | 1 day | Phase 3 |
| Add request/response size limits | 0.5 day | Phase 2 |
| Add retry middleware for DB transient errors | 0.5 day | Phase 2 |
| Audit goroutine management + add leak guards | 1 day | Phase 2 |

### Phase 6: Graph Architecture (Week 8)

| Task | Effort | Depends On |
|------|--------|------------|
| Add missing indexes (characters, locations, GIN) | 0.5 day | Phase 2 |
| Add materialized views for read-heavy queries | 1 day | Phase 2 |
| Implement HNSW index for lore vector search | 0.5 day | Phase 2 |
| Add recursive CTE traversal utilities | 1 day | Phase 2 |
| Pre-compute topological sort (Redis cache) | 1 day | Phase 3 |

### Phase 7: Scale Readiness (Weeks 9-10)

| Task | Effort | Depends On |
|------|--------|------------|
| Table partitioning for `generations`, `character_state` | 2 days | Phase 2 |
| Connection pooling optimization (PgBouncer config) | 1 day | Phase 2 |
| Read replica setup for context compilation | 2 days | Phase 2 |
| Database migration automation (golang-migrate) | 1 day | Phase 2 |
| Load testing (k6 scenarios) | 2 days | Phase 2-6 |
| Performance benchmarks + optimization | 2 days | All |

### Phase 8: Microservice Readiness (Weeks 11-12)

| Task | Effort | Depends On |
|------|--------|------------|
| Evaluate if modular monolith is sufficient (likely yes) | 1 day | Phase 2-7 |
| Define microservice boundaries if needed | 2 days | Evaluation |
| Extract Story API as separate binary | 2 days | Phase 2 |
| Extract Workers as separate binary | 1 day | Phase 2 |
| Add gRPC between services (if split) | 2 days | Split decision |
| Add event-driven communication between services | 2 days | Split decision |
| Staging/production deployment testing | 2 days | All |

**Recommendation:** Stay modular monolith for year 1. The coupling is worth the operational simplicity at this scale. Re-evaluate when:
- Team size exceeds 5 engineers
- Deployment frequency exceeds 10x/week
- Independent scaling needed (API vs workers)
- Different teams own different domains

---

## Summary of Key Recommendations

| # | Recommendation | Impact | Effort | ROI |
|---|---------------|--------|--------|-----|
| 1 | Domain-driven package restructure | High | 10 days | Foundation |
| 2 | Event-driven architecture (River) | High | 5 days | Loose coupling |
| 3 | Redis context + prompt cache | Medium | 5 days | $$ saved + faster |
| 4 | Structured logging + metrics | Medium | 5 days | Debuggability |
| 5 | Circuit breaker + retry | High | 3 days | Reliability |
| 6 | Fix two-phase context compilation | Medium | 1 day | Correctness |
| 7 | DB indexes + partitioning | Medium | 3 days | Performance |
| 8 | OpenTelemetry tracing | Medium | 2 days | Observability |
| 9 | Prompt cache | Medium | 2 days | Cost reduction |
| 10 | Idempotency + DLQ | Medium | 2 days | Reliability |

**Total estimated effort: ~38-40 engineering days for full transformation.**

**Production-readiness checklist for next sprint:**
- [ ] Add indexes for `latest_characters` and `latest_locations`
- [ ] Add HNSW index for lore embeddings
- [ ] Add structured logging
- [ ] Add basic Prometheus metrics
- [ ] Set up Grafana dashboard
- [ ] Add circuit breaker for LLM provider
- [ ] Add Redis context cache
- [ ] Split workers into separate deployment
- [ ] Add health/readiness endpoints
- [ ] Set up k6 load testing baseline
