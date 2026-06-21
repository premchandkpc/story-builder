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
      <div style={{
        background: "var(--surface)",
        borderRadius: "var(--radius-md)",
        padding: "10px 12px",
        border: "1px solid var(--border)",
      }}>
        <div style={{ color: "var(--text-faint)", fontSize: 10, marginBottom: 6, letterSpacing: "0.04em", textTransform: "uppercase" }}>Edge Type</div>
        <span style={{
          color: "var(--accent)", fontWeight: 600, textTransform: "uppercase",
          padding: "2px 8px", borderRadius: "var(--radius-sm)",
          background: "rgba(212,168,83,0.1)", fontSize: 10,
          letterSpacing: "0.05em",
        }}>
          {edgeLabel}
        </span>
      </div>

      <div style={{
        background: "var(--surface)",
        borderRadius: "var(--radius-md)",
        padding: "10px 12px",
        border: "1px solid var(--border)",
      }}>
        <div style={{ display: "flex", gap: 12, fontSize: 12 }}>
          <div style={{ flex: 1 }}>
            <span style={{ color: "var(--text-faint)", fontSize: 10, letterSpacing: "0.04em", textTransform: "uppercase", display: "block", marginBottom: 3 }}>From</span>
            <span style={{ color: "var(--text)", fontFamily: "var(--font-mono)", fontSize: 11 }}>{selectedEdge.source.slice(-8)}</span>
          </div>
          <div style={{ flex: 1 }}>
            <span style={{ color: "var(--text-faint)", fontSize: 10, letterSpacing: "0.04em", textTransform: "uppercase", display: "block", marginBottom: 3 }}>To</span>
            <span style={{ color: "var(--text)", fontFamily: "var(--font-mono)", fontSize: 11 }}>{selectedEdge.target.slice(-8)}</span>
          </div>
        </div>
      </div>

      <button
        onClick={onDelete}
        style={{ ...destructiveBtnStyle, marginTop: 4, justifyContent: "center", fontSize: 11 }}
        onMouseEnter={(e) => { e.currentTarget.style.background = "rgba(212,103,103,0.12)" }}
        onMouseLeave={(e) => { e.currentTarget.style.background = "transparent" }}
      >
        {trashSvg} Delete Edge
      </button>
    </div>
  )
}
