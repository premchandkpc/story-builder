# Phase 7: Blueprint → Arc → Thread Planning

## Problem

Scene generation has no structural contract. A scene can be generated with `beatIntent: "Hero confronts villain"` but nothing ensures the scene advances the correct plot thread, hits the required conflict beat, or respects the character arc stage. The `StoryBlueprint` type exists but doesn't constrain generation.

## Target

A planning layer that formalizes:

```
Story
├── Blueprint (premise, theme, genre, ending)
│   ├── Acts (act 1/2/3 structure)
│   ├── CharacterArcs (each character's growth arc)
│   └── PlotThreads (mystery, romance, political, etc.)
└── ScenePlan (per scene)
    ├── Purpose (which arcs/threads it advances)
    ├── Required beats
    ├── Forbidden contradictions
    ├── Entry/Exit states for characters
    └── Conflict type
```

## 7.1 Domain Model

Extend `domain/blueprint.go`:

```go
type ScenePurpose struct {
    SceneID        string   `bson:"sceneId"`
    StoryID        string   `bson:"storyId"`
    AdvancingArcs  []string `bson:"advancingArcs,omitempty"`  // character arc IDs
    AdvancingThreads []string `bson:"advancingThreads,omitempty"` // plot thread IDs
    RequiredBeats  []BeatDef `bson:"requiredBeats,omitempty"`
    ForbiddenBeats []string `bson:"forbiddenBeats,omitempty"`
    EntryState     map[string]string `bson:"entryState,omitempty"` // charID → expected state
    ExitState      map[string]string `bson:"exitState,omitempty"`  // charID → guaranteed state
    ConflictType   string   `bson:"conflictType,omitempty"` // escalation | revelation | reversal | resolution
}

type BeatDef struct {
    Type        string `bson:"type"`        // "revelation" | "confrontation" | "retreat" | "alliance" | "betrayal"
    Description string `bson:"description"`
    Mandatory   bool   `bson:"mandatory"`
}
```

## 7.2 PlanScene Service

```go
// internal/service/planner.go
type PlannerService struct {
    StoryRepo    repository.StoryRepository
    SceneRepo    repository.SceneRepository
    EdgeRepo     repository.SceneEdgeRepository
    CharRepo     repository.CharacterRepository
    BlueprintRepo repository.BlueprintRepository
    LLM          llm.OutlineService
}

type ScenePlan struct {
    SceneID       string
    Purpose       domain.ScenePurpose
    ParticipantIntent map[string]string // charID → what they want in this scene
    EntryStatePreview string            // narrative summary of state before scene
    SuggestedTone     string
    SuggestedPOV      string
    SuggestedWords    int
}
```

**`PlanScene(ctx, sceneID)` flow:**

1. Load scene + graph position (upstream/downstream edges)
2. Load blueprint (acts, arcs, threads)
3. Determine which arcs should advance based on story position:
   - If scene is in Act 2 → protagonist arc should hit "crisis"
   - If scene is near resolution → threads should be resolving
4. Determine required beats based on:
   - Edge type (fork → decision beat, join → resolution beat)
   - Scene purpose (configured or LLM-suggested)
5. Determine entry state: read latest `CharacterView` for each participant
6. Produce `ScenePlan` with constraints

```go
func (p *PlannerService) PlanScene(ctx context.Context, sceneID string) (*ScenePlan, error) {
    scene, _ := p.SceneRepo.Get(ctx, sceneID)
    bp, _ := p.BlueprintRepo.GetByStory(ctx, scene.StoryID)
    edges, _ := p.EdgeRepo.ListByStory(ctx, scene.StoryID)

    plan := &ScenePlan{SceneID: sceneID}

    // Which arcs should advance based on story progress?
    incomingEdges := filterEdges(edges, func(e *domain.SceneEdge) bool { return e.ToSceneID == sceneID })
    outgoingEdges := filterEdges(edges, func(e *domain.SceneEdge) bool { return e.FromSceneID == sceneID })
    upstreamSceneIDs := collectFromIDs(incomingEdges)

    // Determine arc advancement
    plan.Purpose.AdvancingArcs = p.selectAdvancingArcs(bp, scene, upstreamSceneIDs)

    // Determine beats based on edge type + existing purpose
    if len(outgoingEdges) > 1 {
        plan.Purpose.ConflictType = "choice" // fork → decision point
        plan.Purpose.RequiredBeats = append(plan.Purpose.RequiredBeats, BeatDef{
            Type: "decision", Mandatory: true,
        })
    }

    // Determine participant intent via LLM (optional)
    if scene.BeatIntent != "" {
        plan.ParticipantIntent = p.suggestIntents(ctx, scene, scene.Participants)
    }

    return plan, nil
}
```

## 7.3 Scene Plan UI

### `ScenePlanPanel.tsx`

New tab in GraphPanel sidebar.

