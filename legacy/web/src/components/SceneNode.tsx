import { memo, useMemo } from "react"
import { Handle, Position, type NodeProps, type Node } from "@xyflow/react"
import type { SceneNodeData } from "../api/types"

const statusConfig: Record<string, { dot: string; border: string; label: string; ringGlow: string }> = {
  draft:     { dot: "#8c7e70", border: "var(--border)",     label: "Draft",     ringGlow: "rgba(140,126,112,0.08)" },
  generated: { dot: "#b8735c", border: "#b8735c",           label: "Generated", ringGlow: "rgba(184,115,92,0.15)" },
  accepted:  { dot: "#6b8f5e", border: "#6b8f5e",           label: "Accepted",  ringGlow: "rgba(107,143,94,0.2)" },
  stale:     { dot: "#b05c50", border: "#b05c50",           label: "Stale",     ringGlow: "rgba(176,92,80,0.15)" },
}

function SceneNode({ data }: NodeProps<Node<SceneNodeData>>) {
  const cfg = statusConfig[data.status] || statusConfig.draft
  const isStale = data.status === "stale"
  const isAccepted = data.status === "accepted"
  const isDraft = data.status === "draft"
  const hasChars = data.characterRefs && data.characterRefs.length > 0

  const wordProgress = useMemo(() => {
    if (data.targetWords <= 0) return 0
    return Math.min((data.wordCount || 0) / data.targetWords, 1)
  }, [data.wordCount, data.targetWords])

  const progressColor = useMemo(() => {
    if (wordProgress >= 1) return "var(--success)"
    if (wordProgress >= 0.5) return "var(--accent)"
    if (wordProgress > 0) return "var(--warn)"
    return "var(--border)"
  }, [wordProgress])

  const charInitials = useMemo(() => {
    if (!data.characterRefs) return []
    return data.characterRefs.map((name) => {
      const parts = name.split(/[\s_-]+/)
      return parts.map((p) => p[0] || "").join("").toUpperCase().slice(0, 2)
    })
  }, [data.characterRefs])

  return (
    <div className="stagger-fade-in" style={{ position: "relative" }}>
      {/* Top tab — vintage index card metal ring */}
      <div style={{
        position: "absolute",
        top: -2,
        left: "50%",
        marginLeft: -6,
        width: 12,
        height: 16,
        background: "linear-gradient(180deg, #a08550 0%, #c4a46c 40%, #a08550 100%)",
        borderRadius: "0 0 4px 4px",
        boxShadow: "0 2px 4px rgba(0,0,0,0.25), inset 0 1px 0 rgba(255,255,255,0.15)",
        zIndex: 2,
      }} />

      {/* Status color top border strip */}
      <div style={{
        height: 3,
        background: `linear-gradient(90deg, ${cfg.dot}, ${cfg.border})`,
        borderRadius: "5px 5px 0 0",
      }} />

      {/* Main card body */}
      <div style={{
        background: "linear-gradient(135deg, #f5f0e8 0%, #efe8dc 50%, #f0eadc 100%)",
        borderLeft: `1px solid ${cfg.dot}44`,
        borderRight: `1px solid ${cfg.dot}44`,
        borderBottom: `1px solid ${cfg.dot}33`,
        padding: "12px 16px 10px",
        minWidth: 200,
        position: "relative",
        boxShadow: isAccepted
          ? `0 2px 8px rgba(0,0,0,0.12), 0 0 0 1px ${cfg.dot}30, inset 0 1px 0 rgba(255,255,255,0.7)`
          : "0 2px 6px rgba(0,0,0,0.1), inset 0 1px 0 rgba(255,255,255,0.6)",
        cursor: "pointer",
        transition: "box-shadow var(--transition-base), transform var(--transition-base)",
      }}
        onMouseEnter={(e) => {
          e.currentTarget.style.boxShadow = `0 8px 24px rgba(0,0,0,0.18), 0 2px 4px rgba(0,0,0,0.08), 0 0 0 1px ${cfg.dot}40`
          e.currentTarget.style.transform = "translateY(-3px)"
        }}
        onMouseLeave={(e) => {
          e.currentTarget.style.boxShadow = isAccepted
            ? "0 2px 8px rgba(0,0,0,0.12), 0 0 0 1px rgba(107,143,94,0.2)"
            : "0 2px 6px rgba(0,0,0,0.1), inset 0 1px 0 rgba(255,255,255,0.6)"
          e.currentTarget.style.transform = "none"
        }}
      >
        {/* Source & Target handles */}
        <Handle type="target" position={Position.Left} style={{
          background: cfg.dot,
          width: 8,
          height: 8,
          border: `2px solid #f5f0e8`,
          borderRadius: "50%",
          boxShadow: `0 0 4px ${cfg.dot}66`,
          transition: "transform var(--transition-fast), box-shadow var(--transition-fast)",
        }}
          onMouseEnter={(e) => { e.currentTarget.style.transform = "scale(1.3)"; e.currentTarget.style.boxShadow = `0 0 8px ${cfg.dot}` }}
          onMouseLeave={(e) => { e.currentTarget.style.transform = "scale(1)"; e.currentTarget.style.boxShadow = `0 0 4px ${cfg.dot}66` }}
        />

        {/* Header row: title + compact status */}
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", marginBottom: 6 }}>
          <div style={{ flex: 1, minWidth: 0, marginRight: 8 }}>
            <div style={{
              fontSize: 14,
              fontFamily: "var(--font-heading)",
              fontWeight: 700,
              color: "#2a221e",
              letterSpacing: "-0.01em",
              lineHeight: 1.3,
              whiteSpace: "nowrap",
              overflow: "hidden",
              textOverflow: "ellipsis",
            }}>
              {data.label}
            </div>
          </div>
          <span style={{
            display: "flex",
            alignItems: "center",
            gap: 3,
            fontSize: 9,
            fontWeight: 600,
            color: cfg.dot,
            fontFamily: "var(--font-mono)",
            textTransform: "uppercase",
            letterSpacing: "0.05em",
            whiteSpace: "nowrap",
            flexShrink: 0,
          }}>
            <span style={{
              width: 5,
              height: 5,
              borderRadius: "50%",
              background: cfg.dot,
              flexShrink: 0,
              boxShadow: isStale ? `0 0 6px ${cfg.dot}` : isAccepted ? `0 0 5px ${cfg.dot}` : "none",
              animation: isStale ? "pulse 1.5s ease-in-out infinite" : undefined,
            }} />
            {cfg.label}
          </span>
        </div>

        {/* Beat intent / description */}
        <div style={{
          fontSize: 11.5,
          color: "#5a4e44",
          marginBottom: 8,
          lineHeight: 1.45,
          fontStyle: isDraft ? "italic" : "normal",
          fontFamily: "var(--font-body)",
          display: "-webkit-box",
          WebkitLineClamp: 2,
          WebkitBoxOrient: "vertical",
          overflow: "hidden",
          minHeight: isDraft ? 16 : 0,
        }}>
          {data.beatIntent || <span style={{ color: "#8c7e70" }}>Empty beat</span>}
        </div>

        {/* Character chips */}
        {hasChars && (
          <div style={{ display: "flex", gap: 3, marginBottom: 8, flexWrap: "wrap" }}>
            {charInitials.map((initials, i) => (
              <span key={i} style={{
                display: "inline-flex",
                alignItems: "center",
                justifyContent: "center",
                width: 20,
                height: 20,
                borderRadius: "50%",
                background: `linear-gradient(135deg, ${isAccepted ? "#6b8f5e" : "#8c7e70"}, ${isAccepted ? "#8bb87a" : "#a89888"})`,
                color: "#f5f0e8",
                fontSize: 8,
                fontWeight: 700,
                fontFamily: "var(--font-mono)",
                letterSpacing: "0.02em",
                boxShadow: "0 1px 2px rgba(0,0,0,0.15)",
              }}>
                {initials}
              </span>
            ))}
          </div>
        )}

        {/* Word count progress bar */}
        <div style={{
          display: "flex",
          alignItems: "center",
          gap: 5,
          marginTop: !isDraft && data.wordCount !== undefined ? 0 : 0,
        }}>
          <div style={{
            flex: 1,
            height: 3,
            borderRadius: 2,
            background: "rgba(0,0,0,0.08)",
            overflow: "hidden",
          }}>
            <div style={{
              width: `${wordProgress * 100}%`,
              height: "100%",
              borderRadius: 2,
              background: `linear-gradient(90deg, ${progressColor}, ${progressColor}cc)`,
              transition: "width 0.4s var(--ease-out)",
              boxShadow: wordProgress > 0 ? `0 0 4px ${progressColor}66` : "none",
            }} />
          </div>
          <span style={{
            fontSize: 9,
            fontFamily: "var(--font-mono)",
            color: "#7a6e62",
            flexShrink: 0,
          }}>
            {data.wordCount || 0}/{data.targetWords}
          </span>
        </div>

        {/* POV · Tone · metadata row */}
        <div style={{
          fontSize: 9.5,
          color: "#7a6e62",
          fontFamily: "var(--font-mono)",
          display: "flex",
          gap: 5,
          alignItems: "center",
          marginTop: 4,
        }}>
          <span>{data.pov || "—"}</span>
          <span style={{ opacity: 0.25, fontSize: 8 }}>◆</span>
          <span>{data.tone || "—"}</span>
        </div>

        <Handle type="source" position={Position.Right} style={{
          background: cfg.dot,
          width: 8,
          height: 8,
          border: `2px solid #f5f0e8`,
          borderRadius: "50%",
          boxShadow: `0 0 4px ${cfg.dot}66`,
          transition: "transform var(--transition-fast), box-shadow var(--transition-fast)",
        }}
          onMouseEnter={(e) => { e.currentTarget.style.transform = "scale(1.3)"; e.currentTarget.style.boxShadow = `0 0 8px ${cfg.dot}` }}
          onMouseLeave={(e) => { e.currentTarget.style.transform = "scale(1)"; e.currentTarget.style.boxShadow = `0 0 4px ${cfg.dot}66` }}
        />
      </div>
    </div>
  )
}

export default memo(SceneNode)
