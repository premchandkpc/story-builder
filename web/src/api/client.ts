import type {
  Actor,
  Casting,
  Character,
  CharacterTrait,
  CreateActorPayload,
  CreateCharacterPayload,
  CreateCharacterTraitPayload,
  CreateCastingPayload,
  CreateLocationPayload,
  CreateNodePayload,
  CreateStoryPayload,
  CreateEdgePayload,
  ElevateCheck,
  Generation,
  GraphEdge,
  GraphNode,
  Location,
  Lore,
  Story,
  StoryGenerateResult,
  StorySummary,
  Topology,
  TraitAssignment,
} from "./types"

const BASE = "/api/v1"

async function request<T>(path: string, init?: RequestInit & { timeout?: number }): Promise<T> {
  const timeout = init?.timeout ?? 30000
  const controller = new AbortController()
  const timer = setTimeout(() => controller.abort(), timeout)

  try {
    const res = await fetch(`${BASE}${path}`, {
      headers: { "Content-Type": "application/json" },
      signal: controller.signal,
      ...init,
    })
    if (!res.ok) {
      const text = await res.text()
      throw new Error(`HTTP ${res.status}: ${text}`)
    }
    return res.status === 204 ? (undefined as unknown as T) : res.json()
  } finally {
    clearTimeout(timer)
  }
}

