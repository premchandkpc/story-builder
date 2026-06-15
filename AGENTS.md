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
- `docs/architecture-audit-prompt.md` — 16-phase master audit checklist for full architecture review

## Key types

| Domain   | Package     | Key structs                          |
|----------|-------------|--------------------------------------|
| Canon    | `canon`     | Character (versioned), Location (versioned), Lore (with pgvector) |
| Graph    | `graph`     | Story, Node, Edge — DAG with seq/fork/join/choice |
| Ledger   | `ledger`    | CharacterState per (story, char, node), StateDelta |
| Compiler | `compiler`  | CompiledContext + Hash() — resolved canon + state for prompting |
| Prompt   | `prompt`    | CompilerService (10 layers, 5 merge strategies), PromptTemplate, MemoryStore |
| LLM      | `llm`       | ModelTier enum, 7 service interfaces, PromptRegistry |
| Timeline | `timeline`  | Engine (event-sourced, branches, past/present/future), Event, MemoryStore |
| River    | `river`     | 6 job types matching prompt pipeline |
| Event    | `event`     | Store + Bus (in-memory), 19 event types, evPromptCompiled |
| API      | `api`       | chi handlers + DB/in-memory service implementations |
| DB       | `db`        | sqlc-generated query methods (76) |

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

<!-- code-review-graph MCP tools -->
## MCP Tools: code-review-graph

**IMPORTANT: This project has a knowledge graph. ALWAYS use the
code-review-graph MCP tools BEFORE using Grep/Glob/Read to explore
the codebase.** The graph is faster, cheaper (fewer tokens), and gives
you structural context (callers, dependents, test coverage) that file
scanning cannot.

### When to use graph tools FIRST

- **Exploring code**: `semantic_search_nodes` or `query_graph` instead of Grep
- **Understanding impact**: `get_impact_radius` instead of manually tracing imports
- **Code review**: `detect_changes` + `get_review_context` instead of reading entire files
- **Finding relationships**: `query_graph` with callers_of/callees_of/imports_of/tests_for
- **Architecture questions**: `get_architecture_overview` + `list_communities`

Fall back to Grep/Glob/Read **only** when the graph doesn't cover what you need.

### Key Tools

| Tool | Use when |
| ------ | ---------- |
| `detect_changes` | Reviewing code changes — gives risk-scored analysis |
| `get_review_context` | Need source snippets for review — token-efficient |
| `get_impact_radius` | Understanding blast radius of a change |
| `get_affected_flows` | Finding which execution paths are impacted |
| `query_graph` | Tracing callers, callees, imports, tests, dependencies |
| `semantic_search_nodes` | Finding functions/classes by name or keyword |
| `get_architecture_overview` | Understanding high-level codebase structure |
| `refactor_tool` | Planning renames, finding dead code |

### Workflow

1. The graph auto-updates on file changes (via hooks).
2. Use `detect_changes` for code review.
3. Use `get_affected_flows` to understand impact.
4. Use `query_graph` pattern="tests_for" to check coverage.
