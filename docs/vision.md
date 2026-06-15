# Narrative OS — Platform Vision

> From story graph generator to Narrative Operating System: a reusable simulation platform for books, visual novels, RPGs, movies, anime, comics, interactive stories, AI NPC worlds, and 2D/3D scene generation — all from the same canonical story model.

---

## Why "Operating System"?

Most narrative AI projects model:

```text
Story → Chapters → Scenes
```

That is a **book structure**. It forces every output format into a linear, chapter-based mold.

A Narrative OS models:

```text
Universe → World → Timeline → Story → Scenario → Scene → Frame
```

Books become one output format. Movies become another. Games become another. Anime becomes another. All derived from the same canonical data.

---

## The Canonical Narrative Model

```
Universe                    Marvel, Naruto, Harry Potter, custom
    │
    ├── World              Earth-616, Earth-199999, Konoha
    │   ├── physics_rules
    │   ├── magic_rules
    │   ├── technology_rules
    │   ├── social_rules
    │   ├── economics
    │   └── cultures
    │
    ├── Timeline           Event-sourced sequence of world state
    │   ├── event_id
    │   ├── timestamp
    │   ├── entity_id
    │   ├── state_before
    │   └── state_after
    │
    ├── Story              Belongs to a world, references canon pins
    │   ├── genre, tone, rating, language
    │   ├── target_audience, render_type
    │   └── prompts (story, global, culture, safety, rendering)
    │
    ├── Scenario           Mission / quest / episode / arc
    │   ├── goal, actors, locations, outcomes
    │   └── constraints
    │
    ├── Scene              Aggregate root — the core unit
    │   ├── participants, actions, dialogues
    │   ├── camera, emotions, location, timeline
    │   ├── parent_scene, child_scene, parallel_scene
    │   └── reusable_scene, alternate_scene
    │
    └── Frame              Render-level atomic unit
        ├── camera_angle, distance, focus
        ├── lighting, character_positions
        └── objects, background
```

---

## Prompt Layering System

Prompts compile hierarchically like compiler stages:

```
Global Prompt           "You write PG-13 fantasy fiction"
    │
    ▼
Story Prompt            "A young hero discovers their lineage"
    │
    ▼
Culture Prompt          "Set in Mughal India — use period-appropriate dialogue"
    │
    ▼
Safety Prompt           "Avoid graphic violence, no sexual content"
    │
    ▼
Scene Prompt            "Luke meets Obi-Wan — mysterious tone, 500 words"
    │
    ▼
Character Prompt        "You are Obi-Wan: wise, cryptic, protective"
    │
    ▼
Memory Prompt           "Remember: you just defeated Darth Maul"
    │
    ▼
Compiled Prompt         Final assembled prompt sent to LLM
```

Each layer supports: `override`, `merge`, `append`, `replace`, `disable`.

---

## Character System

Three separate concerns:

### 1. Character Core (never changes)

```json
{
  "id": "hero",
  "name": "Arjun",
  "culture": "indian",
  "personality": "brave",
  "arc": "hero_journey"
}
```

### 2. Character State (changes every scene, event-sourced)

```json
{
  "scene": "S12",
  "health": 80,
  "stress": 20,
  "emotion": "anger",
  "relationships": {"princess": "loves", "villain": "hates"},
  "outfit": "armor",
  "inventory": ["sword", "shield"]
}
```

### 3. Character Memory (vector DB, semantic retrieval)

```
Everything the character has experienced, stored as embeddings.
Queried during prompt compilation to prevent character amnesia.
```

---

## Emotion Engine

| Layer | Description | Example |
|---|---|---|
| `displayed_emotion` | What others see | Smile |
| `inner_emotion` | True feeling | Anger |
| `suppressed_emotion` | Repressed | Revenge |

Supported emotions: anger, fear, joy, sadness, disgust, trust, anticipation, surprise.

---

## Actor Casting Engine

Separate **actor** from **character** (like movies):

```
Character               Darth Vader
    │
    ├── Casting          Mark Hamill (voice), David Prowse (body)
    │
    └── Suitability      Height: 198cm, Voice: bass, Age: 40-50
```

Actor types: human, asian, american, anime, 3d, game. Suitability scoring for casting recommendations.

---

## Culture Engine

One scene, different cultural renderings:

| Culture | Greeting | Gesture | Attire |
|---|---|---|---|
| India | Namaste | Folded hands | Kurta |
| Japan | Bow | 30° angle | Kimono |
| USA | Handshake | Firm grip | Suit |
| Korea | Bow | Slight nod | Hanbok |

The underlying narrative beat stays the same. Only the cultural expression changes.

---

## Camera + Render System

```json
{
  "scene": "S12",
  "frames": [
    {
      "camera_angle": "closeup",
      "camera_distance": "near",
      "lighting": "sunset",
      "focus": "hero",
      "characters": {"hero": {"x": 0, "y": 0}},
      "objects": []
    }
  ]
}
```

