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
      GenerationList.tsx  Generations list with preview, compare, accept
      GenerationCompare.tsx Side-by-side generation diff
      SceneNode.tsx       Custom React Flow node (vintage index card + pin)
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
      StoryListItem.tsx   Sidebar story entry
```

---

## Route Tree

```
"/"                       → Layout + HomeView
"/stories/:storyId"       → Layout + StoryView → StoryGraph
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
       │    ├── "Create story" input + button
       │    └── StoryListItem[] (filtered by search)
       └── <Outlet/>
            ├── HomeView   (at "/")
            └── StoryView  (at "/stories/:id")
                 └── StoryGraph
                      ├── ReactFlow canvas
                      │    ├── SceneNode[] (custom node type "scene")
                      │    └── Edge[] (seq/fork/join/choice styles)
                      └── GraphPanel (300px right sidebar, tabbed)
                           ├── "Add Scene" button
                           ├── Tab: Edit → SceneEditorPanel
                           │    ├── Beat Intent (text)
                           │    ├── POV (dropdown)
                           │    ├── Tone (dropdown)
                           │    ├── Target Words (number)
                           │    └── Save / Generate buttons
                           ├── Tab: Info → NodeInfoPanel
                           │    ├── Status card + beat intent
                           │    └── Edge counts (monospace)
                           ├── Tab: Gen → GenerationList
                           │    └── Generation cards → GenerationCompare
                           ├── Tab: Turns → TurnTimeline
                           │    └── TurnItem[] (expandable I/O)
                           └── Tab: Agents → AgentRunPanel
                                └── AgentRunItem[] (expandable I/O)
          (standalone routes)
          "/audit"         → AuditDashboard
          "/metrics"       → LlmMetricsDashboard + CriticScoreDashboard
```

---

## Data Flow

```
Component
  ↓ useQuery / useMutation
api/hooks.ts  (TanStack React Query — cache, staleTime, retry)
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
api.stories.generate(data)   // POST /api/v1/stories/generate
api.stories.generateTitle()  // POST /api/v1/stories/generate-title

