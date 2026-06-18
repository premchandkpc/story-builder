# Service Layer

## Overview

Services live in `internal/service/`. Each service contains business logic and depends on repository interfaces (not MongoDB directly).

```
internal/
  service/
    generation/     Generation pipeline orchestration
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
| `Create(storyID, name, description, props)` | Creates a new location |
| `Get(id)` | Gets location by ID |
| `ListByStory(storyID)` | Lists all locations in a story |
| `Update(id, name, description, props)` | Updates a location |
| `DeleteByStory(storyID)` | Deletes all locations for a story |
| `GetByName(storyID, name)` | Finds location by name within a story |

### Memory Service

| Method | Description |
|---|---|
| `StoreMemory(storyID, charID, sceneID, content, importance)` | Stores a memory with embedding |
| `RetrieveMemories(charID, query, limit)` | Vector search of relevant memories |
| `ListMemories(charID)` | Lists all memories for a character |

### Timeline Service

| Method | Description |
|---|---|
| `CreateEvent(storyID, sceneID, title, description, order)` | Creates a timeline event |
| `ListEvents(storyID)` | Lists all events sorted by order |
| `GetEventsByRange(storyID, from, to)` | Gets events in a range |

### Summary Service

| Method | Description |
|---|---|
| `UpsertSceneSummary(storyID, sceneID, content)` | Creates/updates scene summary |
| `UpsertStorySummary(storyID, content)` | Creates/updates story-level summary |
| `GetSceneSummary(storyID, sceneID)` | Gets scene summary |
| `GetSummaryByLevel(storyID, level)` | Gets latest summary at a level |

---

## Generation Service

Orchestrates the generation pipeline:

```go
type GenerationService struct {
    llm       ProseService
    extract   ExtractionService
    mem       MemoryService
    timeline  TimelineService
    summary   SummaryService
    validate  ValidationService
    charRepo  CharacterRepository
    locRepo   LocationRepository
}
```

### Pipeline

```
GenerateScene
    → buildPromptParams fetches: characters, character states, locations, story summary
    → compiles rich PromptParams for LLM
    → calls ProseService.GenerateScene (claude-sonnet)
    → stores generation in Mongo
    → spawns pipeline goroutine:

        ExtractState (local-7b via Ollama)
            → extracts state deltas from scene text
            → appends to character_state collection

        MemoryUpdate
            → creates character memories from state changes
            → stores with embeddings in character_memories

        TimelineUpdate
            → creates timeline event for the scene

        SummaryUpdate
            → updates scene-level summary

        ValidateCanon
            → validates scene against canon (claude-haiku)
            → checks: character consistency, timeline, lore, dialogue
            → stores validation result
```

Each pipeline step is a goroutine. Workers are simple structs in `internal/worker/` — no River, no Kafka.

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
