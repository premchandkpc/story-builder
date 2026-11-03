# Phase 5: Developer Experience

## Problem

`README.md` references Postgres/pgvector/River which no longer exist. Doc files are scattered. No single dev command. No API client generation causing frontend/backend type drift. No system health check.

## 5.1 One-Command Dev

### `Makefile` additions

```makefile
.PHONY: dev dev-full doctor lint test generate-api clean

dev: ## Start minimal dev environment (Mongo + Redis + Go + Web)
	docker compose up -d mongo redis
	docker compose up -d headroom 2>/dev/null || true
	@echo "Starting Go API with live reload..."
	@which air > /dev/null 2>&1 || (echo "Installing air..."; go install github.com/air-verse/air@latest)
	@air -c .air.toml &
	@echo "Starting frontend..."
	@cd web && npm run dev &
	@wait

dev-full: ## Start everything (including narrative analysis)
	docker compose up -d --build

doctor: ## Check system health
	@echo "=== Story Builder Health Check ==="
	@printf "Go API       "; curl -sf http://localhost:8080/api/v1/healthz > /dev/null && echo "✅" || echo "❌"
	@printf "MongoDB      "; nc -z localhost 27017 2>/dev/null && echo "✅" || echo "❌"
	@printf "Redis        "; nc -z localhost 6379 2>/dev/null && echo "✅" || echo "❌"
	@printf "Java Service "; curl -sf http://localhost:8081/actuator/health > /dev/null 2>&1 && echo "✅" || echo "⚠  (optional)"
	@printf "OpenCode     "; curl -sf http://localhost:11434/api/tags > /dev/null 2>&1 && echo "✅" || echo "⚠  (optional)"
	@printf "Anthropic Key"; test -n "$$ANTHROPIC_API_KEY" && echo "✅" || echo "⚠  (optional)"

lint: ## Run Go linter
	golangci-lint run ./...
	cd web && npx tsc --noEmit

test: ## Run all tests
	go test ./... -count=1 -race -timeout 120s
	cd web && npx vitest run

generate-api: ## Generate TypeScript client from Go types
	go run ./cmd/api-gen > web/src/api/generated.ts
	cd web && npx prettier --write web/src/api/generated.ts

clean: ## Clean build artifacts
	rm -rf tmp/
	go clean -cache
	cd web && rm -rf dist/
```

### `.air.toml`

```toml
root = "."
tmp_dir = "tmp/"
[build]
  cmd = "go build -o tmp/server ./cmd/server"
  bin = "tmp/server"
  delay = 1000
  exclude_dir = ["web", "tmp", "vendor", "test"]
  include_ext = ["go", "tpl", "tmpl", "html"]
```

## 5.2 `story-builder doctor` CLI

Standalone command at `cmd/doctor/main.go`:

```go
// cmd/doctor/main.go
func main() {
    checks := []Check{
        {Name: "Go API", Check: checkURL("http://localhost:8080/api/v1/healthz")},
        {Name: "MongoDB", Check: checkPort("localhost:27017")},
        {Name: "Redis", Check: checkPort("localhost:6379"), Optional: true},
        {Name: "Anthropic Key", Check: checkEnv("ANTHROPIC_API_KEY"), Optional: true},
        {Name: "OpenCode", Check: checkURL("http://localhost:11434/api/tags"), Optional: true},
        {Name: "Java Analysis", Check: checkURL("http://localhost:8081/actuator/health"), Optional: true},
        {Name: "OTL Exporter", Check: checkOptionalEnv("OTEL_EXPORTER_OTLP_ENDPOINT")},
        {Name: "Collections/Indexes", Check: checkMongoIndexes("mongodb://localhost:27017", "story_builder")},
    }
    // Run checks, print table, exit 1 if any required check fails
}
```

## 5.3 Documentation Refresh

### `README.md` — remove stale references

Current issues:
- References to `pgvector`, `Postgres`, `River`, `Qdrant` in README and docs
- Architecture doc describes graph ops (`FindBranches`, `ValidateDAG`) as implemented when only `TopologicalSortStrings` exists
- `docs/services.md` references Phase 1 repo interfaces that have moved to `internal/repository/`