api.nodes.list(storyId)      // GET  /api/v1/stories/:id/nodes
api.edges.create(storyId, d) // POST /api/v1/stories/:id/edges
api.topology.get(storyId)    // GET  /api/v1/stories/:id/topology
api.generations.generate()   // POST /api/v1/stories/:id/nodes/:nid/generate
// ... chapters, scenes, characters, locations, lore, casting, summaries, etc.
```

---

## React Query Hooks (`api/hooks.ts`)

### Queries

| Hook | Returns | Cache Key |
|---|---|---|
| `useStories()` | `Story[]` | `["stories"]` |
| `useStoryNodeStats(storyId)` | `StoryStats` | `["storyStats", storyId]` |
| `useAllStoryStats(stories)` | `Record<string, StoryStats>` | `["allStoryStats", sortedIds]` |

### Mutations

| Hook | Side Effect |
|---|---|
| `useCreateStory()` | Invalidates `["stories"]`, navigates to `/stories/:id` |
| `useGenerateTitle()` | None (returns `{ title }`) |
| `useGenerateStory()` | Invalidates `["stories"]` after 3s delay |

### Patterns

- Mutations use `useMutation` + `onSuccess` for cache invalidation
- Mutations that navigate use `useNavigate` inside the hook
- `queryKey` arrays scope caches by parameter (storyId, chapterId, etc.)
- `Promise.all` + `Object.fromEntries` pattern in `useAllStoryStats`

---

## Types (`api/types.ts`)

### Domain Types (mirror backend)

| Interface | Purpose |
|---|---|
| `Story` | DAG root |
| `GraphNode` / `GraphEdge` | DAG elements for React Flow |
| `Scene` | Legacy chapter-based model |
| `Generation` | LLM output record |
| `Topology` | Full DAG snapshot |
| `Character` | Character definition |
| `Location` | Story settings |
| `SceneStructure` | Interactive generation flow config |
| `StorySummary` | Hierarchical summary |

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
| `btnStyle(bg, disabled?)` | Function → `React.CSSProperties` | Button styling with letter-spacing + disabled state |
| `spinnerStyle` | `React.CSSProperties` | Inline spinner animation |
| `labelStyle` | `React.CSSProperties` | Uppercase muted form labels |
| `cardStyle` | `React.CSSProperties` | Raised card surface with border |
| `badgeStyle` | `React.CSSProperties` | Tag/pill badge |
| `ghostBtnStyle` | `React.CSSProperties` | Subtle borderless button |
| `destructiveBtnStyle` | `React.CSSProperties` | Red destructive action button |
| `StoryStats` | `{ total, generated, accepted, stale }` | Sidebar progress display |
| `SceneNodeData` | Interface | React Flow node data (id, label, beatIntent, pov, tone, status, wordCount, targetWords) |

---

## Components

### Layout.tsx (app shell)

- `useParams` reads `:storyId` to highlight active story
- `useMemo` filters stories by search query (case-insensitive)
- `useState` for `searchQuery` and `newTitle`
- `useStories()` + `useAllStoryStats()` for sidebar data
- `<Outlet/>` renders child route
- Warm sidebar background (`--bg-warm`), refined empty state

### TopBar.tsx

- Editorial masthead feel with decorative divider
- Controlled search input via `searchQuery` + `onSearchChange` props
- `useNavigate` for programmatic navigation
- Search icon absolutely positioned inside input
- Home button conditionally rendered via `hasActiveStory` prop (smaller cleaner buttons)

### HomeView.tsx

- Two `useState` fields: `newTitle`, `synopsis`
- Error banner via `error` state (dismissible)
- Three mutation hooks: `useCreateStory`, `useGenerateTitle`, `useGenerateStory`
- `mutateAsync` for title generation (awaits result → fills title field)
- Auto-title fallback: first 50 chars of synopsis
- Larger hero (38px), thin gradient rule, tighter card padding (28px), `--text-faint` subtitle

### StoryView.tsx

- Extracts `storyId` from URL params via `useParams`
- Simple pass-through to `StoryGraph`

### StoryGraph.tsx

305 lines. Orchestrates React Flow canvas + `GraphPanel` sidebar.

**State:**
- `nodes`, `setNodes`, `onNodesChange` — React Flow nodes (via `useNodesState`)
- `edges`, `setEdges`, `onEdgesChange` — React Flow edges (via `useEdgesState`)
- `selectedNode` — currently clicked node for side panel
- `form` — edit form fields synced with selected node

**Key callbacks (all `useCallback`-memoized):**
- `fetchGraph` — calls `api.topology.get()` → converts to React Flow format
- `onConnect` — optimistic edge creation + API persistence
- `onNodeClick` — sets selected node + populates form
- `addNode` — creates scene via API → re-fetches graph
- `updateNode` — updates node → re-fetches → deselects
- `generate` — triggers LLM generation for selected node
- `activeTab` state + `setActiveTab` passed to `GraphPanel`

**Helpers:**
- `toReactFlowNodes()` — `GraphNode[]` → `Node<SceneNodeData>[]` (grid layout)
- `toReactFlowEdges()` — `GraphEdge[]` → `Edge[]` (color by type)

**Panel styling:** editorial/leather-bound — `--bg-warm` bg, decorative gradient rule, serif heading, uppercase letter-spaced tabs, italic notebook-style empty state.

### GraphPanel.tsx

300px right sidebar with tabbed interface. Receives `activeTab`, `setActiveTab`, `selectedNode`, and all canvas callbacks via props.

**Tabs:** Edit, Info, Gen, Turns, Agents.

- Tab bar uses uppercase letter-spaced labels
- Each tab renders its respective panel component
- "Add Scene" button at top (with `btn-press` micro-interaction)
- Stats footer (node count, edge count, generated count)
- Wraps panel content in scrollable container

### SceneEditorPanel.tsx

Edit form for selected scene node. Appears in the Edit tab.

- Beat Intent (textarea), POV (select), Tone (select), Target Words (number input)
- Inputs use `--shadow-inner` for inset depth effect
- Save button + Generate button with letter-spacing + hover glow shadows
- Cancel button uses surface color
- Consistent `--radius-*` tokens
- `btn-press` micro-interaction on all buttons

### NodeInfoPanel.tsx

Read-only info display for selected node. Appears in the Info tab.

- Data grouped into card sections (`--surface` + `--border`)
- Uppercase `--text-faint` labels for each field
- Monospace edge count display
- Status badge pill
- `card-hover` class on each section card

### EdgeInfoPanel.tsx

Read-only edge detail display. Appears in the Info tab when an edge is selected.

- Card pattern matching NodeInfoPanel
- Source/target node IDs in monospace, side-by-side
- Uppercase type badge with letter-spacing
- `card-hover` class on card

### GenerationList.tsx

Generations list for selected node. Appears in the Gen tab.

- Each generation as a surfaced card with hover lift
- Monospace model name, compact date format (e.g., "Mar 15")
- Inset shadow on expanded generation preview
- Accept button with glow on hover
- `card-hover` class on each card
- Generations expand on click to show full prose

### GenerationCompare.tsx

Side-by-side generation comparison view.

- Two panels showing different generation outputs
- Select inputs use `--shadow-inner`, `--radius-sm`

### SceneNode.tsx

Custom React Flow node (vintage index card with pin aesthetic):
- Color-coded border by status (draft=gray, generated=amber, accepted=green, stale=red)
- Status badge pill with `glowPulse` animation
- Beat intent + POV · Tone · Word count metadata
- Inset card highlight, reduced shadow on hover
- `Handle` (target) on left, `Handle` (source) on right
- Wrapped in `memo()` for render performance

### StoryListItem.tsx

Sidebar entry with warm dark styling:
- Status dot color (red=stale, green=all accepted, yellow=mixed, gray=empty)
- Title (bold if active)
- Compact stats: "3ch · 12sc · 8✓ · 2○" (chapters, scenes, accepted, generated)
- Short date format
- `--text-faint` for secondary text

### TurnItem.tsx

Individual agent turn display with expandable input/output.

- Stagger entrance via `animation-delay`
- `expandIn` animation on content reveal
- Role badge, model, duration display
- `glowPulse` on active status indicator

### TurnTimeline.tsx

30-line wrapper that maps scene turns to `TurnItem[]`.

- Loading/empty states use `--text-faint`, `--radius-sm`, `--shadow-inner`
- Container with stagger-fade-in entrance

### AgentRunItem.tsx

Individual agent execution log with expandable details.

- Same expand/collapse pattern as TurnItem
- `expandIn` animation on expanded content
- Stagger entrance via inline `animation-delay`

### AgentRunPanel.tsx

27-line wrapper mapping agent runs to `AgentRunItem[]`.

- `--shadow-inner` on container
- Token count display in subtext

### StatCard.tsx

Shared metric card used by both dashboards.

- Surfaced card with `card-hover` interaction
- Metric value in bold, label in `--text-faint`
- Stagger entrance via CSS `animation-delay`
- Consistent `--radius-md` rounding

### LlmMetricsDashboard.tsx

Token and cost metrics dashboard.

- Uses `StatCard[]` for individual metrics
- `--radius-md` cards, `--text-faint` headings
- Stagger entrance on metric rows

### CriticScoreDashboard.tsx

Critic evaluation scores dashboard.

- Same StatCard/heading patterns as LlmMetricsDashboard
- Score values in monospace

### AuditDashboard.tsx

Code audit findings page (standalone route at `/audit`).

### CompressionStats.tsx

Token compression display showing headroom-ai compression stats.

### Toast.tsx

Toast notification system.

- Colors aligned to `#1a1512`/`#f5f0e8` warm palette
- `--radius-lg` for rounded appearance
- Entrance/exit transitions

