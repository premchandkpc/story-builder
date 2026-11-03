import { useCriticScores } from "../api/hooks"
import { fadeInStyle, spinnerStyle } from "../api/types"
import StatCard from "./StatCard"

interface CriticScoreDashboardProps {
  storyId: string
}

const scoreColor = (score: number): string => {
  if (score >= 0.8) return "#22c55e"
  if (score >= 0.6) return "#eab308"
  return "#ef4444"
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
      <div style={{ padding: 16, color: "var(--text-faint)", fontSize: 12, fontStyle: "italic" }}>
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
        fontWeight: 600, letterSpacing: "0.02em",
      }}>
        Critic Evaluations
      </h3>

      <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr 1fr", gap: 8 }}>
        <StatCard
          label="Average Score"
          value={`${(avgScore * 100).toFixed(0)}%`}
          color={scoreColor(avgScore)}
          delay={0.05}
        />
        <StatCard
          label="Best"
          value={`${(maxScore * 100).toFixed(0)}%`}
          color={scoreColor(maxScore)}
          delay={0.1}
        />
        <StatCard
          label="Worst"
          value={`${(minScore * 100).toFixed(0)}%`}
          color={scoreColor(minScore)}
          delay={0.15}
        />
      </div>

      <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
        <h4 style={{
          margin: "8px 0 4px", fontSize: 12, color: "var(--text)",
          fontFamily: "var(--font-heading)",
        }}>
          Scene Scores
        </h4>
        {scores.map((s, i) => (
          <div key={s.generation_id} className="card-hover" style={{
            display: "flex", justifyContent: "space-between", alignItems: "center",
            padding: "8px 10px", background: "var(--surface)", borderRadius: "var(--radius-sm)",
            fontSize: 11,
            border: "1px solid var(--border)",
            animation: `slideUp 0.25s var(--ease-out) ${i * 0.04}s both`,
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
              borderRadius: "var(--radius-sm)",
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