Supports render styles: cinematic, anime, comic, realistic. Feeds into Stable Diffusion, Flux, Sora, Runway, etc.

---

## Generation Pipeline

```
Request
    │
    ▼
Prompt Compiler
    │  Global + Story + Culture + Safety + Scene + Character + Memory
    ▼
Context Builder
    │  Canon, state, lore, relationships, timeline
    ▼
Memory Retrieval
    │  Qdrant semantic search — character memories, relevant lore
    ▼
Culture Layer
    │  Region-aware expression transformation
    ▼
Emotion Layer
    │  Inject inner/displayed/suppressed emotion state
    ▼
LLM
    │  Sonnet / Haiku / Local
    ▼
Validator Pipeline
    │  Character → Timeline → Lore → Dialogue
    ▼
Storage
    │  Scene text → generations table
    │  State deltas → character_state event stream
    │  Memories → Qdrant
```

---

## Validation Pipeline

| Validator | Checks |
|---|---|
| Character | Dead speaking, age mismatch, personality break |
| Timeline | Future event, time overlap, ordering violation |
| Lore | Magic contradiction, world rule break, physics violation |
| Dialogue | Wrong language, wrong culture, wrong emotion |

---

## Data Store Strategy

| Store | Purpose |
|---|---|
| **PostgreSQL** | Transactional: stories, chapters, scenes, users, generation jobs, audit logs |
| **MongoDB** | Documents: character profiles, scene templates, prompt templates, world definitions, culture definitions, emotion definitions |
| **Neo4j** | Relationships: character↔character, character↔scene, scene↔scene, story↔character |
| **Qdrant** | Vectors: story embeddings, scene embeddings, character memories, dialogue embeddings, culture embeddings |
| **Redis** | Speed: prompt cache, context cache, story cache, distributed lock, rate limiter, event dedupe |
| **Kafka** | Events: StoryCreated, SceneGenerated, CharacterChanged, EmotionChanged, TimelineChanged |

---

## Bounded Contexts (Services)

```
┌─────────────────────────────────────────────────────────────┐
│                     Modular Monolith                        │
│                                                             │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────┐  │
│  │  Story    │ │  Scene   │ │ Character│ │  Timeline    │  │
│  │  Service  │ │  Service │ │ Service  │ │  Service     │  │
│  └──────────┘ └──────────┘ └──────────┘ └──────────────┘  │
│                                                             │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────┐  │
│  │  Memory   │ │  Emotion │ │  Culture │ │  Prompt      │  │
│  │  Service  │ │  Service │ │  Service │ │  Compiler    │  │
│  └──────────┘ └──────────┘ └──────────┘ └──────────────┘  │
│                                                             │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────┐  │
│  │  Casting  │ │  Render  │ │Generation│ │  Validation  │  │
│  │  Service  │ │  Service │ │ Service  │ │  Service     │  │
│  └──────────┘ └──────────┘ └──────────┘ └──────────────┘  │
│                                                             │
│  ┌──────────┐ ┌──────────┐                                  │
│  │  Search   │ │Analytics │                                  │
│  │  Service  │ │ Service  │                                  │
│  └──────────┘ └──────────┘                                  │
└─────────────────────────────────────────────────────────────┘
```

Start as modular monolith. Split services only after real load data demands it.

---

## Event Catalog

| Event | Producer | Consumers |
|---|---|---|
| StoryCreated | Story Service | Timeline, Search, Analytics |
| ChapterCreated | Chapter Service | Timeline, Search |
| SceneCreated | Scene Service | Timeline, Character, Prompt |
| SceneGenerated | Generation Service | Validation, Memory, Analytics |
| CharacterChanged | Character Service | Emotion, Memory, Timeline |
| EmotionChanged | Emotion Service | Character, Prompt |
| TimelineChanged | Timeline Service | Validation, Scene |
| DialogueGenerated | Generation Service | Validation, Memory |
| RenderGenerated | Render Service | Storage, Analytics |

---

## Implementation Roadmap

