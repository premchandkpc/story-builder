# Story Builder v2 — Execution Roadmap

**Based on 2026-06-23 architecture review. Targets codebase at `internal/`, `web/src/`, `docs/`.**

## Phase Documents

| Phase | Document | Priority |
|-------|----------|----------|
| 1 — Durable Orchestration | `docs/exe-plan/phase-1-durable-orchestration.md` | **P0** |
| 2 — Narrative Events + Projections | `docs/exe-plan/phase-2-narrative-events.md` | **P1** |
| 3 — Test Harness | `docs/exe-plan/phase-3-test-harness.md` | **P1** |
| 4 — Observability + Run Inspector | `docs/exe-plan/phase-4-observability.md` | **P0** |
| 5 — Developer Experience | `docs/exe-plan/phase-5-dx.md` | P2 |
| 6 — Retrieval Layer | `docs/exe-plan/phase-6-retrieval.md` | P2 |
| 7 — Product Intelligence | `docs/exe-plan/phase-7-product-intelligence.md` | P2 |

## What Already Exists (don't rebuild)

| Component | File(s) | Status |
|-----------|---------|--------|
| `domain.Job` + `domain.StoryRun`/`domain.RunStep` | `internal/domain/job.go:5-88` | ✅ Full types with lease, status, step tracking |
| `domain.NarrativeEvent` + 14 event types | `internal/domain/narrative_event.go:5-44` | ✅ Rich model with subject types, confidence, version |
| `GenerationJobWorker` (poll+lease+in-flight+context-hash) | `internal/service/generation_job_worker.go:1-591` | ✅ Durable pipeline, 6 steps, retries, panic recovery, stuck-job recovery |
| `InMemoryBus` (sync pub/sub + wildcard) | `internal/events/events.go:1-78` | ✅ 48 event type constants |
| `RunInspector` React component | `web/src/components/RunInspector.tsx:1-187` | ✅ Basic run/step display with expand/collapse |
| Run API endpoints | `api/runs` `api/narrative-events` | ✅ CRUD + list-by-story/scene + cancel |
| `domain.StoryBlueprint` | `internal/domain/blueprint.go:1-35` | ✅ Acts, CharacterArcs, PlotThreads |
| `domain.CanonDelta` | `internal/domain/canon_delta.go:1-28` | ✅ Append-only canon change log |
| `domain.SceneTurn` | `internal/domain/scene_turn.go` | ✅ Turn-level agent execution records |
| `TokenBudgetService` | `internal/service/token_budget.go` | ✅ Per-story, per-model cumulative tracking |
| Graph `TopologicalSortStrings` | `internal/graph/traversal.go:1-54` | ✅ Kahn's algorithm (needs more operations) |
| Frontend types + API client | `web/src/api/types.ts`, `client.ts` | ✅ 269-line client, 445-line types |
| OpenTelemetry tracing | `internal/trace/` | ✅ Span wrappers + OTLP export |

## What to Build — P0 (Sprints 1-2)

> **Theme**: Stop losing work, make generation inspectable.

### P0.1: Durable Job Runner Package
Create `internal/orchestration/` as the durable runner layer. The current `GenerationJobWorker` in `internal/service/` is monolithic. Extract into:

```
internal/orchestration/
  job_queue.go         — Claim/release/heartbeat logic using domain.Job
  run_recorder.go      — Create StoryRun + RunStep from pipeline execution
  pipeline.go          — Pipeline definition (ordered steps, critical vs non-critical)
  worker_loop.go       — Generic poll+lease+execute loop (replaces generation_job_worker.go)
```

Key behavior changes from current:

| Current | Target |
|---------|--------|
| `Job.LeaseUntil` set once at pick | Heartbeat every 30s while running |
| `genInFlight sync.Map` dedup | Persistent `job_leases` heartbeat in Mongo |
| Steps tracked via `Generation.StepStatus` map | Steps recorded as `RunStep` documents |
| `context.WithTimeout(5min)` | Per-step timeout + step-level cancel |
| No per-step retry config | Per-step retry configurable |
| Pipeline success/failure from final status | Pipeline produces `StoryRun` with run-level status |

### P0.2: Wire StoryRun + RunStep Into Pipeline

