# Code Rules

## Architecture

1. **MongoDB is the single source of truth.** Redis is never a source of truth (cache, rate limits, locks only).
2. **No PostgreSQL, Kafka, Qdrant, or additional infrastructure** unless a measured bottleneck proves necessary.
3. **Business logic only in services.** Data access only in repositories. Handlers remain thin.
4. **Use dependency injection.** Wire everything in `cmd/server/` (`main.go` + `init.go`).
5. **Use `context.Context` everywhere.** No global state.
6. **Depend on interfaces, not implementations.** All repositories defined as interfaces; MongoDB is an implementation detail.

## Schema

1. **No schema migrations.** MongoDB is schemaless. Add fields to documents freely.
2. **Indexes are code.** Define indexes in `internal/repository/mongo/client.go`. Created on startup. Conflicts are logged as WARN and skipped — not fatal.
3. **Append-only for state.** Never overwrite character state. Always append a new document.
4. **Character definitions are versioned immutably.** Each update creates a new document with a new `_id`, same `charId`, and incremented `version`. Use `GetLatest` for the current version.
5. **Embed when co-accessed.** Store `participants` in scenes. Don't store DAG children in scenes (use `scene_edges` collection).

## Code Structure

```
internal/
  api/               HTTP handlers (thin)
  domain/            Domain models (no infra imports)
  service/           Business logic
  agents/            Runtime narrative agents (orchestrator + 10 agents)
  scene/             Scene turn orchestration
  repository/        Interfaces + mongo/ implementation
  worker/            Async pipeline workers (goroutines)
  graph/             DAG traversal + validation
  llm/               LLM clients + router
  prompt/            Prompt compiler (10 layers)
  cache/             Redis cache + rate limiter
  events/            In-memory event bus
  log/               Structured logging (slog wrapper)
  config/            Environment config
```

## Go Conventions

1. **One file per major type** in domain packages (e.g., `story.go`, `scene.go`, `edge.go`).
2. **Repository interfaces** in `internal/repository/`; MongoDB implementations in `internal/repository/mongo/`.
3. **No `common`, `utils`, `shared`, `helpers` packages.** Each concern gets its own package.
4. **No global variables** except `main()` wiring and package-level constants.
5. **Errors are values.** Define domain-specific error types. Wrap errors with context.
6. **Log 5xx errors server-side.** `writeError` in handlers logs 5xx responses via `slog.Error`. Do not log 4xx client errors.

## State Machine

1. **Stories have strict status transitions:** `draft → active → completed → archived`. The `CanTransitionTo()` method enforces valid transitions in the service layer. An archived story cannot be re-activated.
2. **Scenes have strict status transitions:** `draft → generated → accepted → stale`. Stale scenes may be regenerated (`stale → generated`). Direct status assignment via the API is validated against the current status. Acceptance atomically sets `scene.acceptedGenerationId` + `scene.status = accepted`.
3. **API handlers never set status directly.** Status changes go through service-layer validation. The generation pipeline and acceptance flow are the only paths that move scene status forward.

## DAG Rules

1. **Validate before generation.** Every generation request must pass `ValidateDAG()` first.
2. **Detect cycles.** `TopologicalSort()` returns error on cycle — block the operation.
3. **No orphan scenes.** Every scene (except root) must have at least one incoming edge.
4. **No dead ends** for main path (optional for choice branches).

## Agents

1. **Agents are in-process actors**, not microservices. All agents run in the same Go process via the orchestrator.
2. **Agent registry is the single source of truth** for agent specs. Register all agents at startup in `init.go`.
3. **One agent = one role.** A Character agent role-plays one character. Director plans. Narrator narrates. Don't combine roles.
4. **Turn serialization.** The orchestrator runs turns sequentially. One blocking turn at a time. No parallel agent execution in P0.
5. **Agent context is immutable within a turn.** The context is assembled before the first turn. Agents receive a snapshot, not a live connection to the DB.
6. **Required agents must succeed or the scene fails.** Optional agents can fail silently.
7. **Canon is updated on scene accept, not on generation.** `CanonDelta` documents are accumulated during generation; `stories.canonPins` is only updated when the user accepts the generation.

