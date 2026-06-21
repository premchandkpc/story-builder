import { useState } from "react"
import type { AgentRun } from "../api/types"
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
    return <div style={{ color: "var(--text-muted)", fontSize: 12, padding: 8 }}>Loading agent runs...</div>
  }

  if (!runs || runs.length === 0) {
    return <div style={{ color: "var(--text-muted)", fontSize: 12, fontStyle: "italic", padding: 8 }}>No agent runs recorded.</div>
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
            borderRadius: 6, padding: 10,
            background: isFailed ? "rgba(212,103,103,0.05)" : "var(--surface)",
          }}>
            <div
              onClick={() => setExpanded(isExpanded ? null : run.id)}
              style={{ cursor: "pointer", display: "flex", justifyContent: "space-between", alignItems: "center" }}
            >
              <div style={{ display: "flex", alignItems: "center", gap: 6 }}>
                <span style={{
                  width: 8, height: 8, borderRadius: "50%",
                  background: isFailed ? "var(--error)" :
                    run.status === "done" ? "var(--success)" : "var(--accent)",
                  flexShrink: 0,
                }} />
                <span style={{
                  fontSize: 11, fontWeight: 600, color,
                  textTransform: "uppercase", letterSpacing: "0.04em",
                }}>
                  {run.agent_type}
                </span>
                {run.model && (
                  <span style={{ fontSize: 9, color: "var(--text-muted)", padding: "1px 4px", background: "rgba(136,136,160,0.1)", borderRadius: 3 }}>
                    {run.model}
                  </span>
                )}
              </div>
              <div style={{ display: "flex", alignItems: "center", gap: 6 }}>
                <span style={{ fontSize: 10, color: "var(--text-muted)" }}>
                  {run.duration_ms > 0 ? `${(run.duration_ms / 1000).toFixed(1)}s` : run.status}
                </span>
                <span style={{ fontSize: 10, color: "var(--accent)" }}>
                  {isExpanded ? "▲" : "▼"}
                </span>
              </div>
            </div>

            {isExpanded && (
              <div style={{ marginTop: 8, fontSize: 11, fontFamily: "monospace" }}>
                <div style={{ marginBottom: 6 }}>
                  <div style={{ color: "var(--text-muted)", fontSize: 10, marginBottom: 2 }}>Input</div>
                  <pre style={{
                    margin: 0, padding: 6, background: "var(--bg)", borderRadius: 4,
                    maxHeight: 150, overflowY: "auto", fontSize: 10, color: "var(--text)",
                    whiteSpace: "pre-wrap", wordBreak: "break-all",
                  }}>
                    {JSON.stringify(run.input, null, 2)}
                  </pre>
                </div>
                <div>
                  <div style={{ color: "var(--text-muted)", fontSize: 10, marginBottom: 2 }}>Output</div>
                  <pre style={{
                    margin: 0, padding: 6, background: "var(--bg)", borderRadius: 4,
                    maxHeight: 200, overflowY: "auto", fontSize: 10, color: "var(--text)",
                    whiteSpace: "pre-wrap", wordBreak: "break-all",
                  }}>
                    {JSON.stringify(run.output, null, 2)}
                  </pre>
                </div>
                {run.error && (
                  <div style={{ color: "var(--error)", marginTop: 4, fontSize: 10 }}>
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
