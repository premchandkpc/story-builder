# Phase 4: Observability + Run Inspector

## Problem

Current observability is OpenTelemetry spans + basic `RunInspector` component. No way to answer "why did this scene come out badly" without reading raw Mongo. No prompt/context inspection in the UI. No cost attribution per step.

## Target

A Run Inspector that answers:
- What went into this generation? (prompt sections, tokens, context hash)
- What came out? (per-agent output, events, validation)
- How much did it cost? (tokens per step, model used)
- What was rejected? (validation violations, rejected narrative events)

## UI Architecture

```
RunInspector.tsx (tab in GraphPanel)
  ├── RunList — filterable list of StoryRuns
  │   ├── Status badge (queued/running/completed/failed/cancelled)
  │   ├── Step timeline (horizontal Gantt bars)
  │   └── Cancel button (for running/queued)
  │
  └── RunDetail — expanded view of selected run
      ├── Overview — run type, status, duration, context hash, model
      ├── PromptSections — collapsible sections with token counts
      │   ├── System prompt (collapsed, expand to full text)
      │   ├── Character cards (collapsed)
      │   ├── Memories (collapsed)
      │   ├── Timeline context (collapsed)
      │   └── User prompt (collapsed)
      ├── StepTimeline — visual timeline of steps
      │   ├── Bar per step (color-coded by status)
      │   ├── Duration, tokens, model per step
      │   └── Expand for raw output + error details
      ├── EventsTab — narrative events produced by this run
      │   ├── Accepted events
      │   └── Rejected events (with rule name + reason)
      └── CostSummary — total tokens, estimated cost per model
```

## Backend API

### Enhance `GET /runs/{id}` — return full artifacts

```json
{
  "id": "run_123",
  "story_id": "story_1",
  "scene_id": "scene_42",
  "run_type": "generate_scene",
  "status": "completed",
  "created_by": "user_1",
  "started_at": "2026-06-23T10:00:00Z",
  "finished_at": "2026-06-23T10:02:30Z",
  "input_context_hash": "sha256...",
  "current_step": "validate",
  "error_summary": "",
  "output_gen_id": "gen_42",
  "prompt_snapshot": {
    "system": "You are a narrative director...",
    "token_count": 4500,
    "sections": [
      {"name": "Character Cards", "tokens": 1200, "content_snippet": "..."},
      {"name": "Memories", "tokens": 800, "content_snippet": "..."},
      {"name": "Timeline", "tokens": 300, "content_snippet": "..."}
    ]
  },
  "steps": [
    {
      "step_name": "generate",
      "status": "done",
      "started_at": "2026-06-23T10:00:00Z",
      "finished_at": "2026-06-23T10:01:30Z",
      "model": "claude-sonnet-4-20250514",
      "prompt_hash": "sha256...",
      "tokens_in": 4500,
      "tokens_out": 820,
      "estimated_cost_usd": 0.042,
      "output_snippet": "The castle gates groaned open...",
      "error": ""
    }
  ],
  "events": {
    "accepted": 5,
    "rejected": 1,
    "rejected_details": [
      {
        "event_type": "character.location.changed",
        "subject_id": "char_1",
        "reason": "dead_character_cannot_act: char_1 health=0",
        "payload": {"location": "throne_room"}
      }
    ]
  },
  "cost_summary": {
    "total_tokens": 7150,
    "estimated_cost": 0.065,
    "by_model": {
      "claude-sonnet": {"tokens": 5320, "cost": 0.042},
      "claude-haiku": {"tokens": 1830, "cost": 0.003}
    }
  }
}
```

### New endpoints

| Endpoint | Purpose |
|----------|---------|
| `GET /runs/{id}/prompt-sections` | Full prompt sections for a run (paged) |
| `GET /runs/{id}/events` | Narrative events for the run |
| `GET /runs/{id}/cost` | Cost breakdown |
| `GET /stories/{id}/runs/stats` | Aggregated stats: avg duration, failure rate, cost |

### `RunStep` enhancements

Add to `internal/domain/run_step.go`:

```go
type RunStep struct {
    // ... existing ...
    OutputSnippet     string         `bson:"outputSnippet,omitempty"` // first 500 chars
    PromptSnippet     string         `bson:"promptSnippet,omitempty"` // first 500 chars
    EstimatedCostUSD  float64        `bson:"estimatedCostUsd,omitempty"`
    Artifacts         map[string]any `bson:"artifacts,omitempty"`
    // Store rejected events, validation results, etc.
}
```

## Frontend Components

