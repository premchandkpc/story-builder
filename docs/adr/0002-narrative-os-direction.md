# ADR 0002: Narrative OS — Architectural Direction

## Status

Accepted

## Context

The project started as a story graph editor with LLM prose generation. After multiple iterations (nodes→scenes, append-only canon, multi-agent turns, River jobs, Redis caching), the architecture has evolved beyond a simple "generate chapter" tool.

The current model — Story → Chapter → Scene — reflects a book structure. But the platform is increasingly used for branching narratives, character simulation, timeline management, and multi-format output (prose, visual novel, screenplay, RPG, anime).

Continuing to model everything as "a story with chapters" will:
1. Force-fit non-linear formats into a linear book structure
2. Make character state management increasingly fragile
3. Block multi-format output (screenplay, game, anime, comic)
4. Prevent timeline branching, parallel scenes, and alternate universes
5. Create a god `story` service that violates bounded contexts

## Decision

Adopt a **Narrative Operating System** architecture. The canonical data model shifts from:

```text
Story → Chapter → Scene
```

to:

```text
Universe → World → Timeline → Story → Scenario → Scene → Frame
```

This is not an incremental refactor. It is a multi-phase evolution. Phase 0 (current state) is acknowledged as a valid intermediate.

**Critical ordering principle:** Domain first, infrastructure later. Kafka (#11), Neo4j (#10), and Qdrant (#9) come after the core narrative engine works. Most projects invert this — they build a distributed system that generates beautifully inconsistent nonsense.

### Phase 0 — Current (shipping)
- Story → Chapter → Scene hierarchy
- Append-only character/location versioning
- River job queue for generation pipeline
- Redis for cache + rate limiting
- In-memory fallback for all services

### Phase 1 — Domain Cleanup + Core Domains
- Clean package boundaries: `internal/story/`, `internal/character/`, `internal/scene/`, `internal/timeline/`, `internal/prompt/`, `internal/generation/`
- Eliminate `common`, `utils`, `shared`, `helpers`, `misc` garbage-dump packages
- Split Character into **CharacterDefinition** (immutable core), **CharacterState** (per-scene, event-sourced), **CharacterMemory** (future, vector)
- Build Scene graph with parent/child/parallel/reusable support
- Define concrete Go types for all aggregates (Story, Chapter, Scenario, Scene, SceneEdge, TimelineEvent)

### Phase 2 — Prompt Compiler ✅ Built
Highest-leverage new component. Hierarchical prompt layering implemented in `internal/prompt/`:

```go
// compiler.go
type CompilerService struct {
    store Store
}
func (s *CompilerService) Compile(req *CompileRequest, templateName string) (*CompiledPrompt, error)

// models.go
type CompileRequest struct {
    StoryID         uuid.UUID
    ChapterID       uuid.UUID
    SceneID         uuid.UUID
    CharacterID     uuid.UUID
    StoryPrompt     string
    ChapterPrompt   string
    ScenePrompt     string
    CharacterPrompt string
    CulturePrompt   string
    MemoryContext   string
}

type CompiledPrompt struct {
    System         string
    User           string
    Model          llm.ModelTier
    Temperature    float64
    MaxTokens      int
    LayersApplied  []LayerID
}
```

Layer order: Global → Culture → Story → Memory → Chapter → Scenario → Character → Scene → Frame → Safety.
Merge strategies: `override`, `merge`, `append`, `replace`, `disable`.
Default templates in `store.go`: scene_prose, state_extract, summary_update, canon_validate.
Events: publishes `evPromptCompiled` via event bus.

### Phase 3 — Timeline Engine ✅ Built
Before vector DB, before agents, before graph DB — timeline first. Built on top of `event.Store` + `event.Bus`.

```go
// engine.go
type Engine struct {
    store      event.Store
    bus        event.Bus
    branches   map[BranchID]*Branch
    assignments map[uuid.UUID]BranchID
}

type Branch struct {
    ID          BranchID
    Name        string
    ParentID    BranchID
    ForkPoint   int
    MergedInto  BranchID
    IsAlternate bool
}

func (e *Engine) Past(storyID uuid.UUID, upToOrder int) ([]SceneRef, error)
func (e *Engine) Future(storyID uuid.UUID, afterOrder int) ([]SceneRef, error)
func (e *Engine) CreateBranch(storyID uuid.UUID, name string, forkPoint int, isAlternate bool) (BranchID, error)
func (e *Engine) ForkFrom(parentID BranchID, name string) (BranchID, error)
func (e *Engine) MergeBranch(sourceID, targetID BranchID) error
func (e *Engine) BranchScenes(storyID uuid.UUID, branchID BranchID) ([]SceneRef, error)
```

Branch types: alternate, parallel, forked. Scene N state reconstructable from `event.EvTimelineUpdated` events 1..N-1.

### Phase 4 — Character State Engine

```json
{
  "scene": "20",
  "character": "hero",
  "health": 75,
  "emotion": "anger",
  "stress": 45,
  "outfit": "armor",
  "inventory": ["sword"],
  "relationships": {"princess": "loves"}
}
```

Without this, LLMs hallucinate outfit changes every scene. State snapshots per (story, character, scene).

### Phase 5 — Scene Graph + Validation Engine
- Scene parent/child/parallel/alternate edges
- 4 validators: Character (dead speaking, age mismatch), Timeline (ordering, overlap), Lore (canon contradiction), Dialogue (wrong culture/emotion)

### Phase 6 — MongoDB Migration
Only after domains are stable. Move document-heavy entities:
- Prompt templates, character profiles, scene templates, world definitions, culture definitions
- Keep transactional data (stories, chapters, users) in Postgres

### Phase 7 — Redis Caching Layer
Cache compiled prompts, character state, scene state, story context.
Key pattern: `story:{id}`, `scene:{id}`, `character:{id}`.

### Phase 8 — Qdrant Vector Memory
Collections: `character_memory`, `story_memory`, `scene_memory`, `dialog_memory`.
Solves: "Why does hero hate villain?" → semantic retrieval across the entire narrative.

### Phase 9 — Neo4j Relationship Graph
Only when JSONB relationship queries become painful.
Graph: Character↔Character, Character↔Scene, Scene↔Scene.

### Phase 10 — Kafka Event Bus
Events: `StoryCreated`, `SceneCreated`, `SceneGenerated`, `CharacterUpdated`, `EmotionChanged`, `TimelineUpdated`.
Consumers: Analytics, Memory, Search, Rendering.

### Phase 11 — Multi-Agent Orchestration
Agents: Story Planner, Scene Planner, Dialogue Writer, Emotion Validator, Continuity Validator.
Pipeline: Planner → Scene Generator → Dialogue Generator → Emotion Check → Timeline Check → Lore Check → Store.

### Phase 12 — Rendering Pipeline
Input: scene ID. Output: camera, lighting, character positions, objects, background.
Format-independent. Consumable by Stable Diffusion, Flux, Sora, Unreal, Unity.

## Key Principles

1. **Canon is law.** The canonical model (Universe → World → Timeline → Story → Scenario → Scene → Frame) is the source of truth. All output formats derive from it.

2. **Event sourcing for state.** Character state, timeline, and scene state are event-sourced. No mutable in-memory state across service boundaries.

3. **Prompt layering.** Prompts inherit hierarchically (global ← story ← chapter ← scene ← character). Each layer can override, merge, append, replace, or disable.

4. **Separate actor from character.** Like movies. Actors have physical attributes, availability, suitability scores. Characters have personality, arc, voice. Casting bridges them.

5. **Culture as a first-class concept.** Culture affects dialogue, gesture, clothing, social norms, expressions — without changing the underlying narrative beat.

6. **Modular monolith first.** All services start in-process. Only extract to separate processes when load demands it.

## Consequences

### Positive
- One canonical model supports all output formats
- Character state becomes reliable (event-sourced, not mutated in place)
- Timeline consistency is guaranteed
- Culture-aware generation is built-in, not bolted on
- Relationship queries become trivial (Neo4j)
- Memory retrieval becomes semantic (Qdrant)
- Engineers can work on bounded contexts without touching the god service

### Negative
- Significant migration effort (current DB schema must coexist during transition)
- Team must learn MongoDB, Neo4j, Qdrant, Kafka
- Event sourcing adds complexity for simple CRUD operations
- Prompt layering adds latency to generation pipeline
- Culture engine requires domain expertise to define culture definitions

### Neutral
- Modular monolith means we can defer microservice operational costs
- In-memory services remain for development/testing (degrade gracefully)
- Current River workers map directly to future Kafka consumers

## Implementation Notes

- **Domain before infrastructure.** Kafka (#11), Neo4j (#10), Qdrant (#9) come last. The real hard problem is: "Can character A remain emotionally consistent through 200 scenes across multiple timelines?" Solve that first; databases are implementation details.
- All new entities go into new tables/collections first. Existing tables are migrated in-place via SQL.
- ✅ Prompt compiler built at `internal/prompt/` — 10 layers, 5 merge strategies, in-memory template store with default templates. CompilerService + MemoryStore. Next: wire into River worker pipeline.
- ✅ Timeline engine built at `internal/timeline/engine.go` — event-sourced on top of `event.Store`/`event.Bus`, branch/fork/merge/past/future support. Next: connect character state reconstruction from timeline events.
- Character state tracking (Phase 4) is the minimum viable fix for LLM hallucination. Without per-scene state snapshots, outfits, emotions, and relationships change randomly.
- MongoDB comes at Phase 6 — *after* domains are stable. Don't add MongoDB while the data model is still in flux.
- The emotion engine becomes simple once character state is event-sourced. With event sourcing, inner/displayed/suppressed emotion is just a query across recent state deltas.
- Current River workers map directly to future Kafka consumers — no need to rip them out during migration.
- In-memory services remain for development/testing; they degrade gracefully when infrastructure is unavailable.

### Anti-patterns to avoid

- **Shiny technology first.** Don't add Kafka, Neo4j, Qdrant, MongoDB, or agents until the domain demands it. A distributed system that generates inconsistent nonsense is worse than a monolith that works.
- **God `Character` struct.** Never embed everything in one type. Split into CharacterDefinition (immutable) / CharacterState (per-scene) / CharacterMemory (vector).
- **`common`/`utils`/`shared` packages.** These become garbage dumps. Each domain gets its own package with clear boundaries.
- **Worker direct SQL access.** River workers should call service interfaces, not raw queries. Enforce this from Phase 1.
- **Duplicated prompt compilation.** Every scene should flow through a single PromptCompiler. No ad-hoc prompt building in handlers.
- **Synchronous service coupling.** Design for events even in the monolith. Use Go channels or an in-memory event bus until Kafka arrives.