**Critical ordering principle:** Domain before infrastructure. Kafka (#11), Neo4j (#10), Qdrant (#9) come after the core narrative engine works. The real hard problem is: *"Can character A remain emotionally consistent through 200 scenes across multiple timelines?"* Solve that first. Databases are implementation details.

### Phase 1 — Domain Cleanup + Core Aggregates

Clean package boundaries. Eliminate `common`/`utils`/`shared`/`helpers` garbage dumps. Define concrete Go types:

```go
type Story struct {
    ID        string
    Metadata  StoryMetadata
    Chapters  []ChapterRef
}

type CharacterDefinition struct {
    ID          string
    Name        string
    Personality PersonalityTraits
    Culture     CultureRef
    Arc         CharacterArc
    // Immutable core — never changes
}

type CharacterState struct {
    SceneID       string
    CharacterID   string
    Health        int
    Emotion       EmotionState
    Stress        int
    Outfit        string
    Inventory     []string
    Relationships map[string]string
}

type SceneEdge struct {
    SourceID string
    TargetID string
    EdgeType string // parent, child, parallel, alternate, reusable
}
```

### Phase 2 — Prompt Compiler

Highest-leverage new component. Hierarchical prompt layering:

```go
type PromptCompiler interface {
    Compile(ctx context.Context, req CompileRequest) (*CompiledPrompt, error)
}

type CompileRequest struct {
    Story     StoryRef
    Chapter   ChapterRef
    Scene     SceneRef
    Character CharacterRef
    Culture   CultureRef
    Memory    []MemoryRef
}

type CompiledPrompt struct {
    System    string
    Messages  []Message
    Model     ModelTier
    MaxTokens int
}
```

Layers: Global → Story → Chapter → Scenario → Scene → Frame → Character.
Merge strategies: `override`, `merge`, `append`, `replace`, `disable`.

### Phase 3 — Timeline Engine

Before vector DB, before agents, before graph DB — timeline first.

```go
type TimelineEvent struct {
    ID          string
    Time        int64
    EntityID    string
    EventType   string
    Payload     map[string]any
    StateBefore map[string]any
    StateAfter  map[string]any
}
```

Event types: `CharacterBorn`, `CharacterMoved`, `CharacterDied`, `RelationshipChanged`.
Scene N state is reconstructable from events 1..N-1.

### Phase 4 — Character State Engine

Per-scene state snapshots prevent LLM hallucination:
- Scene 1: "blue shirt" → Scene 10: "blue shirt" → Scene 20: "purple dragon armor (acquired in scene 15)"
- Without this, outfits change randomly across scenes.

### Phase 5 — Scene Graph + Validation

- Scene parent/child/parallel/alternate edges
- 4 validators: Character (dead speaking), Timeline (ordering), Lore (canon), Dialogue (culture/emotion)

### Phase 6 — MongoDB Migration

Only after domains are stable. Move document-heavy entities: prompt templates, character profiles, scene templates, world definitions, culture definitions.
Keep transactional data (stories, chapters, users) in Postgres.

### Phase 7 — Redis Caching

Cache compiled prompts, character state, scene state, story context.
Key pattern: `story:{id}`, `scene:{id}`, `character:{id}`.

### Phase 8 — Qdrant Vector Memory

Collections: `character_memory`, `story_memory`, `scene_memory`, `dialog_memory`.
Solves semantic retrieval across the entire narrative.

### Phase 9 — Neo4j Relationship Graph

Only when JSONB relationship queries become painful.
Graph: Character↔Character, Character↔Scene, Scene↔Scene.

### Phase 10 — Kafka Event Bus

Events: `StoryCreated`, `SceneCreated`, `SceneGenerated`, `CharacterUpdated`, `EmotionChanged`, `TimelineUpdated`.
Consumers: Analytics, Memory, Search, Rendering.

### Phase 11 — Multi-Agent Orchestration

Agents: Story Planner, Scene Planner, Dialogue Writer, Emotion Validator, Continuity Validator.
Pipeline: Planner → Scene Generator → Dialogue Generator → Emotion Check → Timeline Check → Lore Check → Store.

### Phase 12 — Rendering Pipeline

Input: scene ID. Output: camera, lighting, character positions, objects, background.
Format-independent. Consumable by Stable Diffusion, Flux, Sora, Unreal, Unity.

### Anti-patterns to avoid

- **Shiny technology first.** Don't add Kafka, Neo4j, Qdrant, MongoDB, or agents until the domain demands it.
- **God Character struct.** Never embed everything in one type. Split into Definition (immutable) / State (per-scene) / Memory (vector).
- **common/utils/shared packages.** These become garbage dumps. Each domain gets its own package.
- **Worker direct SQL access.** River workers call service interfaces, not raw queries.
- **Duplicated prompt compilation.** Single PromptCompiler for every scene.
- **Synchronous service coupling.** Design for events even in the monolith (Go channels or in-memory event bus until Kafka).

---

## Testing Strategy

| Type | Tool | Scope |
|---|---|---|
| Unit | Go Test, Testify | All services, all methods |
| Integration | Testcontainers | Postgres, Mongo, Redis, Kafka, Qdrant |
| Contract | Pact | Service boundaries |
| API | Playwright | REST + gRPC endpoints |
| BDD | Cucumber/Godog | Generation scenarios, memory retrieval, relationship updates |
| Load | K6 | Scene generation, memory retrieval, search |

---

## Observability

Mandatory stack: OpenTelemetry → Prometheus → Grafana + Tempo + Loki + Jaeger.

Track per request: `story_id`, `scene_id`, `character_id`, `generation_id`, `workflow_id`.

Key metrics:
- Prompt compilation latency
- Token usage per generation
- Generation failure rate
- Hallucination / violation rate
- Cache hit rate (Redis)
- Scene generation time (p95)
