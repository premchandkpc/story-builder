import { useCriticScores } from "../api/hooks"
import { fadeInStyle, spinnerStyle } from "../api/types"

interface CriticScoreDashboardProps {
  storyId: string
}

const scoreColor = (score: number): string => {
  if (score >= 0.8) return "#22c55e"
  if (score >= 0.6) return "#eab308"
  return "#ef4444"
}

function StatCard({ label, value, color }: { label: string; value: string; color: string }) {
  return (
    <div style={{
      background: "var(--surface)", border: "1px solid var(--border)",
      borderRadius: 8, padding: "12px 14px",
      transition: "border-color 0.15s",
    }}>
      <div style={{
        fontSize: 10, color: "var(--text-dim)",
        textTransform: "uppercase", letterSpacing: "0.05em",
        fontWeight: 500,
      }}>
        {label}
      </div>
      <div style={{
        fontSize: 20, fontWeight: 700,
        fontFamily: "var(--font-heading)", color,
        marginTop: 4,
      }}>
        {value}
      </div>
    </div>
  )
}

export default function CriticScoreDashboard({ storyId }: CriticScoreDashboardProps) {
  const { data: scores, isLoading, error } = useCriticScores(storyId)

  if (isLoading) {
    return (
      <div style={{ display: "flex", alignItems: "center", justifyContent: "center", padding: 32 }}>
        <div style={spinnerStyle} />
      </div>
    )
  }

  if (error || !scores || scores.length === 0) {
    return (
      <div style={{ padding: 16, color: "var(--text-dim)", fontSize: 12 }}>
        No critic evaluations yet. Generate scenes with agent mode first.
      </div>
    )
  }

  const avgScore = scores.reduce((sum, s) => sum + s.score, 0) / scores.length
  const maxScore = Math.max(...scores.map(s => s.score))
  const minScore = Math.min(...scores.map(s => s.score))

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 16, ...fadeInStyle }}>
      <h3 style={{
        margin: 0, fontSize: 14,
        fontFamily: "var(--font-heading)", color: "var(--accent)",
      }}>
        Critic Evaluations
      </h3>

      <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr 1fr", gap: 8 }}>
        <StatCard
          label="Average Score"
          value={`${(avgScore * 100).toFixed(0)}%`}
          color={scoreColor(avgScore)}
        />
        <StatCard
          label="Best"
          value={`${(maxScore * 100).toFixed(0)}%`}
          color={scoreColor(maxScore)}
        />
        <StatCard
          label="Worst"
          value={`${(minScore * 100).toFixed(0)}%`}
          color={scoreColor(minScore)}
        />
      </div>

      <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
        <h4 style={{
          margin: "8px 0 4px", fontSize: 12, color: "var(--text)",
          fontFamily: "var(--font-heading)",
        }}>
          Scene Scores
        </h4>
        {scores.map((s) => (
          <div key={s.generation_id} style={{
            display: "flex", justifyContent: "space-between", alignItems: "center",
            padding: "8px 10px", background: "var(--surface)", borderRadius: 4,
            fontSize: 11,
            border: "1px solid var(--border)",
          }}>
            <span style={{
              color: "var(--accent)", fontWeight: 600,
              fontFamily: "var(--font-mono)", fontSize: 10,
            }}>
              {s.scene_id.slice(0, 8)}
            </span>
            <span style={{ color: "var(--text-dim)", flex: 1, marginLeft: 8 }}>
              {s.summary}
            </span>
            <span style={{
              color: scoreColor(s.score),
              fontWeight: 700,
              fontFamily: "var(--font-mono)",
              padding: "2px 6px",
              borderRadius: 4,
              background: `${scoreColor(s.score)}15`,
            }}>
              {(s.score * 100).toFixed(0)}%
            </span>
          </div>
        ))}
      </div>
    </div>
  )
}
