# Deep Tech Mentor

Explain design decisions, graph engine, LLM orchestration, state model for story-builder.

## Trigger
- "explain [concept]"
- "how does [X] work"
- "why is [Y] designed this way"
- "teach me about [topic]"

## Topics this skill covers

### Graph engine (`internal/graph/`)
- DAG vs tree for narrative structure
- Kahn's algorithm for topological sort
- How cycle detection prevents generation on invalid graphs
- Edge types: seq, fork, join, choice, parallel
- Finding unreachable scenes and dead ends

### Scene orchestration
- Pipeline model (current): 6 sequential workers
- Agent model (future): Director/Character/Narrator orchestrator
- When to use each, how to transition

### State model
- Character (immutable definition) vs CharacterState (append-only per scene)
- State delta extraction pattern
- Relationship state as multi-axis (trust, respect, fear, affection)
- Canon pins as projected truth from deltas

### Memory architecture
- Layered: character scene world narrative
- Embedding-based retrieval via MongoDB Atlas Search
- Importance scoring for retention

### LLM pipeline
- Model tier routing: sonnet (generation), haiku (validation), local-7b (extraction)
- Circuit breaker pattern in `internal/llm/client.go`
- Prompt registry in `internal/llm/types.go` — single source of truth for prompts
- Context building in `internal/service/context.go`

### Concurrency
- `sync.Map` for in-flight generation locks
- Event bus for decoupled worker notifications
- Why no Kafka/River (intentional: simpler infra for current scale)

## Output
Clear explanations with code references. Always link to specific file:line.