export const api = {
  // Actors
  actors: {
    list: () => request<Actor[]>("/actors"),
    get: (id: string) => request<Actor>(`/actors/${id}`),
    create: (data: CreateActorPayload) =>
      request<Actor>("/actors", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    update: (id: string, data: CreateActorPayload) =>
      request<Actor>(`/actors/${id}`, {
        method: "PUT",
        body: JSON.stringify(data),
      }),
  },

  // Characters
  characters: {
    list: () => request<Character[]>("/characters"),
    get: (id: string) => request<Character>(`/characters/${id}`),
    create: (data: CreateCharacterPayload) =>
      request<Character>("/characters", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    update: (id: string, data: CreateCharacterPayload) =>
      request<Character>(`/characters/${id}`, {
        method: "PUT",
        body: JSON.stringify(data),
      }),
  },

  // Locations
  locations: {
    list: () => request<Location[]>("/locations"),
    get: (id: string) => request<Location>(`/locations/${id}`),
    create: (data: CreateLocationPayload) =>
      request<Location>("/locations", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    update: (id: string, data: CreateLocationPayload) =>
      request<Location>(`/locations/${id}`, {
        method: "PUT",
        body: JSON.stringify(data),
      }),
  },

  // Lore
  lore: {
    list: () => request<Lore[]>("/lore"),
    create: (data: { tags: string[]; content: string }) =>
      request<Lore>("/lore", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    search: (data: { tags?: string[]; embedding?: number[]; limit?: number }) =>
      request<Lore[]>("/lore/search", {
        method: "POST",
        body: JSON.stringify(data),
      }),
  },

  // Health
  health: () => request<{ status: string }>('/healthz'),

  // Stories
  stories: {
    list: () => request<Story[]>("/stories"),
    get: (id: string) => request<Story>(`/stories/${id}`),
    create: (data: CreateStoryPayload) =>
      request<Story>("/stories", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    generate: (data: { synopsis: string }) =>
      request<StoryGenerateResult>("/stories/generate", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    generateTitle: (data: { synopsis: string }) =>
      request<{ title: string }>("/stories/generate-title", {
        method: "POST",
        body: JSON.stringify(data),
      }),
  },

  // Nodes
  nodes: {
    list: (storyId: string) =>
      request<GraphNode[]>(`/stories/${storyId}/nodes`),
    get: (storyId: string, id: string) =>
      request<GraphNode>(`/stories/${storyId}/nodes/${id}`),
    create: (storyId: string, data: CreateNodePayload) =>
      request<GraphNode>(`/stories/${storyId}/nodes`, {
        method: "POST",
        body: JSON.stringify(data),
      }),
    update: (storyId: string, id: string, data: CreateNodePayload) =>
      request<GraphNode>(`/stories/${storyId}/nodes/${id}`, {
        method: "PUT",
        body: JSON.stringify(data),
      }),
  },

  // Node scene structure (convenience)
  nodeScene: {
    setStructure: (storyId: string, nodeId: string, ss: import("./types").SceneStructure) =>
      request<void>(`/stories/${storyId}/nodes/${nodeId}/scene/structure`, {
        method: "PUT",
        body: JSON.stringify({ scene_structure: ss }),
      }),
    getStructure: (storyId: string, nodeId: string) =>
      request<import("./types").SceneStructure>(`/stories/${storyId}/nodes/${nodeId}/scene/structure`),
    start: (storyId: string, nodeId: string) =>
      request<import("./types").SceneTurn>(`/stories/${storyId}/nodes/${nodeId}/scene/start`, { method: "POST" }),
    next: (storyId: string, nodeId: string) =>
      request<import("./types").SceneTurn>(`/stories/${storyId}/nodes/${nodeId}/scene/next`, { method: "POST" }),
    finish: (storyId: string, nodeId: string) =>
      request<{ output: string }>(`/stories/${storyId}/nodes/${nodeId}/scene/finish`, { method: "POST" }),
    turns: (storyId: string, nodeId: string) =>
      request<import("./types").SceneTurn[]>(`/stories/${storyId}/nodes/${nodeId}/scene/turns`),
  },

  // Edges
  edges: {
    list: (storyId: string) =>
      request<GraphEdge[]>(`/stories/${storyId}/edges`),
    create: (storyId: string, data: CreateEdgePayload) =>
      request<void>(`/stories/${storyId}/edges`, {
        method: "POST",
        body: JSON.stringify(data),
      }),
  },

  // Topology
  topology: {
    get: (storyId: string) =>
      request<Topology>(`/stories/${storyId}/topology`),
  },

  // Character Traits
  characterTraits: {
    list: () => request<CharacterTrait[]>("/character-traits"),
    get: (id: string) => request<CharacterTrait>(`/character-traits/${id}`),
    create: (data: CreateCharacterTraitPayload) =>
      request<CharacterTrait>("/character-traits", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    assign: (characterId: string, traitId: string, intensity: number, note?: string) =>
      request<void>(`/characters/${characterId}/traits/assign`, {
        method: "POST",
        body: JSON.stringify({ trait_id: traitId, intensity, note: note || "" }),
      }),
    unassign: (characterId: string, traitId: string) =>
      request<void>(`/characters/${characterId}/traits/${traitId}`, {
        method: "DELETE",
      }),
    getAssignments: (characterId: string) =>
      request<TraitAssignment[]>(`/characters/${characterId}/traits`),
  },

  // Casting
  casting: {
    create: (storyId: string, data: CreateCastingPayload) =>
      request<Casting>(`/stories/${storyId}/casting`, {
        method: "POST",
        body: JSON.stringify(data),
      }),
    forStory: (storyId: string) =>
      request<Casting[]>(`/stories/${storyId}/casting`),
    forActor: (actorId: string) =>
      request<Casting[]>(`/casting/actor/${actorId}`),
    forCharacter: (characterId: string) =>
      request<Casting[]>(`/casting/character/${characterId}`),
  },

  // Hierarchical Summaries
  summaries: {
    byLevel: (storyId: string, level: "act" | "story") =>
      request<StorySummary>(`/stories/${storyId}/summaries/level?level=${level}`),
    count: (storyId: string, level: "scene" | "act" | "story") =>
      request<{ count: number }>(`/stories/${storyId}/summaries/count?level=${level}`),
    elevate: (storyId: string, level: string, threshold = 10) =>
      request<ElevateCheck>(`/stories/${storyId}/summaries/elevate?level=${level}&threshold=${threshold}`),
    scene: (storyId: string, nodeId: string) =>
      request<StorySummary>(`/stories/${storyId}/summaries/nodes/${nodeId}`),
  },

  // Generations
  generations: {
    list: (storyId: string, nodeId: string) =>
      request<Generation[]>(`/stories/${storyId}/nodes/${nodeId}/generations`),
    generate: (storyId: string, nodeId: string) =>
      request<Generation>(`/stories/${storyId}/nodes/${nodeId}/generate`, {
        method: "POST",
      }),
    accept: (storyId: string, nodeId: string, generationId: string) =>
      request<void>(`/stories/${storyId}/nodes/${nodeId}/accept`, {
        method: "POST",
        body: JSON.stringify({ generation_id: generationId }),
      }),
  },
}
