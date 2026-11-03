# Frontend

React 19 + TypeScript + Vite 8 story graph editor. DAG visualization via React Flow (xyflow). Data fetching via TanStack React Query. Warm dark "writer's study" theme with editorial/leather-bound aesthetic.

---

## Tech Stack

| Library | Version | Purpose |
|---|---|---|
| `react` + `react-dom` | 19 | UI framework |
| `@xyflow/react` | 12 | DAG graph canvas |
| `@tanstack/react-query` | 5 | Server state cache + mutations |
| `react-router-dom` | 7 | Client-side routing |
| `vite` | 8 | Dev server + build |

---

## Project Structure

```
web/
  index.html              HTML entry (Vite injects main.tsx)
  vite.config.ts          Proxy /api/* → Go backend :8080
  tsconfig.json / .app / .node
  eslint.config.js
  Dockerfile
  src/
    main.tsx              React mount point
    routes.tsx            Route tree (createBrowserRouter)
    index.css             Global styles (warm dark theme, animations, utility classes)
    api/
      client.ts           HTTP client — fetch() wrapper with timeout
      hooks.ts            React Query hooks for every query/mutation
      types.ts            All TypeScript interfaces, shared style objects, SceneNodeData type
    components/
      Layout.tsx          App shell: TopBar + sidebar + <Outlet/>
      TopBar.tsx          Search bar + nav (editorial masthead feel)
      HomeView.tsx        Landing page: create/generate stories
      StoryView.tsx       Story wrapper (reads :storyId param)
      StoryGraph.tsx      React Flow canvas + right sidebar (orchestrator)
      GraphPanel.tsx      300px right sidebar with tab routing (Edit/Info/Gen/Turns/Agents)
      SceneEditorPanel.tsx Edit form (beat intent, POV, tone, target words)
      NodeInfoPanel.tsx   Node info tab (status, beat intent, edge counts)
      EdgeInfoPanel.tsx   Edge detail tab (type, source/target)
      GenerationList.tsx  Generations list with auto-polling, progress indicator, error states
      GenerationCompare.tsx Side-by-side generation diff
      SceneNode.tsx       Custom React Flow node (vintage index card with paper texture, status strip, char chips, word count bar)
      TimelineView.tsx    Cross-story timeline display with color-coded events
      BiblePanel.tsx      World bible display + generate + sharing UI
      TurnItem.tsx        Individual agent turn with expandable I/O
      TurnTimeline.tsx    Wrapper mapping turns → TurnItem
      AgentRunItem.tsx    Individual agent run with expandable I/O
      AgentRunPanel.tsx   Wrapper mapping runs → AgentRunItem
      StatCard.tsx        Shared metric card (dashboards)
      LlmMetricsDashboard.tsx Token/cost metrics
      CriticScoreDashboard.tsx Critic evaluation scores
      AuditDashboard.tsx  Code audit findings page
      CompressionStats.tsx Token compression display
      Toast.tsx           Toast notification system
      StoryListItem.tsx   Sidebar story entry (delete confirmation, optimistic placeholder)
```

---

## Route Tree

```
"/"                       → Layout + HomeView
"/stories/:storyId"       → Layout + StoryView → StoryGraph
"/audit"                  → AuditDashboard
"/metrics"                → LlmMetricsDashboard + CriticScoreDashboard
```

`Layout.tsx` renders the sidebar + `TopBar`. Child routes render inside `<Outlet/>`.

---

## Component Tree