---

## React Flow Edge Styles

| Edge Type | Color | Width | Style |
|---|---|---|---|
| `seq` | `#64748b` (gray) | 1.5 | solid |
| `fork` | `#f59e0b` (amber) | 2.5 | solid |
| `join` | `#8b5cf6` (purple) | 2.5 | solid |
| `choice` | `#64748b` (gray) | 2.5 | dashed |

Non-seq edges show their type as a label on the edge.

---

## Color Palette (Warm Dark Theme)

| Token | Hex | Usage |
|---|---|---|
| Page bg | `#1a1512` | Page background (warm dark) |
| Surface | `#2a2420` | Cards, side panel, TopBar |
| --bg-warm | `#231e1a` | Warm accent surfaces |
| --surface-alt | `#332c27` | Hover/lifted states |
| Border | `#3d3530` | Dividers, input borders |
| Text primary | `#f5f0e8` | Body text (warm white) |
| Text muted | `#9c9188` | Secondary labels |
| --text-faint | `#736b64` | Muted metadata, placeholders |
| Accent amber | `#d4a762` | Primary accent, generated status |
| Accent gold | `#e8c876` | Hover/active accent states |
| Green | `#7dab7a` | Accepted status, save |
| Red | `#c4645a` | Stale status, danger |
| Blue | `#6a8fc9` | Info, links |
| Purple | `#9b7bba` | Generation action, join edges |

