// ---- Interface: Character ----
// A "character" is a fictional role in a story.
export interface Character {
  id: string                       // unique identifier
  version: number                  // version number for optimistic concurrency
  name: string                     // character name
  persona: string                  // personality/persona description
  backstory: string                // character's history
  moral_alignment: string          // alignment (e.g. "good", "evil", "neutral")
  personality: string[]            // array of personality trait labels
  flaws: string[]                  // character flaws
  goals: string[]                  // character goals/motivations
  traits: string[]                 // references to CharacterTrait IDs
  voice_samples: string[]          // example dialogue snippets
  parent_id: string | null         // for version branching — links to parent version
  relationships: Record<string, string> // map of character_id -> relationship description
  created_at: string               // ISO timestamp
}

// ---- Interface: Location ----
// A location/setting that scenes can reference.
export interface Location {
  id: string           // unique identifier
  version: number      // version number for concurrency
  name: string         // location name
  description: string  // description of the location
  props: string[]      // notable objects/props at this location
  created_at: string   // ISO timestamp
}

// ---- Interface: Story ----
// A story is the top-level entity — a DAG of scenes/nodes.
export interface Story {
  id: string                              // unique identifier
  title: string                           // story title
  canon_pins: Record<string, unknown>     // pinned canon facts (key-value map)
  createdAt: string                       // ISO timestamp
}

// ---- Interface: StoryStats ----
// Aggregate counts used for displaying story progress in the sidebar.
export interface StoryStats {
  total: number      // total number of nodes/scenes
  generated: number  // count of nodes with generated content
  accepted: number   // count of nodes where generation was accepted
  stale: number      // count of nodes needing regeneration
}

// ---- Type: NodeStatus ----
// Union type representing the lifecycle of a scene/node.
export type NodeStatus = "draft" | "generated" | "accepted" | "stale"
//   draft     = not yet generated
//   generated = LLM output exists but not reviewed
//   accepted  = user approved the generation
//   stale     = dependencies changed, needs re-generation

// ---- Type: EdgeType ----
// Types of directed edges in the story DAG.
export type EdgeType = "seq" | "fork" | "join" | "choice"
//   seq    = sequential progression (default)
//   fork   = story branches into multiple paths
//   join   = branches converge back together
//   choice = reader/character choice point

// ---- Type: FlowType ----
// Defines how a scene's dialogue/prose should be structured.
export type FlowType = "monologue" | "dialogue" | "round_robin" | "parallel" | "custom"
//   monologue   = single character speaks
//   dialogue    = two characters converse
//   round_robin = characters take turns
//   parallel    = simultaneous action
//   custom      = user-defined structure

// ---- Interface: SceneStructure ----
// Configures the interactive generation flow for a scene.
export interface SceneStructure {
  flow_type: FlowType              // how the scene flows
  character_order?: string[]       // turn order for round_robin flow
  situation_flow: string           // description of how the scene should progress
  max_turns?: number               // max LLM turns before finishing
}

// ---- Interface: GraphNode ----
// The primary node model for the story DAG (used by React Flow).
// This is the "newer" graph-based model that replaces the old Scene model.
export interface GraphNode {
  id: string                        // unique identifier
  story_id: string                  // parent story
  beat_intent: string               // narrative purpose of this node
  character_refs: string[]          // characters involved
  location_ref: string | null       // optional location
  pov: string                       // point of view
  tone: string                      // emotional tone
  target_words: number              // word count target
  status: NodeStatus                // generation lifecycle status
  scene_structure?: SceneStructure  // optional structure for interactive flow
  position_x?: number               // persisted X coordinate in graph layout
  position_y?: number               // persisted Y coordinate in graph layout
  created_at: string                // ISO timestamp
  updated_at: string                // ISO timestamp of last update
}

// ---- Interface: GraphEdge ----
// Directed edge connecting two GraphNodes in the DAG.
export interface GraphEdge {
  id: string            // unique identifier (MongoDB ObjectID hex)
  story_id: string      // parent story
  from_node: string     // source node ID
  to_node: string       // target node ID
  edge_type: EdgeType   // relationship type
}