```
RouterProvider
  └── Layout
       ├── TopBar
       │    ├── App title (navigates to /)
       │    ├── Search input (filters sidebar)
       │    └── Home button (visible on story pages)
       ├── Sidebar
       │    ├── "Create story" input + button (optimistic placeholder on create)
       │    ├── StoryListItem[] (filtered by search, 2-click delete with confirmation)
       │    └── Empty states: loading/skeleton, search-no-results, true-empty, error+retry
       └── <Outlet/>
            ├── HomeView   (at "/")
            └── StoryView  (at "/stories/:id")
                 └── StoryGraph
                      ├── ReactFlow canvas
                      │    ├── SceneNode[] (custom node type "scene", paper texture)
                      │    ├── Graph load error overlay with retry
                      │    └── Edge[] (seq/fork/join/choice styles)
                      └── GraphPanel (300px right sidebar, tabbed)
                           ├── Default view (no node selected):
                           │    ├── TimelineView — local + cross-story events
                           │    ├── BiblePanel — world bible + sharing
                           │    ├── LlmMetricsDashboard — token metrics
                           │    └── CriticScoreDashboard — critic scores
                           ├── "Add Scene" button
                           ├── Tab: Edit → SceneEditorPanel
                           │    ├── Beat Intent (text)
                           │    ├── POV (dropdown)
                           │    ├── Tone (dropdown)
                           │    ├── Target Words (number)
                           │    └── Save / Generate buttons (with 5-min timeout)
                           ├── Tab: Info → NodeInfoPanel
                           │    ├── Status card + beat intent
                           │    └── Edge counts (monospace)
                           ├── Tab: Gen → GenerationList (auto-polling, progress bar, error)
                           │    └── Generation cards → GenerationCompare
                           ├── Tab: Turns → TurnTimeline
                           │    └── TurnItem[] (expandable I/O)
                           ├── Tab: Agents → AgentRunPanel
                           │    └── AgentRunItem[] (expandable I/O)
                           └── Tab: Critic → CriticScoreDashboard (per-scene)
```

---

## Data Flow

```
Component
  ↓ useQuery / useMutation
api/hooks.ts  (TanStack React Query — cache, staleTime, retry, optimistic updates)
  ↓
api/client.ts  (request<T>() — generic fetch wrapper)
  ↓  HTTP GET/POST/PUT/DELETE  /api/v1/*
Go API Server (chi — proxied by Vite in dev)
  ↓
MongoDB + Redis
```

### Cache defaults

- `staleTime: 5_000` — data fresh for 5 seconds
- `retry: 1` — one retry on failure

### Optimistic Update Pattern

All mutations that modify visible state use the TanStack React Query optimistic update pattern:

```typescript
onMutate: async (variables) => {
  await queryClient.cancelQueries({ queryKey })
  const prev = queryClient.getQueryData(queryKey)
  queryClient.setQueryData(queryKey, (old) => optimisticUpdate)
  return { prev }
}
onError: (err, vars, context) => {
  queryClient.setQueryData(queryKey, context.prev)  // rollback
  showError("Human-readable message")
}
onSettled: () => {
  queryClient.invalidateQueries({ queryKey })
}
```

Applied to: create story (placeholder), delete story (remove from list immediately).

---

## API Client (`api/client.ts`)

Single `request<T>(path, init)` generic function:

```typescript
async function request<T>(path: string, init?: RequestInit & { timeout?: number }): Promise<T>
```

Features:
- JSON Content-Type header
- Configurable timeout via AbortController (default 30s)
- Non-2xx → throws `Error` with status + body
- 204 No Content → `undefined`
- All responses typed via generic parameter

Exported `api` object groups endpoints:

```typescript
api.stories.list()           // GET  /api/v1/stories
api.stories.get(id)          // GET  /api/v1/stories/:id
api.stories.create(data)     // POST /api/v1/stories
api.stories.delete(id)       // DELETE
api.stories.generate(data)   // POST /api/v1/stories/generate
api.stories.generateTitle()  // POST /api/v1/stories/generate-title

api.nodes.list(storyId)      // GET  /api/v1/stories/:id/nodes
api.nodes.create(storyId, d) // POST
api.nodes.update(storyId, id, data) // PUT
api.nodes.delete(storyId, id) // DELETE
api.nodes.updatePosition()   // PUT (partial)

api.edges.create(storyId, d) // POST
api.edges.deleteById(storyId, id) // DELETE (preferred)

api.topology.get(storyId)    // GET  /api/v1/stories/:id/topology

api.generations.list()       // GET
api.generations.get()        // GET  single generation by ID
api.generations.generate()   // POST (async, returns immediately)
api.generations.accept()     // POST

api.generations.getStatus()  // GET  /api/v1/generations/:genID/status (status, error, tokens)
api.generations.getProgress()// GET  SSE stream

api.turns.list()             // GET  /experimental/...
api.agentRuns.list()         // GET  /experimental/agent-runs
api.metrics.llm()            // GET  /stories/:id/metrics/llm
api.critic.list()            // GET  /stories/:id/critic-scores
api.timeline.list()          // GET  /stories/:id/timeline
api.timeline.crossStoryList()// GET  /stories/:id/timeline/cross-story
api.bible.get()              // GET  /stories/:id/bible
api.bible.generate()         // POST /stories/:id/bible/generate
api.bible.link/unlink()      // POST cross-story bible sharing
```

