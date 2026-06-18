// ---- StoryGraph ----
// The main story graph editor. Uses React Flow (@xyflow/react) to render a
// DAG of story scenes as nodes connected by edges.
//
// Features:
//   - Renders the story DAG with custom SceneNode components
//   - Add/select/edit/delete nodes
//   - Connect nodes by drawing edges
//   - Generate LLM content for selected nodes
//   - Side panel for editing selected node properties

// useCallback: React hook that memoizes a function reference.
//   - The function is only recreated when its dependencies change.
//   - Prevents unnecessary re-renders of child components.
//
// useEffect: React hook that runs side effects after render.
//   - Takes a function (the effect) and a dependency array.
//   - The effect runs when dependencies change (and on mount if array is non-empty).
//
// useState: React hook for component-level state.
import { useCallback, useEffect, useState } from "react"

// React Flow components and utilities:
import {
  ReactFlow,          // The main React Flow canvas component
  Background,         // Grid/dot background
  Controls,           // Zoom controls (+/-/fit)
  MiniMap,            // Small overview map in the corner
  type Node,          // TypeScript type for a React Flow node
  type Edge,          // TypeScript type for a React Flow edge
  type Connection,    // TypeScript type for a new connection being created
  useNodesState,      // React Flow hook: manages array of nodes + provides onChange handler
  useEdgesState,      // React Flow hook: manages array of edges + provides onChange handler
  addEdge,            // Utility: creates a new edge from a Connection object
  type NodeTypes,     // TypeScript type for the node type registry
} from "@xyflow/react"

// React Flow's default styles (grid, controls, minimap appearance)
import "@xyflow/react/dist/style.css"

import SceneNode from "./SceneNode"     // Our custom node component
import { api } from "../api/client"     // API client for backend calls
import type { GraphNode, GraphEdge } from "../api/types"

// ---- Local type: SceneNodeData ----
// Defines the shape of data attached to each React Flow node.
// Extends Record<string, unknown> so it satisfies React Flow's type constraints.
interface SceneNodeData extends Record<string, unknown> {
  label: string         // display label
  status: string        // generation status
  beatIntent: string    // narrative purpose
  pov: string           // point of view
  tone: string          // emotional tone
  targetWords: number   // word count target
}

// ---- Node type registry ----
// Maps node type names to React components.
// "scene" -> SceneNode component. This tells React Flow:
//   "When rendering a node with type='scene', use SceneNode."
const nodeTypes: NodeTypes = { scene: SceneNode }

// ---- Props interface ----
interface StoryGraphProps {
  storyId: string       // which story to load/edit
}

// ---- Helper: toReactFlowNodes ----
// Converts backend GraphNode[] into React Flow Node[].
// Preserves existing positions for known nodes; assigns grid positions for new ones.
function toReactFlowNodes(
  nodes: GraphNode[],
  existingNodes: Node<SceneNodeData>[] = [],
): Node<SceneNodeData>[] {
  const posMap = new Map(existingNodes.map((n) => [n.id, n.position]))
  const cols = 6
  let newIdx = 0
  return nodes.map((n) => {
    const existing = posMap.get(n.id)
    if (existing) {
      return {
        id: n.id,
        type: "scene",
        position: existing,
        data: {
          label: `Node ${n.id.slice(-4)}`,
          status: n.status,
          beatIntent: n.beat_intent || "",
          pov: n.pov || "",
          tone: n.tone || "",
          targetWords: n.target_words,
        },
      }
    }
    const pos = {
      x: 300 * (newIdx % cols),
      y: 200 * Math.floor(newIdx / cols),
    }
    newIdx++
    return {
      id: n.id,
      type: "scene",
      position: pos,
      data: {
        label: `Node ${n.id.slice(-4)}`,
        status: n.status,
        beatIntent: n.beat_intent || "",
        pov: n.pov || "",
        tone: n.tone || "",
        targetWords: n.target_words,
      },
    }
  })
}

// ---- Helper: toReactFlowEdges ----
// Converts backend GraphEdge[] into React Flow Edge[].
// Applies different visual styles based on edge type.
function toReactFlowEdges(edges: GraphEdge[]): Edge[] {
  return edges.map((e, i) => ({
    id: `e-${i}`,                        // unique edge ID (e-0, e-1, ...)
    source: e.from_node,                 // source node ID
    target: e.to_node,                   // target node ID
    label: e.edge_type === "seq" ? "" : e.edge_type, // show label only for non-seq edges
    style: {
      stroke: e.edge_type === "fork"  ? "#f59e0b" :   // amber for forks
              e.edge_type === "join"  ? "#8b5cf6" :    // purple for joins
                                        "#64748b",     // gray for seq/choice
      strokeWidth: e.edge_type === "seq" ? 1.5 : 2.5, // thicker for non-seq
      strokeDasharray: e.edge_type === "choice" ? "5 5" : undefined, // dashed for choice
    },
    labelStyle: { fill: "#94a3b8", fontSize: 10 },
  }))
}

