# Frontend

React 19 + TypeScript + Vite 8 story graph editor. DAG visualization via React Flow (xyflow). Data fetching via TanStack React Query.

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
    index.css             Global styles (dark theme)
    api/
      client.ts           HTTP client — fetch() wrapper with timeout
      hooks.ts            React Query hooks for every query/mutation
      types.ts            All TypeScript interfaces & payload types
    components/
      Layout.tsx          App shell: TopBar + sidebar + <Outlet/>
      TopBar.tsx          Search bar + nav
      HomeView.tsx        Landing page: create/generate stories
      StoryView.tsx       Story wrapper (reads :storyId param)
      StoryGraph.tsx      React Flow canvas + side panel editor
      SceneNode.tsx       Custom React Flow node
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
                      └── Side panel
                           ├── "Add Scene" button
                           └── Edit form (when node selected)
                                ├── Beat Intent (text)
                                ├── POV (dropdown)
                                ├── Tone (dropdown)
                                ├── Target Words (number)
                                └── Save / Generate buttons
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
| `useChapters(storyId)` | `Chapter[]` | `["chapters", storyId]` |
| `useScenes(storyId, chapterId)` | `Scene[]` | `["scenes", storyId, chapterId]` |
| `useStoryNodeStats(storyId)` | `StoryStats` | `["storyStats", storyId]` |
| `useAllStoryStats(stories)` | `Record<string, StoryStats>` | `["allStoryStats", sortedIds]` |

### Mutations

| Hook | Side Effect |
|---|---|
| `useCreateChapter(storyId)` | Invalidates `["chapters", storyId]` |
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
| `Scene` / `SceneEdge` | Legacy chapter-based model |
| `Generation` | LLM output record |
| `Topology` | Full DAG snapshot |
| `Character` / `Actor` / `Casting` | Character + casting models |
| `Location` / `Lore` | Setting + world-building |
| `Chapter` | Chapter grouping scenes |
| `SceneStructure` / `SceneTurn` | Interactive generation flow |
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
| `inputStyle` | `Record<string, string | number>` | Shared input CSS (spread into `style={}`) |
| `btnStyle(bg, disabled)` | Function → style object | Button styling with disabled state |
| `StoryStats` | `{ total, generated, accepted, stale }` | Sidebar progress display |

---

## Components

### Layout.tsx (app shell)

- `useParams` reads `:storyId` to highlight active story
- `useMemo` filters stories by search query (case-insensitive)
- `useState` for `searchQuery` and `newTitle`
- `useStories()` + `useAllStoryStats()` for sidebar data
- `<Outlet/>` renders child route

### TopBar.tsx

- Controlled search input via `searchQuery` + `onSearchChange` props
- `useNavigate` for programmatic navigation
- Search icon absolutely positioned inside input
- Home button conditionally rendered via `hasActiveStory` prop

### HomeView.tsx

- Two `useState` fields: `newTitle`, `synopsis`
- Error banner via `error` state (dismissible)
- Three mutation hooks: `useCreateStory`, `useGenerateTitle`, `useGenerateStory`
- `mutateAsync` for title generation (awaits result → fills title field)
- Auto-title fallback: first 50 chars of synopsis

### StoryView.tsx

- Extracts `storyId` from URL params via `useParams`
- Simple pass-through to `StoryGraph`

### StoryGraph.tsx

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

**Helpers:**
- `toReactFlowNodes()` — `GraphNode[]` → `Node<SceneNodeData>[]` (grid layout)
- `toReactFlowEdges()` — `GraphEdge[]` → `Edge[]` (color by type)

### SceneNode.tsx

Custom React Flow node with:
- Color-coded border by status (draft=gray, generated=amber, accepted=green, stale=red)
- Status badge pill
- Beat intent + POV · Tone · Word count metadata
- `Handle` (target) on left, `Handle` (source) on right
- Wrapped in `memo()` for render performance

### StoryListItem.tsx

Sidebar entry with:
- Status dot color (red=stale, green=all accepted, yellow=mixed, gray=empty)
- Title (bold if active)
- Stats line: "X chapters · Y scenes · Z accepted · W generated"
- Creation date formatted via `toLocaleDateString()`

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

## Color Palette

| Token | Hex | Usage |
|---|---|---|
| Background | `#0f172a` | Page + sidebar |
| Card/surface | `#1e293b` | TopBar, cards, side panel |
| Border | `#334155` | Dividers, input borders |
| Text primary | `#e2e8f0` | Body text |
| Text muted | `#64748b` | Secondary labels |
| Blue (primary) | `#3b82f6` | Buttons, accents |
| Purple | `#8b5cf6` | Generation action, join edges |
| Amber | `#f59e0b` | Generated status, fork edges |
| Green | `#22c55e` | Accepted status, Save button |
| Red | `#ef4444` | Stale status |

---

## Conventions

1. **All data fetching through `api/hooks.ts`** — components never call `fetch()` directly
2. **One custom hook per domain query** — encapsulates cache key + side effects
3. **Props typed with interfaces** — always explicit, never `any`
4. **`useCallback` for handlers** — prevents child re-renders when passed as props
5. **`useMemo` for derived arrays** — filtered/sorted lists, computed strings
6. **`memo()` on graph nodes** — React Flow performance
7. **Inline styles** — no CSS modules, no Tailwind
8. **Dark theme** — `#0f172a` base, `#e2e8f0` text