---

## React Query Hooks (`api/hooks.ts`)

### Queries

| Hook | Returns | Cache Key | Notes |
|---|---|---|---|
| `useStories()` | `Story[]` | `["stories"]` | |
| `useStoryNodeStats(storyId)` | `StoryStats` | `["storyStats", storyId]` | Computed client-side |
| `useAllStoryStats(stories)` | `Record<string, StoryStats>` | `["allStoryStats", sortedIds]` | Batches with concurrency 6 |
| `useGenerationStatusPolling(storyId, nodeId, enabled)` | `{ generations, isLoading, isError, refetch, hasPending }` | `["generations", storyId, nodeId]` | Auto-polls every 2s while status=pending/running/queued |
| `useTimeline(storyId)` | `TimelineEvent[]` | `["timeline", storyId]` | |
| `useCrossStoryTimeline(storyId)` | `TimelineEvent[]` | `["crossStoryTimeline", storyId]` | |
| `useBible(storyId)` | `StoryBible | null` | `["bible", storyId]` | |
| `useReferencingBibles(storyId)` | `StoryBible[]` | `["referencingBibles", storyId]` | |

### Mutations

| Hook | Side Effect |
|---|---|
| `useCreateStory()` | Optimistic placeholder → real story, invalidates `["stories"]`, navigates to `/stories/:id` |
| `useDeleteStory()` | Optimistic removal (rollback on error), navigates home if viewing deleted story |
| `useGenerateTitle()` | None (returns `{ title }`) |
| `useGenerateStory()` | Invalidates `["stories"]`, navigates to new story |
| `useGenerateBible(storyId)` | Invalidates `["bible", storyId]` |
| `useUpdateBible(storyId)` | Invalidates `["bible", storyId]` |
| `useLinkBible(storyId)` / `useUnlinkBible(storyId)` | Invalidates bible + referencing bibles |

### Patterns

- Optimistic updates use `onMutate` → snapshot prev state → `setQueryData` → `return { prev }` → `onError` rollback → `onSettled` invalidate
- `setToastFns()` bridge lets mutation hooks access Toast without being in component tree
- `useGenerationStatusPolling` uses `useEffect` interval + `queryClient.invalidateQueries` every 2s while status is pending/running

---

## Types (`api/types.ts`)

### Domain Types (mirror backend)

| Interface | Purpose |
|---|---|
| `Story` | DAG root |
| `GraphNode` / `GraphEdge` | DAG elements for React Flow |
| `Scene` | Legacy chapter-based model |
| `Generation` | LLM output record (includes status, error, step_status, prompt/completion/total tokens, duration_ms) |
| `Topology` | Full DAG snapshot |
| `Character` | Character definition |
| `Location` | Story settings |
| `SceneStructure` | Interactive generation flow config |
| `StorySummary` | Hierarchical summary |
| `StoryBible` | World bible with dimensions, factions, cultures, sharing |
| `TimelineEvent` | Story timeline events with cross-story support |
| `CriticScoreData` | Agent quality evaluation |

### Union Types

```typescript
type NodeStatus = "draft" | "generated" | "accepted" | "stale"
type EdgeType   = "seq" | "fork" | "join" | "choice"
type FlowType   = "monologue" | "dialogue" | "round_robin" | "parallel" | "custom"
```

### UI Helpers

