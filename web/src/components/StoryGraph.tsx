import { useCallback, useEffect, useRef, useState } from "react"
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  type Node,
  type Edge,
  type Connection,
  type OnNodeDrag,
  useNodesState,
  useEdgesState,
  addEdge,
  type NodeTypes,
} from "@xyflow/react"

import "@xyflow/react/dist/style.css"

import SceneNode from "./SceneNode"
import TurnTimeline from "./TurnTimeline"
import AgentRunPanel from "./AgentRunPanel"
import GenerationCompare from "./GenerationCompare"
import LlmMetricsDashboard from "./LlmMetricsDashboard"
import { api } from "../api/client"
import type { GraphNode, GraphEdge, EdgeType, Generation } from "../api/types"
import { spinnerStyle, slideUpStyle, labelStyle, ghostBtnStyle, destructiveBtnStyle } from "../api/types"
import { useToast } from "./Toast"

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
  return edges.map((e) => ({
    id: e.id,
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

const addSvg = (
  <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
    <path d="M8 3v10M3 8h10" />
  </svg>
)

const closeSvg = (
  <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round">
    <path d="M4 4l8 8M12 4l-8 8" />
  </svg>
)

const trashSvg = (
  <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round">
    <path d="M2 4h12M5 4V2.5A.5.5 0 015.5 2h5a.5.5 0 01.5.5V4M13 4v9.5a1.5 1.5 0 01-1.5 1.5h-7A1.5 1.5 0 013 13.5V4" />
  </svg>
)

const edgeTypes: { value: EdgeType; label: string; desc: string }[] = [
  { value: "seq", label: "Seq", desc: "Default progression" },
  { value: "fork", label: "Fork", desc: "Branch to multiple paths" },
  { value: "join", label: "Join", desc: "Converge branches" },
  { value: "choice", label: "Choice", desc: "Decision point" },
]

const panelInputStyle: React.CSSProperties = {
  display: "block",
  width: "100%",
  marginTop: 4,
  padding: "7px 10px",
  background: "var(--bg)",
  border: "1px solid var(--border)",
  borderRadius: 4,
  color: "var(--text)",
  fontSize: 12,
  fontFamily: "var(--font-body)",
  boxSizing: "border-box",
  transition: "border-color 0.15s, box-shadow 0.15s",
}

const panelBtnStyle: React.CSSProperties = {
  flex: 1,
  padding: "8px 14px",
  background: "var(--success)",
  color: "#1a1a24",
  border: "none",
  borderRadius: 6,
  cursor: "pointer",
  fontWeight: 600,
  fontSize: 12,
  transition: "background 0.15s",
}

const tabBtnStyle = (active: boolean): React.CSSProperties => ({
  flex: 1,
  padding: "8px 0",
  background: "none",
  border: "none",
  borderBottom: active ? "2px solid var(--accent)" : "2px solid transparent",
  color: active ? "var(--accent)" : "var(--text-dim)",
  cursor: "pointer",
  fontWeight: active ? 600 : 400,
  fontSize: 11,
  textTransform: "capitalize",
  transition: "color 0.15s, border-color 0.15s",
})

export default function StoryGraph({ storyId }: StoryGraphProps) {
  const { toast, error: showError } = useToast()

  const [nodes, setNodes, onNodesChange] = useNodesState([] as Node<SceneNodeData>[])
  const [edges, setEdges, onEdgesChange] = useEdgesState([] as Edge[])
  const [selectedNode, setSelectedNode] = useState<Node<SceneNodeData> | null>(null)
  const [selectedEdge, setSelectedEdge] = useState<Edge | null>(null)
  const [loading, setLoading] = useState(true)
  const [activeTab, setActiveTab] = useState<"edit" | "info" | "generations" | "turns" | "agents">("edit")
  const [generations, setGenerations] = useState<Generation[]>([])
  const [gensLoading, setGensLoading] = useState(false)
  const [expandedGen, setExpandedGen] = useState<string | null>(null)
  const [showCompare, setShowCompare] = useState(false)
  const [pendingEdgeType, setPendingEdgeType] = useState<EdgeType>(
    () => (localStorage.getItem("edgeType") as EdgeType) || "seq",
  )
  const [confirmingGenerate, setConfirmingGenerate] = useState(false)

  const [form, setForm] = useState({
    beat_intent: "",
    pov: "",
    tone: "",
    target_words: 300,
  })

  const loadGraphData = useCallback(async () => {
    return await api.topology.get(storyId)
  }, [storyId])

  useEffect(() => {
    let cancelled = false
    loadGraphData()
      .then((topo) => {
        if (cancelled) return
        setNodes(toReactFlowNodes(topo.nodes || []))
        setEdges(toReactFlowEdges(topo.edges || []))
      })
      .catch((err) => {
        showError("Failed to load graph")
        console.error("fetch graph:", err)
      })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [loadGraphData, setNodes, setEdges, showError])

  useEffect(() => {
    localStorage.setItem("edgeType", pendingEdgeType)
  }, [pendingEdgeType])

  const fetchGraph = useCallback(async () => {
    const topo = await loadGraphData()
    setNodes(toReactFlowNodes(topo.nodes || []))
    setEdges(toReactFlowEdges(topo.edges || []))
  }, [loadGraphData, setNodes, setEdges])

  const deleteSelectedNode = useCallback(async () => {
    if (!selectedNode) return
    try {
      await api.nodes.delete(storyId, selectedNode.id)
      toast("Scene deleted", "success")
      setSelectedNode(null)
      await fetchGraph()
    } catch (err) {
      showError("Failed to delete scene")
      console.error("delete node:", err)
    }
  }, [storyId, selectedNode, fetchGraph, toast, showError])

  const deleteFnRef = useRef<() => Promise<void>>(deleteSelectedNode)
  useEffect(() => {
    deleteFnRef.current = deleteSelectedNode
  })
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape") {
        setSelectedNode(null)
        setSelectedEdge(null)
        setConfirmingGenerate(false)
        setExpandedGen(null)
      }
      if ((e.key === "Delete" || e.key === "Backspace") && document.activeElement?.tagName !== "INPUT" && document.activeElement?.tagName !== "TEXTAREA") {
        deleteFnRef.current()
      }
    }
    window.addEventListener("keydown", handleKeyDown)
    return () => window.removeEventListener("keydown", handleKeyDown)
  }, [])

  const onConnect = useCallback(
    async (connection: Connection) => {
      if (!connection.source || !connection.target) return
      const edgeType = pendingEdgeType
      try {
        const created = await api.edges.create(storyId, {
          from_node: connection.source,
          to_node: connection.target,
          edge_type: edgeType,
        })
        setEdges((eds: Edge[]) => addEdge({
          ...connection,
          id: created.id,
          label: edgeType === "seq" ? "" : edgeType,
          style: {
            stroke: edgeType === "fork" ? "#c9734a" : edgeType === "join" ? "#d4a853" : "#8888a0",
            strokeWidth: edgeType === "seq" ? 1.5 : 2.5,
            strokeDasharray: edgeType === "choice" ? "5 5" : undefined,
          },
          labelStyle: { fill: "#8888a0", fontSize: 10 },
        }, eds))
        toast("Edge created", "success")
      } catch (err) {
        showError("Failed to create edge")
        console.error("create edge:", err)
      }
    },
    [storyId, setEdges, pendingEdgeType, toast, showError],
  )

  const loadGenerations = useCallback(async (nodeId: string) => {
    setGensLoading(true)
    try {
      const gens = await api.generations.list(storyId, nodeId)
      setGenerations(gens)
    } catch {
      setGenerations([])
    } finally {
      setGensLoading(false)
    }
  }, [storyId])

  const acceptGeneration = useCallback(async (nodeId: string, genId: string) => {
    try {
      await api.generations.accept(storyId, nodeId, genId)
      toast("Generation accepted", "success")
      await loadGenerations(nodeId)
      await fetchGraph()
    } catch (err) {
      showError("Failed to accept generation")
      console.error("accept:", err)
    }
  }, [storyId, loadGenerations, fetchGraph, toast, showError])

  const generate = useCallback(async () => {
    if (!selectedNode) return
    setConfirmingGenerate(false)
    try {
      await api.generations.generate(storyId, selectedNode.id)
      toast("Generation started (async)", "success")
      setTimeout(() => loadGenerations(selectedNode.id), 1500)
      await fetchGraph()
    } catch (err) {
      showError("Failed to start generation")
      console.error("generate:", err)
    }
  }, [storyId, selectedNode, loadGenerations, fetchGraph, toast, showError])

  const onNodeClick = useCallback((_: unknown, node: Node) => {
    setSelectedEdge(null)
    const d = node.data as unknown as SceneNodeData
    setSelectedNode(node as unknown as Node<SceneNodeData>)
    setActiveTab("edit")
    setConfirmingGenerate(false)
    setForm({
      beat_intent: d.beatIntent || "",
      pov: d.pov || "",
      tone: d.tone || "",
      target_words: d.targetWords || 300,
    })
    loadGenerations(node.id)
  }, [loadGenerations])

  const onEdgeClick = useCallback((_: unknown, edge: Edge) => {
    setSelectedNode(null)
    setSelectedEdge(edge)
    setActiveTab("edit")
  }, [])

  const onNodeDragEnd: OnNodeDrag = useCallback(
    async (_event: any, node: Node) => {
      try {
        await api.nodes.updatePosition(storyId, node.id, node.position.x, node.position.y)
      } catch (err) {
        console.error("persist position:", err)
      }
    },
    [storyId],
  )

  const onPaneClick = useCallback(() => {
    setSelectedNode(null)
    setSelectedEdge(null)
    setConfirmingGenerate(false)
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
      toast("Scene added", "success")
      await fetchGraph()
    } catch (err) {
      showError("Failed to add scene")
      console.error("add node:", err)
    }
  }, [storyId, fetchGraph, toast, showError])

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
      toast("Scene saved", "success")
      await fetchGraph()
      setSelectedNode(null)
    } catch (err) {
      showError("Failed to save scene")
      console.error("update node:", err)
    }
  }, [storyId, selectedNode, form, fetchGraph, toast, showError])

  const deleteEdge = useCallback(async () => {
    if (!selectedEdge) return
    try {
      await api.edges.deleteById(storyId, selectedEdge.id)
      toast("Edge deleted", "success")
      setSelectedEdge(null)
      await fetchGraph()
    } catch (err) {
      showError("Failed to delete edge")
      console.error("delete edge:", err)
    }
  }, [storyId, selectedEdge, fetchGraph, toast, showError])

  const selectedNodeData = selectedNode?.data as SceneNodeData | undefined

  const renderPanelContent = () => {
    if (selectedNode && activeTab === "edit") {
      return (
        <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
          <div>
            <label style={labelStyle}>Beat Intent</label>
            <input
              value={form.beat_intent}
              onChange={(e) => setForm({ ...form, beat_intent: e.target.value })}
              style={panelInputStyle}
              placeholder="Narrative purpose of this scene"
            />
          </div>

          <div>
            <label style={labelStyle}>POV</label>
            <select
              value={form.pov}
              onChange={(e) => setForm({ ...form, pov: e.target.value })}
              style={panelInputStyle}
            >
              <option value="first-person">First person</option>
              <option value="third-person">Third person</option>
              <option value="omniscient">Omniscient</option>
            </select>
          </div>

          <div>
            <label style={labelStyle}>Tone</label>
            <select
              value={form.tone}
              onChange={(e) => setForm({ ...form, tone: e.target.value })}
              style={panelInputStyle}
            >
              <option value="neutral">Neutral</option>
              <option value="tense">Tense</option>
              <option value="melancholy">Melancholy</option>
              <option value="humorous">Humorous</option>
              <option value="dramatic">Dramatic</option>
            </select>
          </div>

          <div>
            <label style={labelStyle}>Target Words</label>
            <input
              type="number"
              value={form.target_words}
              onChange={(e) => setForm({ ...form, target_words: +e.target.value })}
              style={panelInputStyle}
              min={50}
              max={5000}
            />
          </div>

          <div style={{
            display: "flex", gap: 8, marginTop: 8,
            paddingTop: 12, borderTop: "1px solid var(--border)",
          }}>
            <button
              onClick={updateNode}
              style={{ ...panelBtnStyle, flex: 2 }}
              onMouseEnter={(e) => e.currentTarget.style.background = "#7ba06c"}
              onMouseLeave={(e) => e.currentTarget.style.background = "var(--success)"}
            >
              Save
            </button>
            {confirmingGenerate ? (
              <div style={{ display: "flex", gap: 4, flex: 3 }}>
                <button
                  onClick={generate}
                  style={{ ...panelBtnStyle, background: "var(--error)", flex: 1 }}
                  onMouseEnter={(e) => e.currentTarget.style.background = "#c06c60"}
                  onMouseLeave={(e) => e.currentTarget.style.background = "var(--error)"}
                >
                  Confirm
                </button>
                <button
                  onClick={() => setConfirmingGenerate(false)}
                  style={{ ...panelBtnStyle, background: "var(--text-muted)", flex: 1 }}
                >
                  Cancel
                </button>
              </div>
            ) : (
              <button
                onClick={() => setConfirmingGenerate(true)}
                style={{ ...panelBtnStyle, background: "#c9734a", flex: 3 }}
                onMouseEnter={(e) => e.currentTarget.style.background = "#d9865f"}
                onMouseLeave={(e) => e.currentTarget.style.background = "#c9734a"}
              >
                Generate
              </button>
            )}
          </div>

          <div style={{
            display: "flex", gap: 4, marginTop: 4,
          }}>
            <button
              onClick={() => setSelectedNode(null)}
              style={{
                ...ghostBtnStyle, flex: 1, justifyContent: "center",
                border: "1px solid var(--border)", fontSize: 12,
              }}
            >
              {closeSvg} Close
            </button>
            <button
              onClick={deleteSelectedNode}
              style={{ ...destructiveBtnStyle, flex: 1, justifyContent: "center" }}
              onMouseEnter={(e) => { e.currentTarget.style.background = "rgba(212,103,103,0.12)" }}
              onMouseLeave={(e) => { e.currentTarget.style.background = "transparent" }}
              title="Delete scene (Delete key)"
            >
              {trashSvg} Delete
            </button>
          </div>
        </div>
      )
    }

    if (selectedNode && activeTab === "generations") {
      return (
        <div style={{ display: "flex", flexDirection: "column", gap: 8, fontSize: 12 }}>
          <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 4 }}>
            <div style={{ color: "var(--text-dim)", fontWeight: 500 }}>Generations</div>
            {generations.filter(g => g.output).length >= 2 && (
              <button
                onClick={() => setShowCompare(!showCompare)}
                style={{
                  background: showCompare ? "var(--accent)" : "transparent",
                  border: `1px solid ${showCompare ? "var(--accent)" : "var(--border)"}`,
                  borderRadius: 4, padding: "3px 10px",
                  color: showCompare ? "#1a1a24" : "var(--text-dim)",
                  cursor: "pointer", fontSize: 10, fontWeight: 600,
                  transition: "all 0.15s",
                }}
              >
                {showCompare ? "List" : "Compare"}
              </button>
            )}
          </div>

          {showCompare ? (
            <GenerationCompare generations={generations} />
          ) : (
            <>
              {gensLoading && (
                <div style={{ display: "flex", alignItems: "center", gap: 8, color: "var(--text-dim)", padding: 8 }}>
                  <div style={spinnerStyle} />
                  <span style={{ fontSize: 12 }}>Loading generations...</span>
                </div>
              )}
              {!gensLoading && generations.length === 0 && (
                <div style={{
                  color: "var(--text-dim)", fontStyle: "italic",
                  padding: 12, textAlign: "center", fontSize: 11,
                }}>
                  No generations yet. Click <strong style={{ color: "var(--accent)" }}>Generate</strong> to start.
                </div>
              )}
              {!gensLoading && generations.map((g) => {
                const isAccepted = g.accepted
                const isExpanded = expandedGen === g.id
                return (
                  <div key={g.id} style={{
                    border: `1px solid ${isAccepted ? "var(--accent)" : "var(--border)"}`,
                    borderRadius: 6, padding: 10,
                    background: isAccepted ? "rgba(212,168,83,0.06)" : "transparent",
                    transition: "border-color 0.15s",
                  }}>
                    <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 4 }}>
                      <span style={{ fontWeight: 600, fontSize: 11, color: isAccepted ? "var(--accent)" : "var(--text)" }}>
                        {g.model || "unknown"}
                      </span>
                      <span style={{ fontSize: 10, color: "var(--text-dim)" }}>
                        {new Date(g.created_at).toLocaleString()}
                      </span>
                    </div>
                    <div style={{ display: "flex", gap: 6, alignItems: "center" }}>
                      <span style={{
                        fontSize: 10, padding: "2px 7px", borderRadius: 3,
                        background: isAccepted ? "var(--accent)" : g.output ? "rgba(201,115,74,0.15)" : "rgba(140,126,112,0.15)",
                        color: isAccepted ? "#1a1a24" : g.output ? "#c9734a" : "var(--text-dim)",
                        fontWeight: 600,
                      }}>
                        {isAccepted ? "Accepted" : g.output ? "Generated" : "Pending"}
                      </span>
                      {g.output && (
                        <button
                          onClick={() => setExpandedGen(isExpanded ? null : g.id)}
                          style={{
                            background: "none", border: "none", color: "var(--accent)",
                            cursor: "pointer", fontSize: 10, padding: 0,
                          }}
                        >
                          {isExpanded ? "Collapse" : "Preview"}
                        </button>
                      )}
                    </div>
                    {isExpanded && g.output && (
                      <div style={{
                        marginTop: 8, padding: 10,
                        background: "var(--bg)", borderRadius: 4,
                        fontSize: 11, lineHeight: 1.6,
                        maxHeight: 250, overflowY: "auto",
                        whiteSpace: "pre-wrap", color: "var(--text)",
                        border: "1px solid var(--border)",
                      }}>
                        {g.output}
                      </div>
                    )}
                    {g.output && !isAccepted && (
                      <button
                        onClick={() => acceptGeneration(selectedNode.id, g.id)}
                        style={{
                          marginTop: 8, padding: "5px 10px",
                          background: "var(--accent)", color: "#1a1a24",
                          border: "none", borderRadius: 4,
                          cursor: "pointer", fontWeight: 600, fontSize: 11,
                          width: "100%", transition: "background 0.15s",
                        }}
                        onMouseEnter={(e) => e.currentTarget.style.background = "var(--accent-hover)"}
                        onMouseLeave={(e) => e.currentTarget.style.background = "var(--accent)"}
                      >
                        Accept
                      </button>
                    )}
                  </div>
                )
              })}
            </>
          )}
        </div>
      )
    }

    if (selectedNode && activeTab === "info") {
      const inEdges = edges.filter((e) => e.target === selectedNode.id)
      const outEdges = edges.filter((e) => e.source === selectedNode.id)
      return (
        <div style={{ display: "flex", flexDirection: "column", gap: 8, fontSize: 12 }}>
          <div>
            <div style={{ color: "var(--text-dim)", fontSize: 11, marginBottom: 2 }}>Status</div>
            <div style={{
              fontWeight: 600, textTransform: "capitalize",
              display: "inline-flex", alignItems: "center", gap: 6,
              padding: "2px 8px", borderRadius: 4,
              background: "rgba(212,168,83,0.08)",
              color: "var(--accent)",
              fontSize: 12,
            }}>
              <span style={{ width: 5, height: 5, borderRadius: "50%", background: "var(--accent)", flexShrink: 0 }} />
              {selectedNodeData?.status || "Unknown"}
            </div>
          </div>

          <div>
            <div style={{ color: "var(--text-dim)", fontSize: 11, marginBottom: 2 }}>Beat Intent</div>
            <div style={{ color: "var(--text)", lineHeight: 1.4 }}>
              {selectedNodeData?.beatIntent || "—"}
            </div>
          </div>

          <div>
            <div style={{ color: "var(--text-dim)", fontSize: 11, marginBottom: 4 }}>Edges</div>
            <div style={{
              display: "flex", gap: 8, alignItems: "center",
              padding: "6px 10px", background: "var(--bg)",
              borderRadius: 4, fontSize: 12, color: "var(--text)",
            }}>
              <span>Incoming: {inEdges.length}</span>
              {inEdges.length > 0 && (
                <span style={{ fontSize: 10, color: "var(--text-dim)" }}>
                  ({inEdges.map((e) => e.label || "seq").join(", ")})
                </span>
              )}
            </div>
            <div style={{
              display: "flex", gap: 8, alignItems: "center",
              padding: "6px 10px", background: "var(--bg)",
              borderRadius: 4, fontSize: 12, color: "var(--text)", marginTop: 4,
            }}>
              <span>Outgoing: {outEdges.length}</span>
              {outEdges.length > 0 && (
                <span style={{ fontSize: 10, color: "var(--text-dim)" }}>
                  ({outEdges.map((e) => e.label || "seq").join(", ")})
                </span>
              )}
            </div>
          </div>

          <button
            onClick={deleteSelectedNode}
            style={{ ...destructiveBtnStyle, marginTop: 8, justifyContent: "center" }}
            onMouseEnter={(e) => { e.currentTarget.style.background = "rgba(212,103,103,0.12)" }}
            onMouseLeave={(e) => { e.currentTarget.style.background = "transparent" }}
          >
            {trashSvg} Delete Scene
          </button>
        </div>
      )
    }

    if (selectedNode && activeTab === "turns") {
      return (
        <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
          <div style={{ color: "var(--text-dim)", fontSize: 11, marginBottom: 4 }}>Turn Timeline</div>
          <TurnTimeline storyId={storyId} nodeId={selectedNode.id} compact />
        </div>
      )
    }

    if (selectedNode && activeTab === "agents") {
      return (
        <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
          <div style={{ color: "var(--text-dim)", fontSize: 11, marginBottom: 4 }}>Agent Runs</div>
          <AgentRunPanel storyId={storyId} nodeId={selectedNode.id} />
        </div>
      )
    }

    if (selectedEdge) {
      const edgeLabel = selectedEdge.label || "seq"
      return (
        <div style={{ display: "flex", flexDirection: "column", gap: 8, fontSize: 12 }}>
          <div>
            <div style={{ color: "var(--text-dim)", fontSize: 11, marginBottom: 4 }}>Edge Details</div>
            <div style={{
              padding: "6px 10px", background: "var(--bg)",
              borderRadius: 4, display: "flex", gap: 6, alignItems: "center",
            }}>
              <span style={{ color: "var(--text-dim)" }}>Type:</span>
              <span style={{
                color: "var(--accent)", fontWeight: 600, textTransform: "uppercase",
                padding: "1px 6px", borderRadius: 3,
                background: "rgba(212,168,83,0.1)", fontSize: 10,
              }}>
                {edgeLabel}
              </span>
            </div>
          </div>

          <div style={{
            padding: "6px 10px", background: "var(--bg)", borderRadius: 4,
            fontSize: 11, color: "var(--text)", wordBreak: "break-all",
          }}>
            <span style={{ color: "var(--text-dim)" }}>From:</span> Node {selectedEdge.source.slice(-4)}
          </div>
          <div style={{
            padding: "6px 10px", background: "var(--bg)", borderRadius: 4,
            fontSize: 11, color: "var(--text)", wordBreak: "break-all",
          }}>
            <span style={{ color: "var(--text-dim)" }}>To:</span> Node {selectedEdge.target.slice(-4)}
          </div>

          <button
            onClick={deleteEdge}
            style={{ ...destructiveBtnStyle, marginTop: 4, justifyContent: "center" }}
            onMouseEnter={(e) => { e.currentTarget.style.background = "rgba(212,103,103,0.12)" }}
            onMouseLeave={(e) => { e.currentTarget.style.background = "transparent" }}
          >
            {trashSvg} Delete Edge
          </button>
        </div>
      )
    }

    return (
      <div style={{ display: "flex", flexDirection: "column", gap: 4, flex: 1 }}>
        <div style={{ flex: 1, overflowY: "auto" }}>
          <LlmMetricsDashboard storyId={storyId} />
        </div>
        <div style={{
          padding: "12px 16px", textAlign: "center", color: "var(--text-dim)", fontSize: 12,
          borderTop: "1px solid var(--border)",
          lineHeight: 1.6,
        }}>
          Click a node to edit<br />
          <span style={{ fontSize: 11, color: "var(--text-dim)" }}>
            Esc to deselect · Delete to remove
          </span>
        </div>
      </div>
    )
  }

  return (
    <div style={{ display: "flex", height: "calc(100vh - 49px)", position: "relative" }}>
      <div style={{ flex: 1, position: "relative" }}>
        {loading && (
          <div style={{
            position: "absolute", inset: 0, zIndex: 10,
            display: "flex", alignItems: "center", justifyContent: "center",
            background: "rgba(196,164,108,0.4)",
            backdropFilter: "blur(2px)",
          }}>
            <div style={spinnerStyle} />
          </div>
        )}

        {!loading && nodes.length === 0 && (
          <div style={{
            position: "absolute", inset: 0, zIndex: 5,
            display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center",
            gap: 12, color: "#5c4a2e",
          }}>
            <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1" strokeLinecap="round" style={{ opacity: 0.35 }}>
              <circle cx="12" cy="12" r="10" />
              <path d="M12 8v4M12 16h.01" />
            </svg>
            <span style={{ fontSize: 16, fontWeight: 700, fontFamily: "var(--font-heading)" }}>No scenes yet</span>
            <span style={{ fontSize: 13, opacity: 0.6 }}>Add a scene to start building your story DAG</span>
            <button
              onClick={addNode}
              style={{
                marginTop: 8, padding: "10px 24px",
                background: "#5c4a2e", color: "#f5f0e8",
                border: "none", borderRadius: 6,
                cursor: "pointer", fontWeight: 600, fontSize: 13,
                transition: "background 0.15s",
              }}
              onMouseEnter={(e) => e.currentTarget.style.background = "#6d5940"}
              onMouseLeave={(e) => e.currentTarget.style.background = "#5c4a2e"}
            >
              + Add First Scene
            </button>
          </div>
        )}

        <ReactFlow
          nodes={nodes}
          edges={edges}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          onConnect={onConnect}
          onNodeClick={onNodeClick}
          onNodeDragStop={onNodeDragEnd}
          onEdgeClick={onEdgeClick}
          onPaneClick={onPaneClick}
          nodeTypes={nodeTypes}
          fitView
          colorMode="dark"
          style={{ background: "#c4a46c" }}
        >
          <Background color="#b89560" gap={30} size={2} />
          <Controls style={{
            background: "#2c2521",
            color: "#e8e4d8",
            border: "1px solid #3a3a52",
            borderRadius: 8,
            overflow: "hidden",
          }} />
          <MiniMap
            style={{
              background: "#2c2521",
              border: "1px solid #3a3a52",
              borderRadius: 8,
              overflow: "hidden",
            }}
            nodeColor="#d4a853"
            maskColor="rgba(26,26,36,0.7)"
          />
        </ReactFlow>
      </div>

      <div
        style={{
          ...slideUpStyle,
          width: 300,
          background: "var(--surface)",
          borderLeft: "1px solid var(--border)",
          padding: "16px 0",
          display: "flex",
          flexDirection: "column",
          color: "var(--text)",
          fontFamily: "var(--font-body)",
          fontSize: 13,
          flexShrink: 0,
        }}
      >
        <div style={{ padding: "0 16px 12px", display: "flex", alignItems: "center", justifyContent: "space-between" }}>
          <h3 style={{
            margin: 0,
            fontSize: 16,
            fontFamily: "var(--font-heading)",
            color: "var(--accent)",
          }}>
            {selectedNode ? "Scene Editor" : selectedEdge ? "Edge Details" : "Story Graph"}
          </h3>
          {(selectedNode || selectedEdge) && (
            <button
              onClick={() => { setSelectedNode(null); setSelectedEdge(null); setConfirmingGenerate(false) }}
              style={{
                background: "none", border: "none", color: "var(--text-dim)",
                cursor: "pointer", padding: 4, display: "flex",
                transition: "color 0.15s",
              }}
              onMouseEnter={(e) => e.currentTarget.style.color = "var(--text)"}
              onMouseLeave={(e) => e.currentTarget.style.color = "var(--text-dim)"}
            >
              {closeSvg}
            </button>
          )}
        </div>

        <div style={{ padding: "0 16px", marginBottom: 12 }}>
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
              transition: "background 0.15s, box-shadow 0.15s",
              display: "flex",
              alignItems: "center",
              gap: 6,
              width: "100%",
              justifyContent: "center",
            }}
            onMouseEnter={(e) => {
              e.currentTarget.style.background = "var(--accent-hover)"
              e.currentTarget.style.boxShadow = "0 2px 8px rgba(212,168,83,0.3)"
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.background = "var(--accent)"
              e.currentTarget.style.boxShadow = "none"
            }}
          >
            {addSvg}
            Add Scene
          </button>
        </div>

        {(selectedNode || selectedEdge) && (
          <div style={{ display: "flex", borderBottom: "1px solid var(--border)", padding: "0 16px" }}>
            {selectedNode && (["edit", "info", "generations", "turns", "agents"] as const).map((tab) => (
              <button
                key={tab}
                onClick={() => {
                  setActiveTab(tab)
                  if (tab === "generations") loadGenerations(selectedNode!.id)
                  if (tab === "turns" || tab === "agents") setShowCompare(false)
                }}
                style={tabBtnStyle(activeTab === tab)}
                onMouseEnter={(e) => {
                  if (activeTab !== tab) e.currentTarget.style.color = "var(--text)"
                }}
                onMouseLeave={(e) => {
                  if (activeTab !== tab) e.currentTarget.style.color = "var(--text-dim)"
                }}
              >
                {tab === "edit" ? "Edit" : tab === "info" ? "Info" : tab === "generations" ? "Gen" : tab === "turns" ? "Turns" : "Agents"}
              </button>
            ))}
          </div>
        )}

        <div style={{
          flex: 1, overflowY: "auto", padding: "14px 16px",
          display: "flex", flexDirection: "column", gap: 8,
        }}>
          {renderPanelContent()}
        </div>

        {nodes.length > 0 && (
          <div style={{
            borderTop: "1px solid var(--border)", padding: "10px 16px",
            fontSize: 11, color: "var(--text-dim)",
          }}>
            <div style={{ display: "flex", gap: 12, alignItems: "center", flexWrap: "wrap" }}>
              <span>{nodes.length} nodes · {edges.length} edges</span>
              <span style={{ display: "flex", gap: 4 }}>
                {edgeTypes.map((et) => (
                  <button
                    key={et.value}
                    onClick={() => setPendingEdgeType(et.value)}
                    style={{
                      padding: "3px 8px",
                      background: pendingEdgeType === et.value ? "var(--accent-dim)" : "transparent",
                      border: `1px solid ${pendingEdgeType === et.value ? "var(--accent)" : "var(--border)"}`,
                      borderRadius: 4,
                      color: pendingEdgeType === et.value ? "var(--accent)" : "var(--text-dim)",
                      cursor: "pointer",
                      fontSize: 10,
                      fontWeight: pendingEdgeType === et.value ? 600 : 400,
                      transition: "all 0.1s",
                      textTransform: "uppercase",
                      letterSpacing: "0.03em",
                    }}
                    title={et.desc}
                  >
                    {et.label}
                  </button>
                ))}
              </span>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
