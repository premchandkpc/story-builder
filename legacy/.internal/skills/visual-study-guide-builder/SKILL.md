# Visual Study Guide Builder

Generate architecture docs, Mermaid diagrams, and visual study pages for story-builder.

## Trigger
- "generate architecture diagram"
- "create visual doc for [feature]"
- "mermaid diagram for [component]"
- "study guide for [area]"

## Output types

### Architecture overview
System context diagram showing all packages, external deps, data flow.

### Scene orchestration
Sequence diagrams for:
- Pipeline generation (current 6-worker flow)
- Agent turn orchestration (Director → Character → Narrator → Editor → CanonGuard)
- Scene finish phase (StateExtract → Critic)

### Story graph model
Domain model diagram showing relationships between:
Story → Scene → SceneEdge → Character → CharacterState → CharacterMemory

### Canon/timeline model
CanonDelta → Story.CanonPins ← TimelineEvent flow

### Request flow
HTTP → API handler → Service → Repository → Mongo/Redis for each major endpoint.

### DB schema visual
Collection list with key fields and indexes.

### Agent interaction map
All 10 agents with their inputs, outputs, and which other agents they feed.

## Format
- Always Mermaid.js for diagrams (renders in GitHub)
- One `.md` file per visual doc in `docs/visual/`
- Include narrative walkthrough alongside each diagram

## Output location
`docs/visual/{name}.md`
