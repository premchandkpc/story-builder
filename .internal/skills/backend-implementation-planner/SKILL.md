# Backend Implementation Planner

Convert story-builder features into real implementation plans.

## Trigger
- "implement [feature]"
- "plan [feature]"
- "how to build [feature]"

## Method

Given a feature request, produce:

### 1. Schema changes
- New collections / fields / indexes
- Migration ordering

### 2. Domain model changes
- New types in `internal/domain/`
- Updated existing types

### 3. Repository interface changes
- New interfaces or methods in `internal/repository/interfaces.go`
- Repository is always interface-first, mongo impl second

### 4. Service changes
- New service or service methods in `internal/service/`
- Dependency wiring

### 5. LLM / prompt changes
- New prompt template in `internal/prompt/`
- Model routing rules in `internal/llm/types.go` PromptRegistry

### 6. Agent changes (if applicable)
- New agent spec in `internal/agents/`
- Orchestration plan update

### 7. Worker changes (if applicable)
- New worker in `internal/worker/`
- Pipeline integration in `internal/service/generation.go`

### 8. API routes
- New handlers in `internal/api/`
- Route registration in `internal/api/server.go`

### 9. Frontend changes
- New components, hooks, types
- API client updates

### 10. Tests
- Integration test scenarios
- Failure cases

### 11. Rollout plan
- Phased deployment
- Feature flag if risky
- Rollback strategy

## Example output

```text
Feature: multi-agent scene turns

1. Schema
   - scene_turns collection (see domain.SceneTurn)
   - agent_runs collection (see domain.AgentRun)
   - indexes: {sceneId: 1, number: 1}

2. Domain
   - internal/domain/scene_turn.go (SceneTurn, TurnStatus*, TurnRole*)
   - internal/domain/agent_run.go (AgentRun, AgentRunFilter)

3. Repo
   - SceneTurnRepository (Create, Get, Update, ListByScene, ListByRole)
   - AgentRunRepository (Create, List, DeleteByStory)

...
```
