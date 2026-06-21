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

2. **Repository interfaces** (in `internal/scene/turn.go`, pre-existing)
   - [x] `TurnRepository` (Create, Get, Update, ListByScene, ListByRole)
   - [x] `ActorRepository` (Create, List, DeleteByStory)
   - [x] `CanonDeltaRepository` (Create, ListByScene, ListByStory, DeleteByStory)

3. **MongoDB implementations**
   - [x] `internal/repository/mongo/scene_turns.go`
   - [x] `internal/repository/mongo/agent_runs.go`
   - [x] `internal/repository/mongo/canon_deltas.go`
   - [x] Index definitions in `client.go`

4. **Agent registry + orchestrator** (done in this blueprint)
   - [x] `internal/agents/` — all agent types
   - [x] `internal/agents/orchestrator.go` — Plan, Execute, RunFinish
   - [x] FlowType → turn order mapping

5. **Integration tests**
   - [x] SceneTurnRepo CRUD (Create, Get, Update, ListByScene, ListByRole, DeleteByScene, DeleteByStory)
   - [x] AgentRunRepo CRUD (Create, List with filters, DeleteByStory)
   - [x] CanonDeltaRepo CRUD (Create, ListByScene, ListByStory, DeleteByStory)
   - [x] TurnOrchestrator integration (GetTurns, GetTurnsByRole, GetCanonDeltas, RecordStateDelta)
   - [x] Fixed pre-existing integration test breakage (api_test.go, pipeline_test.go)

## Phase 2: Core Agents Online (Month 2)

**Goal:** 5 P0 agents produce real LLM output.

1. **Director Agent**
   - [x] Wire into `internal/agents/director.go` with real LLM call via `LLMClient.Complete()`
   - [x] Reads scene goal, participants, character states, previous turns
   - [x] Outputs JSON turn plan with pressure, escalation, who acts, end_scene
   - [x] Falls back to hardcoded decisions on LLM failure

2. **Character Agent**
   - [x] Real LLM call via `LLMClient.Complete()` (sonnet, 0.8 temp)
   - [x] Reads character card (persona, traits, goals, flaws, voice, backstory)
   - [x] Reads current emotional/physical state, knowledge, active goal
   - [x] Reads relevant memories (top 5) for context
   - [x] Selects character from payload or auto-rotates through participants
   - [x] Builds structured prompt with scene context + character data + prev turns

3. **Narrator Agent**
   - [x] Real LLM call via `LLMClient.Complete()` (sonnet, 0.5 temp)
   - [x] Weaves character turns into narrative prose
   - [x] Maintains tone, POV, pacing
   - [x] Does not invent new character actions — narrates only existing material

4. **CanonGuard Agent** (hybrid: rule-based + LLM)
   - [x] Rule: character location jump detection across states
   - [x] Rule: timeline ordering check
   - [x] Rule: negative health / invalid state detection
   - [x] Rule: low-confidence canon delta flagging
   - [x] LLM evaluation via haiku 0.0 temp with ValidateJSON for nuanced violations

5. **StateExtract Agent**
   - [x] Delegates to `extractSvc.ExtractState()` (local-7b, 0.0 temp via prompt compiler)
   - [x] Extracts emotional state, knowledge, relationship, location changes
   - [x] Maps extracted deltas to `CanonDelta` records
   - [x] Captures open plot threads as canon deltas

6. **Pipeline integration**
   - [x] `AgentService` in `internal/service/agent.go` — builds context, runs orchestrator
   - [x] `GenerationService.Generate()` routes scenes with `SceneStructure` or `FlowType` to agent orchestrator
   - [x] `RegisterAll()` convenience in `internal/agents/register.go`
   - [x] All 10 agents registered in `init.go` via `agentRegistry`
   - [x] Orchestrator wired with registry + LLM router + event bus
   - [x] AgentService wired into `appDependencies` + passed through to API handlers
   - [ ] Hybrid: pipeline for generate, agents for orchestration within (deferred)

## Phase 3: Editorial Agents (Month 3)

**Goal:** 2 P1 agents improve quality.

1. **Editor Agent**
   - [x] Real LLM call via `LLMClient.Complete()` (haiku, 0.2 temp)
   - [x] Reads narrator's turn output, polishes prose
   - [x] Removes repetition, fixes pacing, trims verbosity
   - [x] Preserves all story content — edits only prose quality
   - [x] Detects whether changes were actually made

2. **Critic Agent**
   - [x] Real LLM call via `LLMClient.Complete()` (haiku, 0.0 temp, ValidateJSON)
   - [x] Scores scene 0.0-1.0 across 5 dimensions
   - [x] Returns critiques + strengths + summary verdict
   - [x] Flags redundant, static, or passive scenes

3. **UI for turn playback**
   - [x] Turn-by-turn timeline in frontend (`TurnTimeline.tsx`)
   - [x] Accept/reject per generation (existing generation accept flow)
   - [x] Compare multiple generations side by side (`GenerationCompare.tsx`)
   - [x] Agent debug panel (`AgentRunPanel.tsx`)

## Phase 4: Advanced Agents (Month 4)

**Goal:** 3 more agents for depth + narrative intelligence.

1. **World Agent**
   - [x] Full implementation (`internal/agents/world.go`)
   - [x] Faction relationship tracking
   - [x] World rule consistency
   - [x] Lore violation detection
   - [x] Setting realism checks
   - [x] Wired into orchestrator RunFinish (non-blocking)

2. **Arc Agent**
   - [x] Full implementation (`internal/agents/arc.go`)
   - [x] Act progression tracking (act 1/2/3)
   - [x] Character arc growth stage
   - [x] Plot thread status (open/advancing/resolved/abandoned)
   - [x] Foreshadowing delivery vs payoff
   - [x] Wired into orchestrator RunFinish (non-blocking)

3. **Memory Agent**
   - [x] Full implementation (`internal/agents/memory_agent.go`)
   - [x] Layered memory management (character/scene/world/narrative)
   - [x] Memory importance scoring
   - [x] Retention/forgetting rules
   - [x] Feed relevant memories to context builder
   - [x] Wired into orchestrator RunFinish (non-blocking)

## Phase 5: Platform (Month 5+)

**Goal:** Production hardening, multi-story orchestration, community features.

1. **Performance**
   - [x] Token budget tracking domain type (`internal/domain/token_budget.go`)
   - [ ] Token budget enforcement per agent
   - [x] Parallel agent execution (non-blocking turns in RunFinish)
   - [ ] Prompt caching for repeated contexts

2. **Observability**
   - [x] LLM metrics service (`internal/service/metrics.go`)
   - [x] LLM metrics dashboard (`LlmMetricsDashboard.tsx`)
   - [x] Metrics API endpoint (`GET /api/v1/stories/{storyID}/metrics/llm`)
   - [x] Budget-related event types (`budget.limit_exceeded`, `budget.warning`, `metrics.updated`)
   - [x] Turn-level metrics (duration, tokens, errors already in SceneTurn model)
   - [ ] Critic score dashboards (data exists, frontend pending)
   - [ ] Agent run tracing (OpenTelemetry)

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