Current pipeline flow:
```
Job → GenerationJobWorker.runPipeline()
  → setStepStatus(gen.ID, step, "running"/"done"/"failed")
```

Target flow:
```
Job → PipelineRunner.Execute()
  → Creates StoryRun(status=queued)
  → For each step:
      → Creates RunStep(status=running, startedAt=now)
      → Executes step
      → Updates RunStep(status=done/failed, tokens, model, promptHash, artifacts)
      → Heartbeats lease
  → Updates StoryRun(status=completed/partial/failed, finishedAt=now)
  → Updates Job(status=done/failed)
```

**API changes needed:**

`internal/domain/job.go` — Add heartbeat fields:
```go
type Job struct {
    // ... existing fields ...
    HeartbeatAt *time.Time `bson:"heartbeatAt,omitempty"` // last heartbeat from worker
    Version     int        `bson:"version"`                // optimistic lock for claim
}
```

`internal/repository/interfaces.go` — Add JobRepo methods:
```go
type JobRepository interface {
    // ... existing methods ...
    Heartbeat(ctx context.Context, id string) error
    Claim(ctx context.Context, jobType string, leaseTime time.Duration, workerID string) (*domain.Job, error)
}
```

`internal/domain/run_step.go` (extract from job.go) — Add step-level snapshot for replay:
```go
type RunStepSnapshot struct {
    StepName      string         `bson:"stepName"`
    Status        string         `bson:"status"`
    StartedAt     *time.Time     `bson:"startedAt,omitempty"`
    FinishedAt    *time.Time     `bson:"finishedAt,omitempty"`
    Model         string         `bson:"model,omitempty"`
    PromptHash    string         `bson:"promptHash,omitempty"`
    TokensIn      int            `bson:"tokensIn,omitempty"`
    TokensOut     int            `bson:"tokensOut,omitempty"`
    PromptSnippet string         `bson:"promptSnippet,omitempty"` // first 500 chars for inspection
    OutputSnippet string         `bson:"outputSnippet,omitempty"` // first 500 chars
    Error         string         `bson:"error,omitempty"`
}
```

### P0.3: Run Inspector Frontend — Phase 1

Current `RunInspector` shows runs and steps. Phase 1 adds:

1. **Step timeline visualization** — Gantt-like horizontal bars showing which steps ran, how long they took, which failed
2. **Prompt/Output inspector** — Expandable raw prompt sections with token counts, model used
3. **Context hash display + dedup indicator** — Show when a run was cached via context hash match
4. **Cancel button** — Wire to existing `POST /runs/{id}/cancel`
5. **Filter by status** — queued/running/completed/failed tabs

**Files to modify:**
- `web/src/components/RunInspector.tsx` — Add timeline view, cancel button, filter tabs
- `web/src/api/hooks.ts` — Add `useCancelRun` mutation, `useRunDetails` query with prompt artefacts

### P0.4: Scene-Level Generation Lock

Current `genInFlight sync.Map` is in-memory and per-node. Replace with persistent lock:

```go
type SceneLock struct {
    SceneID    string    `bson:"_id"`
    StoryID    string    `bson:"storyId"`
    GenID      string    `bson:"genId,omitempty"`
    WorkerID   string    `bson:"workerId"`
    AcquiredAt time.Time `bson:"acquiredAt"`
    TTL        time.Time `bson:"ttl"`
}
```

Use Mongo's `FindOneAndUpdate` with `$set` on insert to atomically acquire. Prevent concurrent generation on same scene across workers/restarts.

**Collection:** `scene_locks` in `EnsureIndexes()`.
**TTL index:** `{ ttl: 1 }, expireAfterSeconds: 0` so locks auto-cleanup.

---

## What to Build — P1 (Sprints 3-4)

> **Theme**: State correctness, replayability, test harness.

### P1.1: NarrativeEvent Append-Only Log

`domain.NarrativeEvent` exists. Now wire it into the pipeline:

After `StepExtract` (or agent `RunFinish`), call a new `EventExtractor` that:
1. Reads the generated scene text + extracted character deltas
2. Produces `[]NarrativeEventCandidate` (one per state change)
3. Each candidate goes through the validator gate (P1.2)
4. Accepted events appended to `narrative_events` collection
5. Rejected events logged to `RunStep.Artifacts.rejected_events`

