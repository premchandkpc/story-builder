# System Design Architect

Evolve story-builder from app to narrative platform.

## Trigger
- "design the [feature] system"
- "how should [X] scale"
- "what architecture for [feature]"
- "compare approaches for [Y]"

## Design decisions this skill owns

### Scene generation: pipeline vs agent orchestration
- Current: 6-worker sequential pipeline (`internal/service/generation.go`)
- Future: agent-based turn orchestration (`internal/agents/orchestrator.go`)
- Decision: keep pipeline for simple generation, route multi-agent scenes to orchestrator

### Canon versioning: append-only vs event-sourced
- Approach: CanonDelta as append-only log (like an event stream)
- `domain.CanonDelta` with category, fact, old/new value, confidence, source
- Story.CanonPins as the "current truth" projection from deltas
- Projection rebuilt on scene accept

### Timeline: ordered events vs graph
- Current: `domain.TimelineEvent` with Order field
- Future: Timeline as DAG with dependencies/consequences edges
- TimelineRepository already has Order field — extend with `DependsOn` and `Unlocks`

### Memory: layered vs single bag
- Current: `domain.CharacterMemory` with type/importance/embedding
- Future layers: character → scene → world → narrative
- Each layer has different retention, access scope, and agent visibility

## Output

```text
Feature: [name]

Architecture:
- package boundaries
- data flow
- persistence strategy
- agent topology

Trade-offs considered:
- option A: ...
- option B: ...
- chosen: ... because ...

Migration path:
1. ...
2. ...
3. ...
```
