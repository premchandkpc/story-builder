# Phase 2: Narrative Event Log + State Projections

## Problem

Current pipeline produces state changes through `ExtractStateWorker`, which writes directly to `character_state` via `StateRepo.Append()`. This is a fire-and-forget mutation with no audit trail, no replay, no branching support. The `canon_deltas` collection records some changes, but nothing consumes it — it's a log without readers. `NarrativeEvent` domain type exists but nothing emits or consumes it.

## Target

An append-only `narrative_events` log becomes the source of truth. Downstream projections (`character_view`, `timeline_view`) derive from it. The validator gate approves/rejects events before they enter the log.

## Flow

```
Scene generated (prose output)
        │
        ▼
ExtractState → produces []CharacterState (existing behavior)
        │
        ▼
EventExtractor → converts states to []NarrativeEvent candidates
        │
        ▼
Validator Gate → approves/rejects each candidate
        │
        ▼
Accepted events → appended to narrative_events collection
        │
        ▼
Projections → CharacterView, TimelineView (rebuild on read if stale)
```

## Detailed Design

### 1. `internal/event/` — New Package

```
internal/event/
  extractor.go     NarrativeEventExtractor — converts generation output + states → events
  validator.go     EventValidator — rules engine
  rules/           Individual validation rules
    dead_char.go
    timeline.go
    location.go
    bounds.go
    duplicate.go
```

### 2. EventExtractor

Converts the output of `ExtractStateWorker` (and existing `CharacterState` docs) into `NarrativeEvent` documents.

**Input:** `[]domain.CharacterState` (from current pipeline step 2)
**Output:** `[]domain.NarrativeEvent`

```go
// internal/event/extractor.go
type EventExtractor struct {
    CharRepo repository.CharacterRepository
}

func (e *EventExtractor) ExtractFromStates(ctx context.Context, storyID, sceneID, runID, genID string, states []domain.CharacterState) []domain.NarrativeEvent {
    var events []domain.NarrativeEvent
    for _, s := range states {
        events = append(events, e.extractCharacterEvents(storyID, sceneID, runID, s)...)
    }
    // Also extract timeline events from scene metadata
    events = append(events, e.extractTimelineEvents(storyID, sceneID, runID, genID)...)
    return events
}
```

**Character state → event mapping:**

| CharacterState.Field | NarrativeEvent.Type |
|---|---|
| `location` changed | `character.location.changed` |
| `mood` changed | `character.emotion.changed` |
| `activeGoal` changed | `character.goal.updated` |
| `knowledge` added | `character.knowledge.added` |
| `health` changed | `character.health.changed` |
| `RelationshipData` present | `relationship.trust.changed` |

**Dual write for backward compatibility:**
Existing consumers read from `character_state`. The event log is written alongside. Remove direct `character_state` reads only after projections are stable.

```go
// Write both during transition period
for _, state := range states {
    _ = stateRepo.Append(ctx, &state)                       // existing
    events := extractor.ExtractFromStates(ctx, ..., state)  // new
    for _, event := range events {
        _ = eventRepo.Append(ctx, &event)                   // new
    }
}
```

### 3. EventValidator

Gate that approves or rejects each event candidate before append.

```go
// internal/event/validator.go
type EventValidator struct {
    rules []EventValidationRule
}

type EventValidationRule interface {
    Name() string
    Validate(ctx context.Context, event *domain.NarrativeEvent, state *StoryState) *EventViolation
}

type EventViolation struct {
    RuleName string // e.g. "dead_character_cannot_act"
    Severity string // "reject" | "warn" | "info"
    Reason   string // human-readable explanation
}

type StoryState struct {
    Characters  map[string]*domain.CharacterState // latest state per character
    Timeline    []domain.TimelineEvent
    Scene       *domain.Scene
}
```

**Rule implementations:**