```go
type EventExtractor struct {
    Validator *EventValidator
    Repo      repository.NarrativeEventRepository
}

func (e *EventExtractor) Extract(ctx context.Context, gen *domain.Generation, scene *domain.Scene, states []domain.CharacterState) ([]domain.NarrativeEvent, error)
```

### P1.2: Event Validator Gate

Before any `NarrativeEvent` is appended, validate it:

```go
type EventValidationRule interface {
    Validate(ctx context.Context, event *domain.NarrativeEvent, state *StoryState) *EventViolation
}

type EventViolation struct {
    Severity  string // "reject" | "warn" | "info"
    Reason    string
    EventID   string
}
```

Rules to implement:
- `DeadCharacterCannotAct` — character has `health <= 0` in state → reject
- `TimelineMonotonicity` — scene timestamp cannot precede last locked timeline event → reject
- `LocationConsistency` — character cannot be in two places at same time → warn
- `RelationshipBounds` — trust/respect/fear/affection stays in [0, 100] → clamp + warn
- `DuplicateEvent` — same `(subjectType, subjectId, eventType, sceneId)` already exists → reject

### P1.3: State Projection Views

Build read models that derive from `narrative_events`:

```go
type CharacterView struct {
    CharacterID  string     `bson:"_id"`
    StoryID      string     `bson:"storyId"`
    CurrentState CharacterStateSnapshot
    EventIDs     []string   // event IDs that produced this state
    Version      int64      // projection version
    UpdatedAt    time.Time
}

type TimelineView struct {
    StoryID     string         `bson:"_id"`
    Events      []TimelineEntry
    LastOrder   int
    Version     int64
}
```

**Files:**
- `internal/projection/` — new package
  - `character_view.go` — Rebuild from narrative_events
  - `timeline_view.go` — Rebuild from narrative_events
  - `scheduler.go` — Trigger rebuild after event append
- `internal/repository/interfaces.go` — Add `CharacterViewRepository`, `TimelineViewRepository`

Projections rebuild on read if stale (if `version < latestEvent.version`). For now, rebuild is synchronous on read. In P2, make it async with a notification channel.

### P1.4: Graph Operations — Fill the Gaps

Docs mention `FindBranches`, `FindDeadEnds`, `FindUnreachableScenes`, `ValidateDAG`. Only `TopologicalSortStrings` exists.

Implement in `internal/graph/dag.go`:

```go
func ValidateDAG(scenes []*domain.Scene, edges []*domain.SceneEdge) error
func FindBranches(scenes []*domain.Scene, edges []*domain.SceneEdge) ([][]string, error)
func FindDeadEnds(scenes []*domain.Scene, edges []*domain.SceneEdge) []string
func FindUnreachableScenes(rootSceneID string, edges []*domain.SceneEdge) []string
func FindMergePoints(scenes []*domain.Scene, edges []*domain.SceneEdge) []string
```

### P1.5: Golden Tests for Generation Pipeline

**Directory:** `test/golden/`

**Structure:**
```
test/golden/
  fixtures/
    simple-dialogue/
      story.json           — Story definition
      characters.json      — Character definitions
      scenes/
        scene_1.json        — Scene with participants, beat intent, flow type
      edges.json           — DAG edges
      state.json           — Initial character states
      memories.json        — Initial memories
      bible.json           — World bible
    multi-turn-action/
      ...
    fork-choice-merge/
      ...
  pipeline_test.go        — Test runner
```

**What each test does:**
1. Load fixture from JSON files
2. Insert into test Mongo database
3. Call `GenerationService.Generate(ctx, sceneID)`
4. Assert:
   - Generation status is "success" or "partial_success"
   - Scene has non-empty `GeneratedContent`
   - Character states were extracted (non-empty `character_state` for scene)
   - Timeline event recorded
   - Summary created
   - No cycle detected in graph
   - No duplicate memory writes
   - `StoryRun` + `RunStep` records created

**Mock the LLM** with deterministic responses stored in the fixture (`mocked_outputs/`):
```json
{
  "director": { "content": "...", "decisions": {...} },
  "narrator": { "content": "..." },
  ...
}
```

