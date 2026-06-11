# Repository review and enhancement plan

## Overview
This project already has a solid domain-oriented structure with a Go API layer, a React/Vite frontend, and a graph-first storytelling model. The main opportunity was to improve robustness around request handling and reduce repeated validation logic so the server layer becomes easier to extend.

## What was improved
1. Request validation was centralized for story and node handlers.
   - Story titles are now trimmed and rejected when blank.
   - UUID parsing for character references and optional location references now flows through shared helpers.
   - Scene-structure persistence now reports failures instead of silently ignoring them.

2. The backend build was stabilized.
   - The stale import issue in the lore service layer was removed so the Go build completes successfully again.

3. The API surface was made easier to verify locally.
   - A health endpoint was added at `/api/v1/healthz`.
   - The frontend API client now exposes a simple health check helper.

4. The repository gained regression coverage for the new validation helpers.
   - The tests cover invalid UUID lists and blank story titles.

## Files touched
- [internal/api/request_validation.go](../internal/api/request_validation.go)
- [internal/api/handlers_stories.go](../internal/api/handlers_stories.go)
- [internal/api/router.go](../internal/api/router.go)
- [internal/api/request_validation_test.go](../internal/api/request_validation_test.go)
- [web/src/api/client.ts](../web/src/api/client.ts)
- [docs/review-and-enhancements.md](review-and-enhancements.md)

## Recommended next steps
- Introduce a dedicated validation layer for all request payloads rather than handling them inline in handlers.
- Move domain-specific persistence and business rules behind service interfaces with clearer contracts.
- Expand test coverage around story, node, and scene handlers.
- Add end-to-end API smoke tests for the most critical flows.

## Strategic review: narrative platform direction
The project is significantly more ambitious than a typical side project. The current architecture already moves beyond a simple prompt-to-text generator and is building a true storytelling system with canon, story graphs, state, scene generation, summaries, and branching logic.

### Overall assessment
- Area: 9/10
- Vision: 9/10
- Domain modeling: 7.5/10
- Extensibility: 8/10
- AI architecture: 7/10
- Story modeling: 6/10
- Future agent readiness: 6/10
- Product evolution: 5/10

### Core observation
The largest gap is not code quality. It is that the system currently models scenes and characters, but it has not yet fully modeled narrative. That distinction matters because it separates a story engine from a text generator.

### Missing narrative domain
The current model covers nodes, edges, characters, locations, and state well enough for a graph-first story structure. The next layer should explicitly model:
- premise
- theme
- narrative arc
- conflict
- stakes
- story goals
- character arcs
- act structure
- plot threads
- foreshadowing
- mystery
- resolution
- timeline

### Missing narrative engine
The current flow is effectively:
```text
Story -> Scene -> Scene -> Scene
```

The next evolution should be:
```text
NarrativeEngine
├── Act I
├── Act II
├── Act III
├── Plot Threads
├── Character Arcs
├── Conflicts
└── Themes
```

### Character model evolution
The current character data is a good foundation, but stories are driven by change. The next iteration should add:
- character arc
- current belief
- false belief
- core wound
- fear
- need
- want
- secrets
- motivations
- growth stage

### Relationship model evolution
The current relationship structure is too weak for complex dramatic systems. A richer model should represent:
- trust
- affection
- fear
- respect
- dependency
- history

This allows asymmetric relationships such as one character trusting another while the other distrusts them.

### Timeline engine
A dedicated timeline layer is needed to avoid continuity issues such as a character dying in one chapter and appearing in another without explanation. A future timeline package should model:
- timeline events
- chronology
- story dates
- event dependencies

### World model expansion
Canon is currently useful but too flat. A larger world layer should eventually include:
- nations
- cities
- factions
- religions
- magic systems
- technologies
- cultures
- historical events

### Story graph evolution
The existing graph structure is solid, but it should evolve toward a layered structure:
```text
Story
└── Arc
    └── Thread
        └── Scene
```

That would better support quests, objectives, milestones, checkpoints, and plot threads.

### Memory layers
The current ledger is a strong start, but the system should also distinguish between:
- character memory
- story memory
- world memory
- conversation memory
- narrative memory

These represent different knowledge domains and should not be collapsed into a single state container.

### Versioning
The project already versions characters and locations well. The next step is to add versioning for:
- story versions
- scene versions
- timeline versions
- world versions

This would support alternate timelines and branching narrative experiments.

### Agent architecture direction
The current system is closer to a prompt builder than a multi-agent narrative platform. The eventual architecture should include roles such as:
- character agent
- narrator agent
- director agent
- editor agent
- critic agent
- world agent

A useful flow would be:
```text
Director -> Characters -> Scene -> Editor -> Critic
```

### Recommended restructuring
The highest-value architectural move would be to reorganize the codebase around clearer domain boundaries:
```text
domain/
├── story
├── narrative
├── world
├── character
├── timeline
├── memory

application/
├── generation
├── simulation
├── planning
├── branching

infrastructure/
├── postgres
├── river
├── llm

interfaces/
├── http
├── grpc
├── websocket
```

### Highest-value next build
The most impactful next feature would be a story planning layer before generation. A useful intermediate artifact would be a story blueprint containing:
- premise
- theme
- main conflict
- acts
- character arcs
- plot threads
- end state

Then the system could move from:
```text
Blueprint -> Story Graph -> Scenes -> Dialogue
```

### CTO-style priority order
1. Narrative engine
2. Timeline engine
3. Relationship engine
4. Character arc system
5. Story blueprint layer
6. Multi-agent architecture
7. World-building engine
8. Versioned story branches

### Bottom line
The repository already contains the skeleton of a serious storytelling platform. The next leap is to evolve from a story graph engine with generation capabilities into a narrative operating system where scenes are only one output of a much richer story model.
