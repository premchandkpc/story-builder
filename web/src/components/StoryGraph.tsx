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
import GraphPanel from "./GraphPanel"
import { api } from "../api/client"
import type { GraphNode, GraphEdge, EdgeType, Generation, SceneNodeData } from "../api/types"
import { spinnerStyle } from "../api/types"
import { useToast } from "./Toast"

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

  const handleClose = useCallback(() => {
    setSelectedNode(null)
    setSelectedEdge(null)
    setConfirmingGenerate(false)
  }, [])

  const handleTabChange = useCallback((tab: "edit" | "info" | "generations" | "turns" | "agents") => {
    setActiveTab(tab)
    if (tab === "generations" && selectedNode) loadGenerations(selectedNode.id)
  }, [selectedNode, loadGenerations])

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
              className="btn-press"
              style={{
                marginTop: 8, padding: "10px 24px",
                background: "#5c4a2e", color: "#f5f0e8",
                border: "none", borderRadius: "var(--radius-md)",
                cursor: "pointer", fontWeight: 600, fontSize: 13,
                transition: "background var(--transition-base), transform var(--transition-fast)",
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
            borderRadius: "var(--radius-lg)",
            overflow: "hidden",
          }} />
          <MiniMap
            style={{
              background: "#2c2521",
              border: "1px solid #3a3a52",
              borderRadius: "var(--radius-lg)",
              overflow: "hidden",
            }}
            nodeColor="#d4a853"
            maskColor="rgba(26,26,36,0.7)"
          />
        </ReactFlow>
      </div>

      <GraphPanel
        storyId={storyId}
        nodes={nodes}
        edges={edges}
        selectedNode={selectedNode}
        selectedEdge={selectedEdge}
        onClose={handleClose}
        onAddNode={addNode}
        activeTab={activeTab}
        setActiveTab={handleTabChange}
        form={form}
        onFormChange={setForm}
        confirmingGenerate={confirmingGenerate}
        setConfirmingGenerate={setConfirmingGenerate}
        onSave={updateNode}
        onGenerate={generate}
        onDeleteNode={deleteSelectedNode}
        onDeleteEdge={deleteEdge}
        generations={generations}
        gensLoading={gensLoading}
        onAcceptGeneration={acceptGeneration}
        pendingEdgeType={pendingEdgeType}
        setPendingEdgeType={setPendingEdgeType}
      />
    </div>
  )
}
