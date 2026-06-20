# Service Layer

## Overview

Services live in `internal/service/`. Each service contains business logic and depends on repository interfaces (not MongoDB directly).

```
internal/
  service/
    bible.go        Story Bible CRUD + LLM generation
    chapter.go      Chapter CRUD (Acts→Chapters→Scenes)
    context.go      ContextBuilder (Bible + states + memories + timeline → prompt)
    generation.go   Generation pipeline orchestration (durable, background goroutine)
    validation/     Canon/timeline/character validation
    graph/          DAG traversal + validation utilities
```

Domain models live in `internal/domain/` (no infrastructure dependencies).

Repository interfaces live in `internal/repository/` with MongoDB implementations in `internal/repository/mongo/`.

---

## Handler → Service Pattern

Each API handler depends on a narrow service interface. This enables:
1. Loose coupling — handlers only call what they need
2. Easy testing — mock implementations satisfy the interface
3. Single implementation backed by MongoDB

```go
type StoryHandler struct {
    Service interface {
        Create(ctx context.Context, title string) (*domain.Story, error)
        Get(ctx context.Context, id string) (*domain.Story, error)
        Update(ctx context.Context, id, title string) (*domain.Story, error)
        List(ctx context.Context) ([]domain.Story, error)
    }
}
```

---

## Domain Services

### Story Service

| Method | Description |
|---|---|
| `Create(title)` | Creates a new story document |
| `Get(id)` | Gets story by ID |
| `Update(id, title)` | Updates story title |
| `List()` | Lists all stories |
| `Delete(id)` | Deletes story and all associated data |

### Scene Service

| Method | Description |
|---|---|
| `Create(storyID, ...)` | Creates a scene node in the DAG |
| `Get(id)` | Gets scene by ID |
| `Update(id, ...)` | Updates scene metadata |
| `List(storyID)` | Lists all scenes in a story |
| `Delete(id)` | Deletes scene and edges |
| `SetStructure(id, structure)` | Sets multi-agent scene structure |
| `GetStructure(id)` | Gets scene structure |

### Edge Service

| Method | Description |
|---|---|
| `Create(storyID, from, to, type)` | Creates a directed edge |
| `List(storyID)` | Lists all edges for a story |
| `ListFrom(sceneID)` | Lists outgoing edges |
| `ListTo(sceneID)` | Lists incoming edges |
| `Delete(storyID, from, to)` | Removes an edge |

### Character Service

| Method | Description |
|---|---|
| `Create(storyID, name, ...)` | Creates a version-1 character definition (immutable log) |
| `Get(id)` | Gets character by document ID (specific version) |
| `GetLatest(charID)` | Gets latest version by logical character ID |
| `Update(character)` | Creates new versioned document (immutable append) |
| `List(storyID)` | Lists characters in a story |
| `UpdateState(characterID, sceneID, state)` | Appends a state snapshot |
| `GetState(characterID, sceneID)` | Gets state at a specific scene |
| `GetStateHistory(characterID)` | Gets all state changes (event-sourced) |

### Location Service

| Method | Description |
|---|---|
| `Create(storyID, name, description, props)` | Creates a new location (supports hierarchy via locType, parentId) |
| `Get(id)` | Gets location by ID |
| `ListByStory(storyID)` | Lists all locations in a story |
| `Update(id, name, description, props)` | Updates a location |
| `DeleteByStory(storyID)` | Deletes all locations for a story |
| `GetByName(storyID, name)` | Finds location by name within a story |
| `GetChildren(storyID, parentID)` | Lists direct children of a location |
| `GetAncestors(storyID, locID)` | Walks parent chain up to root dimension |

### Memory Service

| Method | Description |
|---|---|---|
| `ListByCharacter(charID)` | Lists all memories for a character |
| `Search(storyID, charID, query, limit)` | Vector search of relevant memories via embedding |

### Timeline Service

| Method | Description |
|---|---|---|
| `Create(ctx, event)` | Creates a timeline event |
| `List(ctx, storyID)` | Lists all events sorted by order |

### Summary Service

| Method | Description |
|---|---|---|
| `GetByLevel(storyID, level)` | Gets latest summary at a level |
| `GetSceneSummary(storyID, sceneID)` | Gets scene summary |

### Bible Service

| Method | Description |
|---|---|
| `Generate(ctx, storyID)` | Generates Bible via LLM (claude-sonnet, 0.3 temp, 8192 tokens) and persists. Single-flight guard — concurrent calls return error. |
| `Get(ctx, storyID)` | Gets Bible for a story |
| `Update(ctx, storyID, bible)` | Updates Bible fields |
| `DeleteByStory(ctx, storyID)` | Removes Bible when story is deleted |

