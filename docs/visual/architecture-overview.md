# Architecture Overview

## System Context Diagram

```mermaid
graph TB
    subgraph Frontend["React Flow UI (web/)"]
        RF[React Flow Canvas]
        SP[Side Panel]
        API[API Client]
    end

    subgraph Backend["Go Server (cmd/server/)"]
        MW[Middleware<br/>Logger, CORS, Rate Limit]
        AH[API Handlers<br/>internal/api/]
        SV[Services<br/>internal/service/]
        AG[Agents<br/>internal/agents/]
        SC[Scene Turns<br/>internal/scene/]
        GR[Graph Engine<br/>internal/graph/]
        LLM[LLM Pipeline<br/>internal/llm/]
        PM[Prompt Compiler<br/>internal/prompt/]
        WK[Workers<br/>internal/worker/]
        EV[Event Bus<br/>internal/events/]
    end

    subgraph Storage["Data Layer"]
        MG[MongoDB<br/>SSOT]
        RD[Redis<br/>Cache/Locks/RateLimit]
    end

    subgraph External["External Services"]
        AN[Anthropic API]
        OL[Ollama Local]
    end

    RF --> SP
    SP --> API
    API --> AH
    AH --> MW
    AH --> SV
    SV --> GR
    SV --> LLM
    SV --> SC
    SV --> AG
    SV --> EV
    AG --> LLM
    AG --> WK
    LLM --> PM
    SV --> MG
    SV --> RD
    LLM --> AN
    LLM --> OL
```

## Package Dependency Graph

```mermaid
graph TD
    MAIN[cmd/server/main.go] --> API[internal/api]
    MAIN --> REPO[internal/repository/mongo]
    MAIN --> CACHE[internal/cache]
    MAIN --> WORKER[internal/worker]
    MAIN --> AGENTS[internal/agents]

    API --> SVC[internal/service]
    API --> DOMAIN[internal/domain]

    SVC --> REPO[internal/repository]
    SVC --> LLM[internal/llm]
    SVC --> EVENTS[internal/events]
    SVC --> GRAPH[internal/graph]
    SVC --> SCENE[internal/scene]

    AGENTS --> DOMAIN
    AGENTS --> LLM
    AGENTS --> EVENTS

    SCENE --> DOMAIN

    WORKER --> LLM
    WORKER --> REPO
    WORKER --> DOMAIN

    LLM --> PROMPT[internal/prompt]
    LLM --> CACHE

    GRAPH --> DOMAIN

    REPO --> DOMAIN
```

## Scene Generation Flow

```mermaid
sequenceDiagram
    participant User
    participant API as API Handler
    participant GenSvc as GenerationService
    participant Orch as AgentOrchestrator
    participant LLM as LLM Router

    User->>API: POST /generate {sceneId}
    API->>GenSvc: Generate(sceneId)
    GenSvc->>GenSvc: Create Generation doc
    GenSvc->>GenSvc: spawn goroutine

    critical Pipeline Phase
        GenSvc->>GenSvc: ContextBuilder.Build()
        Note over GenSvc: 20k token context

        alt Scene has SceneStructure
            GenSvc->>Orch: Plan(scene)
            Orch-->>GenSvc: TurnOrder

            loop Each Turn
                Orch->>LLM: Complete(turn prompt)
                LLM-->>Orch: Response
                Orch->>Orch: Record Turn
            end

            Orch->>Orch: RunFinish()
            Orch-->>GenSvc: OrchestrationResult
        else Simple scene
            GenSvc->>LLM: GenerateScene (sonnet)
            LLM-->>GenSvc: prose
            GenSvc->>LLM: ExtractState (local-7b)
            LLM-->>GenSvc: deltas
            GenSvc->>GenSvc: Memory/Timeline/Summary/Validate
        end
    end

    GenSvc-->>API: Generation doc (status=running)
    API-->>User: {id: gen_1}

    Note over GenSvc: ... polling for status ...

    User->>API: GET /generations/{id}
    API->>GenSvc: GetGeneration(id)
    GenSvc-->>API: {status: success}
    API-->>User: {output: "..."}
```

## Agent Turn Sequence

