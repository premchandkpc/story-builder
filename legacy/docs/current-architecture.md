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
- ~80 endpoints under `/api/v1/`
- Experimental endpoints under `/experimental/`

## Generation Pipeline
- Two paths: agent orchestrator (structured scenes) or 6-step worker pipeline (simple scenes)
- Jobs are Mongo-backed with poll+lease semantics
- Status: queued → running → success/partial_success/failed

## Agent System
- 10 agents + N character agents (one per character in play)
- Turn ordering by FlowType: monologue/dialogue/round_robin/action/silent
- Character agents are goroutine-per-actor with event loop

## Product Intelligence (Phase 7)
- **Scene Plan**: `GET /stories/{id}/nodes/{id}/plan` — `PlannerService` analyzes scene purpose, participant intents, required beats, suggested tone/POV/word count
- **Generation Diff**: `GET /stories/{id}/nodes/{id}/generations/{genID}/diff?against={genBID}` — `DiffService` compares prose, events (added/removed), and token usage between two generations
- **Plan Tab**: `ScenePlanPanel` in GraphPanel shows structural plan for selected scene
- **Enhanced Compare**: `GenerationCompare` now displays server-side diff (prose diff + event diffs + token delta)

## Run Inspector (Phase 4)
- 5-tab `RunInspector` component: Overview / Prompt / Timeline / Events / Cost
- Lazy data loading per tab — endpoints only called when tab is active
- 4 new components: `RunTimeline` (Gantt bars), `PromptSectionViewer` (collapsible sections), `EventList` (two-tone cards), `CostCard` (by-model breakdown)
- Backend: `GET /runs/{id}/prompt-sections`, `GET /runs/{id}/events`, `GET /runs/{id}/cost`, `GET /stories/{id}/runs/stats`

## Extra Collections (not in schema.md)
- `story_blueprints` — structural plans (acts, arcs, threads)
- `story_runs` — durable orchestration tracking
- `run_steps` — per-step execution records within runs
- `narrative_events` — append-only state mutation log
- `scene_locks` — distributed generation locking (optional, cross-process)
- `character_views` — projected character state cache (event-replay)
- `agent_configs` — shareable agent configuration specs
- `jobs` — durable generation job queue
- `token_budgets` — per-story token allocation tracking
