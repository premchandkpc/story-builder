import { useState } from "react"
import { useAgentRuns } from "../api/hooks"
import { fadeInStyle } from "../api/types"

interface AgentRunPanelProps {
  storyId: string
  nodeId: string
}

const agentColors: Record<string, string> = {
  director: "#d4a853",
  character: "#7bb87b",
  narrator: "#6b9fc4",
  editor: "#c9734a",
  canon_guard: "#d46767",
  critic: "#a87bd4",
  world: "#4ac9a8",
  arc: "#e8a84a",
  state_extract: "#8888a0",
  memory: "#7b9fd4",
  orchestrator: "#d4a853",
}

export default function AgentRunPanel({ storyId, nodeId }: AgentRunPanelProps) {
  const { data: runs, isLoading } = useAgentRuns(storyId, nodeId)
  const [expanded, setExpanded] = useState<string | null>(null)

  if (isLoading) {
    return <div style={{ color: "var(--text-faint)", fontSize: 12, padding: 8, fontStyle: "italic" }}>Loading agent runs...</div>
  }

  if (!runs || runs.length === 0) {
    return <div style={{ color: "var(--text-faint)", fontSize: 12, fontStyle: "italic", padding: 8 }}>No agent runs recorded.</div>
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 6, ...fadeInStyle }}>
      {runs.map((run) => {
        const isExpanded = expanded === run.id
        const color = agentColors[run.agent_type] || "var(--text-muted)"
        const isFailed = run.status === "failed" || !!run.error

        return (
          <div key={run.id} style={{
            border: `1px solid ${isFailed ? "var(--error)" : "var(--border)"}`,
            borderRadius: "var(--radius-md)", padding: 10,
            background: isFailed ? "rgba(212,103,103,0.04)" : "var(--surface)",
          }}>
            <div
              onClick={() => setExpanded(isExpanded ? null : run.id)}
              style={{ cursor: "pointer", display: "flex", justifyContent: "space-between", alignItems: "center" }}
            >
              <div style={{ display: "flex", alignItems: "center", gap: 6 }}>
                <span style={{
                  width: 7, height: 7, borderRadius: "50%",
                  background: isFailed ? "var(--error)" :
                    run.status === "done" ? "var(--success)" : "var(--accent)",
                  flexShrink: 0,
                  boxShadow: isFailed ? "0 0 6px var(--error)" : run.status === "done" ? "0 0 6px var(--success)" : "none",
                }} />
                <span style={{
                  fontSize: 11, fontWeight: 600, color,
                  textTransform: "uppercase", letterSpacing: "0.04em",
                }}>
                  {run.agent_type}
                </span>
                {run.model && (
                  <span style={{
                    fontSize: 9, color: "var(--text-dim)",
                    padding: "1px 5px", background: "rgba(136,136,160,0.08)",
                    borderRadius: "var(--radius-sm)", fontFamily: "var(--font-mono)",
                  }}>
                    {run.model}
                  </span>
                )}
              </div>
              <div style={{ display: "flex", alignItems: "center", gap: 6 }}>
                <span style={{ fontSize: 10, color: "var(--text-dim)", fontFamily: "var(--font-mono)" }}>
                  {run.duration_ms > 0 ? `${(run.duration_ms / 1000).toFixed(1)}s` : run.status}
                </span>
                <span style={{ fontSize: 10, color: "var(--accent)" }}>
                  {isExpanded ? "▲" : "▼"}
                </span>
              </div>
            </div>

            {isExpanded && (
              <div style={{ marginTop: 8, fontSize: 11 }}>
                <div style={{ marginBottom: 6 }}>
                  <div style={{ color: "var(--text-dim)", fontSize: 10, marginBottom: 2 }}>Input</div>
                  <pre style={{
                    margin: 0, padding: 6, background: "var(--bg)", borderRadius: "var(--radius-sm)",
                    maxHeight: 150, overflowY: "auto", fontSize: 10, color: "var(--text)",
                    whiteSpace: "pre-wrap", wordBreak: "break-all",
                    border: "1px solid var(--border)",
                    fontFamily: "var(--font-mono)",
                    boxShadow: "var(--shadow-inner)",
                  }}>
                    {JSON.stringify(run.input, null, 2)}
                  </pre>
                </div>
                <div>
                  <div style={{ color: "var(--text-dim)", fontSize: 10, marginBottom: 2 }}>Output</div>
                  <pre style={{
                    margin: 0, padding: 6, background: "var(--bg)", borderRadius: "var(--radius-sm)",
                    maxHeight: 200, overflowY: "auto", fontSize: 10, color: "var(--text)",
                    whiteSpace: "pre-wrap", wordBreak: "break-all",
                    border: "1px solid var(--border)",
                    fontFamily: "var(--font-mono)",
                    boxShadow: "var(--shadow-inner)",
                  }}>
                    {JSON.stringify(run.output, null, 2)}
                  </pre>
                </div>
                {run.error && (
                  <div style={{ color: "var(--error)", marginTop: 4, fontSize: 10, background: "var(--error-dim)", padding: "4px 6px", borderRadius: "var(--radius-sm)" }}>
                    {run.error}
                  </div>
                )}
              </div>
            )}
          </div>
        )
      })}
    </div>
  )
}
