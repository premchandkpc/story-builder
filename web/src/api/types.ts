export interface Actor {
  id: string
  name: string
  gender: string
  ethnicity: string
  race: string
  skin_tone: string
  eye_color: string
  hair_color: string
  hair_style: string
  build: string
  height_cm: number
  weight_kg: number
  age: number
  nationality: string
  traits: Record<string, unknown>
  created_at: string
}

export interface Character {
  id: string
  version: number
  name: string
  persona: string
  backstory: string
  moral_alignment: string
  personality: string[]
  flaws: string[]
  goals: string[]
  traits: string[]
  voice_samples: string[]
  parent_id: string | null
  relationships: Record<string, string>
  created_at: string
}

export interface CharacterTrait {
  id: string
  name: string
  category: string
  description: string
  created_at: string
}

export interface TraitAssignment {
  character_id: string
  trait_id: string
  intensity: number
  note: string
}

export interface Casting {
  id: string
  story_id: string
  actor_id: string
  character_id: string
  role_type: string
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

export interface Chapter {
  id: string
  story_id: string
  title: string
  description: string
  sort_order: number
  created_at: string
  updated_at: string
}

export interface Scene {
  id: string
  chapter_id: string
  story_id: string
  beat_intent: string
  character_refs: string[]
  location_ref: string | null
  pov: string
  tone: string
  target_words: number
  status: NodeStatus
  scene_structure?: SceneStructure
  sort_order: number
  created_at: string
  updated_at: string
}

export interface SceneEdge {
  story_id: string
  from_scene: string
  to_scene: string
  edge_type: EdgeType
}

export interface Story {
  id: string
  title: string
  canon_pins: Record<string, unknown>
  created_at: string
}

export interface CreateChapterPayload {
  title: string
  description?: string
  sort_order?: number
}

export interface CreateScenePayload {
  beat_intent: string
  character_refs: string[]
  location_ref?: string | null
  pov: string
  tone: string
  target_words: number
  scene_structure?: SceneStructure
  sort_order?: number
}

export interface StoryStats {
  total: number
  generated: number
  accepted: number
  stale: number
}

export type NodeStatus = "draft" | "generated" | "accepted" | "stale"

export type EdgeType = "seq" | "fork" | "join" | "choice"

export type FlowType = "monologue" | "dialogue" | "round_robin" | "parallel" | "custom"

export interface SceneStructure {
  flow_type: FlowType
  character_order?: string[]
  situation_flow: string
  max_turns?: number
}

export interface SceneTurn {
  id: string
  node_id: string
  turn_number: number
  actor_ids: string[]
  prompt: string
  output: string
  model: string
  status: string
  created_at: string
}

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
  scene_structure?: SceneStructure
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
  persona?: string
  backstory?: string
  moral_alignment?: string
  personality?: string[]
  flaws?: string[]
  goals?: string[]
  traits: string[]
  voice_samples: string[]
  parent_id?: string
  relationships: Record<string, string>
}

export interface CreateActorPayload {
  name: string
  gender?: string
  ethnicity?: string
  race?: string
  skin_tone?: string
  eye_color?: string
  hair_color?: string
  hair_style?: string
  build?: string
  height_cm?: number
  weight_kg?: number
  age?: number
  nationality?: string
  traits?: Record<string, unknown>
}

export interface CreateCharacterTraitPayload {
  name: string
  category?: string
  description?: string
}

export interface CreateCastingPayload {
  actor_id: string
  character_id: string
  role_type: string
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
  scene_structure?: SceneStructure
}

export interface CreateEdgePayload {
  from_node: string
  to_node: string
  edge_type: EdgeType
}

export interface StorySummary {
  id: string
  story_id: string
  node_id?: string
  level: "scene" | "act" | "story"
  content: string
  word_count: number
  created_at: string
}

export interface ElevateCheck {
  should_elevate: boolean
  level: string
  threshold: number
}

export interface StoryGenerateResult {
  story_id: string
  status: string
}

const inputStyle: Record<string, string | number> = {
  width: "100%",
  padding: "10px 12px",
  background: "#1e293b",
  border: "1px solid #334155",
  borderRadius: 6,
  color: "#e2e8f0",
  fontSize: 14,
  boxSizing: "border-box",
  outline: "none",
}

export function btnStyle(bg: string, disabled = false): Record<string, string | number> {
  return {
    padding: "10px 16px",
    background: disabled ? "#64748b" : bg,
    color: "#fff",
    border: "none",
    borderRadius: 6,
    cursor: disabled ? "not-allowed" : "pointer",
    fontWeight: 600,
    fontSize: 14,
  }
}

export { inputStyle }