```go
// rules/dead_char.go
type DeadCharacterCannotAct struct{}

func (r *DeadCharacterCannotAct) Name() string { return "dead_character_cannot_act" }

func (r *DeadCharacterCannotAct) Validate(ctx context.Context, event *domain.NarrativeEvent, state *StoryState) *EventViolation {
    if event.SubjectType != domain.NarrativeSubjectChar {
        return nil
    }
    if event.EventType != "character.location.changed" && event.EventType != "character.goal.updated" {
        return nil // only relevant for physical actions
    }
    charState, ok := state.Characters[event.SubjectID]
    if !ok || charState.Health <= 0 {
        return &EventViolation{
            Severity: "reject",
            Reason:   fmt.Sprintf("character %s is dead (health=%d)", event.SubjectID, charState.Health),
        }
    }
    return nil
}

// rules/timeline.go
type TimelineMonotonicity struct{}

func (r *TimelineMonotonicity) Validate(ctx context.Context, event *domain.NarrativeEvent, state *StoryState) *EventViolation {
    if event.EventType != domain.NarrativeEventTypeTimeline {
        return nil
    }
    sceneOrder := state.Scene.TimelinePosition
    if len(state.Timeline) > 0 {
        lastOrder := state.Timeline[len(state.Timeline)-1].Order
        if sceneOrder < lastOrder {
            return &EventViolation{
                Severity: "reject",
                Reason:   fmt.Sprintf("scene order %d precedes last timeline event order %d", sceneOrder, lastOrder),
            }
        }
    }
    return nil
}
```

**Validator gate call in pipeline:**

```go
// After ExtractState step succeeds
state := loadStoryState(ctx, storyID, sceneID)
var accepted, rejected []domain.NarrativeEvent
for _, event := range candidates {
    var hasViolation bool
    for _, rule := range validator.Rules() {
        if violation := rule.Validate(ctx, event, state); violation != nil {
            if violation.Severity == "reject" {
                hasViolation = true
            }
            slog.Warn("event validation", "rule", rule.Name(), "severity", violation.Severity, "reason", violation.Reason)
        }
    }
    if hasViolation {
        rejected = append(rejected, event)
    } else {
        accepted = append(accepted, event)
    }
}
for _, event := range accepted {
    _ = eventRepo.Append(ctx, &event)
}
// Store rejected events in RunStep artifacts
stepCtx.Artifacts["rejected_events"] = rejected
stepCtx.Artifacts["accepted_events_count"] = len(accepted)
```

### 4. `internal/projection/` — State Projections

```go
// internal/projection/character_view.go
type CharacterProjection struct {
    CharRepo repository.CharacterRepository
    EventRepo repository.NarrativeEventRepository
    ViewRepo repository.CharacterViewRepository
    Cache    *cache.RedisCache // optional
}

// EnsureLatest rebuilds the view if stale (version < latest event version)
func (p *CharacterProjection) EnsureLatest(ctx context.Context, storyID, charID string) (*domain.CharacterView, error) {
    view, err := p.ViewRepo.Get(ctx, charID)
    if err != nil || view == nil {
        return p.rebuild(ctx, storyID, charID)
    }
    latestVersion, err := p.EventRepo.LatestVersion(ctx, storyID)
    if err != nil || view.Version < latestVersion {
        return p.rebuild(ctx, storyID, charID)
    }
    return view, nil
}

func (p *CharacterProjection) rebuild(ctx context.Context, storyID, charID string) (*domain.CharacterView, error) {
    events, err := p.EventRepo.ListBySubject(ctx, storyID, charID, 0)
    if err != nil {
        return nil, err
    }
    view := &domain.CharacterView{
        CharacterID: charID,
        StoryID:     storyID,
    }
    for _, event := range events {
        applyEvent(view, &event)
    }
    if len(events) > 0 {
        view.Version = events[len(events)-1].Version
    }
    view.UpdatedAt = time.Now()
    _ = p.ViewRepo.Upsert(ctx, view)
    return view, nil
}

func applyEvent(view *domain.CharacterView, event *domain.NarrativeEvent) {
    view.EventIDs = append(view.EventIDs, event.ID)
    switch event.EventType {
    case domain.NarrativeEventTypeCharLocation:
        if loc, ok := event.Payload["location"].(string); ok {
            view.CurrentState.Location = loc
        }
    case domain.NarrativeEventTypeCharEmotion:
        if mood, ok := event.Payload["mood"].(string); ok {
            view.CurrentState.EmotionalState = mood
        }
    case domain.NarrativeEventTypeCharKnowledge:
        if knowledge, ok := event.Payload["knowledge"].(string); ok {
            view.CurrentState.Knowledge = append(view.CurrentState.Knowledge, knowledge)
        }
    // ... additional event types
    }
}
```