| Export | Type | Usage |
|---|---|---|
| `inputStyle` | `React.CSSProperties` | Shared input style (spread into `style={}`) |
| `btnStyle(bg, disabled?)` | Function → `React.CSSProperties` | Button styling |
| `skeletonStyle(w, h)` | Function → `React.CSSProperties` | Shimmer skeleton |
| `spinnerStyle` | `React.CSSProperties` | Inline spinner |
| `labelStyle` | `React.CSSProperties` | Uppercase muted form labels |
| `cardStyle` | `React.CSSProperties` | Raised card surface |
| `badgeStyle` | `React.CSSProperties` | Tag/pill badge |
| `ghostBtnStyle` | `React.CSSProperties` | Subtle borderless button |
| `destructiveBtnStyle` | `React.CSSProperties` | Red destructive button |
| `SceneNodeData` | Interface | React Flow node data (label, title, status, beatIntent, pov, tone, targetWords, characterRefs, wordCount) |

---

## Components

### Layout.tsx (app shell)

- `useParams` reads `:storyId` to highlight active story
- `useMemo` filters stories by search query (case-insensitive)
- Three empty states: loading skeleton (stagger), no results (search + clear), true empty (descriptive CTA)
- Error state: inline error message + retry button for `useStories`
- `useDeleteStory` mutation wired to each StoryListItem
- `useCreateStory` with optimistic placeholder (pulsing dot, "Creating..." label, reduced opacity)
- `setToastFns` bridge connects Layout's toast to hooks.ts mutations
- Search empty shows "No stories match \"{query}\"" with "Clear search" link

### TopBar.tsx

- Editorial masthead feel with decorative divider
- Controlled search input via `searchQuery` + `onSearchChange` props
- Search icon absolutely positioned inside input
- Home button conditionally rendered via `hasActiveStory` prop

### HomeView.tsx

- Two `useState` fields: `newTitle`, `synopsis`
- Error/success banner (dismissible)
- Three mutation hooks: `useCreateStory`, `useGenerateTitle`, `useGenerateStory`
- `mutateAsync` for title generation (awaits result → fills title field)
- Auto-title fallback: first 50 chars of synopsis
- Larger hero (38px), thin gradient rule, tighter card (28px padding)

### StoryGraph.tsx (orchestrator)

**State:**
- `nodes`, `setNodes`, `onNodesChange` — React Flow nodes via `useNodesState`
- `edges`, `setEdges`, `onEdgesChange` — React Flow edges via `useEdgesState`
- `selectedNode` — currently clicked node for side panel
- `form` — edit form fields synced with selected node
- `graphError` — network error state with contextual message + retry overlay

**Generations via `useGenerationStatusPolling` hook:**
- Auto-polls every 2s while any generation has pending/running/queued status
- Exposes `generations`, `gensLoading`, `gensPending`, `gensError`, `refetchGens`
- No more manual `loadGenerations` state/ref — hook manages everything

**Error handling:**
- Graph load error: full-screen overlay with contextual message (timeout / 500 / 404 / generic) + Retry button
- Node position drag failure: captures pre-drag position in `nodePositionsRef`, rolls back node position + toast "Node snapped back"
- Edge creation conflict: parses 409/duplicate → "Connection already exists", 400 → "Invalid connection"
- Generation timeout: 5-min `setTimeout` warning, handles abort/timeout/429 with specific messages
- Node delete: parses 409+connected → "Remove all edges first", 404 → "Already deleted"

### GraphPanel.tsx

300px right sidebar with tabbed interface. Receives `activeTab`, `setActiveTab`, all canvas callbacks.

**Tabs:** Edit, Info, Gen, Turns, Agents, Critic.

Default view (no node selected): TimelineView → BiblePanel → LlmMetricsDashboard → CriticScoreDashboard.

### GenerationList.tsx

Generations list for selected node. Appears in Gen tab. Now with auto-polling:

- **Pending animation**: pulsing amber border + shimmer progress bar while any generation status is "pending"/"running"
- **Spinner** shows in title bar during loading
- **"● Generating"** live indicator next to title while pending
- **Skeleton loading**: shimmer cards while initial load
- **Error state**: inline error message + retry button when query fails
- **Failed generations**: red border + error message in monospace
- **Token stats**: total tokens + duration displayed per generation
- **Accept button**: shows "Accepting..." with disabled state during mutation
- **Empty state**: sparkle icon + "Configure the scene and click Generate" hint
- Status badge colors: success=green, failed=red, pending/running=amber

