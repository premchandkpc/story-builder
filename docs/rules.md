# Code Rules

## Architecture

1. **MongoDB is the single source of truth.** Redis is never a source of truth (cache, rate limits, locks only).
2. **No PostgreSQL, Kafka, Qdrant, or additional infrastructure** unless a measured bottleneck proves necessary.
3. **Business logic only in services.** Data access only in repositories. Handlers remain thin.
4. **Use dependency injection.** Wire everything in `main.go`.
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
  repository/        Interfaces + mongo/ implementation
  worker/            Async pipeline workers (goroutines)
  graph/             DAG traversal + validation
  llm/               LLM clients + router
  prompt/            Prompt compiler (10 layers)
  cache/             Redis cache + rate limiter
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

## DAG Rules

1. **Validate before generation.** Every generation request must pass `ValidateDAG()` first.
2. **Detect cycles.** `TopologicalSort()` returns error on cycle — block the operation.
3. **No orphan scenes.** Every scene (except root) must have at least one incoming edge.
4. **No dead ends** for main path (optional for choice branches).

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
6. **Inline styles only** (no CSS modules, no Tailwind). Style objects live in the component file or in `api/types.ts` for shared styles.
7. **`memo()` on React Flow custom nodes.** Performance optimization — prevents re-render of nodes whose data hasn't changed.
8. **`useCallback` for handlers passed to child components.** Prevents unnecessary re-renders.
9. **`useMemo` for derived data.** Only recompute when dependencies change.
10. **Dark theme throughout.** Background `#0f172a`, text `#e2e8f0`, cards `#1e293b`.

## Testing Priority

| Area | What to test |
|---|---|
| Graph | Topological sort, cycle detection, branch detection |
| Memory | State changes, memory retrieval, importance ranking |
| Generation | Pipeline execution, context compilation |
| Validation | All 4 validators |
| API | Minimal — business logic belongs in services |

## Making Changes

1. **Update docs first** (or in the same PR). Every structural change updates the relevant `.md` files.
2. **Cascade delete story.** `StoryService.Delete` calls `StoryCascadeDeleter.cascade()` which deletes child collections (character_state, memories, generations, summaries, timeline, edges, scenes, characters) in order before deleting the story document.
3. **No test suite yet.** Run the server and curl endpoints to verify.
3. **Docker Compose for local dev.** `docker compose up -d mongo redis ollama` for minimal infra.