// ---- Interface: Generation ----
// Records an LLM generation attempt for a specific node.
export interface Generation {
  id: string              // unique identifier
  node_id: string         // which node was generated
  context_hash: string    // hash of context at generation time (for staleness detection)
  prompt_snapshot: string // the actual prompt sent to the LLM
  output: string          // the generated prose
  model: string           // model used (e.g. "claude-sonnet")
  accepted: boolean       // whether user accepted this generation
  critic_score?: number   // critic agent score (0.0-1.0)
  critic_summary?: string // critic agent verdict
  created_at: string      // ISO timestamp
}

export interface CriticScoreData {
  generation_id: string
  scene_id: string
  score: number
  summary: string
  created_at: string
}

// ---- Interface: Topology ----
// Full DAG topology fetched from the server for rendering in React Flow.
export interface Topology {
  nodes: GraphNode[]           // all nodes in the story
  edges: GraphEdge[]           // all edges connecting nodes
  topological_order: string[]  // node IDs in topological (dependency) order
}

// ---- Create payloads (sent to API when creating new entities) ----

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
  level: "scene" | "act" | "story"  // which level of summarization
  content: string                     // summary text
  word_count: number                  // word count of the summary
  created_at: string
}

export interface StoryGenerateResult {
  story_id: string         // ID of the newly created story
  status: string           // generation status message
}

// ---- Shared UI style objects ----

const inputStyle: Record<string, string | number> = {
  width: "100%",
  padding: "10px 12px",
  background: "var(--surface)",
  border: "1px solid var(--border)",
  borderRadius: 6,
  color: "var(--text)",
  fontSize: 14,
  boxSizing: "border-box",
  outline: "none",
  transition: "border-color 0.15s var(--ease-in-out), box-shadow 0.15s var(--ease-in-out)",
}

export const inputFocusStyle: Record<string, string | number> = {
  borderColor: "var(--accent)",
  boxShadow: "0 0 0 3px var(--accent-dim)",
}

export function btnStyle(bg: string, disabled = false): Record<string, string | number> {
  return {
    padding: "10px 16px",
    background: disabled ? "var(--text-muted)" : bg,
    color: "#fff",
    border: "none",
    borderRadius: 6,
    cursor: disabled ? "not-allowed" : "pointer",
    fontWeight: 600,
    fontSize: 14,
    transition: "background 0.15s var(--ease-in-out), transform 0.1s var(--ease-out), box-shadow 0.15s",
  }
}

export function skeletonStyle(width: string | number = "100%", height = 14): Record<string, string | number> {
  return {
    width,
    height,
    borderRadius: 4,
    background: "linear-gradient(90deg, var(--surface) 25%, var(--surface-hover) 50%, var(--surface) 75%)",
    backgroundSize: "200% 100%",
    animation: "shimmer 1.5s infinite",
  }
}

export const spinnerStyle: Record<string, string | number> = {
  width: 20,
  height: 20,
  border: "2px solid var(--border)",
  borderTopColor: "var(--accent)",
  borderRadius: "50%",
  animation: "spin 0.7s linear infinite",
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
  fontSize: 12,
  color: "var(--text-muted)",
  fontWeight: 500,
  display: "block",
  marginBottom: 4,
}

export const badgeStyle = (color: string, bg: string): React.CSSProperties => ({
  display: "inline-flex",
  alignItems: "center",
  gap: 4,
  padding: "2px 8px",
  borderRadius: "var(--radius-sm)",
  fontSize: 10,
  fontWeight: 600,
  textTransform: "uppercase",
  letterSpacing: "0.04em",
  color,
  background: bg,
})

export const ghostBtnStyle: React.CSSProperties = {
  background: "none",
  border: "none",
  color: "var(--text-muted)",
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

// ---- Interface: SceneTurn ----
// A single turn in a scene's interactive generation flow.
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

// ---- Interface: AgentRun ----
// Execution log for a single agent during generation.
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

// ---- Interface: LlmMetrics ----
// Aggregated LLM usage metrics for a story.
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

export { inputStyle }
