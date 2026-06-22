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

export interface Location {
  id: string
  version: number
  name: string
  description: string
  props: string[]
  created_at: string
}

export interface Story {
  id: string
  title: string
  canon_pins: Record<string, unknown>
  createdAt: string
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

export interface SceneNodeData extends Record<string, unknown> {
  label: string
  title: string
  status: string
  beatIntent: string
  pov: string
  tone: string
  targetWords: number
}

export interface SceneStructure {
  flow_type: FlowType
  character_order?: string[]
  situation_flow: string
  max_turns?: number
}

export interface GraphNode {
  id: string
  story_id: string
  title: string
  beat_intent: string
  character_refs: string[]
  location_ref: string | null
  pov: string
  tone: string
  target_words: number
  status: NodeStatus
  scene_structure?: SceneStructure
  position_x?: number
  position_y?: number
  created_at: string
  updated_at: string
}

export interface GraphEdge {
  id: string
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
  critic_score?: number
  critic_summary?: string
  created_at: string
}

export interface CriticScoreData {
  generation_id: string
  scene_id: string
  score: number
  summary: string
  created_at: string
}

export interface Topology {
  nodes: GraphNode[]
  edges: GraphEdge[]
  topological_order: string[]
}

export interface StoryBible {
  id: string
  story_id: string
  title: string
  world: string
  dimensions?: { name: string; description: string; physics?: string; timeFlow?: string }[]
  world_rules?: { category: string; description: string; strictness: string }[]
  magic_systems?: { name: string; source: string; cost: string; limitations?: string[]; users?: string[] }[]
  factions?: { name: string; goal: string; resources?: string; members?: string[]; relations?: string }[]
  cultures?: { name: string; values?: string[]; customs?: string[]; technology?: string; government?: string }[]
  tone?: string
  central_theme?: string
  narrative_voice?: string
  reference_stories?: string[]
  created_at: string
}

export interface TimelineEvent {
  id: string
  story_id: string
  related_story_ids?: string[]
  scene_id?: string
  title: string
  event_type?: string
  description?: string
  dependencies?: string[]
  consequences?: string[]
  order: number
  created_at: string
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
  position_x?: number
  position_y?: number
}

export interface CreateEdgePayload {
  from_node: string
  to_node: string
  edge_type: EdgeType
}

export interface UpdateNodePayload {
  beat_intent?: string
  character_refs?: string[]
  location_ref?: string | null
  pov?: string
  tone?: string
  target_words?: number
  scene_structure?: SceneStructure
  position_x?: number
  position_y?: number
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

export interface StoryGenerateResult {
  story_id: string
  status: string
}

export interface SceneTurn {
  id: string
  scene_id: string
  story_id: string
  number: number
  agent_id: string
  role: string
  input: string
  output: string
  model: string
  status: "pending" | "running" | "done" | "failed" | "skipped"
  error: string
  prompt_tokens: number
  completion_tokens: number
  duration_ms: number
  created_at: string
  updated_at: string
}

export interface AgentRun {
  id: string
  story_id: string
  scene_id: string
  turn_id: string
  agent_type: string
  input: Record<string, unknown>
  output: Record<string, unknown>
  model: string
  status: string
  error: string
  duration_ms: number
  created_at: string
}

export interface LlmMetrics {
  total_prompt_tokens: number
  total_completion_tokens: number
  total_tokens: number
  total_cost_estimate: number
  turn_count: number
  generation_count: number
  by_model: Record<string, { prompt_tokens: number; completion_tokens: number; cost: number }>
  by_agent: Record<string, { prompt_tokens: number; completion_tokens: number; turn_count: number }>
}

// ---- Shared UI style objects ----

const inputStyle: Record<string, string | number> = {
  width: "100%",
  padding: "9px 11px",
  background: "var(--surface)",
  border: "1px solid var(--border)",
  borderRadius: 5,
  color: "var(--text)",
  fontSize: 13,
  boxSizing: "border-box",
  outline: "none",
  boxShadow: "var(--shadow-inner)",
  transition: "border-color 0.15s var(--ease-in-out), box-shadow 0.15s var(--ease-in-out)",
}

export function btnStyle(bg: string, disabled = false): Record<string, string | number> {
  return {
    padding: "9px 16px",
    background: disabled ? "var(--text-faint)" : bg,
    color: disabled ? "var(--text-dim)" : bg === "var(--accent)" ? "#1a1512" : "#f5f0e8",
    border: "none",
    borderRadius: 5,
    cursor: disabled ? "not-allowed" : "pointer",
    fontWeight: 600,
    fontSize: 13,
    letterSpacing: "0.01em",
    boxShadow: disabled ? "none" : "0 1px 2px rgba(0,0,0,0.2)",
    transition: "background 0.15s var(--ease-in-out), box-shadow 0.15s, transform 0.1s var(--ease-out)",
  }
}

export function skeletonStyle(width: string | number = "100%", height = 14): Record<string, string | number> {
  return {
    width,
    height,
    borderRadius: 3,
    background: "linear-gradient(90deg, var(--surface) 25%, var(--surface-hover) 50%, var(--surface) 75%)",
    backgroundSize: "200% 100%",
    animation: "shimmer 1.5s infinite",
  }
}

export const spinnerStyle: Record<string, string | number> = {
  width: 18,
  height: 18,
  border: "2px solid var(--border)",
  borderTopColor: "var(--accent)",
  borderRadius: "50%",
  animation: "spin 0.6s linear infinite",
}

export const fadeInStyle: Record<string, string | number> = {
  animation: "fadeIn 0.25s var(--ease-out)",
}

export const slideUpStyle: Record<string, string | number> = {
  animation: "slideUp 0.3s var(--ease-out)",
}

export const scaleInStyle: Record<string, string | number> = {
  animation: "scaleIn 0.2s var(--ease-out)",
}

export const cardStyle: React.CSSProperties = {
  background: "var(--surface)",
  border: "1px solid var(--border)",
  borderRadius: "var(--radius-lg)",
  boxShadow: "var(--shadow-sm)",
}

export const labelStyle: React.CSSProperties = {
  fontSize: 11,
  color: "var(--text-dim)",
  fontWeight: 500,
  display: "block",
  marginBottom: 3,
  letterSpacing: "0.03em",
  textTransform: "uppercase",
}

export function badgeStyle(color: string, bg: string): React.CSSProperties {
  return {
    display: "inline-flex",
    alignItems: "center",
    gap: 4,
    padding: "2px 7px",
    borderRadius: 3,
    fontSize: 10,
    fontWeight: 600,
    textTransform: "uppercase",
    letterSpacing: "0.04em",
    color,
    background: bg,
  }
}

export const ghostBtnStyle: React.CSSProperties = {
  background: "none",
  border: "none",
  color: "var(--text-dim)",
  cursor: "pointer",
  padding: "6px 8px",
  borderRadius: "var(--radius-md)",
  display: "flex",
  alignItems: "center",
  gap: 4,
  fontSize: 12,
  transition: "color var(--transition-fast), background var(--transition-fast)",
}

export const destructiveBtnStyle: React.CSSProperties = {
  padding: "6px 12px",
  background: "transparent",
  border: "1px solid var(--error)",
  borderRadius: "var(--radius-md)",
  color: "var(--error)",
  cursor: "pointer",
  fontSize: 12,
  display: "flex",
  alignItems: "center",
  gap: 4,
  justifyContent: "center",
  transition: "background var(--transition-fast)",
}

export { inputStyle }