// ---- Component ----
export default function StoryGraph({ storyId }: StoryGraphProps) {
  // ---- React Flow state ----
  // useNodesState returns: [nodes, setNodes, onNodesChange]
  //   - nodes: current array of Node objects
  //   - setNodes: function to replace nodes
  //   - onNodesChange: handler for React Flow's built-in node operations (drag, select, etc.)
  const [nodes, setNodes, onNodesChange] = useNodesState([] as Node<SceneNodeData>[])

  // Same pattern for edges
  const [edges, setEdges, onEdgesChange] = useEdgesState([] as Edge[])

  // ---- Local state ----
  // selectedNode: the currently selected node (for editing in side panel)
  const [selectedNode, setSelectedNode] = useState<Node<SceneNodeData> | null>(null)

  // form: the edit form fields, synced with the selected node
  const [form, setForm] = useState({
    beat_intent: "",
    pov: "",
    tone: "",
    target_words: 300,
  })

  // ---- fetchGraph ----
  // useCallback: memoizes this function so it doesn't get recreated on every render.
  // Re-fetches the topology from the server and updates React Flow state.
  // Preserves existing node positions so user-dragged positions aren't lost.
  const fetchGraph = useCallback(async () => {
    try {
      // Fetch topology (nodes + edges + topological order)
      const topo = await api.topology.get(storyId)
      setNodes((prev) => toReactFlowNodes(topo.nodes || [], prev)) // preserve positions
      setEdges(toReactFlowEdges(topo.edges || []))
    } catch (err) {
      console.error("fetch graph:", err)
    }
  }, [storyId, setNodes, setEdges])  // dependencies: re-create when storyId changes

  // ---- useEffect: fetch graph on mount ----
  // The empty dependency array `[fetchGraph]` means this runs when component mounts
  // and whenever fetchGraph changes identity (which only happens when storyId changes).
  useEffect(() => {
    fetchGraph()
  }, [fetchGraph])

  // ---- onConnect ----
  // Called when user drags a connection between two handles.
  // Updates local state immediately, then persists to the server.
  const onConnect = useCallback(
    async (connection: Connection) => {
      // Connection has source, target, sourceHandle, targetHandle
      if (!connection.source || !connection.target) return

      // Optimistically add the edge to local state
      setEdges((eds: Edge[]) => addEdge({
        ...connection,
        id: `e-${Date.now()}`,          // unique ID using timestamp
        style: { stroke: "#64748b" },   // default style
      }, eds))

      // Persist to server
      try {
        await api.edges.create(storyId, {
          from_node: connection.source,
          to_node: connection.target,
          edge_type: "seq",             // default to sequential
        })
      } catch (err) {
        console.error("create edge:", err)
      }
    },
    [storyId, setEdges],  // dependencies
  )

  // ---- onNodeClick ----
  // Called when user clicks a node in the graph.
  // Sets the selected node and populates the edit form.
  const onNodeClick = useCallback((_: unknown, node: Node) => {
    const d = node.data as unknown as SceneNodeData      // extract custom data
    setSelectedNode(node as unknown as Node<SceneNodeData>)
    setForm({
      beat_intent: d.beatIntent || "",
      pov: d.pov || "",
      tone: d.tone || "",
      target_words: d.targetWords || 300,
    })
  }, [])  // no dependencies — stable function

  // ---- addNode ----
  // Creates a new scene node via API, then refreshes the graph.
  const addNode = useCallback(async () => {
    try {
      await api.nodes.create(storyId, {
        beat_intent: "New scene",
        character_refs: [],
        location_ref: null,
        pov: "third-person",
        tone: "neutral",
        target_words: 300,
      })
      await fetchGraph()    // re-fetch to include the new node
    } catch (err) {
      console.error("add node:", err)
    }
  }, [storyId, fetchGraph])

  // ---- updateNode ----
  // Updates the selected node's properties and refreshes the graph.
  const updateNode = useCallback(async () => {
    if (!selectedNode) return
    try {
      await api.nodes.update(storyId, selectedNode.id, {
        beat_intent: form.beat_intent,
        character_refs: [],
        pov: form.pov,
        tone: form.tone,
        target_words: form.target_words,
      })
      await fetchGraph()
      setSelectedNode(null)   // deselect after save
    } catch (err) {
      console.error("update node:", err)
    }
  }, [storyId, selectedNode, form, fetchGraph])

  // ---- generate ----
  // Triggers LLM generation for the selected node.
  const generate = useCallback(async () => {
    if (!selectedNode) return
    try {
      await api.generations.generate(storyId, selectedNode.id)
      alert("Generation started (async)")
    } catch (err) {
      console.error("generate:", err)
    }
  }, [storyId, selectedNode])

  // ---- Render ----
  return (
    <div style={{ display: "flex", height: "100vh" }}>
      {/* ---- React Flow Canvas ---- */}
      <div style={{ flex: 1 }}>
        {/*
          ReactFlow: the main graph canvas component.
          Props:
            nodes/edges        — data arrays
            onNodesChange      — handles node dragging, selection, deletion
            onEdgesChange      — handles edge interaction
            onConnect          — called when a new connection is created
            onNodeClick        — called when a node is clicked
            nodeTypes          — registry of custom node components
            fitView            — auto-zoom to fit all nodes
            colorMode="dark"   — dark theme
        */}
        <ReactFlow
          nodes={nodes}
          edges={edges}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          onConnect={onConnect}
          onNodeClick={onNodeClick}
          nodeTypes={nodeTypes}
          fitView
          colorMode="dark"
        >
          {/* dot grid pattern */}
          <Background color="#334155" gap={20} />
          {/* zoom in/out/fit buttons */}
          <Controls />
          {/* overview map */}
          <MiniMap
            style={{ background: "#0f172a" }}
            nodeColor="#334155"
            maskColor="rgba(0,0,0,0.6)"
          />
        </ReactFlow>
      </div>

      {/* ---- Side Panel ---- */}
      <div
        style={{
          width: 300,
          background: "#1e293b",
          borderLeft: "1px solid #334155",
          padding: 16,
          display: "flex",
          flexDirection: "column",
          gap: 12,
          color: "#e2e8f0",
          fontFamily: "system-ui, sans-serif",
          fontSize: 13,
        }}
      >
        <h3 style={{ margin: 0, fontSize: 16 }}>Story Graph</h3>

        {/* Add Scene button */}
        <button
          onClick={addNode}
          style={{
            padding: "8px 16px",
            background: "#3b82f6",
            color: "#fff",
            border: "none",
            borderRadius: 6,
            cursor: "pointer",
            fontWeight: 600,
          }}
        >
          + Add Scene
        </button>

        {/* Edit form — only shown when a node is selected */}
        {selectedNode && (
          <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
            <h4 style={{ margin: 0, fontSize: 14 }}>Edit Scene</h4>

            {/* Beat intent text input */}
            <label>
              Beat Intent
              <input
                value={form.beat_intent}
                onChange={(e) => setForm({ ...form, beat_intent: e.target.value })}
                style={inputStyle}
              />
            </label>

            {/* POV dropdown */}
            <label>
              POV
              <select
                value={form.pov}
                onChange={(e) => setForm({ ...form, pov: e.target.value })}
                style={inputStyle}
              >
                <option value="first-person">First person</option>
                <option value="third-person">Third person</option>
                <option value="omniscient">Omniscient</option>
              </select>
            </label>

            {/* Tone dropdown */}
            <label>
              Tone
              <select
                value={form.tone}
                onChange={(e) => setForm({ ...form, tone: e.target.value })}
                style={inputStyle}
              >
                <option value="neutral">Neutral</option>
                <option value="tense">Tense</option>
                <option value="melancholy">Melancholy</option>
                <option value="humorous">Humorous</option>
                <option value="dramatic">Dramatic</option>
              </select>
            </label>

            {/* Target word count number input */}
            <label>
              Target Words
              <input
                type="number"
                value={form.target_words}
                onChange={(e) => setForm({ ...form, target_words: +e.target.value })}
                // + in front of e.target.value converts the string to a number
                style={inputStyle}
              />
            </label>

            {/* Action buttons */}
            <div style={{ display: "flex", gap: 8 }}>
              <button onClick={updateNode} style={btnStyle}>
                Save
              </button>
              <button onClick={generate} style={{ ...btnStyle, background: "#f59e0b" }}>
                Generate
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

// ---- Local style objects ----
// Defined outside the component so they don't get recreated on every render.

// inputStyle: reusable style for form inputs in the side panel
const inputStyle: React.CSSProperties = {
  display: "block",
  width: "100%",
  marginTop: 4,
  padding: "6px 8px",
  background: "#0f172a",
  border: "1px solid #334155",
  borderRadius: 4,
  color: "#e2e8f0",
  fontSize: 12,
  fontFamily: "system-ui, sans-serif",
}

// btnStyle: reusable style for side panel buttons
const btnStyle: React.CSSProperties = {
  flex: 1,
  padding: "8px 12px",
  background: "#22c55e",
  color: "#000",
  border: "none",
  borderRadius: 6,
  cursor: "pointer",
  fontWeight: 600,
  fontSize: 12,
}
