import { memo } from "react"
import { Handle, Position, type NodeProps, type Node } from "@xyflow/react"

type SceneNodeData = {
  label: string
  status: "draft" | "generated" | "accepted" | "stale"
  beatIntent: string
  pov: string
  tone: string
  targetWords: number
}

const statusAccents: Record<string, { dot: string; label: string }> = {
  draft:     { dot: "#8888a0", label: "Draft" },
  generated: { dot: "#c9734a", label: "Generated" },
  accepted:  { dot: "#7bb87b", label: "Accepted" },
  stale:     { dot: "#d46767", label: "Stale" },
}

function SceneNode({ data }: NodeProps<Node<SceneNodeData>>) {
  const s = statusAccents[data.status] || statusAccents.draft
  const isStale = data.status === "stale"
  const isAccepted = data.status === "accepted"

  return (
    <div style={{
      background: "#f5f0e8",
      border: `1px solid ${s.dot}`,
      borderRadius: 8,
      padding: "14px 18px 12px",
      minWidth: 200,
      color: "#1a1a24",
      fontFamily: "var(--font-body)",
      fontSize: 13,
      position: "relative",
      boxShadow: isAccepted
        ? `0 1px 3px rgba(0,0,0,0.12), 0 0 0 1px ${s.dot}40`
        : "0 1px 3px rgba(0,0,0,0.12), 0 1px 2px rgba(0,0,0,0.08)",
      cursor: "pointer",
      transition: "box-shadow 0.2s var(--ease-out), transform 0.2s var(--ease-out)",
    }}
      onMouseEnter={(e) => {
        e.currentTarget.style.boxShadow = `0 8px 24px rgba(0,0,0,0.2), 0 2px 4px rgba(0,0,0,0.1)${isAccepted ? `, 0 0 0 1px ${s.dot}` : ""}`
        e.currentTarget.style.transform = "translateY(-2px)"
      }}
      onMouseLeave={(e) => {
        e.currentTarget.style.boxShadow = isAccepted
          ? "0 1px 3px rgba(0,0,0,0.12), 0 0 0 1px rgba(123,184,123,0.25)"
          : "0 1px 3px rgba(0,0,0,0.12), 0 1px 2px rgba(0,0,0,0.08)"
        e.currentTarget.style.transform = "none"
      }}
    >
      <Handle type="target" position={Position.Left} style={{
        background: s.dot, width: 10, height: 10, border: "2px solid #f5f0e8",
        transition: "transform 0.15s",
      }} />

      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", marginBottom: 6 }}>
        <strong style={{ fontSize: 14, fontFamily: "var(--font-heading)", color: "#1a1a24" }}>{data.label}</strong>
        <span style={{
          display: "flex",
          alignItems: "center",
          gap: 4,
          fontSize: 11,
          fontWeight: 600,
          color: s.dot,
        }}>
          <span style={{
            width: 6, height: 6, borderRadius: "50%", background: s.dot, flexShrink: 0,
            animation: isStale ? "pulse 1.5s ease-in-out infinite" : undefined,
          }} />
          {s.label}
        </span>
      </div>

      <div style={{ fontSize: 12, color: "#555", marginBottom: 4, lineHeight: 1.4 }}>
        {data.beatIntent}
      </div>

      <div style={{ fontSize: 11, color: "#888" }}>
        {data.pov} · {data.tone} · {data.targetWords}w
      </div>

      <Handle type="source" position={Position.Right} style={{
        background: s.dot, width: 10, height: 10, border: "2px solid #f5f0e8",
        transition: "transform 0.15s",
      }} />
    </div>
  )
}

export default memo(SceneNode)