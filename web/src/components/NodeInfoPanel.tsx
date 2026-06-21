import type { Edge } from "@xyflow/react"
import { destructiveBtnStyle } from "../api/types"

interface NodeInfoPanelProps {
  edges: Edge[]
  selectedNodeId: string
  status: string
  beatIntent: string
  onDelete: () => void
}

const trashSvg = (
  <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round">
    <path d="M2 4h12M5 4V2.5A.5.5 0 015.5 2h5a.5.5 0 01.5.5V4M13 4v9.5a1.5 1.5 0 01-1.5 1.5h-7A1.5 1.5 0 013 13.5V4" />
  </svg>
)

export default function NodeInfoPanel({ edges, selectedNodeId, status, beatIntent, onDelete }: NodeInfoPanelProps) {
  const inEdges = edges.filter((e) => e.target === selectedNodeId)
  const outEdges = edges.filter((e) => e.source === selectedNodeId)

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
          {status || "Unknown"}
        </div>
      </div>

      <div>
        <div style={{ color: "var(--text-dim)", fontSize: 11, marginBottom: 2 }}>Beat Intent</div>
        <div style={{ color: "var(--text)", lineHeight: 1.4 }}>
          {beatIntent || "—"}
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
        onClick={onDelete}
        style={{ ...destructiveBtnStyle, marginTop: 8, justifyContent: "center" }}
        onMouseEnter={(e) => { e.currentTarget.style.background = "rgba(212,103,103,0.12)" }}
        onMouseLeave={(e) => { e.currentTarget.style.background = "transparent" }}
      >
        {trashSvg} Delete Scene
      </button>
    </div>
  )
}