---

## Design Tokens (`index.css`)

### Shadow System

| Token | Value | Usage |
|---|---|---|
| `--shadow-sm` | `0 1px 2px rgba(0,0,0,0.3)` | Subtle depth |
| `--shadow-md` | `0 2px 8px rgba(0,0,0,0.35)` | Card hover |
| `--shadow-lg` | `0 4px 16px rgba(0,0,0,0.4)` | Modals, dropdowns |
| `--shadow-inner` | `inset 0 1px 3px rgba(0,0,0,0.4)` | Inputs, inset areas |

### Border Radius Tokens

| Token | Value |
|---|---|
| `--radius-sm` | 4px |
| `--radius-md` | 6px |
| `--radius-lg` | 10px |
| `--radius-xl` | 14px |

### Transition Tokens

| Token | Value |
|---|---|
| `--transition-fast` | `150ms ease` |
| `--transition-base` | `250ms ease` |
| `--transition-slow` | `400ms ease` |
| `--ease-spring` | `cubic-bezier(0.34, 1.56, 0.64, 1)` |
| `--transition-spring` | `250ms cubic-bezier(0.34, 1.56, 0.64, 1)` |

---

## Animations (`index.css`)

### Keyframes

| Name | Purpose |
|---|---|
| `fadeIn` / `fadeOut` | Opacity transitions |
| `slideUp` / `slideDown` | Vertical reveal |
| `slideInLeft` / `slideInRight` | Horizontal entrance |
| `scaleIn` / `scaleOut` | Scale-based entrance/exit |
| `pulse` | Slow opacity pulse (idle indicators) |
| `shimmer` | Loading skeleton sweep |
| `spin` | Rotating loader |
| `inkDrop` | Decorative ink drop expansion |
| `glowPulse` | Color glow on status indicators |
| `expandIn` | Scale+fade for expandable sections |
| `breathe` | Subtle scale oscillation |
| `shimmerSlide` | Shimmer with slide motion |

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
8. **Warm dark theme** — `#1a1512` base, `#f5f0e8` text, `#2a2420` surfaces
9. **CSS utility classes for interactions** — `.card-hover`, `.btn-press`, `.stagger-fade-in`, `.stagger-slide-up`
10. **Stagger entrance via CSS `animation-delay`** — keeps component code lean (no IntersectionObserver)
11. **All border-radius via `--radius-*` tokens** — never raw px values
12. **All transitions via `--transition-*` tokens** — consistent timing across components
