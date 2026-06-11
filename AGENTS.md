# story-builder

Full-stack story graph editor with DAG-based plot structure and LLM-generated prose.

## Quick start

```bash
docker compose up -d     # postgres + pgvector
go run ./cmd/server/     # :8080
cd web && npm run dev    # :5173, proxies /api to :8080
```

## Architecture

Go server (chi) + React Flow frontend + Postgres + pgvector + River async jobs.

- `docs/architecture.md` — system overview + package deps
- `docs/api.md` — HTTP endpoints
- `docs/schema.md` — DB schema
- `docs/llm.md` — LLM pipeline (5 prompts)
- `docs/rules.md` — migration conventions

## Key types

| Domain   | Package     | Key structs                          |
|----------|-------------|--------------------------------------|
| Canon    | `canon`     | Character (versioned), Location (versioned), Lore (with pgvector) |
| Graph    | `graph`     | Story, Node, Edge — DAG with seq/fork/join/choice |
| Ledger   | `ledger`    | CharacterState per (story, char, node), StateDelta |
| Compiler | `compiler`  | CompiledContext + Hash() — resolved canon + state for prompting |
| LLM      | `llm`       | ModelTier enum, 5 service interfaces, PromptRegistry |
| River    | `river`     | 5 job types matching prompt pipeline |
| API      | `api`       | chi handlers + DB/in-memory service implementations |
| DB       | `db`        | sqlc-generated query methods (38) |

## DB layer

Postgres with pgvector. Connection via pgxpool. sqlc generates Go code from `sqlc/queries.sql`.

Critical: `toUUID(id)` = `pgtype.UUID{Bytes: id, Valid: true}` — Scan() does NOT accept `[]byte`.

## Tests

No test suite yet. Run the server and curl endpoints to verify.

## Common tasks

```bash
# Add a migration
touch migrations/003_description.sql
# restart server, runner auto-applies

# Regenerate sqlc code
sqlc generate -f sqlc/sqlc.yaml

# Rebuild Go
go build ./...
```