**Test helper** in `internal/service/generation_test.go` — use `testify/suite` pattern from existing tests.

### P1.6: Property/Invariant Tests

Use `testing/quick` or `github.com/leanovate/gopter` for:

**Graph invariants:**
- DAG remains acyclic after edge insertion → validate
- Deleting node removes all incident edges
- Topological sort of any DAG returns all nodes
- Branch merge rules: fork must eventually join

**Narrative invariants:**
- Scenario property: dead character cannot produce events
- Timeline monotonicity: events append in ascending order
- Relationship bounds: trust ∈ [0, 100]
- Event IDs are unique per generation run

---

## What to Build — P2 (Sprints 5-6)

> **Theme**: Product intelligence, developer experience.

### P2.1: Blueprint → Arc → Thread Planning

`domain.StoryBlueprint` exists. Now make it drive generation.

**New types in `internal/domain/blueprint.go`:**
```go
type ScenePurpose struct {
    SceneID        string   `bson:"sceneId"`
    AdvancingArcs  []string // arc IDs this scene advances
    AdvancingThreads []string // thread IDs this scene advances
    RequiredBeats  []string // beats that must happen
    ForbiddenBeats []string // beats that cannot happen
    EntryState     string   // expected character state before scene
    ExitState      string   // guaranteed character state after
    ConflictBeat   string   // escalation / tension / reversal / revelation / resolution
}

type StoryPlan struct {
    StoryID  string
    Blueprint *StoryBlueprint
    ScenePurposes map[string]*ScenePurpose
}
```

**New methods on GenerationService:**
```go
func (s *GenerationService) PlanScene(ctx context.Context, sceneID string) (*ScenePlan, error)
```

**ScenePlan output:** structured plan used as pre-filled prompt context:
- Which arcs advance
- Which threads advance
- Required beats
- Forbidden contradictions
- Entry/exit states

### P2.2: Scene Plan Panel (Frontend)

New React component `ScenePlanPanel` in GraphPanel tabs.

Shows:
- Arc linkages (checkboxes per arc)
- Thread linkages (checkboxes per thread)
- Required beats (editable list)
- Forbidden beats (editable list)
- Entry/exit state previews

**Files:**
- `web/src/components/ScenePlanPanel.tsx` — New component
- `web/src/components/GraphPanel.tsx` — Add "Plan" tab
- `web/src/api/hooks.ts` — Add `useScenePlan` query/mutation

### P2.3: Scene Version Diff

Backend: Compare two generations for a scene:

```go
type GenDiff struct {
    GenAID           string
    GenBID           string
    ProseDiff        string // line-level diff
    StateChangesA    []NarrativeEvent
    StateChangesB    []NarrativeEvent
    AddedEvents      []NarrativeEvent
    RemovedEvents    []NarrativeEvent
    TimelineDiff     []TimelineDiffEntry
    TokenDiff        TokenDiff
}
```

**Endpoint:** `GET /stories/{id}/nodes/{id}/generations/{a}/diff?against={b}`

Frontend: Enhance `GenerationCompare.tsx` to show event-level diffs alongside prose diff.

### P2.4: Memory Retrieval Service

Current `MemoryRepository.Search` does vector search. Upgrade:

```go
type MemoryRetrievalService struct {
    MemRepo   repository.MemoryRepository
    Embedding llm.EmbeddingService
}
```

**Retrieval pipeline:**
1. Fetch hard constraints (active participants, timeline, unresolved arcs)
2. Fetch semantic memories (top-5 by embedding similarity)
3. Fetch world facts from bible (top-3 by entity match)
4. Rerank by: `relevance * 0.6 + importance * 0.3 + recency * 0.1`

**Configurable per scene:**
```go
type RetrievalConfig struct {
    MaxMemories  int     // default 5
    MinImportance float64 // default 0.3
    RecencyWeight float64 // default 0.1
    RelevanceWeight float64 // default 0.6
    ImportanceWeight float64 // default 0.3
}
```

### P2.5: One-Command Dev + Doctor

