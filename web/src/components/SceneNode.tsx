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
  draft:     { dot: "#8c7e70", label: "Draft" },
  generated: { dot: "#b8735c", label: "Generated" },
  accepted:  { dot: "#6b8f5e", label: "Accepted" },
  stale:     { dot: "#b05c50", label: "Stale" },
}

function SceneNode({ data }: NodeProps<Node<SceneNodeData>>) {
  const s = statusAccents[data.status] || statusAccents.draft
  const isStale = data.status === "stale"
  const isAccepted = data.status === "accepted"
  const cardBg = "#f5f0e8"
  const cardText = "#2a221e"

  return (
    <div style={{ position: "relative", paddingTop: 8 }}>
      <div style={{
        position: "absolute",
        top: 0,
        left: "50%",
        marginLeft: -5,
        width: 10,
        height: 10,
        borderRadius: "50%",
        background: "#b89560",
        boxShadow: "0 1px 2px rgba(0,0,0,0.3), inset 0 1px 0 rgba(255,255,255,0.2)",
        zIndex: 1,
      }} />
      <div style={{
        background: cardBg,
        border: `1px solid ${s.dot}88`,
        borderRadius: 4,
        padding: "14px 18px 12px",
        minWidth: 200,
        color: cardText,
        fontFamily: "var(--font-body)",
        fontSize: 13,
        position: "relative",
        boxShadow: isAccepted
          ? `0 2px 6px rgba(0,0,0,0.18), 0 0 0 1px ${s.dot}40`
          : "0 2px 6px rgba(0,0,0,0.18), 0 1px 0 rgba(0,0,0,0.06)",
        cursor: "pointer",
        transition: "box-shadow 0.2s var(--ease-out), transform 0.2s var(--ease-out)",
      }}
        onMouseEnter={(e) => {
          e.currentTarget.style.boxShadow = `0 8px 24px rgba(0,0,0,0.25), 0 2px 4px rgba(0,0,0,0.12)${isAccepted ? `, 0 0 0 1px ${s.dot}` : ""}`
          e.currentTarget.style.transform = "translateY(-3px)"
        }}
        onMouseLeave={(e) => {
          e.currentTarget.style.boxShadow = isAccepted
            ? "0 2px 6px rgba(0,0,0,0.18), 0 0 0 1px rgba(107,143,94,0.25)"
            : "0 2px 6px rgba(0,0,0,0.18), 0 1px 0 rgba(0,0,0,0.06)"
          e.currentTarget.style.transform = "none"
        }}
      >
        <Handle type="target" position={Position.Left} style={{
          background: s.dot, width: 10, height: 10, border: "2px solid #f5f0e8",
          transition: "transform 0.15s",
        }} />

        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", marginBottom: 6 }}>
          <strong style={{
            fontSize: 14,
            fontFamily: "var(--font-heading)",
            color: cardText,
            letterSpacing: "-0.01em",
          }}>
            {data.label}
          </strong>
          <span style={{
            display: "flex",
            alignItems: "center",
            gap: 4,
            fontSize: 10,
            fontWeight: 600,
            color: s.dot,
            fontFamily: "var(--font-meta)",
            textTransform: "uppercase",
            letterSpacing: "0.04em",
          }}>
            <span style={{
              width: 6, height: 6, borderRadius: "50%", background: s.dot, flexShrink: 0,
              animation: isStale ? "pulse 1.5s ease-in-out infinite" : undefined,
            }} />
            {s.label}
          </span>
        </div>

        <div style={{
          fontSize: 12,
          color: "#5a4e44",
          marginBottom: 4,
          lineHeight: 1.4,
          fontStyle: data.status === "draft" ? "italic" : "normal",
        }}>
          {data.beatIntent || "Empty beat"}
        </div>

        <div style={{
          fontSize: 10,
          color: "#8c7e70",
          fontFamily: "var(--font-meta)",
          display: "flex",
          gap: 8,
        }}>
          <span>{data.pov}</span>
          <span>·</span>
          <span>{data.tone}</span>
          <span>·</span>
          <span>{data.targetWords}w</span>
        </div>

        <Handle type="source" position={Position.Right} style={{
          background: s.dot, width: 10, height: 10, border: "2px solid #f5f0e8",
          transition: "transform 0.15s",
        }} />
      </div>
    </div>
  )
}

export default memo(SceneNode)
