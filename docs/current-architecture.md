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