### Chapter Service

| Method | Description |
|---|---|
| `Create(ctx, chapter)` | Creates a chapter via domain.Chapter object |
| `Get(ctx, storyID, chapterID)` | Gets chapter by ID within a story |
| `List(ctx, storyID)` | Lists all chapters for a story, sorted by act+chapter |
| `Update(ctx, chapter)` | Updates chapter metadata via domain.Chapter object |
| `Delete(ctx, storyID, chapterID)` | Deletes a chapter |

### Context Builder

`internal/service/context.go` — Assembles the full narrative context for scene generation.

```
ContextBuilder.Build(ctx, storyID, scene)
    → Loads Bible (world rules, magic, factions, cultures, tone)
    → Loads all latest character states for participants
    → Loads location hierarchy (parent chain for scene location)
    → Loads top-10 memories per participant (sorted by importance)
    → Loads recent timeline events (last 20)
    → Loads story-level + scene-level summaries
    → Loads blueprint (acts, plot threads, theme)
    → Returns BuiltContext with:
        → llm.PromptParams (10-layer compiled prompt)
        → CanonXML (for validation step)
        → CharStateXML (for extraction step)
        → BranchSummary (for merge step)
```

The output is approximately 20k tokens of structured context. Bible is included as system prompt layers (Culture→Story→World layers). Character states and memories go into the Character prompt layer.

---

## Generation Service

Orchestrates the durable generation pipeline:

```go
type GenerationService struct {
    llm        ProseService
    extract    ExtractionService
    mem        MemoryService
    timeline   TimelineService
    summary    SummaryService
    validate   ValidationService
    charRepo   CharacterRepository
    locRepo    LocationRepository
    contextBldr ContextBuilder
}
```

### Pipeline

Uses `context.Background()` so pipeline completion is independent of HTTP request lifetime.

```
GenerateScene
    → sets Generation.Status = "running" before goroutine starts
    → spawns pipeline goroutine with background context (5-min timeout):

        step 1: Generate (critical, 3× retry)
            → ContextBuilder.Build(storyID, scene)
            → ProseService.GenerateScene(builtContext)  (claude-sonnet, 0.8)
            → stores generation in Mongo

        step 2: ExtractState (critical, 3× retry)
            → ExtractStateWorker (local-7b via Ollama, temp 0)
            → extracts state deltas from generated scene text
            → appends to character_state collection

        step 3: MemoryUpdate (non-critical, best-effort)
            → MemoryUpdateWorker
            → creates character memories from state changes
            → stores with embeddings in character_memories

        step 4: TimelineUpdate (non-critical, best-effort)
            → TimelineWorker
            → creates timeline event for the scene

        step 5: SummaryUpdate (non-critical, best-effort)
            → SummaryWorker (local-7b, 0.2)
            → updates scene-level summary

        step 6: ValidateCanon (non-critical, best-effort)
            → ValidationWorker (claude-haiku, temp 0)
            → validates scene against canon
            → checks: character consistency, timeline, lore, dialogue
            → stores validation result

    → final status:
        all critical steps succeeded + all non-critical succeeded => "success"
        all critical steps succeeded + some non-critical failed  => "partial_success"
        any critical step failed after 3 retries                 => "failed"
```

Each pipeline step runs via `runStep` (critical, 3× retry with exponential backoff) or `runNonCriticalStep` (best-effort, logged on failure). Workers are simple structs in `internal/worker/` — no River, no Kafka.

On failure, `Generation.Error` is set and `Generation.Status` reflects the outcome. Status is queryable via `GET /api/v1/generations/{id}` (returns `Status`, `Error`, `UpdatedAt` fields) and via the existing SSE progress endpoint.

---

## Validation Service

| Method | Description |
|---|---|
| `ValidateScene(scene)` | Runs all validators against a generated scene |
| `ValidateCharacter(char, scene)` | Character consistency (alive, in-scene, in-character) |
| `ValidateTimeline(scene, timeline)` | No ordering violations |
| `ValidateLore(scene, lore)` | No world-rule contradictions |
| `ValidateDialogue(scene, characters)` | Dialogue matches character voice |

---

## Repository Pattern