### Create `docs/current-architecture.md`

Single-page summary of the actual architecture (simplified, no aspirational features):

```markdown
# Current Architecture (June 2026)

## Stack
- React 19 + React Flow 12 + TanStack Query 5
- Go (chi) + MongoDB 7 + Redis 7
- Java Spring Boot (narrative analysis, port 8081, optional)

## Key Design Decisions
1. **No message queue** — workers run as in-process goroutines
2. **No Postgres** — MongoDB is single source of truth
3. **No vector database** — embeddings stored in MongoDB
4. **In-memory event bus** — synchronous pub/sub only
5. **All agent state is ephemeral** — character agents hold state in-memory across scene turns

## API
- ~70 endpoints under `/api/v1/`
- Experimental endpoints under `/experimental/`

## Generation Pipeline
- Two paths: agent orchestrator (structured scenes) or 6-step worker pipeline (simple scenes)
- Jobs are Mongo-backed with poll+lease semantics
- Status: queued → running → success/partial_success/failed

## Agent System
- 10 agents + N character agents (one per character in play)
- Turn ordering by FlowType: monologue/dialogue/round_robin/action/silent
- Character agents are goroutine-per-actor with event loop
```

## 5.4 OpenAPI → TypeScript Client

### `cmd/api-gen/main.go`

```go
// Scans Go domain structs and generates OpenAPI 3.0 spec
// Uses goverter or manual struct traversal to emit TS interfaces

type APIGen struct {
    Types   []TypeInfo
    Routes  []RouteInfo
}

func main() {
    gen := &APIGen{}
    // Walk domain/ package for structs with json tags
    // Walk api/ handlers for route definitions
    // Emit OpenAPI 3.0 JSON
    // Optionally: run openapi-typescript CLI
}
```

### Integration into build

```makefile
generate-api:
	go run ./cmd/api-gen > web/src/api/openapi.json
	npx openapi-typescript web/src/api/openapi.json -o web/src/api/generated.ts

precommit:
	make lint generate-api test
```

### Short-term (before generating)

Add a CI check that compares `web/src/api/types.ts` against Go domain structs:

```go
// cmd/api-audit/main.go — compare frontend types with backend structs
type FieldMatch struct {
    GoField     string
    TSField     string
    InGo         bool
    InTS         bool
    TagMatch    bool // json tag matches TS field name
}
```

## 5.5 Fixture-Based Local Demo Stories

### Seed script

`scripts/seed-demo.sh`:

```bash
#!/bin/bash
# Seeds MongoDB with a demo story for local development
# Usage: ./scripts/seed-demo.sh [mongo-uri]

MONGO_URI=${1:-"mongodb://localhost:27017"}
DB="story_builder"

# Load all fixture files
for file in test/golden/fixtures/simple-dialogue/*.json; do
    collection=$(basename "$file" .json)
    mongoimport --uri "$MONGO_URI" --db "$DB" --collection "$collection" --file "$file" --jsonArray
done
```

### Go seed command

```go
// cmd/seed/main.go — programmatic seed
func main() {
    db := connectToMongo(os.Getenv("MONGO_URI"))
    loader := golden.NewFixtureLoader(db)
    if err := loader.Load("test/golden/fixtures/simple-dialogue"); err != nil {
        log.Fatal(err)
    }
    fmt.Println("✅ Seeded demo story")
}
```

## 5.6 File Changes Summary

| File | Change |
|------|--------|
| `Makefile` | Add dev/doctor/lint/test/generate-api targets |
| `.air.toml` | Add for live reload |
| `cmd/doctor/main.go` (new) | System health check CLI |
| `cmd/api-gen/main.go` (new) | OpenAPI → TypeScript generator |
| `cmd/seed/main.go` (new) | Demo story seeder |
| `README.md` | Remove stale Postgres/River/Qdrant references |
| `docs/current-architecture.md` (new) | Single-page architecture summary |
| `scripts/seed-demo.sh` (new) | Bash seed script |
| `.github/workflows/ci.yml` (new) | Lint + test + API audit on PR |
