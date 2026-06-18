import { useCallback, useEffect, useState } from "react"
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  type Node,
  type Edge,
  type Connection,
  useNodesState,
  useEdgesState,
  addEdge,
  type NodeTypes,
} from "@xyflow/react"

import "@xyflow/react/dist/style.css"

import SceneNode from "./SceneNode"
import { api } from "../api/client"
import type { GraphNode, GraphEdge } from "../api/types"

interface SceneNodeData extends Record<string, unknown> {
  label: string
  status: string
  beatIntent: string
  pov: string
  tone: string
  targetWords: number
}

const nodeTypes: NodeTypes = { scene: SceneNode }

interface StoryGraphProps {
  storyId: string
}

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

function toReactFlowEdges(edges: GraphEdge[]): Edge[] {
  return edges.map((e, i) => ({
    id: `e-${i}`,
    source: e.from_node,
    target: e.to_node,
    label: e.edge_type === "seq" ? "" : e.edge_type,
    style: {
      stroke: e.edge_type === "fork"  ? "#c9734a" :
              e.edge_type === "join"  ? "#d4a853" :
                                        "#8888a0",
      strokeWidth: e.edge_type === "seq" ? 1.5 : 2.5,
      strokeDasharray: e.edge_type === "choice" ? "5 5" : undefined,
    },
    labelStyle: { fill: "#8888a0", fontSize: 10 },
  }))
}

export default function StoryGraph({ storyId }: StoryGraphProps) {
  const [nodes, setNodes, onNodesChange] = useNodesState([] as Node<SceneNodeData>[])
  const [edges, setEdges, onEdgesChange] = useEdgesState([] as Edge[])
  const [selectedNode, setSelectedNode] = useState<Node<SceneNodeData> | null>(null)

  const [form, setForm] = useState({
    beat_intent: "",
    pov: "",
    tone: "",
    target_words: 300,
  })

  const fetchGraph = useCallback(async () => {
    try {
      const topo = await api.topology.get(storyId)
      setNodes((prev) => toReactFlowNodes(topo.nodes || [], prev))
      setEdges(toReactFlowEdges(topo.edges || []))
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
      setEdges((eds: Edge[]) => addEdge({
        ...connection,
        id: `e-${Date.now()}`,
        style: { stroke: "#8888a0" },
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
    const d = node.data as unknown as SceneNodeData
    setSelectedNode(node as unknown as Node<SceneNodeData>)
    setForm({
      beat_intent: d.beatIntent || "",
      pov: d.pov || "",
      tone: d.tone || "",
      target_words: d.targetWords || 300,
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
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          onConnect={onConnect}
          onNodeClick={onNodeClick}
          nodeTypes={nodeTypes}
          fitView
          colorMode="dark"
          style={{ background: "#c4a46c" }}
        >
          <Background color="#b89560" gap={30} size={2} />
          <Controls style={{ background: "#24243a", color: "#e8e4d8", border: "1px solid #3a3a52" }} />
          <MiniMap
            style={{ background: "#24243a", border: "1px solid #3a3a52" }}
            nodeColor="#d4a853"
            maskColor="rgba(26,26,36,0.7)"
          />
        </ReactFlow>
      </div>

      <div
        style={{
          width: 300,
          background: "var(--surface)",
          borderLeft: "1px solid var(--border)",
          padding: 16,
          display: "flex",
          flexDirection: "column",
          gap: 12,
          color: "var(--text)",
          fontFamily: "var(--font-body)",
          fontSize: 13,
        }}
      >
        <h3 style={{
          margin: 0,
          fontSize: 16,
          fontFamily: "var(--font-heading)",
          color: "var(--accent)",
        }}>
          Story Graph
        </h3>

        <button
          onClick={addNode}
          style={{
            padding: "8px 16px",
            background: "var(--accent)",
            color: "#1a1a24",
            border: "none",
            borderRadius: 6,
            cursor: "pointer",
            fontWeight: 600,
            fontSize: 13,
            transition: "background 0.1s",
          }}
          onMouseEnter={(e) => { e.currentTarget.style.background = "var(--accent-hover)" }}
          onMouseLeave={(e) => { e.currentTarget.style.background = "var(--accent)" }}
        >
          + Add Scene
        </button>

        {selectedNode && (
          <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
            <h4 style={{
              margin: 0,
              fontSize: 14,
              fontFamily: "var(--font-heading)",
              color: "var(--text)",
            }}>
              Edit Scene
            </h4>

            <label style={{ fontSize: 12, color: "var(--text-muted)" }}>
              Beat Intent
              <input
                value={form.beat_intent}
                onChange={(e) => setForm({ ...form, beat_intent: e.target.value })}
                style={inputStyle}
              />
            </label>

            <label style={{ fontSize: 12, color: "var(--text-muted)" }}>
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

            <label style={{ fontSize: 12, color: "var(--text-muted)" }}>
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

            <label style={{ fontSize: 12, color: "var(--text-muted)" }}>
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
              <button onClick={generate} style={{ ...btnStyle, background: "#c9734a" }}>
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
  background: "var(--bg)",
  border: "1px solid var(--border)",
  borderRadius: 4,
  color: "var(--text)",
  fontSize: 12,
  fontFamily: "var(--font-body)",
}

const btnStyle: React.CSSProperties = {
  flex: 1,
  padding: "8px 12px",
  background: "var(--success)",
  color: "#1a1a24",
  border: "none",
  borderRadius: 6,
  cursor: "pointer",
  fontWeight: 600,
  fontSize: 12,
}