```
┌─────────────────────────────────┐
│ Plan                            │
├─────────────────────────────────┤
│ Purpose:                        │
│ ┌─────────────────────────────┐ │
│ │ Advancing Arcs:             │ │
│ │ ☑ Hero: "Learn to trust"   │ │
│ │ ☐ Villain: "Consolidate    │ │
│ │    power"                   │ │
│ │                             │ │
│ │ Advancing Threads:          │ │
│ │ ☑ Mystery: "Who killed the │ │
│ │    king?"                   │ │
│ │ ☐ Romance                  │ │
│ │                             │ │
│ │ Conflict Type:              │ │
│ │ [escalation ▾]              │ │
│ └─────────────────────────────┘ │
│                                 │
│ Required Beats:                 │
│ ┌─────────────────────────────┐ │
│ │ ★ Revelation: Hero learns   │ │
│ │   king's murderer           │ │
│ │ ★ Decision: Choose to trust │ │
│ │   or confront               │ │
│ │ + Add beat                  │ │
│ └─────────────────────────────┘ │
│                                 │
│ Forbidden:                      │
│ ┌─────────────────────────────┐ │
│ │ ✗ Hero dies                 │ │
│ │ ✗ Villain revealed too early│ │
│ │ + Add forbidden             │ │
│ └─────────────────────────────┘ │
│                                 │
│ Entry State:                    │
│ Hero: throne_room, fearful      │
│ Villain: (offstage)             │
│                                 │
│ [Generate with Plan ▸]          │
└─────────────────────────────────┘
```

### Wire in GraphPanel

```typescript
// web/src/components/GraphPanel.tsx
const TABS = [
    { key: "edit", label: "Edit" },
    { key: "info", label: "Info" },
    { key: "gen", label: "Generation" },
    { key: "plan", label: "Plan" }, // NEW
    { key: "turns", label: "Turns" },
    { key: "agents", label: "Agents" },
    { key: "runs", label: "Runs" },
]
```

## 7.4 Plan Integration with Generation

When `Generate with Plan` is clicked, the plan is passed to the pipeline as structured context:

```typescript
// API: POST /stories/{storyId}/nodes/{id}/generate
// Body can now include:
{
    "plan": {
        "advancingArcs": ["hero_trust"],
        "requiredBeats": [
            {"type": "revelation", "description": "Hero learns truth", "mandatory": true}
        ],
        "forbiddenBeats": ["hero_dies"],
        "conflictType": "escalation"
    }
}
```

Backend merges plan into prompt construction:

```go
// internal/service/context.go — ContextBuilder
func (b *ContextBuilder) Build(ctx context.Context, scene *domain.Scene, plan *domain.ScenePurpose) (*BuiltContext, error) {
    // ... existing context building ...

    // Add plan constraints
    if plan != nil {
        params.RequiredBeats = plan.RequiredBeats
        params.ForbiddenBeats = plan.ForbiddenBeats
        params.ConflictType = plan.ConflictType
        params.AdvancingArcs = plan.AdvancingArcs
    }

    return &BuiltContext{Params: params}, nil
}
```

## 7.5 Scene Version Diff

Backend comparison endpoint:

```go
// internal/service/diff.go
type DiffService struct {
    GenRepo     repository.GenerationRepository
    EventRepo   repository.NarrativeEventRepository
    RunRepo     repository.RunRepository
}

type GenDiff struct {
    GenAID     string
    GenBID     string
    ProseDiff  string // git-style line diff
    EventDiffs []EventDiff
    TokenDiff  struct {
        A int `json:"a"`
        B int `json:"b"`
    }
}

type EventDiff struct {
    EventType string                  `json:"eventType"`
    A         *NarrativeEventSnapshot `json:"a,omitempty"`
    B         *NarrativeEventSnapshot `json:"b,omitempty"`
}
```

**Endpoint:** `GET /stories/{id}/nodes/{id}/generations/{a}/diff?against={b}`

Enhance `GenerationCompare.tsx` to show event diffs:

```
Generation #1 (3.2k tokens)    │ Generation #2 (3.8k tokens)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┿━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
"The castle gates groaned..."  │ "The castle gates crashed open
...                             │ with thunderous force..."
                                │
Events:                         │ Events:
✓ location.changed: hero        │ ✓ location.changed: hero
✓ emotion.changed: hero         │ ✓ emotion.changed: hero
✗ emotion.changed: villain      │ ✓ emotion.changed: villain (NEW)
                                │ ✓ knowledge.added: hero (NEW)
```

## File Changes Summary

| File | Change |
|------|--------|
| `internal/domain/blueprint.go` | Add `ScenePurpose`, `BeatDef` |
| `internal/service/planner.go` (new) | `PlannerService.PlanScene()` |
| `internal/service/diff.go` (new) | `DiffService.GenDiff()` |
| `internal/service/context.go` | Accept optional plan in `Build()` |
| `internal/api/stories.go` | Add `GET nodes/{id}/plan`, `POST nodes/{id}/generate` with plan body |
| `internal/api/generations.go` | Add `GET .../diff?against=` endpoint |
| `web/src/components/ScenePlanPanel.tsx` (new) | Plan tab component |
| `web/src/components/GenerationCompare.tsx` | Show event diffs alongside prose |
| `web/src/components/GraphPanel.tsx` | Add Plan tab |
| `web/src/api/hooks.ts` | Add `useScenePlan`, `useGenDiff` |
| `web/src/api/client.ts` | Add plan/diff endpoints |
| `web/src/api/types.ts` | Add `ScenePlan`, `GenDiff`, `BeatDef` types |
