# Runtime Agent Architecture

## Overview

The agent system extends story-builder's pipeline model into a multi-agent turn engine. Agents are in-process actors orchestrated by a Director-like Orchestrator. Each agent has a role, system prompt, model tier, and execution boundaries.

Each Character is its own autonomous agent — with persistent in-memory state, event-driven reactivity, per-character identity baked into the AgentSpec, and the ability to propose actions proactively (not only when the Director calls on them).

```
Pipeline (current)        →  Agent Orchestration (target)
─────────────────            ─────────────────────────
6 sequential workers         10 agents with turn ordering
LLM calls per scene          LLM calls per agent turn
Single-shot generation       Iterative turn-based generation
Extract/validate after       Agents validate during turns
One Character agent spec     Per-character AgentSpec (identity baked in)
Stateless agent calls        Persistent in-memory agent state across turns
Reactive (Director calls)    Proactive (character initiative)
```

## Agent Registry

All agents are registered in `internal/agents/register.go` via `AgentSpec`:

```go
type AgentSpec struct {
    Name         string        // unique identifier
    Role         string        // semantic role
    Model        string        // model tier
    MaxTurns     int           // max times this agent can act per scene
    SystemPrompt string        // role definition + constraints
    Runner       AgentRunner   // execution function
}
```

Character agents are NOT registered statically. They are created dynamically by `CharacterManager.StartAgent()` which calls `NewCharacterAgentSpec(charID, llmClient, proseSvc, state)` and registers the spec under the character's ID.

## Character Manager

`internal/agents/character_manager.go` manages the lifecycle of all character agents:

```
CharacterManager
  ├── StartAgent(charID, character) → spawns agent goroutine + registers AgentSpec
  ├── StopAgent(charID) → stops agent goroutine + deregisters
  ├── StopAll() → stops all running agents
  ├── GetAgent(charID) → returns running agent instance
  ├── BroadcastEvent(event) → sends event to all character agents
  └── QueryProposals(ctx) → asks all agents what they want to do
```

Each `CharacterAgentInstance` runs a goroutine with an event loop that processes `CharacterEvent`s:

```go
type CharacterEvent struct {
    Type      CharacterEventType  // scene_start, turn_complete, character_action, scene_end, query_intent
    StoryID   string
    SceneID   string
    TurnID    string
    Data      map[string]any
    Timestamp time.Time
}
```

### Persistent Agent State

Each character agent holds in-memory `CharacterAgentState` that persists across turns within a scene:

```go
type CharacterAgentState struct {
    CharacterID      string
    Name             string
    CurrentEmotion   string            // emotional drift across turns
    CurrentMood      string
    ActiveGoal       string            // current goal driving behavior
    SubGoals         []string
    Knowledge        []string
    KnowledgeGaps    []string
    InternalThoughts []InternalThought // reflection/plan/worry/desire log
    RecentActions    []ActionRecord    // last 20 actions for context
    RelState         map[string]*RelState
    Plan             *ActionPlan       // autonomous plan for achieving goals
    RecentDialogue   []string          // last 10 exchanges
}
```

State is updated after each character turn (emotion detected from output text, dialogue recorded, actions logged). Between scenes, state is persisted via the existing append-only `CharacterState` mechanism.

## The 10 Runtime Agents

### P0 Agents (build first)

| # | Agent | Role | Model | Temp | Purpose | Status |
|---|-------|------|-------|------|---------|--------|
| 1 | **Director** | scene_director | sonnet | 0.3 | Plan turns, set pressure, decide who acts next, signal scene end | ✅ Real LLM call w/ JSON parse |
| 2 | **Character** (×N) | character | sonnet | 0.8 | One AgentSpec per character. Identity baked in. Persistent state. Autonomous proposals. | ✅ Per-character specs, persistent state, event-driven, autonomous proposals |
| 3 | **Narrator** | narrator | sonnet | 0.5 | Stitch character actions into coherent narrative prose | ✅ Real LLM call |
| 4 | **CanonGuard** | canon_guard | haiku | 0.0 | Verify character/location/timeline/relationship consistency | ✅ Rule-based + LLM hybrid |
| 5 | **StateExtractor** | state_extractor | local-7b | 0.0 | Extract structured deltas after scene completion | ✅ Delegates to extractSvc.ExtractState() |

