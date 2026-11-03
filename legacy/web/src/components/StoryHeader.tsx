import type { StoryViewMode } from "../api/types"
import { useStoryNodeStats } from "../api/hooks"

interface StoryHeaderProps {
  storyId: string
  title: string
  viewMode: StoryViewMode
  onViewModeChange: (mode: StoryViewMode) => void
}

const viewModes: { key: StoryViewMode; label: string }[] = [
  { key: "read", label: "Read" },
  { key: "outline", label: "Outline" },
  { key: "graph", label: "Graph" },
  { key: "inspect", label: "Inspect" },
]

const tabStyle = (active: boolean): React.CSSProperties => ({
  padding: "6px 14px",
  background: active ? "var(--accent)" : "transparent",
  color: active ? "#1a1512" : "var(--text-dim)",
  border: `1px solid ${active ? "var(--accent)" : "var(--border)"}`,
  borderRadius: "var(--radius-md)",
  cursor: "pointer",
  fontWeight: active ? 600 : 400,
  fontSize: 12,
  letterSpacing: "0.03em",
  transition: "all 0.15s",
})

export default function StoryHeader({ storyId, title, viewMode, onViewModeChange }: StoryHeaderProps) {
  const { data: stats } = useStoryNodeStats(storyId)

  return (
    <div style={{
      display: "flex", alignItems: "center", gap: 16,
      padding: "0 20px", height: 48,
      borderBottom: "1px solid var(--border)",
      background: "var(--bg-warm)",
      flexShrink: 0,
    }}>
      <h1 style={{
        margin: 0, fontSize: 15, fontWeight: 600,
        fontFamily: "var(--font-heading)", color: "var(--text)",
        letterSpacing: "0.01em", whiteSpace: "nowrap", overflow: "hidden", textOverflow: "ellipsis",
      }}>
        {title}
      </h1>

      <div style={{ display: "flex", gap: 4 }}>
        {viewModes.map(({ key, label }) => (
          <button
            key={key}
            onClick={() => onViewModeChange(key)}
            className="btn-press"
            style={tabStyle(viewMode === key)}
          >
            {label}
          </button>
        ))}
      </div>

      {stats && (
        <div style={{
          marginLeft: "auto", display: "flex", gap: 12,
          fontSize: 10, color: "var(--text-faint)", letterSpacing: "0.03em",
        }}>
          <span>{stats.total} scenes</span>
          <span style={{ color: "var(--success)" }}>{stats.accepted} done</span>
          {stats.pending > 0 && <span style={{ color: "var(--warn)" }}>{stats.pending} pending</span>}
          <span style={{ color: "var(--text-faint)" }}>·</span>
          <span>{stats.generated} generated</span>
        </div>
      )}
    </div>
  )
}
