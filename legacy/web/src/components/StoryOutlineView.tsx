import { useState, useCallback, useRef } from "react"
import { useTopology, useUpdateNode, useCreateEdge } from "../api/hooks"
import type { GraphNode } from "../api/types"

interface StoryOutlineViewProps {
  storyId: string
}

const statusDotStyle = (status: string): React.CSSProperties => ({
  width: 8, height: 8, borderRadius: "50%", flexShrink: 0,
  background: status === "accepted" ? "var(--success)"
    : status === "complete" ? "var(--success)"
    : status === "generating" ? "var(--warn)"
    : status === "error" ? "var(--danger)"
    : "var(--text-faint)",
})

export default function StoryOutlineView({ storyId }: StoryOutlineViewProps) {
  const { data: topology, isLoading } = useTopology(storyId)
  const updateNode = useUpdateNode(storyId)
  const createEdge = useCreateEdge(storyId)

  const [editingId, setEditingId] = useState<string | null>(null)
  const [editForm, setEditForm] = useState({ beat_intent: "", pov: "", tone: "", target_words: 0 })
  const [dragId, setDragId] = useState<string | null>(null)
  const dragNodeRef = useRef<HTMLDivElement | null>(null)

  const startEditing = useCallback((node: GraphNode) => {
    setEditingId(node.id)
    setEditForm({
      beat_intent: node.beat_intent,
      pov: node.pov,
      tone: node.tone,
      target_words: node.target_words,
    })
  }, [])

  const saveEdit = useCallback(() => {
    if (!editingId) return
    updateNode.mutate({ nodeId: editingId, data: editForm as unknown as Record<string, unknown> })
    setEditingId(null)
  }, [editingId, editForm, updateNode])

  const onDragStart = useCallback((e: React.DragEvent, nodeId: string) => {
    setDragId(nodeId)
    e.dataTransfer.effectAllowed = "move"
  }, [])

  const onDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.dataTransfer.dropEffect = "move"
  }, [])

  const onDrop = useCallback((e: React.DragEvent, targetId: string) => {
    e.preventDefault()
    if (!dragId || dragId === targetId || !topology) return

    const ordered = topology.topological_order
    const srcIdx = ordered.indexOf(dragId)
    const tgtIdx = ordered.indexOf(targetId)
    if (srcIdx === -1 || tgtIdx === -1) return

    const newOrder = [...ordered]
    newOrder.splice(srcIdx, 1)
    newOrder.splice(tgtIdx, 0, dragId)

    const edges = topology.edges
    const seqEdges = edges.filter((e) => e.edge_type === "seq" || e.edge_type === "choice")

    const nodesInOrder = newOrder
      .map((id) => topology.nodes.find((n) => n.id === id))
      .filter((n): n is GraphNode => n != null)

    for (let i = 0; i < nodesInOrder.length - 1; i++) {
      const from = nodesInOrder[i]
      const to = nodesInOrder[i + 1]
      const exists = seqEdges.some((e) => e.from_node === from.id && e.to_node === to.id)
      if (!exists) {
        createEdge.mutate({ from_node: from.id, to_node: to.id, edge_type: "seq" })
      }
    }

    setDragId(null)
  }, [dragId, topology, createEdge])

  if (isLoading) {
    return (
      <div style={{
        display: "flex", alignItems: "center", justifyContent: "center",
        height: "100%", color: "var(--text-dim)", fontSize: 13,
      }}>
        Loading...
      </div>
    )
  }

  if (!topology || topology.nodes.length === 0) {
    return (
      <div style={{
        display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center",
        height: "100%", gap: 8, color: "var(--text-dim)",
      }}>
        <span style={{ fontSize: 14, fontStyle: "italic", fontFamily: "var(--font-heading)" }}>
          No scenes yet
        </span>
        <span style={{ fontSize: 11, color: "var(--text-faint)" }}>
          Switch to Graph mode to add scenes
        </span>
      </div>
    )
  }

  const ordered = topology.topological_order
    .map((id) => topology.nodes.find((n) => n.id === id))
    .filter((n): n is GraphNode => n != null)

  return (
    <div style={{
      height: "100%", overflowY: "auto",
      display: "flex", flexDirection: "column", alignItems: "center",
    }}>
      <div style={{ maxWidth: 860, width: "100%", padding: "20px 32px 48px" }}>
        <div style={{
          display: "grid", gridTemplateColumns: "24px 1fr 80px 100px 100px 60px",
          gap: 8, padding: "8px 12px", fontSize: 10,
          color: "var(--text-faint)", textTransform: "uppercase",
          letterSpacing: "0.05em", borderBottom: "1px solid var(--border)",
          marginBottom: 4,
        }}>
          <span />
          <span>Scene</span>
          <span>Status</span>
          <span>POV</span>
          <span>Tone</span>
          <span>Words</span>
        </div>

        {ordered.map((node, idx) => (
          <div
            key={node.id}
            draggable
            onDragStart={(e) => onDragStart(e, node.id)}
            onDragOver={onDragOver}
            onDrop={(e) => onDrop(e, node.id)}
            ref={dragId === node.id ? dragNodeRef : undefined}
            style={{
              display: "grid", gridTemplateColumns: "24px 1fr 80px 100px 100px 60px",
              gap: 8, alignItems: "center", padding: "6px 12px",
              borderRadius: "var(--radius-sm)",
              background: editingId === node.id ? "var(--accent-dim)" : "transparent",
              border: dragId === node.id ? "1px dashed var(--accent)" : "1px solid transparent",
              opacity: dragId === node.id ? 0.5 : 1,
              cursor: "grab", fontSize: 12,
              transition: "background 0.1s",
            }}
            onMouseEnter={(e) => {
              if (editingId !== node.id) e.currentTarget.style.background = "var(--bg-warm)"
            }}
            onMouseLeave={(e) => {
              if (editingId !== node.id) e.currentTarget.style.background = "transparent"
            }}
          >
            <div style={{
              display: "flex", alignItems: "center", justifyContent: "center",
              color: "var(--text-faint)", fontSize: 10, fontWeight: 600,
            }}>
              {idx + 1}
            </div>

            <div style={{ minWidth: 0 }}>
              {editingId === node.id ? (
                <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
                  <input
                    value={editForm.beat_intent}
                    onChange={(e) => setEditForm((f) => ({ ...f, beat_intent: e.target.value }))}
                    placeholder="Beat intent"
                    style={{
                      padding: "4px 8px", fontSize: 12,
                      background: "var(--bg)", border: "1px solid var(--border)",
                      borderRadius: "var(--radius-sm)", color: "var(--text)",
                      outline: "none", width: "100%",
                    }}
                    autoFocus
                    onKeyDown={(e) => { if (e.key === "Enter") saveEdit(); if (e.key === "Escape") setEditingId(null) }}
                  />
                  <div style={{ display: "flex", gap: 4 }}>
                    <button
                      onClick={saveEdit}
                      className="btn-press"
                      style={{
                        padding: "2px 10px", fontSize: 10, background: "var(--accent)",
                        color: "#1a1512", border: "none", borderRadius: "var(--radius-sm)",
                        cursor: "pointer", fontWeight: 600,
                      }}
                    >
                      Save
                    </button>
                    <button
                      onClick={() => setEditingId(null)}
                      className="btn-press"
                      style={{
                        padding: "2px 10px", fontSize: 10, background: "none",
                        color: "var(--text-dim)", border: "1px solid var(--border)",
                        borderRadius: "var(--radius-sm)", cursor: "pointer",
                      }}
                    >
                      Cancel
                    </button>
                  </div>
                </div>
              ) : (
                <div
                  onClick={() => startEditing(node)}
                  style={{
                    display: "flex", alignItems: "center", gap: 8,
                    cursor: "pointer", minWidth: 0,
                  }}
                >
                  <span style={{
                    fontWeight: 500, color: "var(--text)", whiteSpace: "nowrap",
                    overflow: "hidden", textOverflow: "ellipsis",
                  }}>
                    {node.title || "Untitled"}
                  </span>
                  <span style={{
                    fontSize: 10, color: "var(--text-faint)", whiteSpace: "nowrap",
                    overflow: "hidden", textOverflow: "ellipsis",
                  }}>
                    {node.beat_intent}
                  </span>
                </div>
              )}
            </div>

            <div style={{ display: "flex", alignItems: "center", gap: 4 }}>
              <span style={statusDotStyle(node.status)} />
              <span style={{ fontSize: 10, color: "var(--text-dim)" }}>{node.status}</span>
            </div>

            <div style={{ fontSize: 11, color: "var(--text-dim)" }}>
              {editingId === node.id ? (
                <select
                  value={editForm.pov}
                  onChange={(e) => setEditForm((f) => ({ ...f, pov: e.target.value }))}
                  style={{
                    padding: "2px 4px", fontSize: 11,
                    background: "var(--bg)", border: "1px solid var(--border)",
                    borderRadius: "var(--radius-sm)", color: "var(--text)", width: "100%",
                    outline: "none",
                  }}
                >
                  <option value="first-person">first-person</option>
                  <option value="second-person">second-person</option>
                  <option value="third-person">third-person</option>
                  <option value="third-limited">third-limited</option>
                  <option value="third-omniscient">third-omniscient</option>
                </select>
              ) : node.pov}
            </div>

            <div style={{ fontSize: 11, color: "var(--text-dim)" }}>
              {editingId === node.id ? (
                <select
                  value={editForm.tone}
                  onChange={(e) => setEditForm((f) => ({ ...f, tone: e.target.value }))}
                  style={{
                    padding: "2px 4px", fontSize: 11,
                    background: "var(--bg)", border: "1px solid var(--border)",
                    borderRadius: "var(--radius-sm)", color: "var(--text)",
                    outline: "none", width: "100%",
                  }}
                >
                  <option value="neutral">neutral</option>
                  <option value="dark">dark</option>
                  <option value="light">light</option>
                  <option value="whimsical">whimsical</option>
                  <option value="gritty">gritty</option>
                  <option value="mysterious">mysterious</option>
                  <option value="romantic">romantic</option>
                  <option value="tragic">tragic</option>
                </select>
              ) : node.tone}
            </div>

            <div style={{ fontSize: 11, color: "var(--text-faint)" }}>
              {editingId === node.id ? (
                <input
                  type="number"
                  value={editForm.target_words}
                  onChange={(e) => setEditForm((f) => ({ ...f, target_words: Number(e.target.value) }))}
                  style={{
                    padding: "2px 4px", fontSize: 11, width: 50,
                    background: "var(--bg)", border: "1px solid var(--border)",
                    borderRadius: "var(--radius-sm)", color: "var(--text)",
                    outline: "none",
                  }}
                />
              ) : node.target_words}
            </div>
          </div>
        ))}

        <div style={{
          marginTop: 16, padding: "8px 12px",
          fontSize: 10, color: "var(--text-faint)",
          borderTop: "1px solid var(--border)",
          display: "flex", justifyContent: "space-between",
        }}>
          <span>{ordered.length} scenes</span>
          <span>Drag to reorder, click to edit</span>
        </div>
      </div>
    </div>
  )
}
