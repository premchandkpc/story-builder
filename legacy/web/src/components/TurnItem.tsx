import { memo, useState } from "react"
import type { SceneTurn } from "../api/types"
import { slideUpStyle } from "../api/types"

const roleColors: Record<string, string> = {
  director: "#d4a853",
  character: "#7bb87b",
  narrator: "#6b9fc4",
  editor: "#c9734a",
  canon_guard: "#d46767",
  critic: "#a87bd4",
  world: "#4ac9a8",
  arc: "#e8a84a",
  state_extractor: "#8888a0",
  memory: "#7b9fd4",
}

const statusIcons: Record<string, string> = {
  done: "●",
  failed: "●",
  running: "◉",
  pending: "○",
  skipped: "—",
}

function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

interface TurnItemProps {
  turn: SceneTurn
  index: number
  compact?: boolean
}

const TurnItem = memo(function TurnItem({ turn, index, compact }: TurnItemProps) {
  const [expanded, setExpanded] = useState(false)
  const roleColor = roleColors[turn.role] || "var(--text-muted)"
  const isAccepted = turn.status === "done"
  const isFailed = turn.status === "failed"
  const isExpanded = expanded

  return (
    <div
      className="card-hover"
      style={{
        ...(compact ? {} : slideUpStyle),
        animationDelay: `${index * 0.04}s`,
        borderLeft: `2px solid ${isFailed ? "var(--error)" : isAccepted ? roleColor : "var(--border)"}`,
        padding: "8px 12px",
        marginLeft: 4,
        position: "relative",
        background: isExpanded ? "rgba(212,168,83,0.04)" : "transparent",
        borderRadius: "0 var(--radius-sm) var(--radius-sm) 0",
        transition: "background var(--transition-base), border-color var(--transition-base)",
      }}
    >
      <div style={{ position: "absolute", left: -7, top: 10, fontSize: 10, color: roleColor }}>
        {statusIcons[turn.status] || "○"}
      </div>

      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", gap: 8 }}>
        <div style={{ display: "flex", alignItems: "center", gap: 6, flex: 1, minWidth: 0 }}>
          <span style={{
            fontSize: 10, fontWeight: 700, color: roleColor,
            textTransform: "uppercase", letterSpacing: "0.04em",
          }}>
            {turn.role}
          </span>
          <span style={{ fontSize: 10, color: "var(--text-dim)" }}>
            #{turn.number}
          </span>
          {turn.model && (
            <span style={{
              fontSize: 9, padding: "1px 5px", borderRadius: "var(--radius-sm)",
              background: "rgba(136,136,160,0.1)", color: "var(--text-dim)",
              fontFamily: "var(--font-mono)",
            }}>
              {turn.model}
            </span>
          )}
        </div>

        <div style={{ display: "flex", alignItems: "center", gap: 6 }}>
          <span style={{
            fontSize: 10, color: turn.status === "done" ? "var(--success)" :
              turn.status === "failed" ? "var(--error)" : "var(--text-dim)",
            fontFamily: "var(--font-mono)",
          }}>
            {turn.duration_ms > 0 ? formatDuration(turn.duration_ms) : turn.status}
          </span>
          {turn.output && (
            <button
              onClick={() => setExpanded(!isExpanded)}
              style={{
                background: "none", border: "none", color: "var(--accent)",
                cursor: "pointer", fontSize: 10, padding: "2px 4px",
                transition: "color var(--transition-fast)",
              }}
              onMouseEnter={(e) => e.currentTarget.style.color = "var(--accent-hover)"}
              onMouseLeave={(e) => e.currentTarget.style.color = "var(--accent)"}
            >
              {isExpanded ? "▲" : "▼"}
            </button>
          )}
        </div>
      </div>

      {isExpanded && (
        <div style={{
          marginTop: 8, fontSize: 11, lineHeight: 1.5,
          animation: "expandIn 0.2s var(--ease-out)",
          transformOrigin: "top",
        }}>
          {turn.input && (
            <div style={{ marginBottom: 6 }}>
              <div style={{ color: "var(--text-dim)", fontSize: 10, marginBottom: 2 }}>Input</div>
              <div style={{
                padding: 6, background: "var(--bg)", borderRadius: "var(--radius-sm)",
                whiteSpace: "pre-wrap", maxHeight: 120, overflowY: "auto",
                color: "var(--text)", fontSize: 10, fontFamily: "var(--font-mono)",
                border: "1px solid var(--border)",
                boxShadow: "var(--shadow-inner)",
              }}>
                {turn.input.length > 500 ? turn.input.slice(0, 500) + "..." : turn.input}
              </div>
            </div>
          )}
          {turn.output && (
            <div>
              <div style={{ color: "var(--text-dim)", fontSize: 10, marginBottom: 2 }}>Output</div>
              <div style={{
                padding: 6, background: "var(--bg)", borderRadius: "var(--radius-sm)",
                whiteSpace: "pre-wrap", maxHeight: 200, overflowY: "auto",
                color: "var(--text)", fontSize: 10, fontFamily: "var(--font-mono)",
                border: "1px solid var(--border)",
                boxShadow: "var(--shadow-inner)",
              }}>
                {turn.output.length > 1000 ? turn.output.slice(0, 1000) + "..." : turn.output}
              </div>
            </div>
          )}
          {turn.error && (
            <div style={{
              color: "var(--error)", fontSize: 10, marginTop: 4,
              background: "var(--error-dim)", padding: "4px 6px",
              borderRadius: "var(--radius-sm)",
              animation: "fadeIn 0.2s var(--ease-out)",
            }}>
              Error: {turn.error}
            </div>
          )}

          <div style={{ display: "flex", gap: 4, marginTop: 6 }}>
            {turn.prompt_tokens > 0 && (
              <span style={{ fontSize: 9, color: "var(--text-dim)", fontFamily: "var(--font-mono)" }}>
                P:{turn.prompt_tokens} C:{turn.completion_tokens} T:{turn.prompt_tokens + turn.completion_tokens}
              </span>
            )}
          </div>
        </div>
      )}
    </div>
  )
})

export default TurnItem
