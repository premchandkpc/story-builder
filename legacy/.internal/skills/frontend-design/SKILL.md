# Frontend Design

Guidance for story-builder UI: graph editor, scene turn UX, narrative dashboards.

## Trigger
- "design [component] UI"
- "improve graph editor"
- "scene turn interface"
- "narrative dashboard"

## Core UI surfaces

### Story Graph (React Flow)
- Nodes are scenes, edges are SceneEdge types
- Color by status: draft/generated/accepted/stale
- Edge labels: seq/fork/join/choice
- Context panel when selecting a node (edit scene, generate, view turns)
- Zoom to fit, minimap, auto-layout

### Scene Turn UI
- Timeline of turns for a scene (vertical list)
- Each turn shows: role badge, character name (if character turn), content, duration, status
- Turn-by-turn playback (step through scene generation)
- Accept/reject per turn output
- Branch view if alternate turns exist

### Scene Generation Panel
- Show pipeline progress (generate → extract → memory → timeline → summary → validate)
- Pipeline step status: pending/running/done/failed
- View generated prose, accept/reject generation
- Compare multiple generations side by side

### Canon Editor
- Table of current canon facts (category, fact, value, confidence)
- Filter by category, search
- Add/remove canon pins
- Show delta history per fact

### Narrative Dashboard
- Story progress: acts completed, scenes, word count
- Character arc tracking (per character growth stage)
- Plot thread status (open/advancing/resolved/abandoned)
- Timeline visualization

## Design principles
- Dark theme (matches existing React Flow setup)
- Minimal, focused tooling — not a generic text editor
- Keyboard shortcuts for graph editing
- Progressive disclosure: basic view first, advanced panels expandable
