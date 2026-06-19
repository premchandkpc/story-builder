import { useNavigate } from "react-router-dom"
import type { Story } from "../api/types"

interface StoryListItemProps {
  story: Story
  chapterCount: number
  sceneCount: number
  accepted: number
  generated: number
  stale: number
  isActive: boolean
  index?: number
}

export default function StoryListItem({
  story, chapterCount, sceneCount, accepted, generated, stale, isActive, index = 0
}: StoryListItemProps) {
  const navigate = useNavigate()

  const info =
    sceneCount === 0
      ? "No scenes"
      : `${chapterCount} chapters · ${sceneCount} scenes · ${accepted} accepted · ${generated} generated`

  const statusColor =
    stale > 0                                ? "var(--error)"
    : sceneCount > 0 && accepted === sceneCount ? "var(--success)"
    : sceneCount > 0                           ? "var(--accent)"
                                                 : "var(--text-muted)"

  const date = story.createdAt ? new Date(story.createdAt).toLocaleDateString() : ""

  return (
    <button
      onClick={() => navigate(`/stories/${story.id}`)}
      style={{
        width: "100%",
        padding: "12px 14px",
        background: isActive ? "var(--surface)" : "transparent",
        border: "none",
        borderBottom: "1px solid var(--border)",
        borderLeft: isActive ? "2px solid var(--accent)" : "2px solid transparent",
        color: "var(--text)",
        cursor: "pointer",
        textAlign: "left",
        fontSize: 14,
        display: "flex",
        flexDirection: "column",
        gap: 3,
        transition: "background 0.15s, border-left-color 0.15s",
        animation: `slideInLeft 0.25s var(--ease-out) ${index * 0.03}s both`,
      }}
      onMouseEnter={(e) => {
        if (!isActive) e.currentTarget.style.background = "var(--surface-hover)"
      }}
      onMouseLeave={(e) => {
        if (!isActive) e.currentTarget.style.background = "transparent"
      }}
    >
      <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
        <span style={{
          width: 8,
          height: 8,
          borderRadius: "50%",
          background: statusColor,
          flexShrink: 0,
          boxShadow: `0 0 6px ${statusColor}`,
        }} />
        <span style={{
          fontWeight: isActive ? 700 : 500,
          fontFamily: isActive ? "var(--font-heading)" : "var(--font-body)",
          color: isActive ? "var(--accent)" : "var(--text)",
        }}>
          {story.title}
        </span>
      </div>
      <div style={{ fontSize: 11, color: "var(--text-muted)", paddingLeft: 16 }}>{info} · {date}</div>
    </button>
  )
}