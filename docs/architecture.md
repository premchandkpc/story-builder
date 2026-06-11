# Architecture

```
web/ (React Flow)
  |  proxies /api -> Go :8080
  v
cmd/server/main.go
  |  chi router @ /api/v1
  |  DB detection -> pgxpool or in-memory
  |  migrate runner (migrations/*.sql)
  |  River async worker pool
  v
internal/
  api/        HTTP handlers + chi router
  canon/      Versioned entities (Character, Location, Lore)
  graph/      DAG types (Story, Node, Edge) + traversal (Kahn's)
  ledger/     CharacterState per node (location, knows, mood, etc.)
  compiler/   CompiledContext + Hash() + 5 prompt builders
  llm/        ModelTier enum + service interfaces + PromptRegistry
  river/      River job types (GenerateScene, ExtractState, etc.)
  db/         sqlc-generated query methods
  migrate/    Flyway-compatible migration runner
```

## Package dependency graph

```
api -> canon, graph, compiler, db
compiler -> canon, ledger
river -> (standalone, uses riverqueue)
llm -> (standalone)
db -> sqlc-generated, uses pgx + pgvector
graph -> (standalone)
ledger -> (standalone)
canon -> (standalone)
migrate -> pgx
```

## Data flow (end-to-end)

```
User (React Flow)
  |
  | POST /api/v1/stories/:id/nodes/:id/generate
  v
chi handler -> GenerationHandler.Generate
  |
  | (future) inserts River job
  v
River GenerateSceneWorker
  |
  | compiler.Compile() -> CompiledContext
  | Hash() -> SHA256 of canonical JSON
  | Compare with stored context_hash (staleness detection)
  v
LLM ProseService.GenerateScene() (Sonnet 0.8)
  |
  | output -> generation row
  | insert River ExtractState job
  v
LLM ExtractionService.ExtractState() (local-7b, temp=0)
  |
  | deltas -> ledger.ApplyDeltas()
  | insert River UpdateSummary job
  v
LLM SummaryService.UpdateSummary() (local-7b, temp=0.2)
  |
  v
MergeBranchesWorker (if fork/join detected)
ValidateSceneWorker (continuity check)
```

## Pipeline model routing

| Step                | Model         | Temp | Tool use |
|---------------------|---------------|------|----------|
| Generate prose      | Sonnet         | 0.8  | no       |
| Extract state       | local 7B       | 0    | yes      |
| Update summary      | local 7B       | 0.2  | no       |
| Join/merge branches | Haiku          | 0.2  | no       |
| Validate canon      | Haiku          | 0    | no       |

## Canon versioning

- Characters and Locations: never UPDATE in place. Each edit = new version row.
- `Story.CanonPins` = map of `{entity_type, entity_id, version}` for pinning specific versions.
- `CompiledContext.Hash()` = SHA256 of canonical JSON → compared against stored hash to detect staleness.
- DB views `latest_characters` and `latest_locations` for latest-version queries.

## State management (ledger)

- `CharacterState` stored per `(story_id, character_id, as_of_node)`.
- State shape: `{location, knows[], does_not_know[], mood, relationships{}, items[]}`.
- `DeriveDoesNotKnow(allFacts, knows)` computes `does_not_know` from global fact set minus known facts.
- Branch state: `GetStateAtBranch(storyID, forkNode, branchNode)` retrieves states for a given branch.

## DAG structure

- Nodes connected by directed edges of types: `seq`, `fork`, `join`, `choice`.
- `TopologicalSort` = Kahn's algorithm.
- `IdentifyBranches` = walks from fork nodes to their corresponding join nodes.
- `BranchCharacterSets` = ensures fork branches have non-overlapping character sets.