```mermaid
sequenceDiagram
    participant Orch as Orchestrator
    participant Dir as Director Agent
    participant Char as Character Agent
    participant Narr as Narrator Agent
    participant Ed as Editor Agent
    participant Guard as CanonGuard
    participant Critic as Critic Agent
    participant Extract as StateExtractor

    Orch->>Orch: Plan(scene)
    Note over Orch: Based on scene.FlowType

    Orch->>Dir: Run(plan)
    Dir-->>Orch: <whoActs, pressure, escalation>

    Orch->>Char: Run(perform)
    Char-->>Orch: <action/dialogue>

    alt Dialogue Flow
        Orch->>Char: Run(respond)
        Char-->>Orch: <response>
    end

    Orch->>Narr: Run(narrate)
    Narr-->>Orch: <prose frame>

    opt Editor available
        Orch->>Ed: Run(refine)
        Ed-->>Orch: <polished text>
    end

    opt CanonGuard available
        Orch->>Guard: Run(validate)
        Guard-->>Orch: <violations>
    end

    Note over Orch: Repeat for N turns

    Orch->>Orch: RunFinish(scene)

    Orch->>Extract: Run(extract)
    Extract-->>Orch: <state deltas>

    Orch->>Critic: Run(score)
    Critic-->>Orch: <score: 0.72>

    Orch->>Dir: Run(evaluate)
    Dir-->>Orch: <complete, next steps>
```

## Agent Data Flow

```mermaid
graph LR
    subgraph Input["AgentContext"]
        S[Story]
        SC[Scene]
        CH[Characters]
        CS[CharStates]
        B[Bible]
        M[Memories]
        T[Timeline]
    end

    subgraph Agents["Runtime Agents"]
        DIR[Director]
        CAR[Character]
        NAR[Narrator]
        ED[Editor]
        CG[CanonGuard]
        CR[Critic]
        SE[StateExtract]
        WO[World]
        AR[Arc]
        ME[Memory]
    end

    subgraph Output["Agent Decisions"]
        D1[whoActs, pressure, endScene]
        D2[dialogue, action, emotion]
        D3[prose frame, pace, POV]
        D4[revisions]
        D5[violations]
        D6[score, critiques]
        D7[state deltas]
        D8[worldViolations]
        D9[arc health]
        D10[memory summary]
    end

    Input --> DIR
    Input --> CAR
    Input --> NAR
    Input --> ED
    Input --> CG
    Input --> CR
    Input --> SE
    Input --> WO
    Input --> AR
    Input --> ME

    DIR --> D1
    CAR --> D2
    NAR --> D3
    ED --> D4
    CG --> D5
    CR --> D6
    SE --> D7
    WO --> D8
    AR --> D9
    ME --> D10
```

## Canon Versioning Flow

```mermaid
graph LR
    subgraph SceneExec["Scene Execution"]
        GEN[Generation] --> ACC[Accept]
    end

    subgraph Canon["Canon System"]
        CD[CanonDelta]
        CP[CanonPins]
    end

    ACC --> CD
    CD --> |projection| CP

    subgraph Consumers["Consumers"]
        CB[ContextBuilder]
        CG2[CanonGuard Agent]
        VL[ValidationWorker]
    end

    CP --> CB
    CP --> CG2
    CP --> VL
```

## Data Model: Story Entity Relationships

```mermaid
erDiagram
    Story ||--o{ Scene : contains
    Story ||--o| Bible : has
    Story ||--o{ Chapter : organized_by
    Story ||--o{ Location : set_in
    Story ||--o{ Character : features
    Story ||--o{ TimelineEvent : traces
    Story ||--o{ Summary : summarized_by
    Story ||--o{ CanonDelta : canon_changes

    Scene ||--o{ SceneEdge : connects
    Scene ||--o{ Generation : generated_as
    Scene ||--o{ SceneTurn : has_turns
    Scene ||--o{ CharacterState : captures_state
    Scene ||--o{ CharacterMemory : creates_memories

    Character ||--o{ CharacterState : evolves
    Character ||--o{ CharacterMemory : remembers
    Character ||--o{ Relationship : relates_to

    AgentRun ||--o| Scene : runs_in
    AgentRun ||--o| SceneTurn : produces

    CanonDelta ||--o| Scene : from_scene
    CanonDelta ||--o| Generation : from_generation
```