## LLM Pipeline

1. **Canon is law.** The scene text, character cards, and world rules are immutable within a generation context.
2. **State before generation.** Always compile character state + memories before calling the LLM.
3. **Validate after generation.** Every generated scene passes through the 4 validators (Character, Timeline, Lore, Dialogue).
4. **Cache aggressively.** Identical prompts hit Redis cache. Only unique context hits the LLM.

## Workers

1. **Simple goroutines.** No River, no Kafka, no message queue. Workers are structs with a `Work(ctx, arg)` method.
2. **Context cancellation.** All worker goroutines respect context cancellation.
3. **Error isolation.** One worker failure doesn't stop the pipeline. Errors are logged and stored in the generation document.

## Frontend Conventions

1. **All data fetching uses TanStack React Query.** No raw `fetch()` calls in components. Use hooks from `api/hooks.ts`.
2. **API client is the only file that calls `fetch()`.** Every HTTP request goes through `api/client.ts`.
3. **One custom hook per logical query.** Hooks encapsulate query keys, cache invalidation, and navigation side effects.
4. **Props are typed with interfaces.** Every component defines or imports a Props interface.
5. **No prop drilling beyond 2 levels.** Use React Router's params or React Query's cache for shared state.
6. **Inline styles + CSS utility classes** (no CSS modules, no Tailwind). Style objects live in the component file or `api/types.ts`. Interaction utilities (`.card-hover`, `.btn-press`) and entrance animations (`.stagger-fade-in`, `.stagger-slide-up`) are in `index.css`.
7. **All border-radius via `--radius-*` tokens** — never raw px values. All transitions via `--transition-*` tokens.
8. **`memo()` on React Flow custom nodes.** Performance optimization — prevents re-render of nodes whose data hasn't changed.
9. **`useCallback` for handlers passed to child components.** Prevents unnecessary re-renders.
10. **`useMemo` for derived data.** Only recompute when dependencies change.
11. **Warm dark theme throughout.** Background `#1a1512`, text `#f5f0e8`, surfaces `#2a2420`, borders `#3d3530`.
12. **Optimistic updates for all mutations.** Use TanStack React Query pattern: `onMutate` saves snapshot + sets optimistic data, returns snapshot, `onError` rolls back, `onSettled` invalidates. Use `setToastFns` bridge for toasts from outside component tree.
13. **Generation polling uses `setInterval` + `queryClient.invalidateQueries`** (not `refetchInterval`). Only poll while explicitly enabled (node selected + pending gens). 2s interval.
14. **Four error handling layers:** (1) network/API errors → parse status code → specific message, (2) React Error Boundary with fallback UI, (3) every `useQuery` handles isLoading/isError/empty, (4) client-side form validation before API call.
15. **`memo()` on all list items** — not just React Flow nodes. Every item returned from `.map()` should be `memo()`'d.
16. **Position drag rollback.** Capture `nodePositionsRef` on drag start, revert via `setNodes` on save failure, show toast "Failed to save position. Node snapped back."

## Testing Priority

| Area | What to test |
|---|---|
| Graph | Topological sort, cycle detection, branch detection |
| Memory | State changes, memory retrieval, importance ranking |
| Generation | Pipeline execution, context compilation, partial success, retries |
| Validation | All 4 validators |
| Bible | Generation via LLM, structured output validation, single-flight guard, cascade delete |
| Chapter | CRUD operations, act/chapter ordering, cascade delete, scene array management |
| API | Minimal — business logic belongs in services |

## Making Changes

1. **Update docs first** (or in the same PR). Every structural change updates the relevant `.md` files.
2. **Cascade delete story.** `StoryService.Delete` calls `StoryCascadeDeleter.cascade()` which deletes child collections (bibles, chapters, character_state, memories, generations, summaries, timeline, edges, scenes, characters) in order before deleting the story document.
3. **Tests live as Go unit tests.** Run `go test ./...` to verify. No test runners beyond the standard toolchain.
4. **Docker Compose for local dev.** `docker compose up -d mongo redis opencode` for minimal infra.