**Domain types for views:**

```go
// internal/domain/character_view.go
type CharacterView struct {
    CharacterID  string                 `bson:"_id"`
    StoryID      string                 `bson:"storyId"`
    CurrentState CharacterStateSnapshot `bson:"currentState"`
    EventIDs     []string               `bson:"eventIds"`
    Version      int64                  `bson:"version"`
    UpdatedAt    time.Time              `bson:"updatedAt"`
}

type CharacterStateSnapshot struct {
    Location        string   `bson:"location,omitempty"`
    Health          int      `bson:"health,omitempty"`
    EmotionalState  string   `bson:"emotionalState,omitempty"`
    Mood            string   `bson:"mood,omitempty"`
    ActiveGoal      string   `bson:"activeGoal,omitempty"`
    Knowledge       []string `bson:"knowledge,omitempty"`
    Relationships   []RelSnapshot `bson:"relationships,omitempty"`
}

type RelSnapshot struct {
    TargetID   string  `bson:"targetId"`
    Trust      float64 `bson:"trust"`
    Respect    float64 `bson:"respect"`
    Fear       float64 `bson:"fear"`
    Affection  float64 `bson:"affection"`
}
```

### 5. Repository Additions

`internal/repository/interfaces.go`:

```go
type NarrativeEventRepository interface {
    Append(ctx context.Context, e *domain.NarrativeEvent) error
    AppendMany(ctx context.Context, events []*domain.NarrativeEvent) error
    ListByStory(ctx context.Context, storyID string, limit int) ([]*domain.NarrativeEvent, error)
    ListByScene(ctx context.Context, sceneID string, limit int) ([]*domain.NarrativeEvent, error)
    ListBySubject(ctx context.Context, storyID, subjectID string, limit int) ([]*domain.NarrativeEvent, error)
    LatestVersion(ctx context.Context, storyID string) (int64, error)
    DeleteByStory(ctx context.Context, storyID string) error
}

type CharacterViewRepository interface {
    Get(ctx context.Context, charID string) (*domain.CharacterView, error)
    Upsert(ctx context.Context, view *domain.CharacterView) error
    ListByStory(ctx context.Context, storyID string) ([]*domain.CharacterView, error)
    DeleteByStory(ctx context.Context, storyID string) error
}
```

### 6. Pipeline Integration

In `internal/service/generation_job_worker.go`, after `StepExtract` succeeds:

```go
// Current:
if !w.runStep(ctx, gen.ID, domain.StepExtract, func(sCtx context.Context) error {
    return extractWorker.Work(sCtx, worker.ExtractStateArgs{...})
}) { anyFailed = true }

// Add after extraction succeeds:
if states, err := w.cfg.StateRepo.ListByScene(ctx, scene.ID); err == nil && len(states) > 0 {
    candidates := w.cfg.EventExtractor.ExtractFromStates(ctx, scene.StoryID, scene.ID, job.RunID, gen.ID, states)
    accepted, rejected := w.cfg.EventValidator.Filter(ctx, candidates, loadStoryState(ctx, scene.StoryID, scene.ID))
    for _, event := range accepted {
        _ = w.cfg.EventRepo.Append(ctx, &event)
    }
    if len(rejected) > 0 {
        slog.Warn("narrative events rejected", "count", len(rejected), "sceneId", scene.ID)
        // Store rejected in run step artifacts
    }
    // Invalidate projection cache
}
```

### 7. Collections / Indexes

