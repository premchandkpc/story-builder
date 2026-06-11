export interface Character {
  id: string
  version: number
  name: string
  traits: string[]
  voice_samples: string[]
  relationships: Record<string, string>
  created_at: string
}

export interface Location {
  id: string
  version: number
  name: string
  description: string
  props: string[]
  created_at: string
}

export interface Lore {
  id: string
  tags: string[]
  content: string
  created_at: string
}

export interface Story {
  id: string
  title: string
  canon_pins: Record<string, unknown>
  created_at: string
}

export type NodeStatus = "draft" | "generated" | "accepted" | "stale"

export type EdgeType = "seq" | "fork" | "join" | "choice"

export interface GraphNode {
  id: string
  story_id: string
  beat_intent: string
  character_refs: string[]
  location_ref: string | null
  pov: string
  tone: string
  target_words: number
  status: NodeStatus
  created_at: string
  updated_at: string
}

export interface GraphEdge {
  story_id: string
  from_node: string
  to_node: string
  edge_type: EdgeType
}

export interface Generation {
  id: string
  node_id: string
  context_hash: string
  prompt_snapshot: string
  output: string
  model: string
  accepted: boolean
  created_at: string
}

export interface Topology {
  nodes: GraphNode[]
  edges: GraphEdge[]
  topological_order: string[]
}

export interface CreateCharacterPayload {
  name: string
  traits: string[]
  voice_samples: string[]
  relationships: Record<string, string>
}

export interface CreateLocationPayload {
  name: string
  description: string
  props: string[]
}

export interface CreateStoryPayload {
  title: string
}

export interface CreateNodePayload {
  beat_intent: string
  character_refs: string[]
  location_ref?: string | null
  pov: string
  tone: string
  target_words: number
}

export interface CreateEdgePayload {
  from_node: string
  to_node: string
  edge_type: EdgeType
}
