# Implementation Roadmap

## Phase 0: Foundation (Current State)

**Working:**
- [x] Go API server (chi) with all CRUD endpoints
- [x] MongoDB repositories for all entities
- [x] Redis cache + rate limiter + distributed locks
- [x] React Flow frontend (graph editing)
- [x] 6-worker generation pipeline
- [x] Prompt compiler (10-layer)
- [x] LLM router (sonnet/haiku/local-7b)
- [x] Event bus (in-memory)
- [x] Graph engine (topological sort, cycle detection)
- [x] Bible generation
- [x] Story cascade delete

**Known gaps (resolved):**
- [x] `UpdateBible` — persists via `bibleSvc.Update()` → `bibleRepo.Update()` (resolved)
- [x] `GetLocation` — real implementation via `locSvc.Get()` (resolved)
- [x] `CreateChapter` — `chapterNumber` parsed from request body (resolved)
- [x] `V2ListCharacters` — returns `[]` only when `story_id` omitted (intentional)
- [x] `UpdateLocation` — persists desc/props; name not mutable by design

## Phase 1: Agent Framework (Month 1)

**Goal:** Agent types, orchestrator, turn repository — no prompts yet.

1. **Domain types** (done in this blueprint)
   - [x] `domain.SceneTurn` — turn record
   - [x] `domain.AgentRun` — agent execution log
   - [x] `domain.CanonDelta` — append-only canon change log

2. **Repository interfaces**
   - [ ] `SceneTurnRepository` (Create, Get, Update, ListByScene, ListByRole)
   - [ ] `AgentRunRepository` (Create, List, DeleteByStory)
   - [ ] `CanonDeltaRepository` (Create, ListByScene, ListByStory, DeleteByStory)

3. **MongoDB implementations**
   - [ ] `internal/repository/mongo/scene_turns.go`
   - [ ] `internal/repository/mongo/agent_runs.go`
   - [ ] `internal/repository/mongo/canon_deltas.go`
   - [ ] Index definitions in `client.go`

4. **Agent registry + orchestrator** (done in this blueprint)
   - [x] `internal/agents/` — all agent types
   - [x] `internal/agents/orchestrator.go` — Plan, Execute, RunFinish
   - [x] FlowType → turn order mapping

5. **Integration tests**
   - [ ] Agent specs produce expected output shape
   - [ ] Orchestrator executes turn plan correctly
   - [ ] Turn order respects flow type

## Phase 2: Core Agents Online (Month 2)

**Goal:** 5 P0 agents produce real LLM output.

1. **Director Agent** — first agent to wire up
   - [ ] Read scene goal, arc, state, canon
   - [ ] Output turn plan with pressure, escalation, who acts
   - [ ] Signal `end_scene` when beat is resolved

2. **Character Agent**
   - [ ] Read character card + state + memories
   - [ ] Generate in-character dialogue/action via LLM (sonnet)
   - [ ] Respect what the character knows/don't know

3. **Narrator Agent**
   - [ ] Take turn outputs → narrative prose
   - [ ] Maintain POV, tone, pacing
   - [ ] Guard against extra-character knowledge leak

4. **CanonGuard Agent**
   - [ ] Check character location consistency
   - [ ] Check timeline ordering
   - [ ] Check relationship state consistency
   - [ ] Flag violations with severity

5. **StateExtract Agent**
   - [ ] After scene finish, extract character state deltas
   - [ ] Extract relationship changes
   - [ ] Extract timeline events
   - [ ] Record as `CanonDelta` documents

6. **Pipeline integration**
   - [ ] Route scenes with `sceneStructure` to orchestrator
   - [ ] Route simple scenes to existing pipeline
   - [ ] Hybrid: pipeline for generate, agents for orchestration within

## Phase 3: Editorial Agents (Month 3)

**Goal:** 2 P1 agents improve quality.

1. **Editor Agent**
   - [ ] Remove repetition
   - [ ] Fix pacing issues
   - [ ] Improve dialogue clarity
   - [ ] Trim verbosity

2. **Critic Agent**
   - [ ] Score scene 0.0-1.0 across 5 dimensions
   - [ ] Flag redundant, static, or passive scenes
   - [ ] Feed scores back to Director for next scene

3. **UI for turn playback**
   - [ ] Turn-by-turn timeline in frontend
   - [ ] Accept/reject per turn
   - [ ] Compare multiple generations side by side
   - [ ] Agent debug panel (what each agent decided)

## Phase 4: Advanced Agents (Month 4)

**Goal:** 3 more agents for depth + narrative intelligence.

1. **World Agent**
   - [ ] Faction relationship tracking
   - [ ] World rule consistency
   - [ ] Lore violation detection
   - [ ] Setting realism checks

2. **Arc Agent**
   - [ ] Act progression tracking (act 1/2/3)
   - [ ] Character arc growth stage
   - [ ] Plot thread status (open/advancing/resolved/abandoned)
   - [ ] Foreshadowing delivery vs payoff

3. **Memory Agent**
   - [ ] Layered memory management (character/scene/world/narrative)
   - [ ] Memory importance scoring
   - [ ] Retention/forgetting rules
   - [ ] Feed relevant memories to context builder

## Phase 5: Platform (Month 5+)

**Goal:** Production hardening, multi-story orchestration, community features.

1. **Performance**
   - [ ] Token budget enforcement per agent
   - [ ] Parallel agent execution (non-blocking turns)
   - [ ] Prompt caching for repeated contexts

2. **Observability**
   - [ ] Agent run tracing (OpenTelemetry)
   - [ ] Turn-level metrics (duration, tokens, errors)
   - [ ] Critic score dashboards

3. **Multi-story orchestration**
   - [ ] Shared world bible across stories
   - [ ] Cross-story timeline
   - [ ] Character migration between stories

4. **Community features**
   - [ ] Export/import agent configs
   - [ ] Custom agent definitions via API
   - [ ] Agent marketplace (prompt templates)

## Priority Matrix

| Agent | Value | Effort | Phase |
|-------|-------|--------|-------|
| Director | critical | low | P2 |
| Character | critical | medium | P2 |
| Narrator | critical | medium | P2 |
| CanonGuard | high | low | P2 |
| StateExtract | high | low | P2 |
| Editor | medium | low | P3 |
| Critic | medium | low | P3 |
| World | medium | medium | P4 |
| Arc | medium | medium | P4 |
| Memory | medium | high | P4 |

## Risk Areas

1. **LLM cost**: Character agent × N participants × M turns = expensive. Mitigation: turn limits, prompt caching, cheaper model for routine turns.
2. **Prompt drift**: 10 agents × evolving prompts = chaos. Mitigation: prompt registry in `internal/prompt/`, versioned templates, golden prompt tests.
3. **State inconsistency**: Multiple agents reading/writing state concurrently. Mitigation: orchestrator serializes turns, in-flight locks per scene, append-only state.
4. **Frontend complexity**: Turn playback, agent debug panels, graph editing. Mitigation: progressive disclosure, ship basic turn list first.