Already exists in `EnsureIndexes()`:
```go
"narrative_events": {
    {Keys: bson.D{{Key: "storyId", Value: 1}, {Key: "createdAt", Value: -1}}},
    {Keys: bson.D{{Key: "sceneId", Value: 1}, {Key: "createdAt", Value: -1}}},
    {Keys: bson.D{{Key: "eventType", Value: 1}}},
},
```

Add:
```go
"character_views": {
    {Keys: bson.D{{Key: "storyId", Value: 1}}},
},
```

Add index for subject-based lookups:
```go
"narrative_events": {
    // ...existing...
    {Keys: bson.D{{Key: "storyId", Value: 1}, {Key: "subjectId", Value: 1}, {Key: "createdAt", Value: -1}}},
},
```

### 8. Backward Compatibility

During transition:
1. **Dual write**: ExtractState writes to `character_state` (existing) AND EventExtractor writes `narrative_events` (new)
2. **Read fallback**: projection view reads from `narrative_events` first. If empty, fall back to `character_state` scan
3. **Gradual migration**: After projections are stable, switch reads to projections, then remove dual write
4. **Feature flag**: `NARRATIVE_EVENTS_ENABLED=true` env var to control dual write

### 9. Testing

```go
// internal/event/extractor_test.go
func TestExtractCharacterLocationChange(t *testing.T) {
    state := domain.CharacterState{
        CharacterID: "char_1", Location: "throne_room",
        Changes: map[string]any{"location": "dungeon"},
    }
    events := extractor.ExtractFromStates(ctx, "story_1", "scene_1", "run_1", state)
    assert.Len(t, events, 1)
    assert.Equal(t, "character.location.changed", events[0].EventType)
}

func TestValidatorRejectsDeadCharacter(t *testing.T) {
    charState := map[string]*domain.CharacterState{
        "char_1": {CharacterID: "char_1", Health: 0},
    }
    event := domain.NarrativeEvent{
        SubjectType: "character", SubjectID: "char_1",
        EventType: "character.location.changed",
    }
    violation := deadCharRule.Validate(ctx, &event, &StoryState{Characters: charState})
    assert.NotNil(t, violation)
    assert.Equal(t, "reject", violation.Severity)
}

func TestProjectionRebuild(t *testing.T) {
    // Insert events
    // Rebuild projection
    // Assert correct current state
}
```

### 10. API Endpoints

Existing `GET /stories/{id}/narrative-events` works. Add:

`GET /stories/{id}/characters/{charId}/view` — returns current `CharacterView`
`GET /stories/{id}/projections/status` — shows projection version vs latest event version

### 11. Event Replay for Debugging

Add CLI command or API endpoint:

`POST /stories/{id}/events/replay` — truncates projections, rebuilds from `narrative_events`

```go
func (s *StoryService) ReplayEvents(ctx context.Context, storyID string) error {
    events, _ := s.eventRepo.ListByStory(ctx, storyID, 0)
    _ = s.charViewRepo.DeleteByStory(ctx, storyID)
    for _, charID := range collectCharacterIDs(events) {
        _, _ = s.charProjection.rebuild(ctx, storyID, charID)
    }
    return nil
}
```

### 12. File Changes Summary

| File | Change |
|------|--------|
| `internal/event/` (new) | `extractor.go`, `validator.go`, `rules/` |
| `internal/projection/` (new) | `character_view.go`, `timeline_view.go` |
| `internal/domain/character_view.go` (new) | `CharacterView`, `CharacterStateSnapshot` |
| `internal/repository/interfaces.go` | Add `NarrativeEventRepository` methods, `CharacterViewRepository` |
| `internal/repository/mongo/narrative_events.go` | Add `AppendMany`, `ListBySubject`, `LatestVersion` |
| `internal/repository/mongo/` (new) | `character_views.go` |
| `internal/service/generation_job_worker.go` | Wire EventExtractor + Validator after StepExtract |
| `internal/repository/mongo/client.go` | Add indexes for `character_views`, narrative_events subject lookup |
