# story-builder

Full-stack story graph editor with DAG-based plot structure and LLM-generated prose. Built with Go (chi), React Flow, Postgres + pgvector, and River async jobs.

## Quick start

```bash
docker compose up -d          # postgres + pgvector + redis
go run ./cmd/server/          # :8080
cd web && npm run dev         # :5173, proxies /api → :8080
```

## Project structure

```
cmd/server/main.go          # Entrypoint — selects DB or in-memory mode
internal/
  api/                      # chi HTTP handlers + middleware
  cache/                    # Redis-backed caching (prompt, context, rate limiter, dist lock)
  canon/                    # Versioned domain types (Character, Location, Lore)
  compiler/                 # CompiledContext + SHA256 hash + prompt builders
  config/                   # Environment-based config
  db/                       # sqlc-generated query layer (38 methods)
  graph/                    # DAG data model + traversal algorithms
  grpc/                     # gRPC server wrapping service interfaces
  ledger/                   # CharacterState per (story, char, node)
  llm/                      # LLM clients + Router + 7 service implementations
  log/                      # Structured logging (slog-based)
  migrate/                  # SQL migration runner
  narrative/                # Narrative domain models (Blueprint, CharacterArc, PlotThread, Act)
  river/                    # River async job types + workers
  scene/                    # Multi-agent scene turn orchestration
  service/                  # Service implementations (DB + memory dual mode)
    blueprint/              # Story blueprint service
    cache/                  # Redis cache wrapper service
    canon/                  # Character, Actor, Trait, Casting, Location, Lore
    edge/                   # Edge CRUD
    generation/             # Generation + story generator
    node/                   # Node CRUD
    scene/                  # Scene CRUD (multi-agent)
    story/                  # Story CRUD
    summary/                # Summary CRUD
    timeline/               # Timeline event service
  timeline/                 # Timeline domain models
```

## Key architecture decisions

- **Dual mode** — DB-backed (Postgres via sqlc) or in-memory (for dev without Docker)
- **LLM Router** — Dispatches `claude-sonnet`/`claude-haiku` to Anthropic, `local-7b` to Ollama
- **Redis cache** — Optional prompt caching + rate limiting + distributed locks
- **River async jobs** — 5 job types for the generation pipeline (generate → extract → summarize → merge → validate)
- **Canon versioning** — Characters and locations are append-only (id, version) PK
- **Blueprints + Timelines** — Narrative planning layer (memory-only, no DB backing yet)
- **No tests yet** — Run the server and curl endpoints to verify

## Docs

| File | Contents |
|------|----------|
| `docs/architecture.md` | System overview, package deps, data flow, dual mode |
| `docs/api.md` | HTTP endpoints reference |
| `docs/schema.md` | Database schema |
| `docs/llm.md` | LLM pipeline, prompts, router, clients |
| `docs/rules.md` | Migration conventions |
| `docs/services.md` | Service layer implementations |
| `docs/scene-agents.md` | Multi-agent scene system design |
| `docs/review-and-enhancements.md` | Architecture review + roadmap |
| `docs/adr/` | Architecture decision records |
