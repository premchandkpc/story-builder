import { useState } from "react"
import { useTurns } from "../api/hooks"
import { fadeInStyle, slideUpStyle } from "../api/types"

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

interface TurnTimelineProps {
  storyId: string
  nodeId: string
  compact?: boolean
}

function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

export default function TurnTimeline({ storyId, nodeId, compact }: TurnTimelineProps) {
  const { data: turns, isLoading } = useTurns(storyId, nodeId)
  const [expandedTurn, setExpandedTurn] = useState<string | null>(null)

  if (isLoading) {
    return <div style={{ color: "var(--text-faint)", fontSize: 12, padding: 8, fontStyle: "italic" }}>Loading turns...</div>
  }

  if (!turns || turns.length === 0) {
    return <div style={{ color: "var(--text-faint)", fontSize: 12, fontStyle: "italic", padding: 8 }}>No turns yet. Generate to populate.</div>
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 0, ...(compact ? {} : fadeInStyle) }}>
      {turns.map((turn) => {
        const isExpanded = expandedTurn === turn.id
        const roleColor = roleColors[turn.role] || "var(--text-muted)"
        const isAccepted = turn.status === "done"
        const isFailed = turn.status === "failed"

        return (
          <div key={turn.id} style={{
            ...(compact ? {} : slideUpStyle),
            borderLeft: `2px solid ${isFailed ? "var(--error)" : isAccepted ? roleColor : "var(--border)"}`,
            padding: "8px 12px",
            marginLeft: 4,
            position: "relative",
            background: isExpanded ? "rgba(212,168,83,0.04)" : "transparent",
            transition: "background 0.15s",
          }}>
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
                    onClick={() => setExpandedTurn(isExpanded ? null : turn.id)}
                    style={{
                      background: "none", border: "none", color: "var(--accent)",
                      cursor: "pointer", fontSize: 10, padding: "2px 4px",
                    }}
                  >
                    {isExpanded ? "▲" : "▼"}
                  </button>
                )}
              </div>
            </div>

            {isExpanded && (
              <div style={{ marginTop: 8, fontSize: 11, lineHeight: 1.5 }}>
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
                  <div style={{ color: "var(--error)", fontSize: 10, marginTop: 4, background: "var(--error-dim)", padding: "4px 6px", borderRadius: "var(--radius-sm)" }}>
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
      })}
    </div>
  )
}
