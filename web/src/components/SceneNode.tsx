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

const statusAccents: Record<string, { dot: string; label: string; bg: string }> = {
  draft:     { dot: "#8c7e70", label: "Draft", bg: "rgba(140,126,112,0.1)" },
  generated: { dot: "#b8735c", label: "Generated", bg: "rgba(184,115,92,0.1)" },
  accepted:  { dot: "#6b8f5e", label: "Accepted", bg: "rgba(107,143,94,0.1)" },
  stale:     { dot: "#b05c50", label: "Stale", bg: "rgba(176,92,80,0.1)" },
}

function SceneNode({ data }: NodeProps<Node<SceneNodeData>>) {
  const s = statusAccents[data.status] || statusAccents.draft
  const isStale = data.status === "stale"
  const isAccepted = data.status === "accepted"
  const cardBg = "#f5f0e8"
  const cardText = "#2a221e"

  return (
    <div style={{ position: "relative", paddingTop: 12 }}>
      <div style={{
        position: "absolute",
        top: 2,
        left: "50%",
        marginLeft: -6,
        width: 12,
        height: 12,
        borderRadius: "50%",
        background: "linear-gradient(135deg, #b89560 40%, #a08550 100%)",
        boxShadow: "0 1px 3px rgba(0,0,0,0.4), inset 0 1px 0 rgba(255,255,255,0.25)",
        zIndex: 1,
      }} />
      <div style={{
        background: cardBg,
        border: `1px solid ${isAccepted ? `${s.dot}99` : `${s.dot}66`}`,
        borderRadius: 6,
        padding: "16px 20px 14px",
        minWidth: 200,
        color: cardText,
        fontFamily: "var(--font-body)",
        fontSize: 13,
        position: "relative",
        boxShadow: isAccepted
          ? "0 2px 8px rgba(0,0,0,0.18), 0 0 0 1px rgba(107,143,94,0.25)"
          : "0 2px 6px rgba(0,0,0,0.15), 0 1px 0 rgba(0,0,0,0.06)",
        cursor: "pointer",
        transition: "box-shadow 0.2s var(--ease-out), transform 0.2s var(--ease-out)",
      }}
        onMouseEnter={(e) => {
          e.currentTarget.style.boxShadow = `0 8px 24px rgba(0,0,0,0.25), 0 2px 4px rgba(0,0,0,0.12)${isAccepted ? `, 0 0 0 1px ${s.dot}` : ""}`
          e.currentTarget.style.transform = "translateY(-3px)"
        }}
        onMouseLeave={(e) => {
          e.currentTarget.style.boxShadow = isAccepted
            ? "0 2px 8px rgba(0,0,0,0.18), 0 0 0 1px rgba(107,143,94,0.25)"
            : "0 2px 6px rgba(0,0,0,0.15), 0 1px 0 rgba(0,0,0,0.06)"
          e.currentTarget.style.transform = "none"
        }}
      >
        <Handle type="target" position={Position.Left} style={{
          background: s.dot, width: 10, height: 10, border: "2px solid #f5f0e8",
          transition: "transform 0.15s",
        }} />

        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", marginBottom: 8 }}>
          <strong style={{
            fontSize: 14,
            fontFamily: "var(--font-heading)",
            color: cardText,
            letterSpacing: "-0.01em",
          }}>
            {data.label}
          </strong>
          <span style={{
            display: "inline-flex",
            alignItems: "center",
            gap: 5,
            fontSize: 10,
            fontWeight: 600,
            color: s.dot,
            fontFamily: "var(--font-mono)",
            textTransform: "uppercase",
            letterSpacing: "0.04em",
            background: s.bg,
            padding: "2px 8px",
            borderRadius: 10,
            whiteSpace: "nowrap",
          }}>
            <span style={{
              width: 5, height: 5, borderRadius: "50%", background: s.dot, flexShrink: 0,
              animation: isStale ? "pulse 1.5s ease-in-out infinite" : undefined,
            }} />
            {s.label}
          </span>
        </div>

        <div style={{
          fontSize: 12,
          color: "#5a4e44",
          marginBottom: 6,
          lineHeight: 1.4,
          fontStyle: data.status === "draft" ? "italic" : "normal",
        }}>
          {data.beatIntent || "Empty beat"}
        </div>

        <div style={{
          fontSize: 10,
          color: "#7a6e62",
          fontFamily: "var(--font-mono)",
          display: "flex",
          gap: 8,
          alignItems: "center",
        }}>
          <span>POV: {data.pov || "—"}</span>
          <span style={{ opacity: 0.4 }}>|</span>
          <span>Tone: {data.tone || "—"}</span>
          <span style={{ opacity: 0.4 }}>|</span>
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
