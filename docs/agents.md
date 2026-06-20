# Runtime Agent Architecture

## Overview

The agent system extends story-builder's pipeline model into a multi-agent turn engine. Agents are in-process actors orchestrated by a Director-like Orchestrator. Each agent has a role, system prompt, model tier, and execution boundaries.

```
Pipeline (current)        →  Agent Orchestration (target)
─────────────────            ─────────────────────────
6 sequential workers         10 agents with turn ordering
LLM calls per scene          LLM calls per agent turn
Single-shot generation       Iterative turn-based generation
Extract/validate after       Agents validate during turns
```

## Agent Registry

All agents are registered in `internal/agents/agent_registry.go` via `AgentSpec`:

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

## The 10 Runtime Agents

### P0 Agents (build first)

| # | Agent | Role | Model | Temp | Purpose |
|---|-------|------|-------|------|---------|
| 1 | **Director** | scene_director | sonnet | 0.3 | Plan turns, set pressure, decide who acts next, signal scene end |
| 2 | **Character** | character | sonnet | 0.8 | Role-play one character per instance (dialogue/action/internal) |
| 3 | **Narrator** | narrator | sonnet | 0.5 | Stitch character actions into coherent narrative prose |
| 4 | **CanonGuard** | canon_guard | haiku | 0.0 | Verify character/location/timeline/relationship consistency |
| 5 | **StateExtractor** | state_extractor | local-7b | 0.0 | Extract structured deltas after scene completion |

### P1 Agents (next wave)

| # | Agent | Role | Model | Temp | Purpose |
|---|-------|------|-------|------|---------|
| 6 | **Editor** | editor | haiku | 0.2 | Trim repetition, fix pacing, improve clarity |
| 7 | **Critic** | critic | haiku | 0.0 | Score scene usefulness (0.0-1.0), flag weak output |
| 8 | **World** | world_keeper | sonnet | 0.3 | Faction politics, world rules, lore consistency |
| 9 | **Arc** | arc_tracker | haiku | 0.2 | Track act progression, character arcs, plot threads |
| 10 | **Memory** | memory_keeper | local-7b | 0.0 | Maintain layered memory (character/scene/world/narrative) |

## Orchestration Flow

```
User triggers scene generation
        │
        ▼
Orchestrator.Plan(scene)
  → Reads scene.FlowType
  → Returns TurnOrder array
        │
        ▼
Orchestrator.Execute(plan, agentContext)
  ┌─────────────────────────────────────┐
  │ Phase 1: Scene Generation           │
  │                                     │
  │  Turn 1: Director (plan)            │
  │   → who acts, pressure, escalation  │
  │                                     │
  │  Turn 2: Character (perform)        │
  │   → dialogue/action per character   │
  │                                     │
  │  Turn 3: Narrator (narrate)         │
  │   → prose frame for the turn        │
  │                                     │
  │  Turn 4: Editor (refine) [optional] │
  │   → polish prose                    │
  │                                     │
  │  Turn 5: CanonGuard (validate)      │
  │   → flag canon violations           │
  │                                     │
  │  → repeat for N turns until done    │
  └─────────────────────────────────────┘
        │
        ▼
Orchestrator.RunFinish(scene)
  ┌─────────────────────────────────────┐
  │ Phase 2: Scene Finish               │
  │                                     │
  │  Turn N+1: StateExtract (extract)   │
  │   → character state, relationships, │
  │     timeline, canon deltas          │
  │                                     │
  │  Turn N+2: Critic (score)           │
  │   → scene usefulness score 0.0-1.0  │
  │                                     │
  │  Turn N+3: Director (evaluate)      │
  │   → scene complete? next steps      │
  └─────────────────────────────────────┘
```

## Turn Ordering by FlowType

| FlowType | Turn Sequence |
|----------|--------------|
| `monologue` | Director → Character(1) → Narrator → Editor → CanonGuard |
| `dialogue` | Director → Character(1) → Character(2) → Narrator → Editor → CanonGuard |
| `round_robin` | Director → (Character × N → Narrator → CanonGuard) × maxTurns |
| `parallel` | Director → Character(all) → Narrator → CanonGuard |
| `action` | Director → Character → Narrator → Editor |
| `silent` | Director → Narrator |

## Model Routing

| Agent | Model | Temp | Priority | MaxRetries | Timeout |
|-------|-------|------|----------|------------|---------|
| Director | claude-sonnet | 0.3 | high | 2 | 30s |
| Character | claude-sonnet | 0.8 | high | 2 | 60s |
| Narrator | claude-sonnet | 0.5 | high | 2 | 30s |
| Editor | claude-haiku | 0.2 | medium | 1 | 20s |
| CanonGuard | claude-haiku | 0.0 | medium | 1 | 15s |
| Critic | claude-haiku | 0.0 | low | 1 | 15s |
| StateExtract | local-7b | 0.0 | low | 1 | 30s |
| World | claude-sonnet | 0.3 | low | 1 | 30s |
| Arc | claude-haiku | 0.2 | low | 1 | 15s |
| Memory | local-7b | 0.0 | low | 1 | 15s |

## Required vs Optional

- **Director**: always required, always blocking
- **Character**: required for monologue/dialogue/action, optional for silent
- **Narrator**: always required for non-silent
- **Editor**: optional (skippable)
- **CanonGuard**: optional (skippable)
- **Critic**: runs after all turns, non-blocking
- **StateExtract**: runs on scene finish, non-blocking
- **World/Arc/Memory**: P1, not part of default turn plan

## Agent Context

Each agent receives an `AgentContext` with:

```go
type AgentContext struct {
    StoryID       string
    SceneID       string
    TurnID        string
    Story         *domain.Story
    Scene         *domain.Scene
    Characters    []*domain.Character
    CharStates    []*domain.CharacterState
    Bible         *domain.StoryBible
    Edges         []*domain.SceneEdge
    Turns         []*domain.SceneTurn
    Timeline      []*domain.TimelineEvent
    Memories      map[string][]*domain.CharacterMemory
    CanonDeltas   []*domain.CanonDelta
    Summaries     []*domain.Summary
    ParticipantIDs []string
}
```

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

See `internal/events/` for event type constants.
