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

async function fetchJSON<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${url}`, {
    headers: { "Content-Type": "application/json" },
    ...init,
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(err.error || res.statusText)
  }
  return res.status === 204 ? (undefined as T) : res.json()
}

export const api = {
  // Actors
  actors: {
    list: () => fetchJSON<Actor[]>("/actors"),
    get: (id: string) => fetchJSON<Actor>(`/actors/${id}`),
    create: (data: CreateActorPayload) =>
      fetchJSON<Actor>("/actors", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    update: (id: string, data: CreateActorPayload) =>
      fetchJSON<Actor>(`/actors/${id}`, {
        method: "PUT",
        body: JSON.stringify(data),
      }),
  },

  // Characters
  characters: {
    list: () => fetchJSON<Character[]>("/characters"),
    get: (id: string) => fetchJSON<Character>(`/characters/${id}`),
    create: (data: CreateCharacterPayload) =>
      fetchJSON<Character>("/characters", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    update: (id: string, data: CreateCharacterPayload) =>
      fetchJSON<Character>(`/characters/${id}`, {
        method: "PUT",
        body: JSON.stringify(data),
      }),
  },

  // Locations
  locations: {
    list: () => fetchJSON<Location[]>("/locations"),
    get: (id: string) => fetchJSON<Location>(`/locations/${id}`),
    create: (data: CreateLocationPayload) =>
      fetchJSON<Location>("/locations", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    update: (id: string, data: CreateLocationPayload) =>
      fetchJSON<Location>(`/locations/${id}`, {
        method: "PUT",
        body: JSON.stringify(data),
      }),
  },

  // Lore
  lore: {
    list: () => fetchJSON<Lore[]>("/lore"),
    create: (data: { tags: string[]; content: string }) =>
      fetchJSON<Lore>("/lore", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    search: (data: { tags?: string[]; embedding?: number[]; limit?: number }) =>
      fetchJSON<Lore[]>("/lore/search", {
        method: "POST",
        body: JSON.stringify(data),
      }),
  },

  // Health
  health: () => fetchJSON<{ status: string }>('/healthz'),

  // Stories
  stories: {
    list: () => fetchJSON<Story[]>("/stories"),
    get: (id: string) => fetchJSON<Story>(`/stories/${id}`),
    create: (data: CreateStoryPayload) =>
      fetchJSON<Story>("/stories", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    generate: (data: { synopsis: string }) =>
      fetchJSON<StoryGenerateResult>("/stories/generate", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    generateTitle: (data: { synopsis: string }) =>
      fetchJSON<{ title: string }>("/stories/generate-title", {
        method: "POST",
        body: JSON.stringify(data),
      }),
  },

  // Nodes
  nodes: {
    list: (storyId: string) =>
      fetchJSON<GraphNode[]>(`/stories/${storyId}/nodes`),
    get: (storyId: string, id: string) =>
      fetchJSON<GraphNode>(`/stories/${storyId}/nodes/${id}`),
    create: (storyId: string, data: CreateNodePayload) =>
      fetchJSON<GraphNode>(`/stories/${storyId}/nodes`, {
        method: "POST",
        body: JSON.stringify(data),
      }),
    update: (storyId: string, id: string, data: CreateNodePayload) =>
      fetchJSON<GraphNode>(`/stories/${storyId}/nodes/${id}`, {
        method: "PUT",
        body: JSON.stringify(data),
      }),
  },

  // Node scene structure (convenience)
  nodeScene: {
    setStructure: (storyId: string, nodeId: string, ss: import("./types").SceneStructure) =>
      fetchJSON<void>(`/stories/${storyId}/nodes/${nodeId}/scene/structure`, {
        method: "PUT",
        body: JSON.stringify({ scene_structure: ss }),
      }),
    getStructure: (storyId: string, nodeId: string) =>
      fetchJSON<import("./types").SceneStructure>(`/stories/${storyId}/nodes/${nodeId}/scene/structure`),
    start: (storyId: string, nodeId: string) =>
      fetchJSON<import("./types").SceneTurn>(`/stories/${storyId}/nodes/${nodeId}/scene/start`, { method: "POST" }),
    next: (storyId: string, nodeId: string) =>
      fetchJSON<import("./types").SceneTurn>(`/stories/${storyId}/nodes/${nodeId}/scene/next`, { method: "POST" }),
    finish: (storyId: string, nodeId: string) =>
      fetchJSON<{ output: string }>(`/stories/${storyId}/nodes/${nodeId}/scene/finish`, { method: "POST" }),
    turns: (storyId: string, nodeId: string) =>
      fetchJSON<import("./types").SceneTurn[]>(`/stories/${storyId}/nodes/${nodeId}/scene/turns`),
  },

  // Edges
  edges: {
    list: (storyId: string) =>
      fetchJSON<GraphEdge[]>(`/stories/${storyId}/edges`),
    create: (storyId: string, data: CreateEdgePayload) =>
      fetchJSON<void>(`/stories/${storyId}/edges`, {
        method: "POST",
        body: JSON.stringify(data),
      }),
  },

  // Topology
  topology: {
    get: (storyId: string) =>
      fetchJSON<Topology>(`/stories/${storyId}/topology`),
  },

  // Character Traits
  characterTraits: {
    list: () => fetchJSON<CharacterTrait[]>("/character-traits"),
    get: (id: string) => fetchJSON<CharacterTrait>(`/character-traits/${id}`),
    create: (data: CreateCharacterTraitPayload) =>
      fetchJSON<CharacterTrait>("/character-traits", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    assign: (characterId: string, traitId: string, intensity: number, note?: string) =>
      fetchJSON<void>(`/characters/${characterId}/traits/assign`, {
        method: "POST",
        body: JSON.stringify({ trait_id: traitId, intensity, note: note || "" }),
      }),
    unassign: (characterId: string, traitId: string) =>
      fetchJSON<void>(`/characters/${characterId}/traits/${traitId}`, {
        method: "DELETE",
      }),
    getAssignments: (characterId: string) =>
      fetchJSON<TraitAssignment[]>(`/characters/${characterId}/traits`),
  },

  // Casting
  casting: {
    create: (storyId: string, data: CreateCastingPayload) =>
      fetchJSON<Casting>(`/stories/${storyId}/casting`, {
        method: "POST",
        body: JSON.stringify(data),
      }),
    forStory: (storyId: string) =>
      fetchJSON<Casting[]>(`/stories/${storyId}/casting`),
    forActor: (actorId: string) =>
      fetchJSON<Casting[]>(`/casting/actor/${actorId}`),
    forCharacter: (characterId: string) =>
      fetchJSON<Casting[]>(`/casting/character/${characterId}`),
  },

  // Hierarchical Summaries
  summaries: {
    byLevel: (storyId: string, level: "act" | "story") =>
      fetchJSON<StorySummary>(`/stories/${storyId}/summaries/level?level=${level}`),
    count: (storyId: string, level: "scene" | "act" | "story") =>
      fetchJSON<{ count: number }>(`/stories/${storyId}/summaries/count?level=${level}`),
    elevate: (storyId: string, level: string, threshold = 10) =>
      fetchJSON<ElevateCheck>(`/stories/${storyId}/summaries/elevate?level=${level}&threshold=${threshold}`),
    scene: (storyId: string, nodeId: string) =>
      fetchJSON<StorySummary>(`/stories/${storyId}/summaries/nodes/${nodeId}`),
  },

  // Generations
  generations: {
    list: (storyId: string, nodeId: string) =>
      fetchJSON<Generation[]>(`/stories/${storyId}/nodes/${nodeId}/generations`),
    generate: (storyId: string, nodeId: string) =>
      fetchJSON<Generation>(`/stories/${storyId}/nodes/${nodeId}/generate`, {
        method: "POST",
      }),
    accept: (storyId: string, nodeId: string, generationId: string) =>
      fetchJSON<void>(`/stories/${storyId}/nodes/${nodeId}/accept`, {
        method: "POST",
        body: JSON.stringify({ generation_id: generationId }),
      }),
  },
}
