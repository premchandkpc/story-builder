# Story Builder — Production Architecture Redesign

> Principal Engineer review prepared for 5-year growth horizon.
> Current codebase: ~11,500 hand-written Go lines, 60+ files, 21 service interfaces, 6 River workers, dual HTTP+gRPC surface.

---

## Table of Contents

1. [Current Architecture](#1-current-architecture)
2. [Critical Issues](#2-critical-issues)
3. [Target Architecture](#3-target-architecture)
4. [Domain Decomposition](#4-domain-decomposition)
5. [Ports & Adapters Layer](#5-ports--adapters-layer)
6. [Event Model](#6-event-model)
7. [Redis Architecture](#7-redis-architecture)
8. [Database Architecture](#8-database-architecture)
9. [Graph Storage Strategy](#9-graph-storage-strategy)
10. [MongoDB Evaluation](#10-mongodb-evaluation)
11. [LLM Architecture](#11-llm-architecture)
12. [Prompt Compiler Architecture](#12-prompt-compiler-architecture)
13. [Observability](#13-observability)
14. [Kubernetes Design](#14-kubernetes-design)
15. [Security Design](#15-security-design)
16. [Reliability Patterns](#16-reliability-patterns)
17. [Testing Strategy](#17-testing-strategy)
18. [CI/CD Pipeline](#18-cicd-pipeline)
19. [Performance Review](#19-performance-review)
20. [Refactoring Roadmap](#20-refactoring-roadmap)

---

## 1. Current Architecture

### High-Level Structure

```
HTTP (chi) ──→ API Handlers ──→ Service Interfaces ──→ MemoryStore / DB Services
                                       │
gRPC ──→ gRPC Handlers ────────────────┤
                                       │
                                  River Workers ←── LLM Services ←── Router ←── Anthropic/Ollama
                                       │
                                  DB (sqlc queries)
```

### Current Package Graph

```
cmd/server
  │
  ├── internal/api          ← HTTP handlers, service interfaces, impls (DB + memory)
  ├── internal/llm          ← LLM client, router, 6 service impls
  ├── internal/river        ← 6 River workers
  ├── internal/db           ← sqlc-generated queries
  │
  ├── internal/canon        ← Domain models (Character, Location, Lore, Actor, etc.)
  ├── internal/graph        ← DAG domain (Story, Node, Edge, traversal)
  ├── internal/compiler     ← CompiledContext, prompt builders
  ├── internal/ledger       ← Character state ledger
  ├── internal/scene        ← Multi-agent scene turn logic
  ├── internal/narrative    ← Blueprint types
  ├── internal/timeline     ← Timeline events
  ├── internal/migrate      ← DB migrations
  ├── internal/grpc/server  ← gRPC service implementations
  └── internal/grpc/gen     ← Protobuf generated code
```

### Dependency Violations (Hexagonal Architecture)

| Violation | Source | Target | Problem |
|-----------|--------|--------|---------|
| API layer imports domain packages directly | `internal/api/` | `canon`, `graph`, `compiler` | Service layer mixes domain & app concerns |
| Domain packages define their own service interfaces | `canon/models.go` | defines 6 interfaces | Duplicated interfaces (canon + api) with different signatures |
| River workers depend on `db.Queries` directly | `internal/river/jobs.go` | `internal/db/` | Workers bypass domain layer, couples to SQL |
| API service impls depend on `db.Queries` directly | `internal/api/dbservices_*.go` | `internal/db/` | No repository abstraction |
| Service interfaces use canonical types directly | `api/interfaces.go` | `canon.*`, `graph.*`, `compiler.*` | Tight coupling, no anti-corruption layer |
| `internal/scene` depends on `graph.SceneStructure` | `internal/scene/types.go` | `internal/graph/` | Circular dependency risk, wrong boundary |
| LLM services depend on `compiler` | `internal/llm/services.go` | `internal/compiler/` | Compiler is not an adapter, should be domain interface |

---

## 2. Critical Issues

### 2.1 God Service Interfaces

`api.StoryService` has 9 methods spanning story CRUD + edge CRUD + node queries + topology. This mixes 3 bounded contexts (Story Aggregate, Graph Topology, Node Management).

### 2.2 Dual Interface Definitions

Same domain concepts defined in both `api/interfaces.go` and `canon/models.go` — with *different method signatures*. The `canon` versions lack `context.Context`. This reveals the layer confusion.

### 2.3 In-Memory vs DB Split at Implementation Level

The service implementations are split between `services_*.go` (in-memory) and `dbservices_*.go` (DB-backed). The `cmd/server/main.go` chooses which to use at startup. This means the in-memory implementations are dead code in production — carried for tests only, but co-located with production code.

### 2.4 Two-Phase Context Compilation Mismatch

The `CompiledContext` is built differently in:
1. `dbGenerationService.compileContext()` — omits `BranchSummary` and `CharState`
2. `GenerateSceneWorker.compilePromptParams()` — includes them

This means `context_hash` stored in `generations` doesn't represent the actual LLM input. Staleness detection is unreliable.

### 2.5 Direct DB Access from Workers

River workers call `w.Queries.*` directly (e.g., `w.Queries.GetCharacterLatest`, `w.Queries.UpdateGenerationOutput`). This bypasses any repository abstraction, couples workers to sqlc, and makes unit testing require a real database.

### 2.6 Missing Caching Layer

Zero caching anywhere. Every request:
- Re-fetches characters, locations, lore from DB
- Re-compiles context (two-phase mismatch)
- Has no prompt cache (identical prompts hit LLM again)

### 2.7 LLM Client Lacks Circuit Breaker

`Router.Complete` has retry + backoff but no:
- Circuit breaker (if Anthropic is down, every request retries 3x before failing)
- Fallback provider (if Sonnet fails, no attempt to use Haiku or Local)
- Rate limiting (bursts could hit API limits)

### 2.8 No Observability

Zero metrics. Zero structured logging. No tracing. No LLM cost tracking. No token usage monitoring.

### 2.9 Thread-Safety in Memory Stores

The `graph/memory.go`, `canon/memory.go`, `ledger/memory.go`, etc. use plain maps without `sync.RWMutex`. These are data races waiting to happen if any concurrent access occurs.

### 2.10 Migration Strategy Risk

Running migrations on startup (`migrate.Run()` in main.go) is fine for single-instance but dangerous for kubernetes deployments (multiple pods racing on migrations). No `rivermigrate` is run in main.go — River migrations might be out of sync.

---

## 3. Target Architecture

### High-Level Modular Monolith

```
┌─────────────────────────────────────────────────────────┐
│                     API Gateway (chi)                    │
│  Routes, Auth, Rate Limit, Request Validation            │
└──────────┬──────────┬──────────┬──────────┬────────────┘
           │          │          │          │
     ┌─────▼──┐ ┌────▼───┐ ┌───▼────┐ ┌───▼─────┐
     │ Story  │ │ Canon  │ │ Gen    │ │ Workflow│
     │ Domain │ │ Domain │ │ Domain │ │ Domain  │
     └────┬───┘ └───┬────┘ └───┬────┘ └───┬─────┘
          │         │          │           │
     ┌────▼─────────▼──────────▼───────────▼─────┐
     │           Domain Events (River)            │
     │  StoryCreated → CompileRequested → ...     │
     └──────────────────┬────────────────────────┘
                        │
     ┌──────────────────▼────────────────────────┐
     │         Infrastructure Layer              │
     │  PostgreSQL  Redis  LLM  Object Store     │
     └───────────────────────────────────────────┘
```

### Target Package Structure

```
internal/
├── domain/                    ← Pure domain, zero imports from infra
│   ├── story/
│   │   ├── model.go           Story, Node, Edge, SceneStructure
│   │   ├── service.go         StoryService interface
│   │   └── event.go           StoryCreated, NodeAccepted, etc.
│   ├── canon/
│   │   ├── model.go           Character, Location, Lore, Actor
│   │   ├── service.go         CanonService interface
│   │   └── event.go           CanonUpdated, LoreAdded
│   ├── generation/
│   │   ├── model.go           Generation, Prompt, Draft
│   │   ├── service.go         GenerationService interface
│   │   └── event.go           GenerationRequested, GenerationCompleted
│   ├── compiler/
│   │   ├── model.go           CompiledContext, PromptParams
│   │   ├── service.go         CompilerService interface
│   │   └── event.go           CompileRequested
│   ├── validation/
│   │   ├── model.go           ValidationResult, Violation
│   │   ├── service.go         ValidationService interface
│   │   └── event.go           ValidationCompleted
│   └── workflow/
│       ├── model.go           Pipeline, PipelineStep
│       ├── service.go         WorkflowService interface
│       └── event.go           PipelineStarted, StepCompleted
│
├── app/                       ← Application layer (use case orchestration)
│   ├── story/
│   │   ├── handler.go         HTTP handler (thin, delegates to use case)
│   │   ├── usecase.go         CreateStoryUseCase, AcceptSceneUseCase
│   │   └── router.go          Route registration
│   ├── canon/
│   ├── generation/
│   ├── compiler/
│   ├── validation/
│   └── workflow/
│
├── adapter/                   ← Infrastructure adapters (ports)
│   ├── repository/
│   │   ├── postgres/
│   │   │   ├── story_repo.go
│   │   │   ├── canon_repo.go
│   │   │   └── gen_repo.go
│   │   └── memory/            Test-only implementations
│   ├── llm/
│   │   ├── client.go          LLMClient interface (domain)
│   │   ├── anthropic.go       Anthropic adapter
│   │   ├── ollama.go          Ollama adapter
│   │   ├── router.go          Model tier router
│   │   ├── circuitbreaker.go  Circuit breaker decorator
│   │   └── cache.go           Prompt cache decorator
│   ├── cache/
│   │   └── redis.go           Redis cache implementations
│   ├── queue/
│   │   └── river.go           River adapter implementing domain event bus
│   └── observability/
│       ├── metrics.go         Prometheus metrics
│       ├── tracing.go         OpenTelemetry
│       └── logging.go         Structured JSON logging
│
├── bootstrap/                 ← Startup wiring
│   ├── config.go
│   ├── database.go
│   ├── redis.go
│   ├── queues.go
│   ├── llm.go
│   ├── observability.go
│   ├── api.go
│   └── app.go
│
├── db/                        ← sqlc-generated (kept as-is, adapted by repos)
├── migrate/
├── grpc/                      ← gRPC handlers (thin, wrap app layer)
└── river/                     ← River workers (events, not direct service calls)
    ├── generate_scene.go
    ├── extract_state.go
    └── ...
```

---

## 4. Domain Decomposition

### 4.1 Bounded Contexts

| Context | Core Entities | Owns | Never Does |
|---------|--------------|------|------------|
| **Story** | Story, Node, Edge, SceneStructure | Graph topology, node lifecycle, scene structure | Call LLM, update canon, validate |
| **Canon** | Character, Location, Lore, Actor, Casting | Character versioning, lore vector search, casting | Generate content, build prompts |
| **Generation** | Generation, Draft, TokenUsage | LLM generation, prompt snapshots, output storage | Persist stories, update canon |
| **Compiler** | CompiledContext, PromptParams | Context assembly, prompt building, caching | Call LLM, persist anything |
| **Validation** | ValidationResult, Violation | Canon consistency, quality checks, continuity | Build prompts, persist stories |
| **Workflow** | Pipeline, PipelineStep | Event routing, job orchestration, pipeline state | Any business logic |
| **Ledger** | CharacterState, StateDelta | Continuity tracking, state diffs, knowledge graph | Content generation |

### 4.2 Service Interface Definitions (Ports)

```go
// domain/story/service.go
type StoryService interface {
    Create(ctx context.Context, title string) (*Story, error)
    Get(ctx context.Context, id uuid.UUID) (*Story, error)
    List(ctx context.Context) ([]Story, error)
    CreateNode(ctx context.Context, storyID uuid.UUID, spec CreateNodeSpec) (*Node, error)
    UpdateNode(ctx context.Context, id uuid.UUID, spec UpdateNodeSpec) (*Node, error)
    GetNode(ctx context.Context, id uuid.UUID) (*Node, error)
    ListNodes(ctx context.Context, storyID uuid.UUID) ([]Node, error)
    CreateEdge(ctx context.Context, storyID, from, to uuid.UUID, edgeType EdgeType) error
    ListEdges(ctx context.Context, storyID uuid.UUID) ([]Edge, error)
    Topology(ctx context.Context, storyID uuid.UUID) (*Topology, error)
    GetIncomingEdges(ctx context.Context, nodeID uuid.UUID) ([]Edge, error)
    GetOutgoingEdges(ctx context.Context, nodeID uuid.UUID) ([]Edge, error)
    SetSceneStructure(ctx context.Context, nodeID uuid.UUID, ss SceneStructure) error
    GetSceneStructure(ctx context.Context, nodeID uuid.UUID) (*SceneStructure, error)
}

// domain/canon/service.go
type CanonService interface {
    CreateCharacter(ctx context.Context, spec CreateCharacterSpec) (*Character, error)
    GetCharacter(ctx context.Context, id uuid.UUID, version int) (*Character, error)
    UpdateCharacter(ctx context.Context, id uuid.UUID, spec UpdateCharacterSpec) (*Character, error)
    ListCharacters(ctx context.Context) ([]Character, error)
    PinCharacter(ctx context.Context, characterID, storyID uuid.UUID, version int) error
    CreateLocation(ctx context.Context, spec CreateLocationSpec) (*Location, error)
    GetLocation(ctx context.Context, id uuid.UUID, version int) (*Location, error)
    UpdateLocation(ctx context.Context, id uuid.UUID, spec UpdateLocationSpec) (*Location, error)
    ListLocations(ctx context.Context) ([]Location, error)
    CreateLore(ctx context.Context, tags []string, content string, embedding []float32) (*Lore, error)
    SearchLoreByTags(ctx context.Context, tags []string) ([]Lore, error)
    SearchLoreSimilar(ctx context.Context, embedding []float32, limit int) ([]Lore, error)
    CreateActor(ctx context.Context, spec CreateActorSpec) (*Actor, error)
    GetActor(ctx context.Context, id uuid.UUID) (*Actor, error)
    UpdateActor(ctx context.Context, id uuid.UUID, spec UpdateActorSpec) (*Actor, error)
    ListActors(ctx context.Context) ([]Actor, error)
    CreateCasting(ctx context.Context, storyID, actorID, characterID uuid.UUID, roleType string) (*Casting, error)
    // ... trait service methods
}

// domain/generation/service.go
type GenerationService interface {
    Generate(ctx context.Context, nodeID uuid.UUID) (*Generation, error)
    AcceptGeneration(ctx context.Context, nodeID, generationID uuid.UUID) error
    ListGenerations(ctx context.Context, nodeID uuid.UUID) ([]Generation, error)
    GetGeneration(ctx context.Context, id uuid.UUID) (*Generation, error)
    IsStale(ctx context.Context, nodeID uuid.UUID, contextHash string) (bool, error)
}

// domain/compiler/service.go
type CompilerService interface {
    CompileSceneContext(ctx context.Context, nodeID uuid.UUID) (*CompiledContext, error)
    ComputeHash(ctx *CompiledContext) string
    BuildScenePrompt(ctx *CompiledContext) (*PromptParams, error)
}

// domain/validation/service.go
type ValidationService interface {
    ValidateScene(ctx context.Context, generationID uuid.UUID, compiledCanon, charState, sceneText string) (*ValidationResult, error)
    ValidateCanonConsistency(ctx context.Context, storyID uuid.UUID) ([]Violation, error)
}

// domain/workflow/service.go
type WorkflowService interface {
    StartGenerationPipeline(ctx context.Context, nodeID uuid.UUID) error
    OnGenerationCompleted(ctx context.Context, generationID uuid.UUID) error
    OnValidationCompleted(ctx context.Context, generationID uuid.UUID, result *ValidationResult) error
    OnStateExtracted(ctx context.Context, generationID uuid.UUID) error
    OnSummaryUpdated(ctx context.Context, nodeID uuid.UUID) error
}

// domain/ledger/service.go
type LedgerService interface {
    GetState(ctx context.Context, storyID, characterID, asOfNode uuid.UUID) (*CharacterState, error)
    GetStatesForNode(ctx context.Context, storyID, nodeID uuid.UUID) (map[uuid.UUID]CharacterState, error)
    ApplyDeltas(ctx context.Context, storyID, nodeID uuid.UUID, deltas StateDeltas) error
    GetStateAtBranch(ctx context.Context, storyID, nodeID uuid.UUID) (map[uuid.UUID]CharacterState, error)
}
```

---

## 5. Ports & Adapters Layer

### 5.1 Repository Interfaces

```go
// domain/story/repository.go
type StoryRepository interface {
    Create(ctx context.Context, s *Story) error
    Get(ctx context.Context, id uuid.UUID) (*Story, error)
    List(ctx context.Context) ([]Story, error)
    UpdateTitle(ctx context.Context, id uuid.UUID, title string) error
}

type NodeRepository interface {
    Create(ctx context.Context, n *Node) error
    Get(ctx context.Context, id uuid.UUID) (*Node, error)
    Update(ctx context.Context, n *Node) error
    SetStatus(ctx context.Context, id uuid.UUID, status NodeStatus) error
    SetSceneStructure(ctx context.Context, id uuid.UUID, ss *SceneStructure) error
    ListByStory(ctx context.Context, storyID uuid.UUID) ([]Node, error)
}

type EdgeRepository interface {
    Create(ctx context.Context, e *Edge) error
    ListByStory(ctx context.Context, storyID uuid.UUID) ([]Edge, error)
    GetOutgoing(ctx context.Context, nodeID uuid.UUID) ([]Edge, error)
    GetIncoming(ctx context.Context, nodeID uuid.UUID) ([]Edge, error)
}

// domain/canon/repository.go
type CharacterRepository interface {
    Create(ctx context.Context, c *Character) error
    Get(ctx context.Context, id uuid.UUID, version int) (*Character, error)
    GetLatest(ctx context.Context, id uuid.UUID) (*Character, error)
    Update(ctx context.Context, c *Character) error
    List(ctx context.Context) ([]Character, error)
}

type LoreRepository interface {
    Create(ctx context.Context, l *Lore) error
    List(ctx context.Context) ([]Lore, error)
    SearchByTags(ctx context.Context, tags []string) ([]Lore, error)
    SearchSimilar(ctx context.Context, embedding []float32, limit int) ([]Lore, error)
}

// domain/generation/repository.go
type GenerationRepository interface {
    Create(ctx context.Context, g *Generation) error
    Get(ctx context.Context, id uuid.UUID) (*Generation, error)
    Accept(ctx context.Context, id uuid.UUID) error
    RejectOthers(ctx context.Context, nodeID, exceptID uuid.UUID) error
    UpdateOutput(ctx context.Context, id uuid.UUID, output, model string) error
    UpdateValidation(ctx context.Context, id uuid.UUID, result json.RawMessage) error
    ListByNode(ctx context.Context, nodeID uuid.UUID) ([]Generation, error)
    IsStale(ctx context.Context, nodeID uuid.UUID, contextHash string) (bool, error)
}
```

### 5.2 LLM Port

```go
// domain/generation/llm.go (this is a PORT, not an adapter)
type LLMGenerator interface {
    Generate(ctx context.Context, prompt PromptParams) (*GenerationResult, error)
    GenerateWithProvider(ctx context.Context, prompt PromptParams, provider ModelProvider) (*GenerationResult, error)
}

type ModelProvider interface {
    Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
    Name() string
    CostPerToken() Cost
}
```

### 5.3 Event Bus Port

```go
// domain/workflow/eventbus.go
type EventBus interface {
    Publish(ctx context.Context, event DomainEvent) error
    Subscribe(eventType string, handler EventHandler)
}

type DomainEvent interface {
    EventType() string
    AggregateID() uuid.UUID
    Timestamp() time.Time
}

type EventHandler func(ctx context.Context, event DomainEvent) error
```

---

## 6. Event Model

### 6.1 Domain Events

```go
// ── Story Events ────────────────────────────
type StoryCreated struct {
    StoryID uuid.UUID `json:"story_id"`
    Title   string    `json:"title"`
    At      time.Time `json:"at"`
}

type NodeCreated struct {
    NodeID  uuid.UUID `json:"node_id"`
    StoryID uuid.UUID `json:"story_id"`
    BeatIntent string `json:"beat_intent"`
}

type NodeStatusChanged struct {
    NodeID      uuid.UUID    `json:"node_id"`
    OldStatus   NodeStatus   `json:"old_status"`
    NewStatus   NodeStatus   `json:"new_status"`
}

// ── Generation Events ───────────────────────
type GenerationRequested struct {
    NodeID       uuid.UUID `json:"node_id"`
    StoryID      uuid.UUID `json:"story_id"`
    GenerationID uuid.UUID `json:"generation_id"`
    ContextHash  string    `json:"context_hash"`
}

type GenerationCompleted struct {
    GenerationID uuid.UUID `json:"generation_id"`
    NodeID       uuid.UUID `json:"node_id"`
    Output       string    `json:"output"`
    Model        string    `json:"model"`
    TokenUsage   TokenUsage `json:"token_usage,omitempty"`
}

type GenerationAccepted struct {
    NodeID       uuid.UUID `json:"node_id"`
    GenerationID uuid.UUID `json:"generation_id"`
    StoryID      uuid.UUID `json:"story_id"`
}

// ── Compiler Events ─────────────────────────
type CompileRequested struct {
    NodeID  uuid.UUID `json:"node_id"`
    StoryID uuid.UUID `json:"story_id"`
}

type CompileCompleted struct {
    NodeID      uuid.UUID `json:"node_id"`
    ContextHash string    `json:"context_hash"`
    PromptSnapshot string `json:"prompt_snapshot,omitempty"`
}

// ── Validation Events ───────────────────────
type ValidationCompleted struct {
    GenerationID uuid.UUID        `json:"generation_id"`
    Result       ValidationResult `json:"result"`
    Passed       bool             `json:"passed"`
}

// ── Ledger Events ───────────────────────────
type StateExtracted struct {
    NodeID  uuid.UUID    `json:"node_id"`
    StoryID uuid.UUID    `json:"story_id"`
    Deltas  StateDeltas  `json:"deltas"`
}

type SummaryUpdated struct {
    StoryID  uuid.UUID `json:"story_id"`
    NodeID   uuid.UUID `json:"node_id,omitempty"`
    Level    string    `json:"level"`
}
```

### 6.2 Event Flow Diagram

```
User accepts generation
        │
        ▼
┌───────────────────┐    GenerationAccepted
│ AcceptSceneUseCase│─────────────────────►
└───────────────────┘
        │
        ├──────────────────────────────────────────────┐
        │                                              │
        ▼                                              ▼
┌───────────────────┐                    ┌─────────────────────┐
│ ExtractStateWorker│                    │ UpdateSummaryWorker │
│  (River queue)    │                    │  (River queue)      │
│  → calls LLM      │                    │  → calls LLM        │
│  → persists       │                    │  → persists         │
│  → StateExtracted │                    │  → SummaryUpdated   │
└───────────────────┘                    └─────────────────────┘
        │                                              │
        ▼                                              ▼
┌───────────────────┐                    ┌─────────────────────┐
│ ValidateWorker    │                    │ ElevateWorker       │
│  (River queue)    │                    │  (triggered on      │
│  → calls LLM      │                    │   count >= threshold)│
│  → persists       │                    │  → merges summaries │
│  → Validation     │                    │  → SummaryUpdated   │
│    Completed      │                    └─────────────────────┘
└───────────────────┘
```

### 6.3 Event-Driven Generation Pipeline

```
START: POST /api/v1/stories/{id}/nodes/{nid}/generate

1. Story Domain:  Validate node exists, check not stale
2. Event:         GenerationRequested{nodeID, storyID}
    │
    ▼
3. Compiler:      CompileRequested → compile context → cache result
    │              Event: CompileCompleted{contextHash}
    ▼
4. Generation:    GenerationRequested → insert pending gen → enqueue
    │              River: GenerateSceneWorker
    ▼
5. LLM Router:    Anthropic/Ollama → response
    │
    ▼
6. Generation:    UpdateGenerationOutput(genID, output, model)
    │              Event: GenerationCompleted
    ▼
7. Story Domain:  Set node status to 'generated'
                  (if completed successfully)

───────────────────────────────────────────────────

START: POST /api/v1/stories/{id}/nodes/{nid}/accept

1. Generation:    AcceptGeneration(genID) → reject others
2. Event:         GenerationAccepted{nodeID, genID, storyID}
    │
    ├──► ExtractStateWorker (queue: extract)
    │      → LedgerService.ApplyDeltas()
    │      → Event: StateExtracted
    │
    ├──► UpdateSummaryWorker (queue: default)
    │      → Event: SummaryUpdated
    │
    └──► ValidateWorker (queue: validate)
           → Event: ValidationCompleted
```

### 6.4 Event Bus Implementation

For a modular monolith, **River** is the right choice — it's already a dependency.

```go
// adapter/queue/river.go
type RiverEventBus struct {
    client *river.Client[pgx.Tx]
}

func (b *RiverEventBus) Publish(ctx context.Context, event DomainEvent) error {
    // Events go through River's unique-by-kind mechanism
    // Each event type maps to a River job type
    switch e := event.(type) {
    case *GenerationRequested:
        _, err := b.client.Insert(ctx, &GenerateSceneArgs{...}, nil)
        return err
    case *GenerationAccepted:
        _, err := b.client.Insert(ctx, &ExtractStateArgs{...}, nil)
        if err != nil { return err }
        _, err = b.client.Insert(ctx, &UpdateSummaryArgs{...}, nil)
        if err != nil { return err }
        _, err = b.client.Insert(ctx, &ValidateSceneArgs{...}, nil)
        return err
    // ...
    }
}
```

**Tradeoff Analysis for Event Infrastructure:**

| Option | Pros | Cons | Verdict |
|--------|------|------|---------|
| **River (current)** | Already in stack, transactional outbox with PG, unique jobs | PG-bound, limited throughput | ✅ Phase 1-2 |
| **Redis Streams** | High throughput, in-memory speed, built-in features | Operational overhead, loss risk | ⬜ Phase 7 |
| **NATS** | High throughput, at-least-once, exactly-once | Separate infra to run | ❌ Too complex now |
| **Kafka** | Industry standard, infinite retention, replay | Heavy operations, overkill for current scale | ❌ Phase 8 |

**Recommendation:** Stay with River through Phase 6. Add Redis Streams in Phase 7 for workflow state. Only consider Kafka at microservice scale (Phase 8).

---

## 7. Redis Architecture

### 7.1 Current State: No Redis

### 7.2 Design: Redis as First-Class Component

```
┌──────────────────────────────────────────────────────────┐
│                       Redis                               │
│                                                          │
│  context:hash:{story_id}:{node_id}   → CompiledContext   │
│  prompt:hash:{sha256}                → {prompt, response}│
│  gen:state:{generation_id}           → {status, step, %} │
│  workflow:{story_id}                 → pipeline state     │
│  rate:llm:{user_id}:{period}         → request count     │
│  lock:story:{story_id}               → distributed lock   │
│  session:{user_id}                   → user session       │
│  job:{id}                            → job status         │
└──────────────────────────────────────────────────────────┘
```

### 7.3 Context Cache

```
Key:    ctx:s:{story_id}:n:{node_id}
Value:  CompiledContext (JSON, ~2-10KB)
TTL:    300s (5 minutes)
Policy: Invalidate on:
        - Character update for any ref in context
        - Lore update matching any tag in context
        - Location update matching context location
        - Branch summary change
```

**Implementation:**

```go
type ContextCache struct {
    rdb *redis.Client
}

func (c *ContextCache) Get(ctx context.Context, storyID, nodeID uuid.UUID) (*compiler.CompiledContext, error) {
    key := fmt.Sprintf("ctx:s:%s:n:%s", storyID, nodeID)
    data, err := c.rdb.Get(ctx, key).Bytes()
    if err != nil {
        return nil, err // miss
    }
    var cc compiler.CompiledContext
    return &cc, json.Unmarshal(data, &cc)
}

func (c *ContextCache) Set(ctx context.Context, storyID, nodeID uuid.UUID, cc *compiler.CompiledContext) error {
    key := fmt.Sprintf("ctx:s:%s:n:%s", storyID, nodeID)
    data, _ := json.Marshal(cc)
    return c.rdb.Set(ctx, key, data, 300*time.Second).Err()
}

func (c *ContextCache) Invalidate(ctx context.Context, storyID uuid.UUID, changedEntity EntityRef) error {
    // On character update: find all nodes referencing this char
    // Invalidate their context cache entries
    // See "cache invalidation" section below
}
```

**Estimated Impact:**

| Metric | Without Cache | With Cache |
|--------|--------------|------------|
| Context build time | ~50-200ms (4-6 DB queries) | ~1ms (Redis get) |
| DB queries per generation | 6-8 | 0-2 |
| P99 generation latency | ~8s | ~7.8s (minimal save) |

### 7.4 Prompt Cache

```
Key:    prompt:{sha256_of_full_prompt}
Value:  {
           "prompt": "...",
           "response": "...",
           "model": "claude-sonnet-4-20250514",
           "created_at": "...",
           "token_count": {input, output}
        }
TTL:    86400s (24 hours)
Policy: LRU eviction, max 10K entries
```

**Hashing Strategy:**

```go
func promptCacheKey(prompt string, model ModelTier, temperature float64) string {
    h := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%.2f", prompt, model, temperature)))
    return fmt.Sprintf("prompt:%x", h[:16]) // 16 bytes = 32 hex chars
}
```

**Cost Savings Estimate:**

| Scenario | Daily LLM Calls | Cache Hit Rate | Daily Savings (Anthropic) |
|----------|----------------|----------------|--------------------------|
| Development | 500 | 40% | ~$30/day |
| Production (10K users) | 50,000 | 25% | ~$1,250/day |
| Production (100K users) | 500,000 | 20% | ~$10,000/day |

*Note: Prompt caching is most effective for:*
- Re-generation of same scene (user tweaks params)
- Similar context scenarios (same characters/location)
- Repeated validation calls on similar content

### 7.5 Workflow State

```
Key:    wf:gen:{generation_id}
Value:  {
           "status": "compiling|generating|validating|extracting|done|failed",
           "current_step": 2,
           "total_steps": 4,
           "started_at": "...",
           "completed_at": "...",
           "error": "..."  // if failed
        }
TTL:    3600s (1 hour after completion)
```

Avoid polling `generations` table for running jobs. River already provides job status, but workflow state adds step-level granularity without DB load.

### 7.6 Distributed Locks

Prevent concurrent generation for the same node:

```go
func (s *GenerationService) Generate(ctx context.Context, nodeID uuid.UUID) error {
    lockKey := fmt.Sprintf("lock:gen:%s", nodeID)
    lock, err := s.rdb.SetNX(ctx, lockKey, "1", 30*time.Second).Result()
    if err != nil {
        return fmt.Errorf("lock check failed: %w", err)
    }
    if !lock {
        return ErrGenerationInProgress
    }
    defer s.rdb.Del(ctx, lockKey)
    // ... actual generation logic
}
```

### 7.7 Rate Limiting

```go
type RateLimiter struct {
    rdb *redis.Client
}

func (rl *RateLimiter) Allow(ctx context.Context, userID string, limit int, window time.Duration) (bool, error) {
    key := fmt.Sprintf("rate:llm:%s:%d", userID, window.Seconds())
    count, err := rl.rdb.Incr(ctx, key).Result()
    if err != nil {
        return false, err
    }
    if count == 1 {
        rl.rdb.Expire(ctx, key, window)
    }
    return count <= int64(limit), nil
}
```

---

## 8. Database Architecture

### 8.1 Schema Assessment

**Strengths:**
- Versioned canon (characters, locations) with composite PK `(id, version)` is correct for append-only history
- Partial unique indexes on summaries are elegant
- pgvector integration is well done (IVFFlat index with cosine distance)
- GIN index on lore tags for array overlap is appropriate

**Weaknesses:**
- Missing FKs on `character_id` in `casting` and `character_state` (due to versioning — acceptable but needs documentation)
- No indexes on `characters(name)` or `locations(name)` — name lookups are common
- No composite index on `nodes(story_id, status)` — filtering by status per story
- `edges.story_id` + `(from_node, to_node)` — needs index on `(from_node)` and `(to_node)` for graph traversal queries
- Generation `created_at` not indexed — list queries sort by this
- No `updated_at` on `stories` — can't detect story-level changes
- `story_summaries` `created_at` not indexed — list queries sort by this

### 8.2 Recommended Schema Changes

```sql
-- Migration: Add missing indexes
CREATE INDEX idx_characters_name ON characters(name);
CREATE INDEX idx_locations_name ON locations(name);
CREATE INDEX idx_nodes_story_status ON nodes(story_id, status);
CREATE INDEX idx_edges_from ON edges(from_node);
CREATE INDEX idx_edges_to ON edges(to_node);
CREATE INDEX idx_generations_created ON generations(created_at DESC);
CREATE INDEX idx_summaries_created ON story_summaries(created_at DESC);

-- Add updated_at to stories for change detection
ALTER TABLE stories ADD COLUMN updated_at timestamptz NOT NULL DEFAULT now();
CREATE INDEX idx_stories_updated ON stories(updated_at DESC);
```

### 8.3 Partitioning Plan

**By time (generations, scene_turns):**

```sql
-- For 1M+ generations, partition by month
CREATE TABLE generations (
    id UUID NOT NULL,
    node_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- ... other columns
) PARTITION BY RANGE (created_at);

CREATE TABLE generations_2026_01 PARTITION OF generations
    FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');
CREATE TABLE generations_2026_02 PARTITION OF generations
    FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');
-- ... monthly partitions
```

**By story_id (nodes, edges, character_state):**

For 100K+ stories, consider hash partitioning:
```sql
CREATE TABLE nodes (
    id UUID NOT NULL,
    story_id UUID NOT NULL,
    -- ...
) PARTITION BY HASH (story_id);

-- 8 partitions for balanced distribution
CREATE TABLE nodes_0 PARTITION OF nodes FOR VALUES WITH (MODULUS 8, REMAINDER 0);
-- ... 7 more partitions
```

### 8.4 Read Models / CQRS

For topology queries (loading entire story DAG), use a materialized view:

```sql
CREATE MATERIALIZED VIEW story_topology AS
SELECT
    s.id AS story_id,
    jsonb_agg(DISTINCT jsonb_build_object(
        'id', n.id,
        'beat_intent', n.beat_intent,
        'status', n.status,
        'character_refs', n.character_refs,
        'pov', n.pov
    )) AS nodes,
    jsonb_agg(DISTINCT jsonb_build_object(
        'from_node', e.from_node,
        'to_node', e.to_node,
        'edge_type', e.edge_type
    )) AS edges
FROM stories s
LEFT JOIN nodes n ON n.story_id = s.id
LEFT JOIN edges e ON e.story_id = s.id
GROUP BY s.id;

REFRESH MATERIALIZED VIEW CONCURRENTLY story_topology;
```

### 8.5 Scaling Plan

| Tier | Users | Stories | PG Strategy | Redis Strategy |
|------|-------|---------|-------------|----------------|
| S | 1K | 10K | Single instance, no partitioning | Single instance |
| M | 10K | 100K | Read replicas (1 primary, 2 replicas), partitions by month | Sentinel cluster (3 nodes) |
| L | 100K | 1M | Connection pooling (PgBouncer), read replicas (1 primary, 4 replicas), partitions by month + hash | Cluster mode (6 shards) |
| XL | 1M | 10M | Sharding (Citus), read replicas, separate analytics DB | Cluster mode (12 shards) |

---

## 9. Graph Storage Strategy

### 9.1 Current: PostgreSQL Relational Model

```
nodes: {id, story_id, beat_intent, character_refs[], ...}
edges: {story_id, from_node, to_node, edge_type}
```

### 9.2 Graph Database Comparison

| Feature | PostgreSQL | Neo4j | Memgraph | Dgraph |
|---------|-----------|-------|----------|--------|
| Story DAG traversal | ✅ Recursive CTE | ✅ Native traversal | ✅ In-memory speed | ✅ Native |
| Character relationships | 🟡 JSONB joins | ✅ Native edges | ✅ Native | ✅ Native |
| Canon traversal | 🟡 JOIN explosion | ✅ Efficient | ✅ Efficient | ✅ Efficient |
| Vector search | ✅ pgvector | 🟡 External | ❌ | 🟡 External |
| Operational complexity | 🟡 Already running | ❌ New infra | ❌ New infra | ❌ New infra |
| Multi-tenancy | ✅ Row-level security | 🟡 Per-db | 🟡 Per-db | 🟡 Per-db |

### 9.3 Recommendation: Hybrid (Phase 6+)

**Phase 1-5:** Stay with PostgreSQL. Graph traversal queries are bounded (single story DAG traversal is a few hundred nodes max). Recursive CTEs handle this efficiently.

**Phase 6+ (100K+ stories, complex canon graphs):** Introduce Neo4j or Memgraph ONLY for canon relationship queries:

```
Neo4j Schema:

(:Character {id, name, persona})
(:Location {id, name, description})
(:Lore {id, tags, content})

(:Character)-[:FRIEND_OF {intensity}]->(:Character)
(:Character)-[:VISITED {at_node}]->(:Location)
(:Character)-[:PARTICIPATED_IN]->(:Scene {node_id})
(:Character)-[:KNOWS_ABOUT]->(:Lore)
```

**Graph Traversal Examples (Neo4j):**

```cypher
-- Find all characters connected to a scene within 2 hops
MATCH (c:Character)-[*1..2]-(s:Scene {node_id: $nodeId})
RETURN DISTINCT c

-- Find all knowledge spread paths
MATCH path = (c1:Character {id: $charId})-[:FRIEND_OF*1..3]-(c2:Character)
WHERE c1 <> c2
RETURN path

-- Find locations visited by a character's social circle
MATCH (c:Character {id: $charId})-[:FRIEND_OF]->(:Character)-[:VISITED]->(l:Location)
RETURN DISTINCT l
```

### 9.4 Graph Storage in PostgreSQL (Until Phase 6)

For now, optimize PostgreSQL for graph traversal:

```sql
-- Recursive CTE for topological sort
WITH RECURSIVE topo AS (
    SELECT n.id, n.beat_intent, 0 AS depth
    FROM nodes n
    WHERE n.story_id = $1
      AND NOT EXISTS (SELECT 1 FROM edges e WHERE e.to_node = n.id AND e.story_id = $1)

    UNION ALL

    SELECT n.id, n.beat_intent, t.depth + 1
    FROM nodes n
    JOIN edges e ON e.from_node = n.id AND e.story_id = $1
    JOIN topo t ON t.id = e.to_node
)
SELECT DISTINCT ON (id) id, beat_intent, depth
FROM topo
ORDER BY id, depth DESC;
```

---

## 10. MongoDB Evaluation

### 10.1 Potential Use Cases

| Use Case | Current Storage | MongoDB Suitability | Verdict |
|----------|----------------|--------------------|---------|
| Prompt snapshots | `generations.prompt_snapshot` (text) | ✅ Document fits well | 🟡 PostgreSQL TEXT works fine |
| Story drafts | `generations.output` (text) | ✅ Large documents | 🟡 Same |
| LLM responses | `generations.validation_result` (JSONB) | ✅ Flexible schema | ❌ JSONB is equivalent |
| Version history | `characters` append-only rows | ✅ Version documents | 🟡 PG versioning works |
| Scene structure | `nodes.scene_structure` (JSONB) | ✅ Nested documents | ❌ JSONB is equivalent |
| Character state | `character_state.state` (JSONB) | ✅ Flexible | ❌ Key-value pattern, Redis better |

### 10.2 Analysis

**JSONB Equivalence:**
PostgreSQL 17's JSONB supports:
- Indexing (GIN, BTREE on expressions)
- Path queries (`#>>`, `@>`, `?`)
- Partial updates (`jsonb_set`)
- ACID compliance
- JOINs with relational data

**MongoDB Advantages:**
- Schema evolution without migrations
- Automatic sharding
- Better write throughput (no MVCC overhead)

**MongoDB Disadvantages:**
- No JOINs (need application-level joins or embedded docs)
- No transactions across collections (until 4.0+)
- Additional operational complexity
- Eventual consistency by default
- No pgvector for embeddings

### 10.3 Recommendation

**Do NOT add MongoDB.** Every use case in this application is equally well-served (or better served) by:

1. **PostgreSQL JSONB** for flexible documents (scene structures, character states, validation results)
2. **Redis** for caching, session state, workflow state
3. **Object Storage (S3/MinIO)** for large artifacts (exported stories, prompt archives)

PostgreSQL JSONB provides ACID compliance with relational data, which is critical for story integrity. MongoDB would add operational complexity with zero benefit for this workload.

---

## 11. LLM Architecture

### 11.1 Provider Abstraction Layer

```
┌────────────────────────────────────────────────────────────┐
│                    LLM Generator (port)                      │
│  Generate(ctx, prompt) → (Result, error)                    │
└────────────────────────────────────────────────────────────┘
          │
          ▼
┌────────────────────────────────────────────────────────────┐
│              Provider Router + Circuit Breaker              │
│  Routes by ModelTier, wraps in circuit breaker              │
│  Falls back on failure (Sonnet→Haiku→Local)                 │
└────────────────────────────────────────────────────────────┘
          │
          ├──────────────┬──────────────┬──────────────────┐
          ▼              ▼              ▼                  ▼
┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────────┐
│  Anthropic  │ │   OpenAI    │ │   Ollama    │ │   Gemini (future)│
│  Adapter    │ │  Adapter    │ │  Adapter    │ │   Adapter        │
└─────────────┘ └─────────────┘ └─────────────┘ └─────────────────┘
```

### 11.2 Circuit Breaker Implementation

```go
type CircuitBreaker struct {
    failures    int
    maxFailures int
    state       State  // Closed, Open, HalfOpen
    lastFailure time.Time
    timeout     time.Duration
    mu          sync.Mutex
}

func (cb *CircuitBreaker) Call(ctx context.Context, fn func() (*CompletionResponse, error)) (*CompletionResponse, error) {
    cb.mu.Lock()
    if cb.state == Open {
        if time.Since(cb.lastFailure) > cb.timeout {
            cb.state = HalfOpen
        } else {
            cb.mu.Unlock()
            return nil, ErrCircuitOpen
        }
    }
    cb.mu.Unlock()

    resp, err := fn()
    if err != nil {
        cb.recordFailure()
        return nil, err
    }
    cb.reset()
    return resp, nil
}

type FallbackRouter struct {
    providers []ModelProvider  // ordered by priority
    cb        *CircuitBreaker
}

func (r *FallbackRouter) Generate(ctx context.Context, prompt PromptParams) (*CompletionResponse, error) {
    for _, provider := range r.providers {
        resp, err := r.cb.Call(ctx, func() (*CompletionResponse, error) {
            return provider.Complete(ctx, prompt.ToRequest())
        })
        if err == nil {
            return resp, nil
        }
        log.Printf("provider %s failed: %v, trying next", provider.Name(), err)
    }
    return nil, ErrAllProvidersFailed
}
```

### 11.3 Rate Limiting

```go
// adapter/llm/ratelimit.go
type RateLimitedProvider struct {
    inner ModelProvider
    limiter *RateLimiter
}

func (p *RateLimitedProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
    allowed, err := p.limiter.Allow(ctx, p.inner.Name(), 60, time.Minute)
    if err != nil || !allowed {
        return nil, ErrRateLimited
    }
    return p.inner.Complete(ctx, req)
}
```

### 11.4 Token Tracking

```go
// Every LLM response includes token usage
type TokenUsage struct {
    InputTokens  int     `json:"input_tokens"`
    OutputTokens int     `json:"output_tokens"`
    Cost         float64 `json:"cost"` // in USD
}

// adapter/llm/costing.go
var ModelCosts = map[string]Cost{
    "claude-sonnet-4-20250514": {Input: 15.0, Output: 75.0},   // per 1M tokens
    "claude-haiku-3-5-20250224": {Input: 0.8, Output: 4.0},
    "llama3.2:3b":              {Input: 0.0, Output: 0.0},     // local, free
}
```

---

## 12. Prompt Compiler Architecture

### 12.1 Current Issues

1. **Two-phase context compilation mismatch** — `dbGenerationService.compileContext()` vs `GenerateSceneWorker.compilePromptParams()` build different contexts
2. **No context caching** — each generation rebuilds from scratch
3. **No token budget management** — no control over prompt size
4. **No prompt optimization** — prompts sent as-is, no truncation/summarization

### 12.2 Target Design

```
┌──────────────────────────────────────────────────────────────────┐
│                        CompilerService                            │
│                                                                   │
│  CompileSceneContext(nodeID) → CompiledContext                     │
│  ├─ StoryRepository.GetStory()                                    │
│  ├─ NodeRepository.Get()                                          │
│  ├─ CharacterRepository.GetLatest() (for each ref)                │
│  ├─ LocationRepository.GetLatest() (if location ref)              │
│  ├─ LoreRepository.SearchByTags() (by character names)            │
│  ├─ LedgerService.GetStatesForNode()                             │
│  └─ SummaryService.GetSceneSummary() (branch summary)             │
│                                                                   │
│  ComputeHash(ctx) → string (SHA256 of JSON)                       │
│  BuildScenePrompt(ctx) → PromptParams                             │
│  OptimizePrompt(params, tokenBudget) → PromptParams               │
│                                                                   │
└──────────────────────────────────────────────────────────────────┘
```

### 12.3 Context Cache Integration

```go
type CompilerService struct {
    storyRepo    StoryRepository
    nodeRepo     NodeRepository
    charRepo     CharacterRepository
    locRepo      LocationRepository
    loreRepo     LoreRepository
    ledger       LedgerService
    summary      SummaryService
    cache        *ContextCache  // Redis-backed
}

func (s *CompilerService) CompileSceneContext(ctx context.Context, nodeID uuid.UUID) (*CompiledContext, error) {
    // 1. Check cache first
    node, err := s.nodeRepo.Get(ctx, nodeID)
    if err != nil {
        return nil, err
    }
    cached, err := s.cache.Get(ctx, node.StoryID, nodeID)
    if err == nil && !s.isStale(ctx, cached, node) {
        return cached, nil
    }

    // 2. Build fresh context
    cc := &CompiledContext{...}
    // ... load from repositories ...

    // 3. Cache it
    s.cache.Set(ctx, node.StoryID, nodeID, cc)
    return cc, nil
}

func (s *CompilerService) isStale(ctx context.Context, cc *CompiledContext, node *Node) bool {
    // Check if any referenced character was updated after context build
    // Check if any matching lore was added
    // Check if branch summary changed
    // Simple TTL check is sufficient for Phase 3
    return time.Since(cc.BuiltAt) > 5*time.Minute
}
```

### 12.4 Token Budget Manager

```go
type TokenBudget struct {
    MaxTotal    int
    UsedCanon   int
    UsedState   int
    UsedSummary int
    UsedUser    int
}

func (s *CompilerService) OptimizePrompt(ctx *CompiledContext, budget TokenBudget) *CompiledContext {
    // Strategy 1: Truncate lore to fit budget
    if estimateTokens(ctx.Lore) > budget.MaxTotal/3 {
        ctx.Lore = selectMostRelevant(ctx.Lore, budget.MaxTotal/3)
    }

    // Strategy 2: Summarize character cards if too large
    if estimateTokens(ctx.CharacterCards) > budget.MaxTotal/3 {
        ctx.CharacterCards = condenseCardDescriptions(ctx.CharacterCards)
    }

    // Strategy 3: Truncate branch summary
    if estimateTokens(ctx.BranchSummary) > 500 {
        ctx.BranchSummary = truncateToTokens(ctx.BranchSummary, 500)
    }

    return ctx
}
```

---

## 13. Observability

### 13.1 Structured Logging

```go
// bootstrap/observability.go
func InitLogging() *slog.Logger {
    return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelInfo,
        ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
            if a.Key == slog.TimeKey {
                a.Key = "timestamp"
                a.Value = slog.StringValue(time.Now().UTC().Format(time.RFC3339Nano))
            }
            return a
        },
    }))
}
```

**Log Levels:**
- `ERROR`: LLM failures, DB connection drops, circuit breaker open
- `WARN`: Rate limit approaching, stale cache, retry > 2
- `INFO`: Generation start/completion, job enqueue/dequeue, scene accept
- `DEBUG`: Full prompt snapshot, cache hit/miss, SQL query

### 13.2 Metrics (Prometheus)

```go
// adapter/observability/metrics.go
var (
    HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
        Name: "http_request_duration_seconds",
        Help: "HTTP request latency",
        Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
    }, []string{"method", "path", "status"})

    GenerationDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
        Name:    "generation_duration_seconds",
        Help:    "LLM generation latency by provider",
        Buckets: []float64{1, 2, 5, 10, 20, 30, 60},
    }, []string{"provider", "model"})

    TokenUsage = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "llm_token_total",
        Help: "Total LLM token usage",
    }, []string{"provider", "model", "type"}) // type = input/output

    LLMCost = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "llm_cost_usd_total",
        Help: "Total LLM cost in USD",
    }, []string{"provider"})

    QueueDepth = promauto.NewGaugeVec(prometheus.GaugeOpts{
        Name: "river_queue_depth",
        Help: "Number of jobs waiting in each River queue",
    }, []string{"queue"})

    CacheHitRate = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "cache_total",
        Help: "Cache hits vs misses",
    }, []string{"cache", "result"}) // result = hit/miss

    CircuitBreakerState = promauto.NewGaugeVec(prometheus.GaugeOpts{
        Name: "circuit_breaker_state",
        Help: "Circuit breaker state: 0=closed, 1=open, 2=half-open",
    }, []string{"provider"})

    ValidationFailures = promauto.NewCounter(prometheus.CounterOpts{
        Name: "validation_failure_total",
        Help: "Total number of validation failures",
    })
)
```

### 13.3 Distributed Tracing (OpenTelemetry)

```go
// adapter/observability/tracing.go
func InitTracing(serviceName string) func() {
    exp, err := otlptracehttp.New(context.Background(),
        otlptracehttp.WithEndpoint("tempo:4318"),
        otlptracehttp.WithInsecure(),
    )
    if err != nil {
        log.Fatalf("failed to create OTLP exporter: %v", err)
    }

    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exp),
        sdktrace.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceNameKey.String(serviceName),
        )),
    )
    otel.SetTracerProvider(tp)
    return func() { tp.Shutdown(context.Background()) }
}
```

**Trace Spans:**

```
Request: POST /api/v1/stories/{id}/nodes/{nid}/generate
  ├── handler: validate input
  ├── usecase: orchestrate generation
  │   ├── compiler: compile context
  │   │   ├── db: GetNode
  │   │   ├── db: GetCharacterLatest (x N)
  │   │   ├── db: GetLocationLatest
  │   │   ├── db: SearchLoreByTags
  │   │   ├── cache: check (context)
  │   │   └── cache: set (context)
  │   ├── repository: CreateGeneration
  │   ├── queue: Insert (River)
  │   └── cache: set (context)
  │
  └── response: 202 Accepted

Worker: GenerateScene
  ├── compiler: compile prompt params
  │   ├── db: GetCharacterLatest (x N)
  │   ├── db: GetLocationLatest
  │   ├── db: GetStatesForNode
  │   ├── db: GetSummaryByLevel
  │   └── cache: check (context, prompt)
  ├── llm: GenerateScene
  │   ├── router: route to provider
  │   ├── circuitbreaker: check
  │   ├── anthropic: POST /v1/messages
  │   └── cache: set (prompt)
  └── db: UpdateGenerationOutput
```

### 13.4 Dashboards

**Grafana panels:**
- **LLM Cost**: Daily/weekly/monthly spend by provider
- **Latency**: P50/P95/P99 by endpoint, by provider
- **Error Rate**: LLM failures, validation failures, circuit breaker state
- **Queue Health**: Depth, processing rate, latency per queue
- **Cache**: Hit rate, memory usage, eviction rate
- **Saturation**: CPU, memory, goroutines, DB connections

---

## 14. Kubernetes Design

### 14.1 Deployment Architecture

```yaml
# deployment.yaml - API Server
apiVersion: apps/v1
kind: Deployment
metadata:
  name: story-builder-api
spec:
  replicas: 3
  selector:
    matchLabels:
      app: story-builder
      tier: api
  template:
    metadata:
      labels:
        app: story-builder
        tier: api
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "2112"
    spec:
      containers:
      - name: api
        image: story-builder:latest
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 9090
          name: grpc
        - containerPort: 2112
          name: metrics
        env:
        - name: PORT
          value: "8080"
        - name: GRPC_PORT
          value: "9090"
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: db-credentials
              key: url
        - name: ANTHROPIC_API_KEY
          valueFrom:
            secretKeyRef:
              name: anthropic
              key: api-key
        - name: REDIS_URL
          valueFrom:
            secretKeyRef:
              name: redis
              key: url
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
        startupProbe:
          httpGet:
            path: /healthz
            port: 8080
          initialDelaySeconds: 3
          periodSeconds: 5
          failureThreshold: 30

---
# worker.yaml - River Workers
apiVersion: apps/v1
kind: Deployment
metadata:
  name: story-builder-workers
spec:
  replicas: 2
  selector:
    matchLabels:
      app: story-builder
      tier: worker
  template:
    metadata:
      labels:
        app: story-builder
        tier: worker
    spec:
      containers:
      - name: worker
        image: story-builder:latest
        command: ["/app/worker"]
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: db-credentials
              key: url
        - name: ANTHROPIC_API_KEY
          valueFrom:
            secretKeyRef:
              name: anthropic
              key: api-key
        - name: REDIS_URL
          valueFrom:
            secretKeyRef:
              name: redis
              key: url
        resources:
          requests:
            cpu: 1000m
            memory: 1Gi
          limits:
            cpu: 4000m
            memory: 2Gi
```

### 14.2 Horizontal Pod Autoscaler

```yaml
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
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: story-builder-workers-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: story-builder-workers
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Pods
    pods:
      metric:
        name: river_queue_depth
      target:
        type: AverageValue
        averageValue: 100
```

### 14.3 Pod Disruption Budget

```yaml
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
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: story-builder-workers-pdb
spec:
  minAvailable: 1
  selector:
    matchLabels:
      app: story-builder
      tier: worker
```

### 14.4 Service Monitors (Prometheus Operator)

```yaml
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
  - port: metrics
    interval: 15s
```

---

## 15. Security Design

### 15.1 Authentication & Authorization

Current state: **None** — all endpoints are public.

**Recommended (Phase 5):**

```go
// adapter/auth/middleware.go
type Authenticator interface {
    Authenticate(ctx context.Context, token string) (*User, error)
}

func AuthMiddleware(authenticator Authenticator) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            token := extractBearerToken(r)
            user, err := authenticator.Authenticate(r.Context(), token)
            if err != nil {
                http.Error(w, "unauthorized", http.StatusUnauthorized)
                return
            }
            ctx := context.WithValue(r.Context(), userKey, user)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

**Multi-tenancy:**
```go
// Add tenant isolation at repository level
type StoryRepository struct {
    q *db.Queries
}

func (r *StoryRepository) List(ctx context.Context) ([]Story, error) {
    user := GetUser(ctx)
    // Stories are inherently scoped — no cross-tenant access
    return r.q.ListStories(ctx)
}
```

### 15.2 Secret Management

- API keys → Kubernetes Secrets (or Vault for enterprise)
- DB credentials → Kubernetes Secrets with rotation
- No secrets in environment variables at rest (use `env-from` with secrets)

### 15.3 Prompt Injection Protection

```go
// adapter/llm/sanitizer.go
func SanitizePrompt(input string) string {
    // Strip XML tags that could break formatting
    input = stripXMLTags(input, []string{"canon", "character", "location"})

    // Limit length
    if len(input) > 2000 {
        input = input[:2000]
    }

    // Remove control characters
    input = strings.Map(func(r rune) rune {
        if r < 32 && r != '\n' && r != '\t' {
            return -1
        }
        return r
    }, input)

    return input
}
```

### 15.4 Abuse Protection

```go
// Rate limiting per user/IP for LLM endpoints
// 10 generations per minute per user
// 100 generations per hour per user
// Implemented via Redis RateLimiter (Section 7.7)

// Budget enforcement
type GenerationBudget struct {
    DailyLimit   int // max generations per day
    MonthlyLimit int // max generations per month
    CostLimit    float64 // max USD per month
}
```

---

## 16. Reliability Patterns

### 16.1 Current Risks

| Risk | Location | Impact | Fix |
|------|----------|--------|-----|
| No circuit breaker | LLM router | Cascading failures on provider outage | Circuit breaker per provider |
| No retry budget | Generation flow | Infinite retries under failure | Max 3 retries, DLQ after |
| No dead letter queue | River workers | Failed jobs block queues | River DLQ (built-in) |
| No idempotency | Generation | Duplicate generations on retry | Idempotency key (genID) |
| No timeout on DB | All DB calls | Hanging requests | pgxpool.MaxConns + context timeouts |
| No graceful degradation | All services | Full outage on dependency failure | Fallbacks, cached defaults |

### 16.2 Circuit Breaker (Implemented in LLM Layer)

```go
type CircuitBreakerState int

const (
    StateClosed   CircuitBreakerState = 0
    StateOpen     CircuitBreakerState = 1
    StateHalfOpen CircuitBreakerState = 2
)

type CircuitBreaker struct {
    mu           sync.Mutex
    state        CircuitBreakerState
    failureCount int
    threshold    int           // open after this many consecutive failures
    timeout      time.Duration // time before transitioning to half-open
    lastFailure  time.Time
}
```

### 16.3 Retry Policies

```go
// Exponential backoff with jitter
type RetryPolicy struct {
    MaxAttempts int
    BaseDelay   time.Duration
    MaxDelay    time.Duration
}

func (p *RetryPolicy) Backoff(attempt int) time.Duration {
    delay := p.BaseDelay * (1 << attempt) // exponential
    jitter := time.Duration(rand.Int63n(int64(delay / 4))) // ±25%
    delay += jitter
    if delay > p.MaxDelay {
        delay = p.MaxDelay
    }
    return delay
}

// Usage:
var DefaultRetry = &RetryPolicy{
    MaxAttempts: 3,
    BaseDelay:   100 * time.Millisecond,
    MaxDelay:    5 * time.Second,
}
```

### 16.4 Dead Letter Queue

River supports this natively via `river.Job[].Errors` and `MaxAttempts`. Once `MaxAttempts` is exceeded, the job moves to the `river_jobs` table with state `discarded`.

**Monitor DLQ:**
```sql
SELECT COUNT(*) FROM river_job WHERE state = 'discarded' AND queue = 'generate';
```

### 16.5 Idempotency

Every generation has a unique `GenerationID` (UUID). The `CreateGeneration` query uses `INSERT ... ON CONFLICT DO NOTHING` pattern:

```go
func (r *GenerationRepository) Create(ctx context.Context, g *Generation) error {
    // If a retry causes duplicate insert, it's safe
    _, err := r.q.CreateGeneration(ctx, db.CreateGenerationParams{
        ID: toUUID(g.ID),
        NodeID: toUUID(g.NodeID),
        ContextHash: g.ContextHash,
        PromptSnapshot: g.PromptSnapshot,
    })
    return err
}
```

### 16.6 Graceful Degradation

```go
func (s *StoryService) GetStory(ctx context.Context, id uuid.UUID) (*Story, error) {
    story, err := s.storyRepo.Get(ctx, id)
    if err != nil {
        // If DB is down, serve from cache
        cached, cacheErr := s.cache.GetStory(ctx, id)
        if cacheErr == nil {
            slog.Warn("serving story from cache (DB unavailable)", "story_id", id)
            return cached, nil
        }
        return nil, err // both DB and cache failed
    }
    // Cache for next time (async)
    go s.cache.SetStory(context.Background(), story)
    return story, nil
}
```

---

## 17. Testing Strategy

### 17.1 Testing Pyramid

```
          ╱╲
         ╱  ╲          E2E / Smoke (5%)
        ╱    ╲
       ╱──────╲
      ╱        ╲       Integration / Contract (15%)
     ╱          ╲
    ╱────────────╲
   ╱              ╲    Unit Tests (80%)
  ╱                ╲
 ╱──────────────────╲
```

### 17.2 Unit Tests

**Target: 80%+ coverage of domain logic.**

```go
// domain/generation/service_test.go
func TestAcceptGeneration_RejectsOthers(t *testing.T) {
    // Arrange
    repo := &mockGenerationRepository{}
    svc := NewGenerationService(repo, nil)

    repo.On("Accept", genID1).Return(nil)
    repo.On("RejectOthers", nodeID, genID1).Return(nil)

    // Act
    err := svc.AcceptGeneration(ctx, nodeID, genID1)

    // Assert
    assert.NoError(t, err)
    repo.AssertExpectations(t)
}
```

### 17.3 Integration Tests

**Repository tests with testcontainers:**

```go
func TestStoryRepository(t *testing.T) {
    ctx := context.Background()
    pgContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: testcontainers.ContainerRequest{
            Image: "postgres:17-alpine",
            Env:   map[string]string{"POSTGRES_PASSWORD": "test"},
            ExposedPorts: []string{"5432/tcp"},
        },
    })
    // ... run migrations, create repository, test CRUD
}
```

### 17.4 Contract Tests

Adapters must satisfy domain interfaces:

```go
// adapter/repository/postgres/story_repo_test.go
// This is a contract test — ensures Postgres adapter satisfies StoryRepository
func TestPostgresStoryRepository_Contract(t *testing.T) {
    repo := newPostgresStoryRepository(t)
    RunStoryRepositoryTests(t, repo) // shared test suite from domain
}

// adapter/llm/anthropic_test.go
func TestAnthropicAdapter_Contract(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping LLM integration test")
    }
    adapter := NewAnthropicClient(os.Getenv("ANTHROPIC_API_KEY"))
    // Run LLM contract tests
}
```

### 17.5 Worker Tests

```go
func TestGenerateSceneWorker(t *testing.T) {
    // Arrange — mock all dependencies
    proseSvc := &mockProseService{}
    proseSvc.On("GenerateScene", mock.Anything, mock.Anything).Return(&llm.CompletionResponse{Content: "scene text"}, nil)

    worker := &GenerateSceneWorker{
        Prose:   proseSvc,
        Queries: &mockQueries{},
    }

    // Act
    err := worker.Work(ctx, &river.Job[GenerateSceneArgs]{Args: args})

    // Assert
    assert.NoError(t, err)
}
```

### 17.6 Load Tests (k6)

```javascript
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
    stages: [
        { duration: '2m', target: 50 },  // ramp up
        { duration: '5m', target: 50 },  // stay
        { duration: '2m', target: 0 },   // ramp down
    ],
    thresholds: {
        http_req_duration: ['p(95)<5000'], // 95% of requests under 5s
    },
};

export default function() {
    const res = http.get('http://localhost:8080/api/v1/healthz');
    check(res, { 'status is 200': (r) => r.status === 200 });
    sleep(1);
}
```

### 17.7 Chaos Tests (Phase 8)

```go
// Inject failures into LLM adapter to verify circuit breaker behavior
func TestCircuitBreaker_OpensOnFailure(t *testing.T) {
    provider := &failingProvider{}
    cb := NewCircuitBreaker(3, 5*time.Second)
    router := NewFallbackRouter([]ModelProvider{provider}, cb)

    // 3 failures should open the circuit
    for i := 0; i < 3; i++ {
        _, err := router.Generate(ctx, prompt)
        assert.Error(t, err)
    }

    // 4th call should fail fast (circuit open)
    _, err := router.Generate(ctx, prompt)
    assert.ErrorIs(t, err, ErrCircuitOpen)
}
```

---

## 18. CI/CD Pipeline

### 18.1 GitHub Actions

```yaml
name: story-builder

on:
  push:
    branches: [main]
  pull_request:

env:
  GO_VERSION: '1.26'

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
      with:
        go-version: ${{ env.GO_VERSION }}
        cache: true
    - name: golangci-lint
      uses: golangci/golangci-lint-action@v6
      with:
        args: --timeout=5m
    - name: gosec
      uses: securego/gosec@master
      with:
        args: ./...

  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:17-alpine
        env:
          POSTGRES_PASSWORD: test
        ports:
          - 5432:5432
      redis:
        image: redis:7-alpine
        ports:
          - 6379:6379

    steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
      with:
        go-version: ${{ env.GO_VERSION }}
        cache: true

    - name: Unit Tests
      run: go test -short -race -coverprofile=coverage.out ./...

    - name: Integration Tests
      run: go test -run Integration -race ./...
      env:
        DATABASE_URL: postgres://postgres:test@localhost:5432/postgres?sslmode=disable
        REDIS_URL: redis://localhost:6379

    - name: Upload Coverage
      uses: codecov/codecov-action@v4
      with:
        file: coverage.out

  security:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    - name: Trivy Scan
      uses: aquasecurity/trivy-action@master
      with:
        scan-type: 'fs'
        severity: 'HIGH,CRITICAL'
    - name: Dependency Check
      run: |
        go list -m all | nancy sleuth

  build:
    runs-on: ubuntu-latest
    needs: [lint, test, security]
    steps:
    - uses: actions/checkout@v4
    - name: Build
      run: go build ./...
    - name: Docker Build
      run: docker build -t story-builder:${{ github.sha }} .
    - name: Push
      if: github.ref == 'refs/heads/main'
      run: |
        docker tag story-builder:${{ github.sha }} ghcr.io/org/story-builder:latest
        docker push ghcr.io/org/story-builder:latest
```

---

## 19. Performance Review

### 19.1 Current Hotspots

| Area | Issue | Impact |
|------|-------|--------|
| Two-phase context comp | DB queried twice for same data | 2x DB load per generation |
| No prompt cache | Identical prompts re-sent to LLM | $, latency |
| No context cache | Context rebuilt every request | DB load, latency |
| JSONB deserialization | `_ = json.Unmarshal` in hot paths | Latency, allocs |
| sqlc reflection | Every query result requires pgtype conversions | Allocation overhead |
| MemoryStore maps | No concurrency protection | Data races, crashes |

### 19.2 Allocation Profile (Estimated)

Top allocations per request:
1. JSONB unmarshal for character traits (5 per character, ~2KB each)
2. JSONB unmarshal for character state (per character per node)
3. Prompt building string concat (multiple XML blocks, ~5-15KB)
4. SHA256 hash of entire context (full JSON marshal of CompiledContext)

### 19.3 Optimization Plan

```go
// 1. Pool CompiledContext to reduce GC pressure
var compiledContextPool = sync.Pool{
    New: func() interface{} { return &CompiledContext{} },
}

func (s *CompilerService) CompileSceneContext(ctx context.Context, nodeID uuid.UUID) (*CompiledContext, error) {
    cc := compiledContextPool.Get().(*CompiledContext)
    defer compiledContextPool.Put(cc)
    cc.Reset() // clear fields
    // ... populate ...
    return cc.Clone() // return a copy (or nil if pool is reused)
}

// 2. Use strings.Builder instead of += for prompt assembly
func (cc *CompiledContext) BuildSceneProseSystemPrompt() string {
    var b strings.Builder
    b.Grow(8192) // pre-allocate typical size
    b.WriteString("<canon>\n")
    // ...
    return b.String()
}

// 3. Batch DB queries where possible
type BatchLoader struct {
    charCache map[uuid.UUID]*Character
    locCache  map[uuid.UUID]*Location
}

func (l *BatchLoader) LoadCharacters(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*Character, error) {
    // Single query: SELECT * FROM characters WHERE id = ANY($1)
    // Instead of N queries: SELECT * FROM characters WHERE id = $1
}
```

### 19.4 Benchmark Plan

```go
func BenchmarkCompileContext(b *testing.B) {
    svc := newCompilerService()
    for i := 0; i < b.N; i++ {
        _, err := svc.CompileSceneContext(ctx, testNodeID)
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkHash(b *testing.B) {
    cc := testCompiledContext()
    for i := 0; i < b.N; i++ {
        cc.Hash()
    }
}

func BenchmarkBuildPrompt(b *testing.B) {
    cc := testCompiledContext()
    for i := 0; i < b.N; i++ {
        cc.BuildSceneProseSystemPrompt()
        cc.BuildSceneProseUserMessage()
    }
}
```

---

## 20. Refactoring Roadmap

### Phase 3 — Service Decoupling (3-4 weeks) ← YOU ARE HERE

**Goals:** Create domain boundaries, separate repositories, introduce event-driven pipeline

| Step | Description | Files | Risk |
|------|-------------|-------|------|
| 3.1 | Extract `domain/` package with pure interfaces (no infra imports) | New: `domain/story/`, `domain/canon/`, etc. | Medium — API surface change |
| 3.2 | Create repository interfaces in each domain | New: 6 repository interfaces | Low |
| 3.3 | Implement Postgres adapters for repositories | New: `adapter/repository/postgres/` | Medium — 38 queries to adapt |
| 3.4 | Create `adapter/cache/` no-op implementation (interface only) | New: `adapter/cache/noop.go` | Low |
| 3.5 | Refactor River workers to use domain interfaces (no direct `w.Queries.*`) | Modify: `internal/river/jobs.go` | High — core change |
| 3.6 | Refactor HTTP handlers to use `app/usecase/` layer | New: `app/story/usecase.go`, etc. | Medium |
| 3.7 | Remove duplicate interfaces (`canon/*` service interfaces) | Remove: `canon/models.go` service interfaces | Medium |
| 3.8 | Bootstrap rewrite | New: `bootstrap/` | Medium |

**Acceptance Criteria:**
- `domain/` packages have zero imports from infra (`db`, `river`, `grpc`, `http`)
- All River workers use domain interfaces, not `db.Queries`
- No duplicate interface definitions
- Build passes, tests pass

### Phase 4 — Redis Introduction (2 weeks)

| Step | Description |
|------|-------------|
| 4.1 | Add Redis client to bootstrap |
| 4.2 | Implement `ContextCache` |
| 4.3 | Implement `PromptCache` |
| 4.4 | Add distributed lock for generation |
| 4.5 | Wire cache into CompilerService |
| 4.6 | Add cache hit/miss metrics |

### Phase 5 — Observability (2 weeks)

| Step | Description |
|------|-------------|
| 5.1 | Add structured logging (slog) across all entry points |
| 5.2 | Add Prometheus metrics (HTTP, generation, queue, cache) |
| 5.3 | Add OpenTelemetry tracing |
| 5.4 | Create Grafana dashboards |
| 5.5 | Add health/readiness endpoints |

### Phase 6 — Reliability (2 weeks)

| Step | Description |
|------|-------------|
| 6.1 | Add circuit breaker to LLM router |
| 6.2 | Add LLM cost tracking |
| 6.3 | Add rate limiting |
| 6.4 | Add dead letter queue monitoring |
| 6.5 | Add graceful degradation for DB/cache |

### Phase 7 — Graph Architecture (1-2 weeks)

| Step | Description |
|------|-------------|
| 7.1 | Optimize PostgreSQL graph queries with indexes |
| 7.2 | Add materialized view for topology |
| 7.3 | Evaluate Neo4j for canon graphs |
| 7.4 | Performance test with 10K-node stories |

### Phase 8 — Scale Readiness (2-3 weeks)

| Step | Description |
|------|-------------|
| 8.1 | Add DB connection pooling (PgBouncer) |
| 8.2 | Add read replicas for reporting queries |
| 8.3 | Implement tenant isolation (row-level security) |
| 8.4 | Load test to 10K concurrent users |
| 8.5 | Profile and optimize top 5 allocation sources |

### Phase 9 — Microservice Readiness (optional, 4-6 weeks)

| Step | Description |
|------|-------------|
| 9.1 | Identify first candidate for extraction (likely Canon service) |
| 9.2 | Extract to separate binary with gRPC interface |
| 9.3 | Add service mesh (Istio/Linkerd) |
| 9.4 | Migrate to event-driven communication (NATS/Kafka) |
| 9.5 | Add saga orchestration for multi-service transactions |

---

## Appendix A: Migration Safety

### Runtime Migration Strategy

Current: `migrate.Run()` at startup (single-instance, locks DB)

Target (Kubernetes):

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: story-builder-migrate
  annotations:
    argocd.argoproj.io/hook: PreSync
spec:
  template:
    spec:
      restartPolicy: Never
      containers:
      - name: migrate
        image: story-builder:latest
        command: ["/app/migrate"]
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: db-credentials
              key: url
```

### Rollback Strategy

- All migrations must be reversible (`DOWN` migration)
- River job queue allows replay of failed jobs
- Context cache TTL means stale data expires automatically
- Prompt cache can be flushed with `redis-cli FLUSHDB`

---

## Appendix B: Cost Analysis

### Monthly LLM Cost Projection (Anthropic)

| Tier | Gens/Day | Tokens/Gen | Cost/Day | Cost/Month |
|------|----------|------------|----------|------------|
| Dev | 100 | 4K in / 1K out | ~$6.75 | ~$200 |
| S (1K users) | 1,000 | 4K in / 1K out | ~$67.50 | ~$2,000 |
| M (10K users) | 10,000 | 4K in / 1K out | ~$675 | ~$20,000 |
| L (100K users) | 50,000 | 4K in / 1K out | ~$3,375 | ~$100,000 |

**With Prompt Cache (30% hit rate):**
| Tier | Savings | Net Cost/Month |
|------|---------|----------------|
| S | ~$600 | ~$1,400 |
| M | ~$6,000 | ~$14,000 |
| L | ~$30,000 | ~$70,000 |

### Infrastructure Cost

| Component | S (1K users) | M (10K users) | L (100K users) |
|-----------|-------------|---------------|----------------|
| PostgreSQL | $50 (2 GB) | $200 (8 GB + replica) | $1,000 (32 GB + replicas) |
| Redis | $15 (1 GB) | $50 (4 GB) | $200 (16 GB cluster) |
| Kubernetes | $100 (3 nodes) | $500 (5 nodes) | $2,000 (10 nodes) |
| **Total** | **~$165/mo** | **~$750/mo** | **~$3,200/mo** |

---

## Appendix C: Key Technical Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Modular monolith vs microservices | Modular monolith (Phase 3-7) | Lower operational complexity, faster iteration. Microservices only at Phase 8 when team size > 15 and traffic justifies it. |
| Graph DB (Neo4j) | Postpone to Phase 7 | PostgreSQL handles current graph workload. Measure first, optimize second, migrate third. |
| MongoDB | Never | JSONB + Redis covers every use case. |
| Event bus | River through Phase 6, Redis Streams Phase 7 | River provides transactional outbox with PG. Redis Streams for higher throughput at Phase 7+. |
| LLM provider abstraction | Circuit-breaking fallback chain | Sonnet → Haiku → Local fallback ensures no hard dependency on any single provider. |
| Cache strategy | Redis read-through | Write-through would add latency; read-through with TTL is simpler and sufficient. |
| Migration strategy | Helm pre-sync job | ArgoCD PreSync hook ensures migrations run before new pods start. |
<!-- -end- -->
