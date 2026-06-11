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
│  │  - Logger     │  │  (12 handlers)   │  │  (6 job types)   │  │
│  │  - Recoverer  │  │  ─────────────── │  │  ───────────────  │  │
│  │  - RequestID  │  │  Character/Actor │  │  GenerateScene    │  │
│  │  - CORS       │  │  Location/Trait  │  │  ExtractState     │  │
│  └──────────────┘  │  Lore/Casting    │  │  UpdateSummary    │  │
│                    │  Story/Node      │  │  MergeBranches    │  │
│                    │  Generation      │  │  ValidateScene    │  │
│                    │  Scene/Summary   │  │  GenerateStory    │  │
│                    │  StoryGenerator  │  └───────────────────┘  │
│                    └────────┬─────────┘                         │
│                             │                                    │
│                    ┌────────▼─────────┐                         │
│                    │  DB Service Layer │                         │
│                    │  (db.Queries)    │                         │
│                    └────────┬─────────┘                         │
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
    ├── internal/api       ─── HTTP handlers + service interfaces
    │   ├── dbservices_*.go ─── Postgres-backed service implementations
    │   └── router.go      ─── chi route definitions
    │
    ├── internal/graph     ─── DAG data model + traversal algorithms
    │   ├── models.go      ─── Story, Node, Edge, SceneStructure
    │   ├── traversal.go   ─── TopologicalSort, IdentifyBranches
    │   └── memory.go      ─── In-memory GraphService implementation
    │
    ├── internal/canon     ─── Versioned canon (Characters, Locations, Lore)
    │   ├── models.go      ─── Domain types + service interfaces
    │   └── memory.go      ─── In-memory implementations
    │
    ├── internal/ledger    ─── CharacterState per (story, char, node)
    │   ├── models.go      ─── CharacterState, StateDelta, LedgerService
    │   └── memory.go      ─── In-memory LedgerService implementation
    │
    ├── internal/compiler  ─── CompiledContext + SHA256 hash for prompts
    │   ├── compiler.go    ─── CompiledContext, Hash(), Generation types
    │   ├── prompts.go     ─── System prompt builders for all 6 prompts
    │   ├── summary.go     ─── SummaryLevel, StorySummary, SummaryService
    │   └── memory.go      ─── In-memory GenerationService implementation
    │
    ├── internal/llm       ─── LLM client + service interfaces
    │   ├── types.go       ─── ModelTier, PromptRegistry, 6 service ifaces
    │   ├── services.go    ─── Service implementations (Prose, Extract, etc.)
    │   └── client.go      ─── Anthropic + Ollama HTTP clients
    │
    ├── internal/scene     ─── Multi-agent scene system
    │   ├── types.go       ─── SceneTurn, SceneService, AgentPromptInput
    │   ├── turn.go        ─── WhoActsNext — turn scheduling logic
    │   └── agent.go       ─── BuildAgentPrompt — character prompt builder
    │
    ├── internal/river     ─── River async job types + workers
    │   ├── jobs.go        ─── 6 job types + worker implementations
    │
    ├── internal/db        ─── sqlc-generated DB layer
    │   ├── db.go          ─── DBTX interface + Queries struct
    │   ├── models.go      ─── Generated Go structs per table/view
    │   ├── queries.sql.go ─── 38 generated query methods
    │   └── extras.go      ─── UpdateGenerationOutput helper
    │
    └── internal/migrate   ─── SQL migration runner
        └── runner.go      ─── _migrations table, apply/pending
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
    │ 4. Create generation row (accepted=false)
    │ 5. Enqueue GenerateSceneWorker
    │
    ▼
river.GenerateSceneWorker.Work()
    │
    │ 1. Re-compile prompt params (char cards, location, lore, state, summary)
    │ 2. Call ProseService.GenerateScene(params)
    │    → ProseServiceImpl builds CompiledContext → system prompt + user message
    │    → LLMClient.Complete() → API call to Anthropic/Ollama
    │ 3. Update generation output (UPDATE generations SET output = ...)
    │
    ▼
api.GenerationHandler.AcceptGeneration()
    │
    │ 1. Mark generation as accepted
    │ 2. Reject other generations for this node
    │ 3. Enqueue: ExtractStateWorker → UpdateSummaryWorker → MergeBranchesWorker
    │
    ▼
river.ExtractStateWorker  →  LLM extracts state deltas from scene text
    │
    ▼
river.UpdateSummaryWorker →  LLM updates scene-level summary
    │
    ▼
river.MergeBranchesWorker →  LLM merges parallel branch summaries at join nodes
```

## Data Flow: Story Generation

```
User POST /api/v1/stories/generate { synopsis }
    │
    ▼
api.StoryGeneratorHandler.Generate()
    │
    │ 1. Enqueue GenerateStoryWorker
    │
    ▼
river.GenerateStoryWorker.Work()
    │
    │ 1. LLM generates StoryOutline (title, characters, beats, edges)
    │ 2. Create characters from outline
    │ 3. Create nodes (beats) from outline
    │ 4. Create edges connecting beats
    │
    ▼
Story is fully outlined in DB
```

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

## LLM Client Selection

`cmd/server/main.go:198` — `createLLMClient`:
- If `ANTHROPIC_API_KEY` is set → `AnthropicClient`
- Otherwise → `OllamaClient` (default `http://localhost:11434`)

## River Queue Configuration

| Queue | Max Workers | Job Types |
|---|---|---|
| `generate` | 2 | GenerateSceneWorker |
| `extract` | 4 | ExtractStateWorker |
| `merge` | 2 | MergeBranchesWorker |
| `validate` | 1 | ValidateSceneWorker |
| `default` | 0 (unlimited) | GenerateStoryWorker |

## Dual Mode: DB vs In-Memory

When Postgres is unavailable (e.g., during development without Docker):
- All services fall back to in-memory stores (thread-safe with `sync.RWMutex`)
- River workers are not started
- LLM calls still work (Anthropic/Ollama clients are independent of DB)
- Full CRUD functionality is preserved, but limited to process lifetime
