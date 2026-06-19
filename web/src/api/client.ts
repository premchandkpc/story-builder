// ---- Type imports ----
// These are TypeScript interfaces describing the shape of API responses and request payloads.
// We import only the *type* versions (using `import type`) — they vanish at runtime.
import type {
  Actor, Casting, Chapter, Character, CharacterTrait,
  CreateActorPayload, CreateCharacterPayload, CreateCharacterTraitPayload,
  CreateCastingPayload, CreateChapterPayload, CreateLocationPayload,
  CreateNodePayload, CreateScenePayload, CreateStoryPayload, CreateEdgePayload,
  ElevateCheck, Generation, GraphEdge, GraphNode, Location, Lore,
  Scene, SceneEdge, Story, StoryGenerateResult, StorySummary, Topology, TraitAssignment,
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
  // Actors — real people who can be cast
  // ==========================================
  actors: {
    list:   ()                    => request<Actor[]>("/actors"),
    get:    (id: string)          => request<Actor>(`/actors/${id}`),
    create: (data: CreateActorPayload) =>
      request<Actor>("/actors", { method: "POST", body: JSON.stringify(data) }),
    update: (id: string, data: CreateActorPayload) =>
      request<Actor>(`/actors/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  },

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
  // Lore — world-building entries with tag search
  // ==========================================
  lore: {
    list:   () => request<Lore[]>("/lore"),
    create: (data: { tags: string[]; content: string }) =>
      request<Lore>("/lore", { method: "POST", body: JSON.stringify(data) }),
    // search uses POST because it sends a body (tags + optional embeddings)
    search: (data: { tags?: string[]; embedding?: number[]; limit?: number }) =>
      request<Lore[]>("/lore/search", { method: "POST", body: JSON.stringify(data) }),
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
  // Chapters — groups of scenes within a story
  // ==========================================
  chapters: {
    list:   (storyId: string)                  => request<Chapter[]>(`/stories/${storyId}/chapters`),
    get:    (storyId: string, id: string)      => request<Chapter>(`/stories/${storyId}/chapters/${id}`),
    create: (storyId: string, data: CreateChapterPayload) =>
      request<Chapter>(`/stories/${storyId}/chapters`, { method: "POST", body: JSON.stringify(data) }),
    update: (storyId: string, id: string, data: CreateChapterPayload) =>
      request<Chapter>(`/stories/${storyId}/chapters/${id}`, { method: "PUT", body: JSON.stringify(data) }),
    delete: (storyId: string, id: string) =>
      request<void>(`/stories/${storyId}/chapters/${id}`, { method: "DELETE" }),
  },

  // ==========================================
  // Scenes (legacy chapter-based model)
  // ==========================================
  scenes: {
    list:   (storyId: string, chapterId: string)          => request<Scene[]>(`/stories/${storyId}/chapters/${chapterId}/scenes`),
    get:    (storyId: string, chapterId: string, id: string) => request<Scene>(`/stories/${storyId}/chapters/${chapterId}/scenes/${id}`),
    create: (storyId: string, chapterId: string, data: CreateScenePayload) =>
      request<Scene>(`/stories/${storyId}/chapters/${chapterId}/scenes`, { method: "POST", body: JSON.stringify(data) }),
    update: (storyId: string, chapterId: string, id: string, data: CreateScenePayload) =>
      request<Scene>(`/stories/${storyId}/chapters/${chapterId}/scenes/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  },

  // ==========================================
  // Scene Edges (legacy scene-based edges)
  // ==========================================
  sceneEdges: {
    list:   (storyId: string) => request<SceneEdge[]>(`/stories/${storyId}/scene-edges`),
    create: (storyId: string, data: { from_scene: string; to_scene: string; edge_type: string }) =>
      request<void>(`/stories/${storyId}/scene-edges`, { method: "POST", body: JSON.stringify(data) }),
  },

  // ==========================================
  // Nodes (newer graph-based model)
  // ==========================================
  nodes: {
    list:   (storyId: string)                => request<GraphNode[]>(`/stories/${storyId}/nodes`),
    get:    (storyId: string, id: string)    => request<GraphNode>(`/stories/${storyId}/nodes/${id}`),
    create: (storyId: string, data: CreateNodePayload) =>
      request<GraphNode>(`/stories/${storyId}/nodes`, { method: "POST", body: JSON.stringify(data) }),
    update: (storyId: string, id: string, data: CreateNodePayload) =>
      request<GraphNode>(`/stories/${storyId}/nodes/${id}`, { method: "PUT", body: JSON.stringify(data) }),
    delete: (storyId: string, id: string) =>
      request<void>(`/stories/${storyId}/nodes/${id}`, { method: "DELETE" }),
  },

  // ==========================================
  // Node Scene — interactive turn-based scene generation
  // Each node can have an interactive scene with multiple "turns"
  // ==========================================
  nodeScene: {
    // setStructure: configure how the scene flows (monologue, dialogue, etc.)
    setStructure: (storyId: string, nodeId: string, ss: import("./types").SceneStructure) =>
      request<void>(`/stories/${storyId}/nodes/${nodeId}/scene/structure`, {
        method: "PUT",
        body: JSON.stringify({ scene_structure: ss }),
      }),
    // getStructure: retrieve the current scene structure
    getStructure: (storyId: string, nodeId: string) =>
      request<import("./types").SceneStructure>(`/stories/${storyId}/nodes/${nodeId}/scene/structure`),
    // start: begin a new interactive scene generation
    start: (storyId: string, nodeId: string) =>
      request<import("./types").SceneTurn>(`/stories/${storyId}/nodes/${nodeId}/scene/start`, { method: "POST" }),
    // next: advance to the next turn
    next: (storyId: string, nodeId: string) =>
      request<import("./types").SceneTurn>(`/stories/${storyId}/nodes/${nodeId}/scene/next`, { method: "POST" }),
    // finish: end the interactive scene and get final output
    finish: (storyId: string, nodeId: string) =>
      request<{ output: string }>(`/stories/${storyId}/nodes/${nodeId}/scene/finish`, { method: "POST" }),
    // turns: list all turns for this scene
    turns: (storyId: string, nodeId: string) =>
      request<import("./types").SceneTurn[]>(`/stories/${storyId}/nodes/${nodeId}/scene/turns`),
  },
  // Note: `import("./types").SceneStructure` and `import("./types").SceneTurn`
  // are TypeScript "inline import" expressions — they import types without needing
  // to add them to the top-level import block.

  // ==========================================
  // Edges (graph-based connections between nodes)
  // ==========================================
  edges: {
    list:   (storyId: string) => request<GraphEdge[]>(`/stories/${storyId}/edges`),
    get:    (storyId: string, id: string) => request<GraphEdge>(`/stories/${storyId}/edges/${id}`),
    create: (storyId: string, data: CreateEdgePayload) =>
      request<void>(`/stories/${storyId}/edges`, { method: "POST", body: JSON.stringify(data) }),
    delete: (storyId: string, id: string) => request<void>(`/stories/${storyId}/edges/${id}`, { method: "DELETE" }),
  },

  // ==========================================
  // Topology — full DAG structure
  // ==========================================
  topology: {
    get: (storyId: string) => request<Topology>(`/stories/${storyId}/topology`),
  },

  // ==========================================
  // Character Traits — assign/unassign traits
  // ==========================================
  characterTraits: {
    list:          ()                                           => request<CharacterTrait[]>("/character-traits"),
    get:           (id: string)                                 => request<CharacterTrait>(`/character-traits/${id}`),
    create:        (data: CreateCharacterTraitPayload)          =>
      request<CharacterTrait>("/character-traits", { method: "POST", body: JSON.stringify(data) }),
    assign:        (characterId: string, traitId: string, intensity: number, note?: string) =>
      request<void>(`/characters/${characterId}/traits/assign`, {
        method: "POST",
        body: JSON.stringify({ trait_id: traitId, intensity, note: note || "" }),
      }),
    unassign:      (characterId: string, traitId: string) =>
      request<void>(`/characters/${characterId}/traits/${traitId}`, { method: "DELETE" }),
    getAssignments: (characterId: string) =>
      request<TraitAssignment[]>(`/characters/${characterId}/traits`),
  },

  // ==========================================
  // Casting — link actors to characters in stories
  // ==========================================
  casting: {
    create:       (storyId: string, data: CreateCastingPayload) =>
      request<Casting>(`/stories/${storyId}/casting`, { method: "POST", body: JSON.stringify(data) }),
    forStory:     (storyId: string) =>  request<Casting[]>(`/stories/${storyId}/casting`),
    forActor:     (actorId: string) => request<Casting[]>(`/casting/actor/${actorId}`),
    forCharacter: (characterId: string) => request<Casting[]>(`/casting/character/${characterId}`),
  },

  // ==========================================
  // Hierarchical Summaries — multi-level story summaries
  // ==========================================
  summaries: {
    byLevel: (storyId: string, level: "act" | "story") =>
      request<StorySummary>(`/stories/${storyId}/summaries/level?level=${level}`),
    count:   (storyId: string, level: "scene" | "act" | "story") =>
      request<{ count: number }>(`/stories/${storyId}/summaries/count?level=${level}`),
    elevate: (storyId: string, level: string, threshold = 10) =>
      request<ElevateCheck>(`/stories/${storyId}/summaries/elevate?level=${level}&threshold=${threshold}`),
    scene:   (storyId: string, nodeId: string) =>
      request<StorySummary>(`/stories/${storyId}/summaries/nodes/${nodeId}`),
  },

  // ==========================================
  // Generations — LLM prose output
  // ==========================================
  generations: {
    list:     (storyId: string, nodeId: string) =>
      request<Generation[]>(`/stories/${storyId}/nodes/${nodeId}/generations`),
    generate: (storyId: string, nodeId: string) =>
      request<Generation>(`/stories/${storyId}/nodes/${nodeId}/generate`, { method: "POST" }),
    accept:   (storyId: string, nodeId: string, generationId: string) =>
      request<void>(`/stories/${storyId}/nodes/${nodeId}/accept`, {
        method: "POST",
        body: JSON.stringify({ generation_id: generationId }),
      }),
  },
}