```go
type StoryRepository interface {
    Create(ctx context.Context, story Story) error
    Get(ctx context.Context, id string) (*Story, error)
    Update(ctx context.Context, story Story) error
    List(ctx context.Context) ([]Story, error)
    Delete(ctx context.Context, id string) error
}

type CharacterRepository interface {
    Create(ctx context.Context, c Character) error
    Get(ctx context.Context, id string) (*Character, error)
    ListByStory(ctx context.Context, storyID string) ([]Character, error)
    AppendState(ctx context.Context, state CharacterState) error
}
```

All repositories depend on interfaces, not `*mongo.Collection` directly. MongoDB implementations live in `internal/repository/mongo/`.

---

## Frontend Service Layer

### API Client (`web/src/api/client.ts`)

A single `request<T>()` generic function wraps all HTTP calls with:
- Automatic JSON headers
- Configurable timeout (default 30s) via `AbortController`
- HTTP error → thrown `Error`
- 204 No Content → `undefined`

```typescript
async function request<T>(path: string, init?: RequestInit & { timeout?: number }): Promise<T>
```

The exported `api` object groups endpoints by domain (stories, chapters, scenes, nodes, edges, generations, etc.), each returning typed promises.

### React Query Hooks (`web/src/api/hooks.ts`)

Custom hooks that wrap `api.*` calls with TanStack React Query caching:

| Hook | Type | Description |
|---|---|---|
| `useStories()` | Query | Fetch all stories |
| `useChapters(storyId)` | Query | Fetch chapters for a story |
| `useCreateChapter(storyId)` | Mutation | Create chapter + invalidate cache |
| `useScenes(storyId, chapterId)` | Query | Fetch scenes for a chapter |
| `useStoryNodeStats(storyId)` | Query | Compute node status counts |
| `useAllStoryStats(stories)` | Query | Parallel stats for all stories |
| `useCreateStory()` | Mutation | Create story + navigate to it |
| `useGenerateTitle()` | Mutation | LLM title generation |
| `useGenerateStory()` | Mutation | Full LLM story generation |

**Pattern:** Queries use `useQuery` with explicit `queryKey` arrays for cache scoping. Mutations use `useMutation` with `onSuccess` invalidating related query caches. Some mutations also navigate via `useNavigate`.

### Component Services

The frontend has no separate service layer — business logic lives in:
- **Custom hooks** (`api/hooks.ts`) — data fetching + cache management
- **Component state** (`useState`) — UI state (search query, form values, selected node)
- **Derived data** (`useMemo`) — computed values (filtered story list, status colors)

## Cache Service

Optional Redis (via `REDIS_ADDR` env var). Degrades gracefully.

| Component | Purpose |
|---|---|
| `PromptCache` | Caches LLM responses keyed by prompt hash (TTL: 1h) |
| `SlidingWindowRateLimiter` | Rate limits LLM API calls per provider |
| `DistLock` | Prevents duplicate scene generation |

---

## Agent Orchestrator

The agent orchestrator in `internal/agents/orchestrator.go` manages multi-agent scene generation:

### AgentRegistry

```go
type AgentRegistry struct {
    agents map[string]AgentSpec
}

func (r *AgentRegistry) Register(spec AgentSpec)
func (r *AgentRegistry) Get(name string) (AgentSpec, bool)
func (r *AgentRegistry) List() []AgentSpec
```

### Orchestrator

| Method | Description |
|--------|-------------|
| `Plan(scene)` | Returns turn order based on `scene.FlowType` |
| `Execute(plan, agentContext)` | Runs each agent in sequence, recording turns |
| `RunFinish(scene, agentContext)` | Runs StateExtract + Critic + Director after all turns |

### Integration with GenerationService

```go
// In runPipeline:
if scene.SceneStructure != nil {
    plan := orchestrator.Plan(scene)
    result := orchestrator.Execute(plan, agentContext)
    orchestrator.RunFinish(scene, agentContext)
} else {
    // existing 6-worker pipeline
}
```

### Agent Context

Each agent receives an `AgentContext` with story, scene, characters, states, bible, memories, timeline, canon deltas, and summaries. The context is assembled by the orchestrator before the first turn and refreshed between phases.

---

## Workers

Workers in `internal/worker/` are goroutine-based. Each implements a `Work` method:

| Worker | Model | Purpose |
|---|---|---|
| `GenerateSceneWorker` | claude-sonnet | Prose generation |
| `ExtractStateWorker` | local-7b | State delta extraction |
| `MemoryUpdateWorker` | local-7b | Memory creation |
| `TimelineWorker` | none | Timeline event recording |
| `SummaryWorker` | local-7b | Summary update |
| `ValidationWorker` | claude-haiku | Canon validation |

No River, no message queue. Workers are launched as goroutines with context cancellation.