### `RunTimeline.tsx` (new)

Horizontal step timeline:

```
[generate ██████████████████████ 90s] [extract ██████ 15s] [memory ██ 5s]
[tl █ 2s] [summary ███ 8s] [validate █ 3s]
```

Props: `steps: RunStep[]`
Behavior:
- Bar width proportional to duration
- Color: green=done, red=failed, yellow=running, gray=pending
- Click bar → expand step details
- Tooltip: step name, duration, tokens, model

### `PromptSectionViewer.tsx` (new)

Collapsible tree of prompt sections:

```
▶ System Prompt (2.3k tokens)
▶ Character Cards (1.2k tokens)
  ▶ Hero (400 tokens) ...
  ▶ Villain (350 tokens) ...
▶ Memories (800 tokens)
▶ Timeline Context (300 tokens)
▶ User Prompt (1.1k tokens)
```

Props: `sections: PromptSection[]`
Behavior: expand/collapse per section, token count badge, copy-to-clipboard

### `EventList.tsx` (new)

Two-column event viewer:

```
[Accepted Events]           [Rejected Events (1)]
✓ location.changed: char_1  ✗ location.changed: char_1
✓ knowledge.added: char_1     Reason: dead_character_cannot_act
✓ emotion.changed: char_2
```

Props: `events: {accepted: NarrativeEvent[], rejected: RejectedEvent[]}`
Behavior: click event → show full payload in side panel

### `CostCard.tsx` (new)

Cost summary card for a run:

```
Cost Summary
━━━━━━━━━━━━━━━━━━━━━
Total tokens: 7,150
Estimated cost: $0.065
By model:
  claude-sonnet  5,320 tokens  $0.042
  claude-haiku   1,830 tokens  $0.003
```

## Run Inspector — Wireframe

```
┌─────────────────────────────────────────────────────────────┐
│ Runs [All ▾] [Running ▾] Status: ● completed ▼ Filter...   │
├─────────────────────────────────────────────────────────────┤
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ #3812  generate_scene        ● completed                │ │
│ │ [████████████████] [████] [██] [█] [███] [█]           │ │
│ │ Step: generate | 2m30s | claude-sonnet | 5320 tokens   │ │
│ ├─────────────────────────────────────────────────────────┤ │
│ │ Cancel                                                    │ │
│ └─────────────────────────────────────────────────────────┘ │
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ #3811  generate_scene        ● failed                   │ │
│ │ [████████████] ✗                                         │ │
│ │ Step: generate failed: context length exceeded           │ │
│ ├─────────────────────────────────────────────────────────┤ │
│ │ [Show Details ▸] [Retry ▸]                                │ │
│ └─────────────────────────────────────────────────────────┘ │
├─────────────────────────────────────────────────────────────┤
│ Run #3812 Detail                     [▾ Hide]              │
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ Overview │ Prompt │ Timeline │ Events │ Cost            │ │
│ ├─────────────────────────────────────────────────────────┤ │
│ │ [Prompt Section Viewer]                                  │ │
│ │ ▶ System (2.3k tokens)                                   │ │
│ │ ▶ Characters (1.2k) [expand]                             │ │
│ │ │  Hero: Arya Stormborn...                               │ │
│ │ │  Villain: The Iron King...                             │ │
│ │ ▶ Memories (0.8k)                                        │ │
│ │ ▶ Timeline (0.3k)                                        │ │
│ │ ▶ User (1.1k)                                             │ │
│ └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

## File Changes Summary

| File | Change |
|------|--------|
| `internal/domain/run_step.go` | Add `OutputSnippet`, `PromptSnippet`, `EstimatedCostUSD` |
| `internal/api/runs.go` | Add `GET /runs/{id}/prompt-sections`, `events`, `cost` |
| `internal/service/generation.go` | Capture prompt snippet + cost in RunStep |
| `web/src/components/RunInspector.tsx` | Full rewrite with tabs, timeline, events |
| `web/src/components/RunTimeline.tsx` (new) | Step timeline component |
| `web/src/components/PromptSectionViewer.tsx` (new) | Prompt section inspector |
| `web/src/components/EventList.tsx` (new) | Accepted/rejected event viewer |
| `web/src/components/CostCard.tsx` (new) | Cost summary card |
| `web/src/api/hooks.ts` | Add hooks for prompt sections, events, cost |
| `web/src/api/client.ts` | Add prompt/events/cost endpoints |
| `web/src/api/types.ts` | Add `PromptSection`, `CostSummary`, `RejectedEvent` types |