### P1 Agents (next wave)

| # | Agent | Role | Model | Temp | Purpose | Status |
|---|-------|------|-------|------|---------|--------|
| 6 | **Editor** | editor | haiku | 0.2 | Trim repetition, fix pacing, improve clarity | ✅ Real LLM call, polished prose output |
| 7 | **Critic** | critic | haiku | 0.0 | Score scene usefulness (0.0-1.0), flag weak output | ✅ Real LLM call w/ JSON parse, 5-dimension scoring |
| 8 | **World** | world_keeper | sonnet | 0.3 | Faction politics, world rules, lore consistency | ✅ Full impl, wired into orchestrator RunFinish |
| 9 | **Arc** | arc_tracker | haiku | 0.2 | Track act progression, character arcs, plot threads | ✅ Full impl, wired into orchestrator RunFinish |
| 10 | **Memory** | memory_keeper | local-7b | 0.0 | Maintain layered memory (character/scene/world/narrative) | ✅ Full impl, wired into orchestrator RunFinish |

## Orchestration Flow

```
User triggers scene generation
        │
        ▼
AgentService.GenerateScene(sceneID)
  → Load scene from DB
  → Build AgentContext (characters, states, memories, timeline, etc.)
  → EnsureAgentsRunning — spawns per-character agents if not already running
        │
        ▼
Orchestrator.Plan(scene)
  → Collect autonomous proposals from character agents (QueryProposals)
  → Gather participating character IDs (scene participants + proposers)
  → Build TurnOrder by FlowType with per-character AgentSpec names
        │
        ▼
Orchestrator.Execute(plan, agentContext)
  ┌─────────────────────────────────────┐
  │ Broadcast EventSceneStart           │
  │                                     │
  │  Turn 1: Director (plan)            │
  │   → who acts, pressure, escalation  │
  │                                     │
  │  Turn 2: Char_A (perform)           │
  │   → uses persistent state + context │
  │                                     │
  │  Turn 3: Char_B (respond)           │
  │   → uses persistent state + context │
  │                                     │
  │  Broadcast EventTurnComplete        │
  │   → all character agents notified   │
  │                                     │
  │  Turn 4: Narrator (narrate)         │
  │   → prose frame for the turn        │
  │                                     │
  │  ... repeat for N turns ...         │
  │                                     │
  │  Broadcast EventSceneEnd             │
  └─────────────────────────────────────┘
        │
        ▼
Orchestrator.RunFinish(scene)
  ┌─────────────────────────────────────┐
  │ Phase 2: Scene Finish               │
  │  StateExtract → World → Arc →      │
  │  Memory → Director(evaluate)        │
  └─────────────────────────────────────┘
        │
        ▼
StopAll character agents
Persist generation + turns
```

## Turn Ordering by FlowType

Now uses per-character AgentSpec names instead of generic "character" agent type:

| FlowType | Turn Sequence |
|----------|--------------|
| `monologue` | Director → Char_A → Char_B → ... → Narrator → Editor → CanonGuard |
| `dialogue` | Director → (Char_A → Char_B → ...) → (Char_A_respond → Char_B_respond → ...) → Narrator → Editor → CanonGuard |
| `round_robin` | Director → (Char_A → Char_B → ... → Narrator → CanonGuard) × maxTurns |
| `parallel` | Director → (all characters) → Narrator → CanonGuard |
| `action` | Director → (all characters act) → Narrator → Editor |
| `silent` | Director → Narrator |

Participating characters are gathered from both `scene.Participants` and autonomous proposal authors.

## Model Routing

