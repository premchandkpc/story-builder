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
import {
  useTopology,
  useGenerationStatusPolling,
  useCreateNode,
  useUpdateNode,
  useDeleteNode,
  useCreateEdge,
  useDeleteEdge,
  useUpdateNodePosition,
} from "../api/hooks"
import type { GraphNode, GraphEdge, EdgeType, SceneNodeData } from "../api/types"
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
          label: n.title || n.beat_intent || `Node ${n.id.slice(-4)}`,
          title: n.title || "",
          status: n.status,
          beatIntent: n.beat_intent || "",
          pov: n.pov || "",
          tone: n.tone || "",
          targetWords: n.target_words,
          characterRefs: n.character_refs || [],
          wordCount: 0,
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
        label: n.title || n.beat_intent || `Node ${n.id.slice(-4)}`,
        title: n.title || "",
        status: n.status,
        beatIntent: n.beat_intent || "",
        pov: n.pov || "",
        tone: n.tone || "",
        targetWords: n.target_words,
        characterRefs: n.character_refs || [],
        wordCount: 0,
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
  const [activeTab, setActiveTab] = useState<"edit" | "info" | "generations" | "turns" | "agents">("edit")
  const { generations, isLoading: gensLoading, refetch: refetchGens, hasPending: gensPending, isError: gensError }
    = useGenerationStatusPolling(storyId, selectedNode?.id || null, !!selectedNode)
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

  // Mutation hooks
  const createNodeMutation = useCreateNode(storyId)
  const updateNodeMutation = useUpdateNode(storyId)
  const deleteNodeMutation = useDeleteNode(storyId)
  const createEdgeMutation = useCreateEdge(storyId)
  const deleteEdgeMutation = useDeleteEdge(storyId)
  const updatePositionMutation = useUpdateNodePosition(storyId)

  const topo = useTopology(storyId)

  useEffect(() => {
    if (!topo.data) return
    setNodes(toReactFlowNodes(topo.data.nodes || []))
    setEdges(toReactFlowEdges(topo.data.edges || []))
  }, [topo.data, setNodes, setEdges])

  useEffect(() => {
    localStorage.setItem("edgeType", pendingEdgeType)
  }, [pendingEdgeType])

  const deleteSelectedNode = useCallback(async () => {
    if (!selectedNode) return
    // Optimistic: remove node from local state
    const prevNodes = nodes
    const prevEdges = edges
    setNodes((nds) => nds.filter((n) => n.id !== selectedNode.id))
    setEdges((eds) => eds.filter((e) => e.source !== selectedNode.id && e.target !== selectedNode.id))
    setSelectedNode(null)
    try {
      await deleteNodeMutation.mutateAsync(selectedNode.id)
      toast("Scene deleted", "success")
    } catch (err) {
      // Rollback local state
      setNodes(prevNodes)
      setEdges(prevEdges)
      const msg = err instanceof Error ? err.message : ""
      if (msg.includes("409") || msg.includes("in use") || msg.includes("connected")) {
        showError("Remove all edges connected to this scene first.")
      } else if (msg.includes("404")) {
        showError("Scene was already deleted.")
      } else {
        showError("Failed to delete scene")
      }
      console.error("delete node:", err)
    }
  }, [selectedNode, nodes, edges, deleteNodeMutation, toast, showError])

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
      const tempId = `temp-edge-${Date.now()}`
      // Optimistic: add edge immediately
      const prevEdges = edges
      setEdges((eds) =>
        addEdge({
          id: tempId,
          source: connection.source!,
          target: connection.target!,
          label: edgeType === "seq" ? "" : edgeType,
          style: {
            stroke: edgeType === "fork" ? "#c9734a" : edgeType === "join" ? "#d4a853" : "#8888a0",
            strokeWidth: edgeType === "seq" ? 1.5 : 2.5,
            strokeDasharray: edgeType === "choice" ? "5 5" : undefined,
          },
          labelStyle: { fill: "#8888a0", fontSize: 10 },
        }, eds),
      )
      try {
        const created = await createEdgeMutation.mutateAsync({
          from_node: connection.source,
          to_node: connection.target,
          edge_type: edgeType,
        })
        // Replace temp edge with real one
        setEdges((eds) =>
          eds.map((e) =>
            e.id === tempId
              ? { ...e, id: created.id, selected: false }
              : e,
          ),
        )
        toast("Edge created", "success")
      } catch (err) {
        // Rollback
        setEdges(prevEdges)
        const msg = err instanceof Error ? err.message : ""
        if (msg.includes("409") || msg.includes("already exists") || msg.includes("duplicate")) {
          showError("Connection already exists between these nodes")
        } else if (msg.includes("400")) {
          showError("Invalid connection. Check edge type or direction.")
        } else {
          showError("Failed to create edge")
        }
        console.error("create edge:", err)
      }
    },
    [storyId, edges, setEdges, pendingEdgeType, createEdgeMutation, toast, showError],
  )

  const acceptGeneration = useCallback(async (nodeId: string, genId: string) => {
    try {
      await api.generations.accept(storyId, nodeId, genId)
      toast("Generation accepted", "success")
      await refetchGens()
    } catch (err) {
      showError("Failed to accept generation")
      console.error("accept:", err)
    }
  }, [storyId, refetchGens, toast, showError])

  const generate = useCallback(async () => {
    if (!selectedNode) return
    setConfirmingGenerate(false)
    const genTimeout = setTimeout(() => {
      showError("Generation request timed out after 5 minutes.")
    }, 300_000)
    try {
      await api.generations.generate(storyId, selectedNode.id)
      clearTimeout(genTimeout)
      toast("Generation started (async)", "success")
      await refetchGens()
    } catch (err) {
      clearTimeout(genTimeout)
      const msg = err instanceof Error ? err.message : ""
      if (msg.includes("timeout") || msg.includes("abort")) {
        showError("Generation request timed out. Check server and try again.")
      } else if (msg.includes("429")) {
        showError("Rate limited. Wait a moment and retry.")
      } else {
        showError("Failed to start generation")
      }
      console.error("generate:", err)
    }
  }, [storyId, selectedNode, refetchGens, toast, showError])

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
  }, [])

  const onEdgeClick = useCallback((_: unknown, edge: Edge) => {
    setSelectedNode(null)
    setSelectedEdge(edge)
    setActiveTab("edit")
  }, [])

  /** Track node positions before drag for rollback on save failure. */
  const nodePositionsRef = useRef<Map<string, { x: number; y: number }>>(new Map())

  const onNodeDragStart = useCallback((_event: any, node: Node) => {
    nodePositionsRef.current.set(node.id, { ...node.position })
  }, [])

  const onNodeDragEnd: OnNodeDrag = useCallback(
    async (_event: any, node: Node) => {
      try {
        await updatePositionMutation.mutateAsync({
          nodeId: node.id,
          x: node.position.x,
          y: node.position.y,
        })
        nodePositionsRef.current.delete(node.id)
      } catch (err) {
        // Rollback to pre-drag position
        const prev = nodePositionsRef.current.get(node.id)
        if (prev) {
          setNodes((nds) =>
            nds.map((n) =>
              n.id === node.id
                ? { ...n, position: prev, data: { ...n.data } }
                : n,
            ),
          )
          nodePositionsRef.current.delete(node.id)
        }
        showError("Failed to save position. Node snapped back.")
        console.error("persist position:", err)
      }
    },
    [updatePositionMutation, setNodes, showError],
  )

  const onPaneClick = useCallback(() => {
    setSelectedNode(null)
    setSelectedEdge(null)
    setConfirmingGenerate(false)
  }, [])

  const addNode = useCallback(async () => {
    const tempId = `temp-${Date.now()}`
    // Optimistic: add temp node immediately
    const tempNode: Node<SceneNodeData> = {
      id: tempId,
      type: "scene",
      position: { x: 100 + nodes.length * 30, y: 100 + nodes.length * 30 },
      data: {
        label: "New scene",
        title: "",
        status: "draft",
        beatIntent: "New scene",
        pov: "third-person",
        tone: "neutral",
        targetWords: 300,
        characterRefs: [],
        wordCount: 0,
      },
    }
    const prevNodes = nodes
    setNodes((nds) => [...nds, tempNode])
    try {
      await createNodeMutation.mutateAsync({
        beat_intent: "New scene",
        character_refs: [],
        location_ref: null,
        pov: "third-person",
        tone: "neutral",
        target_words: 300,
      })
      toast("Scene added", "success")
    } catch (err) {
      // Rollback
      setNodes(prevNodes)
      showError("Failed to add scene")
      console.error("add node:", err)
    }
  }, [storyId, nodes, createNodeMutation, toast, showError])

  const updateNode = useCallback(async () => {
    if (!selectedNode) return
    // Optimistic: update node in local state immediately
    const prevNode = nodes.find((n) => n.id === selectedNode.id)
    const updatedNode: Node<SceneNodeData> = {
      ...selectedNode,
      data: {
        ...selectedNode.data,
        beatIntent: form.beat_intent,
        pov: form.pov,
        tone: form.tone,
        targetWords: form.target_words,
      },
    }
    setNodes((nds) => nds.map((n) => (n.id === selectedNode.id ? updatedNode : n)))
    setSelectedNode(updatedNode)
    try {
      await updateNodeMutation.mutateAsync({
        nodeId: selectedNode.id,
        data: {
          beat_intent: form.beat_intent,
          character_refs: [],
          pov: form.pov,
          tone: form.tone,
          target_words: form.target_words,
        },
      })
      toast("Scene saved", "success")
      setSelectedNode(null)
    } catch (err) {
      // Rollback
      if (prevNode) {
        setNodes((nds) => nds.map((n) => (n.id === prevNode.id ? prevNode : n)))
        setSelectedNode(prevNode)
      }
      showError("Failed to save scene")
      console.error("update node:", err)
    }
  }, [storyId, selectedNode, nodes, form, updateNodeMutation, toast, showError])

  const deleteEdge = useCallback(async () => {
    if (!selectedEdge) return
    // Optimistic: remove edge from local state
    const prevEdgeId = selectedEdge.id
    const prevEdges = edges
    setEdges((eds) => eds.filter((e) => e.id !== prevEdgeId))
    setSelectedEdge(null)
    try {
      await deleteEdgeMutation.mutateAsync(prevEdgeId)
      toast("Edge deleted", "success")
    } catch (err) {
      // Rollback
      const restored = prevEdges.find((e) => e.id === prevEdgeId)
      if (restored) setEdges((eds) => [...eds, restored])
      showError("Failed to delete edge")
      console.error("delete edge:", err)
    }
  }, [selectedEdge, edges, deleteEdgeMutation, toast, showError])

  const handleClose = useCallback(() => {
    setSelectedNode(null)
    setSelectedEdge(null)
    setConfirmingGenerate(false)
  }, [])

  const handleTabChange = useCallback((tab: "edit" | "info" | "generations" | "turns" | "agents") => {
    setActiveTab(tab)
    if (tab === "generations" && selectedNode) refetchGens()
  }, [selectedNode, refetchGens])

  return (
    <div style={{ display: "flex", height: "calc(100vh - 49px)", position: "relative" }}>
      <div style={{ flex: 1, position: "relative" }}>
        {topo.isError && !topo.isLoading && (
          <div style={{
            position: "absolute", inset: 0, zIndex: 10,
            display: "flex", flexDirection: "column", gap: 14,
            alignItems: "center", justifyContent: "center",
            background: "rgba(196,164,108,0.5)",
            backdropFilter: "blur(3px)",
          }}>
            <svg width="36" height="36" viewBox="0 0 24 24" fill="none"
              stroke="var(--error)" strokeWidth="1.5" strokeLinecap="round">
              <circle cx="12" cy="12" r="10" />
              <path d="M12 8v4M12 16h.01" />
            </svg>
            <span style={{ fontSize: 14, color: "var(--error)", fontWeight: 600, fontFamily: "var(--font-heading)" }}>
              Failed to load graph
            </span>
            <span style={{ fontSize: 12, color: "#5c4a2e", maxWidth: 300, textAlign: "center" }}>
              {(topo.error instanceof Error ? topo.error.message : "").includes("timeout") ? "Request timed out. Check your connection." :
               (topo.error instanceof Error ? topo.error.message : "").includes("500") ? "Server error. We've logged it." :
               (topo.error instanceof Error ? topo.error.message : "").includes("404") ? "Story was deleted." :
               "Cannot reach server. Check connection."}
            </span>
            <button
              onClick={() => topo.refetch()}
              style={{
                marginTop: 4, padding: "8px 20px",
                background: "#5c4a2e", color: "#f5f0e8",
                border: "none", borderRadius: "var(--radius-md)",
                cursor: "pointer", fontWeight: 600, fontSize: 13,
                transition: "background var(--transition-base)",
              }}
              onMouseEnter={(e) => e.currentTarget.style.background = "#6d5940"}
              onMouseLeave={(e) => e.currentTarget.style.background = "#5c4a2e"}
            >
              Retry
            </button>
          </div>
        )}

        {topo.isLoading && (
          <div style={{
            position: "absolute", inset: 0, zIndex: 10,
            display: "flex", alignItems: "center", justifyContent: "center",
            background: "rgba(196,164,108,0.4)",
            backdropFilter: "blur(2px)",
          }}>
            <div style={spinnerStyle} />
          </div>
        )}

        {!topo.isLoading && nodes.length === 0 && (
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
          onNodeDragStart={onNodeDragStart}
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
        gensPending={gensPending}
        gensError={gensError}
        onRetryGens={refetchGens}
        onAcceptGeneration={acceptGeneration}
        pendingEdgeType={pendingEdgeType}
        setPendingEdgeType={setPendingEdgeType}
      />
    </div>
  )
}