### SceneNode.tsx

Custom React Flow node — premium index card aesthetic:

- **Top tab**: vintage metal ring (gold gradient, 3D shadow)
- **Status color strip**: 3px gradient top border per status (draft/generated/accepted/stale)
- **Paper texture**: warm gradient background (`#f5f0e8 → #efe8dc → #f0eadc`)
- **Compact status**: dot + label text inline (no pill background)
- **Character chips**: circular avatar initials (20px, gradient background, shadow)
- **Word count progress bar**: animated width, color by progress (success/accent/warn/border), shows "wordCount/targetWords"
- **Beat intent**: 2-line clamp with ellipsis, italic when draft
- **Metadata row**: POV ◆ Tone (monospace, diamond separator)
- **Handle hover**: scale(1.3) + glow on source/target handles
- **Card hover**: translateY(-3px) + enhanced shadow
- Wrapped in `memo()` for render performance

### StoryListItem.tsx

Sidebar entry with warm dark styling:

- Status dot: red=stale, green=all accepted, amber=mixed, gray=empty
- **2-click delete confirmation**: first click → button turns red + "Confirm Delete?" — 3s timeout auto-reverts
- **Optimistic placeholder**: pulsing dot, reduced opacity (0.8), "Creating..." label, expandIn entrance
- Title bold if active, heading font if active
- Compact stats: "3ch · 12sc · 8✓ · 2○"
- Delete button visible on hover (uses `role="button"` div pattern for keyboard accessibility)

### TimelineView.tsx

Cross-story timeline display in default panel view:

- Local + cross-story events merged, sorted by order + created_at
- Color-coded event type dots (scene, choice, branch, converge, climax)
- Shared events labelled with "shared" tag + "N cross-story" count
- Event title, description, order number, type badge

### BiblePanel.tsx

World bible display + generate + sharing UI:

- Shows bible content when exists (world, theme, rules, factions, dimensions)
- Generate button when missing
- Sharing UI: link/unlink with target story ID input
- Referencing stories list

---

## Error Handling Layers

### Network/API Errors
All `api/*` calls throw `Error("HTTP {status}: {body}")`. Components parse status codes for contextual messages:
- 400 → "Invalid input"
- 404 → "Not found. It may have been deleted."
- 409 → "Conflict — refresh and try again"
- 429 → "Rate limited. Wait a moment and retry."
- 5xx → "Server error. We've logged it."
- Timeout → "Request timed out. Check connection."

### Query Error States
Every `useQuery` handles: `isLoading` → skeleton, `isError` → inline error + retry, empty → CTA.

### React Error Boundary
`ErrorBoundary.tsx` wraps the entire app. Shows warm-dark styled fallback with "Try again" button.

### Form Validation
Client-side validation before API call: title required, beat_intent warned if empty, target_words clamped 50-5000. Disabled submit while invalid or pending.

---

## Loading State Patterns

- **Skeleton** (preferred over spinner for content): shimmer via `skeletonStyle()`
- **Spinner** for button actions: 16px rotating border
- **Button loading**: spinner + label + disabled pointer events
- **Generation polling**: `useGenerationStatusPolling` interval every 2s while pending
- **Progress indicator**: shimmer bar on pending generation cards

---

## Empty States (every list needs one)

- StoryList empty → icon + heading + descriptive copy
- StoryList search → "No stories match \"{query}\"" + "Clear search"
- Canvas empty → icon + "No scenes yet" + "Add First Scene" CTA
- GenerationList → sparkle icon + "No generations yet" + generate hint
- TurnTimeline → "No turns recorded. Use agent mode to generate."
- AgentRunPanel → "No agent runs. Generate with scene structure set."

---

## React Flow Edge Styles

| Edge Type | Color | Width | Style |
|---|---|---|---|
| `seq` | `#8888a0` (gray) | 1.5 | solid |
| `fork` | `#c9734a` (amber) | 2.5 | solid |
| `join` | `#d4a853` (gold) | 2.5 | solid |
| `choice` | `#8888a0` (gray) | 2.5 | dashed |

