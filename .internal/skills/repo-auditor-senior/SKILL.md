# Repo Auditor (Senior)

Audit story-builder architecture, boundaries, and drift.

## Trigger
- "audit the repo"
- "review architecture"
- "find architectural drift"
- "check package boundaries"

## Scope

Audit these boundaries:

### domain isolation
- No repository calls from domain types
- No LLM imports in domain
- No service imports in handlers (only via interfaces)

### service/repo separation
- Services depend on repository interfaces, not mongo impls
- No raw Mongo queries in services
- No business logic in handlers

### package deps
- `internal/api` → `internal/service` only
- `internal/service` → `internal/repository`, `internal/llm`, `internal/events`
- `internal/worker` → `internal/llm`, `internal/repository`
- `internal/agents` → `internal/domain`, `internal/llm`, `internal/events`
- No circular imports

### Known half-finished areas (from current audit)
- `UpdateBible` — reads body, never persists
- `GetLocation` / `DeleteLocation` — stubs
- `CreateChapter` — missing `chapterNumber` parse
- `V2ListCharacters` — returns `[]` stub
- V1 scene handlers — commented out

## Output

```text
Findings (severity: critical/high/medium/low):
1. [critical] UpdateBile: reads req body, never calls repo — data loss bug
   file: internal/api/bible.go:42
   fix:  call s.bibleService.Update(...) after parsing

2. [high] Location endpoints half-implemented
   file: internal/api/locations.go
   fix:  implement GetLocation, DeleteLocation, UpdateBody

...
```

Include: package dep map, anti-corruption violations, test gap matrix, "merge now / fix later" triage.
