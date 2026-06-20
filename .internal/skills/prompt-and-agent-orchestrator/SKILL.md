# Prompt & Agent Orchestrator

Centralize how prompts, agents, scene context, and orchestration decisions are built for story-builder.

## Trigger
- "create prompt for [agent]"
- "design agent orchestration"
- "refactor prompt assembly"
- "add new agent type"

## Prompt design rules

### Template structure
All prompts live in `internal/prompt/` with the prompt compiler.
Each template has:
- LayerID (unique string identifier)
- Content (Go template text)
- MergeStrategy (append/prepend/replace)

### Agent prompts
Each agent spec in `internal/agents/` has:
- SystemPrompt — defines role, constraints, output format
- Model tier assignment
- Temperature setting
- Max turns

### Context layering
The prompt compiler builds context in this order:
1. System prompt (role definition)
2. Character cards (personality, backstory, goals)
3. State (current emotional, physical, location)
4. Memories (recent, relevant, importance-filtered)
5. Bible (world rules, factions, lore)
6. Canon (current truth from pins/deltas)
7. Scene objective (beat intent, tone, POV)
8. Previous turns (scene history)
9. User instruction (what to do this turn)

## Agent orchestration rules

### Turn ordering by flow type
```text
monologue:   Director → Character → Narrator → Editor → CanonGuard
dialogue:    Director → Character → Character → Narrator → Editor → CanonGuard
round_robin: Director → (Character → Narrator → CanonGuard) × N
action:      Director → Character → Narrator → Editor
silent:      Director → Narrator
custom:      defined in Scene.SceneStructure
```

### Required vs optional agents
- Director: always required, always blocking
- Character: required for monologue/dialogue/action, optional for silent
- Narrator: always required for non-silent
- Editor: optional (skippable)
- CanonGuard: optional (skippable)
- Critic: runs after all turns, non-blocking
- StateExtract: runs on scene finish, non-blocking

### Model routing
| Agent          | Model        | Temp | Priority |
|----------------|-------------|------|----------|
| Director       | sonnet       | 0.3  | high     |
| Character      | sonnet       | 0.8  | high     |
| Narrator       | sonnet       | 0.5  | high     |
| Editor         | haiku        | 0.2  | medium   |
| CanonGuard     | haiku        | 0.0  | medium   |
| Critic         | haiku        | 0.0  | low      |
| StateExtract   | local-7b     | 0.0  | low      |
| World          | sonnet       | 0.3  | low      |
| Arc            | haiku        | 0.2  | low      |
| Memory         | local-7b     | 0.0  | low      |

## Output
- New prompt template with compiler integration
- New agent spec with system prompt and model config
- Updated orchestration plan
- Migration notes if changing existing prompt