Non-seq edges show their type as a label on the edge.

---

## Design Tokens (`index.css`)

### Colors

| Token | Hex | Usage |
|---|---|---|
| `--bg` | `#15110e` | Page background |
| `--bg-warm` | `#1a1512` | Warm accent surfaces |
| `--surface` | `#1e1916` | Cards, side panel |
| `--surface-hover` | `#28221e` | Hover states |
| `--border` | `#3d322a` | Dividers, input borders |
| `--text` | `#e8ddd0` | Body text |
| `--text-muted` | `#8c7e70` | Secondary labels |
| `--text-dim` | `#6b5f53` | Muted metadata |
| `--text-faint` | `#4d4238` | Placeholders |
| `--accent` | `#d4a853` | Primary CTA, amber gold |
| `--success` | `#6b8f5e` | Accepted/done |
| `--error` | `#b05c50` | Failed/danger |
| `--warn` | `#c9734a` | Pending/warning |
| `--info` | `#6b9fc4` | Informational |

### Shadow Tokens

| Token | Value |
|---|---|
| `--shadow-xs` | `0 1px 2px rgba(0,0,0,0.25)` |
| `--shadow-sm` | `0 1px 3px rgba(0,0,0,0.3)` |
| `--shadow-md` | `0 3px 8px rgba(0,0,0,0.3)` |
| `--shadow-lg` | `0 6px 20px rgba(0,0,0,0.35)` |
| `--shadow-inner` | `inset 0 1px 2px rgba(0,0,0,0.15)` |

### Transitions

| Token | Value |
|---|---|
| `--transition-fast` | `0.12s cubic-bezier(0.16,1,0.3,1)` |
| `--transition-base` | `0.2s cubic-bezier(0.16,1,0.3,1)` |
| `--transition-spring` | `0.35s cubic-bezier(0.34,1.56,0.64,1)` |

### Animations

| Name | Purpose |
|---|---|
| `fadeIn` / `fadeOut` | Opacity transitions |
| `slideUp` / `slideDown` | Vertical reveal |
| `slideInLeft` / `slideInRight` | Horizontal entrance |
| `scaleIn` / `scaleOut` | Scale-based entrance |
| `pulse` | Slow opacity pulse (idle indicators) |
| `shimmer` | Loading skeleton sweep |
| `spin` | Rotating loader |
| `glowPulse` | Color glow on status indicators |
| `expandIn` | Scale+fade for expandable sections |
| `breathe` | Subtle scale oscillation |

### CSS Utility Classes

| Class | Effect |
|---|---|
| `.stagger-fade-in` | Opacity 0→1 with `animation-delay` per child |
| `.stagger-slide-up` | Slide up + fade with cascading delay |
| `.card-hover` | `translateY(-1px)`, `shadow-md`, border lighten |
| `.btn-press` | Active: `scale(0.97)` on mousedown |

---

## Conventions

1. **All data fetching through `api/hooks.ts`** — components never call `fetch()` directly
2. **One custom hook per domain query** — encapsulates cache key + side effects
3. **Props typed with interfaces** — always explicit, never `any`
4. **`useCallback` for handlers** — prevents child re-renders when passed as props
5. **`useMemo` for derived arrays** — filtered/sorted lists, computed strings
6. **`memo()` on graph nodes** — React Flow performance
7. **Inline styles** — no CSS modules, no Tailwind
8. **Warm dark theme** — `#15110e` base, `#e8ddd0` text, `#1e1916` surfaces
9. **CSS utility classes for interactions** — `.card-hover`, `.btn-press`, `.stagger-fade-in`
10. **Stagger entrance via CSS `animation-delay`** — no IntersectionObserver
11. **All border-radius via `--radius-*` tokens** — never raw px values
12. **All transitions via `--transition-*` tokens** — consistent timing
13. **Optimistic updates for all mutations** — placeholder/remove immediately, rollback on error
14. **Three-state queries** — skeleton for loading, inline error + retry for error, CTA for empty
15. **Error parsing by status code** — 400/404/409/429/5xx get specific messages
16. **Toast wiring via `setToastFns` bridge** — mutations outside component tree can still show toasts
