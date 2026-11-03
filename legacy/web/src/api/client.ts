// ---- Type imports ----
// These are TypeScript interfaces describing the shape of API responses and request payloads.
// We import only the *type* versions (using `import type`) — they vanish at runtime.
import type {
  Character, CreateCharacterPayload, CreateLocationPayload,
  CreateNodePayload, CreateStoryPayload, CreateEdgePayload,
  CriticScoreData, Generation, GraphEdge, GraphNode, Location,
  SceneTurn, AgentRun, LlmMetrics,
  Story, StoryGenerateResult, StorySummary, Topology, UpdateNodePayload,
  StoryRun, RunStep, NarrativeEvent, PromptSnapshot, CostSummary, RunStats,
  ScenePlan, GenDiff, TimelineEvent, StoryBible,
} from "./types"

// ---- Base URL ----
// All API paths are prefixed with this.
// Vite dev server proxies `/api/v1/*` to the Go backend (configured in vite.config.ts).
const BASE = "/api/v1"

// ---- Generic request helper ----
// A reusable async function that wraps `fetch()` with:
//   - JSON content-type headers
//   - timeout handling via AbortController
//   - error handling for non-2xx responses
//   - 204 No Content handling
//
// Generics: <T> is the expected response type (e.g. `request<Story[]>` returns `Promise<Story[]>`)
async function request<T>(path: string, init?: RequestInit & { timeout?: number }): Promise<T> {
  // Use provided timeout or default to 30 seconds
  const timeout = init?.timeout ?? 30000
  // AbortController lets us cancel the fetch after timeout
  const controller = new AbortController()
  // Schedule abort after timeout milliseconds
  const timer = setTimeout(() => controller.abort(), timeout)

  try {
    // Make the HTTP request
    const res = await fetch(`${BASE}${path}`, {
      headers: { "Content-Type": "application/json" }, // tell server we send/receive JSON
      signal: controller.signal, // wire up the abort signal
      ...init, // spread any additional options (method, body, etc.)
    })
    // If response is not OK (status outside 200-299), throw an error
    if (!res.ok) {
      const text = await res.text() // get error body as text
      throw new Error(`HTTP ${res.status}: ${text}`)
    }
    // 204 = No Content (used for DELETE, some POSTs). Return undefined cast as T.
    // Otherwise parse JSON response body as type T.
    return res.status === 204 ? (undefined as unknown as T) : res.json()
  } finally {
    // Always clear the timeout (whether request succeeded or failed)
    clearTimeout(timer)
  }
}

