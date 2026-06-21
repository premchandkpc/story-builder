import type { Edge } from "@xyflow/react"
import { destructiveBtnStyle } from "../api/types"

interface EdgeInfoPanelProps {
  selectedEdge: Edge
  onDelete: () => void
}

const trashSvg = (
  <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round">
    <path d="M2 4h12M5 4V2.5A.5.5 0 015.5 2h5a.5.5 0 01.5.5V4M13 4v9.5a1.5 1.5 0 01-1.5 1.5h-7A1.5 1.5 0 013 13.5V4" />
  </svg>
)

export default function EdgeInfoPanel({ selectedEdge, onDelete }: EdgeInfoPanelProps) {
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
        onClick={onDelete}
        style={{ ...destructiveBtnStyle, marginTop: 4, justifyContent: "center" }}
        onMouseEnter={(e) => { e.currentTarget.style.background = "rgba(212,103,103,0.12)" }}
        onMouseLeave={(e) => { e.currentTarget.style.background = "transparent" }}
      >
        {trashSvg} Delete Edge
      </button>
    </div>
  )
}
