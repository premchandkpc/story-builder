# ADR 0003: Simplify Stack — MongoDB Only, No Message Queue

## Status

Accepted. Supersedes ADR 0002 (Narrative OS direction) for infrastructure decisions.

## Context

ADR 0002 charted a 12-phase evolution toward a Narrative OS with PostgreSQL, MongoDB, Neo4j, Qdrant, Kafka, and gRPC. That vision is ambitious but has led to:

- 7 infrastructure services (Postgres + pgvector + Redis + Mongo + Qdrant + Kafka + OpenCode)
- SQL schema management (migrations, sqlc, 9+ migration files)
- Dual-mode code paths (DB vs in-memory) for every service
- River job queue (wraps Postgres as a queue, adds migration complexity)
- gRPC layer (unused by the React frontend)
- Adapter code for every database

The underlying problem: story data is **document-shaped**, not relational. Stories, scenes, character states, and memories are evolving documents with nested structure — exactly what MongoDB was built for.

The complexity cost of maintaining 7 services for a 1-developer project is not justified.

## Decision

Adopt a simplified architecture:

```
React Flow → Go API (chi) → Service Layer → Repository Interfaces
                                               ├── MongoDB (single source of truth)
                                               └── Redis (cache, rate limits, locks only)
```

**Removed:**
- PostgreSQL + pgvector — all data moves to MongoDB
- Kafka — workers are in-process goroutines
- Qdrant — MongoDB Atlas Search handles vector embeddings
- River — in-process goroutine workers replace the job queue
- gRPC — removed (not used by the React frontend)
- sqlc + migrations — MongoDB is schemaless; indexes are code
- Adapter layer (`internal/adapter/kafka/`, `internal/adapter/qdrant/`)

**Kept:**
- Go + Chi — works well, no reason to change
- React Flow — same
- Redis — cache, rate limits, distributed locks only (never a source of truth)
- OpenCode / Anthropic — LLM providers

**Changed:**
- Workers from River jobs → goroutine-based workers in `internal/worker/`
- Memory storage from Qdrant → MongoDB vector search
- Character state from Postgres JSONB → MongoDB append-only documents
- Repository from sqlc → `internal/repository/mongo/` implementations
- Index management from SQL migrations → application startup code

## Consequences

### Positive
- 4 infrastructure services instead of 7 (MongoDB, Redis, OpenCode, server)
- No migration system to maintain
- No dual-mode code paths (no in-memory fallback needed)
- No River schema management
- Simpler deployment (single docker-compose.yml)
- Faster local development (fewer services to start)
- Story-shaped data in a document database
- Workers are simple goroutines — no queue infrastructure

### Negative
- MongoDB Atlas Search requires an Atlas cluster (or a local workaround for dev)
- No strict relational integrity (application must enforce)
- Existing SQL migrations and sqlc code must be removed
- Character versioning model changes (append-only documents instead of versioned rows)
- River workers must be rewritten as goroutine workers
- gRPC service definitions become unused

### Neutral
- Prompt compiler (10-layer) and LLM service interfaces are unchanged
- DAG engine (graph/traversal.go) is unchanged
- Validation logic is unchanged
- API handlers mostly unchanged (minor payload adjustments)
- Aggregate-root design remains the same

## Key Principles

1. **MongoDB is the only database.** Redis is cache/locks/rate limits only.
2. **No new infrastructure without a measured bottleneck.** Not before 100k users or 10 concurrent writers.
3. **No message queue.** Goroutine workers with context cancellation are sufficient at this scale.
4. **Indexes are code.** Defined in `internal/repository/mongo/client.go`, created on startup. Conflicts are logged and skipped — not fatal.
5. **Append-only for state.** Never overwrite character state. Event-sourcing lite.
6. **The moat is story intelligence.** Better DAGs, better memory, better validation. Not database count.