// ---- API object ----
// `api` is a namespaced object with all available API methods.
// Each nested group groups related endpoints together.
export const api = {
  // ==========================================
  // Characters — fictional roles
  // ==========================================
  characters: {
    list:   ()                          => request<Character[]>("/characters"),
    get:    (id: string)                => request<Character>(`/characters/${id}`),
    create: (data: CreateCharacterPayload) =>
      request<Character>("/characters", { method: "POST", body: JSON.stringify(data) }),
    update: (id: string, data: CreateCharacterPayload) =>
      request<Character>(`/characters/${id}`, { method: "PUT", body: JSON.stringify(data) }),
    migrate: (storyId: string, charId: string) =>
      request<Character>(`/stories/${storyId}/characters/${charId}/migrate`, { method: "POST" }),
    listByStory: (storyId: string) =>
      request<Character[]>(`/stories/${storyId}/characters`),
  },

  // ==========================================
  // Locations — story settings
  // ==========================================
  locations: {
    list:   ()                         => request<Location[]>("/locations"),
    get:    (id: string)               => request<Location>(`/locations/${id}`),
    create: (data: CreateLocationPayload) =>
      request<Location>("/locations", { method: "POST", body: JSON.stringify(data) }),
    update: (id: string, data: CreateLocationPayload) =>
      request<Location>(`/locations/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  },

  // ==========================================
  // Health check
  // ==========================================
  health: () => request<{ status: string }>('/healthz'),

  // ==========================================
  // Stories — top-level story entities
  // ==========================================
  stories: {
    list:   ()                          => request<Story[]>("/stories"),
    get:    (id: string)                => request<Story>(`/stories/${id}`),
    create: (data: CreateStoryPayload)  =>
      request<Story>("/stories", { method: "POST", body: JSON.stringify(data) }),
    delete: (id: string)                => request<void>(`/stories/${id}`, { method: "DELETE" }),
    // generate: LLM-based full story generation from a synopsis
    generate: (data: { synopsis: string }) =>
      request<StoryGenerateResult>("/stories/generate", { method: "POST", body: JSON.stringify(data), timeout: 300000 }),
    // generateTitle: LLM-based title suggestion from synopsis
    generateTitle: (data: { synopsis: string }) =>
      request<{ title: string }>("/stories/generate-title", { method: "POST", body: JSON.stringify(data) }),
  },

  // ==========================================
  // Nodes (newer graph-based model)
  // ==========================================
  nodes: {
    list:   (storyId: string)                => request<GraphNode[]>(`/stories/${storyId}/nodes`),
    get:    (storyId: string, id: string)    => request<GraphNode>(`/stories/${storyId}/nodes/${id}`),
    create: (storyId: string, data: CreateNodePayload) =>
      request<GraphNode>(`/stories/${storyId}/nodes`, { method: "POST", body: JSON.stringify(data) }),
    update: (storyId: string, id: string, data: UpdateNodePayload) =>
      request<GraphNode>(`/stories/${storyId}/nodes/${id}`, { method: "PUT", body: JSON.stringify(data) }),
    delete: (storyId: string, id: string) =>
      request<void>(`/stories/${storyId}/nodes/${id}`, { method: "DELETE" }),
    updatePosition: (storyId: string, id: string, x: number, y: number) =>
      request<GraphNode>(`/stories/${storyId}/nodes/${id}`, {
        method: "PUT",
        body: JSON.stringify({ position_x: x, position_y: y }),
      }),
    plan: (storyId: string, nodeId: string) =>
      request<ScenePlan>(`/stories/${storyId}/nodes/${nodeId}/plan`),
    diff: (storyId: string, nodeId: string, genA: string, genB: string) =>
      request<GenDiff>(`/stories/${storyId}/nodes/${nodeId}/generations/${genA}/diff?against=${encodeURIComponent(genB)}`),
  },

  // ==========================================
  // Edges (graph-based connections between nodes)
  // ==========================================
  edges: {
    list:   (storyId: string) => request<GraphEdge[]>(`/stories/${storyId}/edges`),
    create: (storyId: string, data: CreateEdgePayload) =>
      request<GraphEdge>(`/stories/${storyId}/edges`, { method: "POST", body: JSON.stringify(data) }),
    delete: (storyId: string, fromNode: string, toNode: string) =>
      request<void>(`/stories/${storyId}/edges?from_scene=${encodeURIComponent(fromNode)}&to_scene=${encodeURIComponent(toNode)}`, { method: "DELETE" }),
    deleteById: (storyId: string, edgeId: string) =>
      request<void>(`/stories/${storyId}/edges/${edgeId}`, { method: "DELETE" }),
  },

  // ==========================================
  // Topology — full DAG structure
  // ==========================================
  topology: {
    get: (storyId: string) => request<Topology>(`/stories/${storyId}/topology`),
  },

  // ==========================================
  // Hierarchical Summaries — multi-level story summaries
  // ==========================================
  summaries: {
    byLevel: (storyId: string, level: "act" | "story") =>
      request<StorySummary>(`/stories/${storyId}/summaries/level?level=${level}`),
    scene:   (storyId: string, nodeId: string) =>
      request<StorySummary>(`/stories/${storyId}/summaries/nodes/${nodeId}`),
  },

  // ==========================================
  // Generations — LLM prose output
  // ==========================================
  generations: {
    list:     (storyId: string, nodeId: string) =>
      request<Generation[]>(`/stories/${storyId}/nodes/${nodeId}/generations`),
    get:      (storyId: string, nodeId: string, genId: string) =>
      request<Generation>(`/stories/${storyId}/nodes/${nodeId}/generations/${genId}`),
    generate: (storyId: string, nodeId: string) =>
      request<Generation>(`/stories/${storyId}/nodes/${nodeId}/generate`, { method: "POST" }),
    accept:   (storyId: string, nodeId: string, generationId: string) =>
      request<void>(`/stories/${storyId}/nodes/${nodeId}/accept`, {
        method: "POST",
        body: JSON.stringify({ generation_id: generationId }),
      }),
  },

  // ==========================================
  // Turns — scene turn playback
  // ==========================================
  turns: {
    list: (storyId: string, nodeId: string) =>
      request<SceneTurn[]>(`/experimental/stories/${storyId}/nodes/${nodeId}/scene/turns`),
    get: (storyId: string, nodeId: string, turnId: string) =>
      request<SceneTurn>(`/experimental/stories/${storyId}/nodes/${nodeId}/scene/turns/${turnId}`),
  },

  // ==========================================
  // Agent Runs — agent execution logs
  // ==========================================
  agentRuns: {
    list: () =>
      request<AgentRun[]>(`/experimental/agent-runs`),
  },

  // ==========================================
  // Runs — durable orchestration tracking
  // ==========================================
  runs: {
    listByStory: (storyId: string, limit?: number) =>
      request<StoryRun[]>(`/stories/${storyId}/runs${limit ? `?limit=${limit}` : ""}`),
    get: (runId: string) =>
      request<StoryRun>(`/runs/${runId}`),
    steps: (runId: string) =>
      request<RunStep[]>(`/runs/${runId}/steps`),
    promptSections: (runId: string) =>
      request<PromptSnapshot>(`/runs/${runId}/prompt-sections`),
    events: (runId: string, limit?: number) =>
      request<NarrativeEvent[]>(`/runs/${runId}/events${limit ? `?limit=${limit}` : ""}`),
    cost: (runId: string) =>
      request<CostSummary>(`/runs/${runId}/cost`),
    cancel: (runId: string) =>
      request<{ status: string }>(`/runs/${runId}/cancel`, { method: "POST" }),
    stats: (storyId: string) =>
      request<RunStats>(`/stories/${storyId}/runs/stats`),
  },

  // ==========================================
  // Narrative Events — append-only state mutation log
  // ==========================================
  narrativeEvents: {
    listByStory: (storyId: string, limit?: number) =>
      request<NarrativeEvent[]>(`/stories/${storyId}/narrative-events${limit ? `?limit=${limit}` : ""}`),
    listByScene: (storyId: string, nodeId: string, limit?: number) =>
      request<NarrativeEvent[]>(`/experimental/stories/${storyId}/nodes/${nodeId}/narrative-events${limit ? `?limit=${limit}` : ""}`),
  },

  // ==========================================
  // LLM Metrics — aggregated token usage
  // ==========================================
  metrics: {
    llm: (storyId: string) =>
      request<LlmMetrics>(`/stories/${storyId}/metrics/llm`),
  },

  // ==========================================
  // Critic Scores — agent quality evaluation
  // ==========================================
  critic: {
    list: (storyId: string) =>
      request<CriticScoreData[]>(`/stories/${storyId}/critic-scores`),
  },

  // Timeline — story events
  // =================================
  timeline: {
    list: (storyId: string) =>
      request<TimelineEvent[]>(`/stories/${storyId}/timeline`),
    crossStoryList: (storyId: string) =>
      request<TimelineEvent[]>(`/stories/${storyId}/timeline/cross-story`),
    createCrossStory: (storyId: string, event: { title: string; related_story_ids?: string[]; description?: string; event_type?: string; order?: number }) =>
      request<TimelineEvent>(`/stories/${storyId}/timeline/cross-story`, { method: "POST", body: JSON.stringify(event) }),
  },

  // Bible — world building + cross-story sharing
  // ============================================
  bible: {
    get: (storyId: string) =>
      request<StoryBible | null>(`/stories/${storyId}/bible`).catch(() => null),
    generate: (storyId: string) =>
      request<StoryBible>(`/stories/${storyId}/bible/generate`, { method: "POST" }),
    update: (storyId: string, bible: Partial<StoryBible>) =>
      request<StoryBible>(`/stories/${storyId}/bible`, {
        method: "PUT",
        body: JSON.stringify(bible),
      }),
    listReferencing: (storyId: string) =>
      request<StoryBible[]>(`/stories/${storyId}/bibles/referencing`),
    link: (storyId: string, bibleId: string) =>
      request<{ status: string }>(`/stories/${storyId}/bibles/link`, {
        method: "POST",
        body: JSON.stringify({ bibleId }),
      }),
    unlink: (storyId: string, bibleId: string) =>
      request<{ status: string }>(`/stories/${storyId}/bibles/unlink`, {
        method: "POST",
        body: JSON.stringify({ bibleId }),
      }),
  },
}