**`Makefile` additions:**
```makefile
.PHONY: dev doctor

dev: ## Start full dev environment
	docker compose up -d mongo redis
	@sleep 2
	@echo "Starting Go API (live reload)..."
	@air -c .air.toml &
	@echo "Starting frontend..."
	@cd web && npm run dev &
	@wait

doctor: ## Check system health
	@echo "Checking dependencies..."
	@which go || (echo "❌ Go not found"; exit 1)
	@which node || (echo "❌ Node not found"; exit 1)
	@curl -sf http://localhost:8080/api/v1/healthz > /dev/null && echo "✅ Go API" || echo "❌ Go API"
	@curl -sf http://localhost:8081/actuator/health > /dev/null 2>&1 && echo "✅ Java Analysis" || echo "⚠ Java Analysis (optional)"
	@echo "✅ OK"
```

**`story-builder doctor` CLI command** — standalone Go binary or subcommand at `cmd/doctor/main.go`.

### P2.6: OpenAPI → TypeScript Client

Add `cmd/api-gen/main.go` that:
1. Reads Go handler types + domain structs
2. Generates OpenAPI 3.0 spec
3. Runs `openapi-typescript` to generate TS client

Integration into `Makefile`:
```makefile
generate-api:
	go run ./cmd/api-gen > web/src/api/generated.ts
```

Short-term alternative: shared Zod schemas between frontend/backend with JSONSchema generation.

---

## Dependency Graph

```
P0.1 Durable Job Runner
  └── P0.2 Wire StoryRun/RunStep
       └── P0.3 Run Inspector UI
            └── P4 Observability
P0.4 Scene Locks
  └── P1.1 Narrative Events
       └── P1.2 Event Validator
            └── P1.3 Projection Views
                 └── P6 Retrieval Service
P1.4 Graph Ops
  └── P1.6 Property Tests
P1.5 Golden Tests (independent)
P2.1 Blueprint Planning
  └── P2.2 Scene Plan UI
P2.3 Version Diff
P2.5 DX (independent)
P2.6 API Client (independent)
```

---

## Current Gaps vs Your Review

| Your Suggestion | Existing | Gap |
|----------------|----------|-----|
| Persistent `StoryRun` + `RunStep` | Types exist | Not wired into pipeline |
| Mongo-backed durable job queue | `Job` type + `PickPending` exist | No heartbeat, no version lock, no dead-letter |
| Run cancellation + retry | Cancel endpoint exists | Not wired into worker loop |
| Per-scene generation lock | `sync.Map` dedup | Not persistent |
| Run Inspector UI (basic) | Component exists | No timeline, no prompt inspection |
| `NarrativeEvent` append-only model | Type + 14 constants exist | Not emitted by pipeline |
| Validator gate before applying state | Post-generation validation exists | Not event-aware |
| State projections | None | Need `internal/projection/` |
| Graph invariant tests | 5 tests exist | Need property-based tests |
| Golden tests | Integration tests exist | No deterministic pipeline fixture tests |
| Blueprint → Arc → Thread planning | `StoryBlueprint` type exists | No `ScenePurpose`, no plan generation |
| "Plan Scene" UI | None | Need `ScenePlanPanel` |
| Memory retrieval/reranking | Vector search exists | No reranking service |
| Scene version diff | `GenerationCompare` shows prose | Need structured event diff |
| `make dev` / doctor | Docker compose exists | No single command |
| OpenAPI TS client | Manual types exist | No generation |
| Run Inspector → agent forensics | Basic display | No prompt sections, no step timeline |

---

## How to Start

1. Read `internal/service/generation_job_worker.go` — understand current 591-line pipeline
2. Read `internal/domain/job.go` — understand existing Job/StoryRun/RunStep types
3. Create `internal/orchestration/pipeline.go` with the extracted step runner
4. Create `internal/orchestration/job_queue.go` with heartbeat + version lock
5. Wire StoryRun/RunStep creation into pipeline execution
6. Update `RunInspector.tsx` to show step timeline

## Key Insight

You already have most of the **types** you need (`Job`, `StoryRun`, `RunStep`, `NarrativeEvent`, `StoryBlueprint`). The gap is between "type exists" and "wired into pipeline." Phase 1 and Phase 2 are primarily about wiring, not about building new abstractions. The hardest engineering work is in Phase 3 (test harness) because it requires deterministic fixture infrastructure for a stochastic system.
