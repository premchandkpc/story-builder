import type {
  Character,
  CreateCharacterPayload,
  CreateLocationPayload,
  CreateNodePayload,
  CreateStoryPayload,
  CreateEdgePayload,
  EdgeType,
  Generation,
  GraphEdge,
  GraphNode,
  Location,
  Lore,
  Story,
  Topology,
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

  // Stories
  stories: {
    list: () => fetchJSON<Story[]>("/stories"),
    get: (id: string) => fetchJSON<Story>(`/stories/${id}`),
    create: (data: CreateStoryPayload) =>
      fetchJSON<Story>("/stories", {
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
