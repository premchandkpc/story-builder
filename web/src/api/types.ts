// ---- Interface: Actor ----
// Defines the shape of an actor object returned from the API.
// An "actor" is a real person who can be cast into a role.
export interface Actor {
  id: string              // unique identifier from MongoDB
  name: string            // display name
  gender: string          // gender identity
  ethnicity: string       // ethnic background
  race: string            // racial category
  skin_tone: string       // skin color description
  eye_color: string       // eye color
  hair_color: string      // hair color
  hair_style: string      // hairstyle description
  build: string           // body type (e.g. "athletic", "slim")
  height_cm: number       // height in centimeters
  weight_kg: number       // weight in kilograms
  age: number             // age in years
  nationality: string     // nationality/country
  traits: Record<string, unknown> // key-value map of miscellaneous traits
  created_at: string      // ISO timestamp when record was created
}

// ---- Interface: Character ----
// A "character" is a fictional role in a story.
// Multiple actors can be cast as the same character across different stories.
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

// ---- Interface: CharacterTrait ----
// Defines a reusable trait (e.g. "brave", "cunning") that can be assigned to characters.
export interface CharacterTrait {
  id: string                // unique identifier
  name: string              // trait name
  category: string          // category grouping (e.g. "personality", "flaw")
  description: string       // explanation of the trait
  created_at: string        // ISO timestamp
}

// ---- Interface: TraitAssignment ----
// Links a specific character to a trait with intensity level.
export interface TraitAssignment {
  character_id: string   // which character has the trait
  trait_id: string       // which trait is assigned
  intensity: number      // how strong the trait manifests (e.g. 1-10)
  note: string           // optional note about the assignment
}

// ---- Interface: Casting ----
// Links an actor to a character within a specific story.
export interface Casting {
  id: string           // unique identifier
  story_id: string     // which story
  actor_id: string     // which actor is cast
  character_id: string // which role/character they play
  role_type: string    // type of role (e.g. "protagonist", "antagonist")
  created_at: string   // ISO timestamp
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

// ---- Interface: Lore ----
// World-building lore entries with tags for search.
export interface Lore {
  id: string           // unique identifier
  tags: string[]       // categorization tags
  content: string      // the lore text content
  created_at: string   // ISO timestamp
}

// ---- Interface: Chapter ----
// A chapter groups scenes within a story, ordered by sort_order.
export interface Chapter {
  id: string           // unique identifier
  story_id: string     // parent story
  title: string        // chapter title
  description: string  // brief description
  sort_order: number   // ordering position within the story
  created_at: string   // ISO timestamp
  updated_at: string   // ISO timestamp of last update
}

// ---- Interface: Scene ----
// A scene is a narrative beat within a chapter.
// This is the "old" scene model (used alongside the newer GraphNode).
export interface Scene {
  id: string                        // unique identifier
  chapter_id: string                // parent chapter
  story_id: string                  // parent story
  beat_intent: string               // what this scene aims to achieve
  character_refs: string[]          // character IDs involved
  location_ref: string | null       // location ID (nullable)
  pov: string                       // point of view style (first-person, third-person, omniscient)
  tone: string                      // emotional tone (e.g. "tense", "neutral")
  target_words: number              // target word count for generation
  status: NodeStatus                // current status (draft/generated/accepted/stale)
  scene_structure?: SceneStructure  // optional structure config for multi-turn scenes
  sort_order: number                // ordering within the chapter
  created_at: string                // ISO timestamp
  updated_at: string                // ISO timestamp of last update
}

// ---- Interface: SceneEdge ----
// Directed edge connecting two scenes in the DAG.
export interface SceneEdge {
  story_id: string       // parent story
  from_scene: string     // source scene ID
  to_scene: string       // target scene ID
  edge_type: EdgeType    // relationship: seq/fork/join/choice
}

// ---- Interface: Story ----
// A story is the top-level entity — a DAG of scenes/nodes.
export interface Story {
  id: string                              // unique identifier
  title: string                           // story title
  canon_pins: Record<string, unknown>     // pinned canon facts (key-value map)
  createdAt: string                       // ISO timestamp
}

// ---- Interface: CreateChapterPayload ----
// Data shape required when creating a new chapter via API.
export interface CreateChapterPayload {
  title: string          // chapter title
  description?: string   // optional description
  sort_order?: number    // optional position override
}

// ---- Interface: CreateScenePayload ----
// Data shape required when creating a new scene via API.
export interface CreateScenePayload {
  beat_intent: string               // what the scene should accomplish
  character_refs: string[]          // characters involved
  location_ref?: string | null      // optional location
  pov: string                       // point of view style
  tone: string                      // emotional tone
  target_words: number              // desired word count
  scene_structure?: SceneStructure  // optional structure for interactive scenes
  sort_order?: number               // optional position
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

// ---- Interface: SceneTurn ----
// Represents one "turn" in an interactive scene generation.
export interface SceneTurn {
  id: string           // unique identifier
  node_id: string      // which node this turn belongs to
  turn_number: number  // turn sequence number
  actor_ids: string[]  // which actors participated in this turn
  prompt: string       // the prompt sent to the LLM
  output: string       // the LLM's generated output
  model: string        // which model generated this turn
  status: string       // status of this turn (e.g. "completed", "pending")
  created_at: string   // ISO timestamp
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
  created_at: string                // ISO timestamp
  updated_at: string                // ISO timestamp of last update
}

// ---- Interface: GraphEdge ----
// Directed edge connecting two GraphNodes in the DAG.
export interface GraphEdge {
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
  created_at: string      // ISO timestamp
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
  level: "scene" | "act" | "story"  // which level of summarization
  content: string                     // summary text
  word_count: number                  // word count of the summary
  created_at: string
}

export interface ElevateCheck {
  should_elevate: boolean  // whether summaries should be promoted to next level
  level: string            // current level checked
  threshold: number        // threshold used for the check
}

export interface StoryGenerateResult {
  story_id: string         // ID of the newly created story
  status: string           // generation status message
}

// ---- Shared UI style objects ----

// inputStyle: a reusable style object for input fields across the app.
// Record<string, string | number> means it's an object with keys→values that are either strings or numbers.
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

// btnStyle: a function that returns a style object for buttons.
// @param bg - background color (CSS value)
// @param disabled - if true, uses gray background and "not-allowed" cursor
export function btnStyle(bg: string, disabled = false): Record<string, string | number> {
  return {
    padding: "10px 16px",
    background: disabled ? "#64748b" : bg,     // gray out if disabled
    color: "#fff",
    border: "none",
    borderRadius: 6,
    cursor: disabled ? "not-allowed" : "pointer", // change cursor when disabled
    fontWeight: 600,
    fontSize: 14,
  }
}

export { inputStyle }
