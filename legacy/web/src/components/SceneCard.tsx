import { useState, useCallback } from "react"
import type { GraphNode, GraphEdge, Generation } from "../api/types"
import SceneSlideDown from "./SceneSlideDown"
import SceneConnections from "./SceneConnections"

interface SceneCardProps {
  node: GraphNode
  nodes: GraphNode[]
  edges: GraphEdge[]
  generations?: Generation[]
  onOpenInGraph?: (sceneId: string) => void
}

const statusColor: Record<string, string> = {
  draft: "var(--text-faint)",
  pending: "var(--warn)",
  generating: "var(--warn)",
  complete: "var(--success)",
  accepted: "var(--success)",
  error: "var(--danger)",
}

const statusDotStyle = (status: string): React.CSSProperties => ({
  width: 6, height: 6, borderRadius: "50%",
  background: statusColor[status] || "var(--text-faint)",
  flexShrink: 0,
})

export default function SceneCard({ node, nodes, edges, generations, onOpenInGraph }: SceneCardProps) {
  const [expanded, setExpanded] = useState(false)
  const [proseExpanded, setProseExpanded] = useState(true)
  const toggleCard = useCallback(() => setExpanded((v) => !v), [])
  const toggleProse = useCallback(() => setProseExpanded((v) => !v), [])

  const accepted = generations?.find((g) => g.accepted)
  const lastGen = generations?.[generations.length - 1]
  const prose = accepted?.output || lastGen?.output || ""

  return (
    <div style={{
      border: "1px solid var(--border)",
      borderRadius: "var(--radius-md)",
      background: "var(--bg-warm)",
      overflow: "hidden",
    }}>
      <button
        onClick={toggleCard}
        className="btn-press"
        style={{
          width: "100%", display: "flex", alignItems: "center", gap: 10,
          padding: "12px 16px", background: "none", border: "none",
          cursor: "pointer", textAlign: "left",
          borderBottom: expanded ? "1px solid var(--border)" : "none",
        }}
      >
        <span style={{
          display: "inline-block",
          transform: expanded ? "rotate(90deg)" : "rotate(0deg)",
          transition: "transform 0.15s", fontSize: 12, color: "var(--text-faint)",
        }}>
          ▸
        </span>
        <span style={statusDotStyle(node.status)} />
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ fontSize: 14, fontWeight: 600, color: "var(--text)", fontFamily: "var(--font-heading)" }}>
            {node.title || "Untitled Scene"}
          </div>
          {node.beat_intent && (
            <div style={{ fontSize: 11, color: "var(--text-dim)", marginTop: 2, whiteSpace: "nowrap", overflow: "hidden", textOverflow: "ellipsis" }}>
              {node.beat_intent}
            </div>
          )}
        </div>
        <div style={{ display: "flex", gap: 8, alignItems: "center", flexShrink: 0 }}>
          {node.target_words > 0 && (
            <span style={{ fontSize: 10, color: "var(--text-faint)" }}>
              {node.target_words}w
            </span>
          )}
          {onOpenInGraph && (
            <span
              onClick={(e) => { e.stopPropagation(); onOpenInGraph(node.id) }}
              style={{
                padding: "2px 6px", fontSize: 9, color: "var(--accent)",
                border: "1px solid var(--accent-dim)", borderRadius: "var(--radius-sm)",
                cursor: "pointer", textTransform: "uppercase", letterSpacing: "0.04em",
              }}
            >
              Graph
            </span>
          )}
        </div>
      </button>

      {expanded && (
        <div style={{ padding: "0 16px 12px" }}>
          <button
            onClick={toggleProse}
            className="btn-press"
            style={{
              width: "100%", display: "flex", alignItems: "center", gap: 6,
              padding: "8px 0", background: "none", border: "none",
              cursor: "pointer", color: "var(--text)", fontSize: 13,
              fontWeight: 500, fontFamily: "var(--font-heading)",
            }}
          >
            <span style={{
              display: "inline-block",
              transform: proseExpanded ? "rotate(90deg)" : "rotate(0deg)",
              transition: "transform 0.15s", fontSize: 10, color: "var(--text-faint)",
            }}>
              ▸
            </span>
            Prose
            {accepted && <span style={{ fontSize: 9, color: "var(--success)", marginLeft: 6 }}>✓ accepted</span>}
          </button>

          {proseExpanded && (
            <div style={{
              padding: "8px 12px", background: "var(--bg)",
              borderRadius: "var(--radius-sm)", fontSize: 13,
              lineHeight: 1.7, color: "var(--text)", whiteSpace: "pre-wrap",
              marginBottom: 4,
            }}>
              {prose || <span style={{ color: "var(--text-faint)", fontStyle: "italic" }}>No prose generated yet</span>}
            </div>
          )}

          <SceneSlideDown label="Metadata">
            <div style={{ display: "grid", gridTemplateColumns: "auto 1fr", gap: "2px 12px", fontSize: 11 }}>
              <span style={{ color: "var(--text-faint)" }}>Status</span>
              <span style={{ color: "var(--text)" }}>{node.status}</span>
              <span style={{ color: "var(--text-faint)" }}>POV</span>
              <span style={{ color: "var(--text)" }}>{node.pov}</span>
              <span style={{ color: "var(--text-faint)" }}>Tone</span>
              <span style={{ color: "var(--text)" }}>{node.tone}</span>
              <span style={{ color: "var(--text-faint)" }}>Target</span>
              <span style={{ color: "var(--text)" }}>{node.target_words} words</span>
              {node.location_ref && (
                <>
                  <span style={{ color: "var(--text-faint)" }}>Location</span>
                  <span style={{ color: "var(--text)" }}>{node.location_ref}</span>
                </>
              )}
            </div>
          </SceneSlideDown>

          <SceneSlideDown label="Connections">
            <SceneConnections sceneId={node.id} nodes={nodes} edges={edges} />
          </SceneSlideDown>

          {generations && generations.length > 0 && (
            <SceneSlideDown label="Generations" badge={generations.length}>
              {generations.map((g, i) => (
                <div key={g.id} style={{
                  display: "flex", alignItems: "center", gap: 6,
                  padding: "4px 0", borderBottom: i < generations.length - 1 ? "1px solid var(--border)" : "none",
                  fontSize: 11,
                }}>
                  <span style={{
                    width: 6, height: 6, borderRadius: "50%",
                    background: g.accepted ? "var(--success)" : g.status === "error" ? "var(--danger)" : "var(--text-faint)",
                    flexShrink: 0,
                  }} />
                  <span style={{ color: "var(--text-dim)", flex: 1 }}>
                    {g.model || "unknown model"} · {g.status}
                  </span>
                  {g.total_tokens && (
                    <span style={{ color: "var(--text-faint)" }}>{g.total_tokens}t</span>
                  )}
                  <span style={{ color: "var(--text-faint)", fontSize: 10 }}>
                    {g.created_at ? new Date(g.created_at).toLocaleDateString() : ""}
                  </span>
                </div>
              ))}
            </SceneSlideDown>
          )}
        </div>
      )}
    </div>
  )
}
