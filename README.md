# Story Builder

Full-stack story graph editor with DAG-based plot structure and LLM-generated prose.
Built with Go (chi), React Flow, MongoDB, and Redis.

## Quick start

```bash
docker compose up -d mongo redis       # minimal infra (OpenCode runs locally on host)
docker compose up -d --build           # everything
make dev                               # Go API + Vite frontend with live reload
```

## Project structure

```
cmd/server/main.go          # Entrypoint
internal/
  api/                      # chi HTTP handlers + middleware
  agents/                   # Multi-agent scene orchestration (10 agents)
  cache/                    # Redis-backed caching (prompt, rate limiter, dist lock)
  config/                   # Environment-based config
  domain/                   # Domain models (no infra dependencies)
  events/                   # In-memory event bus
  graph/                    # DAG traversal + validation
  llm/                      # LLM clients + Router + 7 service implementations
  log/                      # Structured logging (slog-based)
  narrative/                # Narrative domain services
  prompt/                   # Prompt compilation + token budgeting
  repository/               # Interfaces + MongoDB implementations
  scene/                    # Scene turn orchestration
  service/                  # Business logic layer
  test/                     # Integration test fixtures
  trace/                    # OpenTelemetry wrapper
  validation/               # Scene/canon validation
  worker/                   # Pipeline workers (generate, extract, memory, timeline, summary, validate)
web/                        # Vite + React + React Flow frontend
docs/                       # Architecture, schema, API, LLM docs
```

## Key architecture decisions

- **SSOT MongoDB** — No Postgres, pgvector, Kafka, or River. Single-process Go with in-process workers.
- **LLM Router** — Dispatches `claude-sonnet`/`claude-haiku` to Anthropic, `local-7b` to OpenCode.
- **Redis cache** — Optional prompt caching + rate limiting + distributed locks.
- **Durable job queue** — MongoDB-backed lease-based worker pool with crash recovery.
- **Multi-agent orchestrator** — 10 agents (Director, Character×N, Narrator, Editor, CanonGuard, Critic, StateExtract, World, Arc, Memory) for structured scene generation.
- **Event-sourced character state** — Append-only character_state + canon_deltas collections.
- **Per-character agent identities** — Each character registers its own AgentSpec with baked-in identity.
- **StoryRun + RunStep** — Persistent orchestration tracking with per-step artifacts.
- **NarrativeEvent log** — Append-only universal state mutation log for replayability.
- **Context caching** — SHA256 context hash dedup avoids regenerating identical scenes.
- **Headroom-ai** — Optional context compression proxy for all LLM calls.
- **Java analysis service** — Readability/sentiment/pacing scores (port 8081, async).

## Tests

```bash
go test ./...
```

Test priority: graph (cycle detection, topological sort), memory (state changes, retrieval),
generation (pipeline), validation (4 validators).

## Environment

See `.env` for defaults. Key vars:
- `MONGO_URI` — MongoDB connection string
- `REDIS_ADDR` — Redis address (optional)
- `HEADROOM_BASE_URL` — Context compression proxy (optional)
- `ANTHROPIC_API_KEY` — For claude-sonnet/claude-haiku calls
- `OPENCODE_BASE_URL` — For local-7b model

## Docs

| File | Contents |
|------|----------|
| `docs/architecture.md` | System overview, package deps, data flow |
| `docs/api.md` | HTTP endpoints reference |
| `docs/schema.md` | MongoDB collections, indexes |
| `docs/llm.md` | LLM pipeline, prompts, router, clients |
| `docs/services.md` | Service layer implementations |
| `docs/scene-agents.md` | Multi-agent scene system design |
| `docs/rules.md` | Code conventions |