| Agent | Model | Temp | Priority | MaxRetries | Timeout |
|-------|-------|------|----------|------------|---------|
| Director | claude-sonnet | 0.3 | high | 2 | 30s |
| Character (each) | claude-sonnet | 0.8 | high | 2 | 60s |
| Narrator | claude-sonnet | 0.5 | high | 2 | 30s |
| Editor | claude-haiku | 0.2 | medium | 1 | 20s |
| CanonGuard | claude-haiku | 0.0 | medium | 1 | 15s |
| Critic | claude-haiku | 0.0 | low | 1 | 15s |
| StateExtract | local-7b | 0.0 | low | 1 | 30s |
| World | claude-sonnet | 0.3 | low | 1 | 30s |
| Arc | claude-haiku | 0.2 | low | 1 | 20s |
| Memory | local-7b | 0.0 | low | 1 | 30s |

## Required vs Optional

- **Director**: always required, always blocking
- **Character (each)**: required for monologue/dialogue/action, optional for silent
- **Narrator**: always required for non-silent
- **Editor**: optional (skippable)
- **CanonGuard**: optional (skippable)
- **Critic**: runs after all turns, non-blocking
- **StateExtract**: runs on scene finish, non-blocking
- **World/Arc/Memory**: run in RunFinish phase, non-blocking, non-required

## Agent Context

Each agent receives an `AgentContext` with:

```go
type AgentContext struct {
    StoryID        string
    SceneID        string
    TurnID         string
    Story          *domain.Story
    Scene          *domain.Scene
    Characters     []*domain.Character
    CharStates     []*domain.CharacterState
    Bible          *domain.StoryBible
    Edges          []*domain.SceneEdge
    Turns          []*domain.SceneTurn
    Timeline       []*domain.TimelineEvent
    Memories       map[string][]*domain.CharacterMemory
    CanonDeltas    []*domain.CanonDelta
    Summaries      []*domain.Summary
    ParticipantIDs []string
}
```

Character agents also maintain their own `CharacterAgentState` in memory (not passed through AgentContext).

## Autonomous Proposals

Before the Director plans a scene, the Orchestrator calls `CharacterManager.QueryProposals()` which asks each character agent what they want to do:

1. Each agent's runner is called with `Directive: "propose"`
2. The agent generates an in-character intention statement based on current goals, emotion, and scene context
3. Proposals are collected and attached to the `OrchestrationPlan.Proposals`
4. The Director's plan incorporates these proposals
5. Characters that proposed actions are included in the turn order even if not in `scene.Participants`

## Event-Driven Reactivity

Character agents receive events during scene execution:

| Event | When | Effect |
|-------|------|--------|
| `EventSceneStart` | Before first turn | Initialize state (emotion, mood, goal, knowledge from DB) |
| `EventTurnComplete` | After each turn | Update emotion from output, record dialogue |
| `EventCharAction` | After character turn | Update internal state with character's action |
| `EventNarratorOutput` | After narrator turn | Context for future proposals |
| `EventSceneEnd` | After all turns | Cleanup |

## Integration with Pipeline

The agent orchestrator can replace or augment the existing 6-worker pipeline:

- **Simple scenes** (no sceneStructure): use existing pipeline
- **Multi-agent scenes** (sceneStructure set): use agent orchestrator
- **Hybrid**: pipeline runs turn generation, agents orchestrate within each turn

## Event Integration

Each agent turn publishes events via `events.Bus`:

- `EventAgentTurnCompleted` — per turn completion
- `EventSceneTurnsComplete` — all turns done
- `EventAgentError` — agent failure

Character agents also receive `CharacterEvent`s during scene execution.

See `internal/events/` for event type constants.

## File Map

```
internal/agents/
  types.go                AgentContext, AgentSpec, CharacterAgentState, CharacterEvent, CharacterProposal
  register.go             RegisterAll (non-character agents) + RegisterCharacterAgent helper
  character_manager.go    CharacterManager — lifecycle, event loop, proposals
  character_agent.go      NewCharacterAgentSpec factory, per-character runner with persistent state
  orchestrator.go         Orchestrator — Plan (collects proposals), Execute (broadcasts events), RunFinish
  director.go             Director agent
  narrator.go             Narrator agent
  editor.go               Editor agent
  canon_guard.go          CanonGuard agent
  critic.go               Critic agent
  state_extractor.go      StateExtract agent
  world.go                World agent
  arc.go                  Arc agent
  memory_agent.go         Memory agent
  agent_repository.go     SceneTurnRepository interface
```
