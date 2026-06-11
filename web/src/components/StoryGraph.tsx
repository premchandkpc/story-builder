import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  type Node,
  type Edge,
  type Connection,
  type OnNodesChange,
  type OnEdgesChange,
  useNodesState,
  useEdgesState,
  addEdge,
  applyNodeChanges,
  applyEdgeChanges,
  type NodeTypes,
} from "@xyflow/react"
import "@xyflow/react/dist/style.css"
import SceneNode from "./SceneNode"
import { api } from "../api/client"
import type { GraphNode, GraphEdge, EdgeType } from "../api/types"

const nodeTypes: NodeTypes = { scene: SceneNode }

interface StoryGraphProps {
  storyId: string
}

function toReactFlowNodes(nodes: GraphNode[]): Node[] {
  const cols = 6
  return nodes.map((n, i) => ({
    id: n.id,
    type: "scene",
    position: {
      x: 300 * (i % cols),
      y: 200 * Math.floor(i / cols),
    },
    data: {
      label: `Node ${i + 1}`,
      status: n.status,
      beatIntent: n.beat_intent || "—",
      pov: n.pov || "—",
      tone: n.tone || "—",
      targetWords: n.target_words,
    },
  }))
}

function toReactFlowEdges(edges: GraphEdge[]): Edge[] {
  return edges.map((e, i) => ({
    id: `e-${i}`,
    source: e.from_node,
    target: e.to_node,
    label: e.edge_type === "seq" ? "" : e.edge_type,
    style: {
      stroke: e.edge_type === "fork" ? "#f59e0b" : e.edge_type === "join" ? "#8b5cf6" : "#64748b",
      strokeWidth: e.edge_type === "seq" ? 1.5 : 2.5,
      strokeDasharray: e.edge_type === "choice" ? "5 5" : undefined,
    },
    labelStyle: { fill: "#94a3b8", fontSize: 10 },
  }))
}

export default function StoryGraph({ storyId }: StoryGraphProps) {
  const [nodes, setNodes] = useNodesState([])
  const [edges, setEdges, onEdgesChange] = useEdgesState([])
  const [selectedNode, setSelectedNode] = useState<Node | null>(null)
  const [form, setForm] = useState({
    beat_intent: "",
    pov: "",
    tone: "",
    target_words: 300,
  })

  const fetchGraph = useCallback(async () => {
    try {
      const topo = await api.topology.get(storyId)
      const rfNodes = toReactFlowNodes(topo.nodes)
      const rfEdges = toReactFlowEdges(topo.edges)
      setNodes(rfNodes)
      setEdges(rfEdges)
    } catch (err) {
      console.error("fetch graph:", err)
    }
  }, [storyId, setNodes, setEdges])

  useEffect(() => {
    fetchGraph()
  }, [fetchGraph])

  const onConnect = useCallback(
    async (connection: Connection) => {
      if (!connection.source || !connection.target) return
      setEdges((eds) => addEdge({
        ...connection,
        id: `e-${Date.now()}`,
        style: { stroke: "#64748b" },
      }, eds))
      try {
        await api.edges.create(storyId, {
          from_node: connection.source,
          to_node: connection.target,
          edge_type: "seq",
        })
      } catch (err) {
        console.error("create edge:", err)
      }
    },
    [storyId, setEdges],
  )

  const onNodeClick = useCallback((_: unknown, node: Node) => {
    setSelectedNode(node)
    setForm({
      beat_intent: node.data.beatIntent || "",
      pov: node.data.pov || "",
      tone: node.data.tone || "",
      target_words: node.data.targetWords || 300,
    })
  }, [])

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
      await fetchGraph()
    } catch (err) {
      console.error("add node:", err)
    }
  }, [storyId, fetchGraph])

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
      setSelectedNode(null)
    } catch (err) {
      console.error("update node:", err)
    }
  }, [storyId, selectedNode, form, fetchGraph])

  const generate = useCallback(async () => {
    if (!selectedNode) return
    try {
      await api.generations.generate(storyId, selectedNode.id)
      alert("Generation started (async)")
    } catch (err) {
      console.error("generate:", err)
    }
  }, [storyId, selectedNode])

  return (
    <div style={{ display: "flex", height: "100vh" }}>
      <div style={{ flex: 1 }}>
        <ReactFlow
          nodes={nodes}
          edges={edges}
          onNodesChange={setNodes as OnNodesChange}
          onEdgesChange={onEdgesChange}
          onConnect={onConnect}
          onNodeClick={onNodeClick}
          nodeTypes={nodeTypes}
          fitView
          colorMode="dark"
        >
          <Background color="#334155" gap={20} />
          <Controls />
          <MiniMap
            style={{ background: "#0f172a" }}
            nodeColor="#334155"
            maskColor="rgba(0,0,0,0.6)"
          />
        </ReactFlow>
      </div>
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
        {selectedNode && (
          <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
            <h4 style={{ margin: 0, fontSize: 14 }}>Edit Scene</h4>
            <label>
              Beat Intent
              <input
                value={form.beat_intent}
                onChange={(e) => setForm({ ...form, beat_intent: e.target.value })}
                style={inputStyle}
              />
            </label>
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
            <label>
              Target Words
              <input
                type="number"
                value={form.target_words}
                onChange={(e) => setForm({ ...form, target_words: +e.target.value })}
                style={inputStyle}
              />
            </label>
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
