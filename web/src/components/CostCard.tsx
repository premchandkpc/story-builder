import type { CostSummary } from "../api/types"
import { cardStyle } from "../api/types"

interface CostCardProps {
  cost: CostSummary | null
  loading?: boolean
}

export default function CostCard({ cost, loading }: CostCardProps) {
  if (loading) {
    return <div style={{ color: "var(--text-faint)", fontStyle: "italic", fontSize: 11, padding: 8 }}>Loading cost...</div>
  }

  if (!cost) {
    return <div style={{ color: "var(--text-dim)", fontStyle: "italic", fontSize: 11, padding: 8 }}>No cost data.</div>
  }

  return (
    <div style={{ ...cardStyle, padding: "12px 14px", fontSize: 11 }}>
      <div style={{ fontWeight: 600, marginBottom: 8, color: "var(--text)" }}>Cost Summary</div>
      <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
        <div style={{ display: "flex", justifyContent: "space-between" }}>
          <span style={{ color: "var(--text-dim)" }}>Total tokens</span>
          <span style={{ fontWeight: 600 }}>{cost.totalTokens.toLocaleString()}</span>
        </div>
        <div style={{ display: "flex", justifyContent: "space-between" }}>
          <span style={{ color: "var(--text-dim)" }}>Estimated cost</span>
          <span style={{ fontWeight: 600 }}>${cost.estimatedCost.toFixed(4)}</span>
        </div>
        {Object.entries(cost.byModel).length > 0 && (
          <div style={{ marginTop: 4, borderTop: "1px solid var(--border)", paddingTop: 6 }}>
            <div style={{ color: "var(--text-dim)", marginBottom: 4, fontSize: 10, textTransform: "uppercase", letterSpacing: "0.04em" }}>By Model</div>
            {Object.entries(cost.byModel).map(([model, mc]) => (
              <div key={model} style={{ display: "flex", justifyContent: "space-between", fontSize: 10, padding: "2px 0" }}>
                <span style={{ color: "var(--text-dim)" }}>{model}</span>
                <span>{mc.tokens.toLocaleString()} tokens · ${mc.cost.toFixed(4)}</span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
