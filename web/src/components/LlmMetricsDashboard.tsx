import { useLlmMetrics } from "../api/hooks"
import { fadeInStyle, spinnerStyle } from "../api/types"

interface LlmMetricsDashboardProps {
  storyId: string
}


function StatCard({ label, value }: { label: string; value: string }) {
  return (
    <div style={{
      background: "var(--surface)", border: "1px solid var(--border)",
      borderRadius: "var(--radius-md)", padding: "12px 14px",
    }}>
      <div style={{
        fontSize: 10, color: "var(--text-dim)",
        textTransform: "uppercase", letterSpacing: "0.05em",
        fontWeight: 500,
      }}>
        {label}
      </div>
      <div style={{
        fontSize: 18, fontWeight: 700,
        fontFamily: "var(--font-heading)", color: "var(--accent)",
        marginTop: 4,
      }}>
        {value}
      </div>
    </div>
  )
}

export default function LlmMetricsDashboard({ storyId }: LlmMetricsDashboardProps) {
  const { data: metrics, isLoading, error } = useLlmMetrics(storyId)

  if (isLoading) {
    return (
      <div style={{ display: "flex", alignItems: "center", justifyContent: "center", padding: 32 }}>
        <div style={spinnerStyle} />
      </div>
    )
  }

  if (error || !metrics) {
    return (
      <div style={{ padding: 16, color: "var(--text-faint)", fontSize: 12, fontStyle: "italic" }}>
        No metrics available yet. Generate some scenes first.
      </div>
    )
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 16, ...fadeInStyle }}>
      <h3 style={{
        margin: 0, fontSize: 14,
        fontFamily: "var(--font-heading)", color: "var(--accent)",
        fontWeight: 600, letterSpacing: "0.02em",
      }}>
        LLM Usage
      </h3>

      <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 8 }}>
        <StatCard label="Total Tokens" value={metrics.total_tokens.toLocaleString()} />
        <StatCard label="Est. Cost" value={`$${metrics.total_cost_estimate.toFixed(4)}`} />
        <StatCard label="Prompt" value={metrics.total_prompt_tokens.toLocaleString()} />
        <StatCard label="Completion" value={metrics.total_completion_tokens.toLocaleString()} />
        <StatCard label="Turns" value={metrics.turn_count.toString()} />
        <StatCard label="Generations" value={metrics.generation_count.toString()} />
      </div>

      <div>
        <h4 style={{
          margin: "8px 0 4px", fontSize: 12, color: "var(--text)",
          fontFamily: "var(--font-heading)",
        }}>
          By Model
        </h4>
        <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
          {Object.entries(metrics.by_model).map(([model, data]) => (
            <div key={model} style={{
              display: "flex", justifyContent: "space-between", alignItems: "center",
              padding: "6px 10px", background: "var(--surface)", borderRadius: "var(--radius-sm)",
              fontSize: 11, border: "1px solid var(--border)",
            }}>
              <span style={{ color: "var(--accent)", fontWeight: 600 }}>{model}</span>
              <span style={{ color: "var(--text-dim)", fontFamily: "var(--font-mono)" }}>
                P:{data.prompt_tokens.toLocaleString()} C:{data.completion_tokens.toLocaleString()}
              </span>
              <span style={{ color: "var(--text)" }}>
                ${data.cost.toFixed(4)}
              </span>
            </div>
          ))}
        </div>
      </div>

      <div>
        <h4 style={{
          margin: "8px 0 4px", fontSize: 12, color: "var(--text)",
          fontFamily: "var(--font-heading)",
        }}>
          By Agent
        </h4>
        <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
          {Object.entries(metrics.by_agent).map(([agent, data]) => (
            <div key={agent} style={{
              display: "flex", justifyContent: "space-between", alignItems: "center",
              padding: "6px 10px", background: "var(--surface)", borderRadius: 4,
              fontSize: 11, border: "1px solid var(--border)",
            }}>
              <span style={{
                color: "var(--text)", fontWeight: 500,
                textTransform: "capitalize",
              }}>
                {agent}
              </span>
              <span style={{ color: "var(--text-dim)" }}>
                {data.turn_count} turns · {data.prompt_tokens.toLocaleString()} tokens
              </span>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
